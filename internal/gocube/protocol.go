// Package gocube implements the GoCube BLE protocol decoding.
package gocube

import (
	"encoding/base64"
	"errors"
	"fmt"
)

// GoCube BLE Service and Characteristic UUIDs
const (
	ServiceUUID = "6e400001-b5a3-f393-e0a9-e50e24dcca9e"
	TxCharUUID  = "6e400003-b5a3-f393-e0a9-e50e24dcca9e" // Notify
	RxCharUUID  = "6e400002-b5a3-f393-e0a9-e50e24dcca9e" // Write
)

// Message type constants
const (
	MsgTypeRotation    byte = 0x01
	MsgTypeState       byte = 0x02
	MsgTypeOrientation byte = 0x03
	MsgTypeBattery     byte = 0x05
	MsgTypeOfflineStats byte = 0x07
	MsgTypeCubeType    byte = 0x08
)

// Command codes for writing to RX characteristic
const (
	CmdRequestBattery     byte = 0x32
	CmdRequestState       byte = 0x33
	CmdReboot             byte = 0x34
	CmdResetSolved        byte = 0x35
	CmdDisableOrientation byte = 0x37
	CmdEnableOrientation  byte = 0x38
	CmdRequestOfflineStats byte = 0x39
	CmdFlashBacklight     byte = 0x41
	CmdToggleAnimatedBL   byte = 0x42
	CmdSlowFlashBacklight byte = 0x43
	CmdToggleBacklight    byte = 0x44
	CmdRequestCubeType    byte = 0x56
	CmdCalibrateOrientation byte = 0x57
)

// Message frame constants
const (
	FramePrefix  byte = 0x2A // '*'
	FrameSuffix1 byte = 0x0D // CR
	FrameSuffix2 byte = 0x0A // LF
)

// Errors
var (
	ErrInvalidPrefix   = errors.New("invalid message prefix")
	ErrInvalidSuffix   = errors.New("invalid message suffix")
	ErrInvalidChecksum = errors.New("invalid checksum")
	ErrMessageTooShort = errors.New("message too short")
	ErrInvalidLength   = errors.New("invalid message length")
)

// Message represents a parsed GoCube BLE message.
type Message struct {
	Type       byte   // Message type identifier
	Payload    []byte // Decoded payload (without frame overhead)
	RawBase64  string // Base64 encoded raw bytes for storage
}

// ParseMessage parses a raw BLE notification into a Message.
// Frame format: [0x2A] [length] [type] [payload...] [checksum] [0x0D 0x0A]
// The length byte indicates bytes from position 2 to end (type + payload + checksum + suffix)
func ParseMessage(data []byte) (*Message, error) {
	if len(data) < 5 {
		return nil, ErrMessageTooShort
	}

	// Check prefix
	if data[0] != FramePrefix {
		return nil, ErrInvalidPrefix
	}

	// Length field = bytes from position 2 to end (type + payload + checksum + suffix)
	length := int(data[1])

	// Total message length = prefix(1) + length_byte(1) + length
	expectedLen := 2 + length
	if len(data) < expectedLen {
		return nil, fmt.Errorf("%w: expected %d, got %d", ErrInvalidLength, expectedLen, len(data))
	}

	// Checksum is at position (length - 1) from start, suffix follows
	checksumIdx := length - 1
	if checksumIdx < 2 {
		return nil, ErrMessageTooShort
	}

	// Check suffix
	if data[checksumIdx+1] != FrameSuffix1 || data[checksumIdx+2] != FrameSuffix2 {
		return nil, ErrInvalidSuffix
	}

	// Validate checksum: sum of bytes 0 through checksumIdx-1
	var checksum byte
	for i := 0; i < checksumIdx; i++ {
		checksum += data[i]
	}
	if checksum != data[checksumIdx] {
		return nil, fmt.Errorf("%w: expected 0x%02X, got 0x%02X", ErrInvalidChecksum, data[checksumIdx], checksum)
	}

	msgType := data[2]
	payload := data[3:checksumIdx]

	return &Message{
		Type:      msgType,
		Payload:   payload,
		RawBase64: base64.StdEncoding.EncodeToString(data[:expectedLen]),
	}, nil
}

// BuildCommand creates a command message to send to the cube.
func BuildCommand(cmdCode byte) []byte {
	// Simple commands with no payload
	// Format: [0x2A] [0x01] [cmd] [checksum] [0x0D] [0x0A]
	length := byte(0x01)
	checksum := FramePrefix + length + cmdCode

	return []byte{FramePrefix, length, cmdCode, checksum, FrameSuffix1, FrameSuffix2}
}

// MessageTypeName returns a human-readable name for the message type.
func MessageTypeName(msgType byte) string {
	switch msgType {
	case MsgTypeRotation:
		return "rotation"
	case MsgTypeState:
		return "state"
	case MsgTypeOrientation:
		return "orientation"
	case MsgTypeBattery:
		return "battery"
	case MsgTypeOfflineStats:
		return "offline_stats"
	case MsgTypeCubeType:
		return "cube_type"
	default:
		return fmt.Sprintf("unknown_0x%02X", msgType)
	}
}
