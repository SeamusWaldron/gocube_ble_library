# GoCube Recorder - Quick Start

## Build

```bash
go build -o gocube ./cmd/gocube
```

## Record a Solve

1. Wake your GoCube (rotate it)
2. Disconnect from phone app if connected
3. Run:

```bash
./gocube solve record
```

## Workflow

1. **Press `s`** - Start recording (cube must be solved)
2. **Scramble the cube** - Moves are tracked as scramble phase
3. **Press `SPACE`** - Signal you're ready (starts inspection)
4. **Make first move** - Timer starts, phases auto-detected
5. **Solve the cube** - Auto-ends when solved, report generated

## Output

Reports saved to `reports/YYYY-MM-DD_HHMMSS/`:
- `visualizer.html` - Interactive 3D playback
- `solve_summary.json` - Stats and metrics
- `diagnostics.json` - Detailed analysis

## Keys

| Key | Action |
|-----|--------|
| `s` | Start new solve |
| `SPACE` | End scramble, start inspection |
| `e` | End solve manually |
| `d` | Toggle debug view |
| `q` | Quit |
