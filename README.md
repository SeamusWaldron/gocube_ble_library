# GoCube Solve Recorder

A CLI application for recording and analyzing Rubik's cube solves using a GoCube smart cube.

## Features

- **Real-time Move Tracking**: Connects to GoCube via Bluetooth and captures every move with timestamps
- **Automatic Phase Detection**: Detects when you complete each solving phase (cross, corners, layers, etc.)
- **Orientation Tracking**: Records cube rotations for visualization and analysis
- **Solve Recording**: Records complete solve sessions with timestamps and phase markers
- **Interactive TUI**: Beautiful terminal interface for recording solves
- **Comprehensive Reports**: Generates detailed analysis reports with diagnostics
- **Playback Export**: JSON timeline for web-based solve visualization
- **SQLite Storage**: Persistent storage for all solve data

## Requirements

- macOS (with Bluetooth)
- Go 1.24+
- GoCube smart cube (tested with GoCube Edge)

## Installation

```bash
# Clone the repository
git clone https://github.com/seamusw/gocube.git
cd gocube

# Build (requires CGO for Bluetooth)
CGO_ENABLED=1 go build -o gocube ./cmd/gocube

# Build debug tools (optional)
go build -o ble-tracker ./cmd/ble-tracker
```

## Quick Start

### 1. Check Connection

```bash
./gocube status
```

This scans for your GoCube and shows connection status. Make sure:
- Your GoCube is NOT connected to your phone
- The cube is awake (rotate it to wake it up)

### 2. Record a Solve

```bash
./gocube solve record
```

This starts the interactive recording TUI:

1. Press `s` to start a new solve
2. Scramble your cube (screen shows "SCRAMBLE THE CUBE")
3. Inspect the cube (screen shows "INSPECTION")
4. Press `SPACE` when ready to start solving
5. Solve the cube - phases are auto-detected!
6. Press `e` to end, or it auto-ends when solved

### 3. Generate Report

```bash
./gocube report solve --last
```

This generates a comprehensive analysis report in `reports/YYYY-MM-DD_HHMMSS/` including:
- `solve_summary.json` - Overview statistics
- `playback.json` - Timeline for visualization playback
- `moves.json` - Detailed move data with timestamps
- `diagnostics.json` - Performance metrics and patterns
- `phase_analysis.json` - Per-phase breakdown
- `ngram_report.json` - Repeated move patterns

### 4. View Solves

```bash
./gocube solve list
./gocube solve show --last
```

## Commands

| Command | Description |
|---------|-------------|
| `gocube status` | Show connection and cube status |
| `gocube solve record` | Interactive solve recording |
| `gocube solve replay` | Replay a recorded session |
| `gocube solve list` | List recent solves |
| `gocube solve show <id>` | Show solve details |
| `gocube report solve --last` | Generate analysis report |
| `gocube report trend --window 50` | Trend analysis across solves |

## Report Output

Reports are saved to `reports/YYYY-MM-DD_HHMMSS/` and include:

### Diagnostics
- **Reversals**: Immediate cancellations (R R') and wasted moves
- **Face Entropy**: Measures searching vs algorithmic solving
- **Edge Placements**: Cross building efficiency (white_cross phase)
- **Pause Analysis**: Gaps between moves indicating thinking time
- **Short Loops**: Patterns like A B A' indicating uncertainty

### Playback Export
The `playback.json` file contains a chronological timeline of moves and orientation changes, suitable for web-based 3D visualization:

```json
{
  "duration_ms": 101917,
  "total_moves": 200,
  "timeline": [
    {"ts_ms": 150, "type": "move", "face": "R", "turn": 1, "notation": "R"},
    {"ts_ms": 1200, "type": "orientation", "up_face": "F", "front_face": "D"}
  ]
}
```

## Replay Mode

Every solve session is automatically logged to `~/.gocube_recorder/logs/`. You can replay these logs to debug phase detection without needing the physical cube:

```bash
# List available logs
./gocube solve replay

# Replay a specific log
./gocube solve replay solve_20241221_143052.jsonl

# Replay at 2x speed
./gocube solve replay <log-file> --speed 2.0

# Step through events manually
./gocube solve replay <log-file> --step
```

### Replay Keyboard Shortcuts
| Key | Action |
|-----|--------|
| `SPACE/n` | Next event (step mode) / Play-pause |
| `p` | Pause/resume |
| `r` | Reset to beginning |
| `d` | Toggle debug mode (show cube state) |
| `+/-` | Increase/decrease speed |
| `q` | Quit |

## Keyboard Shortcuts (Recording TUI)

### Before Starting
| Key | Action |
|-----|--------|
| `s` | Start new solve |
| `d` | Toggle debug mode |
| `q` | Quit |

### During Inspection/Scramble
| Key | Action |
|-----|--------|
| `SPACE` | Start solve timer |
| `d` | Toggle debug mode |
| `e` | End/cancel |
| `q` | Quit |

### During Solve
| Key | Action |
|-----|--------|
| `1-7` | Manually mark phase |
| `r` | Mark RHS algorithm |
| `l` | Mark LHS algorithm |
| `d` | Toggle debug mode |
| `e` | End solve |
| `q` | Quit |

## Solving Phases

The app tracks standard layer-by-layer solving:

| # | Phase | What to Complete |
|---|-------|------------------|
| 0 | Inspection | Scramble + look at cube |
| 1 | White Cross | 4 white edges on top |
| 2 | Top Corners | Complete white face |
| 3 | Middle Layer | Middle layer edges |
| 4 | Bottom Cross | Yellow cross on bottom |
| 5 | Position Corners | Yellow corners in place |
| 6 | Rotate Corners | Orient yellow corners |
| 7 | Complete | Solved! |

## Debug Tools

```bash
# Real-time cube state tracker
./ble-tracker

# Raw BLE message viewer
./ble-raw

# BLE device scanner
./ble-debug
```

## Troubleshooting

### "No GoCube devices found"

1. **Disconnect from phone**: Go to iPhone Settings > Bluetooth > GoCube > Forget This Device
2. **Wake the cube**: Rotate it to turn on the LED
3. **Run status twice**: Sometimes macOS BLE needs multiple scans
   ```bash
   ./gocube status
   ./gocube status  # Second time usually works
   ```

### Connection drops

The cube may disconnect after inactivity. Just run the command again - it will reconnect.

### Phases not detecting correctly

Make sure you're holding the cube with **white on top, green facing you** when you start. The tracker assumes standard orientation.

### No orientation data in playback.json

Orientation tracking is enabled automatically on connection. If `total_orientations` is 0, ensure your GoCube supports orientation (GoCube Edge does).

## Data Storage

- Database: `~/.gocube_recorder/gocube.db`
- State: `~/.gocube_recorder/state.json`
- Logs: `~/.gocube_recorder/logs/`
- Reports: `./reports/YYYY-MM-DD_HHMMSS/`

## Technical Details

See [docs/SPECIFICATION.md](docs/SPECIFICATION.md) for complete technical documentation including:
- GoCube BLE protocol
- Database schema
- Cube model implementation
- Phase detection algorithms

## License

MIT

## Contributing

Contributions welcome! Please read the specification first to understand the architecture.
