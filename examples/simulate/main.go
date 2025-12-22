// Package main demonstrates standalone cube simulation without BLE.
//
// This example shows how to:
//   - Create and manipulate a virtual cube
//   - Apply moves using predefined constants
//   - Parse and apply moves from notation strings
//   - Check solving progress and phases
//   - Visualize the cube state
//
// No hardware or BLE connection is required. This is useful for:
//   - Testing algorithms before trying on a real cube
//   - Building cube visualization applications
//   - Learning how moves affect cube state
//   - Developing solving algorithms
//
// Usage:
//
//	go run main.go
package main

import (
	"fmt"
	"strings"

	"github.com/SeamusWaldron/gocube_ble_library"
)

func main() {
	fmt.Println("GoCube Simulation Example")
	fmt.Println("=========================")
	fmt.Println()

	// ─────────────────────────────────────────────────────────────────────────
	// PART 1: Basic cube operations
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("PART 1: Basic Operations")
	fmt.Println("────────────────────────")
	fmt.Println()

	// Create a new solved cube.
	// This creates a 3x3 Rubik's cube in the solved state.
	cube := gocube.NewCube()

	fmt.Println("New cube created (solved state)")
	fmt.Printf("  IsSolved: %v\n", cube.IsSolved())
	fmt.Printf("  Phase: %s\n", cube.Phase().String())
	fmt.Println()

	// Apply moves using predefined constants.
	// These constants are available for all 18 possible moves:
	//   R, RPrime, R2, L, LPrime, L2,
	//   U, UPrime, U2, D, DPrime, D2,
	//   F, FPrime, F2, B, BPrime, B2
	fmt.Println("Applying moves: R U R' U'")
	cube.Apply(gocube.R, gocube.U, gocube.RPrime, gocube.UPrime)

	fmt.Printf("  IsSolved: %v\n", cube.IsSolved())
	fmt.Printf("  Phase: %s\n", cube.Phase().String())
	fmt.Println()

	// ─────────────────────────────────────────────────────────────────────────
	// PART 2: The "sexy move" identity
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("PART 2: The Sexy Move (R U R' U')")
	fmt.Println("──────────────────────────────────")
	fmt.Println()

	// The "sexy move" (R U R' U') repeated 6 times returns to solved.
	// This is a well-known property useful for testing cube implementations.
	cube.Reset() // Reset to solved state
	fmt.Println("Applying (R U R' U') x 6 should return to solved...")

	for i := 1; i <= 6; i++ {
		cube.Apply(gocube.R, gocube.U, gocube.RPrime, gocube.UPrime)
		fmt.Printf("  After %d iteration(s): IsSolved=%v\n", i, cube.IsSolved())
	}
	fmt.Println()

	// ─────────────────────────────────────────────────────────────────────────
	// PART 3: Parsing notation strings
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("PART 3: Parsing Notation")
	fmt.Println("────────────────────────")
	fmt.Println()

	cube.Reset()

	// ApplyNotation parses standard cube notation.
	// Supported formats:
	//   R, R', R2 (standard)
	//   Space or no space between moves
	scramble := "F R U' R' U' R U R' F' R U R' U' R' F R F'"
	fmt.Printf("Applying scramble: %s\n", scramble)

	err := cube.ApplyNotation(scramble)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  IsSolved: %v\n", cube.IsSolved())
		fmt.Printf("  Phase: %s\n", cube.Phase().String())
	}
	fmt.Println()

	// ─────────────────────────────────────────────────────────────────────────
	// PART 4: Move parsing and properties
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("PART 4: Move Properties")
	fmt.Println("───────────────────────")
	fmt.Println()

	// Parse individual moves
	testMoves := []string{"R", "R'", "R2", "U", "F2", "B'"}
	for _, notation := range testMoves {
		move, err := gocube.ParseMove(notation)
		if err != nil {
			fmt.Printf("  %s: error - %v\n", notation, err)
			continue
		}

		// Get the inverse move
		inverse := move.Inverse()

		fmt.Printf("  %s: Face=%s Turn=%d Inverse=%s\n",
			move.Notation(),
			move.Face,
			move.Turn,
			inverse.Notation(),
		)
	}
	fmt.Println()

	// ─────────────────────────────────────────────────────────────────────────
	// PART 5: Checking progress
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("PART 5: Progress Tracking")
	fmt.Println("─────────────────────────")
	fmt.Println()

	// GetProgress returns detailed information about solving progress.
	// This is useful for tracking layer-by-layer solving.
	cube.Reset()

	// Apply a scramble that partially solves the cube
	// This scramble leaves the white cross intact
	cube.ApplyNotation("R U R'") // Simple scramble

	progress := cube.GetProgress()
	fmt.Println("Progress after R U R':")
	fmt.Printf("  White Cross:     %v\n", progress.WhiteCross)
	fmt.Printf("  First Layer:     %v\n", progress.FirstLayer)
	fmt.Printf("  Second Layer:    %v\n", progress.SecondLayer)
	fmt.Printf("  Yellow Cross:    %v\n", progress.YellowCross)
	fmt.Printf("  Yellow Corners:  %v\n", progress.YellowCorners)
	fmt.Printf("  Yellow Oriented: %v\n", progress.YellowOriented)
	fmt.Printf("  Solved:          %v\n", progress.Solved)
	fmt.Println()

	// ─────────────────────────────────────────────────────────────────────────
	// PART 6: Cloning and comparison
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("PART 6: Cloning")
	fmt.Println("───────────────")
	fmt.Println()

	cube.Reset()
	cube.Apply(gocube.R, gocube.U)

	// Clone creates an independent copy of the cube state.
	// Modifying the clone won't affect the original.
	clone := cube.Clone()

	fmt.Println("Original after R U, Clone created")
	fmt.Printf("  Original IsSolved: %v\n", cube.IsSolved())
	fmt.Printf("  Clone IsSolved:    %v\n", clone.IsSolved())

	// Modify only the clone
	clone.Reset()

	fmt.Println("\nAfter resetting clone:")
	fmt.Printf("  Original IsSolved: %v\n", cube.IsSolved())
	fmt.Printf("  Clone IsSolved:    %v\n", clone.IsSolved())
	fmt.Println()

	// ─────────────────────────────────────────────────────────────────────────
	// PART 7: Visualization
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("PART 7: Cube Visualization")
	fmt.Println("──────────────────────────")
	fmt.Println()

	// Reset and apply a visible scramble
	cube.Reset()
	cube.ApplyNotation("R U R' U' F' L F L'")

	fmt.Println("Cube state after R U R' U' F' L F L':")
	fmt.Println()

	// String() returns an ASCII visualization of the cube.
	// The visualization shows all 6 faces in a cross pattern.
	fmt.Println(cube.String())

	// ─────────────────────────────────────────────────────────────────────────
	// PART 8: Solving demonstration
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("PART 8: Solving Demonstration")
	fmt.Println("─────────────────────────────")
	fmt.Println()

	// Demonstrate that applying the inverse of a scramble solves the cube
	cube.Reset()
	scrambleMoves := "R U F L B D R2 U2"
	fmt.Printf("Scramble: %s\n", scrambleMoves)

	// Apply scramble
	cube.ApplyNotation(scrambleMoves)
	fmt.Printf("  After scramble: IsSolved=%v Phase=%s\n", cube.IsSolved(), cube.Phase())

	// Parse the scramble so we can reverse it
	moves, _ := gocube.ParseMoves(scrambleMoves)

	// Apply moves in reverse order with inverted turns
	fmt.Println("  Applying inverse moves...")
	for i := len(moves) - 1; i >= 0; i-- {
		cube.Apply(moves[i].Inverse())
	}

	fmt.Printf("  After reversing: IsSolved=%v Phase=%s\n", cube.IsSolved(), cube.Phase())
	fmt.Println()

	// ─────────────────────────────────────────────────────────────────────────
	// Summary
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("                        SUMMARY                                 ")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("The gocube.Cube type provides:")
	fmt.Println()
	fmt.Println("  • NewCube()              - Create a solved cube")
	fmt.Println("  • Apply(moves...)        - Apply moves using constants")
	fmt.Println("  • ApplyNotation(string)  - Apply moves from notation string")
	fmt.Println("  • IsSolved()             - Check if cube is solved")
	fmt.Println("  • Phase()                - Get current solving phase")
	fmt.Println("  • GetProgress()          - Detailed progress information")
	fmt.Println("  • Clone()                - Create independent copy")
	fmt.Println("  • Reset()                - Reset to solved state")
	fmt.Println("  • String()               - ASCII visualization")
	fmt.Println()
	fmt.Println("Predefined moves: R, RPrime, R2, U, UPrime, U2, etc.")
	fmt.Println()

	// Show all available moves
	fmt.Println("All 18 moves:")
	allMoves := []gocube.Move{
		gocube.R, gocube.RPrime, gocube.R2,
		gocube.L, gocube.LPrime, gocube.L2,
		gocube.U, gocube.UPrime, gocube.U2,
		gocube.D, gocube.DPrime, gocube.D2,
		gocube.F, gocube.FPrime, gocube.F2,
		gocube.B, gocube.BPrime, gocube.B2,
	}

	var notations []string
	for _, m := range allMoves {
		notations = append(notations, m.Notation())
	}
	fmt.Printf("  %s\n", strings.Join(notations, " "))
}
