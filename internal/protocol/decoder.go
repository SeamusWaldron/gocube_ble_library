package protocol

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RotationEvent represents a single face rotation from the cube.
type RotationEvent struct {
	FaceCode          byte   // Raw face+direction code (0x00-0x0B)
	CenterOrientation byte   // Center piece orientation
	Clockwise         bool   // Direction of rotation
	Color             string // Color name (blue, green, white, yellow, red, orange)
}

// BatteryEvent represents a battery level notification.
type BatteryEvent struct {
	Level int // 0-100 percentage
}

// CubeTypeEvent represents a cube type notification.
type CubeTypeEvent struct {
	TypeCode byte
	TypeName string
}

// OrientationEvent represents a cube orientation notification.
type OrientationEvent struct {
	X float64
	Y float64
	Z float64
	W float64

	// Derived discrete orientation
	UpFace    string // Which face is pointing up (U, D, F, B, R, L)
	FrontFace string // Which face is facing the solver
}

// OfflineStatsEvent represents offline statistics.
type OfflineStatsEvent struct {
	Moves  int
	Time   int // seconds
	Solves int
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
// Format: ASCII string "x#y#z#w[checksum]\r\n" where # is the separator.
// The checksum byte and trailing CRLF are stripped before parsing.
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

	// The last part may have a trailing checksum byte and CRLF, extract only the numeric portion
	wStr := parts[3]
	wStr = extractNumeric(wStr)
	w, err := strconv.ParseFloat(wStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid w value: %w", err)
	}

	event := &OrientationEvent{X: x, Y: y, Z: z, W: w}

	// Derive discrete face orientations from quaternion
	event.UpFace, event.FrontFace = quaternionToFaces(x, y, z, w)

	return event, nil
}

// extractNumeric extracts the leading numeric portion (including optional minus sign) from a string.
func extractNumeric(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r == '-' && i == 0 {
			result.WriteRune(r)
		} else if r >= '0' && r <= '9' {
			result.WriteRune(r)
		} else if r == '.' {
			result.WriteRune(r)
		} else {
			break
		}
	}
	return result.String()
}

// quaternionToFaces converts a quaternion to discrete face orientations.
// Returns which cube face is pointing up and which is facing the solver.
func quaternionToFaces(x, y, z, w float64) (upFace, frontFace string) {
	// Normalize the quaternion (GoCube sends raw integer values)
	mag := math.Sqrt(x*x + y*y + z*z + w*w)
	if mag > 0 {
		x /= mag
		y /= mag
		z /= mag
		w /= mag
	}

	// Rotate the up vector (0, 1, 0) by the quaternion
	upX := 2 * (x*y - w*z)
	upY := 1 - 2*(x*x+z*z)
	upZ := 2 * (y*z + w*x)

	// Rotate the front vector (0, 0, 1) by the quaternion
	frontX := 2 * (x*z + w*y)
	frontY := 2 * (y*z - w*x)
	frontZ := 1 - 2*(x*x+y*y)

	upFace = vectorToFace(upX, upY, upZ)
	frontFace = vectorToFace(frontX, frontY, frontZ)

	return upFace, frontFace
}

// vectorToFace determines which cube face a vector points to.
func vectorToFace(x, y, z float64) string {
	absX := math.Abs(x)
	absY := math.Abs(y)
	absZ := math.Abs(z)

	if absY >= absX && absY >= absZ {
		if y > 0 {
			return "U"
		}
		return "D"
	}
	if absZ >= absX && absZ >= absY {
		if z > 0 {
			return "F"
		}
		return "B"
	}
	if x > 0 {
		return "R"
	}
	return "L"
}

// DecodeOfflineStats decodes an offline stats message payload.
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
