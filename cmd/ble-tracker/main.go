// Command ble-tracker connects to a GoCube and shows all messages with cube state tracking.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/seamusw/gocube/internal/ble"
	"github.com/seamusw/gocube/internal/cube"
	"github.com/seamusw/gocube/internal/gocube"
)

func main() {
	fmt.Println("GoCube Tracker Tool - Shows all messages with cube state tracking")
	fmt.Println("==================================================================")

	client, err := ble.NewClient()
	if err != nil {
		fmt.Printf("Failed to create BLE client: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create cube tracker
	tracker := cube.NewTracker()
	tracker.SetPhaseCallback(func(phase cube.DetectedPhase, phaseKey string) {
		fmt.Printf("\n>>> PHASE CHANGE: %s <<<\n\n", phaseKey)
	})

	// Set up message handler BEFORE connecting
	client.SetMessageCallback(func(msg *gocube.Message) {
		handleMessage(msg, tracker)
	})

	fmt.Println("Scanning for GoCube...")
	if err := client.ConnectFirst(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connected to: %s\n", client.DeviceName())
	fmt.Println("\nMake moves on the cube - you'll see rotation messages and cube state.")
	fmt.Println("Rotate the cube (without making moves) to see orientation messages.")
	fmt.Println("Press Ctrl+C to exit.\n")
	fmt.Println("Cube starts SOLVED. Make moves to see phase detection in action.\n")
	fmt.Println(strings.Repeat("-", 70))

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nDisconnecting...")
	client.Disconnect()
}

func handleMessage(msg *gocube.Message, tracker *cube.Tracker) {
	timestamp := time.Now().Format("15:04:05.000")

	switch msg.Type {
	case gocube.MsgTypeRotation:
		rotations, err := gocube.DecodeRotation(msg.Payload)
		if err != nil {
			fmt.Printf("[%s] ROTATION ERROR: %v\n", timestamp, err)
			return
		}

		moves := gocube.RotationsToMoves(rotations, 0)
		var moveStrs []string
		for _, m := range moves {
			moveStrs = append(moveStrs, m.Notation())

			// Apply to tracker
			tracker.ApplyMove(m)
		}

		phase := tracker.CurrentPhaseKey()
		progress := tracker.GetProgress()

		fmt.Printf("[%s] MOVE: %-6s | Phase: %-18s | Solved: %v\n",
			timestamp, strings.Join(moveStrs, " "), phase, tracker.IsSolved())
		fmt.Printf("         Cross: %v | TopLayer: %v | Middle: %v | BotCross: %v | CornersPos: %v | CornersOri: %v\n",
			progress.WhiteCross, progress.TopLayer, progress.MiddleLayer,
			progress.BottomCross, progress.CornersPositioned, progress.CornersOriented)

	case gocube.MsgTypeOrientation:
		orient, err := gocube.DecodeOrientation(msg.Payload)
		if err != nil {
			fmt.Printf("[%s] ORIENTATION ERROR: %v\n", timestamp, err)
			return
		}
		fmt.Printf("[%s] ORIENT: x=%.3f y=%.3f z=%.3f w=%.3f\n",
			timestamp, orient.X, orient.Y, orient.Z, orient.W)

	case gocube.MsgTypeBattery:
		battery, err := gocube.DecodeBattery(msg.Payload)
		if err != nil {
			fmt.Printf("[%s] BATTERY ERROR: %v\n", timestamp, err)
			return
		}
		fmt.Printf("[%s] BATTERY: %d%%\n", timestamp, battery.Level)

	case gocube.MsgTypeCubeType:
		cubeType, err := gocube.DecodeCubeType(msg.Payload)
		if err != nil {
			fmt.Printf("[%s] CUBE_TYPE ERROR: %v\n", timestamp, err)
			return
		}
		fmt.Printf("[%s] CUBE_TYPE: %s (0x%02X)\n", timestamp, cubeType.TypeName, cubeType.TypeCode)

	case gocube.MsgTypeOfflineStats:
		stats, err := gocube.DecodeOfflineStats(msg.Payload)
		if err != nil {
			fmt.Printf("[%s] OFFLINE_STATS ERROR: %v\n", timestamp, err)
			return
		}
		fmt.Printf("[%s] OFFLINE_STATS: %d moves, %d sec, %d solves\n",
			timestamp, stats.Moves, stats.Time, stats.Solves)

	case gocube.MsgTypeState:
		fmt.Printf("[%s] STATE: %d bytes - %X\n", timestamp, len(msg.Payload), msg.Payload)

	default:
		fmt.Printf("[%s] UNKNOWN (0x%02X): %X\n", timestamp, msg.Type, msg.Payload)
	}
}
