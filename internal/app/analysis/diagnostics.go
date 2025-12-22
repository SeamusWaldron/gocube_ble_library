package analysis

import (
	"fmt"
	"math"

	"github.com/SeamusWaldron/gocube_ble_library/internal/app/storage"
)

// PhaseDiagnostics contains diagnostic metrics for a phase.
type PhaseDiagnostics struct {
	PhaseKey    string  `json:"phase_key"`
	DisplayName string  `json:"display_name"`
	MoveCount   int     `json:"move_count"`
	DurationMs  int64   `json:"duration_ms"`
	TPS         float64 `json:"tps"`

	// Reversal metrics
	ImmediateReversals    int     `json:"immediate_reversals"`     // X X' patterns
	ReversalRate          float64 `json:"reversal_rate"`           // reversals / moves
	FullCycleWaste        int     `json:"full_cycle_waste"`        // X X X X patterns

	// Base layer (D) metrics
	BaseTurns      int     `json:"base_turns"`       // D and D' moves
	BaseTurnRatio  float64 `json:"base_turn_ratio"`  // base_turns / total_moves
	LongestBaseRun int     `json:"longest_base_run"` // longest consecutive D moves

	// Gap/pause metrics
	MinGapMs       int64   `json:"min_gap_ms"`
	MaxGapMs       int64   `json:"max_gap_ms"`
	AvgGapMs       float64 `json:"avg_gap_ms"`
	GapsOver750ms  int     `json:"gaps_over_750ms"`
	GapsOver1500ms int     `json:"gaps_over_1500ms"`
	GapsOver3000ms int     `json:"gaps_over_3000ms"`

	// Short-loop detection (A B A' patterns)
	ShortLoops int `json:"short_loops"`

	// Phase entropy - measures face switching (high = searching, low = algorithmic)
	FaceEntropy   float64 `json:"face_entropy"`    // Shannon entropy of face distribution
	DistinctFaces int     `json:"distinct_faces"`  // Number of different faces used

	// Cross-specific metrics (only for white_cross phase)
	EdgePlacements     int     `json:"edge_placements,omitempty"`      // Detected edge insertions
	AvgMovesPerEdge    float64 `json:"avg_moves_per_edge,omitempty"`   // Average moves between placements
	MaxMovesPerEdge    int     `json:"max_moves_per_edge,omitempty"`   // Worst edge (most moves)
	LongestSearchRun   int     `json:"longest_search_run,omitempty"`   // Longest run without placement
}

// OrientationDiagnostics contains diagnostic metrics for cube orientation.
type OrientationDiagnostics struct {
	TotalChanges       int     `json:"total_changes"`        // Total orientation changes
	RotationBursts     int     `json:"rotation_bursts"`      // Rapid orientation changes (>2 in 500ms)
	WhiteOnTopPct      float64 `json:"white_on_top_pct"`     // Percentage of time with U face up
	GreenFrontPct      float64 `json:"green_front_pct"`      // Percentage of time with F face front
	PauseWithRotation  int     `json:"pause_with_rotation"`  // Pauses (>750ms) that have rotation
	AvgChangeGapMs     float64 `json:"avg_change_gap_ms"`    // Average time between orientation changes
	OrientationEntropy float64 `json:"orientation_entropy"`  // Entropy of orientation distribution
}

// SolveDiagnostics contains diagnostics for an entire solve.
type SolveDiagnostics struct {
	SolveID     string                 `json:"solve_id"`
	Phases      []PhaseDiagnostics     `json:"phases"`
	Overall     PhaseDiagnostics       `json:"overall"`
	Orientation OrientationDiagnostics `json:"orientation"`
}

// AnalyzeDiagnostics generates diagnostic metrics for a solve.
func AnalyzeDiagnostics(solveID string, moveRepo *storage.MoveRepository, phaseRepo *storage.PhaseRepository, orientRepo *storage.OrientationRepository) (*SolveDiagnostics, error) {
	// Get phase segments
	segments, err := phaseRepo.GetPhaseSegments(solveID)
	if err != nil {
		return nil, err
	}

	result := &SolveDiagnostics{
		SolveID: solveID,
		Phases:  make([]PhaseDiagnostics, 0, len(segments)),
	}

	// Get all moves for overall stats
	allMoves, err := moveRepo.GetBySolve(solveID)
	if err != nil {
		return nil, err
	}

	// Analyze each phase
	for _, seg := range segments {
		moves, err := moveRepo.GetBySolveRange(solveID, seg.StartTsMs, seg.EndTsMs)
		if err != nil {
			continue
		}

		diag := analyzePhaseMoves(moves, seg)
		result.Phases = append(result.Phases, diag)
	}

	// Analyze overall
	overallSeg := storage.PhaseSegment{
		PhaseKey:  "overall",
		MoveCount: len(allMoves),
	}
	if len(allMoves) > 0 {
		overallSeg.StartTsMs = allMoves[0].TsMs
		overallSeg.EndTsMs = allMoves[len(allMoves)-1].TsMs
		overallSeg.DurationMs = overallSeg.EndTsMs - overallSeg.StartTsMs
		if overallSeg.DurationMs > 0 {
			overallSeg.TPS = float64(len(allMoves)) / (float64(overallSeg.DurationMs) / 1000.0)
		}
	}
	result.Overall = analyzePhaseMoves(allMoves, overallSeg)
	result.Overall.DisplayName = "Overall"

	// Analyze orientation if repository is provided
	if orientRepo != nil {
		orientations, err := orientRepo.GetBySolve(solveID)
		if err == nil && len(orientations) > 0 {
			result.Orientation = analyzeOrientations(orientations, allMoves, overallSeg.DurationMs)
		}
	}

	return result, nil
}

func analyzePhaseMoves(moves []storage.MoveRecord, seg storage.PhaseSegment) PhaseDiagnostics {
	diag := PhaseDiagnostics{
		PhaseKey:    seg.PhaseKey,
		DisplayName: storage.PhaseDisplayName(seg.PhaseKey),
		MoveCount:   seg.MoveCount,
		DurationMs:  seg.DurationMs,
		TPS:         seg.TPS,
	}

	if len(moves) == 0 {
		return diag
	}

	// Analyze reversals
	diag.ImmediateReversals, diag.FullCycleWaste = countReversals(moves)
	if len(moves) > 0 {
		diag.ReversalRate = float64(diag.ImmediateReversals) / float64(len(moves))
	}

	// Analyze base layer (D) usage
	diag.BaseTurns, diag.LongestBaseRun = analyzeBaseTurns(moves)
	if len(moves) > 0 {
		diag.BaseTurnRatio = float64(diag.BaseTurns) / float64(len(moves))
	}

	// Analyze gaps
	analyzeGaps(moves, &diag)

	// Analyze short loops
	diag.ShortLoops = countShortLoops(moves)

	// Analyze phase entropy (face switching)
	diag.FaceEntropy, diag.DistinctFaces = analyzeFaceEntropy(moves)

	// White cross specific: edge placement detection
	if seg.PhaseKey == "white_cross" {
		diag.EdgePlacements, diag.AvgMovesPerEdge, diag.MaxMovesPerEdge, diag.LongestSearchRun = analyzeEdgePlacements(moves)
	}

	return diag
}

// countReversals counts immediate reversal patterns (X X', X' X) and full cycles (X X X X)
func countReversals(moves []storage.MoveRecord) (reversals, fullCycles int) {
	if len(moves) < 2 {
		return 0, 0
	}

	for i := 1; i < len(moves); i++ {
		prev := moves[i-1]
		curr := moves[i]

		// Check for X X' or X' X (same face, opposite direction)
		if prev.Face == curr.Face && prev.Turn == -curr.Turn {
			reversals++
		}
	}

	// Check for full cycles (4 consecutive same face turns)
	for i := 3; i < len(moves); i++ {
		if moves[i-3].Face == moves[i-2].Face &&
			moves[i-2].Face == moves[i-1].Face &&
			moves[i-1].Face == moves[i].Face {
			// Check if they sum to 0 (full cycle)
			sum := moves[i-3].Turn + moves[i-2].Turn + moves[i-1].Turn + moves[i].Turn
			if sum == 0 || sum == 4 || sum == -4 {
				fullCycles++
			}
		}
	}

	return reversals, fullCycles
}

// analyzeBaseTurns counts D-face turns and finds the longest consecutive run
func analyzeBaseTurns(moves []storage.MoveRecord) (count, longestRun int) {
	currentRun := 0

	for _, m := range moves {
		if m.Face == "D" {
			count++
			currentRun++
			if currentRun > longestRun {
				longestRun = currentRun
			}
		} else {
			currentRun = 0
		}
	}

	return count, longestRun
}

// analyzeGaps analyzes inter-move timing gaps
func analyzeGaps(moves []storage.MoveRecord, diag *PhaseDiagnostics) {
	if len(moves) < 2 {
		return
	}

	var totalGap int64
	diag.MinGapMs = moves[1].TsMs - moves[0].TsMs
	diag.MaxGapMs = diag.MinGapMs

	for i := 1; i < len(moves); i++ {
		gap := moves[i].TsMs - moves[i-1].TsMs

		if gap < diag.MinGapMs {
			diag.MinGapMs = gap
		}
		if gap > diag.MaxGapMs {
			diag.MaxGapMs = gap
		}
		totalGap += gap

		if gap > 750 {
			diag.GapsOver750ms++
		}
		if gap > 1500 {
			diag.GapsOver1500ms++
		}
		if gap > 3000 {
			diag.GapsOver3000ms++
		}
	}

	diag.AvgGapMs = float64(totalGap) / float64(len(moves)-1)
}

// countShortLoops detects patterns like A B A', A B C A', A B A B'
func countShortLoops(moves []storage.MoveRecord) int {
	if len(moves) < 3 {
		return 0
	}

	loops := 0

	// A B A' pattern (length 3)
	for i := 2; i < len(moves); i++ {
		if moves[i-2].Face == moves[i].Face && moves[i-2].Turn == -moves[i].Turn {
			// Check middle move is different face
			if moves[i-1].Face != moves[i-2].Face {
				loops++
			}
		}
	}

	// A B C A' pattern (length 4)
	for i := 3; i < len(moves); i++ {
		if moves[i-3].Face == moves[i].Face && moves[i-3].Turn == -moves[i].Turn {
			// Check middle moves are different faces
			if moves[i-2].Face != moves[i-3].Face && moves[i-1].Face != moves[i-3].Face {
				loops++
			}
		}
	}

	return loops
}

// FormatDiagnosticsReport formats a diagnostics report as a string
func FormatDiagnosticsReport(diag *SolveDiagnostics) string {
	var result string

	result += "Solve Diagnostics\n"
	result += "=================\n\n"

	for _, phase := range diag.Phases {
		result += formatPhaseDiagnostics(phase)
		result += "\n"
	}

	result += "Overall\n"
	result += "-------\n"
	result += formatPhaseDiagnostics(diag.Overall)

	return result
}

func formatPhaseDiagnostics(d PhaseDiagnostics) string {
	var result string

	result += d.DisplayName + "\n"
	result += "  Moves: " + itoa(d.MoveCount) + "\n"

	if d.MoveCount > 0 {
		result += "  Reversals: " + itoa(d.ImmediateReversals)
		result += " (" + ftoa(d.ReversalRate*100, 1) + "%)\n"

		result += "  Base (D) turns: " + itoa(d.BaseTurns)
		result += " (" + ftoa(d.BaseTurnRatio*100, 1) + "%)"
		result += ", longest run: " + itoa(d.LongestBaseRun) + "\n"

		result += "  Short loops: " + itoa(d.ShortLoops) + "\n"

		if d.MoveCount > 1 {
			result += "  Gaps: min=" + itoa64(d.MinGapMs) + "ms"
			result += ", max=" + itoa64(d.MaxGapMs) + "ms"
			result += ", avg=" + ftoa(d.AvgGapMs, 0) + "ms\n"
			result += "  Pauses: >750ms=" + itoa(d.GapsOver750ms)
			result += ", >1.5s=" + itoa(d.GapsOver1500ms)
			result += ", >3s=" + itoa(d.GapsOver3000ms) + "\n"
		}
	}

	return result
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func itoa64(i int64) string {
	return fmt.Sprintf("%d", i)
}

func ftoa(f float64, decimals int) string {
	return fmt.Sprintf("%.*f", decimals, f)
}

// analyzeFaceEntropy calculates Shannon entropy of face distribution.
// High entropy = lots of face switching (searching/exploring)
// Low entropy = concentrated on few faces (algorithmic flow)
// Max entropy for 6 faces = log2(6) ≈ 2.58
func analyzeFaceEntropy(moves []storage.MoveRecord) (entropy float64, distinctFaces int) {
	if len(moves) == 0 {
		return 0, 0
	}

	// Count face occurrences
	faceCounts := make(map[string]int)
	for _, m := range moves {
		faceCounts[m.Face]++
	}

	distinctFaces = len(faceCounts)
	total := float64(len(moves))

	// Calculate Shannon entropy: H = -Σ p(x) * log2(p(x))
	for _, count := range faceCounts {
		if count > 0 {
			p := float64(count) / total
			entropy -= p * math.Log2(p)
		}
	}

	return entropy, distinctFaces
}

// analyzeEdgePlacements detects edge placement events in white cross phase.
// Heuristic: An "edge placement" is detected when we see a pattern that
// looks like inserting an edge into the cross position:
//   - D rotations to position the edge
//   - Followed by F2/R2/L2/B2 or similar insertion moves
//
// This is a heuristic since we don't have full cube state.
func analyzeEdgePlacements(moves []storage.MoveRecord) (placements int, avgMoves float64, maxMoves int, longestSearch int) {
	if len(moves) < 2 {
		return 0, 0, 0, len(moves)
	}

	// Track moves since last placement
	movesSincePlacement := 0
	var movesPerPlacement []int

	for i := 0; i < len(moves); i++ {
		movesSincePlacement++

		// Detect placement patterns:
		// 1. Double moves on F/R/L/B faces (F2, R2, etc.) - common cross insertions
		// 2. Single moves on F/R/L/B after D positioning
		isDoubleTurn := moves[i].Turn == 2 || moves[i].Turn == -2
		isSideFace := moves[i].Face == "F" || moves[i].Face == "R" ||
			moves[i].Face == "L" || moves[i].Face == "B"

		// Check for placement pattern
		placementDetected := false

		if isSideFace && isDoubleTurn {
			// F2, R2, L2, B2 are common cross insertions
			placementDetected = true
		} else if isSideFace && i > 0 {
			// Single side move after D moves could be an insertion
			// Look for pattern: D... then F/R/L/B
			dCount := 0
			for j := i - 1; j >= 0 && j >= i-3; j-- {
				if moves[j].Face == "D" {
					dCount++
				}
			}
			if dCount >= 1 {
				// Possible edge insertion after D positioning
				placementDetected = true
			}
		}

		if placementDetected {
			placements++
			movesPerPlacement = append(movesPerPlacement, movesSincePlacement)
			if movesSincePlacement > longestSearch {
				longestSearch = movesSincePlacement
			}
			movesSincePlacement = 0
		}
	}

	// Account for trailing moves (after last placement)
	if movesSincePlacement > longestSearch {
		longestSearch = movesSincePlacement
	}

	// Calculate statistics
	if placements > 0 {
		total := 0
		for _, m := range movesPerPlacement {
			total += m
			if m > maxMoves {
				maxMoves = m
			}
		}
		avgMoves = float64(total) / float64(placements)
	}

	// Cap placements at 4 (there are only 4 cross edges)
	if placements > 4 {
		placements = 4
	}

	return placements, avgMoves, maxMoves, longestSearch
}

// analyzeOrientations analyzes orientation changes during a solve.
func analyzeOrientations(orientations []storage.OrientationRecord, moves []storage.MoveRecord, totalDurationMs int64) OrientationDiagnostics {
	diag := OrientationDiagnostics{
		TotalChanges: len(orientations),
	}

	if len(orientations) == 0 {
		return diag
	}

	// Calculate time spent in each orientation (weighted by duration)
	var whiteUpDuration, greenFrontDuration int64
	orientCounts := make(map[string]int) // count of each up_face

	for i, o := range orientations {
		var duration int64
		if i < len(orientations)-1 {
			duration = orientations[i+1].TsMs - o.TsMs
		} else {
			// Last orientation - use remaining time until end
			if totalDurationMs > o.TsMs {
				duration = totalDurationMs - o.TsMs
			}
		}

		if o.UpFace == "U" {
			whiteUpDuration += duration
		}
		if o.FrontFace == "F" {
			greenFrontDuration += duration
		}

		orientCounts[o.UpFace]++
	}

	// Calculate percentages
	if totalDurationMs > 0 {
		diag.WhiteOnTopPct = float64(whiteUpDuration) / float64(totalDurationMs) * 100
		diag.GreenFrontPct = float64(greenFrontDuration) / float64(totalDurationMs) * 100
	}

	// Calculate orientation entropy
	total := float64(len(orientations))
	for _, count := range orientCounts {
		if count > 0 {
			p := float64(count) / total
			diag.OrientationEntropy -= p * math.Log2(p)
		}
	}

	// Detect rotation bursts (multiple changes within 500ms window)
	const burstWindowMs = 500
	for i := 0; i < len(orientations); i++ {
		changesInWindow := 1
		for j := i + 1; j < len(orientations); j++ {
			if orientations[j].TsMs-orientations[i].TsMs <= burstWindowMs {
				changesInWindow++
			} else {
				break
			}
		}
		if changesInWindow > 2 {
			diag.RotationBursts++
			// Skip past this burst
			i += changesInWindow - 1
		}
	}

	// Calculate average gap between orientation changes
	if len(orientations) > 1 {
		var totalGap int64
		for i := 1; i < len(orientations); i++ {
			totalGap += orientations[i].TsMs - orientations[i-1].TsMs
		}
		diag.AvgChangeGapMs = float64(totalGap) / float64(len(orientations)-1)
	}

	// Detect pauses (>750ms between moves) that coincide with orientation changes
	const pauseThresholdMs = 750
	for i := 1; i < len(moves); i++ {
		gap := moves[i].TsMs - moves[i-1].TsMs
		if gap > pauseThresholdMs {
			// Check if any orientation change occurred during this pause
			pauseStart := moves[i-1].TsMs
			pauseEnd := moves[i].TsMs
			for _, o := range orientations {
				if o.TsMs > pauseStart && o.TsMs < pauseEnd {
					diag.PauseWithRotation++
					break
				}
			}
		}
	}

	return diag
}
