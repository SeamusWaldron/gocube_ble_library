package gocube

import (
	"testing"
)

func TestNewCubeIsSolved(t *testing.T) {
	c := NewCube()
	if !c.IsSolved() {
		t.Error("New cube should be solved")
	}
}

func TestSingleMoveBreaksSolved(t *testing.T) {
	c := NewCube()
	c.Apply(R) // R
	if c.IsSolved() {
		t.Error("Cube should not be solved after R move")
	}
}

func TestRx4_ReturnsToSolved(t *testing.T) {
	c := NewCube()
	// R R R R = identity
	c.Apply(R, R, R, R)
	if !c.IsSolved() {
		t.Error("R R R R should return to solved")
		t.Log(c.String())
	}
}

func TestR2R2_ReturnsToSolved(t *testing.T) {
	c := NewCube()
	c.Apply(R2, R2)
	if !c.IsSolved() {
		t.Error("R2 R2 should return to solved")
		t.Log(c.String())
	}
}

func TestAllFacesX4_ReturnsToSolved(t *testing.T) {
	moves := []Move{U, D, F, B, R, L}
	for _, m := range moves {
		c := NewCube()
		c.Apply(m, m, m, m)
		if !c.IsSolved() {
			t.Errorf("%s x 4 should return to solved", m.Notation())
			t.Log(c.String())
		}
	}
}

func TestSexyMove_6Times_ReturnsToSolved(t *testing.T) {
	// (R U R' U') x 6 = identity
	c := NewCube()
	for i := 0; i < 6; i++ {
		c.Apply(R, U, RPrime, UPrime)
	}
	if !c.IsSolved() {
		t.Error("Sexy move x 6 should return to solved")
		t.Log(c.String())
	}
}

func TestApplyNotation(t *testing.T) {
	c := NewCube()
	err := c.ApplyNotation("R U R' U'")
	if err != nil {
		t.Errorf("ApplyNotation failed: %v", err)
	}
	if c.IsSolved() {
		t.Error("Cube should not be solved after R U R' U'")
	}

	// Apply 5 more times to get back to solved
	for i := 0; i < 5; i++ {
		c.ApplyNotation("R U R' U'")
	}
	if !c.IsSolved() {
		t.Error("Sexy move x 6 should return to solved")
		t.Log(c.String())
	}
}

func TestApply_RRPrime_ReturnsToSolved(t *testing.T) {
	c := NewCube()
	c.Apply(R)
	if c.IsSolved() {
		t.Error("Cube should not be solved after R")
	}
	c.Apply(RPrime)
	if !c.IsSolved() {
		t.Error("Cube should be solved after R R'")
		t.Log(c.String())
	}
}

func TestPhaseDetection(t *testing.T) {
	c := NewCube()
	phase := c.Phase()
	if phase != PhaseSolved {
		t.Errorf("Solved cube should detect as PhaseSolved, got %v", phase)
	}

	c.Apply(R)
	phase = c.Phase()
	if phase == PhaseSolved {
		t.Error("Scrambled cube should not detect as solved")
	}
}

func TestReset(t *testing.T) {
	c := NewCube()
	c.Apply(R, U, F)
	if c.IsSolved() {
		t.Error("Cube should not be solved after moves")
	}

	c.Reset()
	if !c.IsSolved() {
		t.Error("Cube should be solved after reset")
	}
}

func TestClone(t *testing.T) {
	c := NewCube()
	c.Apply(R, U)

	clone := c.Clone()

	// Clone should have same state
	if clone.IsSolved() != c.IsSolved() {
		t.Error("Clone should have same solved state")
	}

	// Modifying clone shouldn't affect original
	clone.Reset()
	if clone.IsSolved() == c.IsSolved() {
		t.Error("Modifying clone shouldn't affect original")
	}
}

func TestScrambleAndReverse(t *testing.T) {
	c := NewCube()

	// Simple scramble
	scramble := []Move{R, U, RPrime, UPrime, F, D, L2}

	// Apply scramble
	c.Apply(scramble...)

	if c.IsSolved() {
		t.Error("Cube should be scrambled after moves")
	}

	phase := c.Phase()
	t.Logf("After scramble: phase=%s", phase.String())

	// Reverse the scramble
	for i := len(scramble) - 1; i >= 0; i-- {
		c.Apply(scramble[i].Inverse())
	}

	if !c.IsSolved() {
		t.Error("Cube should be solved after reversing scramble")
		t.Log(c.String())
	}
}

func TestGetProgress(t *testing.T) {
	c := NewCube()
	progress := c.GetProgress()

	// All phases should be complete on solved cube
	if !progress.WhiteCross {
		t.Error("Solved cube should have white cross")
	}
	if !progress.FirstLayer {
		t.Error("Solved cube should have first layer")
	}
	if !progress.SecondLayer {
		t.Error("Solved cube should have second layer")
	}
	if !progress.YellowCross {
		t.Error("Solved cube should have yellow cross")
	}
	if !progress.YellowCorners {
		t.Error("Solved cube should have yellow corners positioned")
	}
	if !progress.YellowOriented {
		t.Error("Solved cube should have yellow corners oriented")
	}
	if !progress.Solved {
		t.Error("Solved cube should be solved")
	}
}

func TestPhaseProgression(t *testing.T) {
	c := NewCube()

	// Verify solved cube is at PhaseSolved
	if c.Phase() != PhaseSolved {
		t.Errorf("Solved cube phase should be PhaseSolved, got %v", c.Phase())
	}

	// Apply R to break the solve
	c.Apply(R)
	phase := c.Phase()
	t.Logf("After R: phase=%s", phase.String())

	// Should not be solved anymore
	if phase == PhaseSolved {
		t.Error("Cube should not be solved after R")
	}

	// Apply R' to restore
	c.Apply(RPrime)
	if c.Phase() != PhaseSolved {
		t.Errorf("Cube should be solved after R R', got phase=%s", c.Phase().String())
	}
}

func TestMoveNotation(t *testing.T) {
	tests := []struct {
		move     Move
		expected string
	}{
		{R, "R"},
		{RPrime, "R'"},
		{R2, "R2"},
		{U, "U"},
		{UPrime, "U'"},
		{U2, "U2"},
		{F, "F"},
		{FPrime, "F'"},
		{F2, "F2"},
	}

	for _, tc := range tests {
		if tc.move.Notation() != tc.expected {
			t.Errorf("Move.Notation() = %s, expected %s", tc.move.Notation(), tc.expected)
		}
	}
}

func TestMoveInverse(t *testing.T) {
	tests := []struct {
		move    Move
		inverse Move
	}{
		{R, RPrime},
		{RPrime, R},
		{R2, R2},
		{U, UPrime},
		{UPrime, U},
	}

	for _, tc := range tests {
		inv := tc.move.Inverse()
		if inv.Face != tc.inverse.Face || inv.Turn != tc.inverse.Turn {
			t.Errorf("%s.Inverse() = %s, expected %s", tc.move.Notation(), inv.Notation(), tc.inverse.Notation())
		}
	}
}

func TestParseMoves(t *testing.T) {
	moves, err := ParseMoves("R U R' U'")
	if err != nil {
		t.Errorf("ParseMoves failed: %v", err)
	}

	expected := []Move{R, U, RPrime, UPrime}
	if len(moves) != len(expected) {
		t.Errorf("ParseMoves returned %d moves, expected %d", len(moves), len(expected))
	}

	for i, m := range moves {
		if m.Face != expected[i].Face || m.Turn != expected[i].Turn {
			t.Errorf("Move %d: got %s, expected %s", i, m.Notation(), expected[i].Notation())
		}
	}
}
