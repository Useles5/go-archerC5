package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Useles5/go-archerC5/pkg/archerC5"
)

func main() {
	// API Key generation
	var generateKey bool
	flag.BoolVar(&generateKey, "generate-key", false, "Generate an API key")
	flag.Parse()

	if generateKey {
		bytes := make([]byte, 32)
		_, err := rand.Read(bytes)
		if err != nil {
			log.Fatalf("Failed to generate an API key: %s", err)
		}

		// Use RawURLEncoding so the key is safe to use anywhere (URLs, headers, bash scripts).
		fmt.Println(base64.RawURLEncoding.EncodeToString(bytes))
		// os.Exit 0 -> Successful
		// os.Exit 1 -> Unsuccessful
		os.Exit(0)
	}

	// configuration parsing
	var cfg config

	// defaults
	cfg.port = 8080
	cfg.routerIP = archerC5.DefaultRouterIP

	if portStr := os.Getenv("PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)

		// check for error and range of provided port number
		// Start at 1; port 0 triggers OS dynamic port allocation.
		if err != nil || port < 1 || port > 65535 {
			log.Fatalf("Invalid PORT env variable '%s', Must be a number between 1 and 65535.", portStr)
		}
		cfg.port = port
	}

	if ipStr := os.Getenv("ROUTER_IP"); ipStr != "" {
		// check for valid IP
		if net.ParseIP(ipStr) == nil {
			log.Fatalf("Invalid ROUTER_IP env variable '%s'", ipStr)
		}
		cfg.routerIP = ipStr
	}

	// strictly required
	if cfg.routerPass = os.Getenv("ROUTER_PASS"); cfg.routerPass == "" {
		log.Fatal("$ROUTER_PASS env variable is not set")
	}

	if cfg.apiKey = os.Getenv("API_KEY"); cfg.apiKey == "" {
		log.Fatal("$API_KEY env variable is not set. Run with -generate-key to create one.")
	}

	// initialize
	routerClient, err := archerC5.NewClient(cfg.routerPass, cfg.routerIP)
	if err != nil {
		log.Fatalf("failed to initialize router client: %v", err)
	}

	// inject
	app := &application{
		config: cfg,
		client: routerClient,
	}

	// set up the router
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", app.healthHandler)
	mux.HandleFunc("/api/v1/devices", app.devicesHandler)
	mux.HandleFunc("/api/v1/devices/{mac}", app.deviceLookupHandler)
	mux.HandleFunc("/api/v1/reboot", app.rebootHandler)

	// server struct to manage its lifecycle
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.port),
		Handler: mux,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)

	go func() {
		log.Printf("Starting web server on http://localhost%s", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Server failed to start: %v", err)
		}
	case <-quit:
		log.Println("Received shutdown signal. Shutting down web server...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}

	log.Println("Logging out of TP-Link router...")
	if err := app.client.Logout(); err != nil {
		log.Printf("failed to logout of router gracefully: %v", err)
	}
	log.Println("Server gracefully stopped")
}
