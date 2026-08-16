package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// JSONResponse defines the shape of every API response.
type JSONResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// writeJSON is a helper to format and send all http responses uniformly.
func (app *application) writeJSON(w http.ResponseWriter, status int, data any, msg string) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := JSONResponse{
		Status:  "success",
		Message: msg,
		Data:    data,
	}

	if status >= 400 {
		resp.Status = "error"
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("failed to encode json response: %v", err)
	}

}
