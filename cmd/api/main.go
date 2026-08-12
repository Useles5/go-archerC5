package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/Useles5/go-archerC5/pkg/archerC5"
)

// application holds all dependencies for the http handlers.
type application struct {
	client *archerC5.RouterClient
}

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

// healthHandler proves the server is running and is reachable.
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

// deviceHandler will fetch all devices and apply status query filters.
func (app *application) deviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.writeJSON(w, http.StatusMethodNotAllowed, nil, "Method is not allowed")
		return
	}

	// fetch all connected devices
	allDevices, err := app.client.GetConnectedDevices()
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, nil, "Failed to fetch connected devices")
		return
	}

	statusFilter := r.URL.Query().Get("status")
	var filteredDevices []archerC5.ConnectedDevice

	for _, device := range allDevices {
		// Design pattern : Negative Exclusion(Guard clause)

		/* Instead of writing nested if/else chains to find positive matches,
		   we immediately 'continue' (skip) any iteration that violates the filter.
		   This keeps the logic perfectly flat and natively handles the
		   "no filter provided" scenario without needing an extra 'else' block. */

		// THE OLD WAY
		// (≖_≖ ) -> [If Match?]
		//              ↳ ( •_•) -> [If Another Match?]
		//                            ↳ ( º﹃º) -> [Processing...]

		//  THE GUARD CLAUSE WAY (Negative Exclusion)
		// (•_•)┌┛ -> ❌ [Violates Filter?] -> Kick it out! (continue)
		//
		// (⌐■_■) ──> [Perfect Match] ──> Proceed in a flat line.

		// user asked for online devices , but this is offline -> skip
		if statusFilter == "online" && !device.Active {
			continue
		}

		// user asked for offline devices , but this is online -> skip
		if statusFilter == "offline" && device.Active {
			continue
		}

		// happy path
		filteredDevices = append(filteredDevices, device)
	}
	app.writeJSON(w, http.StatusOK, filteredDevices, "Successfully fetched requested devices")
}

func main() {
	pass := os.Getenv("ROUTER_PASS")
	if pass == "" {
		log.Fatal("$ROUTER_PASS env variable is not set")
	}

	// initialize
	routerClient, err := archerC5.NewClient(pass, archerC5.DefaultRouterIP)
	if err != nil {
		log.Fatalf("failed to initialize router client: %v", err)
	}
	defer routerClient.Logout()

	// inject
	app := &application{
		client: routerClient,
	}

	// set up the router
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", app.healthHandler)
	mux.HandleFunc("/api/v1/device", app.deviceHandler)

	// start the server
	addr := ":8080"
	log.Printf("Starting web server on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("failed to start web server: %v", err)
	}
}
