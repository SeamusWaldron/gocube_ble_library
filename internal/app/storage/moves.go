package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/SeamusWaldron/gocube_ble_library"
)

// MoveRecord represents a move in the database.
type MoveRecord struct {
	MoveID        int64
	SolveID       string
	MoveIndex     int
	TsMs          int64
	Face          string
	Turn          int
	Notation      string
	SourceEventID *int64
}

// MoveRepository provides CRUD operations for moves.
type MoveRepository struct {
	db *DB
}

// NewMoveRepository creates a new move repository.
func NewMoveRepository(db *DB) *MoveRepository {
	return &MoveRepository{db: db}
}

// Create creates a new move and returns its ID.
func (r *MoveRepository) Create(solveID string, moveIndex int, tsMs int64, move gocube.Move, sourceEventID *int64) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO moves (solve_id, move_index, ts_ms, face, turn, notation, source_event_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, solveID, moveIndex, tsMs, string(move.Face), int(move.Turn), move.Notation(), sourceEventID)

	if err != nil {
		return 0, fmt.Errorf("failed to create move: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get move ID: %w", err)
	}

	return id, nil
}

// CreateBatch creates multiple moves in a single transaction.
func (r *MoveRepository) CreateBatch(solveID string, moves []gocube.Move, startIndex int, sourceEventID *int64) error {
	return r.db.Transaction(func(tx *sql.Tx) error {
		for i, move := range moves {
			tsMs := move.Time.UnixMilli()
			_, err := tx.Exec(`
				INSERT INTO moves (solve_id, move_index, ts_ms, face, turn, notation, source_event_id)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, solveID, startIndex+i, tsMs, string(move.Face), int(move.Turn), move.Notation(), sourceEventID)
			if err != nil {
				return fmt.Errorf("failed to create move %d: %w", startIndex+i, err)
			}
		}
		return nil
	})
}

// GetBySolve retrieves all moves for a solve in order.
func (r *MoveRepository) GetBySolve(solveID string) ([]MoveRecord, error) {
	rows, err := r.db.Query(`
		SELECT move_id, solve_id, move_index, ts_ms, face, turn, notation, source_event_id
		FROM moves
		WHERE solve_id = ?
		ORDER BY move_index
	`, solveID)

	if err != nil {
		return nil, fmt.Errorf("failed to get moves: %w", err)
	}
	defer rows.Close()

	var moves []MoveRecord
	for rows.Next() {
		var m MoveRecord
		err := rows.Scan(&m.MoveID, &m.SolveID, &m.MoveIndex, &m.TsMs, &m.Face, &m.Turn, &m.Notation, &m.SourceEventID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan move: %w", err)
		}
		moves = append(moves, m)
	}

	return moves, nil
}

// GetBySolveRange retrieves moves in a time range for a solve.
// Uses inclusive start (>=) and exclusive end (<) to prevent moves at phase
// boundaries from being counted in both phases.
func (r *MoveRepository) GetBySolveRange(solveID string, startTsMs, endTsMs int64) ([]MoveRecord, error) {
	rows, err := r.db.Query(`
		SELECT move_id, solve_id, move_index, ts_ms, face, turn, notation, source_event_id
		FROM moves
		WHERE solve_id = ? AND ts_ms >= ? AND ts_ms < ?
		ORDER BY move_index
	`, solveID, startTsMs, endTsMs)

	if err != nil {
		return nil, fmt.Errorf("failed to get moves in range: %w", err)
	}
	defer rows.Close()

	var moves []MoveRecord
	for rows.Next() {
		var m MoveRecord
		err := rows.Scan(&m.MoveID, &m.SolveID, &m.MoveIndex, &m.TsMs, &m.Face, &m.Turn, &m.Notation, &m.SourceEventID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan move: %w", err)
		}
		moves = append(moves, m)
	}

	return moves, nil
}

// GetNextIndex returns the next move index for a solve.
func (r *MoveRepository) GetNextIndex(solveID string) (int, error) {
	var maxIndex int
	err := r.db.QueryRow(`
		SELECT COALESCE(MAX(move_index), -1) FROM moves WHERE solve_id = ?
	`, solveID).Scan(&maxIndex)
	if err != nil {
		return 0, fmt.Errorf("failed to get max move index: %w", err)
	}
	return maxIndex + 1, nil
}

// Count returns the number of moves for a solve.
func (r *MoveRepository) Count(solveID string) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM moves WHERE solve_id = ?", solveID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count moves: %w", err)
	}
	return count, nil
}

// ToMoves converts MoveRecords to gocube.Move slice.
func ToMoves(records []MoveRecord) []gocube.Move {
	moves := make([]gocube.Move, len(records))
	for i, r := range records {
		moves[i] = gocube.Move{
			Face: gocube.Face(r.Face),
			Turn: gocube.Turn(r.Turn),
			Time: time.UnixMilli(r.TsMs),
		}
	}
	return moves
}
