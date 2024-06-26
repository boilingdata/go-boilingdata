package boilingdata

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/boilingdata/go-boilingdata/constants"
	message "github.com/boilingdata/go-boilingdata/messages"
	"github.com/boilingdata/go-boilingdata/wsclient"
	"github.com/golang-jwt/jwt/v4"
	cmap "github.com/orcaman/concurrent-map"
)

type Instance struct {
	Wsc  *wsclient.WSSClient
	Auth *Auth
}

var queryServiceMap = cmap.New()
var muLock sync.Mutex

func GetInstanceByToken(token string) (*Instance, error) {
	muLock.Lock()
	defer muLock.Unlock()
	// Parse the token
	jwtToken, _ := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Provide the secret key used to sign the token
		return []byte(""), nil
	})
	// Check for errors
	// if err != nil {
	// 	return nil, fmt.Errorf("Error parsing token:", err)
	// }
	// Check if the token is valid
	var userName string
	if claims, ok := jwtToken.Claims.(jwt.MapClaims); ok {
		// Access individual claims
		userName, ok = claims["email"].(string)
		if !ok {
			return nil, fmt.Errorf("Failed to convert username claim to string")
		}
	} else {
		return nil, fmt.Errorf("Invalid token claims")
	}
	// End parsing token

	qs, ok := queryServiceMap.Get(userName)
	if !ok {
		return nil, fmt.Errorf("Token not valid, please login using credentials")
	}
	return qs.(*Instance), nil
}

func GetInstance(userName string, password string) *Instance {
	muLock.Lock()
	defer muLock.Unlock()
	qs, ok := queryServiceMap.Get(userName)
	if !ok {
		wsclient := wsclient.NewWSSClient(constants.WssUrl, 0, nil)
		qs = &Instance{Wsc: wsclient, Auth: &Auth{userName: userName, password: password}}
		queryServiceMap.Set(userName, qs)
	}
	return qs.(*Instance)
}

func RemoveUser(userName string) {
	queryServiceMap.Remove(userName)
}

func (instance *Instance) Query(payloadMessage []byte) (*message.Response, error) {
	// If web socket is closed, in case of timeout/user signout/os intruptions etc
	if instance.Wsc.IsWebSocketClosed() {
		idToken, err := instance.Auth.Authenticate()
		if err != nil {
			return &message.Response{}, fmt.Errorf("Error : " + err.Error())
		}
		header, err := instance.Auth.GetSignedWssHeader(idToken)
		if err != nil {
			return &message.Response{}, fmt.Errorf("Error Signing wssUrl: " + err.Error())
		}
		instance.Wsc.SignedHeader = header
		instance.Wsc.Connect()
		if instance.Wsc.IsWebSocketClosed() {
			return &message.Response{}, fmt.Errorf(instance.Wsc.Error)
		}
	}
	var payload message.Payload
	if err := json.Unmarshal(payloadMessage, &payload); err != nil {
		log.Println("error unmarshalling Payload : " + err.Error())
		return &message.Response{}, fmt.Errorf("error unmarshalling Payload : " + err.Error())
	}
	instance.Wsc.SendMessage(payloadMessage, payload)
	response, err := instance.Wsc.GetResponseSync(payload.RequestID)
	if err != nil || response.Data == nil {
		errorMessage := ""
		if err != nil {
			errorMessage = err.Error()
		}
		return &message.Response{}, fmt.Errorf(errorMessage)
	}
	return response, nil
}
