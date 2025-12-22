package gocube

import "fmt"

// Color represents a face color on the cube.
type Color byte

const (
	White  Color = 0 // Up face when solved
	Yellow Color = 1 // Down face when solved
	Green  Color = 2 // Front face when solved
	Blue   Color = 3 // Back face when solved
	Red    Color = 4 // Right face when solved
	Orange Color = 5 // Left face when solved
)

func (c Color) String() string {
	switch c {
	case White:
		return "W"
	case Yellow:
		return "Y"
	case Green:
		return "G"
	case Blue:
		return "B"
	case Red:
		return "R"
	case Orange:
		return "O"
	default:
		return "?"
	}
}

// CubeFace represents a cube face for the internal cube model.
type CubeFace int

const (
	CubeFaceU CubeFace = 0 // Up (White)
	CubeFaceD CubeFace = 1 // Down (Yellow)
	CubeFaceF CubeFace = 2 // Front (Green)
	CubeFaceB CubeFace = 3 // Back (Blue)
	CubeFaceR CubeFace = 4 // Right (Red)
	CubeFaceL CubeFace = 5 // Left (Orange)
)

// Cube represents a 3x3 Rubik's cube state.
// Can be used standalone without a BLE connection for simulation.
//
// Each face has 9 facelets indexed as:
//
//	0 1 2
//	3 4 5
//	6 7 8
//
// The center (index 4) defines the face color and never moves.
type Cube struct {
	// Facelets[face][position] = color
	Facelets [6][9]Color
}

// NewCube creates a solved cube with standard orientation:
// White on top, Green in front.
func NewCube() *Cube {
	c := &Cube{}
	c.Reset()
	return c
}

// Reset resets the cube to the solved state.
func (c *Cube) Reset() {
	for face := CubeFace(0); face < 6; face++ {
		color := faceToSolvedColor(face)
		for i := 0; i < 9; i++ {
			c.Facelets[face][i] = color
		}
	}
}

// faceToSolvedColor returns the color of a face when solved.
func faceToSolvedColor(f CubeFace) Color {
	switch f {
	case CubeFaceU:
		return White
	case CubeFaceD:
		return Yellow
	case CubeFaceF:
		return Green
	case CubeFaceB:
		return Blue
	case CubeFaceR:
		return Red
	case CubeFaceL:
		return Orange
	default:
		return White
	}
}

// Clone creates a deep copy of the cube.
func (c *Cube) Clone() *Cube {
	clone := &Cube{}
	for f := 0; f < 6; f++ {
		for i := 0; i < 9; i++ {
			clone.Facelets[f][i] = c.Facelets[f][i]
		}
	}
	return clone
}

// Apply applies one or more moves to the cube.
//
// Example:
//
//	cube.Apply(gocube.R, gocube.U, gocube.RPrime, gocube.UPrime)
func (c *Cube) Apply(moves ...Move) {
	for _, m := range moves {
		c.applyMove(m)
	}
}

// ApplyNotation parses and applies moves from notation string.
//
// Example:
//
//	cube.ApplyNotation("R U R' U'")
func (c *Cube) ApplyNotation(notation string) error {
	moves, err := ParseMoves(notation)
	if err != nil {
		return err
	}
	c.Apply(moves...)
	return nil
}

// applyMove applies a single Move to the cube.
func (c *Cube) applyMove(m Move) {
	face := moveFaceToCubeFace(m.Face)
	turn := int(m.Turn)
	c.moveFace(face, turn)
}

// IsSolved returns true if the cube is in the solved state.
func (c *Cube) IsSolved() bool {
	for face := CubeFace(0); face < 6; face++ {
		expectedColor := faceToSolvedColor(face)
		for i := 0; i < 9; i++ {
			if c.Facelets[face][i] != expectedColor {
				return false
			}
		}
	}
	return true
}

// Phase returns the current solving phase.
func (c *Cube) Phase() Phase {
	return c.detectPhase()
}

// GetProgress returns detailed progress through all phases.
func (c *Cube) GetProgress() Progress {
	return Progress{
		WhiteCross:     c.isWhiteCrossComplete(),
		FirstLayer:     c.isTopLayerComplete(),
		SecondLayer:    c.isMiddleLayerComplete(),
		YellowCross:    c.isBottomCrossComplete(),
		YellowCorners:  c.areBottomCornersPositioned(),
		YellowOriented: c.areBottomCornersOriented(),
		Solved:         c.IsSolved(),
	}
}

// String returns an ASCII visualization of the cube.
func (c *Cube) String() string {
	result := ""

	// U face (indented)
	for row := 0; row < 3; row++ {
		result += "      "
		for col := 0; col < 3; col++ {
			result += c.Facelets[CubeFaceU][row*3+col].String() + " "
		}
		result += "\n"
	}

	// L, F, R, B faces (side by side)
	for row := 0; row < 3; row++ {
		for _, face := range []CubeFace{CubeFaceL, CubeFaceF, CubeFaceR, CubeFaceB} {
			for col := 0; col < 3; col++ {
				result += c.Facelets[face][row*3+col].String() + " "
			}
		}
		result += "\n"
	}

	// D face (indented)
	for row := 0; row < 3; row++ {
		result += "      "
		for col := 0; col < 3; col++ {
			result += c.Facelets[CubeFaceD][row*3+col].String() + " "
		}
		result += "\n"
	}

	return result
}

// Debug returns a simple debug string.
func (c *Cube) Debug() string {
	return fmt.Sprintf("Solved: %v, Phase: %s", c.IsSolved(), c.Phase())
}

// moveFace applies a move to the cube using CubeFace.
func (c *Cube) moveFace(face CubeFace, turn int) {
	switch turn {
	case 1: // CW
		c.moveCW(face)
	case -1: // CCW
		c.moveCCW(face)
	case 2: // 180
		c.moveCW(face)
		c.moveCW(face)
	}
}

// rotateFaceCW rotates a face 90 degrees clockwise.
func (c *Cube) rotateFaceCW(face CubeFace) {
	f := &c.Facelets[face]
	temp := f[0]
	f[0] = f[6]
	f[6] = f[8]
	f[8] = f[2]
	f[2] = temp

	temp = f[1]
	f[1] = f[3]
	f[3] = f[7]
	f[7] = f[5]
	f[5] = temp
}

// rotateFaceCCW rotates a face 90 degrees counter-clockwise.
func (c *Cube) rotateFaceCCW(face CubeFace) {
	f := &c.Facelets[face]
	temp := f[0]
	f[0] = f[2]
	f[2] = f[8]
	f[8] = f[6]
	f[6] = temp

	temp = f[1]
	f[1] = f[5]
	f[5] = f[7]
	f[7] = f[3]
	f[3] = temp
}

// moveCW applies a clockwise move.
func (c *Cube) moveCW(face CubeFace) {
	c.rotateFaceCW(face)
	c.cycleEdgesCW(face)
}

// moveCCW applies a counter-clockwise move.
func (c *Cube) moveCCW(face CubeFace) {
	c.rotateFaceCCW(face)
	c.cycleEdgesCCW(face)
}

// cycleEdgesCW cycles the edge facelets around a face (clockwise).
func (c *Cube) cycleEdgesCW(face CubeFace) {
	switch face {
	case CubeFaceU:
		c.cycle4(
			[3]int{int(CubeFaceF), 0, 1}, [3]int{int(CubeFaceF), 1, 1}, [3]int{int(CubeFaceF), 2, 1},
			[3]int{int(CubeFaceL), 0, 1}, [3]int{int(CubeFaceL), 1, 1}, [3]int{int(CubeFaceL), 2, 1},
			[3]int{int(CubeFaceB), 0, 1}, [3]int{int(CubeFaceB), 1, 1}, [3]int{int(CubeFaceB), 2, 1},
			[3]int{int(CubeFaceR), 0, 1}, [3]int{int(CubeFaceR), 1, 1}, [3]int{int(CubeFaceR), 2, 1},
		)
	case CubeFaceD:
		c.cycle4(
			[3]int{int(CubeFaceF), 6, 1}, [3]int{int(CubeFaceF), 7, 1}, [3]int{int(CubeFaceF), 8, 1},
			[3]int{int(CubeFaceR), 6, 1}, [3]int{int(CubeFaceR), 7, 1}, [3]int{int(CubeFaceR), 8, 1},
			[3]int{int(CubeFaceB), 6, 1}, [3]int{int(CubeFaceB), 7, 1}, [3]int{int(CubeFaceB), 8, 1},
			[3]int{int(CubeFaceL), 6, 1}, [3]int{int(CubeFaceL), 7, 1}, [3]int{int(CubeFaceL), 8, 1},
		)
	case CubeFaceF:
		c.cycle4Edge(
			int(CubeFaceU), []int{6, 7, 8},
			int(CubeFaceR), []int{0, 3, 6},
			int(CubeFaceD), []int{2, 1, 0},
			int(CubeFaceL), []int{8, 5, 2},
		)
	case CubeFaceB:
		c.cycle4Edge(
			int(CubeFaceU), []int{2, 1, 0},
			int(CubeFaceL), []int{0, 3, 6},
			int(CubeFaceD), []int{6, 7, 8},
			int(CubeFaceR), []int{8, 5, 2},
		)
	case CubeFaceR:
		c.cycle4Edge(
			int(CubeFaceU), []int{2, 5, 8},
			int(CubeFaceB), []int{6, 3, 0},
			int(CubeFaceD), []int{2, 5, 8},
			int(CubeFaceF), []int{2, 5, 8},
		)
	case CubeFaceL:
		c.cycle4Edge(
			int(CubeFaceU), []int{0, 3, 6},
			int(CubeFaceF), []int{0, 3, 6},
			int(CubeFaceD), []int{0, 3, 6},
			int(CubeFaceB), []int{8, 5, 2},
		)
	}
}

// cycleEdgesCCW cycles the edge facelets around a face (counter-clockwise).
func (c *Cube) cycleEdgesCCW(face CubeFace) {
	c.cycleEdgesCW(face)
	c.cycleEdgesCW(face)
	c.cycleEdgesCW(face)
}

// cycle4 cycles 4 groups of 3 facelets.
func (c *Cube) cycle4(a1, a2, a3, b1, b2, b3, c1, c2, c3, d1, d2, d3 [3]int) {
	t1 := c.Facelets[a1[0]][a1[1]]
	t2 := c.Facelets[a2[0]][a2[1]]
	t3 := c.Facelets[a3[0]][a3[1]]

	c.Facelets[a1[0]][a1[1]] = c.Facelets[d1[0]][d1[1]]
	c.Facelets[a2[0]][a2[1]] = c.Facelets[d2[0]][d2[1]]
	c.Facelets[a3[0]][a3[1]] = c.Facelets[d3[0]][d3[1]]

	c.Facelets[d1[0]][d1[1]] = c.Facelets[c1[0]][c1[1]]
	c.Facelets[d2[0]][d2[1]] = c.Facelets[c2[0]][c2[1]]
	c.Facelets[d3[0]][d3[1]] = c.Facelets[c3[0]][c3[1]]

	c.Facelets[c1[0]][c1[1]] = c.Facelets[b1[0]][b1[1]]
	c.Facelets[c2[0]][c2[1]] = c.Facelets[b2[0]][b2[1]]
	c.Facelets[c3[0]][c3[1]] = c.Facelets[b3[0]][b3[1]]

	c.Facelets[b1[0]][b1[1]] = t1
	c.Facelets[b2[0]][b2[1]] = t2
	c.Facelets[b3[0]][b3[1]] = t3
}

// cycle4Edge cycles 4 edges with arbitrary indices.
func (c *Cube) cycle4Edge(f1 int, i1 []int, f2 int, i2 []int, f3 int, i3 []int, f4 int, i4 []int) {
	t := [3]Color{
		c.Facelets[f1][i1[0]],
		c.Facelets[f1][i1[1]],
		c.Facelets[f1][i1[2]],
	}

	c.Facelets[f1][i1[0]] = c.Facelets[f4][i4[0]]
	c.Facelets[f1][i1[1]] = c.Facelets[f4][i4[1]]
	c.Facelets[f1][i1[2]] = c.Facelets[f4][i4[2]]

	c.Facelets[f4][i4[0]] = c.Facelets[f3][i3[0]]
	c.Facelets[f4][i4[1]] = c.Facelets[f3][i3[1]]
	c.Facelets[f4][i4[2]] = c.Facelets[f3][i3[2]]

	c.Facelets[f3][i3[0]] = c.Facelets[f2][i2[0]]
	c.Facelets[f3][i3[1]] = c.Facelets[f2][i2[1]]
	c.Facelets[f3][i3[2]] = c.Facelets[f2][i2[2]]

	c.Facelets[f2][i2[0]] = t[0]
	c.Facelets[f2][i2[1]] = t[1]
	c.Facelets[f2][i2[2]] = t[2]
}

// moveFaceToCubeFace converts Face to CubeFace.
func moveFaceToCubeFace(f Face) CubeFace {
	switch f {
	case FaceU:
		return CubeFaceU
	case FaceD:
		return CubeFaceD
	case FaceF:
		return CubeFaceF
	case FaceB:
		return CubeFaceB
	case FaceR:
		return CubeFaceR
	case FaceL:
		return CubeFaceL
	default:
		return CubeFaceU
	}
}

// Phase detection methods

func (c *Cube) detectPhase() Phase {
	if c.IsSolved() {
		return PhaseSolved
	}
	if c.areBottomCornersOriented() {
		return PhaseYellowOriented
	}
	if c.areBottomCornersPositioned() {
		return PhaseYellowCorners
	}
	if c.isBottomCrossComplete() {
		return PhaseYellowCross
	}
	if c.isMiddleLayerComplete() {
		return PhaseSecondLayer
	}
	if c.isTopLayerComplete() {
		return PhaseFirstLayer
	}
	if c.isWhiteCrossComplete() {
		return PhaseWhiteCross
	}
	return PhaseScrambled
}

func (c *Cube) isWhiteCrossComplete() bool {
	uEdges := []int{1, 3, 5, 7}
	for _, pos := range uEdges {
		if c.Facelets[CubeFaceU][pos] != White {
			return false
		}
	}

	if c.Facelets[CubeFaceB][1] != c.Facelets[CubeFaceB][4] {
		return false
	}
	if c.Facelets[CubeFaceL][1] != c.Facelets[CubeFaceL][4] {
		return false
	}
	if c.Facelets[CubeFaceR][1] != c.Facelets[CubeFaceR][4] {
		return false
	}
	if c.Facelets[CubeFaceF][1] != c.Facelets[CubeFaceF][4] {
		return false
	}

	return true
}

func (c *Cube) isTopLayerComplete() bool {
	if !c.isWhiteCrossComplete() {
		return false
	}

	for i := 0; i < 9; i++ {
		if c.Facelets[CubeFaceU][i] != White {
			return false
		}
	}

	if c.Facelets[CubeFaceF][0] != c.Facelets[CubeFaceF][4] || c.Facelets[CubeFaceF][2] != c.Facelets[CubeFaceF][4] {
		return false
	}
	if c.Facelets[CubeFaceR][0] != c.Facelets[CubeFaceR][4] || c.Facelets[CubeFaceR][2] != c.Facelets[CubeFaceR][4] {
		return false
	}
	if c.Facelets[CubeFaceB][0] != c.Facelets[CubeFaceB][4] || c.Facelets[CubeFaceB][2] != c.Facelets[CubeFaceB][4] {
		return false
	}
	if c.Facelets[CubeFaceL][0] != c.Facelets[CubeFaceL][4] || c.Facelets[CubeFaceL][2] != c.Facelets[CubeFaceL][4] {
		return false
	}

	return true
}

func (c *Cube) isMiddleLayerComplete() bool {
	if !c.isTopLayerComplete() {
		return false
	}

	for _, face := range []CubeFace{CubeFaceF, CubeFaceR, CubeFaceB, CubeFaceL} {
		center := c.Facelets[face][4]
		if c.Facelets[face][3] != center || c.Facelets[face][5] != center {
			return false
		}
	}

	return true
}

func (c *Cube) isBottomCrossComplete() bool {
	if !c.isMiddleLayerComplete() {
		return false
	}

	dEdges := []int{1, 3, 5, 7}
	for _, pos := range dEdges {
		if c.Facelets[CubeFaceD][pos] != Yellow {
			return false
		}
	}

	return true
}

func (c *Cube) areBottomCornersPositioned() bool {
	if !c.isBottomCrossComplete() {
		return false
	}

	corners := []struct {
		positions [][2]int
		colors    []Color
	}{
		{[][2]int{{int(CubeFaceF), 8}, {int(CubeFaceR), 6}, {int(CubeFaceD), 2}}, []Color{Green, Red, Yellow}},
		{[][2]int{{int(CubeFaceR), 8}, {int(CubeFaceB), 6}, {int(CubeFaceD), 8}}, []Color{Red, Blue, Yellow}},
		{[][2]int{{int(CubeFaceB), 8}, {int(CubeFaceL), 6}, {int(CubeFaceD), 6}}, []Color{Blue, Orange, Yellow}},
		{[][2]int{{int(CubeFaceL), 8}, {int(CubeFaceF), 6}, {int(CubeFaceD), 0}}, []Color{Orange, Green, Yellow}},
	}

	for _, corner := range corners {
		actualColors := make([]Color, 3)
		for i, pos := range corner.positions {
			actualColors[i] = c.Facelets[pos[0]][pos[1]]
		}

		if !sameColors(actualColors, corner.colors) {
			return false
		}
	}

	return true
}

func (c *Cube) areBottomCornersOriented() bool {
	if !c.areBottomCornersPositioned() {
		return false
	}

	for i := 0; i < 9; i++ {
		if c.Facelets[CubeFaceD][i] != Yellow {
			return false
		}
	}

	for _, face := range []CubeFace{CubeFaceF, CubeFaceR, CubeFaceB, CubeFaceL} {
		center := c.Facelets[face][4]
		if c.Facelets[face][6] != center || c.Facelets[face][8] != center {
			return false
		}
	}

	return true
}

func sameColors(a, b []Color) bool {
	if len(a) != len(b) {
		return false
	}

	count := make(map[Color]int)
	for _, col := range a {
		count[col]++
	}
	for _, col := range b {
		count[col]--
	}
	for _, v := range count {
		if v != 0 {
			return false
		}
	}
	return true
}
