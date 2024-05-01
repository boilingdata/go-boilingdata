package wsclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/pavi6691/go-boilingdata/constants"
	"github.com/pavi6691/go-boilingdata/models"
)

// WSSClient represents the WebSocket client.
type WSSClient struct {
	URL                 string
	Conn                *websocket.Conn
	DialOpts            *websocket.Dialer
	idleTimeoutMinutes  time.Duration
	idleTimer           *time.Timer
	Wg                  sync.WaitGroup
	ConnInit            sync.WaitGroup
	SignedHeader        http.Header
	Error               string
	mu                  sync.Mutex
	queryMessageChannel chan []byte
	isEveythingOK       bool
	resultsMap          cmap.ConcurrentMap
}

// NewWSSClient creates a new instance of WSSClient.
// Either fully signed url needs to be provided OR signedHeader
func NewWSSClient(url string, idleTimeoutMinutes time.Duration, signedHeader http.Header) *WSSClient {
	if signedHeader == nil {
		signedHeader = make(http.Header)
	}
	return &WSSClient{
		URL:                 url,
		DialOpts:            &websocket.Dialer{},
		idleTimeoutMinutes:  idleTimeoutMinutes,
		SignedHeader:        signedHeader,
		queryMessageChannel: make(chan []byte),
		isEveythingOK:       true,
		resultsMap:          cmap.New(),
	}
}

func (wsc *WSSClient) Connect() {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()
	if wsc.IsWebSocketClosed() {
		log.Println("Connecting to web socket..")
		wsc.ConnInit.Add(1)
		go wsc.connect()
		wsc.ConnInit.Wait()
		if !wsc.IsWebSocketClosed() {
			log.Println("Websocket Connected!")
		}
	}
}

func (wsc *WSSClient) connect() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	wsc.Wg = sync.WaitGroup{}
	wsc.Wg.Add(1)

	go func() {
		defer wsc.Wg.Done()
		for {
			select {
			case <-interrupt:
				log.Println("Interrupt signal received, closing connection")
				wsc.Close()
				return
			}
		}
	}()
	// Connect to WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial(wsc.URL, wsc.SignedHeader)
	if err != nil {
		wsc.Error = err.Error()
		log.Println("dial:", err)
		wsc.ConnInit.Done()
		return
	}
	wsc.Conn = conn // Assign the connection to the Conn field
	if wsc.idleTimeoutMinutes <= 0 {
		wsc.idleTimeoutMinutes = constants.IdleTimeoutMinutes
	} else {
		wsc.idleTimeoutMinutes = wsc.idleTimeoutMinutes * time.Minute
	}
	wsc.isEveythingOK = true
	go wsc.sendMessageAsync()
	go wsc.receiveMessageAsync()
	wsc.resetIdleTimer()
	wsc.ConnInit.Done()
}

// SendMessage sends a message over the WebSocket connection.
func (wsc *WSSClient) SendMessage(message []byte, payload models.Payload) {
	wsc.resultsMap.Set("error", nil)
	wsc.resultsMap.Set(payload.RequestID, nil)
	wsc.queryMessageChannel <- message
}

// Close closes the WebSocket connection. perform clean up
func (wsc *WSSClient) Close() {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()
	if wsc.Conn != nil {
		wsc.isEveythingOK = false
		wsc.Conn.Close()
		wsc.Conn = nil
		wsc.idleTimer = nil
		wsc.resultsMap.Clear()
		log.Println("Websocket connnection closed")
	}
}

func (wsc *WSSClient) IsWebSocketClosed() bool {
	return wsc.Conn == nil || !wsc.isEveythingOK
}

// resetIdleTimer resets the idle timer.
func (wsc *WSSClient) resetIdleTimer() {
	if wsc.idleTimer != nil {
		wsc.idleTimer.Stop()
	}
	wsc.idleTimer = time.AfterFunc(wsc.idleTimeoutMinutes, func() {
		log.Println("Idle timeout reached, closing connection")
		wsc.Close()
	})
}

// Async function to send message through channel
func (wsc *WSSClient) sendMessageAsync() {
	defer wsc.Close()
	for {
		// Read message from the query message channel
		message, ok := <-wsc.queryMessageChannel
		if !ok || !wsc.isEveythingOK {
			log.Println("SendMessageAsync process interrupted. No messages will be sent to websocket now onwards.  Action : Reconnect websocket")
			break
		}
		if wsc.Conn == nil {
			wsc.Error = "Could not send message to websocket -> Not connected to WebSocket server"
			wsc.resultsMap.Set("error", fmt.Errorf("Could not send message to websocket -> "+"Not connected to WebSocket server"))
			return
		}
		wsc.idleTimer.Reset(constants.IdleTimeoutMinutes)
		wsc.mu.Lock()
		err := wsc.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			wsc.isEveythingOK = false
			wsc.resultsMap.Set("error", fmt.Errorf("Could not send message to websocket -> "+"Not connected to WebSocket server"))
		}
		wsc.mu.Unlock()
	}
}

// Async function to receive message through channel
func (wsc *WSSClient) receiveMessageAsync() {
	defer wsc.Close()
	for {
		if wsc.Conn == nil {
			wsc.Error = "Could not receive message from websocket -> Not connected to WebSocket server"
			wsc.resultsMap.Set("error", fmt.Errorf("Could not recieve message from websocket -> "+"Not connected to WebSocket server"))
			return
		}
		if !wsc.isEveythingOK {
			log.Println("ReceiveMessageAsync process intrrupted. No message will be consumed further. Action : Reconnect websocket")
			break
		}
		_, message, err := wsc.Conn.ReadMessage()
		if err != nil {
			wsc.isEveythingOK = false
			wsc.resultsMap.Set("error", fmt.Errorf("Could not read message from websocket -> ", err.Error()))
		} else if message != nil {
			var response *models.Response
			err = json.Unmarshal([]byte(message), &response)
			if err != nil {
				log.Println("Error parsing JSON:", err)
				wsc.resultsMap.Set(response.RequestID, fmt.Errorf("Error parsing JSON: "+err.Error()))
			}
			if v, ok := wsc.resultsMap.Get(response.RequestID); !ok || v == nil {
				var responses = cmap.New()
				wsc.resultsMap.Set(response.RequestID, responses)
			}
			v, _ := wsc.resultsMap.Get(response.RequestID)
			if response.TotalSubBatches == 0 || response.TotalSubBatches == response.SubBatchSerial {
				response.Keys = extractKeys(message)
			}
			v.(cmap.ConcurrentMap).Set(string(response.SubBatchSerial), response)
		}
	}
}

// Function to extract keys from the "data" array
func extractKeys(jsonData []byte) []string {
	// Define a struct to hold the "data" array
	var data struct {
		Data []json.RawMessage `json:"data"`
	}

	// Unmarshal the JSON data into the struct
	err := json.Unmarshal(jsonData, &data)
	if err != nil {
		log.Println("Error extracting keys from response data:", err)
		return nil
	}

	// If there's no data, return nil
	if len(data.Data) == 0 {
		log.Println("No data found")
		return nil
	}

	// Define an empty map to store the keys of the first entry
	var firstEntry json.RawMessage

	// Unmarshal the first entry to extract the keys
	err = json.Unmarshal(data.Data[0], &firstEntry)
	if err != nil {
		log.Println("Error extracting keys from response data:", err)
		return nil
	}
	return parse(firstEntry)
}

func parse(raw json.RawMessage) []string {
	// Convert RawMessage to byte slice
	rawData := []byte(raw)

	var keys []string

	// Index keeps track of the position in the JSON byte slice
	index := 0

	// Loop until the end of the JSON byte slice
	for index < len(rawData) {
		// Find the next double quote, which indicates the start of a key
		keyStart := bytes.IndexByte(rawData[index:], '"')
		if keyStart == -1 {
			// If no double quote is found, break the loop
			break
		}

		// Adjust the index to the position of the double quote
		index += keyStart + 1

		// Find the end of the key by searching for the closing double quote
		keyEnd := bytes.IndexByte(rawData[index:], '"')
		if keyEnd == -1 {
			// If no closing double quote is found, break the loop
			break
		}

		// Extract the key from the JSON byte slice
		key := string(rawData[index : index+keyEnd])

		// Check if the key is followed by a colon (":")
		if index+keyEnd+1 < len(rawData) && string(rawData[index+keyEnd:index+keyEnd+2]) == "\":" {
			// Append the key to the keys slice
			keys = append(keys, key)
		}

		// Adjust the index to the position after the closing double quote
		index += keyEnd + 1
	}

	return keys
}

func (wsc *WSSClient) GetResponseSync(requestID string) (*models.Response, error) {
	var temp *models.Response
	for {
		if v, ok := wsc.resultsMap.Get("error"); ok {
			if v != nil {
				return &models.Response{}, v.(error)
			}
		}
		if _, ok := wsc.resultsMap.Get(requestID); !ok {
			continue
		}
		responses, _ := wsc.resultsMap.Get(requestID)
		if v, ok := responses.(error); ok {
			return &models.Response{}, v
		}
		if responses == nil {
			continue
		}
		if v, ok := responses.(cmap.ConcurrentMap); ok {
			if v.Count() > 0 {
				if temp == nil {
					for item := range v.IterBuffered() {
						temp = item.Val.(*models.Response)
						break
					}
				}
				if len(temp.Data) <= 0 {
					return &models.Response{}, fmt.Errorf("No response from server. Check SQL syntax")
				} else if temp.TotalSubBatches == 0 || temp.TotalSubBatches == v.Count() {
					var data []map[string]interface{}
					for i := 0; i <= v.Count(); i++ {
						v, _ := v.Get(string(rune(i)))
						if v != nil {
							data = append(data, v.(*models.Response).Data...)
						}
					}
					if v.Count() > 0 {
						val, _ := v.Get(string(rune(v.Count())))
						if val == nil {
							val, _ = v.Get(string(rune(0)))
						}
						finalResponse := val.(*models.Response)
						finalResponse.Data = data
						return finalResponse, nil
					}
				}
			}
		}
	}
}
