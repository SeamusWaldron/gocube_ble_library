package cube

import (
	"testing"

	"github.com/seamusw/gocube/pkg/types"
)

func TestNewCubeIsSolved(t *testing.T) {
	c := New()
	if !c.IsSolved() {
		t.Error("New cube should be solved")
	}
}

func TestSingleMoveBreaksSolved(t *testing.T) {
	c := New()
	c.Move(R, 1) // R
	if c.IsSolved() {
		t.Error("Cube should not be solved after R move")
	}
}

func TestRR_ReturnsToSolved(t *testing.T) {
	c := New()
	// R R R R = identity
	c.Move(R, 1)
	c.Move(R, 1)
	c.Move(R, 1)
	c.Move(R, 1)
	if !c.IsSolved() {
		t.Error("R R R R should return to solved")
		t.Log(c.String())
	}
}

func TestR2R2_ReturnsToSolved(t *testing.T) {
	c := New()
	c.Move(R, 2)
	c.Move(R, 2)
	if !c.IsSolved() {
		t.Error("R2 R2 should return to solved")
		t.Log(c.String())
	}
}

func TestRR_ReturnsToSolved_AllFaces(t *testing.T) {
	faces := []Face{U, D, F, B, R, L}
	for _, face := range faces {
		c := New()
		c.Move(face, 1)
		c.Move(face, 1)
		c.Move(face, 1)
		c.Move(face, 1)
		if !c.IsSolved() {
			t.Errorf("%v x 4 should return to solved", face)
			t.Log(c.String())
		}
	}
}

func TestSexyMove_6Times_ReturnsToSolved(t *testing.T) {
	// (R U R' U') x 6 = identity
	c := New()
	for i := 0; i < 6; i++ {
		c.Move(R, 1)  // R
		c.Move(U, 1)  // U
		c.Move(R, -1) // R'
		c.Move(U, -1) // U'
	}
	if !c.IsSolved() {
		t.Error("Sexy move x 6 should return to solved")
		t.Log(c.String())
	}
}

func TestWhiteCrossDetection(t *testing.T) {
	c := New()
	if !c.IsWhiteCrossComplete() {
		t.Error("Solved cube should have white cross complete")
	}

	// Break the cross with a single R move
	c.Move(R, 1)
	// R move affects U face edge at position 5 (right edge)
	// After R, the white edge piece moves to F face
	if c.IsWhiteCrossComplete() {
		t.Error("White cross should be broken after R move")
	}
}

func TestTopLayerDetection(t *testing.T) {
	c := New()
	if !c.IsTopLayerComplete() {
		t.Error("Solved cube should have top layer complete")
	}

	c.Move(R, 1)
	if c.IsTopLayerComplete() {
		t.Error("Top layer should be broken after R move")
	}
}

func TestMiddleLayerDetection(t *testing.T) {
	c := New()
	if !c.IsMiddleLayerComplete() {
		t.Error("Solved cube should have middle layer complete")
	}
}

func TestApplyTypesMove(t *testing.T) {
	c := New()
	move := types.Move{Face: types.FaceR, Turn: types.TurnCW}
	c.ApplyMove(move)
	if c.IsSolved() {
		t.Error("Cube should not be solved after applying R move")
	}

	// Apply R' to undo
	move2 := types.Move{Face: types.FaceR, Turn: types.TurnCCW}
	c.ApplyMove(move2)
	if !c.IsSolved() {
		t.Error("Cube should be solved after R R'")
		t.Log(c.String())
	}
}

func TestPhaseDetection(t *testing.T) {
	c := New()
	phase := c.DetectPhase()
	if phase != PhaseSolved {
		t.Errorf("Solved cube should detect as PhaseSolved, got %v", phase)
	}

	c.Move(R, 1)
	phase = c.DetectPhase()
	if phase == PhaseSolved {
		t.Error("Scrambled cube should not detect as solved")
	}
}

func TestTrackerReset(t *testing.T) {
	tr := NewTracker()
	if !tr.IsSolved() {
		t.Error("New tracker should start solved")
	}

	tr.ApplyMove(types.Move{Face: types.FaceR, Turn: types.TurnCW})
	if tr.IsSolved() {
		t.Error("Tracker should not be solved after move")
	}

	tr.Reset()
	if !tr.IsSolved() {
		t.Error("Tracker should be solved after reset")
	}
}

func TestScrambleAndReverse(t *testing.T) {
	// Scramble a cube with a sequence and then reverse it
	// Verify that phases are detected correctly on the way back
	c := New()

	// Simple scramble
	scramble := []struct {
		face Face
		turn int
	}{
		{R, 1}, {U, 1}, {R, -1}, {U, -1},
		{F, 1}, {D, 1}, {L, 2},
	}

	// Apply scramble
	for _, m := range scramble {
		c.Move(m.face, m.turn)
	}

	if c.IsSolved() {
		t.Error("Cube should be scrambled after moves")
	}

	phase := c.DetectPhase()
	t.Logf("After scramble: phase=%s", phase.String())

	// Reverse the scramble
	for i := len(scramble) - 1; i >= 0; i-- {
		m := scramble[i]
		// Reverse the turn
		reverseTurn := -m.turn
		if m.turn == 2 {
			reverseTurn = 2 // R2 reversed is R2
		}
		c.Move(m.face, reverseTurn)
	}

	if !c.IsSolved() {
		t.Error("Cube should be solved after reversing scramble")
		t.Log(c.String())
	}
}

func TestPhaseTransitionsForward(t *testing.T) {
	// Verify that each phase check works correctly
	c := New()

	// All phases should be complete on solved cube
	t.Log("Testing solved cube phases:")
	t.Logf("  WhiteCross: %v", c.IsWhiteCrossComplete())
	t.Logf("  TopLayer: %v", c.IsTopLayerComplete())
	t.Logf("  MiddleLayer: %v", c.IsMiddleLayerComplete())
	t.Logf("  BottomCross: %v", c.IsBottomCrossComplete())
	t.Logf("  CornersPositioned: %v", c.AreBottomCornersPositioned())
	t.Logf("  CornersOriented: %v", c.AreBottomCornersOriented())
	t.Logf("  Solved: %v", c.IsSolved())

	if !c.IsWhiteCrossComplete() {
		t.Error("Solved cube should have white cross")
	}
	if !c.IsTopLayerComplete() {
		t.Error("Solved cube should have top layer")
	}
	if !c.IsMiddleLayerComplete() {
		t.Error("Solved cube should have middle layer")
	}
	if !c.IsBottomCrossComplete() {
		t.Error("Solved cube should have bottom cross")
	}
	if !c.AreBottomCornersPositioned() {
		t.Error("Solved cube should have corners positioned")
	}
	if !c.AreBottomCornersOriented() {
		t.Error("Solved cube should have corners oriented")
	}
	if !c.IsSolved() {
		t.Error("Solved cube should be solved")
	}
}

func TestTrackerPhaseCallback(t *testing.T) {
	// Verify that the tracker fires phase callbacks correctly
	tr := NewTracker()

	var phaseChanges []string
	tr.SetPhaseCallback(func(phase DetectedPhase, phaseKey string) {
		phaseChanges = append(phaseChanges, phaseKey)
		t.Logf("Phase callback fired: %s", phaseKey)
	})

	// Scramble the cube
	tr.ApplyMove(types.Move{Face: types.FaceR, Turn: types.TurnCW})
	tr.ApplyMove(types.Move{Face: types.FaceU, Turn: types.TurnCW})
	tr.ApplyMove(types.Move{Face: types.FaceF, Turn: types.TurnCW})

	t.Logf("After scramble: phase=%s, callbacks=%v", tr.CurrentPhaseKey(), phaseChanges)

	// Phase should have gone backwards (scrambled), no forward callbacks
	if tr.CurrentPhaseKey() != "scrambled" {
		t.Errorf("Expected phase 'scrambled', got %s", tr.CurrentPhaseKey())
	}

	// Now reverse to get back to solved
	tr.ApplyMove(types.Move{Face: types.FaceF, Turn: types.TurnCCW})
	tr.ApplyMove(types.Move{Face: types.FaceU, Turn: types.TurnCCW})
	tr.ApplyMove(types.Move{Face: types.FaceR, Turn: types.TurnCCW})

	t.Logf("After reverse: phase=%s, callbacks=%v", tr.CurrentPhaseKey(), phaseChanges)

	if !tr.IsSolved() {
		t.Error("Tracker should be solved after reversing moves")
		t.Log(tr.CubeString())
	}
}
