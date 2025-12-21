// Package types contains shared type definitions for the gocube application.
package types

// Face represents a cube face in standard notation.
type Face string

const (
	FaceR Face = "R" // Right
	FaceL Face = "L" // Left
	FaceU Face = "U" // Up
	FaceD Face = "D" // Down
	FaceF Face = "F" // Front
	FaceB Face = "B" // Back
)

// Turn represents the direction and magnitude of a face turn.
type Turn int

const (
	TurnCW  Turn = 1  // Clockwise quarter turn
	TurnCCW Turn = -1 // Counter-clockwise quarter turn
	Turn180 Turn = 2  // 180 degree turn (half turn)
)

// Move represents a single cube move with face and turn direction.
type Move struct {
	Face      Face  `json:"face"`
	Turn      Turn  `json:"turn"`
	Timestamp int64 `json:"ts_ms"` // Milliseconds since solve start
}

// Notation returns the standard cube notation string for this move.
// Examples: R, R', R2, U, U', U2
func (m Move) Notation() string {
	suffix := ""
	switch m.Turn {
	case TurnCCW:
		suffix = "'"
	case Turn180:
		suffix = "2"
	}
	return string(m.Face) + suffix
}

// Inverse returns the inverse of this move.
func (m Move) Inverse() Move {
	inv := m
	switch m.Turn {
	case TurnCW:
		inv.Turn = TurnCCW
	case TurnCCW:
		inv.Turn = TurnCW
	// Turn180 is its own inverse
	}
	return inv
}

// IsCancellation returns true if the other move cancels this move.
func (m Move) IsCancellation(other Move) bool {
	if m.Face != other.Face {
		return false
	}
	return m.Turn == -other.Turn ||
		(m.Turn == Turn180 && other.Turn == Turn180)
}

// CanMerge returns true if two adjacent same-face moves can be merged.
func (m Move) CanMerge(other Move) bool {
	return m.Face == other.Face
}

// Merge combines two same-face moves into one (or returns nil if they cancel).
// Returns nil if the moves cannot be merged or if they cancel out completely.
func (m Move) Merge(other Move) *Move {
	if m.Face != other.Face {
		return nil
	}

	combined := int(m.Turn) + int(other.Turn)
	// Normalize: -2 and 2 are both half turns, values outside [-2,2] wrap
	combined = ((combined + 2) % 4) - 2
	if combined > 2 {
		combined -= 4
	} else if combined < -2 {
		combined += 4
	}

	if combined == 0 {
		return nil // Moves cancel out
	}

	// Normalize half turn direction
	if combined == -2 {
		combined = 2
	}

	return &Move{
		Face:      m.Face,
		Turn:      Turn(combined),
		Timestamp: other.Timestamp,
	}
}

// Token encodes the move as a single byte for efficient n-gram processing.
// Encoding: face*3 + turn_code where:
//   - face: R=0, L=1, U=2, D=3, F=4, B=5
//   - turn_code: CCW=0, CW=1, 180=2
func (m Move) Token() uint8 {
	var faceCode uint8
	switch m.Face {
	case FaceR:
		faceCode = 0
	case FaceL:
		faceCode = 1
	case FaceU:
		faceCode = 2
	case FaceD:
		faceCode = 3
	case FaceF:
		faceCode = 4
	case FaceB:
		faceCode = 5
	}

	var turnCode uint8
	switch m.Turn {
	case TurnCCW:
		turnCode = 0
	case TurnCW:
		turnCode = 1
	case Turn180:
		turnCode = 2
	}

	return faceCode*3 + turnCode
}

// MoveFromToken decodes a token back into a Move.
func MoveFromToken(token uint8) Move {
	faceCode := token / 3
	turnCode := token % 3

	var face Face
	switch faceCode {
	case 0:
		face = FaceR
	case 1:
		face = FaceL
	case 2:
		face = FaceU
	case 3:
		face = FaceD
	case 4:
		face = FaceF
	case 5:
		face = FaceB
	}

	var turn Turn
	switch turnCode {
	case 0:
		turn = TurnCCW
	case 1:
		turn = TurnCW
	case 2:
		turn = Turn180
	}

	return Move{Face: face, Turn: turn}
}
