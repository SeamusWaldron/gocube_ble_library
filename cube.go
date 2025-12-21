package gocube

import "fmt"

// Color represents a face color.
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

// CubeFace represents a cube face for the cube model.
// This is distinct from Face which is used for move notation.
type CubeFace int

const (
	CubeFaceU CubeFace = 0 // Up (White)
	CubeFaceD CubeFace = 1 // Down (Yellow)
	CubeFaceF CubeFace = 2 // Front (Green)
	CubeFaceB CubeFace = 3 // Back (Blue)
	CubeFaceR CubeFace = 4 // Right (Red)
	CubeFaceL CubeFace = 5 // Left (Orange)
)

func (f CubeFace) String() string {
	switch f {
	case CubeFaceU:
		return "U"
	case CubeFaceD:
		return "D"
	case CubeFaceF:
		return "F"
	case CubeFaceB:
		return "B"
	case CubeFaceR:
		return "R"
	case CubeFaceL:
		return "L"
	default:
		return "?"
	}
}

// Cube represents a 3x3 Rubik's cube.
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
	// Initialize each face with its solved color
	for face := CubeFace(0); face < 6; face++ {
		color := faceToSolvedColor(face)
		for i := 0; i < 9; i++ {
			c.Facelets[face][i] = color
		}
	}
	return c
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

// rotateFaceCW rotates a face 90 degrees clockwise.
func (c *Cube) rotateFaceCW(face CubeFace) {
	f := &c.Facelets[face]
	// Corner rotation: 0->2->8->6->0
	// Edge rotation: 1->5->7->3->1
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
	// Corner rotation: 0->6->8->2->0
	// Edge rotation: 1->3->7->5->1
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

// MoveFace applies a move to the cube using CubeFace.
// turn: 1 = CW, -1 = CCW, 2 = 180 degrees
func (c *Cube) MoveFace(face CubeFace, turn int) {
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
	// Each face affects 4 adjacent faces' edges
	// The indices depend on which face is being rotated
	switch face {
	case CubeFaceU:
		// U affects F, L, B, R top rows
		c.cycle4(
			[3]int{int(CubeFaceF), 0, 1}, [3]int{int(CubeFaceF), 1, 1}, [3]int{int(CubeFaceF), 2, 1}, // F top: 0,1,2
			[3]int{int(CubeFaceL), 0, 1}, [3]int{int(CubeFaceL), 1, 1}, [3]int{int(CubeFaceL), 2, 1}, // L top: 0,1,2
			[3]int{int(CubeFaceB), 0, 1}, [3]int{int(CubeFaceB), 1, 1}, [3]int{int(CubeFaceB), 2, 1}, // B top: 0,1,2
			[3]int{int(CubeFaceR), 0, 1}, [3]int{int(CubeFaceR), 1, 1}, [3]int{int(CubeFaceR), 2, 1}, // R top: 0,1,2
		)
	case CubeFaceD:
		// D affects F, R, B, L bottom rows (opposite direction)
		c.cycle4(
			[3]int{int(CubeFaceF), 6, 1}, [3]int{int(CubeFaceF), 7, 1}, [3]int{int(CubeFaceF), 8, 1},
			[3]int{int(CubeFaceR), 6, 1}, [3]int{int(CubeFaceR), 7, 1}, [3]int{int(CubeFaceR), 8, 1},
			[3]int{int(CubeFaceB), 6, 1}, [3]int{int(CubeFaceB), 7, 1}, [3]int{int(CubeFaceB), 8, 1},
			[3]int{int(CubeFaceL), 6, 1}, [3]int{int(CubeFaceL), 7, 1}, [3]int{int(CubeFaceL), 8, 1},
		)
	case CubeFaceF:
		// F affects U bottom, R left, D top, L right
		c.cycle4Edge(
			int(CubeFaceU), []int{6, 7, 8},
			int(CubeFaceR), []int{0, 3, 6},
			int(CubeFaceD), []int{2, 1, 0},
			int(CubeFaceL), []int{8, 5, 2},
		)
	case CubeFaceB:
		// B affects U top, L left, D bottom, R right
		c.cycle4Edge(
			int(CubeFaceU), []int{2, 1, 0},
			int(CubeFaceL), []int{0, 3, 6},
			int(CubeFaceD), []int{6, 7, 8},
			int(CubeFaceR), []int{8, 5, 2},
		)
	case CubeFaceR:
		// R affects U right, B left, D right, F right
		c.cycle4Edge(
			int(CubeFaceU), []int{2, 5, 8},
			int(CubeFaceB), []int{6, 3, 0},
			int(CubeFaceD), []int{2, 5, 8},
			int(CubeFaceF), []int{2, 5, 8},
		)
	case CubeFaceL:
		// L affects U left, F left, D left, B right
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
	// CCW is just CW three times, or we can reverse the cycle
	c.cycleEdgesCW(face)
	c.cycleEdgesCW(face)
	c.cycleEdgesCW(face)
}

// cycle4 cycles 4 groups of 3 facelets (for U and D moves).
func (c *Cube) cycle4(a1, a2, a3, b1, b2, b3, c1, c2, c3, d1, d2, d3 [3]int) {
	// Save first group
	t1 := c.Facelets[a1[0]][a1[1]]
	t2 := c.Facelets[a2[0]][a2[1]]
	t3 := c.Facelets[a3[0]][a3[1]]

	// a <- d
	c.Facelets[a1[0]][a1[1]] = c.Facelets[d1[0]][d1[1]]
	c.Facelets[a2[0]][a2[1]] = c.Facelets[d2[0]][d2[1]]
	c.Facelets[a3[0]][a3[1]] = c.Facelets[d3[0]][d3[1]]

	// d <- c
	c.Facelets[d1[0]][d1[1]] = c.Facelets[c1[0]][c1[1]]
	c.Facelets[d2[0]][d2[1]] = c.Facelets[c2[0]][c2[1]]
	c.Facelets[d3[0]][d3[1]] = c.Facelets[c3[0]][c3[1]]

	// c <- b
	c.Facelets[c1[0]][c1[1]] = c.Facelets[b1[0]][b1[1]]
	c.Facelets[c2[0]][c2[1]] = c.Facelets[b2[0]][b2[1]]
	c.Facelets[c3[0]][c3[1]] = c.Facelets[b3[0]][b3[1]]

	// b <- a (saved)
	c.Facelets[b1[0]][b1[1]] = t1
	c.Facelets[b2[0]][b2[1]] = t2
	c.Facelets[b3[0]][b3[1]] = t3
}

// cycle4Edge cycles 4 edges with arbitrary indices.
func (c *Cube) cycle4Edge(f1 int, i1 []int, f2 int, i2 []int, f3 int, i3 []int, f4 int, i4 []int) {
	// Save first edge
	t := [3]Color{
		c.Facelets[f1][i1[0]],
		c.Facelets[f1][i1[1]],
		c.Facelets[f1][i1[2]],
	}

	// 1 <- 4
	c.Facelets[f1][i1[0]] = c.Facelets[f4][i4[0]]
	c.Facelets[f1][i1[1]] = c.Facelets[f4][i4[1]]
	c.Facelets[f1][i1[2]] = c.Facelets[f4][i4[2]]

	// 4 <- 3
	c.Facelets[f4][i4[0]] = c.Facelets[f3][i3[0]]
	c.Facelets[f4][i4[1]] = c.Facelets[f3][i3[1]]
	c.Facelets[f4][i4[2]] = c.Facelets[f3][i3[2]]

	// 3 <- 2
	c.Facelets[f3][i3[0]] = c.Facelets[f2][i2[0]]
	c.Facelets[f3][i3[1]] = c.Facelets[f2][i2[1]]
	c.Facelets[f3][i3[2]] = c.Facelets[f2][i2[2]]

	// 2 <- 1 (saved)
	c.Facelets[f2][i2[0]] = t[0]
	c.Facelets[f2][i2[1]] = t[1]
	c.Facelets[f2][i2[2]] = t[2]
}

// String returns a text representation of the cube.
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
	return fmt.Sprintf("Solved: %v", c.IsSolved())
}

// ApplyMove applies a Move to the cube.
func (c *Cube) ApplyMove(m Move) {
	face := moveFaceToCubeFace(m.Face)
	turn := int(m.Turn)
	c.MoveFace(face, turn)
}

// ApplyMoves applies a sequence of moves to the cube.
func (c *Cube) ApplyMoves(moves []Move) {
	for _, m := range moves {
		c.ApplyMove(m)
	}
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
