// Package main demonstrates basic GoCube BLE connection.
//
// This example shows how to:
//   - Scan for nearby GoCube devices
//   - Connect to the first device found
//   - Set up event handlers for moves and phase changes
//   - Handle disconnection gracefully
//
// Usage:
//
//	go run main.go
//
// Make sure your GoCube is:
//   - Disconnected from your phone (Bluetooth settings > Forget Device)
//   - Awake (rotate it to wake)
//   - Within Bluetooth range
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SeamusWaldron/gocube_ble_library"
)

func main() {
	// Create a context that we can cancel on Ctrl+C.
	// This allows graceful shutdown of BLE scanning and connection.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown.
	// When the user presses Ctrl+C, we'll clean up properly.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("GoCube Connect Example")
	fmt.Println("======================")
	fmt.Println()
	fmt.Println("Scanning for GoCube devices...")
	fmt.Println("(Make sure your cube is awake and not connected to another device)")
	fmt.Println()

	// ConnectFirst is a convenience function that:
	// 1. Scans for nearby GoCube devices
	// 2. Connects to the first one found
	// 3. Returns a ready-to-use GoCube instance
	//
	// For more control, you can use Scan() and Connect() separately.
	cube, err := gocube.ConnectFirst(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		fmt.Println()
		fmt.Println("Troubleshooting tips:")
		fmt.Println("  1. Disconnect the cube from your phone's Bluetooth settings")
		fmt.Println("  2. Rotate the cube to wake it up")
		fmt.Println("  3. Try running the scan again")
		os.Exit(1)
	}

	// Always close the connection when done.
	// This releases BLE resources and allows the cube to connect elsewhere.
	defer cube.Close()

	fmt.Printf("Connected to: %s\n", cube.DeviceName())
	fmt.Printf("Battery: %d%%\n", cube.Battery())
	fmt.Println()
	fmt.Println("Cube is ready! Try making some moves...")
	fmt.Println("Press Ctrl+C to disconnect and exit.")
	fmt.Println()

	// Set up the move callback.
	// This function is called every time you rotate a face on the cube.
	// The Move struct contains:
	//   - Face: which face was turned (R, L, U, D, F, B)
	//   - Turn: direction/amount (CW, CCW, or Double)
	//   - Time: when the move occurred
	cube.OnMove(func(m gocube.Move) {
		// Notation() returns standard cube notation like "R", "R'", "R2"
		fmt.Printf("  Move: %s\n", m.Notation())
	})

	// Set up the phase change callback.
	// Phase detection tracks your progress through the layer-by-layer method.
	// Phases progress: scrambled → white_cross → first_layer → ... → solved
	cube.OnPhaseChange(func(p gocube.Phase) {
		fmt.Printf("  Phase: %s\n", p.String())
	})

	// Set up the solved callback.
	// This is triggered when the cube reaches the solved state.
	cube.OnSolved(func() {
		fmt.Println()
		fmt.Println("🎉 Cube solved!")
		fmt.Println()
	})

	// Set up the disconnect callback.
	// This is called if the BLE connection is lost unexpectedly.
	cube.OnDisconnect(func(err error) {
		if err != nil {
			fmt.Printf("\nDisconnected with error: %v\n", err)
		} else {
			fmt.Println("\nDisconnected")
		}
		cancel() // Cancel context to exit the program
	})

	// Set up battery level callback.
	// Battery updates are sent periodically by the cube.
	cube.OnBattery(func(level int) {
		fmt.Printf("  Battery: %d%%\n", level)
	})

	// Keep the program running until:
	// - User presses Ctrl+C (SIGINT/SIGTERM)
	// - Connection is lost
	// - Context is cancelled
	select {
	case <-sigChan:
		fmt.Println("\nShutting down...")
	case <-ctx.Done():
		// Context was cancelled (e.g., by disconnect handler)
	}

	// Print final statistics
	fmt.Println()
	fmt.Printf("Total moves made: %d\n", len(cube.Moves()))
	fmt.Printf("Final state: %s\n", cube.Phase().String())
	fmt.Println()

	// Calculate and display solve time if applicable
	moves := cube.Moves()
	if len(moves) >= 2 {
		duration := moves[len(moves)-1].Time.Sub(moves[0].Time)
		fmt.Printf("Session duration: %s\n", duration.Round(time.Millisecond))
	}
}
