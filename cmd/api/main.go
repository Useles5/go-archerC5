package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Useles5/go-archerC5/pkg/archerC5"
)

func main() {
	pass := os.Getenv("ROUTER_PASS")
	if pass == "" {
		log.Fatal("$ROUTER_PASS env variable is not set")
	}

	// initialize
	routerClient, err := archerC5.NewClient(pass, archerC5.DefaultRouterIP)
	if err != nil {
		log.Fatalf("failed to initialize router client: %v", err)
	}

	// inject
	app := &application{
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
		Addr:    ":8080",
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
