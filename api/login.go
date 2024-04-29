package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pavi6691/go-boilingdata/boilingdata"
)

type Credentials struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// Check if the request method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Define a variable of type Credentials to store the parsed JSON
	var creds Credentials
	// Unmarshal the JSON into the Credentials struct
	if err := json.Unmarshal(body, &creds); err != nil {
		http.Error(w, "failed to parse JSON: %v", http.StatusInternalServerError)
		return
	}
	instance := boilingdata.GetInstance(creds.UserName, creds.Password)
	_, err = instance.Auth.Authenticate()
	if err != nil {
		http.Error(w, "Error : "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.instance = *instance
	w.Write([]byte("Login Successful!"))
}
