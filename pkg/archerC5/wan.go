package archerC5

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// WANStatus represents the active Internet connection details
type WANStatus struct {
	MACAddress   string
	IPAddress    string
	SubnetMask   string
	Gateway      string
	PrimaryDNS   string
	SecondaryDNS string
	ConnType     string
	Status       string
}

// GetWANStatus fetches the router's internet connection settings
func (c *RouterClient) GetWANStatus() (*WANStatus, error) {
	url := c.BaseURL + "/cgi?6=&6=&6=&6=&6=&6="

	payload := "[WAN_IP_CONN#0,0,0,0,0,0#2,0,0,0,0,0]0,0\r\n" +
		"[WAN_PPP_CONN#0,0,0,0,0,0#2,0,0,0,0,0]1,0\r\n" +
		"[WAN_IP_CONN#0,0,0,0,0,0#1,0,0,0,0,0]2,0\r\n" +
		"[WAN_PPP_CONN#0,0,0,0,0,0#1,0,0,0,0,0]3,0\r\n" +
		"[WAN_L2TP_CONN#0,0,0,0,0,0#1,0,0,0,0,0]4,0\r\n" +
		"[WAN_PPTP_CONN#0,0,0,0,0,0#1,0,0,0,0,0]5,0\r\n"

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create WAN status request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Origin", c.BaseURL)
	req.Header.Set("Referer", c.BaseURL+"/")
	req.Header.Set("TokenID", c.TokenID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send WAN status request: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return parseWANStatus(string(respBodyBytes))
}

func parseWANStatus(s string) (*WANStatus, error) {
	var activeWAN *WANStatus
	var currentWAN *WANStatus

	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[error]") {
			if line != "[error]0" {
				return nil, fmt.Errorf("router API returned a data error: %s", line)
			}
			break
		}

		if strings.HasPrefix(line, "[") {
			if currentWAN != nil && currentWAN.Status == "Connected" {
				activeWAN = currentWAN
			}

			currentWAN = &WANStatus{}
			continue
		}

		if currentWAN == nil {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "MACAddress":
			currentWAN.MACAddress = value
		case "externalIPAddress":
			currentWAN.IPAddress = value
		case "subnetMask":
			currentWAN.SubnetMask = value
		case "defaultGateway":
			currentWAN.Gateway = value
		case "DNSServers":
			// The router sends DNS as "8.8.8.8,103.87.164.61"  -> PriDNS,SecDns
			dnsParts := strings.Split(value, ",")
			if len(dnsParts) >= 1 {
				currentWAN.PrimaryDNS = dnsParts[0]
			}
			if len(dnsParts) >= 2 {
				currentWAN.SecondaryDNS = dnsParts[1]
			}
		case "transportType":
			currentWAN.ConnType = value
		case "connectionStatus":
			currentWAN.Status = value
		}
	}

	// Catch the last block if it was the active one
	if currentWAN != nil && currentWAN.Status == "Connected" {
		activeWAN = currentWAN
	}

	// graceful fallback
	if activeWAN == nil {
		return &WANStatus{
			Status:    "Disconnected",
			ConnType:  "Unknown",
			IPAddress: "0.0.0.0",
		}, nil
	}

	// condition found in a frontend HTML response
	if activeWAN.ConnType == "PPPoE" && activeWAN.SubnetMask == "" {
		activeWAN.SubnetMask = "255.255.255.255"
	}

	return activeWAN, nil
}
