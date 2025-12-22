package cli

import (
	"time"

	"github.com/SeamusWaldron/gocube_ble_library"
	"github.com/SeamusWaldron/gocube_ble_library/internal/protocol"
)

// colorToFace maps GoCube color names to Face constants.
var colorToFace = map[string]gocube.Face{
	"white":  gocube.FaceU,
	"yellow": gocube.FaceD,
	"green":  gocube.FaceF,
	"blue":   gocube.FaceB,
	"red":    gocube.FaceR,
	"orange": gocube.FaceL,
}

// rotationsToMoves converts rotation events to Move objects.
func rotationsToMoves(rotations []protocol.RotationEvent, t time.Time) []gocube.Move {
	moves := make([]gocube.Move, len(rotations))
	for i, rot := range rotations {
		face := colorToFace[rot.Color]
		var turn gocube.Turn
		if rot.Clockwise {
			turn = gocube.CW
		} else {
			turn = gocube.CCW
		}
		moves[i] = gocube.Move{Face: face, Turn: turn, Time: t}
	}
	return moves
}

// phaseToKey converts a Phase enum to the storage key string.
func phaseToKey(p gocube.Phase) string {
	switch p {
	case gocube.PhaseScrambled:
		return "scrambled"
	case gocube.PhaseWhiteCross:
		return "white_cross"
	case gocube.PhaseFirstLayer:
		return "top_corners"
	case gocube.PhaseSecondLayer:
		return "middle_layer"
	case gocube.PhaseYellowCross:
		return "bottom_cross"
	case gocube.PhaseYellowCorners:
		return "position_corners"
	case gocube.PhaseYellowOriented:
		return "orient_corners"
	case gocube.PhaseSolved:
		return "complete"
	default:
		return "scrambled"
	}
}
