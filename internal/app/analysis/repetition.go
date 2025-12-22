// Package analysis provides solve analysis algorithms.
package analysis

import (
	"github.com/SeamusWaldron/gocube_ble_library"
)

// Cancellation represents an immediate move cancellation (e.g., R followed by R').
type Cancellation struct {
	Index1    int    `json:"index1"`
	Index2    int    `json:"index2"`
	Move1     string `json:"move1"`
	Move2     string `json:"move2"`
	TsMs      int64  `json:"ts_ms"`
}

// MergeOpportunity represents adjacent same-face moves that could be merged.
type MergeOpportunity struct {
	Index1     int    `json:"index1"`
	Index2     int    `json:"index2"`
	Move1      string `json:"move1"`
	Move2      string `json:"move2"`
	MergedMove string `json:"merged_move"`
	TsMs       int64  `json:"ts_ms"`
}

// BackAndForthPattern represents alternating moves (e.g., R U R U R U).
type BackAndForthPattern struct {
	StartIndex int      `json:"start_index"`
	EndIndex   int      `json:"end_index"`
	Pattern    []string `json:"pattern"`
	Count      int      `json:"count"`
	TsMs       int64    `json:"ts_ms"`
}

// RepetitionReport contains all repetition analysis results.
type RepetitionReport struct {
	ImmediateCancellations []Cancellation        `json:"immediate_cancellations"`
	MergeOpportunities     []MergeOpportunity    `json:"merge_opportunities"`
	BackAndForthPatterns   []BackAndForthPattern `json:"back_and_forth_patterns"`
	TotalWastedMoves       int                   `json:"total_wasted_moves"`
}

// AnalyzeRepetitions analyzes a move sequence for repetitions and wasted motion.
func AnalyzeRepetitions(moves []gocube.Move) *RepetitionReport {
	report := &RepetitionReport{
		ImmediateCancellations: []Cancellation{},
		MergeOpportunities:     []MergeOpportunity{},
		BackAndForthPatterns:   []BackAndForthPattern{},
	}

	if len(moves) < 2 {
		return report
	}

	// Find immediate cancellations
	for i := 0; i < len(moves)-1; i++ {
		m1, m2 := moves[i], moves[i+1]

		// Check for cancellation (R followed by R')
		if m1.Face == m2.Face && m1.Turn == -m2.Turn {
			report.ImmediateCancellations = append(report.ImmediateCancellations, Cancellation{
				Index1: i,
				Index2: i + 1,
				Move1:  m1.Notation(),
				Move2:  m2.Notation(),
				TsMs:   m1.Time.UnixMilli(),
			})
			report.TotalWastedMoves += 2
		}

		// Check for merge opportunity (R followed by R = R2, but not already R2)
		if m1.Face == m2.Face && m1.Turn != -m2.Turn {
			// Only report if not already optimal
			merged := mergeMoves(m1, m2)
			if merged != nil {
				// This is a merge opportunity (not a full cancellation)
				report.MergeOpportunities = append(report.MergeOpportunities, MergeOpportunity{
					Index1:     i,
					Index2:     i + 1,
					Move1:      m1.Notation(),
					Move2:      m2.Notation(),
					MergedMove: merged.Notation(),
					TsMs:       m1.Time.UnixMilli(),
				})
				report.TotalWastedMoves += 1
			}
		}
	}

	// Find back-and-forth patterns (alternating moves)
	report.BackAndForthPatterns = findBackAndForth(moves)

	return report
}

// findBackAndForth finds alternating move patterns like R U R U R U.
func findBackAndForth(moves []gocube.Move) []BackAndForthPattern {
	var patterns []BackAndForthPattern

	if len(moves) < 4 {
		return patterns
	}

	i := 0
	for i < len(moves)-3 {
		// Look for pattern AB repeated
		a, b := moves[i], moves[i+1]

		// Check if we have at least 2 repetitions of AB
		count := 1
		j := i + 2
		for j < len(moves)-1 {
			if moves[j].Face == a.Face && moves[j].Turn == a.Turn &&
				moves[j+1].Face == b.Face && moves[j+1].Turn == b.Turn {
				count++
				j += 2
			} else {
				break
			}
		}

		// Require at least 3 repetitions to be noteworthy
		if count >= 3 {
			pattern := []string{a.Notation(), b.Notation()}
			patterns = append(patterns, BackAndForthPattern{
				StartIndex: i,
				EndIndex:   i + count*2 - 1,
				Pattern:    pattern,
				Count:      count,
				TsMs:       a.Time.UnixMilli(),
			})
			i = j // Skip past the pattern
		} else {
			i++
		}
	}

	return patterns
}

// OptimizeMoves returns an optimized move sequence with cancellations and merges applied.
func OptimizeMoves(moves []gocube.Move) []gocube.Move {
	if len(moves) == 0 {
		return moves
	}

	result := make([]gocube.Move, 0, len(moves))

	for _, move := range moves {
		if len(result) == 0 {
			result = append(result, move)
			continue
		}

		last := &result[len(result)-1]
		if last.Face == move.Face {
			merged := mergeMoves(*last, move)
			if merged == nil {
				// Full cancellation
				result = result[:len(result)-1]
			} else {
				// Merge
				*last = *merged
			}
		} else {
			result = append(result, move)
		}
	}

	return result
}

// CalculateEfficiency calculates the efficiency ratio (optimized/original).
func CalculateEfficiency(original, optimized []gocube.Move) float64 {
	if len(original) == 0 {
		return 1.0
	}
	return float64(len(optimized)) / float64(len(original))
}
