// Package notation provides move notation conversion utilities.
package notation

import (
	"strings"

	"github.com/SeamusWaldron/gocube_ble_library"
)

// ParseNotation parses a standard cube notation string into a Move.
// Examples: R, R', R2, U, U', U2
func ParseNotation(s string) (gocube.Move, bool) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return gocube.Move{}, false
	}

	// Extract face
	faceChar := s[0]
	var face gocube.Face
	switch faceChar {
	case 'R', 'r':
		face = gocube.FaceR
	case 'L', 'l':
		face = gocube.FaceL
	case 'U', 'u':
		face = gocube.FaceU
	case 'D', 'd':
		face = gocube.FaceD
	case 'F', 'f':
		face = gocube.FaceF
	case 'B', 'b':
		face = gocube.FaceB
	default:
		return gocube.Move{}, false
	}

	// Extract turn
	turn := gocube.CW // Default is clockwise
	if len(s) > 1 {
		suffix := s[1:]
		switch suffix {
		case "'", "`":
			turn = gocube.CCW
		case "2":
			turn = gocube.Double
		case "2'":
			turn = gocube.Double // Same as 180
		default:
			return gocube.Move{}, false
		}
	}

	return gocube.Move{Face: face, Turn: turn}, true
}

// ParseSequence parses a space-separated sequence of moves.
func ParseSequence(s string) ([]gocube.Move, error) {
	parts := strings.Fields(s)
	moves := make([]gocube.Move, 0, len(parts))

	for _, part := range parts {
		move, ok := ParseNotation(part)
		if !ok {
			continue // Skip invalid moves
		}
		moves = append(moves, move)
	}

	return moves, nil
}

// FormatSequence formats a slice of moves as a space-separated string.
func FormatSequence(moves []gocube.Move) string {
	if len(moves) == 0 {
		return ""
	}

	parts := make([]string, len(moves))
	for i, m := range moves {
		parts[i] = m.Notation()
	}

	return strings.Join(parts, " ")
}

// NormalizeTurn normalizes a turn value to the range [-1, 2].
// -3 -> 1, -2 -> 2, -1 -> -1, 0 -> 0, 1 -> 1, 2 -> 2, 3 -> -1
func NormalizeTurn(turn int) gocube.Turn {
	// Normalize to [-2, 2]
	turn = ((turn % 4) + 4) % 4
	if turn == 3 {
		turn = -1
	}
	if turn == 0 {
		return gocube.CW // Shouldn't happen, but treat as CW
	}
	return gocube.Turn(turn)
}
