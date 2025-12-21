package cube

import "github.com/seamusw/gocube/pkg/types"

// ApplyMove applies a types.Move to the cube.
func (c *Cube) ApplyMove(m types.Move) {
	face := typesFaceToFace(m.Face)
	turn := int(m.Turn)
	c.Move(face, turn)
}

// ApplyMoves applies a sequence of moves to the cube.
func (c *Cube) ApplyMoves(moves []types.Move) {
	for _, m := range moves {
		c.ApplyMove(m)
	}
}

// typesFaceToFace converts types.Face to cube.Face.
func typesFaceToFace(f types.Face) Face {
	switch f {
	case types.FaceU:
		return U
	case types.FaceD:
		return D
	case types.FaceF:
		return F
	case types.FaceB:
		return B
	case types.FaceR:
		return R
	case types.FaceL:
		return L
	default:
		return U
	}
}

// Tracker wraps a Cube and provides state change detection.
type Tracker struct {
	cube          *Cube
	lastPhase     DetectedPhase
	highestPhase  DetectedPhase // Monotonic - never goes backwards
	phaseCallback func(phase DetectedPhase, phaseKey string)
}

// NewTracker creates a new cube tracker starting from a solved state.
func NewTracker() *Tracker {
	return &Tracker{
		cube:      New(),
		lastPhase: PhaseSolved,
	}
}

// SetPhaseCallback sets a callback that fires when a phase is completed.
func (t *Tracker) SetPhaseCallback(cb func(phase DetectedPhase, phaseKey string)) {
	t.phaseCallback = cb
}

// Reset resets the tracker to a solved cube state.
func (t *Tracker) Reset() {
	t.cube = New()
	t.lastPhase = PhaseSolved
	t.highestPhase = PhaseScrambled // Start at lowest phase
}

// ApplyMove applies a move and checks for phase transitions.
func (t *Tracker) ApplyMove(m types.Move) {
	t.cube.ApplyMove(m)
	t.checkPhaseTransition()
}

// ApplyMoves applies multiple moves.
func (t *Tracker) ApplyMoves(moves []types.Move) {
	for _, m := range moves {
		t.ApplyMove(m)
	}
}

// checkPhaseTransition checks if we've completed a new phase.
func (t *Tracker) checkPhaseTransition() {
	currentPhase := t.cube.DetectPhase()

	// Track current state for display purposes
	t.lastPhase = currentPhase

	// Only trigger callback and update highest phase when reaching a NEW high
	// (phase values are ordered from scrambled to solved)
	// This is monotonic - once you've reached a phase, we don't go backwards
	if currentPhase > t.highestPhase {
		t.highestPhase = currentPhase
		if t.phaseCallback != nil {
			t.phaseCallback(currentPhase, currentPhase.String())
		}
	}
}

// CurrentPhase returns the current detected phase.
func (t *Tracker) CurrentPhase() DetectedPhase {
	return t.cube.DetectPhase()
}

// CurrentPhaseKey returns the current phase as a key string.
// This reflects the raw cube state and may go backwards during solving.
func (t *Tracker) CurrentPhaseKey() string {
	return t.CurrentPhase().String()
}

// HighestPhaseKey returns the highest phase reached as a key string.
// This is monotonic and never goes backwards - use for phase marking.
func (t *Tracker) HighestPhaseKey() string {
	return t.highestPhase.String()
}

// HighestPhase returns the highest phase reached.
func (t *Tracker) HighestPhase() DetectedPhase {
	return t.highestPhase
}

// GetProgress returns the detailed progress.
func (t *Tracker) GetProgress() PhaseProgress {
	return t.cube.GetProgress()
}

// IsSolved returns true if the cube is solved.
func (t *Tracker) IsSolved() bool {
	return t.cube.IsSolved()
}

// Cube returns the underlying cube for inspection.
func (t *Tracker) Cube() *Cube {
	return t.cube
}

// CubeString returns a string representation of the cube.
func (t *Tracker) CubeString() string {
	return t.cube.String()
}
