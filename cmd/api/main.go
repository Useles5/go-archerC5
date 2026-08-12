package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/Useles5/go-archerC5/pkg/archerC5"
)

type application struct {
	client *archerC5.RouterClient
}

type JSONResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

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

func (app *application) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.writeJSON(w, http.StatusServiceUnavailable, nil, "Router is unreachable")
		return
	}

	_, err := app.client.GetLANStatus()
	if err != nil {
		app.writeJSON(w, http.StatusServiceUnavailable, nil, "Router is unreachable")
		return
	}

	// dummy data
	data := map[string]string{
		"environment": "development",
		"version":     "1.0.0",
		"router":      "connected",
	}

	app.writeJSON(w, http.StatusOK, data, "API is up and running")

}

func main() {
	pass := os.Getenv("ROUTER_PASS")
	if pass == "" {
		log.Fatal("$ROUTER_PASS env variable is not set")
	}

	routerClient, err := archerC5.NewClient(pass, archerC5.DefaultRouterIP)
	if err != nil {
		log.Fatalf("failed to initialize router client: %v", err)
	}
	defer routerClient.Logout()

	app := &application{
		client: routerClient,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", app.healthHandler)

	addr := ":8080"
	log.Printf("Starting web server on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("failed to start web server: %v", err)
	}
}
