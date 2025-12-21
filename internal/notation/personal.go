package notation

import (
	"github.com/SeamusWaldron/gocube"
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
func ToPersonalNotation(m gocube.Move) string {
	switch m.Face {
	case gocube.FaceR:
		switch m.Turn {
		case gocube.TurnCW:
			return "R up"
		case gocube.TurnCCW:
			return "R down"
		case gocube.Turn180:
			return "R up x 2"
		}

	case gocube.FaceL:
		switch m.Turn {
		case gocube.TurnCW:
			return "L down"
		case gocube.TurnCCW:
			return "L up"
		case gocube.Turn180:
			return "L down x 2"
		}

	case gocube.FaceU:
		switch m.Turn {
		case gocube.TurnCW:
			return "T rotate right"
		case gocube.TurnCCW:
			return "T rotate left"
		case gocube.Turn180:
			return "T rotate right x 2"
		}

	case gocube.FaceD:
		switch m.Turn {
		case gocube.TurnCW:
			return "B rotate right"
		case gocube.TurnCCW:
			return "B rotate left"
		case gocube.Turn180:
			return "B rotate right x 2"
		}

	case gocube.FaceF:
		switch m.Turn {
		case gocube.TurnCW:
			return "F rotate clockwise"
		case gocube.TurnCCW:
			return "F rotate anti-clockwise"
		case gocube.Turn180:
			return "F rotate x 2"
		}

	case gocube.FaceB:
		switch m.Turn {
		case gocube.TurnCW:
			return "Back rotate clockwise"
		case gocube.TurnCCW:
			return "Back rotate anti-clockwise"
		case gocube.Turn180:
			return "Back rotate x 2"
		}
	}

	return m.Notation() // Fallback to standard notation
}

// ToPersonalSequence converts a slice of moves to personal notation strings.
func ToPersonalSequence(moves []gocube.Move) []string {
	result := make([]string, len(moves))
	for i, m := range moves {
		result[i] = ToPersonalNotation(m)
	}
	return result
}

// FormatPersonalSequence formats moves as a comma-separated personal notation string.
func FormatPersonalSequence(moves []gocube.Move) string {
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
