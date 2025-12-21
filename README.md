# GoCube

A Go library for interacting with GoCube smart Rubik's cubes via Bluetooth Low Energy (BLE).

[![Go Reference](https://pkg.go.dev/badge/github.com/SeamusWaldron/gocube_ble_library.svg)](https://pkg.go.dev/github.com/SeamusWaldron/gocube_ble_library)
[![CI](https://github.com/SeamusWaldron/gocube_ble_library/actions/workflows/ci.yml/badge.svg)](https://github.com/SeamusWaldron/gocube_ble_library/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Device Discovery**: Scan for and connect to GoCube devices via BLE
- **Real-time Move Tracking**: Capture every move with millisecond timestamps
- **Cube State Simulation**: Track the virtual cube state as moves are applied
- **Phase Detection**: Automatically detect solving phases (cross, F2L, OLL, PLL)
- **Orientation Tracking**: Monitor cube orientation via quaternion data
- **Analysis Algorithms**: Analyze solve performance with detailed metrics

## Installation

### Library

```bash
go get github.com/SeamusWaldron/gocube_ble_library
```

### CLI Application

```bash
go install github.com/SeamusWaldron/gocube_ble_library/cmd/gocube@latest
```

## Requirements

- macOS (BLE functionality is currently macOS-only)
- Go 1.22+
- GoCube smart cube (tested with GoCube Edge)

## Quick Start

### Using as a Library

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/SeamusWaldron/gocube_ble_library"
)

func main() {
    // Create a new BLE client
    client, err := gocube.NewClient()
    if err != nil {
        panic(err)
    }
    defer client.Disconnect()

    // Scan for GoCube devices
    ctx := context.Background()
    results, err := client.Scan(ctx, 5*time.Second)
    if err != nil {
        panic(err)
    }

    if len(results) == 0 {
        fmt.Println("No GoCube found")
        return
    }

    // Connect to the first device found
    if err := client.Connect(ctx, results[0].UUID); err != nil {
        panic(err)
    }

    // Create a cube state tracker
    tracker := gocube.NewTracker()
    tracker.SetPhaseCallback(func(phase gocube.DetectedPhase, key string) {
        fmt.Printf("Phase completed: %s\n", key)
    })

    // Set up message callback for moves
    client.SetMessageCallback(func(msg *gocube.Message) {
        if msg.Type == gocube.MsgTypeRotation {
            rotations, _ := gocube.DecodeRotation(msg.Payload)
            moves := gocube.RotationsToMoves(rotations, 0)
            for _, move := range moves {
                fmt.Printf("Move: %s\n", move.Notation())
                tracker.ApplyMove(move)
            }
        }
    })

    // Keep running...
    select {}
}
```

### Using the CLI

```bash
# Check connection status
gocube status

# Record a solve
gocube solve record

# Generate analysis report
gocube report solve --last

# List recent solves
gocube solve list
```

## Library API

### Core Types

```go
// Move represents a single cube move
type Move struct {
    Face      Face   // R, L, U, D, F, B
    Turn      Turn   // TurnCW (1), TurnCCW (-1), Turn180 (2)
    Timestamp int64  // Milliseconds
}

// Cube represents the cube state
type Cube struct {
    // 6 faces x 9 facelets
}

// Tracker wraps Cube with phase detection
type Tracker struct {
    // ...
}

// Client handles BLE communication
type Client struct {
    // ...
}
```

### Key Functions

```go
// BLE
func NewClient() (*Client, error)
func (c *Client) Scan(ctx context.Context, timeout time.Duration) ([]ScanResult, error)
func (c *Client) Connect(ctx context.Context, uuid string) error
func (c *Client) SetMessageCallback(cb func(*Message))

// Cube State
func NewCube() *Cube
func (c *Cube) ApplyMove(m Move)
func (c *Cube) IsSolved() bool
func (c *Cube) DetectPhase() DetectedPhase

// Tracking
func NewTracker() *Tracker
func (t *Tracker) ApplyMove(m Move)
func (t *Tracker) SetPhaseCallback(cb func(DetectedPhase, string))

// Message Decoding
func DecodeRotation(payload []byte) ([]RotationEvent, error)
func DecodeBattery(payload []byte) (*BatteryEvent, error)
func DecodeOrientation(payload []byte) (*OrientationEvent, error)
```

## Solving Phases

The library detects these standard layer-by-layer solving phases:

| Phase | Description |
|-------|-------------|
| `scrambled` | Cube is scrambled |
| `inspection` | Pre-solve inspection |
| `white_cross` | Building the white cross |
| `white_corners` | Completing first layer corners |
| `middle_layer` | Completing F2L |
| `bottom_cross` | Yellow cross (OLL cross) |
| `bottom_perm` | Positioning last layer (PLL) |
| `bottom_orient` | Orienting last layer corners |
| `solved` | Cube is solved |

## CLI Features

The included CLI application provides:

- **Interactive Recording TUI**: Beautiful terminal interface for recording solves
- **Automatic Phase Detection**: Real-time phase tracking during solves
- **Comprehensive Reports**: Detailed analysis including:
  - Move statistics and TPS (turns per second)
  - Phase-by-phase breakdown
  - Pattern detection (n-grams)
  - Inefficiency analysis (cancellations, merges)
  - Playback export for visualization
- **Session Replay**: Debug phase detection without the physical cube
- **SQLite Storage**: Persistent storage for all solve data

### Recording Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `s` | Start new solve |
| `SPACE` | Start solve timer |
| `1-7` | Manually mark phase |
| `d` | Toggle debug mode |
| `e` | End solve |
| `q` | Quit |

## Troubleshooting

### "No GoCube devices found"

1. Disconnect the cube from your phone (Bluetooth settings > Forget Device)
2. Wake the cube by rotating it
3. Try scanning twice (macOS BLE sometimes needs multiple scans)

### Phases not detecting correctly

Ensure standard orientation: **white on top, green facing you** when starting.

## Data Storage

The CLI stores data in `~/.gocube_recorder/`:
- `gocube.db` - SQLite database
- `state.json` - Application state
- `logs/` - Session logs for replay

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT - see [LICENSE](LICENSE) for details.
