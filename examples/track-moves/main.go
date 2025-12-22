// Package main demonstrates real-time move tracking with statistics.
//
// This example shows how to:
//   - Track moves in real-time with timestamps
//   - Calculate turns per second (TPS)
//   - Monitor solving phases as they complete
//   - Generate move statistics and analysis
//
// This is useful for:
//   - Practicing solves and tracking improvement
//   - Analyzing solving patterns
//   - Building training applications
//
// Usage:
//
//	go run main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/SeamusWaldron/gocube_ble_library"
)

// SolveStats tracks statistics for a single solve session.
// It accumulates data as moves are made and can compute
// various metrics like TPS, phase durations, and move distribution.
type SolveStats struct {
	// StartTime is when the first move was made
	StartTime time.Time

	// Moves is the list of all moves made
	Moves []gocube.Move

	// PhaseTimes records when each phase was reached
	PhaseTimes map[gocube.Phase]time.Time

	// FaceCounts tracks how many times each face was turned
	FaceCounts map[gocube.Face]int

	// LastMoveTime is the timestamp of the most recent move
	LastMoveTime time.Time
}

// NewSolveStats creates a new SolveStats instance with initialized maps.
func NewSolveStats() *SolveStats {
	return &SolveStats{
		PhaseTimes: make(map[gocube.Phase]time.Time),
		FaceCounts: make(map[gocube.Face]int),
	}
}

// RecordMove adds a move to the statistics.
// It updates all relevant counters and timestamps.
func (s *SolveStats) RecordMove(m gocube.Move) {
	// Record start time on first move
	if len(s.Moves) == 0 {
		s.StartTime = m.Time
	}

	s.Moves = append(s.Moves, m)
	s.LastMoveTime = m.Time
	s.FaceCounts[m.Face]++
}

// RecordPhase marks when a phase was completed.
func (s *SolveStats) RecordPhase(p gocube.Phase, t time.Time) {
	if _, exists := s.PhaseTimes[p]; !exists {
		s.PhaseTimes[p] = t
	}
}

// Duration returns the total time from first to last move.
func (s *SolveStats) Duration() time.Duration {
	if len(s.Moves) < 2 {
		return 0
	}
	return s.LastMoveTime.Sub(s.StartTime)
}

// TPS calculates turns per second over the entire session.
// Returns 0 if duration is zero.
func (s *SolveStats) TPS() float64 {
	duration := s.Duration()
	if duration == 0 || len(s.Moves) == 0 {
		return 0
	}
	return float64(len(s.Moves)) / duration.Seconds()
}

// RecentTPS calculates TPS over the last N moves.
// This is useful for seeing current solving speed vs overall average.
func (s *SolveStats) RecentTPS(n int) float64 {
	if len(s.Moves) < 2 {
		return 0
	}

	// Get the last N moves (or all if fewer)
	startIdx := len(s.Moves) - n
	if startIdx < 0 {
		startIdx = 0
	}

	recentMoves := s.Moves[startIdx:]
	if len(recentMoves) < 2 {
		return 0
	}

	duration := recentMoves[len(recentMoves)-1].Time.Sub(recentMoves[0].Time)
	if duration == 0 {
		return 0
	}

	return float64(len(recentMoves)) / duration.Seconds()
}

// MoveSequence returns all moves as a notation string.
// Example: "R U R' U' F B2"
func (s *SolveStats) MoveSequence() string {
	notations := make([]string, len(s.Moves))
	for i, m := range s.Moves {
		notations[i] = m.Notation()
	}
	return strings.Join(notations, " ")
}

// PrintSummary displays a formatted summary of the solve statistics.
func (s *SolveStats) PrintSummary() {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("           SOLVE STATISTICS            ")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()

	// Basic stats
	fmt.Printf("Total moves:     %d\n", len(s.Moves))
	fmt.Printf("Duration:        %s\n", s.Duration().Round(time.Millisecond))
	fmt.Printf("Average TPS:     %.2f\n", s.TPS())
	fmt.Println()

	// Face distribution
	fmt.Println("Face Distribution:")
	faces := []gocube.Face{gocube.FaceR, gocube.FaceL, gocube.FaceU, gocube.FaceD, gocube.FaceF, gocube.FaceB}
	for _, face := range faces {
		count := s.FaceCounts[face]
		if count > 0 {
			pct := float64(count) / float64(len(s.Moves)) * 100
			bar := strings.Repeat("█", int(pct/5))
			fmt.Printf("  %s: %3d (%5.1f%%) %s\n", face, count, pct, bar)
		}
	}
	fmt.Println()

	// Move sequence (truncate if too long)
	sequence := s.MoveSequence()
	if len(sequence) > 60 {
		sequence = sequence[:60] + "..."
	}
	fmt.Printf("Moves: %s\n", sequence)
	fmt.Println()
}

func main() {
	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("GoCube Move Tracker")
	fmt.Println("===================")
	fmt.Println()
	fmt.Println("Scanning for GoCube devices...")

	// Connect to the first available cube
	cube, err := gocube.ConnectFirst(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer cube.Close()

	fmt.Printf("Connected to: %s (Battery: %d%%)\n", cube.DeviceName(), cube.Battery())
	fmt.Println()
	fmt.Println("Start solving! Statistics will be shown on exit.")
	fmt.Println("Press Ctrl+C to stop and see summary.")
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────")

	// Initialize statistics tracker
	stats := NewSolveStats()

	// Track the current phase to detect changes
	currentPhase := cube.Phase()

	// Set up move callback with real-time TPS display
	cube.OnMove(func(m gocube.Move) {
		// Record the move
		stats.RecordMove(m)

		// Calculate recent TPS (last 5 moves)
		recentTPS := stats.RecentTPS(5)

		// Format the output with move count, notation, and TPS
		fmt.Printf("[%3d] %-3s  TPS: %.2f (recent: %.2f)\n",
			len(stats.Moves),
			m.Notation(),
			stats.TPS(),
			recentTPS,
		)
	})

	// Track phase changes with timing
	cube.OnPhaseChange(func(p gocube.Phase) {
		// Record when this phase was reached
		stats.RecordPhase(p, time.Now())

		// Calculate how many moves since last phase
		moveCount := len(stats.Moves)

		fmt.Println()
		fmt.Printf("  ▶ Phase: %s (at move %d)\n", p.String(), moveCount)
		fmt.Println()

		currentPhase = p
	})

	// Celebrate when solved
	cube.OnSolved(func() {
		fmt.Println()
		fmt.Println("  ╔═══════════════════════════════╗")
		fmt.Println("  ║       🎉 CUBE SOLVED! 🎉      ║")
		fmt.Println("  ╚═══════════════════════════════╝")
		fmt.Println()

		// Show quick stats
		fmt.Printf("  Time: %s | Moves: %d | TPS: %.2f\n",
			stats.Duration().Round(time.Millisecond),
			len(stats.Moves),
			stats.TPS(),
		)
		fmt.Println()
	})

	// Handle disconnection
	cube.OnDisconnect(func(err error) {
		if err != nil {
			fmt.Printf("\nConnection lost: %v\n", err)
		}
		cancel()
	})

	// Wait for shutdown signal
	select {
	case <-sigChan:
		fmt.Println("\n─────────────────────────────────────────────────")
	case <-ctx.Done():
		fmt.Println("\n─────────────────────────────────────────────────")
	}

	// Print final statistics
	if len(stats.Moves) > 0 {
		stats.PrintSummary()
	} else {
		fmt.Println("\nNo moves recorded.")
	}

	// Report final phase
	fmt.Printf("Final phase: %s\n", currentPhase.String())
}
