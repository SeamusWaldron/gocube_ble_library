package notation

import (
	"github.com/seamusw/gocube/pkg/types"
)

// ToPersonalNotation converts a Move to your personal notation.
// Reference frame: White on top, Green in front, facing the cube.
//
// Mapping:
//   R  -> "R up"        R' -> "R down"       R2 -> "R up x 2"
//   L  -> "L down"      L' -> "L up"         L2 -> "L down x 2"
//   U  -> "T rotate right"  U' -> "T rotate left"  U2 -> "T rotate right x 2"
//   D  -> "B rotate right"  D' -> "B rotate left"  D2 -> "B rotate right x 2"
//   F  -> "F rotate clockwise"  F' -> "F rotate anti-clockwise"  F2 -> "F rotate x 2"
//   B  -> "Back rotate clockwise"  B' -> "Back rotate anti-clockwise"  B2 -> "Back rotate x 2"
func ToPersonalNotation(m types.Move) string {
	switch m.Face {
	case types.FaceR:
		switch m.Turn {
		case types.TurnCW:
			return "R up"
		case types.TurnCCW:
			return "R down"
		case types.Turn180:
			return "R up x 2"
		}

	case types.FaceL:
		switch m.Turn {
		case types.TurnCW:
			return "L down"
		case types.TurnCCW:
			return "L up"
		case types.Turn180:
			return "L down x 2"
		}

	case types.FaceU:
		switch m.Turn {
		case types.TurnCW:
			return "T rotate right"
		case types.TurnCCW:
			return "T rotate left"
		case types.Turn180:
			return "T rotate right x 2"
		}

	case types.FaceD:
		switch m.Turn {
		case types.TurnCW:
			return "B rotate right"
		case types.TurnCCW:
			return "B rotate left"
		case types.Turn180:
			return "B rotate right x 2"
		}

	case types.FaceF:
		switch m.Turn {
		case types.TurnCW:
			return "F rotate clockwise"
		case types.TurnCCW:
			return "F rotate anti-clockwise"
		case types.Turn180:
			return "F rotate x 2"
		}

	case types.FaceB:
		switch m.Turn {
		case types.TurnCW:
			return "Back rotate clockwise"
		case types.TurnCCW:
			return "Back rotate anti-clockwise"
		case types.Turn180:
			return "Back rotate x 2"
		}
	}

	return m.Notation() // Fallback to standard notation
}

// ToPersonalSequence converts a slice of moves to personal notation strings.
func ToPersonalSequence(moves []types.Move) []string {
	result := make([]string, len(moves))
	for i, m := range moves {
		result[i] = ToPersonalNotation(m)
	}
	return result
}

// FormatPersonalSequence formats moves as a comma-separated personal notation string.
func FormatPersonalSequence(moves []types.Move) string {
	if len(moves) == 0 {
		return ""
	}

	result := ""
	for i, m := range moves {
		if i > 0 {
			result += ", "
		}
		result += ToPersonalNotation(m)
	}
	return result
}
