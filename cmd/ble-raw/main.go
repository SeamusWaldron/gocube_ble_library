// BLE Raw Data Debug - shows raw data from GoCube
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"tinygo.org/x/bluetooth"
)

// GoCube UUIDs
var (
	serviceUUIDStr = "6e400001-b5a3-f393-e0a9-e50e24dcca9e"
	txCharUUIDStr  = "6e400003-b5a3-f393-e0a9-e50e24dcca9e" // Notify (cube -> us)
	rxCharUUIDStr  = "6e400002-b5a3-f393-e0a9-e50e24dcca9e" // Write (us -> cube)
)

func main() {
	fmt.Println("BLE Raw Data Debug (Detailed)")
	fmt.Println("==============================")
	fmt.Println()

	adapter := bluetooth.DefaultAdapter
	err := adapter.Enable()
	if err != nil {
		fmt.Printf("Failed to enable adapter: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Scanning for GoCube...")

	var targetAddr bluetooth.Address
	var targetName string
	found := make(chan struct{})
	var foundOnce sync.Once

	go func() {
		adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			name := result.LocalName()
			if strings.HasPrefix(strings.ToLower(name), "gocube") {
				targetAddr = result.Address
				targetName = name
				foundOnce.Do(func() {
					close(found)
				})
			}
		})
	}()

	select {
	case <-found:
		adapter.StopScan()
	case <-time.After(10 * time.Second):
		adapter.StopScan()
		fmt.Println("GoCube not found")
		os.Exit(1)
	}

	// Give time for StopScan to take effect
	time.Sleep(100 * time.Millisecond)

	fmt.Printf("Found: %s (%s)\n", targetName, targetAddr.String())
	fmt.Println()

	fmt.Println("Connecting...")
	device, err := adapter.Connect(targetAddr, bluetooth.ConnectionParams{})
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected!")
	fmt.Println()

	// Discover ALL services
	fmt.Println("Discovering services...")
	services, err := device.DiscoverServices(nil)
	if err != nil {
		fmt.Printf("Failed to discover services: %v\n", err)
		device.Disconnect()
		os.Exit(1)
	}

	fmt.Printf("Found %d services:\n", len(services))
	for i, svc := range services {
		fmt.Printf("  [%d] %s\n", i, svc.UUID().String())
	}
	fmt.Println()

	// Find GoCube service
	var gocubeService bluetooth.DeviceService
	for _, svc := range services {
		uuidStr := strings.ToLower(svc.UUID().String())
		if strings.Contains(uuidStr, "6e400001") {
			gocubeService = svc
			fmt.Printf("Found GoCube service: %s\n", svc.UUID().String())
			break
		}
	}

	if gocubeService.UUID().String() == "00000000-0000-0000-0000-000000000000" {
		fmt.Println("GoCube service not found!")
		device.Disconnect()
		os.Exit(1)
	}

	// Discover characteristics
	fmt.Println()
	fmt.Println("Discovering characteristics...")
	chars, err := gocubeService.DiscoverCharacteristics(nil)
	if err != nil {
		fmt.Printf("Failed to discover characteristics: %v\n", err)
		device.Disconnect()
		os.Exit(1)
	}

	fmt.Printf("Found %d characteristics:\n", len(chars))
	var txChar bluetooth.DeviceCharacteristic
	for i, ch := range chars {
		uuidStr := strings.ToLower(ch.UUID().String())
		fmt.Printf("  [%d] %s\n", i, ch.UUID().String())

		if strings.Contains(uuidStr, "6e400003") {
			txChar = ch
			fmt.Printf("       ^ This is TX (notify)\n")
		}
		if strings.Contains(uuidStr, "6e400002") {
			fmt.Printf("       ^ This is RX (write)\n")
		}
	}
	fmt.Println()

	if txChar.UUID().String() == "00000000-0000-0000-0000-000000000000" {
		fmt.Println("TX characteristic not found!")
		device.Disconnect()
		os.Exit(1)
	}

	// Enable notifications
	fmt.Println("Enabling notifications on TX characteristic...")
	err = txChar.EnableNotifications(func(data []byte) {
		fmt.Printf("[RAW] %s\n", hex.EncodeToString(data))

		// Try to parse as GoCube message
		if len(data) >= 5 && data[0] == 0x2A {
			msgType := data[2]
			fmt.Printf("      Type: 0x%02X, Length: %d\n", msgType, data[1])
		}
	})
	if err != nil {
		fmt.Printf("Failed to enable notifications: %v\n", err)
		device.Disconnect()
		os.Exit(1)
	}
	fmt.Println("Notifications enabled!")
	fmt.Println()

	fmt.Println("Rotate the cube to see data...")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Keep running
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	select {
	case <-sigChan:
		fmt.Println("\nDisconnecting...")
	case <-ctx.Done():
		fmt.Println("\nTimeout, disconnecting...")
	}

	device.Disconnect()
}
