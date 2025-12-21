package storage

import (
	"fmt"
)

// PhaseDef represents a phase definition.
type PhaseDef struct {
	PhaseKey    string
	DisplayName string
	OrderIndex  int
	Description *string
	IsActive    bool
}

// PhaseMark represents a phase mark during a solve.
type PhaseMark struct {
	PhaseMarkID int64
	SolveID     string
	TsMs        int64
	PhaseKey    string
	MarkType    string
	Notes       *string
}

// PhaseSegment represents a derived phase segment.
type PhaseSegment struct {
	SegmentID  int64
	SolveID    string
	PhaseKey   string
	StartTsMs  int64
	EndTsMs    int64
	DurationMs int64
	MoveCount  int
	TPS        float64
}

// PhaseRepository provides CRUD operations for phases.
type PhaseRepository struct {
	db *DB
}

// NewPhaseRepository creates a new phase repository.
func NewPhaseRepository(db *DB) *PhaseRepository {
	return &PhaseRepository{db: db}
}

// GetAllPhaseDefs retrieves all active phase definitions in order.
func (r *PhaseRepository) GetAllPhaseDefs() ([]PhaseDef, error) {
	rows, err := r.db.Query(`
		SELECT phase_key, display_name, order_index, description, is_active
		FROM phase_defs
		WHERE is_active = 1
		ORDER BY order_index
	`)

	if err != nil {
		return nil, fmt.Errorf("failed to get phase defs: %w", err)
	}
	defer rows.Close()

	var defs []PhaseDef
	for rows.Next() {
		var d PhaseDef
		var isActive int
		err := rows.Scan(&d.PhaseKey, &d.DisplayName, &d.OrderIndex, &d.Description, &isActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan phase def: %w", err)
		}
		d.IsActive = isActive == 1
		defs = append(defs, d)
	}

	return defs, nil
}

// GetPhaseDef retrieves a specific phase definition.
func (r *PhaseRepository) GetPhaseDef(phaseKey string) (*PhaseDef, error) {
	var d PhaseDef
	var isActive int
	err := r.db.QueryRow(`
		SELECT phase_key, display_name, order_index, description, is_active
		FROM phase_defs
		WHERE phase_key = ?
	`, phaseKey).Scan(&d.PhaseKey, &d.DisplayName, &d.OrderIndex, &d.Description, &isActive)

	if err != nil {
		return nil, fmt.Errorf("failed to get phase def: %w", err)
	}
	d.IsActive = isActive == 1

	return &d, nil
}

// CreatePhaseMark creates a new phase mark.
func (r *PhaseRepository) CreatePhaseMark(solveID string, tsMs int64, phaseKey string, notes *string) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO phase_marks (solve_id, ts_ms, phase_key, mark_type, notes)
		VALUES (?, ?, ?, 'start', ?)
	`, solveID, tsMs, phaseKey, notes)

	if err != nil {
		return 0, fmt.Errorf("failed to create phase mark: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get phase mark ID: %w", err)
	}

	return id, nil
}

// GetPhaseMarks retrieves all phase marks for a solve.
func (r *PhaseRepository) GetPhaseMarks(solveID string) ([]PhaseMark, error) {
	rows, err := r.db.Query(`
		SELECT phase_mark_id, solve_id, ts_ms, phase_key, mark_type, notes
		FROM phase_marks
		WHERE solve_id = ?
		ORDER BY ts_ms
	`, solveID)

	if err != nil {
		return nil, fmt.Errorf("failed to get phase marks: %w", err)
	}
	defer rows.Close()

	var marks []PhaseMark
	for rows.Next() {
		var m PhaseMark
		err := rows.Scan(&m.PhaseMarkID, &m.SolveID, &m.TsMs, &m.PhaseKey, &m.MarkType, &m.Notes)
		if err != nil {
			return nil, fmt.Errorf("failed to scan phase mark: %w", err)
		}
		marks = append(marks, m)
	}

	return marks, nil
}

// CreatePhaseSegment creates a derived phase segment.
func (r *PhaseRepository) CreatePhaseSegment(segment PhaseSegment) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO derived_phase_segments (solve_id, phase_key, start_ts_ms, end_ts_ms, duration_ms, move_count, tps)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, segment.SolveID, segment.PhaseKey, segment.StartTsMs, segment.EndTsMs, segment.DurationMs, segment.MoveCount, segment.TPS)

	if err != nil {
		return 0, fmt.Errorf("failed to create phase segment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get segment ID: %w", err)
	}

	return id, nil
}

// GetPhaseSegments retrieves all phase segments for a solve.
func (r *PhaseRepository) GetPhaseSegments(solveID string) ([]PhaseSegment, error) {
	rows, err := r.db.Query(`
		SELECT segment_id, solve_id, phase_key, start_ts_ms, end_ts_ms, duration_ms, move_count, tps
		FROM derived_phase_segments
		WHERE solve_id = ?
		ORDER BY start_ts_ms
	`, solveID)

	if err != nil {
		return nil, fmt.Errorf("failed to get phase segments: %w", err)
	}
	defer rows.Close()

	var segments []PhaseSegment
	for rows.Next() {
		var s PhaseSegment
		err := rows.Scan(&s.SegmentID, &s.SolveID, &s.PhaseKey, &s.StartTsMs, &s.EndTsMs, &s.DurationMs, &s.MoveCount, &s.TPS)
		if err != nil {
			return nil, fmt.Errorf("failed to scan segment: %w", err)
		}
		segments = append(segments, s)
	}

	return segments, nil
}

// DeletePhaseSegments deletes all phase segments for a solve.
func (r *PhaseRepository) DeletePhaseSegments(solveID string) error {
	_, err := r.db.Exec("DELETE FROM derived_phase_segments WHERE solve_id = ?", solveID)
	if err != nil {
		return fmt.Errorf("failed to delete phase segments: %w", err)
	}
	return nil
}

// PhaseKeyToNumber returns the phase number (0-7) for keyboard shortcuts.
func PhaseKeyToNumber(phaseKey string) int {
	switch phaseKey {
	case "inspection":
		return 0
	case "white_cross":
		return 1
	case "top_corners":
		return 2
	case "middle_layer":
		return 3
	case "bottom_cross":
		return 4
	case "position_corners":
		return 5
	case "rotate_corners":
		return 6
	case "complete":
		return 7
	default:
		return -1
	}
}

// NumberToPhaseKey returns the phase key for a keyboard number (0-7).
func NumberToPhaseKey(num int) string {
	switch num {
	case 0:
		return "inspection"
	case 1:
		return "white_cross"
	case 2:
		return "top_corners"
	case 3:
		return "middle_layer"
	case 4:
		return "bottom_cross"
	case 5:
		return "position_corners"
	case 6:
		return "rotate_corners"
	case 7:
		return "complete"
	default:
		return ""
	}
}

// AlgoKeyToPhaseKey returns the phase key for algorithm markers (r/l).
func AlgoKeyToPhaseKey(key string) string {
	switch key {
	case "r":
		return "middle_rhs"
	case "l":
		return "middle_lhs"
	default:
		return ""
	}
}

// PhaseDisplayName returns a short display name for a phase key.
func PhaseDisplayName(phaseKey string) string {
	switch phaseKey {
	case "scramble":
		return "Scramble"
	case "inspection":
		return "Inspection"
	case "white_cross":
		return "White Cross"
	case "top_corners":
		return "Top Corners"
	case "middle_layer":
		return "Middle Layer"
	case "middle_rhs":
		return "Middle RHS"
	case "middle_lhs":
		return "Middle LHS"
	case "bottom_cross":
		return "Bottom Cross"
	case "position_corners":
		return "Pos Corners"
	case "rotate_corners":
		return "Rot Corners"
	case "complete":
		return "Complete"
	default:
		return phaseKey
	}
}
