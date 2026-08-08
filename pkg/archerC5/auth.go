package archerC5

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

func Authenticate(password string) error {
	// Custom client
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, "http://tplinkwifi.net/cgi/getParm", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set Required Headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "http://tplinkwifi.net/")
	req.Header.Set("Accept", "*/*")

	// Send Req
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the entire response into memory
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	bodyString := string(bodyBytes)

	// Regex
	eeRegex := regexp.MustCompile(`var ee="([^"]+)"`)
	nnRegex := regexp.MustCompile(`var nn="([^"]+)"`)

	// Search the text
	eeMatch := eeRegex.FindStringSubmatch(bodyString)
	nnMatch := nnRegex.FindStringSubmatch(bodyString)

	// Check if found match
	if len(eeMatch) < 2 || len(nnMatch) < 2 {
		return errors.New("failed to find N and E in router response")
	}

	exponentHex := eeMatch[1]
	modulusHex := nnMatch[1]

	encryptedPayload, err := EncryptCredentials(password, modulusHex, exponentHex)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}
	fmt.Printf("Success! Encrypted Payload generated: %s...\n", encryptedPayload[:30])
	return nil

}
