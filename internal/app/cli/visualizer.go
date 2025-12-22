package cli

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/SeamusWaldron/gocube_ble_library/internal/app/storage"
)

//go:embed visualizer_template.html
var visualizerTemplate string

// VisualizerData contains all data needed for the 3D solve visualization.
type VisualizerData struct {
	SolveID         string             `json:"solve_id"`
	TotalDurationMs int64              `json:"total_duration_ms"`
	SolveDurationMs int64              `json:"solve_duration_ms"`
	Phases          []VisualizerPhase  `json:"phases"`
	Moves           []VisualizerMove   `json:"moves"`
	Orientations    []VisualizerOrient `json:"orientations"`
	Report          *VisualizerReport  `json:"report,omitempty"`
}

// VisualizerReport contains the analysis report data.
type VisualizerReport struct {
	// Summary stats
	SolveTimeMs          int64   `json:"solve_time_ms"`
	TotalMoves           int     `json:"total_moves"`
	SolveMoves           int     `json:"solve_moves"`
	OptimizedMoves       int     `json:"optimized_moves"`
	Efficiency           float64 `json:"efficiency"`
	TPS                  float64 `json:"tps"`
	LongestPauseMs       int64   `json:"longest_pause_ms"`
	ImmediateCancels     int     `json:"immediate_cancels"`
	MergeOpportunities   int     `json:"merge_opportunities"`

	// Phase analysis
	PhaseAnalysis []VisualizerPhaseAnalysis `json:"phase_analysis"`

	// Diagnostics
	Diagnostics *VisualizerDiagnostics `json:"diagnostics,omitempty"`
}

// VisualizerPhaseAnalysis contains per-phase analysis.
type VisualizerPhaseAnalysis struct {
	PhaseKey       string   `json:"phase_key"`
	DisplayName    string   `json:"display_name"`
	MoveCount      int      `json:"move_count"`
	DurationMs     int64    `json:"duration_ms"`
	TPS            float64  `json:"tps"`
	Moves          string   `json:"moves"`
	Cancellations  int      `json:"cancellations"`
	TopPatterns    []string `json:"top_patterns,omitempty"`
}

// VisualizerDiagnostics contains diagnostic metrics.
type VisualizerDiagnostics struct {
	ReversalCount   int     `json:"reversal_count"`
	ReversalRate    float64 `json:"reversal_rate"`
	BaseTurns       int     `json:"base_turns"`
	BaseTurnRatio   float64 `json:"base_turn_ratio"`
	LongestBaseRun  int     `json:"longest_base_run"`
	ShortLoops      int     `json:"short_loops"`
	MinGapMs        int64   `json:"min_gap_ms"`
	MaxGapMs        int64   `json:"max_gap_ms"`
	AvgGapMs        float64 `json:"avg_gap_ms"`
	PausesOver750   int     `json:"pauses_over_750ms"`
	PausesOver1500  int     `json:"pauses_over_1500ms"`
	PausesOver3000  int     `json:"pauses_over_3000ms"`

	// White cross specific
	WhiteCrossBaseTurns      int     `json:"white_cross_base_turns,omitempty"`
	WhiteCrossBaseTurnRatio  float64 `json:"white_cross_base_turn_ratio,omitempty"`
	WhiteCrossReversals      int     `json:"white_cross_reversals,omitempty"`
	WhiteCrossReversalRate   float64 `json:"white_cross_reversal_rate,omitempty"`
	WhiteCrossEdgePlacements int     `json:"white_cross_edge_placements,omitempty"`
	WhiteCrossAvgMovesPerEdge float64 `json:"white_cross_avg_moves_per_edge,omitempty"`

	// Orientation
	OrientationChanges   int     `json:"orientation_changes"`
	RotationBursts       int     `json:"rotation_bursts"`
	WhiteOnTopPct        float64 `json:"white_on_top_pct"`
	GreenFrontPct        float64 `json:"green_front_pct"`

	// Phase entropy
	PhaseEntropy []VisualizerPhaseEntropy `json:"phase_entropy,omitempty"`
}

// VisualizerPhaseEntropy contains entropy data for a phase.
type VisualizerPhaseEntropy struct {
	PhaseKey      string  `json:"phase_key"`
	DisplayName   string  `json:"display_name"`
	Entropy       float64 `json:"entropy"`
	DistinctFaces int     `json:"distinct_faces"`
}

// VisualizerPhase represents a solving phase with timing data.
type VisualizerPhase struct {
	PhaseKey    string  `json:"phase_key"`
	DisplayName string  `json:"display_name"`
	StartTsMs   int64   `json:"start_ts_ms"`
	EndTsMs     int64   `json:"end_ts_ms"`
	DurationMs  int64   `json:"duration_ms"`
	MoveCount   int     `json:"move_count"`
	TPS         float64 `json:"tps"`
}

// VisualizerMove represents a single move with its actual timestamp.
type VisualizerMove struct {
	TsMs     int64  `json:"ts_ms"`
	Face     string `json:"face"`
	Turn     int    `json:"turn"`
	Notation string `json:"notation"`
}

// VisualizerOrient represents a cube orientation change.
type VisualizerOrient struct {
	TsMs      int64  `json:"ts_ms"`
	UpFace    string `json:"up_face"`
	FrontFace string `json:"front_face"`
}

// buildVisualizerData constructs VisualizerData from database records.
func buildVisualizerData(
	solve *storage.Solve,
	moves []storage.MoveRecord,
	phases []storage.PhaseSegment,
	orientations []storage.OrientationRecord,
	phaseDefMap map[string]string,
	report *VisualizerReport,
) VisualizerData {
	// Convert moves
	vizMoves := make([]VisualizerMove, len(moves))
	for i, m := range moves {
		vizMoves[i] = VisualizerMove{
			TsMs:     m.TsMs,
			Face:     m.Face,
			Turn:     m.Turn,
			Notation: m.Notation,
		}
	}

	// Convert phases
	vizPhases := make([]VisualizerPhase, len(phases))
	for i, p := range phases {
		displayName := p.PhaseKey
		if dn, ok := phaseDefMap[p.PhaseKey]; ok {
			displayName = dn
		}
		vizPhases[i] = VisualizerPhase{
			PhaseKey:    p.PhaseKey,
			DisplayName: displayName,
			StartTsMs:   p.StartTsMs,
			EndTsMs:     p.EndTsMs,
			DurationMs:  p.DurationMs,
			MoveCount:   p.MoveCount,
			TPS:         p.TPS,
		}
	}

	// Convert orientations
	vizOrients := make([]VisualizerOrient, len(orientations))
	for i, o := range orientations {
		vizOrients[i] = VisualizerOrient{
			TsMs:      o.TsMs,
			UpFace:    o.UpFace,
			FrontFace: o.FrontFace,
		}
	}

	// Calculate solve duration (excluding scramble if present)
	var solveDurationMs int64
	if len(phases) > 0 {
		// Find first non-scramble phase
		for _, p := range phases {
			if p.PhaseKey != "scramble" && p.PhaseKey != "inspection" {
				solveDurationMs = phases[len(phases)-1].EndTsMs - p.StartTsMs
				break
			}
		}
	}

	var totalDurationMs int64
	if solve.DurationMs != nil {
		totalDurationMs = *solve.DurationMs
	} else if len(phases) > 0 {
		totalDurationMs = phases[len(phases)-1].EndTsMs
	} else if len(moves) > 0 {
		totalDurationMs = moves[len(moves)-1].TsMs + 1000 // Add 1 second buffer
	}

	return VisualizerData{
		SolveID:         solve.SolveID,
		TotalDurationMs: totalDurationMs,
		SolveDurationMs: solveDurationMs,
		Phases:          vizPhases,
		Moves:           vizMoves,
		Orientations:    vizOrients,
		Report:          report,
	}
}

// generateVisualizerHTML creates the standalone HTML visualization file.
func generateVisualizerHTML(
	reportDir string,
	solve *storage.Solve,
	moves []storage.MoveRecord,
	phases []storage.PhaseSegment,
	orientations []storage.OrientationRecord,
	phaseDefMap map[string]string,
	report *VisualizerReport,
) error {
	// Build the data structure
	data := buildVisualizerData(solve, moves, phases, orientations, phaseDefMap, report)

	// Convert to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling visualizer data: %w", err)
	}

	// Parse the template
	tmpl, err := template.New("visualizer").Parse(visualizerTemplate)
	if err != nil {
		return fmt.Errorf("parsing visualizer template: %w", err)
	}

	// Create output file
	outputPath := filepath.Join(reportDir, "visualizer.html")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating visualizer file: %w", err)
	}
	defer f.Close()

	// Execute template with JSON data
	templateData := map[string]template.JS{
		"SolveDataJSON": template.JS(jsonData),
	}

	if err := tmpl.Execute(f, templateData); err != nil {
		return fmt.Errorf("executing visualizer template: %w", err)
	}

	return nil
}
