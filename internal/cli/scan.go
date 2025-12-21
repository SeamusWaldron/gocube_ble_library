package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/SeamusWaldron/gocube_ble_library"
)

// ScanForGoCube scans for GoCube devices using the same logic everywhere.
// It performs a single 5-second scan which is sufficient for macOS BLE discovery.
func ScanForGoCube() (*gocube.Client, []gocube.ScanResult, error) {
	fmt.Println("Scanning for GoCube devices...")

	client, err := gocube.NewClient()
	if err != nil {
		return nil, nil, fmt.Errorf("BLE not available: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := client.Scan(ctx, 5*time.Second)
	if err != nil {
		return client, nil, fmt.Errorf("scan failed: %w", err)
	}

	if len(results) == 0 {
		return client, nil, nil
	}

	fmt.Printf("Found: %s\n", results[0].Name)
	return client, results, nil
}

// ScanForGoCubeWithRetry scans for GoCube devices with retries.
// Uses the same 5-second scan as status, with up to maxAttempts retries.
func ScanForGoCubeWithRetry(maxAttempts int) (*gocube.Client, []gocube.ScanResult, error) {
	var client *gocube.Client
	var results []gocube.ScanResult
	var err error

	fmt.Println("Scanning for GoCube devices...")

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client, err = gocube.NewClient()
		if err != nil {
			return nil, nil, fmt.Errorf("BLE not available: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		results, err = client.Scan(ctx, 5*time.Second)
		cancel()

		if err != nil {
			fmt.Printf("Scan %d failed: %v\n", attempt, err)
			continue
		}

		if len(results) > 0 {
			fmt.Printf("Found: %s\n", results[0].Name)
			return client, results, nil
		}

		if attempt < maxAttempts {
			fmt.Printf("Scan %d: No devices found, retrying...\n", attempt)
		}
	}

	return client, nil, nil
}
