package analysis

import (
	"github.com/SeamusWaldron/gocube_ble_library"
)

// Token encoding for efficient n-gram detection.
// Encodes a Move as a single byte: face (0-5) * 3 + turn (0-2)
// This gives us 18 unique values for all possible moves.

var faceToIndex = map[gocube.Face]uint8{
	gocube.FaceR: 0,
	gocube.FaceL: 1,
	gocube.FaceU: 2,
	gocube.FaceD: 3,
	gocube.FaceF: 4,
	gocube.FaceB: 5,
}

var indexToFace = []gocube.Face{
	gocube.FaceR,
	gocube.FaceL,
	gocube.FaceU,
	gocube.FaceD,
	gocube.FaceF,
	gocube.FaceB,
}

func turnToIndex(t gocube.Turn) uint8 {
	switch t {
	case gocube.CW:
		return 0
	case gocube.CCW:
		return 1
	case gocube.Double:
		return 2
	default:
		return 0
	}
}

func indexToTurn(i uint8) gocube.Turn {
	switch i {
	case 0:
		return gocube.CW
	case 1:
		return gocube.CCW
	case 2:
		return gocube.Double
	default:
		return gocube.CW
	}
}

// moveToken encodes a Move as a single byte for efficient hashing.
func moveToken(m gocube.Move) uint8 {
	faceIdx := faceToIndex[m.Face]
	turnIdx := turnToIndex(m.Turn)
	return faceIdx*3 + turnIdx
}

// moveFromToken decodes a token back to a Move.
func moveFromToken(token uint8) gocube.Move {
	faceIdx := token / 3
	turnIdx := token % 3
	if int(faceIdx) >= len(indexToFace) {
		faceIdx = 0
	}
	return gocube.Move{
		Face: indexToFace[faceIdx],
		Turn: indexToTurn(turnIdx),
	}
}

// mergeMoves merges two same-face moves into one.
// Returns nil if they cancel out (e.g., R + R' = nothing).
// Assumes m1 and m2 have the same face.
func mergeMoves(m1, m2 gocube.Move) *gocube.Move {
	// Sum the turns: CW=1, CCW=-1, Double=2
	totalTurn := int(m1.Turn) + int(m2.Turn)

	// Normalize to [-2, 2] then to valid turn
	totalTurn = ((totalTurn % 4) + 4) % 4
	if totalTurn == 3 {
		totalTurn = -1 // 3 quarter turns = 1 CCW
	}

	// 0 means full cancellation
	if totalTurn == 0 {
		return nil
	}

	return &gocube.Move{
		Face: m1.Face,
		Turn: gocube.Turn(totalTurn),
		Time: m1.Time, // Keep timestamp of first move
	}
}
