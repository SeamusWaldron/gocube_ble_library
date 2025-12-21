package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/seamusw/gocube/internal/gocube"
)

// LogEventType identifies the type of logged event
type LogEventType string

const (
	LogEventBLEMessage LogEventType = "ble_message"
	LogEventKeyPress   LogEventType = "key_press"
	LogEventPhase      LogEventType = "phase_change"
)

// LogEvent represents a single logged event
type LogEvent struct {
	Timestamp   time.Time       `json:"timestamp"`
	ElapsedMs   int64           `json:"elapsed_ms"`
	EventType   LogEventType    `json:"event_type"`
	KeyPress    string          `json:"key_press,omitempty"`
	BLEType     byte            `json:"ble_type,omitempty"`
	BLEPayload  []byte          `json:"ble_payload,omitempty"`
	Phase       string          `json:"phase,omitempty"`
	Description string          `json:"description,omitempty"`
}

// SolveLog represents a complete solve session log
type SolveLog struct {
	Version     string     `json:"version"`
	CreatedAt   time.Time  `json:"created_at"`
	DeviceName  string     `json:"device_name,omitempty"`
	SolveID     string     `json:"solve_id,omitempty"`
	Events      []LogEvent `json:"events"`
}

// SolveLogger handles logging events during a solve
type SolveLogger struct {
	log       *SolveLog
	startTime time.Time
	file      *os.File
	enabled   bool
}

// NewSolveLogger creates a new logger
func NewSolveLogger() *SolveLogger {
	return &SolveLogger{
		enabled: false,
	}
}

// Start begins logging to a file
func (l *SolveLogger) Start(logDir string) error {
	// Create log directory if needed
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp
	filename := fmt.Sprintf("solve_%s.jsonl", time.Now().Format("20060102_150405"))
	path := filepath.Join(logDir, filename)

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	l.file = file
	l.startTime = time.Now()
	l.enabled = true
	l.log = &SolveLog{
		Version:   "1.0",
		CreatedAt: l.startTime,
		Events:    make([]LogEvent, 0),
	}

	// Write header
	header := map[string]interface{}{
		"version":    "1.0",
		"created_at": l.startTime,
		"type":       "header",
	}
	if err := l.writeJSON(header); err != nil {
		return err
	}

	return nil
}

// SetDeviceInfo sets device information
func (l *SolveLogger) SetDeviceInfo(name, solveID string) {
	if l.log != nil {
		l.log.DeviceName = name
		l.log.SolveID = solveID
	}
}

// LogBLEMessage logs a BLE message
func (l *SolveLogger) LogBLEMessage(msg *gocube.Message, description string) {
	if !l.enabled || l.file == nil {
		return
	}

	event := LogEvent{
		Timestamp:   time.Now(),
		ElapsedMs:   time.Since(l.startTime).Milliseconds(),
		EventType:   LogEventBLEMessage,
		BLEType:     msg.Type,
		BLEPayload:  msg.Payload,
		Description: description,
	}

	l.log.Events = append(l.log.Events, event)
	l.writeJSON(event)
}

// LogKeyPress logs a key press
func (l *SolveLogger) LogKeyPress(key string) {
	if !l.enabled || l.file == nil {
		return
	}

	event := LogEvent{
		Timestamp: time.Now(),
		ElapsedMs: time.Since(l.startTime).Milliseconds(),
		EventType: LogEventKeyPress,
		KeyPress:  key,
	}

	l.log.Events = append(l.log.Events, event)
	l.writeJSON(event)
}

// LogPhaseChange logs a phase change
func (l *SolveLogger) LogPhaseChange(phase string) {
	if !l.enabled || l.file == nil {
		return
	}

	event := LogEvent{
		Timestamp: time.Now(),
		ElapsedMs: time.Since(l.startTime).Milliseconds(),
		EventType: LogEventPhase,
		Phase:     phase,
	}

	l.log.Events = append(l.log.Events, event)
	l.writeJSON(event)
}

func (l *SolveLogger) writeJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = l.file.Write(append(data, '\n'))
	return err
}

// Close closes the log file
func (l *SolveLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// FilePath returns the current log file path
func (l *SolveLogger) FilePath() string {
	if l.file != nil {
		return l.file.Name()
	}
	return ""
}

// LoadSolveLog loads a solve log from a JSONL file
func LoadSolveLog(path string) (*SolveLog, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	log := &SolveLog{
		Events: make([]LogEvent, 0),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		// First line is the header
		if lineNum == 1 {
			var header map[string]interface{}
			if err := json.Unmarshal(line, &header); err != nil {
				return nil, fmt.Errorf("failed to parse header: %w", err)
			}
			if v, ok := header["version"].(string); ok {
				log.Version = v
			}
			if v, ok := header["created_at"].(string); ok {
				if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
					log.CreatedAt = t
				}
			}
			continue
		}

		// Parse event
		var event LogEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, fmt.Errorf("failed to parse event at line %d: %w", lineNum, err)
		}
		log.Events = append(log.Events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	return log, nil
}
