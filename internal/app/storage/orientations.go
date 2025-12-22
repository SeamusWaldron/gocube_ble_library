package storage

import (
	"fmt"
)

// OrientationRecord represents an orientation state in the database.
type OrientationRecord struct {
	OrientationID int64
	SolveID       string
	TsMs          int64
	UpFace        string
	FrontFace     string
	SourceEventID *int64
}

// OrientationRepository provides CRUD operations for orientations.
type OrientationRepository struct {
	db *DB
}

// NewOrientationRepository creates a new orientation repository.
func NewOrientationRepository(db *DB) *OrientationRepository {
	return &OrientationRepository{db: db}
}

// Create creates a new orientation record and returns its ID.
func (r *OrientationRepository) Create(solveID string, tsMs int64, upFace, frontFace string, sourceEventID *int64) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO orientations (solve_id, ts_ms, up_face, front_face, source_event_id)
		VALUES (?, ?, ?, ?, ?)
	`, solveID, tsMs, upFace, frontFace, sourceEventID)

	if err != nil {
		return 0, fmt.Errorf("failed to create orientation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get orientation ID: %w", err)
	}

	return id, nil
}

// GetBySolve retrieves all orientation records for a solve.
func (r *OrientationRepository) GetBySolve(solveID string) ([]OrientationRecord, error) {
	rows, err := r.db.Query(`
		SELECT orientation_id, solve_id, ts_ms, up_face, front_face, source_event_id
		FROM orientations
		WHERE solve_id = ?
		ORDER BY ts_ms
	`, solveID)

	if err != nil {
		return nil, fmt.Errorf("failed to get orientations: %w", err)
	}
	defer rows.Close()

	var orientations []OrientationRecord
	for rows.Next() {
		var o OrientationRecord
		err := rows.Scan(&o.OrientationID, &o.SolveID, &o.TsMs, &o.UpFace, &o.FrontFace, &o.SourceEventID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan orientation: %w", err)
		}
		orientations = append(orientations, o)
	}

	return orientations, nil
}

// GetBySolveRange retrieves orientation records in a timestamp range.
func (r *OrientationRepository) GetBySolveRange(solveID string, startMs, endMs int64) ([]OrientationRecord, error) {
	rows, err := r.db.Query(`
		SELECT orientation_id, solve_id, ts_ms, up_face, front_face, source_event_id
		FROM orientations
		WHERE solve_id = ? AND ts_ms >= ? AND ts_ms < ?
		ORDER BY ts_ms
	`, solveID, startMs, endMs)

	if err != nil {
		return nil, fmt.Errorf("failed to get orientations by range: %w", err)
	}
	defer rows.Close()

	var orientations []OrientationRecord
	for rows.Next() {
		var o OrientationRecord
		err := rows.Scan(&o.OrientationID, &o.SolveID, &o.TsMs, &o.UpFace, &o.FrontFace, &o.SourceEventID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan orientation: %w", err)
		}
		orientations = append(orientations, o)
	}

	return orientations, nil
}

// Count returns the number of orientation changes for a solve.
func (r *OrientationRepository) Count(solveID string) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM orientations WHERE solve_id = ?", solveID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count orientations: %w", err)
	}
	return count, nil
}

// GetLast returns the most recent orientation for a solve.
func (r *OrientationRepository) GetLast(solveID string) (*OrientationRecord, error) {
	row := r.db.QueryRow(`
		SELECT orientation_id, solve_id, ts_ms, up_face, front_face, source_event_id
		FROM orientations
		WHERE solve_id = ?
		ORDER BY ts_ms DESC
		LIMIT 1
	`, solveID)

	var o OrientationRecord
	err := row.Scan(&o.OrientationID, &o.SolveID, &o.TsMs, &o.UpFace, &o.FrontFace, &o.SourceEventID)
	if err != nil {
		return nil, nil // No orientation found
	}

	return &o, nil
}
