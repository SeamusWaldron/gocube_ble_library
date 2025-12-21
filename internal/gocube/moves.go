package gocube

import (
	"github.com/seamusw/gocube/pkg/types"
)

// ColorToFace maps GoCube color names to standard cube face notation.
// This mapping assumes the standard orientation: White on top, Green in front.
var ColorToFace = map[string]types.Face{
	"white":  types.FaceU, // Up
	"yellow": types.FaceD, // Down
	"green":  types.FaceF, // Front
	"blue":   types.FaceB, // Back
	"red":    types.FaceR, // Right
	"orange": types.FaceL, // Left
}

// RotationToMove converts a GoCube rotation event to a canonical Move.
func RotationToMove(rot RotationEvent, timestampMs int64) types.Move {
	face := ColorToFace[rot.Color]

	var turn types.Turn
	if rot.Clockwise {
		turn = types.TurnCW
	} else {
		turn = types.TurnCCW
	}

	return types.Move{
		Face:      face,
		Turn:      turn,
		Timestamp: timestampMs,
	}
}

// RotationsToMoves converts a slice of rotation events to canonical moves.
// It also handles merging adjacent same-face moves (e.g., two R clockwise = R2).
func RotationsToMoves(rotations []RotationEvent, timestampMs int64) []types.Move {
	if len(rotations) == 0 {
		return nil
	}

	moves := make([]types.Move, 0, len(rotations))
	for _, rot := range rotations {
		move := RotationToMove(rot, timestampMs)
		moves = append(moves, move)
	}

	// Merge adjacent same-face moves
	return MergeMoves(moves)
}

// MergeMoves merges adjacent same-face moves.
// For example: R R becomes R2, R R R becomes R2 R', R R R R cancels out.
func MergeMoves(moves []types.Move) []types.Move {
	if len(moves) <= 1 {
		return moves
	}

	result := make([]types.Move, 0, len(moves))

	for _, move := range moves {
		if len(result) == 0 {
			result = append(result, move)
			continue
		}

		last := &result[len(result)-1]
		if last.Face == move.Face {
			// Try to merge
			merged := last.Merge(move)
			if merged == nil {
				// Moves cancelled out - remove the last move
				result = result[:len(result)-1]
			} else {
				// Replace last with merged
				*last = *merged
			}
		} else {
			result = append(result, move)
		}
	}

	return result
}

// MovesToNotation converts a slice of moves to notation strings.
func MovesToNotation(moves []types.Move) []string {
	result := make([]string, len(moves))
	for i, m := range moves {
		result[i] = m.Notation()
	}
	return result
}

// MovesToNotationString converts a slice of moves to a single space-separated notation string.
func MovesToNotationString(moves []types.Move) string {
	notations := MovesToNotation(moves)
	result := ""
	for i, n := range notations {
		if i > 0 {
			result += " "
		}
		result += n
	}
	return result
}
