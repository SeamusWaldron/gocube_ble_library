// Package gocube provides a Go library for interacting with GoCube smart
// Rubik's cubes via Bluetooth Low Energy (BLE).
//
// The library supports:
//   - Device discovery and connection
//   - Real-time move tracking with timestamps
//   - Cube state simulation and phase detection
//   - Orientation tracking via quaternion conversion
//
// # Basic Usage
//
// Connect to a GoCube and track moves:
//
//	client, _ := gocube.NewClient()
//	defer client.Disconnect()
//
//	ctx := context.Background()
//	results, _ := client.Scan(ctx, 5*time.Second)
//	client.Connect(ctx, results[0].UUID)
//
//	client.SetMessageCallback(func(msg *gocube.Message) {
//	    if msg.Type == gocube.MsgTypeRotation {
//	        rotations, _ := gocube.DecodeRotation(msg.Payload)
//	        moves := gocube.RotationsToMoves(rotations, 0)
//	        for _, m := range moves {
//	            fmt.Println(m.Notation())
//	        }
//	    }
//	})
//
// # Cube State Tracking
//
// Track the cube state and detect solving phases:
//
//	tracker := gocube.NewTracker()
//	tracker.SetPhaseCallback(func(phase gocube.DetectedPhase, key string) {
//	    fmt.Printf("Completed: %s\n", key)
//	})
//
//	tracker.ApplyMove(gocube.Move{Face: gocube.FaceR, Turn: gocube.TurnCW})
//	fmt.Println(tracker.IsSolved())
//
// # Message Types
//
// The library decodes these message types from the GoCube:
//   - Rotation: Face turns with direction
//   - Orientation: Quaternion orientation data
//   - Battery: Battery level percentage
//   - CubeType: Device model information
//   - OfflineStats: Stored move statistics
package gocube
