package main

import (
	"log"
	"net/http"

	"github.com/pavi6691/go-boilingdata/api"
)

func main() {
	handler := &api.Handler{}
	http.HandleFunc("/login", handler.Login)
	http.HandleFunc("/connect", handler.ConnectWSS)
	http.HandleFunc("/query", handler.Query)
	http.HandleFunc("/wssurl", handler.GetSignedWSSUrl)
	log.Println("Server is running on port 8088...")
	http.ListenAndServe(":8088", nil)
}
