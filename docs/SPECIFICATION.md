# GoCube Solve Recorder - Project Specification

## Overview

GoCube Solve Recorder is a CLI application written in Go that connects to a GoCube smart Rubik's cube via Bluetooth Low Energy (BLE), records solve sessions with automatic phase detection, persists data in SQLite, and provides analysis capabilities.

## Goals

1. **Real-time Move Capture**: Connect to GoCube via BLE and capture every move in real-time
2. **Automatic Phase Detection**: Track cube state and automatically detect when solving phases are completed
3. **Solve Recording**: Record complete solve sessions with timestamps, moves, and phase markers
4. **Data Persistence**: Store all solve data in SQLite for later analysis
5. **Analysis & Reporting**: Generate statistics and identify patterns in solving technique

---

## Architecture

### Project Structure

```
gocube/
├── cmd/
│   ├── gocube/          # Main CLI application
│   ├── ble-debug/       # BLE scanner/debugger
│   ├── ble-raw/         # Raw BLE message viewer
│   ├── ble-state/       # Cube state message debugger
│   └── ble-tracker/     # Real-time cube state tracker
├── internal/
│   ├── ble/             # BLE client and connection management
│   ├── gocube/          # GoCube protocol decoder
│   ├── cube/            # 3x3 cube model and phase detection
│   ├── recorder/        # Solve session management
│   ├── storage/         # SQLite database layer
│   ├── analysis/        # Solve analysis algorithms
│   ├── notation/        # Move notation handling
│   └── cli/             # Cobra CLI commands
├── pkg/
│   └── types/           # Shared types (Move, Face, Turn)
├── migrations/          # SQLite schema migrations
├── docs/                # Documentation
└── reports/             # Generated analysis reports
```

### Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Language | Go 1.24+ | Core application |
| BLE | tinygo.org/x/bluetooth | Bluetooth connectivity |
| Database | modernc.org/sqlite | Data persistence (pure Go) |
| CLI | github.com/spf13/cobra | Command-line interface |
| TUI | github.com/charmbracelet/bubbletea | Interactive terminal UI |
| Styling | github.com/charmbracelet/lipgloss | Terminal styling |

---

## GoCube BLE Protocol

### Service UUIDs

| UUID | Description |
|------|-------------|
| `6e400001-b5a3-f393-e0a9-e50e24dcca9e` | Nordic UART Service |
| `6e400002-b5a3-f393-e0a9-e50e24dcca9e` | RX Characteristic (Write) |
| `6e400003-b5a3-f393-e0a9-e50e24dcca9e` | TX Characteristic (Notify) |

### Message Frame Format

```
[0x2A] [length] [type] [payload...] [checksum] [0x0D 0x0A]
  │       │       │         │           │         └── Suffix (CR LF)
  │       │       │         │           └── Sum of bytes 0 to checksum-1, mod 256
  │       │       │         └── Variable payload
  │       │       └── Message type identifier
  │       └── Bytes from position 2 to end
  └── Prefix ('*')
```

### Message Types

| Type | Name | Description |
|------|------|-------------|
| 0x01 | Rotation | Face rotation event(s) |
| 0x02 | State | Full cube state (54 facelets) |
| 0x03 | Orientation | Quaternion orientation data |
| 0x05 | Battery | Battery level (0-100%) |
| 0x07 | Offline Stats | Moves/time/solves while disconnected |
| 0x08 | Cube Type | Standard or Edge cube |

### Rotation Message Format

Payload contains pairs of bytes: `[face_dir] [center_orientation]`

| Face Code | Color | Direction |
|-----------|-------|-----------|
| 0x00 | Blue | Clockwise |
| 0x01 | Blue | Counter-clockwise |
| 0x02 | Green | Clockwise |
| 0x03 | Green | Counter-clockwise |
| 0x04 | White | Clockwise |
| 0x05 | White | Counter-clockwise |
| 0x06 | Yellow | Clockwise |
| 0x07 | Yellow | Counter-clockwise |
| 0x08 | Red | Clockwise |
| 0x09 | Red | Counter-clockwise |
| 0x0A | Orange | Clockwise |
| 0x0B | Orange | Counter-clockwise |

### Color to Face Mapping (Standard Orientation)

Standard orientation: White on top, Green in front.

| GoCube Color | Standard Face | Notation |
|--------------|---------------|----------|
| White | Up | U |
| Yellow | Down | D |
| Green | Front | F |
| Blue | Back | B |
| Red | Right | R |
| Orange | Left | L |

---

## Cube Model

### Facelet Indexing

Each face has 9 facelets indexed as:
```
0 1 2
3 4 5
6 7 8
```

Position 4 is always the center (fixed color).

### Face Adjacency

```
        +---+
        | U |
    +---+---+---+---+
    | L | F | R | B |
    +---+---+---+---+
        | D |
        +---+
```

### Move Representation

| Turn Value | Meaning | Notation |
|------------|---------|----------|
| 1 | Clockwise 90° | R, U, F, etc. |
| -1 | Counter-clockwise 90° | R', U', F', etc. |
| 2 | 180° | R2, U2, F2, etc. |

---

## Solve Phases

The application tracks progress through the layer-by-layer solving method:

| # | Phase Key | Display Name | Detection |
|---|-----------|--------------|-----------|
| 0 | inspection | Inspection | Manual (start of solve) |
| 1 | white_cross | White Cross | 4 white edges on U face, correctly oriented |
| 2 | top_corners | Top Corners | All 9 U face facelets white, corners match centers |
| 3 | middle_layer | Middle Layer | Middle edges on F, R, B, L match centers |
| 4 | bottom_cross | Bottom Cross | 4 yellow edges on D face |
| 5 | position_corners | Position Corners | D corners have correct color sets |
| 6 | rotate_corners | Rotate Corners | All D face facelets yellow |
| 7 | complete | Complete | Cube solved |

### Algorithm Markers

| Key | Phase | Description |
|-----|-------|-------------|
| r | middle_rhs | Right-hand-side algorithm |
| l | middle_lhs | Left-hand-side algorithm |

---

## Database Schema

### Tables

#### solves
```sql
CREATE TABLE solves (
    solve_id TEXT PRIMARY KEY,
    started_at DATETIME NOT NULL,
    ended_at DATETIME,
    scramble TEXT,
    notes TEXT,
    device_name TEXT,
    device_id TEXT,
    app_version TEXT
);
```

#### raw_events
```sql
CREATE TABLE raw_events (
    event_id INTEGER PRIMARY KEY,
    solve_id TEXT NOT NULL,
    ts_ms INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    payload_json TEXT,
    raw_base64 TEXT,
    FOREIGN KEY (solve_id) REFERENCES solves(solve_id)
);
```

#### moves
```sql
CREATE TABLE moves (
    move_id INTEGER PRIMARY KEY,
    solve_id TEXT NOT NULL,
    seq INTEGER NOT NULL,
    ts_ms INTEGER NOT NULL,
    face TEXT NOT NULL,
    turn INTEGER NOT NULL,
    notation TEXT NOT NULL,
    source_event_id INTEGER,
    FOREIGN KEY (solve_id) REFERENCES solves(solve_id)
);
```

#### phase_marks
```sql
CREATE TABLE phase_marks (
    phase_mark_id INTEGER PRIMARY KEY,
    solve_id TEXT NOT NULL,
    ts_ms INTEGER NOT NULL,
    phase_key TEXT NOT NULL,
    mark_type TEXT DEFAULT 'start',
    notes TEXT,
    FOREIGN KEY (solve_id) REFERENCES solves(solve_id)
);
```

#### phase_defs
```sql
CREATE TABLE phase_defs (
    phase_key TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    order_index INTEGER NOT NULL,
    description TEXT,
    is_active INTEGER DEFAULT 1
);
```

---

## CLI Commands

### Main Commands

| Command | Description |
|---------|-------------|
| `gocube status` | Show connection status, battery, solve count |
| `gocube solve record` | Interactive TUI for recording solves |
| `gocube solve list` | List recent solves |
| `gocube solve show <id>` | Show details of a specific solve |
| `gocube export moves --id <id>` | Export moves in various formats |
| `gocube report solve --last` | Generate solve analysis report |

### TUI Keyboard Shortcuts

#### Before Solve Started
| Key | Action |
|-----|--------|
| s | Start new solve (begin inspection) |
| q / Esc | Quit |

#### During Inspection
| Key | Action |
|-----|--------|
| SPACE | Start solve timer (after scrambling) |
| e | End solve |
| q | Quit |

#### During Solve
| Key | Action |
|-----|--------|
| 1-7 | Manually mark phase |
| r | Mark RHS algorithm |
| l | Mark LHS algorithm |
| e | End solve |
| q | Quit |

---

## Solve Workflow

1. **Start**: Press `s` to begin a new solve session
2. **Scramble**: Scramble the cube (state shows "SCRAMBLE THE CUBE")
3. **Inspect**: Look at the cube (state shows "INSPECTION")
4. **Begin Solve**: Press `SPACE` to start the solve timer
5. **Solve**: Complete each phase - auto-detected and marked
6. **End**: Press `e` or complete the solve (auto-detected)

---

## Analysis Features

### Per-Solve Analysis
- Total moves and duration
- Turns per second (TPS)
- Phase breakdown with individual times
- Pause detection

### Pattern Detection
- Immediate cancellations (R followed by R')
- Merge opportunities (R + R → R2)
- N-gram mining for repeated sequences
- RHS/LHS algorithm detection

### Trend Analysis
- Rolling averages over solve window
- Per-phase trends
- Most repeated patterns

---

## Configuration

### State File
Location: `~/.gocube_recorder/state.json`

```json
{
    "active_solve_id": "",
    "last_device_id": "uuid-string",
    "last_device_name": "GoCube_XXXXXX",
    "db_path": "/path/to/gocube.db"
}
```

### Database
Default location: `~/.gocube_recorder/gocube.db`

---

## Platform Notes

### macOS
- Requires Bluetooth permission for Terminal
- Device addresses are UUIDs (not MAC addresses)
- BLE adapter may need multiple scan cycles to discover devices
- Requires CGO for TinyGo Bluetooth (CoreBluetooth bindings)

### Building
```bash
CGO_ENABLED=1 go build -o gocube ./cmd/gocube
```

---

## Future Enhancements

1. **State Message Decoding**: Decode 0x02 state messages for full cube state sync
2. **Scramble Generation**: Generate and display scramble sequences
3. **Competition Mode**: WCA-compliant timing with inspection countdown
4. **Cross-Platform**: Linux and Windows support
5. **Web Dashboard**: Visualization of solve statistics
6. **Algorithm Library**: Recognition and naming of common algorithms
