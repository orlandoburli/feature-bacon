package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

type PingResponse struct {
	Message string `json:"message"`
}

func Ping(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data, err := json.Marshal(PingResponse{Message: "pong"})

	if err != nil {
		log.Fatalf("Error parsing json: %s", err)
		return
	}

	_, err2 := w.Write(data)
	if err2 != nil {
		log.Fatalf("Error writing output: %s", err2)
		return
	}
}
