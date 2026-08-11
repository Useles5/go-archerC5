package archerC5

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// LANStatus represents the local network configuration (IPv4 & IPv6)
type LANStatus struct {
	MACAddress    string
	IPAddress     string
	SubnetMask    string
	IPv6Address   string
	IPv6PrefixLen string
	IPv6Assigned  string
	DHCPOn        bool
}

func (c *RouterClient) GetLANStatus() (*LANStatus, error) {

	url := c.BaseURL + "/cgi?1&6&1&6"

	payload := "[LAN_HOST_CFG#1,0,0,0,0,0#0,0,0,0,0,0]0,0\r\n" +
		"[LAN_IP_INTF#0,0,0,0,0,0#1,0,0,0,0,0]1,3\r\n" +
		"IPInterfaceIPAddress\r\n" +
		"IPInterfaceSubnetMask\r\n" +
		"X_TP_MACAddress\r\n" +
		"[LAN_IP6_HOST_CFG#1,0,0,0,0,0#0,0,0,0,0,0]2,0\r\n" +
		"[LAN_IP6_INTF#0,0,0,0,0,0#1,0,0,0,0,0]3,3\r\n" +
		"IPv6InterfaceAddress\r\n" +
		"IPv6InterfaceAddressPrefixLength\r\n" +
		"IPv6InterfaceAddressingType\r\n"

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create LAN status request: %v", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Origin", c.BaseURL)
	req.Header.Set("Referer", c.BaseURL+"/")
	req.Header.Set("TokenID", c.TokenID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send LAN status request: %v", err)
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read LAN status response: %v", err)
	}

	return parseLANStatus(string(respBodyBytes))
}

func parseLANStatus(respBody string) (*LANStatus, error) {

	lan := &LANStatus{}

	lines := strings.Split(respBody, "\n")
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
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "X_TP_MACAddress":
			lan.MACAddress = value
		case "IPInterfaceIPAddress":
			lan.IPAddress = value
		case "IPInterfaceSubnetMask":
			lan.SubnetMask = value
		case "DHCPServerEnable":
			lan.DHCPOn = (value == "1")
		case "IPv6InterfaceAddress":
			// if the field is blank, fronted defaults it to "::"
			if value == "" {
				lan.IPv6Address = "::"
			} else {
				lan.IPv6Address = value
			}
		case "IPv6SitePrefixLength":
			lan.IPv6PrefixLen = value
		case "IPv6SitePrefixConfigType":
			lan.IPv6Assigned = value
		}
	}

	return lan, nil
}
