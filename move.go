package gocube

import (
	"strings"
	"time"
)

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
	CW     Turn = 1  // Clockwise (90 degrees)
	CCW    Turn = -1 // Counter-clockwise (90 degrees)
	Double Turn = 2  // Half turn (180 degrees)
)

// Move represents a single cube move with face, turn direction, and optional timestamp.
type Move struct {
	Face Face      // Which face to turn
	Turn Turn      // Direction and amount
	Time time.Time // When the move occurred (optional)
}

// Notation returns the standard cube notation string for this move.
// Examples: R, R', R2, U, U', U2
func (m Move) Notation() string {
	suffix := ""
	switch m.Turn {
	case CCW:
		suffix = "'"
	case Double:
		suffix = "2"
	}
	return string(m.Face) + suffix
}

// Inverse returns the inverse of this move.
// R becomes R', R' becomes R, R2 stays R2.
func (m Move) Inverse() Move {
	inv := m
	switch m.Turn {
	case CW:
		inv.Turn = CCW
	case CCW:
		inv.Turn = CW
	// Double is its own inverse
	}
	return inv
}

// WithTime returns a copy of the move with the specified timestamp.
func (m Move) WithTime(t time.Time) Move {
	m.Time = t
	return m
}

// String returns the notation string (alias for Notation).
func (m Move) String() string {
	return m.Notation()
}

// ParseMove parses a standard notation string into a Move.
// Examples: R, R', R2, U, U', U2
// Returns an error if the notation is invalid.
func ParseMove(s string) (Move, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return Move{}, ErrInvalidNotation
	}

	// Extract face
	faceChar := s[0]
	var face Face
	switch faceChar {
	case 'R', 'r':
		face = FaceR
	case 'L', 'l':
		face = FaceL
	case 'U', 'u':
		face = FaceU
	case 'D', 'd':
		face = FaceD
	case 'F', 'f':
		face = FaceF
	case 'B', 'b':
		face = FaceB
	default:
		return Move{}, ErrInvalidNotation
	}

	// Extract turn
	turn := CW // Default is clockwise
	if len(s) > 1 {
		suffix := s[1:]
		switch suffix {
		case "'", "`":
			turn = CCW
		case "2":
			turn = Double
		case "2'", "2`":
			turn = Double // Same as 180
		default:
			return Move{}, ErrInvalidNotation
		}
	}

	return Move{Face: face, Turn: turn}, nil
}

// ParseMoves parses a space-separated sequence of moves.
// Example: "R U R' U'"
// Invalid moves are skipped.
func ParseMoves(s string) ([]Move, error) {
	parts := strings.Fields(s)
	moves := make([]Move, 0, len(parts))

	for _, part := range parts {
		move, err := ParseMove(part)
		if err != nil {
			continue // Skip invalid moves
		}
		moves = append(moves, move)
	}

	return moves, nil
}

// FormatMoves formats a slice of moves as a space-separated notation string.
func FormatMoves(moves []Move) string {
	if len(moves) == 0 {
		return ""
	}

	parts := make([]string, len(moves))
	for i, m := range moves {
		parts[i] = m.Notation()
	}

	return strings.Join(parts, " ")
}
