package gocube

// Phase detection for layer-by-layer solving method.
// Standard orientation: White on top (U), Green in front (F).

// IsWhiteCrossComplete checks if the white cross is complete.
// The white cross has:
// - White center on U face (always true for solved start)
// - 4 white edge pieces on U face (positions 1, 3, 5, 7)
// - Each edge's other color matches the adjacent center
func (c *Cube) IsWhiteCrossComplete() bool {
	// Check U face edge positions are white
	uEdges := []int{1, 3, 5, 7}
	for _, pos := range uEdges {
		if c.Facelets[CubeFaceU][pos] != White {
			return false
		}
	}

	// Check that adjacent edges match their center colors
	// U[1] is adjacent to B[1], U[3] is adjacent to L[1]
	// U[5] is adjacent to R[1], U[7] is adjacent to F[1]
	if c.Facelets[CubeFaceB][1] != c.Facelets[CubeFaceB][4] { // B edge matches B center
		return false
	}
	if c.Facelets[CubeFaceL][1] != c.Facelets[CubeFaceL][4] { // L edge matches L center
		return false
	}
	if c.Facelets[CubeFaceR][1] != c.Facelets[CubeFaceR][4] { // R edge matches R center
		return false
	}
	if c.Facelets[CubeFaceF][1] != c.Facelets[CubeFaceF][4] { // F edge matches F center
		return false
	}

	return true
}

// IsTopLayerComplete checks if the entire top layer is complete.
// This means white cross + white corners all in place.
func (c *Cube) IsTopLayerComplete() bool {
	// First, cross must be complete
	if !c.IsWhiteCrossComplete() {
		return false
	}

	// Check all U face facelets are white
	for i := 0; i < 9; i++ {
		if c.Facelets[CubeFaceU][i] != White {
			return false
		}
	}

	// Check corner facelets on adjacent faces match their centers
	// F face: top-left (0) and top-right (2) should match F center
	if c.Facelets[CubeFaceF][0] != c.Facelets[CubeFaceF][4] || c.Facelets[CubeFaceF][2] != c.Facelets[CubeFaceF][4] {
		return false
	}
	// R face: top-left (0) and top-right (2)
	if c.Facelets[CubeFaceR][0] != c.Facelets[CubeFaceR][4] || c.Facelets[CubeFaceR][2] != c.Facelets[CubeFaceR][4] {
		return false
	}
	// B face: top-left (0) and top-right (2)
	if c.Facelets[CubeFaceB][0] != c.Facelets[CubeFaceB][4] || c.Facelets[CubeFaceB][2] != c.Facelets[CubeFaceB][4] {
		return false
	}
	// L face: top-left (0) and top-right (2)
	if c.Facelets[CubeFaceL][0] != c.Facelets[CubeFaceL][4] || c.Facelets[CubeFaceL][2] != c.Facelets[CubeFaceL][4] {
		return false
	}

	return true
}

// IsMiddleLayerComplete checks if the middle layer is complete.
// Middle layer edges are at positions 3 and 5 on F, R, B, L faces.
func (c *Cube) IsMiddleLayerComplete() bool {
	// Top layer must be complete first
	if !c.IsTopLayerComplete() {
		return false
	}

	// Check middle edges on each side face
	for _, face := range []CubeFace{CubeFaceF, CubeFaceR, CubeFaceB, CubeFaceL} {
		center := c.Facelets[face][4]
		if c.Facelets[face][3] != center || c.Facelets[face][5] != center {
			return false
		}
	}

	return true
}

// IsBottomCrossComplete checks if the yellow cross is formed on the bottom.
// Note: This only checks that the 4 edges on D face are yellow,
// not that they're in the correct positions.
func (c *Cube) IsBottomCrossComplete() bool {
	if !c.IsMiddleLayerComplete() {
		return false
	}

	// Check D face edge positions are yellow
	dEdges := []int{1, 3, 5, 7}
	for _, pos := range dEdges {
		if c.Facelets[CubeFaceD][pos] != Yellow {
			return false
		}
	}

	return true
}

// AreBottomCornersPositioned checks if bottom corners are in correct positions.
// They may not be oriented correctly yet.
func (c *Cube) AreBottomCornersPositioned() bool {
	if !c.IsBottomCrossComplete() {
		return false
	}

	// Check each corner has the right set of colors (ignoring orientation)
	// Corner FRD should have Green, Red, Yellow
	// Corner RBD should have Red, Blue, Yellow
	// Corner BLD should have Blue, Orange, Yellow
	// Corner LFD should have Orange, Green, Yellow

	corners := []struct {
		positions [][2]int // [face][index] pairs for a corner
		colors    []Color  // expected colors (in any order)
	}{
		{[][2]int{{int(CubeFaceF), 8}, {int(CubeFaceR), 6}, {int(CubeFaceD), 2}}, []Color{Green, Red, Yellow}},
		{[][2]int{{int(CubeFaceR), 8}, {int(CubeFaceB), 6}, {int(CubeFaceD), 8}}, []Color{Red, Blue, Yellow}},
		{[][2]int{{int(CubeFaceB), 8}, {int(CubeFaceL), 6}, {int(CubeFaceD), 6}}, []Color{Blue, Orange, Yellow}},
		{[][2]int{{int(CubeFaceL), 8}, {int(CubeFaceF), 6}, {int(CubeFaceD), 0}}, []Color{Orange, Green, Yellow}},
	}

	for _, corner := range corners {
		// Get the actual colors at this corner
		actualColors := make([]Color, 3)
		for i, pos := range corner.positions {
			actualColors[i] = c.Facelets[pos[0]][pos[1]]
		}

		// Check if actual colors match expected (in any order)
		if !sameColors(actualColors, corner.colors) {
			return false
		}
	}

	return true
}

// AreBottomCornersOriented checks if bottom corners are correctly oriented.
// This is the final step - cube should be solved after this.
func (c *Cube) AreBottomCornersOriented() bool {
	if !c.AreBottomCornersPositioned() {
		return false
	}

	// All D face facelets should be yellow
	for i := 0; i < 9; i++ {
		if c.Facelets[CubeFaceD][i] != Yellow {
			return false
		}
	}

	// And corner facelets on side faces should match their centers
	// Check bottom corners of F, R, B, L
	for _, face := range []CubeFace{CubeFaceF, CubeFaceR, CubeFaceB, CubeFaceL} {
		center := c.Facelets[face][4]
		if c.Facelets[face][6] != center || c.Facelets[face][8] != center {
			return false
		}
	}

	return true
}

// sameColors checks if two color slices contain the same colors (in any order).
func sameColors(a, b []Color) bool {
	if len(a) != len(b) {
		return false
	}

	// Count occurrences
	count := make(map[Color]int)
	for _, c := range a {
		count[c]++
	}
	for _, c := range b {
		count[c]--
	}
	for _, v := range count {
		if v != 0 {
			return false
		}
	}
	return true
}

// DetectedPhase represents which phase the cube is currently in.
type DetectedPhase int

const (
	PhaseScrambled DetectedPhase = iota
	PhaseWhiteCross
	PhaseTopLayer
	PhaseMiddleLayer
	PhaseBottomCross
	PhaseCornersPositioned
	PhaseCornersOriented
	PhaseSolved
)

func (p DetectedPhase) String() string {
	switch p {
	case PhaseScrambled:
		return "scrambled"
	case PhaseWhiteCross:
		return "white_cross"
	case PhaseTopLayer:
		return "top_corners"
	case PhaseMiddleLayer:
		return "middle_layer"
	case PhaseBottomCross:
		return "bottom_cross"
	case PhaseCornersPositioned:
		return "position_corners"
	case PhaseCornersOriented:
		return "rotate_corners"
	case PhaseSolved:
		return "complete"
	default:
		return "unknown"
	}
}

// DetectPhase returns the current solve phase based on cube state.
func (c *Cube) DetectPhase() DetectedPhase {
	if c.IsSolved() {
		return PhaseSolved
	}
	if c.AreBottomCornersOriented() {
		return PhaseCornersOriented // Corners oriented but edges might need AUF
	}
	if c.AreBottomCornersPositioned() {
		return PhaseCornersPositioned
	}
	if c.IsBottomCrossComplete() {
		return PhaseBottomCross
	}
	if c.IsMiddleLayerComplete() {
		return PhaseMiddleLayer
	}
	if c.IsTopLayerComplete() {
		return PhaseTopLayer
	}
	if c.IsWhiteCrossComplete() {
		return PhaseWhiteCross
	}
	return PhaseScrambled
}

// PhaseProgress returns which phases are complete.
type PhaseProgress struct {
	WhiteCross        bool
	TopLayer          bool
	MiddleLayer       bool
	BottomCross       bool
	CornersPositioned bool
	CornersOriented   bool
	Solved            bool
}

// GetProgress returns the current progress through all phases.
func (c *Cube) GetProgress() PhaseProgress {
	return PhaseProgress{
		WhiteCross:        c.IsWhiteCrossComplete(),
		TopLayer:          c.IsTopLayerComplete(),
		MiddleLayer:       c.IsMiddleLayerComplete(),
		BottomCross:       c.IsBottomCrossComplete(),
		CornersPositioned: c.AreBottomCornersPositioned(),
		CornersOriented:   c.AreBottomCornersOriented(),
		Solved:            c.IsSolved(),
	}
}
