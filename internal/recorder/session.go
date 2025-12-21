package recorder

import (
	"fmt"
	"sync"
	"time"

	"github.com/SeamusWaldron/gocube"
	"github.com/SeamusWaldron/gocube/internal/storage"
)

// SessionState represents the current state of a recording session.
type SessionState int

const (
	StateIdle SessionState = iota
	StateRecording
	StateEnded
)

// String returns the string representation of the session state.
func (s SessionState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRecording:
		return "recording"
	case StateEnded:
		return "ended"
	default:
		return "unknown"
	}
}

// Session manages a solve recording session.
type Session struct {
	db        *storage.DB
	stateFile *StateFile

	mu        sync.RWMutex
	state     SessionState
	solveID   string
	startTime time.Time
	moveIndex int

	// Current orientation state (tracked to detect changes)
	lastUpFace    string
	lastFrontFace string

	// Repositories
	solveRepo       *storage.SolveRepository
	eventRepo       *storage.EventRepository
	moveRepo        *storage.MoveRepository
	phaseRepo       *storage.PhaseRepository
	orientationRepo *storage.OrientationRepository

	// Callbacks
	onMove        func(gocube.Move)
	onPhase       func(string)
	onOrientation func(upFace, frontFace string)
}

// NewSession creates a new session manager.
func NewSession(db *storage.DB, stateFile *StateFile) *Session {
	return &Session{
		db:              db,
		stateFile:       stateFile,
		state:           StateIdle,
		solveRepo:       storage.NewSolveRepository(db),
		eventRepo:       storage.NewEventRepository(db),
		moveRepo:        storage.NewMoveRepository(db),
		phaseRepo:       storage.NewPhaseRepository(db),
		orientationRepo: storage.NewOrientationRepository(db),
	}
}

// SetMoveCallback sets the callback for new moves.
func (s *Session) SetMoveCallback(cb func(gocube.Move)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onMove = cb
}

// SetPhaseCallback sets the callback for phase changes.
func (s *Session) SetPhaseCallback(cb func(string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onPhase = cb
}

// SetOrientationCallback sets the callback for orientation changes.
func (s *Session) SetOrientationCallback(cb func(upFace, frontFace string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onOrientation = cb
}

// CurrentOrientation returns the current orientation (up_face, front_face).
func (s *Session) CurrentOrientation() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpFace, s.lastFrontFace
}

// State returns the current session state.
func (s *Session) State() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// SolveID returns the current solve ID.
func (s *Session) SolveID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.solveID
}

// ElapsedMs returns the elapsed time since solve start in milliseconds.
func (s *Session) ElapsedMs() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state != StateRecording {
		return 0
	}
	return time.Since(s.startTime).Milliseconds()
}

// CurrentTimestamp returns the current timestamp relative to solve start (thread-safe).
func (s *Session) CurrentTimestamp() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.startTime).Milliseconds()
}

// MoveCount returns the current move count.
func (s *Session) MoveCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.moveIndex
}

// Start starts a new solve recording session.
func (s *Session) Start(notes, scramble, deviceName, deviceID, appVersion string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateRecording {
		return "", fmt.Errorf("solve already in progress")
	}

	solveID, err := s.solveRepo.Create(notes, scramble, deviceName, deviceID, appVersion)
	if err != nil {
		return "", fmt.Errorf("failed to create solve: %w", err)
	}

	s.solveID = solveID
	s.startTime = time.Now()
	s.moveIndex = 0
	s.lastUpFace = ""
	s.lastFrontFace = ""
	s.state = StateRecording

	// Update state file
	if s.stateFile != nil {
		if err := s.stateFile.SetActiveSolve(solveID); err != nil {
			// Log error but don't fail
		}
	}

	return solveID, nil
}

// End ends the current solve recording session.
func (s *Session) End() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateRecording {
		return fmt.Errorf("no solve in progress")
	}

	if err := s.solveRepo.End(s.solveID); err != nil {
		return fmt.Errorf("failed to end solve: %w", err)
	}

	s.state = StateEnded

	// Clear state file
	if s.stateFile != nil {
		if err := s.stateFile.ClearActiveSolve(); err != nil {
			// Log error but don't fail
		}
	}

	// Compute derived phase segments
	if err := s.computePhaseSegments(); err != nil {
		// Log error but don't fail
	}

	return nil
}

// MarkPhase marks a phase transition.
func (s *Session) MarkPhase(phaseKey string, notes *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateRecording {
		return fmt.Errorf("no solve in progress")
	}

	tsMs := time.Since(s.startTime).Milliseconds()

	_, err := s.phaseRepo.CreatePhaseMark(s.solveID, tsMs, phaseKey, notes)
	if err != nil {
		return fmt.Errorf("failed to mark phase: %w", err)
	}

	// Notify callback
	if s.onPhase != nil {
		go s.onPhase(phaseKey)
	}

	return nil
}

// MarkPhaseAt marks a phase transition at a specific timestamp.
// Use this when you need precise control over the timestamp (e.g., ensuring
// a phase mark comes before a move recorded in the same millisecond window).
func (s *Session) MarkPhaseAt(phaseKey string, tsMs int64, notes *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateRecording {
		return fmt.Errorf("no solve in progress")
	}

	_, err := s.phaseRepo.CreatePhaseMark(s.solveID, tsMs, phaseKey, notes)
	if err != nil {
		return fmt.Errorf("failed to mark phase: %w", err)
	}

	// Notify callback
	if s.onPhase != nil {
		go s.onPhase(phaseKey)
	}

	return nil
}

// HandleMessage processes an incoming BLE message.
func (s *Session) HandleMessage(msg *gocube.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateRecording {
		return nil // Not recording, ignore
	}

	tsMs := time.Since(s.startTime).Milliseconds()

	// Decode and store event
	eventType, payloadJSON, err := gocube.DecodeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	rawBase64 := msg.RawBase64
	eventID, err := s.eventRepo.Create(s.solveID, tsMs, eventType, payloadJSON, &rawBase64)
	if err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	// Process rotation events into moves
	if msg.Type == gocube.MsgTypeRotation {
		rotations, err := gocube.DecodeRotation(msg.Payload)
		if err != nil {
			return fmt.Errorf("failed to decode rotations: %w", err)
		}

		moves := gocube.RotationsToMoves(rotations, tsMs)

		for _, move := range moves {
			_, err := s.moveRepo.Create(s.solveID, s.moveIndex, tsMs, move, &eventID)
			if err != nil {
				return fmt.Errorf("failed to store move: %w", err)
			}
			s.moveIndex++

			// Notify callback
			if s.onMove != nil {
				go s.onMove(move)
			}
		}
	}

	// Process orientation events
	if msg.Type == gocube.MsgTypeOrientation {
		orient, err := gocube.DecodeOrientation(msg.Payload)
		if err != nil {
			return fmt.Errorf("failed to decode orientation: %w", err)
		}

		// Check if orientation has changed
		if orient.UpFace != s.lastUpFace || orient.FrontFace != s.lastFrontFace {
			// Record orientation change
			_, err := s.orientationRepo.Create(s.solveID, tsMs, orient.UpFace, orient.FrontFace, &eventID)
			if err != nil {
				return fmt.Errorf("failed to store orientation: %w", err)
			}

			s.lastUpFace = orient.UpFace
			s.lastFrontFace = orient.FrontFace

			// Notify callback
			if s.onOrientation != nil {
				go s.onOrientation(orient.UpFace, orient.FrontFace)
			}
		}
	}

	return nil
}

// computePhaseSegments computes derived phase segments after solve ends.
func (s *Session) computePhaseSegments() error {
	// Get phase marks
	marks, err := s.phaseRepo.GetPhaseMarks(s.solveID)
	if err != nil {
		return err
	}

	if len(marks) == 0 {
		return nil
	}

	// Get solve end time
	solve, err := s.solveRepo.Get(s.solveID)
	if err != nil {
		return err
	}
	if solve == nil || solve.DurationMs == nil {
		return nil
	}

	endTsMs := *solve.DurationMs

	// Compute segments
	for i, mark := range marks {
		var segmentEndMs int64
		isLastSegment := i >= len(marks)-1
		if !isLastSegment {
			segmentEndMs = marks[i+1].TsMs
		} else {
			// For the last segment, add 1ms to ensure all remaining moves are included
			// (since GetBySolveRange uses exclusive end bound)
			segmentEndMs = endTsMs + 1
		}

		durationMs := segmentEndMs - mark.TsMs
		if isLastSegment {
			// Use actual duration for last segment (not +1)
			durationMs = endTsMs - mark.TsMs
		}
		if durationMs <= 0 {
			continue
		}

		// Get moves in this segment
		moveRecords, err := s.moveRepo.GetBySolveRange(s.solveID, mark.TsMs, segmentEndMs)
		if err != nil {
			continue
		}

		moveCount := len(moveRecords)
		tps := 0.0
		if durationMs > 0 {
			tps = float64(moveCount) / (float64(durationMs) / 1000.0)
		}

		// Store the actual end timestamp (not the query end which may be +1)
		storedEndMs := segmentEndMs
		if isLastSegment {
			storedEndMs = endTsMs // Use actual end, not +1
		}

		segment := storage.PhaseSegment{
			SolveID:    s.solveID,
			PhaseKey:   mark.PhaseKey,
			StartTsMs:  mark.TsMs,
			EndTsMs:    storedEndMs,
			DurationMs: durationMs,
			MoveCount:  moveCount,
			TPS:        tps,
		}

		if _, err := s.phaseRepo.CreatePhaseSegment(segment); err != nil {
			// Log but continue
		}
	}

	return nil
}

// Resume attempts to resume an interrupted solve.
func (s *Session) Resume(solveID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify solve exists and is not ended
	solve, err := s.solveRepo.Get(solveID)
	if err != nil {
		return fmt.Errorf("failed to get solve: %w", err)
	}
	if solve == nil {
		return fmt.Errorf("solve not found: %s", solveID)
	}
	if solve.EndedAt != nil {
		return fmt.Errorf("solve already ended")
	}

	// Get next move index
	nextIndex, err := s.moveRepo.GetNextIndex(solveID)
	if err != nil {
		return fmt.Errorf("failed to get next move index: %w", err)
	}

	s.solveID = solveID
	s.startTime = solve.StartedAt
	s.moveIndex = nextIndex
	s.state = StateRecording

	// Restore last orientation state
	lastOrient, err := s.orientationRepo.GetLast(solveID)
	if err == nil && lastOrient != nil {
		s.lastUpFace = lastOrient.UpFace
		s.lastFrontFace = lastOrient.FrontFace
	} else {
		s.lastUpFace = ""
		s.lastFrontFace = ""
	}

	return nil
}
