package analysis

import (
	"github.com/SeamusWaldron/gocube"
)

// Tool represents a known algorithm/tool for the final phase.
type Tool struct {
	Name     string
	Sequence []gocube.Move
}

// RHS (Right Hand Sune) variants - for corner orientation
var (
	// RHS Forward: R U R' U R U2 R'
	RHSForward = Tool{
		Name: "RHS Forward",
		Sequence: []gocube.Move{
			{Face: gocube.FaceR, Turn: gocube.TurnCW},
			{Face: gocube.FaceU, Turn: gocube.TurnCW},
			{Face: gocube.FaceR, Turn: gocube.TurnCCW},
			{Face: gocube.FaceU, Turn: gocube.TurnCW},
			{Face: gocube.FaceR, Turn: gocube.TurnCW},
			{Face: gocube.FaceU, Turn: gocube.Turn180},
			{Face: gocube.FaceR, Turn: gocube.TurnCCW},
		},
	}

	// RHS Reverse: R U2 R' U' R U' R'
	RHSReverse = Tool{
		Name: "RHS Reverse",
		Sequence: []gocube.Move{
			{Face: gocube.FaceR, Turn: gocube.TurnCW},
			{Face: gocube.FaceU, Turn: gocube.Turn180},
			{Face: gocube.FaceR, Turn: gocube.TurnCCW},
			{Face: gocube.FaceU, Turn: gocube.TurnCCW},
			{Face: gocube.FaceR, Turn: gocube.TurnCW},
			{Face: gocube.FaceU, Turn: gocube.TurnCCW},
			{Face: gocube.FaceR, Turn: gocube.TurnCCW},
		},
	}

	// LHS Forward: L' U' L U' L' U2 L
	LHSForward = Tool{
		Name: "LHS Forward",
		Sequence: []gocube.Move{
			{Face: gocube.FaceL, Turn: gocube.TurnCCW},
			{Face: gocube.FaceU, Turn: gocube.TurnCCW},
			{Face: gocube.FaceL, Turn: gocube.TurnCW},
			{Face: gocube.FaceU, Turn: gocube.TurnCCW},
			{Face: gocube.FaceL, Turn: gocube.TurnCCW},
			{Face: gocube.FaceU, Turn: gocube.Turn180},
			{Face: gocube.FaceL, Turn: gocube.TurnCW},
		},
	}

	// LHS Reverse: L' U2 L U L' U L
	LHSReverse = Tool{
		Name: "LHS Reverse",
		Sequence: []gocube.Move{
			{Face: gocube.FaceL, Turn: gocube.TurnCCW},
			{Face: gocube.FaceU, Turn: gocube.Turn180},
			{Face: gocube.FaceL, Turn: gocube.TurnCW},
			{Face: gocube.FaceU, Turn: gocube.TurnCW},
			{Face: gocube.FaceL, Turn: gocube.TurnCCW},
			{Face: gocube.FaceU, Turn: gocube.TurnCW},
			{Face: gocube.FaceL, Turn: gocube.TurnCW},
		},
	}
)

// AllTools is a list of all known tools.
var AllTools = []Tool{RHSForward, RHSReverse, LHSForward, LHSReverse}

// ToolMatch represents a detected tool usage.
type ToolMatch struct {
	ToolName   string `json:"tool_name"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	TsMs       int64  `json:"ts_ms"`
}

// FinalPhaseReport contains the analysis of the final phase (bottom corner orientation).
type FinalPhaseReport struct {
	FinalPhaseMoveCount int          `json:"final_phase_move_count"`
	FinalPhaseDurationMs int64       `json:"final_phase_duration_ms"`
	RHSForwardCount     int          `json:"rhs_forward_count"`
	RHSReverseCount     int          `json:"rhs_reverse_count"`
	LHSForwardCount     int          `json:"lhs_forward_count"`
	LHSReverseCount     int          `json:"lhs_reverse_count"`
	TotalToolsUsed      int          `json:"total_tools_used"`
	ToolMatches         []ToolMatch  `json:"tool_matches"`
	ConsecutiveRepeats  int          `json:"consecutive_tool_repeats"`
	TimeBetweenToolsMs  []int64      `json:"time_between_tools_ms"`
	AvgTimeBetweenMs    float64      `json:"avg_time_between_tools_ms"`
	UnmatchedMoves      int          `json:"unmatched_moves"`
}

// AnalyzeFinalPhase analyzes the final phase (bottom_orient) of a solve.
func AnalyzeFinalPhase(moves []gocube.Move) *FinalPhaseReport {
	report := &FinalPhaseReport{
		FinalPhaseMoveCount: len(moves),
		ToolMatches:         []ToolMatch{},
		TimeBetweenToolsMs:  []int64{},
	}

	if len(moves) == 0 {
		return report
	}

	// Calculate duration
	if len(moves) > 1 {
		report.FinalPhaseDurationMs = moves[len(moves)-1].Timestamp - moves[0].Timestamp
	}

	// Find all tool matches
	matched := make([]bool, len(moves))
	var lastMatchEnd int = -1
	var lastMatchTs int64 = 0

	for i := 0; i < len(moves); i++ {
		if matched[i] {
			continue
		}

		for _, tool := range AllTools {
			if matchesTool(moves, i, tool.Sequence) {
				match := ToolMatch{
					ToolName:   tool.Name,
					StartIndex: i,
					EndIndex:   i + len(tool.Sequence) - 1,
					TsMs:       moves[i].Timestamp,
				}
				report.ToolMatches = append(report.ToolMatches, match)

				// Mark moves as matched
				for j := i; j < i+len(tool.Sequence); j++ {
					matched[j] = true
				}

				// Track time between tools
				if lastMatchEnd >= 0 && i > lastMatchEnd {
					gap := moves[i].Timestamp - lastMatchTs
					report.TimeBetweenToolsMs = append(report.TimeBetweenToolsMs, gap)
				}

				// Check for consecutive repeats
				if lastMatchEnd == i-1 {
					report.ConsecutiveRepeats++
				}

				lastMatchEnd = i + len(tool.Sequence) - 1
				lastMatchTs = moves[lastMatchEnd].Timestamp

				// Update counts
				switch tool.Name {
				case "RHS Forward":
					report.RHSForwardCount++
				case "RHS Reverse":
					report.RHSReverseCount++
				case "LHS Forward":
					report.LHSForwardCount++
				case "LHS Reverse":
					report.LHSReverseCount++
				}

				i = lastMatchEnd // Skip to end of this match
				break
			}
		}
	}

	report.TotalToolsUsed = report.RHSForwardCount + report.RHSReverseCount +
		report.LHSForwardCount + report.LHSReverseCount

	// Count unmatched moves
	for _, m := range matched {
		if !m {
			report.UnmatchedMoves++
		}
	}

	// Calculate average time between tools
	if len(report.TimeBetweenToolsMs) > 0 {
		var total int64
		for _, t := range report.TimeBetweenToolsMs {
			total += t
		}
		report.AvgTimeBetweenMs = float64(total) / float64(len(report.TimeBetweenToolsMs))
	}

	return report
}

// matchesTool checks if the move sequence starting at index matches the tool.
func matchesTool(moves []gocube.Move, startIdx int, tool []gocube.Move) bool {
	if startIdx+len(tool) > len(moves) {
		return false
	}

	for i, t := range tool {
		m := moves[startIdx+i]
		if m.Face != t.Face || m.Turn != t.Turn {
			return false
		}
	}

	return true
}

// DetectToolVariants detects tools with minor variations (e.g., setup moves).
func DetectToolVariants(moves []gocube.Move) []ToolMatch {
	var matches []ToolMatch

	// Look for partial matches or variations
	// This is a simplified version - could be enhanced with fuzzy matching

	for i := 0; i < len(moves); i++ {
		for _, tool := range AllTools {
			// Try matching with 1 move tolerance at start
			if i > 0 && matchesTool(moves, i, tool.Sequence) {
				matches = append(matches, ToolMatch{
					ToolName:   tool.Name + " (with setup)",
					StartIndex: i - 1,
					EndIndex:   i + len(tool.Sequence) - 1,
					TsMs:       moves[i-1].Timestamp,
				})
			}
		}
	}

	return matches
}

// SuggestImprovement suggests how to improve the final phase based on analysis.
func SuggestImprovement(report *FinalPhaseReport) []string {
	var suggestions []string

	if report.UnmatchedMoves > report.FinalPhaseMoveCount/4 {
		suggestions = append(suggestions, "High number of unmatched moves - consider learning more algorithms")
	}

	if report.AvgTimeBetweenMs > 2000 {
		suggestions = append(suggestions, "Long pauses between algorithms - practice recognition speed")
	}

	if report.ConsecutiveRepeats > 2 {
		suggestions = append(suggestions, "Multiple consecutive tool repeats - consider using different algorithm combinations")
	}

	// Check for imbalanced tool usage
	if report.RHSForwardCount > 0 && report.RHSReverseCount == 0 {
		suggestions = append(suggestions, "Only using RHS Forward - learn RHS Reverse for efficiency")
	}
	if report.LHSForwardCount == 0 && report.LHSReverseCount == 0 {
		suggestions = append(suggestions, "Not using LHS tools - they can be faster for some cases")
	}

	return suggestions
}
