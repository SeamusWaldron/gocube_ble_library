package gocube

import (
	"context"
	"sync"
	"time"

	"github.com/SeamusWaldron/gocube_ble_library/internal/ble"
	"github.com/SeamusWaldron/gocube_ble_library/internal/protocol"
)

// Device represents a discovered GoCube device.
// Devices are returned by the Scan function and can be passed to Connect.
type Device struct {
	Name    string      // Device name (e.g., "GoCube_XXXX")
	UUID    string      // Device UUID for connection
	RSSI    int16       // Signal strength in dBm (higher = stronger, typical range -30 to -90)
	address interface{} // Internal: platform-specific address
}

// GoCube represents a connected GoCube smart cube.
// It wraps the BLE connection and provides a clean callback-based API.
//
// Create a GoCube using Connect or ConnectFirst:
//
//	cube, err := gocube.ConnectFirst(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cube.Close()
//
//	cube.OnMove(func(m gocube.Move) {
//	    fmt.Println("Move:", m.Notation())
//	})
//
// GoCube maintains an internal Cube state that tracks the current cube state.
// Access it with the Cube() method.
type GoCube struct {
	client *ble.Client
	cube   *Cube
	device Device

	mu           sync.RWMutex
	moveHistory  []Move
	highestPhase Phase
	config       *config

	// Callbacks
	onMove        func(Move)
	onPhaseChange func(Phase)
	onOrientation func(Orientation)
	onBattery     func(int)
	onDisconnect  func(error)
	onSolved      func()
}

// Orientation represents the cube's physical orientation in space.
type Orientation struct {
	UpFace    Face // Which face is pointing up
	FrontFace Face // Which face is facing the user
}

// Scan discovers nearby GoCube devices via Bluetooth Low Energy.
// Returns all devices found within the timeout period.
//
// Typical usage:
//
//	devices, err := gocube.Scan(ctx, 10*time.Second)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, d := range devices {
//	    fmt.Printf("Found: %s (RSSI: %d)\n", d.Name, d.RSSI)
//	}
//
// Note: On macOS, BLE scanning sometimes requires multiple attempts.
// Ensure the cube is not connected to another device (e.g., phone app).
func Scan(ctx context.Context, timeout time.Duration) ([]Device, error) {
	client, err := ble.NewClient()
	if err != nil {
		return nil, err
	}
	defer client.Disconnect()

	results, err := client.Scan(ctx, timeout)
	if err != nil {
		return nil, err
	}

	devices := make([]Device, len(results))
	for i, r := range results {
		devices[i] = Device{
			Name:    r.Name,
			UUID:    r.UUID,
			RSSI:    r.RSSI,
			address: r.Address,
		}
	}

	return devices, nil
}

// Connect connects to a specific GoCube device.
func Connect(ctx context.Context, device Device, opts ...Option) (*GoCube, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	client, err := ble.NewClient()
	if err != nil {
		return nil, err
	}

	if err := client.Connect(ctx, device.UUID); err != nil {
		return nil, err
	}

	g := &GoCube{
		client:       client,
		cube:         NewCube(),
		device:       device,
		moveHistory:  make([]Move, 0),
		highestPhase: PhaseScrambled,
		config:       cfg,
	}

	// Set up internal message handling
	client.SetMessageCallback(g.handleMessage)

	return g, nil
}

// ConnectFirst scans and connects to the first GoCube found.
// This is a convenience function for quick prototyping and single-cube setups.
//
// It performs a 10-second scan and connects to the first device discovered.
// For production use with multiple cubes, use Scan and Connect separately.
//
// Example:
//
//	cube, err := gocube.ConnectFirst(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cube.Close()
func ConnectFirst(ctx context.Context, opts ...Option) (*GoCube, error) {
	devices, err := Scan(ctx, 10*time.Second)
	if err != nil {
		return nil, err
	}

	if len(devices) == 0 {
		return nil, ErrDeviceNotFound
	}

	return Connect(ctx, devices[0], opts...)
}

// Close disconnects from the cube and cleans up resources.
func (g *GoCube) Close() error {
	return g.client.Disconnect()
}

// IsConnected returns true if still connected to the cube.
func (g *GoCube) IsConnected() bool {
	return g.client.IsConnected()
}

// DeviceName returns the connected device name.
func (g *GoCube) DeviceName() string {
	return g.client.DeviceName()
}

// Event callbacks

// OnMove sets a callback that fires for each move detected.
func (g *GoCube) OnMove(cb func(Move)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.onMove = cb
}

// OnPhaseChange sets a callback that fires when a solving phase is completed.
// The callback receives the newly completed phase.
func (g *GoCube) OnPhaseChange(cb func(Phase)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.onPhaseChange = cb
}

// OnOrientationChange sets a callback for cube orientation changes.
func (g *GoCube) OnOrientationChange(cb func(Orientation)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.onOrientation = cb
}

// OnBattery sets a callback for battery level updates.
func (g *GoCube) OnBattery(cb func(int)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.onBattery = cb
}

// OnDisconnect sets a callback for disconnection events.
func (g *GoCube) OnDisconnect(cb func(error)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.onDisconnect = cb
}

// OnSolved sets a callback that fires when the cube reaches the solved state.
func (g *GoCube) OnSolved(cb func()) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.onSolved = cb
}

// State access

// Cube returns the current cube state.
// The returned cube can be inspected but modifications won't affect the GoCube.
func (g *GoCube) Cube() *Cube {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.cube.Clone()
}

// Phase returns the current solving phase.
func (g *GoCube) Phase() Phase {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.cube.Phase()
}

// HighestPhase returns the highest phase reached since connection or last reset.
// This is monotonic - it never goes backwards.
func (g *GoCube) HighestPhase() Phase {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.highestPhase
}

// IsSolved returns true if the cube is currently solved.
func (g *GoCube) IsSolved() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.cube.IsSolved()
}

// Battery returns the last known battery level (0-100), or -1 if unknown.
func (g *GoCube) Battery() int {
	return g.client.Battery()
}

// Moves returns the move history since connection or last clear.
func (g *GoCube) Moves() []Move {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([]Move, len(g.moveHistory))
	copy(result, g.moveHistory)
	return result
}

// Control

// Reset resets the internal cube state to solved.
// Does not affect the physical cube.
func (g *GoCube) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cube.Reset()
	g.highestPhase = PhaseScrambled
}

// ClearHistory clears the move history.
func (g *GoCube) ClearHistory() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.moveHistory = make([]Move, 0)
}

// FlashBacklight flashes the cube backlight.
func (g *GoCube) FlashBacklight() error {
	return g.client.FlashBacklight()
}

// EnableOrientation enables orientation tracking.
func (g *GoCube) EnableOrientation() error {
	return g.client.EnableOrientation()
}

// DisableOrientation disables orientation tracking.
func (g *GoCube) DisableOrientation() error {
	return g.client.DisableOrientation()
}

// Internal message handling

func (g *GoCube) handleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgTypeRotation:
		g.handleRotation(msg)
	case protocol.MsgTypeBattery:
		g.handleBattery(msg)
	case protocol.MsgTypeOrientation:
		g.handleOrientation(msg)
	}
}

func (g *GoCube) handleRotation(msg *protocol.Message) {
	rotations, err := protocol.DecodeRotation(msg.Payload)
	if err != nil {
		return
	}

	now := time.Now()
	for _, rot := range rotations {
		move := rotationToMove(rot, now)

		g.mu.Lock()
		g.cube.Apply(move)
		if g.config.moveHistory {
			g.moveHistory = append(g.moveHistory, move)
		}

		// Check for phase transitions
		currentPhase := g.cube.Phase()
		phaseCallback := g.onPhaseChange
		solvedCallback := g.onSolved
		isSolved := currentPhase == PhaseSolved
		phaseChanged := currentPhase > g.highestPhase
		if phaseChanged {
			g.highestPhase = currentPhase
		}
		g.mu.Unlock()

		// Fire callbacks outside the lock
		if phaseChanged && phaseCallback != nil {
			phaseCallback(currentPhase)
		}
		if isSolved && phaseChanged && solvedCallback != nil {
			solvedCallback()
		}

		// Call move callback
		g.mu.RLock()
		moveCallback := g.onMove
		g.mu.RUnlock()
		if moveCallback != nil {
			moveCallback(move)
		}
	}
}

func (g *GoCube) handleBattery(msg *protocol.Message) {
	battery, err := protocol.DecodeBattery(msg.Payload)
	if err != nil {
		return
	}

	g.mu.RLock()
	cb := g.onBattery
	g.mu.RUnlock()

	if cb != nil {
		cb(battery.Level)
	}
}

func (g *GoCube) handleOrientation(msg *protocol.Message) {
	orient, err := protocol.DecodeOrientation(msg.Payload)
	if err != nil {
		return
	}

	g.mu.RLock()
	cb := g.onOrientation
	g.mu.RUnlock()

	if cb != nil {
		cb(Orientation{
			UpFace:    Face(orient.UpFace),
			FrontFace: Face(orient.FrontFace),
		})
	}
}

// Color to face mapping based on GoCube protocol
var colorToFace = map[string]Face{
	"white":  FaceU,
	"yellow": FaceD,
	"green":  FaceF,
	"blue":   FaceB,
	"red":    FaceR,
	"orange": FaceL,
}

func rotationToMove(rot protocol.RotationEvent, t time.Time) Move {
	face := colorToFace[rot.Color]

	var turn Turn
	if rot.Clockwise {
		turn = CW
	} else {
		turn = CCW
	}

	return Move{
		Face: face,
		Turn: turn,
		Time: t,
	}
}
