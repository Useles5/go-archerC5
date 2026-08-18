package main

import (
	"net/http"
	"strings"

	"github.com/Useles5/go-archerC5/pkg/archerC5"
)

// config holds all the configurations for the application
type config struct {
	port       int
	routerIP   string
	routerPass string
	apiKey     string
}

// application holds all dependencies for the http handlers.
type application struct {
	config config
	client *archerC5.RouterClient
}

// healthHandler proves the server is running and is reachable.
func (app *application) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		app.writeJSON(w, http.StatusMethodNotAllowed, nil, "Method is not allowed. Please use GET")
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

// devicesHandler will fetch all devices and apply status query filters.
func (app *application) devicesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		// according to HTTP spec, if you tell a client a method is not allowed
		// you are technically required to tell them which methods are allowed
		w.Header().Set("Allow", http.MethodGet)
		app.writeJSON(w, http.StatusMethodNotAllowed, nil, "Method is not allowed. Please use GET")
		return
	}

	// fetch all connected devices
	allDevices, err := app.client.GetConnectedDevices()
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, nil, "Failed to fetch connected devices")
		return
	}

	statusFilter := r.URL.Query().Get("status")

	// strict input validation
	if statusFilter != "" && statusFilter != "online" && statusFilter != "offline" {
		app.writeJSON(w, http.StatusBadRequest, nil, "Invalid status filter. Allowed values: 'online' or 'offline'")
		return
	}

	// initialize empty slice to prevent JSON 'null', sized to capacity to avoid resizing
	filteredDevices := make([]archerC5.ConnectedDevice, 0, len(allDevices))

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

// deviceLookupHandler fetches a single device by its MAC address.
func (app *application) deviceLookupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		app.writeJSON(w, http.StatusMethodNotAllowed, nil, "Method is not allowed. Please use GET")
		return
	}
	macParam := r.PathValue("mac")

	// from Go 1.22+, router prevents empty params
	//if macParam == "" {
	//	app.writeJSON(w, http.StatusBadRequest, nil, "MAC address is required")
	//	return
	//}

	// fetch all connected devices
	allDevices, err := app.client.GetConnectedDevices()
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, nil, "Failed to fetch connected devices")
		return
	}

	for _, device := range allDevices {
		// strings.EqualFold does a case-insensitive comparison (e.g., AA:BB == aa:bb)
		if strings.EqualFold(device.MACAddress, macParam) {
			app.writeJSON(w, http.StatusOK, device, "Successfully fetched requested device")
			return
		}
	}

	app.writeJSON(w, http.StatusNotFound, nil, "Device not found")
}

func (app *application) rebootHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		app.writeJSON(w, http.StatusMethodNotAllowed, nil, "Method is not allowed. Please use POST")
		return
	}

	err := app.client.Reboot()
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, nil, "Failed to reboot")
		return
	}

	app.writeJSON(w, http.StatusAccepted, nil, "Rebooting now. Network will be offline till rebooting is complete(approx 2 min")
}
