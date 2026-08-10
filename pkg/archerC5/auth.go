package archerC5

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
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

	// Strip protocol
	routerIP = strings.TrimPrefix(routerIP, "http://")
	routerIP = strings.TrimPrefix(routerIP, "https://")

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

	// router attaches newline character(\n) at the end of response body
	// Trim it
	bodyString = strings.TrimSpace(string(loginBodyBytes))

	switch bodyString {
	case routerSuccess:
		homeReq, err := http.NewRequest(http.MethodGet, baseURL+"/", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create homepage request: %w", err)
		}

		homeReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		homeReq.Header.Set("Referer", baseURL+"/")

		homeResp, err := client.Do(homeReq)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch homepage: %w", err)
		}
		defer homeResp.Body.Close()

		homeBytes, err := io.ReadAll(homeResp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read homepage: %w", err)
		}

		// Looking for something like: var token = "1df27f60cf9ee8157bb3bb0bf06004";
		tokenRegex := regexp.MustCompile(`var\s+token\s*=\s*["']([^"']+)["']`)
		tokenMatch := tokenRegex.FindStringSubmatch(string(homeBytes))

		var extractedToken string
		if len(tokenMatch) >= 2 {
			extractedToken = tokenMatch[1]
		} else {
			return nil, errors.New("logged in successfully, but failed to find TokenID in homepage HTML")
		}

		return &RouterClient{
			BaseURL:    baseURL,
			httpClient: client,
			TokenID:    extractedToken,
		}, nil

	case routerErrAuthFailed:
		return nil, ErrAuthFailed
	case routerErrFormat:
		return nil, ErrFormat
	default:
		// Catch-all
		return nil, fmt.Errorf("%w: %s", ErrUnknownAPI, bodyString)
	}
}

// Logout terminates the current router session
func (c *RouterClient) Logout() error {

	// ACTION 8 -> ACT_CGI = 8 is the code for Logout
	// (╭ರ_•́) from proxy.js
	logoutURL := c.BaseURL + "/cgi?8"

	// Request Body Vanishes the moment the req is made
	// can't trace in network payload/request tab
	// use XHR Breakpoints in Debugger(FF)/Sources(Chrome) tab and set custom "/cgi" breakpoint
	// watch Scope tab and look for data that is being sent
	// Note: use breakpoint after logging in to avoid many /cgi calls
	payload := "[/cgi/logout#0,0,0,0,0,0#0,0,0,0,0,0]0,0\r\n"

	req, err := http.NewRequest(http.MethodPost, logoutURL, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create logout request: %w", err)
	}

	// Set Headers
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Origin", c.BaseURL)
	req.Header.Set("Referer", c.BaseURL+"/")
	req.Header.Set("TokenID", c.TokenID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send logout request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("logout failed with status %d", resp.StatusCode)
	}

	return nil
}
