// BLE Debug Scanner - scans for all BLE devices to help identify GoCube
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tinygo.org/x/bluetooth"
)

// GoCube service UUID
var gocubeServiceUUID = bluetooth.NewUUID([16]byte{
	0x6e, 0x40, 0x00, 0x01, 0xb5, 0xa3, 0xf3, 0x93,
	0xe0, 0xa9, 0xe5, 0x0e, 0x24, 0xdc, 0xca, 0x9e,
})

func main() {
	fmt.Println("BLE Debug Scanner for GoCube")
	fmt.Println("============================")
	fmt.Println()
	fmt.Println("IMPORTANT: Disconnect the GoCube from your iPhone first!")
	fmt.Println("  iPhone Settings > Bluetooth > GoCube > Forget This Device")
	fmt.Println()
	fmt.Println("Then rotate the cube to wake it up.")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop scanning...")
	fmt.Println()

	adapter := bluetooth.DefaultAdapter
	err := adapter.Enable()
	if err != nil {
		fmt.Printf("ERROR: Failed to enable Bluetooth adapter: %v\n", err)
		fmt.Println()
		fmt.Println("Try: System Settings > Privacy & Security > Bluetooth")
		fmt.Println("     Add Terminal (or your terminal app) to the allowed list")
		os.Exit(1)
	}

	fmt.Println("Bluetooth adapter enabled. Scanning for 60 seconds...")
	fmt.Println()
	fmt.Printf("%-40s %-25s %-6s %s\n", "ADDRESS/UUID", "NAME", "RSSI", "NOTES")
	fmt.Println(strings.Repeat("-", 90))

	seen := make(map[string]bool)
	foundGoCube := false

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println()
		printSummary(foundGoCube)
		adapter.StopScan()
		os.Exit(0)
	}()

	// Stop after 60 seconds
	go func() {
		time.Sleep(60 * time.Second)
		fmt.Println()
		fmt.Println("Scan timeout (60s).")
		printSummary(foundGoCube)
		adapter.StopScan()
		os.Exit(0)
	}()

	err = adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		addr := result.Address.String()

		// Skip duplicates
		if seen[addr] {
			return
		}
		seen[addr] = true

		name := result.LocalName()
		notes := ""

		// Check for GoCube by name
		nameLower := strings.ToLower(name)
		if strings.Contains(nameLower, "gocube") || strings.Contains(nameLower, "cube") {
			notes = "*** GOCUBE FOUND! ***"
			foundGoCube = true
		}

		// Check for GoCube by service UUID
		for _, uuid := range result.AdvertisementPayload.ServiceUUIDs() {
			if uuid == gocubeServiceUUID {
				notes = "*** GOCUBE (by UUID)! ***"
				foundGoCube = true
			}
		}

		if name == "" {
			name = "(no name)"
		}

		// Only show named devices or potential GoCubes to reduce noise
		if name != "(no name)" || notes != "" {
			fmt.Printf("%-40s %-25s %-6d %s\n", addr, truncate(name, 25), result.RSSI, notes)
		}
	})

	if err != nil {
		fmt.Printf("ERROR: Scan failed: %v\n", err)
		os.Exit(1)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func printSummary(found bool) {
	fmt.Println()
	if found {
		fmt.Println("SUCCESS: GoCube was detected!")
		fmt.Println()
		fmt.Println("Now run: ./gocube status")
	} else {
		fmt.Println("GoCube was NOT detected.")
		fmt.Println()
		fmt.Println("Troubleshooting:")
		fmt.Println("  1. Make sure GoCube is NOT connected to your iPhone")
		fmt.Println("     (Forget the device in iPhone Bluetooth settings)")
		fmt.Println("  2. Rotate the cube to wake it up")
		fmt.Println("  3. The cube's LED should be on/blinking")
		fmt.Println("  4. Try moving closer to your Mac")
		fmt.Println("  5. Try: sudo ./ble-debug")
	}
}
