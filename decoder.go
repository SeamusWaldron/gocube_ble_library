package gocube

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RotationEvent represents a single face rotation from the cube.
type RotationEvent struct {
	FaceCode          byte   `json:"face_code"`          // Raw face+direction code (0x00-0x0B)
	CenterOrientation byte   `json:"center_orientation"` // Center piece orientation
	Clockwise         bool   `json:"clockwise"`          // Direction of rotation
	Color             string `json:"color"`              // Color name (blue, green, white, yellow, red, orange)
}

// BatteryEvent represents a battery level notification.
type BatteryEvent struct {
	Level int `json:"level"` // 0-100 percentage
}

// CubeTypeEvent represents a cube type notification.
type CubeTypeEvent struct {
	TypeCode byte   `json:"type_code"`
	TypeName string `json:"type_name"`
}

// OrientationEvent represents a cube orientation notification.
type OrientationEvent struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	W float64 `json:"w"`

	// Derived discrete orientation
	UpFace    string `json:"up_face"`    // Which face is pointing up (U, D, F, B, R, L)
	FrontFace string `json:"front_face"` // Which face is facing the solver
}

// OfflineStatsEvent represents offline statistics.
type OfflineStatsEvent struct {
	Moves  int `json:"moves"`
	Time   int `json:"time"`   // seconds
	Solves int `json:"solves"`
}

// Color to face code mapping based on GoCube protocol
var colorNames = map[byte]string{
	0: "blue",
	1: "green",
	2: "white",
	3: "yellow",
	4: "red",
	5: "orange",
}

// DecodeRotation decodes a rotation message payload into rotation events.
// Rotation payloads contain pairs of bytes: [face_dir] [center_orientation]
func DecodeRotation(payload []byte) ([]RotationEvent, error) {
	if len(payload)%2 != 0 {
		return nil, fmt.Errorf("rotation payload must have even length, got %d", len(payload))
	}

	var events []RotationEvent
	for i := 0; i < len(payload); i += 2 {
		faceCode := payload[i]
		centerOrient := payload[i+1]

		// Face codes: 0x00-0x0B
		// Even codes (0,2,4,6,8,10) = clockwise
		// Odd codes (1,3,5,7,9,11) = counter-clockwise
		clockwise := faceCode%2 == 0
		colorIdx := faceCode / 2

		colorName, ok := colorNames[colorIdx]
		if !ok {
			return nil, fmt.Errorf("unknown color index %d from face code 0x%02X", colorIdx, faceCode)
		}

		events = append(events, RotationEvent{
			FaceCode:          faceCode,
			CenterOrientation: centerOrient,
			Clockwise:         clockwise,
			Color:             colorName,
		})
	}

	return events, nil
}

// DecodeBattery decodes a battery message payload.
func DecodeBattery(payload []byte) (*BatteryEvent, error) {
	if len(payload) < 1 {
		return nil, fmt.Errorf("battery payload too short")
	}
	return &BatteryEvent{
		Level: int(payload[0]),
	}, nil
}

// DecodeCubeType decodes a cube type message payload.
func DecodeCubeType(payload []byte) (*CubeTypeEvent, error) {
	if len(payload) < 1 {
		return nil, fmt.Errorf("cube type payload too short")
	}

	typeName := "standard"
	if payload[0] == 0x01 {
		typeName = "edge"
	}

	return &CubeTypeEvent{
		TypeCode: payload[0],
		TypeName: typeName,
	}, nil
}

// DecodeOrientation decodes an orientation message payload.
// Format: ASCII string "x#y#z#w" where # is the separator.
func DecodeOrientation(payload []byte) (*OrientationEvent, error) {
	str := string(payload)
	parts := strings.Split(str, "#")
	if len(parts) != 4 {
		return nil, fmt.Errorf("orientation payload must have 4 parts, got %d", len(parts))
	}

	x, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid x value: %w", err)
	}
	y, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid y value: %w", err)
	}
	z, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid z value: %w", err)
	}
	w, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid w value: %w", err)
	}

	event := &OrientationEvent{X: x, Y: y, Z: z, W: w}

	// Derive discrete face orientations from quaternion
	event.UpFace, event.FrontFace = quaternionToFaces(x, y, z, w)

	return event, nil
}

// quaternionToFaces converts a quaternion to discrete face orientations.
// Returns which cube face is pointing up and which is facing the solver.
//
// Reference orientation (identity quaternion):
// - White (U) faces up (+Y)
// - Green (F) faces front (+Z)
// - Red (R) faces right (+X)
func quaternionToFaces(x, y, z, w float64) (upFace, frontFace string) {
	// Rotate the up vector (0, 1, 0) by the quaternion
	// Using quaternion rotation formula: v' = q * v * q^(-1)
	upX := 2 * (x*y - w*z)
	upY := 1 - 2*(x*x+z*z)
	upZ := 2 * (y*z + w*x)

	// Rotate the front vector (0, 0, 1) by the quaternion
	frontX := 2 * (x*z + w*y)
	frontY := 2 * (y*z - w*x)
	frontZ := 1 - 2*(x*x+y*y)

	// Determine which face is up (closest to +Y in world space)
	upFace = vectorToFace(upX, upY, upZ)

	// Determine which face is front (closest to +Z in world space)
	frontFace = vectorToFace(frontX, frontY, frontZ)

	return upFace, frontFace
}

// vectorToFace determines which cube face a vector points to.
// Maps the vector to the closest axis direction.
func vectorToFace(x, y, z float64) string {
	// Find the dominant axis
	absX := math.Abs(x)
	absY := math.Abs(y)
	absZ := math.Abs(z)

	if absY >= absX && absY >= absZ {
		// Y-axis dominant
		if y > 0 {
			return "U" // Up (White)
		}
		return "D" // Down (Yellow)
	}
	if absZ >= absX && absZ >= absY {
		// Z-axis dominant
		if z > 0 {
			return "F" // Front (Green)
		}
		return "B" // Back (Blue)
	}
	// X-axis dominant
	if x > 0 {
		return "R" // Right (Red)
	}
	return "L" // Left (Orange)
}

// DecodeOfflineStats decodes an offline stats message payload.
// Format: ASCII string "moves#time#solves"
func DecodeOfflineStats(payload []byte) (*OfflineStatsEvent, error) {
	str := string(payload)
	parts := strings.Split(str, "#")
	if len(parts) != 3 {
		return nil, fmt.Errorf("offline stats payload must have 3 parts, got %d", len(parts))
	}

	moves, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid moves value: %w", err)
	}
	time, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid time value: %w", err)
	}
	solves, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid solves value: %w", err)
	}

	return &OfflineStatsEvent{
		Moves:  moves,
		Time:   time,
		Solves: solves,
	}, nil
}

// DecodeMessage decodes a message and returns the event type and JSON payload.
func DecodeMessage(msg *Message) (eventType string, payloadJSON string, err error) {
	eventType = MessageTypeName(msg.Type)

	var payload interface{}

	switch msg.Type {
	case MsgTypeRotation:
		events, err := DecodeRotation(msg.Payload)
		if err != nil {
			return eventType, "", err
		}
		payload = events

	case MsgTypeBattery:
		event, err := DecodeBattery(msg.Payload)
		if err != nil {
			return eventType, "", err
		}
		payload = event

	case MsgTypeCubeType:
		event, err := DecodeCubeType(msg.Payload)
		if err != nil {
			return eventType, "", err
		}
		payload = event

	case MsgTypeOrientation:
		event, err := DecodeOrientation(msg.Payload)
		if err != nil {
			return eventType, "", err
		}
		payload = event

	case MsgTypeOfflineStats:
		event, err := DecodeOfflineStats(msg.Payload)
		if err != nil {
			return eventType, "", err
		}
		payload = event

	default:
		// Unknown message type - store raw payload
		payload = map[string]interface{}{
			"raw_hex": fmt.Sprintf("%X", msg.Payload),
		}
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return eventType, "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	return eventType, string(jsonBytes), nil
}
