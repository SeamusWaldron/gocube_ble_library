package analysis

import (
	"github.com/SeamusWaldron/gocube_ble_library"
)

// SolveSummary contains comprehensive statistics for a single solve.
type SolveSummary struct {
	SolveID              string         `json:"solve_id"`
	StartedAt            string         `json:"started_at"`
	EndedAt              string         `json:"ended_at,omitempty"`
	DurationMs           int64          `json:"duration_ms"`
	TotalMoves           int            `json:"total_moves"`
	OptimizedMoves       int            `json:"optimized_moves"`
	Efficiency           float64        `json:"efficiency"`
	TPSOverall           float64        `json:"tps_overall"`
	PhaseStats           []PhaseStats   `json:"phase_stats,omitempty"`
	LongestPauseMs       int64          `json:"longest_pause_ms"`
	PauseCountOver1500   int            `json:"pause_count_over_1500ms"`
	AvgMoveDurationMs    float64        `json:"avg_move_duration_ms"`
	Notes                string         `json:"notes,omitempty"`
}

// PhaseStats contains statistics for a single phase.
type PhaseStats struct {
	PhaseKey    string  `json:"phase_key"`
	DisplayName string  `json:"display_name"`
	StartTsMs   int64   `json:"start_ts_ms"`
	EndTsMs     int64   `json:"end_ts_ms"`
	DurationMs  int64   `json:"duration_ms"`
	MoveCount   int     `json:"move_count"`
	TPS         float64 `json:"tps"`
}

// PauseInfo represents a pause during solving.
type PauseInfo struct {
	AfterMoveIndex int   `json:"after_move_index"`
	DurationMs     int64 `json:"duration_ms"`
	TsMs           int64 `json:"ts_ms"`
}

// AnalyzePauses finds all significant pauses in a move sequence.
func AnalyzePauses(moves []gocube.Move, thresholdMs int64) []PauseInfo {
	var pauses []PauseInfo

	for i := 1; i < len(moves); i++ {
		gap := moves[i].Timestamp - moves[i-1].Timestamp
		if gap >= thresholdMs {
			pauses = append(pauses, PauseInfo{
				AfterMoveIndex: i - 1,
				DurationMs:     gap,
				TsMs:           moves[i-1].Timestamp,
			})
		}
	}

	return pauses
}

// CalculateTPS calculates turns per second for a move sequence.
func CalculateTPS(moves []gocube.Move, durationMs int64) float64 {
	if durationMs <= 0 {
		return 0
	}
	return float64(len(moves)) / (float64(durationMs) / 1000.0)
}

// CalculateAvgMoveDuration calculates the average time between moves.
func CalculateAvgMoveDuration(moves []gocube.Move) float64 {
	if len(moves) < 2 {
		return 0
	}

	totalGap := moves[len(moves)-1].Timestamp - moves[0].Timestamp
	return float64(totalGap) / float64(len(moves)-1)
}

// FindLongestPause finds the longest pause in a move sequence.
func FindLongestPause(moves []gocube.Move) int64 {
	var longest int64

	for i := 1; i < len(moves); i++ {
		gap := moves[i].Timestamp - moves[i-1].Timestamp
		if gap > longest {
			longest = gap
		}
	}

	return longest
}

// CountPausesOver counts pauses over a threshold.
func CountPausesOver(moves []gocube.Move, thresholdMs int64) int {
	count := 0
	for i := 1; i < len(moves); i++ {
		gap := moves[i].Timestamp - moves[i-1].Timestamp
		if gap > thresholdMs {
			count++
		}
	}
	return count
}

// MovementProfile analyzes the movement patterns in a solve.
type MovementProfile struct {
	FaceCounts    map[gocube.Face]int     `json:"face_counts"`
	TurnCounts    map[gocube.Turn]int     `json:"turn_counts"`
	MostUsedFace  gocube.Face             `json:"most_used_face"`
	MostUsedTurn  gocube.Turn             `json:"most_used_turn"`
	FaceSequences map[string]int         `json:"face_sequences"` // e.g., "RU" -> count
}

// AnalyzeMovementProfile analyzes which faces and turns are most used.
func AnalyzeMovementProfile(moves []gocube.Move) *MovementProfile {
	profile := &MovementProfile{
		FaceCounts:    make(map[gocube.Face]int),
		TurnCounts:    make(map[gocube.Turn]int),
		FaceSequences: make(map[string]int),
	}

	for i, m := range moves {
		profile.FaceCounts[m.Face]++
		profile.TurnCounts[m.Turn]++

		// Track 2-move face sequences
		if i > 0 {
			seq := string(moves[i-1].Face) + string(m.Face)
			profile.FaceSequences[seq]++
		}
	}

	// Find most used
	maxFaceCount := 0
	for face, count := range profile.FaceCounts {
		if count > maxFaceCount {
			maxFaceCount = count
			profile.MostUsedFace = face
		}
	}

	maxTurnCount := 0
	for turn, count := range profile.TurnCounts {
		if count > maxTurnCount {
			maxTurnCount = count
			profile.MostUsedTurn = turn
		}
	}

	return profile
}
