package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/SeamusWaldron/gocube_ble_library"
	"github.com/SeamusWaldron/gocube_ble_library/internal/app/analysis"
	"github.com/SeamusWaldron/gocube_ble_library/internal/app/storage"
)

var (
	reportSolveID   string
	reportLast      bool
	reportOutputDir string
	trendWindow     int
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate analysis reports",
	Long:  `Generate analysis reports for solves and trends.`,
}

var reportSolveCmd = &cobra.Command{
	Use:   "solve",
	Short: "Generate a solve report",
	Long: `Generate a detailed analysis report for a specific solve.

Reports include:
  - solve_summary.json: Overview statistics
  - moves.txt: Move sequence in notation
  - moves.json: Detailed move data
  - repetition_report.json: Cancellations, merges, patterns
  - ngram_report.json: Repeated move sequences (n=4-14)
  - final_phase_report.json: Tool detection for bottom_orient phase
  - phase_moves/: Per-phase move sequences`,
	RunE: runReportSolve,
}

var reportTrendCmd = &cobra.Command{
	Use:   "trend",
	Short: "Generate a trend report",
	Long:  `Generate a trend report across recent solves with improvement metrics.`,
	RunE:  runReportTrend,
}

func init() {
	rootCmd.AddCommand(reportCmd)

	reportCmd.AddCommand(reportSolveCmd)
	reportSolveCmd.Flags().StringVar(&reportSolveID, "id", "", "Solve ID to report")
	reportSolveCmd.Flags().BoolVar(&reportLast, "last", false, "Report on the last solve")
	reportSolveCmd.Flags().StringVarP(&reportOutputDir, "output", "o", "", "Output directory (default: ./reports/<solve_id>)")

	reportCmd.AddCommand(reportTrendCmd)
	reportTrendCmd.Flags().IntVar(&trendWindow, "window", 50, "Number of recent solves to analyze")
	reportTrendCmd.Flags().StringVarP(&reportOutputDir, "output", "o", "", "Output directory")
}

// FullSolveSummary is the JSON structure for solve_summary.json
type FullSolveSummary struct {
	SolveID             string                 `json:"solve_id"`
	StartedAt           string                 `json:"started_at"`
	EndedAt             string                 `json:"ended_at,omitempty"`
	SolveDurationMs     int64                  `json:"solve_duration_ms"`      // Actual solve time (excludes scramble/inspection)
	SessionDurationMs   int64                  `json:"session_duration_ms"`    // Total session time
	SolveMoves          int                    `json:"solve_moves"`            // Moves during solve (excludes scramble)
	TotalMoves          int                    `json:"total_moves"`            // All moves including scramble
	OptimizedMoves      int                    `json:"optimized_moves"`
	Efficiency          float64                `json:"efficiency"`
	TPSOverall          float64                `json:"tps_overall"`
	PhaseStats          []PhaseStatsReport     `json:"phase_stats,omitempty"`
	LongestPauseMs      int64                  `json:"longest_pause_ms"`
	PauseCountOver1500  int                    `json:"pause_count_over_1500ms"`
	AvgMoveDurationMs   float64                `json:"avg_move_duration_ms"`
	MovementProfile     *analysis.MovementProfile `json:"movement_profile,omitempty"`
	Notes               string                 `json:"notes,omitempty"`
}

// PhaseStatsReport is the JSON structure for phase statistics
type PhaseStatsReport struct {
	PhaseKey    string  `json:"phase_key"`
	DisplayName string  `json:"display_name"`
	StartTsMs   int64   `json:"start_ts_ms"`
	EndTsMs     int64   `json:"end_ts_ms"`
	DurationMs  int64   `json:"duration_ms"`
	MoveCount   int     `json:"move_count"`
	TPS         float64 `json:"tps"`
}

// PlaybackEvent is a single event in the playback timeline
type PlaybackEvent struct {
	TsMs      int64  `json:"ts_ms"`                  // Milliseconds since solve start
	Type      string `json:"type"`                   // "move" or "orientation"
	Face      string `json:"face,omitempty"`         // For moves: R, L, U, D, F, B
	Turn      int    `json:"turn,omitempty"`         // For moves: 1, -1, 2
	Notation  string `json:"notation,omitempty"`     // For moves: R, R', R2, etc.
	UpFace    string `json:"up_face,omitempty"`      // For orientation: which face is up
	FrontFace string `json:"front_face,omitempty"`   // For orientation: which face is front
}

// PlaybackData contains all data needed for visualization playback
type PlaybackData struct {
	SolveID       string                 `json:"solve_id"`
	DurationMs    int64                  `json:"duration_ms"`
	TotalMoves    int                    `json:"total_moves"`
	TotalOrients  int                    `json:"total_orientations"`
	Phases        []PhaseStatsReport     `json:"phases,omitempty"`
	Timeline      []PlaybackEvent        `json:"timeline"`
}

// PhaseAnalysis contains per-phase analysis data
type PhaseAnalysis struct {
	PhaseKey    string                     `json:"phase_key"`
	DisplayName string                     `json:"display_name"`
	MoveCount   int                        `json:"move_count"`
	DurationMs  int64                      `json:"duration_ms"`
	TPS         float64                    `json:"tps"`
	Moves       string                     `json:"moves"`
	Repetitions *analysis.RepetitionReport `json:"repetitions,omitempty"`
	TopPatterns []analysis.NGram           `json:"top_patterns,omitempty"`
}

func runReportSolve(cmd *cobra.Command, args []string) error {
	if reportSolveID == "" && !reportLast {
		return fmt.Errorf("specify --id or --last")
	}

	// Open database
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// Get solve
	solveRepo := storage.NewSolveRepository(db)
	moveRepo := storage.NewMoveRepository(db)
	phaseRepo := storage.NewPhaseRepository(db)
	orientRepo := storage.NewOrientationRepository(db)

	var solve *storage.Solve
	if reportLast {
		solve, err = solveRepo.GetLast()
	} else {
		solve, err = solveRepo.Get(reportSolveID)
	}

	if err != nil {
		return fmt.Errorf("failed to get solve: %w", err)
	}
	if solve == nil {
		return fmt.Errorf("solve not found")
	}

	// Get moves
	moveRecords, err := moveRepo.GetBySolve(solve.SolveID)
	if err != nil {
		return fmt.Errorf("failed to get moves: %w", err)
	}

	// Convert to gocube.Move for analysis
	moves := storage.ToMoves(moveRecords)

	// Get phase segments
	segments, err := phaseRepo.GetPhaseSegments(solve.SolveID)
	if err != nil {
		segments = nil
	}

	// Get phase defs for display names
	phaseDefs, _ := phaseRepo.GetAllPhaseDefs()
	phaseDefMap := make(map[string]string)
	for _, pd := range phaseDefs {
		phaseDefMap[pd.PhaseKey] = pd.DisplayName
	}

	// Determine output directory
	outputDir := reportOutputDir
	if outputDir == "" {
		// Use date-time format for directory name: YYYY-MM-DD_HHMMSS
		dirName := solve.StartedAt.Format("2006-01-02_150405")
		outputDir = filepath.Join("reports", dirName)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Run all analyses
	fmt.Println("Analyzing solve...")

	// 1. Basic stats
	longestPause := analysis.FindLongestPause(moves)
	pauseCount := analysis.CountPausesOver(moves, 1500)
	avgMoveDuration := analysis.CalculateAvgMoveDuration(moves)

	// 2. Optimization analysis
	optimized := analysis.OptimizeMoves(moves)
	efficiency := analysis.CalculateEfficiency(moves, optimized)

	// 3. Movement profile
	profile := analysis.AnalyzeMovementProfile(moves)

	// Calculate actual solve time (excluding scramble and inspection)
	var solveDurationMs int64
	var solveMoves int
	for _, seg := range segments {
		if seg.PhaseKey != "scramble" && seg.PhaseKey != "inspection" {
			solveDurationMs += seg.DurationMs
			solveMoves += seg.MoveCount
		}
	}

	// Build summary
	summary := FullSolveSummary{
		SolveID:            solve.SolveID,
		StartedAt:          solve.StartedAt.Format(time.RFC3339),
		SolveDurationMs:    solveDurationMs,
		SolveMoves:         solveMoves,
		TotalMoves:         len(moves),
		OptimizedMoves:     len(optimized),
		Efficiency:         efficiency,
		LongestPauseMs:     longestPause,
		PauseCountOver1500: pauseCount,
		AvgMoveDurationMs:  avgMoveDuration,
		MovementProfile:    profile,
	}

	if solve.EndedAt != nil {
		summary.EndedAt = solve.EndedAt.Format(time.RFC3339)
	}

	if solve.DurationMs != nil {
		summary.SessionDurationMs = *solve.DurationMs
	}

	// Calculate TPS based on actual solve time
	if solveDurationMs > 0 && solveMoves > 0 {
		summary.TPSOverall = float64(solveMoves) / (float64(solveDurationMs) / 1000.0)
	}

	if solve.Notes != nil {
		summary.Notes = *solve.Notes
	}

	// Add phase stats
	for _, seg := range segments {
		displayName := seg.PhaseKey
		if dn, ok := phaseDefMap[seg.PhaseKey]; ok {
			displayName = dn
		}
		summary.PhaseStats = append(summary.PhaseStats, PhaseStatsReport{
			PhaseKey:    seg.PhaseKey,
			DisplayName: displayName,
			StartTsMs:   seg.StartTsMs,
			EndTsMs:     seg.EndTsMs,
			DurationMs:  seg.DurationMs,
			MoveCount:   seg.MoveCount,
			TPS:         seg.TPS,
		})
	}

	// Write solve_summary.json
	if err := writeJSON(filepath.Join(outputDir, "solve_summary.json"), summary); err != nil {
		return err
	}

	// Write moves.txt
	var notations []string
	for _, m := range moves {
		notations = append(notations, m.Notation())
	}
	movesText := ""
	for i, n := range notations {
		if i > 0 {
			movesText += " "
		}
		movesText += n
	}
	if err := os.WriteFile(filepath.Join(outputDir, "moves.txt"), []byte(movesText+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write moves.txt: %w", err)
	}

	// Write moves.json
	type MoveJSON struct {
		MoveIndex int    `json:"move_index"`
		TsMs      int64  `json:"ts_ms"`
		Face      string `json:"face"`
		Turn      int    `json:"turn"`
		Notation  string `json:"notation"`
	}
	var movesJSON []MoveJSON
	for i, m := range moves {
		movesJSON = append(movesJSON, MoveJSON{
			MoveIndex: i,
			TsMs:      m.Time.UnixMilli(),
			Face:      string(m.Face),
			Turn:      int(m.Turn),
			Notation:  m.Notation(),
		})
	}
	if err := writeJSON(filepath.Join(outputDir, "moves.json"), movesJSON); err != nil {
		return err
	}

	// Write playback.json - combined timeline of moves and orientations for visualization
	fmt.Println("  - Generating playback data...")
	orientations, _ := orientRepo.GetBySolve(solve.SolveID)

	var timeline []PlaybackEvent

	// Add all moves to timeline
	for _, m := range moveRecords {
		timeline = append(timeline, PlaybackEvent{
			TsMs:     m.TsMs,
			Type:     "move",
			Face:     m.Face,
			Turn:     m.Turn,
			Notation: m.Notation,
		})
	}

	// Add all orientation changes to timeline
	for _, o := range orientations {
		timeline = append(timeline, PlaybackEvent{
			TsMs:      o.TsMs,
			Type:      "orientation",
			UpFace:    o.UpFace,
			FrontFace: o.FrontFace,
		})
	}

	// Sort timeline by timestamp
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].TsMs < timeline[j].TsMs
	})

	// Build playback data
	playback := PlaybackData{
		SolveID:      solve.SolveID,
		TotalMoves:   len(moveRecords),
		TotalOrients: len(orientations),
		Timeline:     timeline,
	}

	if solve.DurationMs != nil {
		playback.DurationMs = *solve.DurationMs
	}

	// Add phase info
	for _, seg := range segments {
		displayName := seg.PhaseKey
		if dn, ok := phaseDefMap[seg.PhaseKey]; ok {
			displayName = dn
		}
		playback.Phases = append(playback.Phases, PhaseStatsReport{
			PhaseKey:    seg.PhaseKey,
			DisplayName: displayName,
			StartTsMs:   seg.StartTsMs,
			EndTsMs:     seg.EndTsMs,
			DurationMs:  seg.DurationMs,
			MoveCount:   seg.MoveCount,
			TPS:         seg.TPS,
		})
	}

	if err := writeJSON(filepath.Join(outputDir, "playback.json"), playback); err != nil {
		return err
	}

	// 4. Repetition analysis (needed for visualizer report)
	fmt.Println("  - Analyzing repetitions...")
	repReport := analysis.AnalyzeRepetitions(moves)
	if err := writeJSON(filepath.Join(outputDir, "repetition_report.json"), repReport); err != nil {
		return err
	}

	// 5. N-gram mining
	fmt.Println("  - Mining n-grams...")
	ngramReport := analysis.MineNGrams(moves, 4, 14, 50)
	if err := writeJSON(filepath.Join(outputDir, "ngram_report.json"), ngramReport); err != nil {
		return err
	}

	// 6. Final phase analysis (if we have bottom_orient phase)
	var finalPhaseMoves []gocube.Move
	for _, seg := range segments {
		if seg.PhaseKey == "bottom_orient" {
			phaseMoveRecords, _ := moveRepo.GetBySolveRange(solve.SolveID, seg.StartTsMs, seg.EndTsMs)
			finalPhaseMoves = storage.ToMoves(phaseMoveRecords)
			break
		}
	}

	if len(finalPhaseMoves) > 0 {
		fmt.Println("  - Analyzing final phase tools...")
		finalReport := analysis.AnalyzeFinalPhase(finalPhaseMoves)
		finalReport.FinalPhaseMoveCount = len(finalPhaseMoves)
		if err := writeJSON(filepath.Join(outputDir, "final_phase_report.json"), finalReport); err != nil {
			return err
		}
	}

	// Write phase_moves directory and per-phase analysis
	var phaseAnalyses []PhaseAnalysis

	if len(segments) > 0 {
		phaseMoveDir := filepath.Join(outputDir, "phase_moves")
		if err := os.MkdirAll(phaseMoveDir, 0755); err != nil {
			return fmt.Errorf("failed to create phase_moves directory: %w", err)
		}

		fmt.Println("  - Analyzing phases...")
		for _, seg := range segments {
			phaseMoveRecords, _ := moveRepo.GetBySolveRange(solve.SolveID, seg.StartTsMs, seg.EndTsMs)
			phaseMoves := storage.ToMoves(phaseMoveRecords)
			var phaseNotations []string
			for _, m := range phaseMoves {
				phaseNotations = append(phaseNotations, m.Notation())
			}
			phaseText := ""
			for i, n := range phaseNotations {
				if i > 0 {
					phaseText += " "
				}
				phaseText += n
			}
			os.WriteFile(filepath.Join(phaseMoveDir, seg.PhaseKey+".txt"), []byte(phaseText+"\n"), 0644)

			// Per-phase analysis
			displayName := seg.PhaseKey
			if dn, ok := phaseDefMap[seg.PhaseKey]; ok {
				displayName = dn
			}

			pa := PhaseAnalysis{
				PhaseKey:    seg.PhaseKey,
				DisplayName: displayName,
				MoveCount:   len(phaseMoves),
				DurationMs:  seg.DurationMs,
				TPS:         seg.TPS,
				Moves:       phaseText,
			}

			// Analyze repetitions in this phase
			if len(phaseMoves) > 0 {
				pa.Repetitions = analysis.AnalyzeRepetitions(phaseMoves)
			}

			// Mine n-grams for patterns (4-8 move sequences)
			if len(phaseMoves) >= 4 {
				phaseNgrams := analysis.MineNGrams(phaseMoves, 4, 8, 10)
				// Collect top patterns across all n values
				var topPatterns []analysis.NGram
				for n := 4; n <= 8; n++ {
					if ngrams, ok := phaseNgrams.TopNGrams[n]; ok {
						for _, ng := range ngrams {
							if ng.Count >= 2 { // Only patterns that repeat
								topPatterns = append(topPatterns, ng)
							}
						}
					}
				}
				pa.TopPatterns = topPatterns
			}

			phaseAnalyses = append(phaseAnalyses, pa)
		}

		// Write phase_analysis.json
		if err := writeJSON(filepath.Join(outputDir, "phase_analysis.json"), phaseAnalyses); err != nil {
			return err
		}
	}

	// 7. Diagnostics analysis
	fmt.Println("  - Generating diagnostics...")
	diagnostics, err := analysis.AnalyzeDiagnostics(solve.SolveID, moveRepo, phaseRepo, orientRepo)
	if err == nil {
		if err := writeJSON(filepath.Join(outputDir, "diagnostics.json"), diagnostics); err != nil {
			return err
		}
	}

	// 8. Generate interactive visualizer HTML with full report data
	fmt.Println("  - Generating visualizer...")
	vizReport := buildVisualizerReport(
		solveDurationMs, solveMoves, len(moves), len(optimized), efficiency, summary.TPSOverall,
		longestPause, repReport, phaseAnalyses, diagnostics, phaseDefMap,
	)
	if err := generateVisualizerHTML(outputDir, solve, moveRecords, segments, orientations, phaseDefMap, vizReport); err != nil {
		return fmt.Errorf("generating visualizer: %w", err)
	}

	fmt.Println()
	fmt.Printf("Solve: %s\n", solve.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Report generated: %s\n", outputDir)
	fmt.Println()
	fmt.Println("Files created:")
	fmt.Println("  - solve_summary.json")
	fmt.Println("  - moves.txt")
	fmt.Println("  - moves.json")
	fmt.Println("  - playback.json")
	fmt.Println("  - visualizer.html")
	fmt.Println("  - repetition_report.json")
	fmt.Println("  - ngram_report.json")
	if len(finalPhaseMoves) > 0 {
		fmt.Println("  - final_phase_report.json")
	}
	if len(segments) > 0 {
		fmt.Println("  - phase_moves/")
		fmt.Println("  - phase_analysis.json")
	}
	fmt.Println("  - diagnostics.json")
	fmt.Println()

	// Print summary stats
	fmt.Println("Summary:")
	fmt.Printf("  Solve time: %.1fs\n", float64(solveDurationMs)/1000.0)
	fmt.Printf("  Moves: %d (optimized: %d, efficiency: %.1f%%)\n",
		solveMoves, len(optimized), efficiency*100)
	fmt.Printf("  TPS: %.2f\n", summary.TPSOverall)
	fmt.Printf("  Longest pause: %dms\n", longestPause)
	fmt.Printf("  Immediate cancellations: %d\n", len(repReport.ImmediateCancellations))
	fmt.Printf("  Merge opportunities: %d\n", len(repReport.MergeOpportunities))

	// Show per-phase analysis
	if len(phaseAnalyses) > 0 {
		fmt.Println()
		fmt.Println("Phase Analysis:")
		for _, pa := range phaseAnalyses {
			fmt.Printf("\n  %s (%d moves, %.1fs, %.2f TPS):\n",
				pa.DisplayName, pa.MoveCount, float64(pa.DurationMs)/1000.0, pa.TPS)
			fmt.Printf("    Moves: %s\n", pa.Moves)

			if pa.Repetitions != nil {
				if len(pa.Repetitions.ImmediateCancellations) > 0 {
					fmt.Printf("    Cancellations: %d\n", len(pa.Repetitions.ImmediateCancellations))
				}
			}

			if len(pa.TopPatterns) > 0 {
				fmt.Println("    Repeated patterns:")
				shown := 0
				for _, ng := range pa.TopPatterns {
					if shown >= 3 {
						break
					}
					fmt.Printf("      %dx: %v\n", ng.Count, ng.Sequence)
					shown++
				}
			}
		}
	}

	// Show top overall n-grams
	if ngrams, ok := ngramReport.TopNGrams[6]; ok && len(ngrams) > 0 {
		fmt.Println()
		fmt.Println("Top 6-move patterns (overall):")
		for i, ng := range ngrams {
			if i >= 3 {
				break
			}
			fmt.Printf("  %dx: %v\n", ng.Count, ng.Sequence)
		}
	}

	// Show diagnostics summary
	if diagnostics != nil {
		fmt.Println()
		fmt.Println("Diagnostics:")
		fmt.Printf("  Overall reversals: %d (%.1f%%)\n",
			diagnostics.Overall.ImmediateReversals, diagnostics.Overall.ReversalRate*100)
		fmt.Printf("  Base (D) turns: %d (%.1f%%), longest run: %d\n",
			diagnostics.Overall.BaseTurns, diagnostics.Overall.BaseTurnRatio*100, diagnostics.Overall.LongestBaseRun)
		fmt.Printf("  Short loops: %d\n", diagnostics.Overall.ShortLoops)
		if diagnostics.Overall.MoveCount > 1 {
			fmt.Printf("  Gaps: min=%dms, max=%dms, avg=%.0fms\n",
				diagnostics.Overall.MinGapMs, diagnostics.Overall.MaxGapMs, diagnostics.Overall.AvgGapMs)
			fmt.Printf("  Pauses: >750ms=%d, >1.5s=%d, >3s=%d\n",
				diagnostics.Overall.GapsOver750ms, diagnostics.Overall.GapsOver1500ms, diagnostics.Overall.GapsOver3000ms)
		}

		// Show per-phase diagnostics for key phases
		for _, pd := range diagnostics.Phases {
			if pd.PhaseKey == "white_cross" && pd.MoveCount > 0 {
				fmt.Println()
				fmt.Println("White Cross Diagnostics:")
				fmt.Printf("  Base (D) turns: %d (%.1f%%), longest run: %d\n",
					pd.BaseTurns, pd.BaseTurnRatio*100, pd.LongestBaseRun)
				fmt.Printf("  Reversals: %d (%.1f%%)\n", pd.ImmediateReversals, pd.ReversalRate*100)
				fmt.Printf("  Short loops: %d\n", pd.ShortLoops)
				fmt.Printf("  Face entropy: %.2f (distinct faces: %d)\n", pd.FaceEntropy, pd.DistinctFaces)
				if pd.EdgePlacements > 0 {
					fmt.Printf("  Edge placements: %d, avg %.1f moves/edge, max %d moves\n",
						pd.EdgePlacements, pd.AvgMovesPerEdge, pd.MaxMovesPerEdge)
					fmt.Printf("  Longest search run: %d moves\n", pd.LongestSearchRun)
				}
				if pd.MoveCount > 1 {
					fmt.Printf("  Pauses: >750ms=%d, >1.5s=%d\n",
						pd.GapsOver750ms, pd.GapsOver1500ms)
				}
			}
		}

		// Show entropy for all phases
		fmt.Println()
		fmt.Println("Phase Entropy (low=algorithmic, high=searching):")
		for _, pd := range diagnostics.Phases {
			if pd.MoveCount > 0 && pd.PhaseKey != "scramble" && pd.PhaseKey != "inspection" {
				fmt.Printf("  %s: %.2f (%d faces)\n",
					pd.DisplayName, pd.FaceEntropy, pd.DistinctFaces)
			}
		}

		// Show orientation diagnostics
		if diagnostics.Orientation.TotalChanges > 0 {
			fmt.Println()
			fmt.Println("Orientation:")
			fmt.Printf("  Cube rotations: %d\n", diagnostics.Orientation.TotalChanges)
			fmt.Printf("  Rotation bursts: %d\n", diagnostics.Orientation.RotationBursts)
			fmt.Printf("  White on top: %.1f%%\n", diagnostics.Orientation.WhiteOnTopPct)
			fmt.Printf("  Green facing front: %.1f%%\n", diagnostics.Orientation.GreenFrontPct)
			if diagnostics.Orientation.PauseWithRotation > 0 {
				fmt.Printf("  Pauses with rotation: %d\n", diagnostics.Orientation.PauseWithRotation)
			}
			if diagnostics.Orientation.AvgChangeGapMs > 0 {
				fmt.Printf("  Avg time between rotations: %.0fms\n", diagnostics.Orientation.AvgChangeGapMs)
			}
		}
	}

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// GenerateReportForSolve generates a full report for a solve and returns the output directory.
// This can be called from both CLI commands and the TUI.
func GenerateReportForSolve(db *storage.DB, solveID string) (string, error) {
	solveRepo := storage.NewSolveRepository(db)
	moveRepo := storage.NewMoveRepository(db)
	phaseRepo := storage.NewPhaseRepository(db)
	orientRepo := storage.NewOrientationRepository(db)

	solve, err := solveRepo.Get(solveID)
	if err != nil {
		return "", fmt.Errorf("failed to get solve: %w", err)
	}
	if solve == nil {
		return "", fmt.Errorf("solve not found")
	}

	// Get moves
	moveRecords, err := moveRepo.GetBySolve(solve.SolveID)
	if err != nil {
		return "", fmt.Errorf("failed to get moves: %w", err)
	}

	moves := storage.ToMoves(moveRecords)

	// Get phase segments
	segments, err := phaseRepo.GetPhaseSegments(solve.SolveID)
	if err != nil {
		segments = nil
	}

	// Get phase defs for display names
	phaseDefs, _ := phaseRepo.GetAllPhaseDefs()
	phaseDefMap := make(map[string]string)
	for _, pd := range phaseDefs {
		phaseDefMap[pd.PhaseKey] = pd.DisplayName
	}

	// Create output directory
	dirName := solve.StartedAt.Format("2006-01-02_150405")
	outputDir := filepath.Join("reports", dirName)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Basic stats
	longestPause := analysis.FindLongestPause(moves)
	pauseCount := analysis.CountPausesOver(moves, 1500)
	avgMoveDuration := analysis.CalculateAvgMoveDuration(moves)

	// Optimization analysis
	optimized := analysis.OptimizeMoves(moves)
	efficiency := analysis.CalculateEfficiency(moves, optimized)

	// Movement profile
	profile := analysis.AnalyzeMovementProfile(moves)

	// Calculate actual solve time
	var solveDurationMs int64
	var solveMoves int
	for _, seg := range segments {
		if seg.PhaseKey != "scramble" && seg.PhaseKey != "inspection" {
			solveDurationMs += seg.DurationMs
			solveMoves += seg.MoveCount
		}
	}

	// Build summary
	summary := FullSolveSummary{
		SolveID:            solve.SolveID,
		StartedAt:          solve.StartedAt.Format(time.RFC3339),
		SolveDurationMs:    solveDurationMs,
		SolveMoves:         solveMoves,
		TotalMoves:         len(moves),
		OptimizedMoves:     len(optimized),
		Efficiency:         efficiency,
		LongestPauseMs:     longestPause,
		PauseCountOver1500: pauseCount,
		AvgMoveDurationMs:  avgMoveDuration,
		MovementProfile:    profile,
	}

	if solve.EndedAt != nil {
		summary.EndedAt = solve.EndedAt.Format(time.RFC3339)
	}
	if solve.DurationMs != nil {
		summary.SessionDurationMs = *solve.DurationMs
	}
	if solveDurationMs > 0 && solveMoves > 0 {
		summary.TPSOverall = float64(solveMoves) / (float64(solveDurationMs) / 1000.0)
	}
	if solve.Notes != nil {
		summary.Notes = *solve.Notes
	}

	// Add phase stats
	for _, seg := range segments {
		displayName := seg.PhaseKey
		if dn, ok := phaseDefMap[seg.PhaseKey]; ok {
			displayName = dn
		}
		summary.PhaseStats = append(summary.PhaseStats, PhaseStatsReport{
			PhaseKey:    seg.PhaseKey,
			DisplayName: displayName,
			StartTsMs:   seg.StartTsMs,
			EndTsMs:     seg.EndTsMs,
			DurationMs:  seg.DurationMs,
			MoveCount:   seg.MoveCount,
			TPS:         seg.TPS,
		})
	}

	// Write solve_summary.json
	if err := writeJSON(filepath.Join(outputDir, "solve_summary.json"), summary); err != nil {
		return "", err
	}

	// Write moves.txt
	var notations []string
	for _, m := range moves {
		notations = append(notations, m.Notation())
	}
	movesText := ""
	for i, n := range notations {
		if i > 0 {
			movesText += " "
		}
		movesText += n
	}
	if err := os.WriteFile(filepath.Join(outputDir, "moves.txt"), []byte(movesText+"\n"), 0644); err != nil {
		return "", fmt.Errorf("failed to write moves.txt: %w", err)
	}

	// Write moves.json
	type MoveJSON struct {
		MoveIndex int    `json:"move_index"`
		TsMs      int64  `json:"ts_ms"`
		Face      string `json:"face"`
		Turn      int    `json:"turn"`
		Notation  string `json:"notation"`
	}
	var movesJSON []MoveJSON
	for i, m := range moves {
		movesJSON = append(movesJSON, MoveJSON{
			MoveIndex: i,
			TsMs:      m.Time.UnixMilli(),
			Face:      string(m.Face),
			Turn:      int(m.Turn),
			Notation:  m.Notation(),
		})
	}
	if err := writeJSON(filepath.Join(outputDir, "moves.json"), movesJSON); err != nil {
		return "", err
	}

	// Write playback.json
	orientations, _ := orientRepo.GetBySolve(solve.SolveID)
	var timeline []PlaybackEvent

	for _, m := range moveRecords {
		timeline = append(timeline, PlaybackEvent{
			TsMs:     m.TsMs,
			Type:     "move",
			Face:     m.Face,
			Turn:     m.Turn,
			Notation: m.Notation,
		})
	}
	for _, o := range orientations {
		timeline = append(timeline, PlaybackEvent{
			TsMs:      o.TsMs,
			Type:      "orientation",
			UpFace:    o.UpFace,
			FrontFace: o.FrontFace,
		})
	}
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].TsMs < timeline[j].TsMs
	})

	playback := PlaybackData{
		SolveID:      solve.SolveID,
		TotalMoves:   len(moveRecords),
		TotalOrients: len(orientations),
		Timeline:     timeline,
	}
	if solve.DurationMs != nil {
		playback.DurationMs = *solve.DurationMs
	}
	for _, seg := range segments {
		displayName := seg.PhaseKey
		if dn, ok := phaseDefMap[seg.PhaseKey]; ok {
			displayName = dn
		}
		playback.Phases = append(playback.Phases, PhaseStatsReport{
			PhaseKey:    seg.PhaseKey,
			DisplayName: displayName,
			StartTsMs:   seg.StartTsMs,
			EndTsMs:     seg.EndTsMs,
			DurationMs:  seg.DurationMs,
			MoveCount:   seg.MoveCount,
			TPS:         seg.TPS,
		})
	}
	if err := writeJSON(filepath.Join(outputDir, "playback.json"), playback); err != nil {
		return "", err
	}

	// Repetition analysis
	repReport := analysis.AnalyzeRepetitions(moves)
	if err := writeJSON(filepath.Join(outputDir, "repetition_report.json"), repReport); err != nil {
		return "", err
	}

	// N-gram mining
	ngramReport := analysis.MineNGrams(moves, 4, 14, 50)
	if err := writeJSON(filepath.Join(outputDir, "ngram_report.json"), ngramReport); err != nil {
		return "", err
	}

	// Final phase analysis
	var finalPhaseMoves []gocube.Move
	for _, seg := range segments {
		if seg.PhaseKey == "bottom_orient" {
			phaseMoveRecords, _ := moveRepo.GetBySolveRange(solve.SolveID, seg.StartTsMs, seg.EndTsMs)
			finalPhaseMoves = storage.ToMoves(phaseMoveRecords)
			break
		}
	}
	if len(finalPhaseMoves) > 0 {
		finalReport := analysis.AnalyzeFinalPhase(finalPhaseMoves)
		finalReport.FinalPhaseMoveCount = len(finalPhaseMoves)
		writeJSON(filepath.Join(outputDir, "final_phase_report.json"), finalReport)
	}

	// Phase analysis
	var phaseAnalyses []PhaseAnalysis
	if len(segments) > 0 {
		phaseMoveDir := filepath.Join(outputDir, "phase_moves")
		os.MkdirAll(phaseMoveDir, 0755)

		for _, seg := range segments {
			phaseMoveRecords, _ := moveRepo.GetBySolveRange(solve.SolveID, seg.StartTsMs, seg.EndTsMs)
			phaseMoves := storage.ToMoves(phaseMoveRecords)
			var phaseNotations []string
			for _, m := range phaseMoves {
				phaseNotations = append(phaseNotations, m.Notation())
			}
			phaseText := ""
			for i, n := range phaseNotations {
				if i > 0 {
					phaseText += " "
				}
				phaseText += n
			}
			os.WriteFile(filepath.Join(phaseMoveDir, seg.PhaseKey+".txt"), []byte(phaseText+"\n"), 0644)

			displayName := seg.PhaseKey
			if dn, ok := phaseDefMap[seg.PhaseKey]; ok {
				displayName = dn
			}

			pa := PhaseAnalysis{
				PhaseKey:    seg.PhaseKey,
				DisplayName: displayName,
				MoveCount:   len(phaseMoves),
				DurationMs:  seg.DurationMs,
				TPS:         seg.TPS,
				Moves:       phaseText,
			}

			if len(phaseMoves) > 0 {
				pa.Repetitions = analysis.AnalyzeRepetitions(phaseMoves)
			}
			if len(phaseMoves) >= 4 {
				phaseNgrams := analysis.MineNGrams(phaseMoves, 4, 8, 10)
				var topPatterns []analysis.NGram
				for n := 4; n <= 8; n++ {
					if ngrams, ok := phaseNgrams.TopNGrams[n]; ok {
						for _, ng := range ngrams {
							if ng.Count >= 2 {
								topPatterns = append(topPatterns, ng)
							}
						}
					}
				}
				pa.TopPatterns = topPatterns
			}

			phaseAnalyses = append(phaseAnalyses, pa)
		}

		writeJSON(filepath.Join(outputDir, "phase_analysis.json"), phaseAnalyses)
	}

	// Diagnostics
	diagnostics, _ := analysis.AnalyzeDiagnostics(solve.SolveID, moveRepo, phaseRepo, orientRepo)
	if diagnostics != nil {
		writeJSON(filepath.Join(outputDir, "diagnostics.json"), diagnostics)
	}

	// Generate visualiser
	vizReport := buildVisualizerReport(
		solveDurationMs, solveMoves, len(moves), len(optimized), efficiency, summary.TPSOverall,
		longestPause, repReport, phaseAnalyses, diagnostics, phaseDefMap,
	)
	if err := generateVisualizerHTML(outputDir, solve, moveRecords, segments, orientations, phaseDefMap, vizReport); err != nil {
		return "", fmt.Errorf("generating visualizer: %w", err)
	}

	return outputDir, nil
}

func runReportTrend(cmd *cobra.Command, args []string) error {
	// Open database
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	solveRepo := storage.NewSolveRepository(db)
	moveRepo := storage.NewMoveRepository(db)
	phaseRepo := storage.NewPhaseRepository(db)

	// Get recent solves
	solves, err := solveRepo.List(trendWindow)
	if err != nil {
		return fmt.Errorf("failed to get solves: %w", err)
	}

	if len(solves) == 0 {
		return fmt.Errorf("no solves found")
	}

	fmt.Printf("Analyzing %d solves...\n", len(solves))

	// Build solve data for trend analysis
	var solveData []analysis.SolveData
	for _, s := range solves {
		if s.DurationMs == nil || *s.DurationMs <= 0 {
			continue
		}

		moveCount, _ := moveRepo.Count(s.SolveID)
		tps := float64(moveCount) / (float64(*s.DurationMs) / 1000.0)

		sd := analysis.SolveData{
			SolveID:    s.SolveID,
			StartedAt:  s.StartedAt,
			DurationMs: *s.DurationMs,
			MoveCount:  moveCount,
			TPS:        tps,
			PhaseData:  make(map[string]analysis.PhaseData),
		}

		// Get phase data
		segments, _ := phaseRepo.GetPhaseSegments(s.SolveID)
		for _, seg := range segments {
			sd.PhaseData[seg.PhaseKey] = analysis.PhaseData{
				DurationMs: seg.DurationMs,
				MoveCount:  seg.MoveCount,
				TPS:        seg.TPS,
			}
		}

		solveData = append(solveData, sd)
	}

	if len(solveData) == 0 {
		return fmt.Errorf("no completed solves found")
	}

	// Run trend analysis
	trendReport := analysis.AnalyzeTrends(solveData)

	// Determine output
	outputDir := reportOutputDir
	if outputDir == "" {
		outputDir = "reports"
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputFile := filepath.Join(outputDir, "trend_report.json")
	if err := writeJSON(outputFile, trendReport); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Trend report generated: %s\n", outputFile)
	fmt.Println()
	fmt.Printf("Analyzed %d completed solves\n", trendReport.CompletedSolves)
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("  Average duration: %.1fs\n", trendReport.AvgDurationMs/1000.0)
	fmt.Printf("  Average moves: %.1f\n", trendReport.AvgMoves)
	fmt.Printf("  Average TPS: %.2f\n", trendReport.AvgTPS)
	fmt.Println()
	fmt.Printf("  Best solve: %.1fs (%s)\n", float64(trendReport.BestSolve.DurationMs)/1000.0, trendReport.BestSolve.SolveID[:8])
	fmt.Printf("  Worst solve: %.1fs (%s)\n", float64(trendReport.WorstSolve.DurationMs)/1000.0, trendReport.WorstSolve.SolveID[:8])
	fmt.Println()
	fmt.Printf("  Improvement: %.1f%%\n", trendReport.ImprovementPct)
	fmt.Printf("  Consistency: %.1f/100\n", trendReport.ConsistencyScore)

	// Rolling averages
	if len(trendReport.RollingAvgs) > 0 {
		fmt.Println()
		fmt.Println("Rolling averages:")
		for _, n := range []int{5, 10, 25, 50} {
			if avg, ok := trendReport.RollingAvgs[n]; ok {
				fmt.Printf("  ao%d: %.1fs\n", n, avg/1000.0)
			}
		}
	}

	// Phase trends
	if len(trendReport.PhaseTrends) > 0 {
		fmt.Println()
		fmt.Println("Phase trends:")
		for key, trend := range trendReport.PhaseTrends {
			fmt.Printf("  %s: %.1fs avg, %.1f%% improvement\n",
				key, trend.AvgDurationMs/1000.0, trend.ImprovementPct)
		}
	}

	return nil
}

// writeJSON writes data as formatted JSON to a file.
func writeJSON(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

// buildVisualizerReport constructs the report data for the visualizer.
func buildVisualizerReport(
	solveDurationMs int64,
	solveMoves int,
	totalMoves int,
	optimizedMoves int,
	efficiency float64,
	tps float64,
	longestPauseMs int64,
	repReport *analysis.RepetitionReport,
	phaseAnalyses []PhaseAnalysis,
	diagnostics *analysis.SolveDiagnostics,
	phaseDefMap map[string]string,
) *VisualizerReport {
	report := &VisualizerReport{
		SolveTimeMs:        solveDurationMs,
		TotalMoves:         totalMoves,
		SolveMoves:         solveMoves,
		OptimizedMoves:     optimizedMoves,
		Efficiency:         efficiency,
		TPS:                tps,
		LongestPauseMs:     longestPauseMs,
		ImmediateCancels:   len(repReport.ImmediateCancellations),
		MergeOpportunities: len(repReport.MergeOpportunities),
	}

	// Add phase analysis
	for _, pa := range phaseAnalyses {
		var topPatterns []string
		for _, ng := range pa.TopPatterns {
			if len(topPatterns) < 3 { // Limit to top 3
				topPatterns = append(topPatterns, fmt.Sprintf("%dx: %v", ng.Count, ng.Sequence))
			}
		}

		cancellations := 0
		if pa.Repetitions != nil {
			cancellations = len(pa.Repetitions.ImmediateCancellations)
		}

		report.PhaseAnalysis = append(report.PhaseAnalysis, VisualizerPhaseAnalysis{
			PhaseKey:      pa.PhaseKey,
			DisplayName:   pa.DisplayName,
			MoveCount:     pa.MoveCount,
			DurationMs:    pa.DurationMs,
			TPS:           pa.TPS,
			Moves:         pa.Moves,
			Cancellations: cancellations,
			TopPatterns:   topPatterns,
		})
	}

	// Add diagnostics if available
	if diagnostics != nil {
		vizDiag := &VisualizerDiagnostics{
			ReversalCount:  diagnostics.Overall.ImmediateReversals,
			ReversalRate:   diagnostics.Overall.ReversalRate,
			BaseTurns:      diagnostics.Overall.BaseTurns,
			BaseTurnRatio:  diagnostics.Overall.BaseTurnRatio,
			LongestBaseRun: diagnostics.Overall.LongestBaseRun,
			ShortLoops:     diagnostics.Overall.ShortLoops,
			MinGapMs:       diagnostics.Overall.MinGapMs,
			MaxGapMs:       diagnostics.Overall.MaxGapMs,
			AvgGapMs:       diagnostics.Overall.AvgGapMs,
			PausesOver750:  diagnostics.Overall.GapsOver750ms,
			PausesOver1500: diagnostics.Overall.GapsOver1500ms,
			PausesOver3000: diagnostics.Overall.GapsOver3000ms,
		}

		// Orientation diagnostics
		vizDiag.OrientationChanges = diagnostics.Orientation.TotalChanges
		vizDiag.RotationBursts = diagnostics.Orientation.RotationBursts
		vizDiag.WhiteOnTopPct = diagnostics.Orientation.WhiteOnTopPct
		vizDiag.GreenFrontPct = diagnostics.Orientation.GreenFrontPct

		// White cross specific
		for _, pd := range diagnostics.Phases {
			if pd.PhaseKey == "white_cross" && pd.MoveCount > 0 {
				vizDiag.WhiteCrossBaseTurns = pd.BaseTurns
				vizDiag.WhiteCrossBaseTurnRatio = pd.BaseTurnRatio
				vizDiag.WhiteCrossReversals = pd.ImmediateReversals
				vizDiag.WhiteCrossReversalRate = pd.ReversalRate
				vizDiag.WhiteCrossEdgePlacements = pd.EdgePlacements
				vizDiag.WhiteCrossAvgMovesPerEdge = pd.AvgMovesPerEdge
				break
			}
		}

		// Phase entropy
		for _, pd := range diagnostics.Phases {
			if pd.MoveCount > 0 && pd.PhaseKey != "scramble" && pd.PhaseKey != "inspection" {
				displayName := pd.PhaseKey
				if dn, ok := phaseDefMap[pd.PhaseKey]; ok {
					displayName = dn
				}
				vizDiag.PhaseEntropy = append(vizDiag.PhaseEntropy, VisualizerPhaseEntropy{
					PhaseKey:      pd.PhaseKey,
					DisplayName:   displayName,
					Entropy:       pd.FaceEntropy,
					DistinctFaces: pd.DistinctFaces,
				})
			}
		}

		report.Diagnostics = vizDiag
	}

	return report
}
