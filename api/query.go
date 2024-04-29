package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pavi6691/go-boilingdata/boilingdata"
)

type Handler struct {
	instance boilingdata.Instance
}

func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	// Check if the request method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.instance.Auth == nil || !h.instance.Auth.IsUserLoggedIn() {
		http.Error(w, "User signed out, Please Login!", http.StatusUnauthorized)
		return
	}
	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Could not read http request body", http.StatusInternalServerError)
		return
	}

	response, err := h.instance.Query(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Set response content type to JSON
	w.Header().Set("Content-Type", "application/json")
	// Write JSON response to the response body
	responseJSON, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		http.Error(w, "Could marshal response", http.StatusInternalServerError)
		return
	}
	w.Write(responseJSON)

}
