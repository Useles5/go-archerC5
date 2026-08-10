package archerC5

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type WirelessRadio struct {
	BSSID             string
	SSID              string
	Band              string // X_TP_Band
	Channel           int
	Enable            bool
	AutoChannelEnable bool
}

func (c *RouterClient) GetWirelessRadios() ([]WirelessRadio, error) {

	url := c.BaseURL + "/cgi?5="

	payload := "[LAN_WLAN#0,0,0,0,0,0#0,0,0,0,0,0]0,8\r\n" +
		"Enable\r\n" +
		"BSSID\r\n" +
		"SSID\r\n" +
		"X_TP_Band\r\n" +
		"Channel\r\n" +
		"AutoChannelEnable\r\n" +
		"BasicEncryptionModes\r\n" +
		"BeaconType\r\n"

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create wireless radio request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Origin", c.BaseURL)
	req.Header.Set("Referer", c.BaseURL+"/")
	req.Header.Set("TokenID", c.TokenID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send wireless radio request: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return parseWirelessRadio(string(respBodyBytes))

}

func parseWirelessRadio(respBody string) ([]WirelessRadio, error) {
	var radios []WirelessRadio
	var currentRadio *WirelessRadio

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
			if currentRadio != nil {
				radios = append(radios, *currentRadio)
			}

			// Create a fresh WirelessRadio struct for the next block of data
			currentRadio = &WirelessRadio{}
			continue
		}

		if currentRadio == nil {
			continue
		}

		// Guard: If the line isn't a key=value pair, skip it.
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "BSSID":
			currentRadio.BSSID = value
		case "SSID":
			currentRadio.SSID = value
		case "X_TP_Band":
			currentRadio.Band = value
		case "channel":
			currentRadio.Channel, _ = strconv.Atoi(value)
		case "enable":
			currentRadio.Enable = (value == "1")
		case "autoChannelEnable":
			currentRadio.AutoChannelEnable = (value == "1")
		}

	}

	// Catch the very last radio in the loop before it hit the [error]0 line
	if currentRadio != nil {
		radios = append(radios, *currentRadio)
	}

	return radios, nil

}
