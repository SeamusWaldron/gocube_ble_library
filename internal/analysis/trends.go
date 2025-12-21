package analysis

import (
	"sort"
	"time"
)

// SolveData represents minimal solve data for trend analysis.
type SolveData struct {
	SolveID    string
	StartedAt  time.Time
	DurationMs int64
	MoveCount  int
	TPS        float64
	PhaseData  map[string]PhaseData
}

// PhaseData represents phase data for a single solve.
type PhaseData struct {
	DurationMs int64
	MoveCount  int
	TPS        float64
}

// TrendReport contains trend analysis across multiple solves.
type TrendReport struct {
	WindowSize       int              `json:"window_size"`
	TotalSolves      int              `json:"total_solves"`
	CompletedSolves  int              `json:"completed_solves"`
	DateRange        DateRange        `json:"date_range"`

	// Overall trends
	AvgDurationMs    float64          `json:"avg_duration_ms"`
	AvgMoves         float64          `json:"avg_moves"`
	AvgTPS           float64          `json:"avg_tps"`

	// Best/worst
	BestSolve        SolveStats       `json:"best_solve"`
	WorstSolve       SolveStats       `json:"worst_solve"`

	// Improvement metrics
	ImprovementPct   float64          `json:"improvement_pct"`
	ConsistencyScore float64          `json:"consistency_score"`

	// Per-phase trends
	PhaseTrends      map[string]PhaseTrend `json:"phase_trends"`

	// Rolling averages (last 5, 10, 25, 50)
	RollingAvgs      map[int]float64  `json:"rolling_averages"`

	// Solve list
	Solves           []SolveStats     `json:"solves"`
}

// DateRange represents a date range.
type DateRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// SolveStats represents statistics for a single solve in trend context.
type SolveStats struct {
	SolveID    string  `json:"solve_id"`
	Timestamp  string  `json:"timestamp"`
	DurationMs int64   `json:"duration_ms"`
	MoveCount  int     `json:"move_count"`
	TPS        float64 `json:"tps"`
}

// PhaseTrend represents trends for a specific phase.
type PhaseTrend struct {
	PhaseKey       string  `json:"phase_key"`
	AvgDurationMs  float64 `json:"avg_duration_ms"`
	AvgMoves       float64 `json:"avg_moves"`
	AvgTPS         float64 `json:"avg_tps"`
	ImprovementPct float64 `json:"improvement_pct"`
}

// AnalyzeTrends analyzes trends across multiple solves.
func AnalyzeTrends(solves []SolveData) *TrendReport {
	report := &TrendReport{
		WindowSize:  len(solves),
		TotalSolves: len(solves),
		PhaseTrends: make(map[string]PhaseTrend),
		RollingAvgs: make(map[int]float64),
		Solves:      make([]SolveStats, 0, len(solves)),
	}

	if len(solves) == 0 {
		return report
	}

	// Sort by time
	sort.Slice(solves, func(i, j int) bool {
		return solves[i].StartedAt.Before(solves[j].StartedAt)
	})

	// Date range
	report.DateRange = DateRange{
		Start: solves[0].StartedAt.Format(time.RFC3339),
		End:   solves[len(solves)-1].StartedAt.Format(time.RFC3339),
	}

	// Calculate averages and find best/worst
	var totalDuration, totalMoves int64
	var totalTPS float64
	var bestDuration, worstDuration int64 = -1, -1
	var bestSolve, worstSolve *SolveData

	completedSolves := []SolveData{}

	for i := range solves {
		s := &solves[i]
		if s.DurationMs <= 0 {
			continue
		}

		completedSolves = append(completedSolves, *s)
		totalDuration += s.DurationMs
		totalMoves += int64(s.MoveCount)
		totalTPS += s.TPS

		report.Solves = append(report.Solves, SolveStats{
			SolveID:    s.SolveID,
			Timestamp:  s.StartedAt.Format(time.RFC3339),
			DurationMs: s.DurationMs,
			MoveCount:  s.MoveCount,
			TPS:        s.TPS,
		})

		if bestDuration < 0 || s.DurationMs < bestDuration {
			bestDuration = s.DurationMs
			bestSolve = s
		}
		if worstDuration < 0 || s.DurationMs > worstDuration {
			worstDuration = s.DurationMs
			worstSolve = s
		}
	}

	report.CompletedSolves = len(completedSolves)

	if len(completedSolves) > 0 {
		report.AvgDurationMs = float64(totalDuration) / float64(len(completedSolves))
		report.AvgMoves = float64(totalMoves) / float64(len(completedSolves))
		report.AvgTPS = totalTPS / float64(len(completedSolves))

		if bestSolve != nil {
			report.BestSolve = SolveStats{
				SolveID:    bestSolve.SolveID,
				Timestamp:  bestSolve.StartedAt.Format(time.RFC3339),
				DurationMs: bestSolve.DurationMs,
				MoveCount:  bestSolve.MoveCount,
				TPS:        bestSolve.TPS,
			}
		}

		if worstSolve != nil {
			report.WorstSolve = SolveStats{
				SolveID:    worstSolve.SolveID,
				Timestamp:  worstSolve.StartedAt.Format(time.RFC3339),
				DurationMs: worstSolve.DurationMs,
				MoveCount:  worstSolve.MoveCount,
				TPS:        worstSolve.TPS,
			}
		}
	}

	// Calculate improvement (compare first quarter to last quarter)
	report.ImprovementPct = calculateImprovement(completedSolves)

	// Calculate consistency (coefficient of variation)
	report.ConsistencyScore = calculateConsistency(completedSolves)

	// Rolling averages
	for _, n := range []int{5, 10, 25, 50} {
		if len(completedSolves) >= n {
			recent := completedSolves[len(completedSolves)-n:]
			var sum int64
			for _, s := range recent {
				sum += s.DurationMs
			}
			report.RollingAvgs[n] = float64(sum) / float64(n)
		}
	}

	// Phase trends
	report.PhaseTrends = analyzePhasetrends(completedSolves)

	return report
}

// calculateImprovement calculates improvement percentage from first to last quarter.
func calculateImprovement(solves []SolveData) float64 {
	if len(solves) < 4 {
		return 0
	}

	quarterSize := len(solves) / 4
	if quarterSize == 0 {
		quarterSize = 1
	}

	// First quarter average
	var firstSum int64
	for i := 0; i < quarterSize; i++ {
		firstSum += solves[i].DurationMs
	}
	firstAvg := float64(firstSum) / float64(quarterSize)

	// Last quarter average
	var lastSum int64
	for i := len(solves) - quarterSize; i < len(solves); i++ {
		lastSum += solves[i].DurationMs
	}
	lastAvg := float64(lastSum) / float64(quarterSize)

	if firstAvg <= 0 {
		return 0
	}

	// Improvement is reduction in time (negative = got worse)
	return ((firstAvg - lastAvg) / firstAvg) * 100
}

// calculateConsistency calculates a consistency score (0-100, higher = more consistent).
func calculateConsistency(solves []SolveData) float64 {
	if len(solves) < 2 {
		return 100
	}

	// Calculate mean
	var sum float64
	for _, s := range solves {
		sum += float64(s.DurationMs)
	}
	mean := sum / float64(len(solves))

	// Calculate standard deviation
	var sumSquares float64
	for _, s := range solves {
		diff := float64(s.DurationMs) - mean
		sumSquares += diff * diff
	}
	stdDev := (sumSquares / float64(len(solves))) // variance

	// Coefficient of variation (CV) = stdDev / mean
	if mean <= 0 {
		return 100
	}
	cv := stdDev / (mean * mean) // Normalized

	// Convert to 0-100 score (lower CV = higher score)
	// CV of 0 = 100, CV of 0.5 = 50, CV of 1+ = 0
	score := 100 - (cv * 100)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// analyzePhasetrends analyzes trends for each phase.
func analyzePhasetrends(solves []SolveData) map[string]PhaseTrend {
	trends := make(map[string]PhaseTrend)

	// Collect phase data
	phaseData := make(map[string][]PhaseData)
	for _, s := range solves {
		for phaseKey, data := range s.PhaseData {
			phaseData[phaseKey] = append(phaseData[phaseKey], data)
		}
	}

	// Calculate trends per phase
	for phaseKey, data := range phaseData {
		if len(data) == 0 {
			continue
		}

		var totalDuration, totalMoves int64
		var totalTPS float64

		for _, d := range data {
			totalDuration += d.DurationMs
			totalMoves += int64(d.MoveCount)
			totalTPS += d.TPS
		}

		n := float64(len(data))
		trend := PhaseTrend{
			PhaseKey:      phaseKey,
			AvgDurationMs: float64(totalDuration) / n,
			AvgMoves:      float64(totalMoves) / n,
			AvgTPS:        totalTPS / n,
		}

		// Calculate improvement for this phase
		if len(data) >= 4 {
			quarterSize := len(data) / 4
			if quarterSize == 0 {
				quarterSize = 1
			}

			var firstSum, lastSum int64
			for i := 0; i < quarterSize; i++ {
				firstSum += data[i].DurationMs
			}
			for i := len(data) - quarterSize; i < len(data); i++ {
				lastSum += data[i].DurationMs
			}

			firstAvg := float64(firstSum) / float64(quarterSize)
			lastAvg := float64(lastSum) / float64(quarterSize)

			if firstAvg > 0 {
				trend.ImprovementPct = ((firstAvg - lastAvg) / firstAvg) * 100
			}
		}

		trends[phaseKey] = trend
	}

	return trends
}
