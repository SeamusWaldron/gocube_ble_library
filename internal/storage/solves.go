package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Solve represents a solve session in the database.
type Solve struct {
	SolveID     string
	StartedAt   time.Time
	EndedAt     *time.Time
	DurationMs  *int64
	ScrambleText *string
	Notes       *string
	DeviceName  *string
	DeviceID    *string
	AppVersion  *string
}

// SolveRepository provides CRUD operations for solves.
type SolveRepository struct {
	db *DB
}

// NewSolveRepository creates a new solve repository.
func NewSolveRepository(db *DB) *SolveRepository {
	return &SolveRepository{db: db}
}

// Create creates a new solve and returns its ID.
func (r *SolveRepository) Create(notes, scramble, deviceName, deviceID, appVersion string) (string, error) {
	id := uuid.New().String()
	startedAt := time.Now().UTC()

	var notesPtr, scramblePtr, deviceNamePtr, deviceIDPtr, appVersionPtr *string
	if notes != "" {
		notesPtr = &notes
	}
	if scramble != "" {
		scramblePtr = &scramble
	}
	if deviceName != "" {
		deviceNamePtr = &deviceName
	}
	if deviceID != "" {
		deviceIDPtr = &deviceID
	}
	if appVersion != "" {
		appVersionPtr = &appVersion
	}

	_, err := r.db.Exec(`
		INSERT INTO solves (solve_id, started_at, notes, scramble_text, device_name, device_id, app_version)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, startedAt.Format(time.RFC3339), notesPtr, scramblePtr, deviceNamePtr, deviceIDPtr, appVersionPtr)

	if err != nil {
		return "", fmt.Errorf("failed to create solve: %w", err)
	}

	return id, nil
}

// End marks a solve as complete.
func (r *SolveRepository) End(solveID string) error {
	endedAt := time.Now().UTC()

	// Get start time to calculate duration
	var startedAtStr string
	err := r.db.QueryRow("SELECT started_at FROM solves WHERE solve_id = ?", solveID).Scan(&startedAtStr)
	if err != nil {
		return fmt.Errorf("failed to get solve start time: %w", err)
	}

	startedAt, err := time.Parse(time.RFC3339, startedAtStr)
	if err != nil {
		return fmt.Errorf("failed to parse start time: %w", err)
	}

	durationMs := endedAt.Sub(startedAt).Milliseconds()

	_, err = r.db.Exec(`
		UPDATE solves
		SET ended_at = ?, duration_ms = ?
		WHERE solve_id = ?
	`, endedAt.Format(time.RFC3339), durationMs, solveID)

	if err != nil {
		return fmt.Errorf("failed to end solve: %w", err)
	}

	return nil
}

// Get retrieves a solve by ID.
func (r *SolveRepository) Get(solveID string) (*Solve, error) {
	var s Solve
	var startedAtStr string
	var endedAtStr sql.NullString

	err := r.db.QueryRow(`
		SELECT solve_id, started_at, ended_at, duration_ms, scramble_text, notes, device_name, device_id, app_version
		FROM solves
		WHERE solve_id = ?
	`, solveID).Scan(
		&s.SolveID, &startedAtStr, &endedAtStr,
		&s.DurationMs, &s.ScrambleText, &s.Notes,
		&s.DeviceName, &s.DeviceID, &s.AppVersion,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get solve: %w", err)
	}

	s.StartedAt, _ = time.Parse(time.RFC3339, startedAtStr)
	if endedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, endedAtStr.String)
		s.EndedAt = &t
	}

	return &s, nil
}

// GetLast retrieves the most recent solve.
func (r *SolveRepository) GetLast() (*Solve, error) {
	var solveID string
	err := r.db.QueryRow(`
		SELECT solve_id FROM solves
		ORDER BY started_at DESC
		LIMIT 1
	`).Scan(&solveID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last solve: %w", err)
	}

	return r.Get(solveID)
}

// List retrieves recent solves.
func (r *SolveRepository) List(limit int) ([]Solve, error) {
	rows, err := r.db.Query(`
		SELECT solve_id, started_at, ended_at, duration_ms, scramble_text, notes, device_name, device_id, app_version
		FROM solves
		ORDER BY started_at DESC
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to list solves: %w", err)
	}
	defer rows.Close()

	var solves []Solve
	for rows.Next() {
		var s Solve
		var startedAtStr string
		var endedAtStr sql.NullString

		err := rows.Scan(
			&s.SolveID, &startedAtStr, &endedAtStr,
			&s.DurationMs, &s.ScrambleText, &s.Notes,
			&s.DeviceName, &s.DeviceID, &s.AppVersion,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan solve: %w", err)
		}

		s.StartedAt, _ = time.Parse(time.RFC3339, startedAtStr)
		if endedAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, endedAtStr.String)
			s.EndedAt = &t
		}

		solves = append(solves, s)
	}

	return solves, nil
}

// Delete deletes a solve and all related data (cascading).
func (r *SolveRepository) Delete(solveID string) error {
	_, err := r.db.Exec("DELETE FROM solves WHERE solve_id = ?", solveID)
	if err != nil {
		return fmt.Errorf("failed to delete solve: %w", err)
	}
	return nil
}

// GetMoveCount returns the number of moves in a solve.
func (r *SolveRepository) GetMoveCount(solveID string) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM moves WHERE solve_id = ?", solveID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get move count: %w", err)
	}
	return count, nil
}
