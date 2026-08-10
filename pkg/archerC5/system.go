package archerC5

import (
	"fmt"
	"net/http"
	"strings"
)

func (c *RouterClient) Reboot() error {

	url := c.BaseURL + "/cgi?7"

	payload := "[ACT_REBOOT#0,0,0,0,0,0#0,0,0,0,0,0]0,0\r\n"

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create reboot request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Origin", c.BaseURL)
	req.Header.Set("Referer", c.BaseURL+"/")
	req.Header.Set("TokenID", c.TokenID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send reboot request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("reboot request failed: %s", resp.Status)
	}

	return nil
}
