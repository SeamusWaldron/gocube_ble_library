// Package recorder manages solve recording sessions.
package recorder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AppState represents the persistent application state.
type AppState struct {
	DBPath        string `json:"db_path"`
	ActiveSolveID string `json:"active_solve_id,omitempty"`
	LastDeviceID  string `json:"last_device_id,omitempty"`
	LastDeviceName string `json:"last_device_name,omitempty"`
}

// StateFile manages the application state file.
type StateFile struct {
	path  string
	state AppState
}

// DefaultStatePath returns the default state file path.
func DefaultStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	dir := filepath.Join(home, ".gocube_recorder")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(dir, "state.json"), nil
}

// NewStateFile creates a new state file manager.
func NewStateFile(path string) (*StateFile, error) {
	sf := &StateFile{path: path}

	// Try to load existing state
	if err := sf.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return sf, nil
}

// NewDefaultStateFile creates a state file manager with the default path.
func NewDefaultStateFile() (*StateFile, error) {
	path, err := DefaultStatePath()
	if err != nil {
		return nil, err
	}
	return NewStateFile(path)
}

// Load loads the state from disk.
func (sf *StateFile) Load() error {
	data, err := os.ReadFile(sf.path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &sf.state)
}

// Save saves the state to disk.
func (sf *StateFile) Save() error {
	data, err := json.MarshalIndent(sf.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(sf.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// State returns the current state.
func (sf *StateFile) State() AppState {
	return sf.state
}

// SetDBPath sets the database path.
func (sf *StateFile) SetDBPath(path string) error {
	sf.state.DBPath = path
	return sf.Save()
}

// SetActiveSolve sets the active solve ID.
func (sf *StateFile) SetActiveSolve(solveID string) error {
	sf.state.ActiveSolveID = solveID
	return sf.Save()
}

// ClearActiveSolve clears the active solve ID.
func (sf *StateFile) ClearActiveSolve() error {
	sf.state.ActiveSolveID = ""
	return sf.Save()
}

// SetLastDevice sets the last connected device.
func (sf *StateFile) SetLastDevice(deviceID, deviceName string) error {
	sf.state.LastDeviceID = deviceID
	sf.state.LastDeviceName = deviceName
	return sf.Save()
}

// HasActiveSolve returns true if there is an active solve.
func (sf *StateFile) HasActiveSolve() bool {
	return sf.state.ActiveSolveID != ""
}

// ActiveSolveID returns the active solve ID.
func (sf *StateFile) ActiveSolveID() string {
	return sf.state.ActiveSolveID
}

// LastDeviceID returns the last connected device ID.
func (sf *StateFile) LastDeviceID() string {
	return sf.state.LastDeviceID
}

// DBPath returns the database path.
func (sf *StateFile) DBPath() string {
	return sf.state.DBPath
}
