package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/Useles5/go-archerC5/pkg/archerC5"
)

func main() {
	password := os.Getenv("ROUTER_PASS")
	if password == "" {
		log.Fatal("ROUTER_PASS environment variable is not set")
	}

	client, err := archerC5.NewClient(password, archerC5.DefaultRouterIP)
	if err != nil {
		if errors.Is(err, archerC5.ErrAuthFailed) {
			fmt.Println("Wrong Password!")
			os.Exit(1)
		}

		fmt.Printf("Error logging in: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Authenticated successfully!")
	fmt.Printf("Session established with: %s\n", client.BaseURL)
}
