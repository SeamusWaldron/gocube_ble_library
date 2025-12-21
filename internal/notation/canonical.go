// Package notation provides move notation conversion utilities.
package notation

import (
	"strings"

	"github.com/seamusw/gocube/pkg/types"
)

// ParseNotation parses a standard cube notation string into a Move.
// Examples: R, R', R2, U, U', U2
func ParseNotation(s string) (types.Move, bool) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return types.Move{}, false
	}

	// Extract face
	faceChar := s[0]
	var face types.Face
	switch faceChar {
	case 'R', 'r':
		face = types.FaceR
	case 'L', 'l':
		face = types.FaceL
	case 'U', 'u':
		face = types.FaceU
	case 'D', 'd':
		face = types.FaceD
	case 'F', 'f':
		face = types.FaceF
	case 'B', 'b':
		face = types.FaceB
	default:
		return types.Move{}, false
	}

	// Extract turn
	turn := types.TurnCW // Default is clockwise
	if len(s) > 1 {
		suffix := s[1:]
		switch suffix {
		case "'", "`":
			turn = types.TurnCCW
		case "2":
			turn = types.Turn180
		case "2'":
			turn = types.Turn180 // Same as 180
		default:
			return types.Move{}, false
		}
	}

	return types.Move{Face: face, Turn: turn}, true
}

// ParseSequence parses a space-separated sequence of moves.
func ParseSequence(s string) ([]types.Move, error) {
	parts := strings.Fields(s)
	moves := make([]types.Move, 0, len(parts))

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
func FormatSequence(moves []types.Move) string {
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
func NormalizeTurn(turn int) types.Turn {
	// Normalize to [-2, 2]
	turn = ((turn % 4) + 4) % 4
	if turn == 3 {
		turn = -1
	}
	if turn == 0 {
		return types.TurnCW // Shouldn't happen, but treat as CW
	}
	return types.Turn(turn)
}
