package storage

import (
	"fmt"
)

// Event represents a raw BLE event in the database.
type Event struct {
	EventID         int64
	SolveID         string
	TsMs            int64
	EventType       string
	PayloadJSON     string
	RawPayloadBase64 *string
}

// EventRepository provides CRUD operations for events.
type EventRepository struct {
	db *DB
}

// NewEventRepository creates a new event repository.
func NewEventRepository(db *DB) *EventRepository {
	return &EventRepository{db: db}
}

// Create creates a new event and returns its ID.
func (r *EventRepository) Create(solveID string, tsMs int64, eventType, payloadJSON string, rawBase64 *string) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO events (solve_id, ts_ms, event_type, payload_json, raw_payload_base64)
		VALUES (?, ?, ?, ?, ?)
	`, solveID, tsMs, eventType, payloadJSON, rawBase64)

	if err != nil {
		return 0, fmt.Errorf("failed to create event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get event ID: %w", err)
	}

	return id, nil
}

// GetBySolve retrieves all events for a solve.
func (r *EventRepository) GetBySolve(solveID string) ([]Event, error) {
	rows, err := r.db.Query(`
		SELECT event_id, solve_id, ts_ms, event_type, payload_json, raw_payload_base64
		FROM events
		WHERE solve_id = ?
		ORDER BY ts_ms
	`, solveID)

	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		err := rows.Scan(&e.EventID, &e.SolveID, &e.TsMs, &e.EventType, &e.PayloadJSON, &e.RawPayloadBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, e)
	}

	return events, nil
}

// GetByType retrieves all events of a specific type for a solve.
func (r *EventRepository) GetByType(solveID, eventType string) ([]Event, error) {
	rows, err := r.db.Query(`
		SELECT event_id, solve_id, ts_ms, event_type, payload_json, raw_payload_base64
		FROM events
		WHERE solve_id = ? AND event_type = ?
		ORDER BY ts_ms
	`, solveID, eventType)

	if err != nil {
		return nil, fmt.Errorf("failed to get events by type: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		err := rows.Scan(&e.EventID, &e.SolveID, &e.TsMs, &e.EventType, &e.PayloadJSON, &e.RawPayloadBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, e)
	}

	return events, nil
}

// Count returns the number of events for a solve.
func (r *EventRepository) Count(solveID string) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM events WHERE solve_id = ?", solveID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}
	return count, nil
}
