package archerC5

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
)

// Found in err.js
const (
	routerSuccess       = "$.ret=0;"     // NO_ERROR
	routerErrFormat     = "$.ret=71011;" // HTTP_ERR_FORMAT
	routerErrAuthFailed = "$.ret=71233;" // ERR_HTTP_ERR_USER_PWD_NOT_CORRECT
)

// Sentinel errors
var (
	ErrAuthFailed = errors.New("authentication failed: incorrect password")
	ErrFormat     = errors.New("request format error")
	ErrUnknownAPI = errors.New("router returned an unknown response code")
)

func NewClient(password, routerIP string) (*RouterClient, error) {

	baseURL := fmt.Sprintf("http://%s", routerIP)

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %v", err)
	}
	// Custom client
	client := &http.Client{
		Jar: jar,
	}

	req, err := http.NewRequest(http.MethodGet, baseURL+"/cgi/getParm", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Required Headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", baseURL+"/")
	req.Header.Set("Accept", "*/*")

	// Send Req
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the entire response into memory
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
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
		return nil, errors.New("failed to find N and E in router response")
	}

	exponentHex := eeMatch[1]
	modulusHex := nnMatch[1]

	encryptedUsername, err := EncryptPayload("admin", modulusHex, exponentHex)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt username: %w", err)
	}
	encryptedPassword, err := EncryptPayload(password, modulusHex, exponentHex)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Construct the payload
	// url.Values automatically handles url encoding

	// Had to abandon because url.Values automatically sorts keys in alphabetical order
	// Router needs the queryString in correct Order unlike modern web servers

	//formData := url.Values{}
	//formData.Set("UserName", encryptedUsername)
	//formData.Set("Passwd", encryptedPassword)
	//formData.Set("Action", "1")
	//formData.Set("LoginStatus", "0")

	//queryString := formData.Encode()

	queryString := fmt.Sprintf("UserName=%s&Passwd=%s&Action=1&LoginStatus=0",
		url.QueryEscape(encryptedUsername),
		url.QueryEscape(encryptedPassword),
	)

	fullURL := fmt.Sprintf("%s/cgi/login?%s", baseURL, queryString)

	// Data is in URL, no body is needed so nil
	loginReq, err := http.NewRequest(http.MethodPost, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Headers
	loginReq.Header.Set("Origin", baseURL)
	loginReq.Header.Set("Referer", baseURL+"/")
	loginReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	loginReq.Header.Set("Accept", "*/*")

	loginRes, err := client.Do(loginReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer loginRes.Body.Close()

	if loginRes.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed with status %d", loginRes.StatusCode)
	}

	loginBodyBytes, err := io.ReadAll(loginRes.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	//fmt.Printf("Login Response: %s\n", string(loginBodyBytes))

	bodyString = string(loginBodyBytes)

	switch bodyString {
	case routerSuccess:
		// nothing to do
	case routerErrAuthFailed:
		return nil, ErrAuthFailed
	case routerErrFormat:
		return nil, ErrFormat
	default:
		// Catch-all
		return nil, fmt.Errorf("%w: %s", ErrUnknownAPI, bodyString)
	}
	return &RouterClient{
		BaseURL:    baseURL,
		httpClient: client,
	}, nil

}
