package archerC5

import "net/http"

// DefaultRouterIP is the standard fallback for TP-Link routers
const DefaultRouterIP = "tplinkwifi.net"

// RouterClient holds the authenticated session and base configuration
// for future interactions
type RouterClient struct {
	BaseURL    string
	httpClient *http.Client
	TokenID    string
}
