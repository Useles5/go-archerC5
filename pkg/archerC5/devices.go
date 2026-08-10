package archerC5

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ConnectedDevice represents a single client on the router's network
type ConnectedDevice struct {
	IPAddress          string
	MACAddress         string
	HostName           string
	AddressSource      string
	LeaseTimeRemaining string
	ConnType           string // X_TP_ConnType
	Active             bool
}

// GetConnectedDevices queries the router's Data Model for the LAN_HOST_ENTRY table
func (c *RouterClient) GetConnectedDevices() ([]ConnectedDevice, error) {

	// ACTION 5 -> ACT_GL = 5 refers to Get List
	// (╭ರ_•́) from proxy.js
	url := c.BaseURL + "/cgi?5="

	// HTTP/1.1 protocols standard requires all text/plain to have \r\n as line breakers
	// exact payload required by the router
	payload := "[LAN_HOST_ENTRY#0,0,0,0,0,0#0,0,0,0,0,0]0,0\r\n"

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Origin", c.BaseURL)
	req.Header.Set("Referer", c.BaseURL+"/")
	req.Header.Set("TokenID", c.TokenID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	return parseDeviceList(string(respBodyBytes))
}

// parseDeviceList parses the received response
func parseDeviceList(respBody string) ([]ConnectedDevice, error) {
	var devices []ConnectedDevice
	var currentDevice *ConnectedDevice

	lines := strings.Split(respBody, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[error]") {
			// [error]0 = No error(Success)
			if line != "[error]0" {
				return nil, fmt.Errorf("router API returned a data error: %s", line)
			}
			break // End of list
		}

		if strings.HasPrefix(line, "[") {
			if currentDevice != nil {
				devices = append(devices, *currentDevice)
			}

			// Create a fresh device struct for the next block of data
			currentDevice = &ConnectedDevice{}
			continue
		}

		if currentDevice == nil {
			continue
		}

		// SplitN ensures we only split on the FIRST equals sign,
		// just in case a hostname contains an '=' character
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]

		switch key {
		case "IPAddress":
			currentDevice.IPAddress = value
		case "MACAddress":
			currentDevice.MACAddress = value
		case "hostName":
			currentDevice.HostName = value
		case "addressSource":
			currentDevice.AddressSource = value
		case "leaseTimeRemaining":
			currentDevice.LeaseTimeRemaining = value
		case "X_TP_ConnType":
			currentDevice.ConnType = value
		case "active":
			currentDevice.Active = (value == "1")
		}

	}
	// Catch the very last device in the loop before it hit the [error]0 line
	if currentDevice != nil {
		devices = append(devices, *currentDevice)
	}

	return devices, nil
}
