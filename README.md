# GoCube

A Go library for interacting with GoCube smart Rubik's cubes via Bluetooth Low Energy (BLE).

[![Go Reference](https://pkg.go.dev/badge/github.com/SeamusWaldron/gocube_ble_library.svg)](https://pkg.go.dev/github.com/SeamusWaldron/gocube_ble_library)
[![CI](https://github.com/SeamusWaldron/gocube_ble_library/actions/workflows/ci.yml/badge.svg)](https://github.com/SeamusWaldron/gocube_ble_library/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Clean API**: Simple, callback-based interface for cube events
- **Device Discovery**: Scan for and connect to GoCube devices via BLE
- **Real-time Move Tracking**: Capture every move with timestamps
- **Cube State Simulation**: Track the virtual cube state as moves are applied
- **Phase Detection**: Automatically detect solving phases (cross, F2L, OLL, PLL)
- **Standalone Simulation**: Use the cube model without BLE for testing/visualization
- **Predefined Moves**: Convenient constants like `gocube.R`, `gocube.UPrime`, `gocube.F2`

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

### Connect to a GoCube

```go
package main

import (
    "context"
    "fmt"

    "github.com/SeamusWaldron/gocube_ble_library"
)

func main() {
    ctx := context.Background()

    // Scan and connect to the first GoCube found
    cube, err := gocube.ConnectFirst(ctx)
    if err != nil {
        panic(err)
    }
    defer cube.Close()

    fmt.Printf("Connected to: %s\n", cube.DeviceName())

    // React to moves
    cube.OnMove(func(m gocube.Move) {
        fmt.Printf("Move: %s\n", m.Notation())
    })

    // React to phase changes
    cube.OnPhaseChange(func(p gocube.Phase) {
        fmt.Printf("Phase completed: %s\n", p.String())
    })

    // React to solved state
    cube.OnSolved(func() {
        fmt.Println("Cube solved!")
    })

    // Keep running until Ctrl+C
    select {}
}
```

### Standalone Cube Simulation

```go
package main

import (
    "fmt"

    "github.com/SeamusWaldron/gocube_ble_library"
)

func main() {
    // Create a solved cube (no BLE needed)
    cube := gocube.NewCube()

    // Apply moves using predefined constants
    cube.Apply(gocube.R, gocube.U, gocube.RPrime, gocube.UPrime)

    // Or parse from notation
    cube.ApplyNotation("F B2 L' D")

    // Check state
    fmt.Printf("Solved: %v\n", cube.IsSolved())
    fmt.Printf("Phase: %s\n", cube.Phase().String())

    // Get detailed progress
    progress := cube.GetProgress()
    fmt.Printf("White Cross: %v\n", progress.WhiteCross)
    fmt.Printf("First Layer: %v\n", progress.FirstLayer)

    // Visualize
    fmt.Println(cube.String())
}
```

### Using the CLI

```bash
# Check connection status
gocube status

# Record a solve interactively
gocube solve record

# Generate analysis report
gocube report solve --last

# List recent solves
gocube solve list
```

## API Reference

### Core Types

#### Move

Represents a single cube move.

```go
type Move struct {
    Face Face      // Which face: R, L, U, D, F, B
    Turn Turn      // Direction: CW (1), CCW (-1), Double (2)
    Time time.Time // When the move occurred (optional)
}

// Methods
func (m Move) Notation() string  // Returns "R", "R'", "R2", etc.
func (m Move) Inverse() Move     // R -> R', R' -> R, R2 -> R2
```

#### Predefined Moves

```go
// Right face
gocube.R       // R  (clockwise)
gocube.RPrime  // R' (counter-clockwise)
gocube.R2      // R2 (180 degrees)

// All faces available: R, L, U, D, F, B
// Each with: X, XPrime, X2 variants
```

#### Cube

Represents a 3x3 Rubik's cube state. Can be used standalone without BLE.

```go
func NewCube() *Cube                        // Create solved cube
func (c *Cube) Apply(moves ...Move)         // Apply moves
func (c *Cube) ApplyNotation(s string) error // Apply from notation string
func (c *Cube) IsSolved() bool              // Check if solved
func (c *Cube) Phase() Phase                // Current solving phase
func (c *Cube) GetProgress() Progress       // Detailed phase progress
func (c *Cube) Reset()                      // Reset to solved state
func (c *Cube) Clone() *Cube                // Deep copy
func (c *Cube) String() string              // ASCII visualization
```

#### Phase

Represents solving phases in layer-by-layer method.

```go
const (
    PhaseScrambled      Phase = iota // Cube is scrambled
    PhaseWhiteCross                  // White cross complete
    PhaseFirstLayer                  // First layer complete
    PhaseSecondLayer                 // Second layer (F2L) complete
    PhaseYellowCross                 // Yellow cross formed
    PhaseYellowCorners               // Yellow corners positioned
    PhaseYellowOriented              // Yellow corners oriented
    PhaseSolved                      // Cube is solved
)

func (p Phase) String() string // "scrambled", "white_cross", etc.
```

#### GoCube (BLE Connection)

Represents a connected GoCube device.

```go
// Discovery
func Scan(ctx context.Context, timeout time.Duration) ([]Device, error)
func Connect(ctx context.Context, device Device, opts ...Option) (*GoCube, error)
func ConnectFirst(ctx context.Context, opts ...Option) (*GoCube, error)

// Connection
func (g *GoCube) Close() error
func (g *GoCube) IsConnected() bool
func (g *GoCube) DeviceName() string

// Callbacks
func (g *GoCube) OnMove(cb func(Move))
func (g *GoCube) OnPhaseChange(cb func(Phase))
func (g *GoCube) OnOrientationChange(cb func(Orientation))
func (g *GoCube) OnBattery(cb func(int))
func (g *GoCube) OnDisconnect(cb func(error))
func (g *GoCube) OnSolved(cb func())

// State
func (g *GoCube) Cube() *Cube     // Current cube state
func (g *GoCube) Phase() Phase    // Current phase
func (g *GoCube) IsSolved() bool  // Convenience check
func (g *GoCube) Battery() int    // Battery percentage
func (g *GoCube) Moves() []Move   // Move history
```

#### Options

```go
func WithAutoReconnect(enabled bool) Option  // Auto-reconnect on disconnect
func WithMoveHistory(enabled bool) Option    // Track move history
func WithPhaseDetection(enabled bool) Option // Auto phase detection
```

### Parsing Moves

```go
// Parse single move
move, err := gocube.ParseMove("R'")

// Parse sequence
moves, err := gocube.ParseMoves("R U R' U'")
```

## Solving Phases

The library detects these standard layer-by-layer solving phases:

| Phase | Description |
|-------|-------------|
| `scrambled` | Cube is scrambled |
| `white_cross` | White cross complete |
| `first_layer` | First layer (white face + corners) complete |
| `second_layer` | Middle layer (F2L) complete |
| `yellow_cross` | Yellow cross formed on bottom |
| `yellow_corners` | Bottom corners positioned correctly |
| `yellow_oriented` | Bottom corners oriented (OLL complete) |
| `solved` | Cube is solved |

## Examples

See the [examples/](examples/) directory for complete working examples:

- **[connect/](examples/connect/)** - Basic device connection
- **[track-moves/](examples/track-moves/)** - Real-time move tracking with phase detection
- **[simulate/](examples/simulate/)** - Standalone cube simulation without BLE

## CLI Features

The included CLI application provides:

- **Interactive Recording TUI**: Beautiful terminal interface for recording solves
- **Automatic Phase Detection**: Real-time phase tracking during solves
- **Comprehensive Reports**: Detailed analysis including:
  - Move statistics and TPS (turns per second)
  - Phase-by-phase breakdown
  - Pattern detection (n-grams)
  - Inefficiency analysis (cancellations, merges)
- **Session Replay**: Debug phase detection without the physical cube
- **SQLite Storage**: Persistent storage for all solve data

### Recording Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `s` | Start new solve |
| `SPACE` | Start solve timer (after scramble) |
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
- `gocube.db` - SQLite database with all solve data
- `state.json` - Application state (last device, active solve)
- `logs/` - Session logs for replay debugging

## Architecture

The library uses a layered architecture:

```
Public API (gocube package)
├── Move, Face, Turn      - Core types
├── Cube                  - Cube simulation (standalone)
├── Phase, Progress       - Phase detection
├── GoCube, Device        - BLE device connection
└── Options               - Configuration

Internal (not for external use)
├── internal/ble/         - BLE transport layer
├── internal/protocol/    - GoCube protocol decoding
└── internal/app/         - CLI-specific code
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT - see [LICENSE](LICENSE) for details.
