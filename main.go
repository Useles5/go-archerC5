package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/Useles5/go-archerC5/pkg/archerC5"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	password := os.Getenv("ROUTER_PASS")
	if password == "" {
		log.Fatal("ROUTER_PASS environment variable is not set")
	}

	client, err := archerC5.NewClient(password, archerC5.DefaultRouterIP)
	if err != nil {
		if errors.Is(err, archerC5.ErrAuthFailed) {
			return errors.New("wrong password")
		}
		return fmt.Errorf("error logging in: %w", err)
	}

	fmt.Println("Authenticated successfully!")
	fmt.Printf("Session established with: %s\n", client.BaseURL)

	defer func() {
		fmt.Println("\nLogging out to free the router session...")
		if err := client.Logout(); err != nil {
			fmt.Printf("Failed to cleanly logout: %v\n", err)
		} else {
			fmt.Println("Logout successful!")
		}
	}()

	devices, err := client.GetConnectedDevices()
	if err != nil {
		return fmt.Errorf("failed to fetch devices: %w", err)
	}

	fmt.Printf("\n--- Connected Devices (%d found) ---\n", len(devices))
	for _, dev := range devices {
		status := "Offline"
		connection := "N/A"

		if dev.Active {
			// 0=Wired, 1=2.4GHz, 3=5.0GHz
			status = "Online"
			switch dev.ConnType {
			case "0":
				connection = "Wired"
			case "1":
				connection = "2.4 GHz"
			case "3":
				connection = "5.0 GHz"
			default:
				connection = fmt.Sprintf("Unknown (%s)", dev.ConnType)
			}
		}
		fmt.Printf("[%s] %s (%s) - %s [Conn: %s]\n", status, dev.HostName, dev.IPAddress, dev.MACAddress, connection)
	}

	radios, err := client.GetWirelessRadios()
	if err != nil {
		return fmt.Errorf("failed to fetch wireless radios: %w", err)
	}

	fmt.Printf("\n--- Wireless Radios (%d found) ---\n", len(radios))
	for _, r := range radios {
		status := "Disabled"
		if r.Enable {
			status = "Enabled"
		}

		auto := ""
		if r.AutoChannelEnable {
			auto = " (Auto)"
		}

		fmt.Printf("[%s] %s | %s | Channel: %d%s | MAC: %s\n",
			status, r.Band, r.SSID, r.Channel, auto, r.BSSID)
	}

	return nil
}
