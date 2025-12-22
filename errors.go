package gocube

import "errors"

// Sentinel errors for the gocube package.
var (
	// Connection errors
	ErrNotConnected     = errors.New("gocube: not connected to device")
	ErrAlreadyConnected = errors.New("gocube: already connected")
	ErrDeviceNotFound   = errors.New("gocube: device not found")
	ErrConnectionFailed = errors.New("gocube: connection failed")
	ErrTimeout          = errors.New("gocube: operation timed out")

	// Parsing errors
	ErrInvalidNotation = errors.New("gocube: invalid move notation")

	// State errors
	ErrCubeNotReady = errors.New("gocube: cube not ready")
)
