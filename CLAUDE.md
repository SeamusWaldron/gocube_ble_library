# CLAUDE.md - Instructions for Claude Code

This file provides context for Claude Code when working on this project.

## Project Overview

GoCube Solve Recorder is a Go CLI application that connects to a GoCube smart Rubik's cube via BLE, records solve sessions, provides automatic phase detection, and generates comprehensive analysis reports.

## Build Commands

```bash
# Build main application
CGO_ENABLED=1 go build -o gocube ./cmd/gocube

# Build all debug tools
go build -o ble-tracker ./cmd/ble-tracker
go build -o ble-raw ./cmd/ble-raw
go build -o ble-debug ./cmd/ble-debug
go build -o ble-state ./cmd/ble-state

# Run tests
go test ./internal/cube/... -v
go test ./... -v
```

## Key Architecture

### Package Structure

- `internal/ble/` - BLE client, scanning, connection management
- `internal/gocube/` - GoCube protocol parsing and decoding
- `internal/cube/` - 3x3 cube model with move simulation and phase detection
- `internal/recorder/` - Solve session lifecycle management
- `internal/storage/` - SQLite database with migrations
- `internal/analysis/` - Solve analysis, diagnostics, n-gram mining
- `internal/cli/` - Cobra commands and BubbleTea TUI
- `pkg/types/` - Shared Move type

### Important Files

| File | Purpose |
|------|---------|
| `internal/cli/record.go` | Main TUI for solve recording |
| `internal/cli/report.go` | Report generation commands |
| `internal/cube/cube.go` | Cube model with move application |
| `internal/cube/phases.go` | Phase detection logic |
| `internal/cube/apply.go` | Tracker for state changes |
| `internal/ble/client.go` | BLE connection and messaging |
| `internal/gocube/protocol.go` | Message frame parsing |
| `internal/gocube/decoder.go` | Payload decoding (rotation, orientation) |
| `internal/gocube/moves.go` | Color to face mapping |
| `internal/analysis/diagnostics.go` | Solve diagnostics and metrics |
| `internal/storage/orientations.go` | Orientation state storage |

## BLE Protocol Notes

### Message Frame
```
[0x2A] [length] [type] [payload...] [checksum] [0x0D 0x0A]
```
- Length = bytes from position 2 to end
- Checksum = sum of bytes 0 to checksum-1, mod 256

### Rotation Message (0x01)
Pairs of bytes: `[face_dir] [center_orientation]`
- Face codes 0-11: even = CW, odd = CCW
- Color index = face_code / 2
- Colors: 0=blue(B), 1=green(F), 2=white(U), 3=yellow(D), 4=red(R), 5=orange(L)

### Orientation Message (0x03)
ASCII string: `x#y#z#w` (quaternion format)
- Parsed to derive up_face and front_face
- Automatically enabled on connection via `CmdEnableOrientation`

## Cube Model

### Face Indexing
```
0 1 2
3 4 5  (4 = center, never moves)
6 7 8
```

### Standard Orientation
- White on top (U)
- Green in front (F)
- This matches GoCube calibration orientation

## Report System

Reports are generated to `reports/YYYY-MM-DD_HHMMSS/` and include:

| File | Content |
|------|---------|
| `solve_summary.json` | Overview statistics |
| `playback.json` | Timeline for visualization (moves + orientations) |
| `moves.json` | Detailed move data with timestamps |
| `diagnostics.json` | Performance metrics |
| `phase_analysis.json` | Per-phase breakdown |
| `ngram_report.json` | Repeated patterns |
| `repetition_report.json` | Cancellations and merge opportunities |

### Playback Format
```json
{
  "timeline": [
    {"ts_ms": 150, "type": "move", "face": "R", "turn": 1, "notation": "R"},
    {"ts_ms": 1200, "type": "orientation", "up_face": "F", "front_face": "D"}
  ]
}
```

### Diagnostics Metrics
- `ImmediateReversals` - X X' patterns
- `FaceEntropy` - Shannon entropy of face distribution (high = searching)
- `EdgePlacements` - Cross building efficiency (white_cross phase)
- `ShortLoops` - A B A' patterns indicating uncertainty
- `OrientationDiagnostics` - Cube rotation analysis

## Common Tasks

### Adding a New Phase
1. Add constant in `internal/cube/phases.go`
2. Add detection function (e.g., `IsXXXComplete()`)
3. Update `DetectPhase()` order (most complete first)
4. Add to `String()` method
5. Add to `internal/storage/phases.go` mappings

### Adding New Diagnostics
1. Add fields to `PhaseDiagnostics` or `OrientationDiagnostics` in `diagnostics.go`
2. Implement analysis function
3. Call from `analyzePhaseMoves()` or `analyzeOrientations()`
4. Update report display in `report.go`

### Debugging BLE Issues
1. Run `./ble-tracker` to see all messages with cube state
2. Check phase detection output after each move
3. Verify color-to-face mapping in decoder

### Testing Cube Logic
```bash
go test ./internal/cube/... -v
```

## Database

Location: `~/.gocube_recorder/gocube.db`

### Key Tables
- `solves` - Solve sessions
- `moves` - Individual moves with timestamps
- `events` - Raw BLE events
- `phase_marks` - Manual phase markers
- `derived_phase_segments` - Computed phase timing
- `orientations` - Orientation state changes

Migrations are in `internal/storage/migrations/` and auto-applied on startup.

## Known Issues

### macOS BLE Warm-up
The BLE adapter often needs multiple scan cycles to discover devices. The code includes retry logic in `runRecord()`.

### Phase Detection During Scramble
Auto-phase detection only activates after user presses SPACE to start solving. During scramble/inspection, phases are tracked but not auto-marked.

### Orientation Tracking
Orientation is enabled automatically on BLE connection. If no orientation events appear, the cube may not support it or needs firmware update.

## Testing Workflow

1. Build: `go build -o gocube ./cmd/gocube`
2. Run: `./gocube solve record`
3. Start solve with `s`
4. Scramble cube
5. Press `SPACE` to start timer
6. Solve - watch phases auto-detect
7. End with `e` or complete solve
8. Generate report: `./gocube report solve --last`

## Code Style

- Standard Go formatting (`gofmt`)
- Error handling: return errors, don't panic
- Concurrency: use channels for BLE message passing
- TUI: BubbleTea model-update-view pattern
