package gocube

// Phase represents the current solving phase in the layer-by-layer method.
// Phases progress from Scrambled (0) to Solved (7), allowing comparison
// with < and > operators.
type Phase int

const (
	// PhaseScrambled indicates the cube is in a scrambled state.
	PhaseScrambled Phase = iota

	// PhaseWhiteCross indicates the white cross is complete.
	// The 4 white edge pieces are correctly positioned on the U face
	// with their adjacent colors matching the center colors.
	PhaseWhiteCross

	// PhaseFirstLayer indicates the first layer (white face) is complete.
	// All 4 white corners are correctly positioned and oriented.
	PhaseFirstLayer

	// PhaseSecondLayer indicates the second (middle) layer is complete.
	// All 4 middle layer edges are correctly positioned.
	PhaseSecondLayer

	// PhaseYellowCross indicates the yellow cross is formed.
	// The 4 yellow edge pieces are showing yellow on the D face.
	PhaseYellowCross

	// PhaseYellowCorners indicates the yellow corners are positioned.
	// All 4 bottom corners are in their correct positions (may be mis-oriented).
	PhaseYellowCorners

	// PhaseYellowOriented indicates the yellow corners are oriented.
	// All 4 bottom corners are correctly oriented with yellow on D face.
	PhaseYellowOriented

	// PhaseSolved indicates the cube is completely solved.
	PhaseSolved
)

// String returns a short identifier for the phase.
func (p Phase) String() string {
	switch p {
	case PhaseScrambled:
		return "scrambled"
	case PhaseWhiteCross:
		return "white_cross"
	case PhaseFirstLayer:
		return "first_layer"
	case PhaseSecondLayer:
		return "second_layer"
	case PhaseYellowCross:
		return "yellow_cross"
	case PhaseYellowCorners:
		return "yellow_corners"
	case PhaseYellowOriented:
		return "yellow_oriented"
	case PhaseSolved:
		return "solved"
	default:
		return "unknown"
	}
}

// DisplayName returns a human-readable name for the phase.
func (p Phase) DisplayName() string {
	switch p {
	case PhaseScrambled:
		return "Scrambled"
	case PhaseWhiteCross:
		return "White Cross"
	case PhaseFirstLayer:
		return "First Layer"
	case PhaseSecondLayer:
		return "Second Layer (F2L)"
	case PhaseYellowCross:
		return "Yellow Cross"
	case PhaseYellowCorners:
		return "Yellow Corners Positioned"
	case PhaseYellowOriented:
		return "Yellow Corners Oriented"
	case PhaseSolved:
		return "Solved"
	default:
		return "Unknown"
	}
}

// IsComplete returns true if the cube is solved.
func (p Phase) IsComplete() bool {
	return p == PhaseSolved
}

// Progress represents which phases have been completed.
type Progress struct {
	WhiteCross     bool
	FirstLayer     bool
	SecondLayer    bool
	YellowCross    bool
	YellowCorners  bool
	YellowOriented bool
	Solved         bool
}
