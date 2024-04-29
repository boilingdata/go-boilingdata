package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type WSSPayload struct {
	WssURL string `json:"wssURL"`
}

func (h *Handler) ConnectWSS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.instance.Auth == nil || !h.instance.Auth.IsUserLoggedIn() {
		http.Error(w, "User signed out, Please Login!", http.StatusUnauthorized)
		return
	}
	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Define a variable of type Credentials to store the parsed JSON
	var wssPayload WSSPayload
	// Unmarshal the JSON into the Credentials struct
	if err := json.Unmarshal(body, &wssPayload); err != nil {
		log.Fatalf("failed to parse JSON: %v", err)
	}
	headers, err := h.instance.Auth.GetAWSSingingHeaders(wssPayload.WssURL)
	if err != nil {
		http.Error(w, "Error preparing request", http.StatusInternalServerError)
		return
	}
	h.instance.Wsc.SignedHeader = headers
	if h.instance.Wsc.IsWebSocketClosed() {
		h.instance.Wsc.Connect()
		if h.instance.Wsc.IsWebSocketClosed() {
			http.Error(w, h.instance.Wsc.Error, http.StatusInternalServerError)
		} else {
			w.Write([]byte("Connected!"))
		}
	} else {
		w.Write([]byte("Already Connected!"))
	}
}

func (h *Handler) GetSignedWSSUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.instance.Auth == nil || !h.instance.Auth.IsUserLoggedIn() {
		http.Error(w, "User signed out, Please Login!", http.StatusUnauthorized)
		return
	}

	idToken, err := h.instance.Auth.Authenticate()
	if err != nil {
		http.Error(w, "Error : "+err.Error(), http.StatusInternalServerError)
		return
	}

	headers, err := h.instance.Auth.GetSignedWssHeader(idToken)
	if err != nil {
		http.Error(w, "Error getting signed headers: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sigedUrl, err := h.instance.Auth.GetSignedWssUrl(headers)
	if err != nil {
		http.Error(w, "Error Signing wssUrl: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(sigedUrl))
}
