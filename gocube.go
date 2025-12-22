// Package gocube provides a Go library for interacting with GoCube smart
// Rubik's cubes via Bluetooth Low Energy (BLE).
//
// # Features
//
//   - Device discovery and connection
//   - Real-time move tracking with timestamps
//   - Cube state simulation (works standalone without BLE)
//   - Automatic solving phase detection
//   - Orientation tracking
//
// # Quick Start
//
// Connect to a GoCube and track moves:
//
//	ctx := context.Background()
//	cube, err := gocube.ConnectFirst(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cube.Close()
//
//	cube.OnMove(func(m gocube.Move) {
//	    fmt.Println("Move:", m.Notation())
//	})
//
//	cube.OnPhaseChange(func(p gocube.Phase) {
//	    fmt.Println("Phase completed:", p.DisplayName())
//	})
//
//	// Keep running...
//	select {}
//
// # Standalone Cube Simulation
//
// The Cube type can be used without a BLE connection:
//
//	cube := gocube.NewCube()
//
//	// Apply moves using predefined constants
//	cube.Apply(gocube.R, gocube.U, gocube.RPrime, gocube.UPrime)
//
//	// Or from notation
//	cube.ApplyNotation("F B2 L' D")
//
//	fmt.Println("Solved:", cube.IsSolved())
//	fmt.Println("Phase:", cube.Phase())
//
// # Predefined Moves
//
// The package provides predefined moves for convenience:
//
//	gocube.R      // Right clockwise
//	gocube.RPrime // Right counter-clockwise
//	gocube.R2     // Right 180
//	// ... and similarly for L, U, D, F, B
//
// # Solving Phases
//
// The library detects standard layer-by-layer solving phases:
//
//   - PhaseScrambled: Cube is scrambled
//   - PhaseWhiteCross: White cross complete
//   - PhaseFirstLayer: First layer complete
//   - PhaseSecondLayer: F2L complete
//   - PhaseYellowCross: Yellow cross formed
//   - PhaseYellowCorners: Yellow corners positioned
//   - PhaseYellowOriented: Yellow corners oriented
//   - PhaseSolved: Cube is solved
package gocube
