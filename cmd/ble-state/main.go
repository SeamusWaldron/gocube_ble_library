// BLE State Debug - requests and displays cube state
package main

import (
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

func main() {
	fmt.Println("BLE State Debug")
	fmt.Println("================")
	fmt.Println()
	fmt.Println("This tool requests the cube state and shows the raw data.")
	fmt.Println("Please have the cube in a SOLVED state first, then we'll scramble it.")
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

	time.Sleep(100 * time.Millisecond)
	fmt.Printf("Found: %s\n", targetName)

	fmt.Println("Connecting...")
	device, err := adapter.Connect(targetAddr, bluetooth.ConnectionParams{})
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer device.Disconnect()

	// Discover services
	services, err := device.DiscoverServices(nil)
	if err != nil {
		fmt.Printf("Failed to discover services: %v\n", err)
		os.Exit(1)
	}

	var gocubeService bluetooth.DeviceService
	for _, svc := range services {
		if strings.Contains(strings.ToLower(svc.UUID().String()), "6e400001") {
			gocubeService = svc
			break
		}
	}

	chars, err := gocubeService.DiscoverCharacteristics(nil)
	if err != nil {
		fmt.Printf("Failed to discover characteristics: %v\n", err)
		os.Exit(1)
	}

	var txChar, rxChar bluetooth.DeviceCharacteristic
	for _, ch := range chars {
		uuidStr := strings.ToLower(ch.UUID().String())
		if strings.Contains(uuidStr, "6e400003") {
			txChar = ch
		}
		if strings.Contains(uuidStr, "6e400002") {
			rxChar = ch
		}
	}

	fmt.Println("Connected!")
	fmt.Println()

	// Store received messages
	var lastStateData []byte
	var mu sync.Mutex

	// Enable notifications
	err = txChar.EnableNotifications(func(data []byte) {
		mu.Lock()
		defer mu.Unlock()

		if len(data) >= 3 && data[0] == 0x2A {
			msgType := data[2]

			// Parse length correctly
			length := int(data[1])
			checksumIdx := length - 1
			payload := data[3:checksumIdx]

			switch msgType {
			case 0x01: // Rotation
				fmt.Printf("[MOVE] %s\n", hex.EncodeToString(payload))
			case 0x02: // State
				lastStateData = make([]byte, len(payload))
				copy(lastStateData, payload)
				fmt.Printf("[STATE] Length: %d bytes\n", len(payload))
				fmt.Printf("        Hex: %s\n", hex.EncodeToString(payload))

				// Try to interpret the state
				if len(payload) >= 54 {
					fmt.Println("        Possible facelet interpretation:")
					// Each byte might represent a facelet color (0-5)
					faces := []string{"U", "R", "F", "D", "L", "B"}
					for f := 0; f < 6 && f*9 < len(payload); f++ {
						start := f * 9
						end := start + 9
						if end > len(payload) {
							end = len(payload)
						}
						faceletBytes := payload[start:end]
						var facelets []string
						for _, b := range faceletBytes {
							if b < 6 {
								facelets = append(facelets, fmt.Sprintf("%d", b))
							} else {
								facelets = append(facelets, "?")
							}
						}
						fmt.Printf("        %s: %v\n", faces[f], facelets)
					}
				}
			case 0x03: // Orientation - skip (too noisy)
			case 0x05: // Battery
				if len(payload) > 0 {
					fmt.Printf("[BATTERY] %d%%\n", payload[0])
				}
			default:
				fmt.Printf("[TYPE 0x%02X] %s\n", msgType, hex.EncodeToString(payload))
			}
		}
	})
	if err != nil {
		fmt.Printf("Failed to enable notifications: %v\n", err)
		os.Exit(1)
	}

	// Build command to request state
	// Format: [0x2A] [length] [cmd] [checksum] [0x0D] [0x0A]
	cmdRequestState := byte(0x33)
	length := byte(0x04) // 4 bytes after length: cmd + checksum + suffix
	checksum := byte(0x2A) + length + cmdRequestState
	stateCmd := []byte{0x2A, length, cmdRequestState, checksum, 0x0D, 0x0A}

	fmt.Println("Requesting cube state...")
	_, err = rxChar.WriteWithoutResponse(stateCmd)
	if err != nil {
		fmt.Printf("Failed to request state: %v\n", err)
	}

	fmt.Println()
	fmt.Println("Listening for messages... (rotate the cube to see moves)")
	fmt.Println("Press 'r' + Enter to request state again")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Input goroutine
	go func() {
		var input string
		for {
			fmt.Scanln(&input)
			if strings.ToLower(input) == "r" {
				fmt.Println("Requesting state...")
				rxChar.WriteWithoutResponse(stateCmd)
			}
		}
	}()

	<-sigChan
	fmt.Println("\nDisconnecting...")
}
