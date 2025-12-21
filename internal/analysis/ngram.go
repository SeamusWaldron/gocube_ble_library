package analysis

import (
	"sort"

	"github.com/seamusw/gocube/pkg/types"
)

// NGram represents a repeated move sequence.
type NGram struct {
	N           int      `json:"n"`
	Sequence    []string `json:"sequence"`
	Tokens      []uint8  `json:"-"`
	Count       int      `json:"count"`
	Occurrences []NGramOccurrence `json:"occurrences,omitempty"`
}

// NGramOccurrence represents where an n-gram was found.
type NGramOccurrence struct {
	SolveID    string `json:"solve_id,omitempty"`
	StartIndex int    `json:"start_index"`
	TsMs       int64  `json:"ts_ms"`
}

// NGramReport contains the results of n-gram mining.
type NGramReport struct {
	TopNGrams map[int][]NGram `json:"top_ngrams"` // Keyed by n
}

// RollingHash implements Rabin-Karp rolling hash for efficient n-gram detection.
type RollingHash struct {
	base   uint64
	hash   uint64
	pow    uint64 // base^(n-1) for removal
	window []uint8
	n      int
}

// NewRollingHash creates a new rolling hash for window size n.
func NewRollingHash(n int) *RollingHash {
	rh := &RollingHash{
		base:   31, // Prime base
		n:      n,
		window: make([]uint8, 0, n),
	}

	// Precompute base^(n-1)
	rh.pow = 1
	for i := 0; i < n-1; i++ {
		rh.pow *= rh.base
	}

	return rh
}

// Add adds a token to the rolling hash.
func (rh *RollingHash) Add(token uint8) {
	if len(rh.window) < rh.n {
		rh.window = append(rh.window, token)
		rh.hash = rh.hash*rh.base + uint64(token)
	}
}

// Roll removes the oldest token and adds a new one.
func (rh *RollingHash) Roll(token uint8) {
	if len(rh.window) < rh.n {
		rh.Add(token)
		return
	}

	old := rh.window[0]
	rh.hash = (rh.hash-uint64(old)*rh.pow)*rh.base + uint64(token)

	// Shift window
	copy(rh.window, rh.window[1:])
	rh.window[rh.n-1] = token
}

// Hash returns the current hash value.
func (rh *RollingHash) Hash() uint64 {
	return rh.hash
}

// Window returns a copy of the current window.
func (rh *RollingHash) Window() []uint8 {
	result := make([]uint8, len(rh.window))
	copy(result, rh.window)
	return result
}

// Ready returns true if the window is full.
func (rh *RollingHash) Ready() bool {
	return len(rh.window) == rh.n
}

// ngramEntry tracks n-gram occurrences during mining.
type ngramEntry struct {
	tokens      []uint8
	count       int
	occurrences []NGramOccurrence
}

// MineNGrams finds the top-K most frequent n-grams for each n in [minN, maxN].
func MineNGrams(moves []types.Move, minN, maxN, topK int) *NGramReport {
	report := &NGramReport{
		TopNGrams: make(map[int][]NGram),
	}

	if len(moves) < minN {
		return report
	}

	// Convert moves to tokens
	tokens := make([]uint8, len(moves))
	for i, m := range moves {
		tokens[i] = m.Token()
	}

	// Mine for each n
	for n := minN; n <= maxN && n <= len(moves); n++ {
		ngrams := mineNGramsForN(tokens, moves, n, topK)
		if len(ngrams) > 0 {
			report.TopNGrams[n] = ngrams
		}
	}

	return report
}

// mineNGramsForN mines n-grams of a specific length.
func mineNGramsForN(tokens []uint8, moves []types.Move, n, topK int) []NGram {
	if len(tokens) < n {
		return nil
	}

	// Use rolling hash to count n-grams
	counts := make(map[uint64]*ngramEntry)
	rh := NewRollingHash(n)

	// Initialize with first n-1 tokens
	for i := 0; i < n-1 && i < len(tokens); i++ {
		rh.Add(tokens[i])
	}

	// Roll through the rest
	for i := n - 1; i < len(tokens); i++ {
		rh.Roll(tokens[i])

		if !rh.Ready() {
			continue
		}

		hash := rh.Hash()
		startIdx := i - n + 1

		if entry, exists := counts[hash]; exists {
			// Verify it's actually the same sequence (handle hash collisions)
			window := rh.Window()
			if slicesEqual(entry.tokens, window) {
				entry.count++
				if len(entry.occurrences) < 10 {
					entry.occurrences = append(entry.occurrences, NGramOccurrence{
						StartIndex: startIdx,
						TsMs:       moves[startIdx].Timestamp,
					})
				}
			}
		} else {
			counts[hash] = &ngramEntry{
				tokens: rh.Window(),
				count:  1,
				occurrences: []NGramOccurrence{{
					StartIndex: startIdx,
					TsMs:       moves[startIdx].Timestamp,
				}},
			}
		}
	}

	// Convert to sorted list and take top K
	entries := make([]*ngramEntry, 0, len(counts))
	for _, entry := range counts {
		// Only include n-grams that appear more than once
		if entry.count >= 2 {
			entries = append(entries, entry)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	// Take top K
	if len(entries) > topK {
		entries = entries[:topK]
	}

	// Convert to NGram structs
	result := make([]NGram, len(entries))
	for i, entry := range entries {
		sequence := make([]string, len(entry.tokens))
		for j, token := range entry.tokens {
			move := types.MoveFromToken(token)
			sequence[j] = move.Notation()
		}

		result[i] = NGram{
			N:           n,
			Sequence:    sequence,
			Tokens:      entry.tokens,
			Count:       entry.count,
			Occurrences: entry.occurrences,
		}
	}

	return result
}

// slicesEqual compares two uint8 slices.
func slicesEqual(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// MineNGramsAcrossSolves aggregates n-grams across multiple solves.
func MineNGramsAcrossSolves(solveNGrams map[string]*NGramReport, topK int) *NGramReport {
	report := &NGramReport{
		TopNGrams: make(map[int][]NGram),
	}

	// Aggregate counts per n
	for n := 4; n <= 14; n++ {
		aggregated := make(map[string]*NGram) // Key: sequence string

		for solveID, solveReport := range solveNGrams {
			ngrams, ok := solveReport.TopNGrams[n]
			if !ok {
				continue
			}

			for _, ng := range ngrams {
				key := ngramKey(ng.Tokens)
				if existing, exists := aggregated[key]; exists {
					existing.Count += ng.Count
					// Add sample occurrences from this solve
					for _, occ := range ng.Occurrences {
						if len(existing.Occurrences) < 10 {
							occ.SolveID = solveID
							existing.Occurrences = append(existing.Occurrences, occ)
						}
					}
				} else {
					newNg := NGram{
						N:           ng.N,
						Sequence:    ng.Sequence,
						Tokens:      ng.Tokens,
						Count:       ng.Count,
						Occurrences: make([]NGramOccurrence, 0, 10),
					}
					for _, occ := range ng.Occurrences {
						if len(newNg.Occurrences) < 10 {
							occ.SolveID = solveID
							newNg.Occurrences = append(newNg.Occurrences, occ)
						}
					}
					aggregated[key] = &newNg
				}
			}
		}

		// Convert to sorted list
		ngrams := make([]NGram, 0, len(aggregated))
		for _, ng := range aggregated {
			ngrams = append(ngrams, *ng)
		}

		sort.Slice(ngrams, func(i, j int) bool {
			return ngrams[i].Count > ngrams[j].Count
		})

		if len(ngrams) > topK {
			ngrams = ngrams[:topK]
		}

		if len(ngrams) > 0 {
			report.TopNGrams[n] = ngrams
		}
	}

	return report
}

// ngramKey creates a string key for an n-gram token sequence.
func ngramKey(tokens []uint8) string {
	result := make([]byte, len(tokens))
	for i, t := range tokens {
		result[i] = t + 'A' // Make printable
	}
	return string(result)
}
