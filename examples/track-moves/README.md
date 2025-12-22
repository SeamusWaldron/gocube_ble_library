# Track Moves Example

Advanced example demonstrating real-time move tracking with statistics and TPS (turns per second) calculation.

## What This Example Does

1. Connects to a GoCube device
2. Tracks every move with timestamps
3. Calculates TPS in real-time (both average and recent)
4. Monitors phase changes during solving
5. Shows face distribution statistics
6. Provides a comprehensive summary on exit

## Features

- **Real-time TPS**: See your current solving speed
- **Recent TPS**: Shows TPS over last 5 moves (more responsive to speed changes)
- **Phase tracking**: Know when you transition between solving phases
- **Face distribution**: See which faces you turn most often
- **Move sequence**: Full move history in standard notation

## Running the Example

```bash
# From the examples/track-moves directory
go run main.go

# Or from the repository root
go run ./examples/track-moves
```

## Expected Output

```
GoCube Move Tracker
===================

Scanning for GoCube devices...
Connected to: GoCube_XXXX (Battery: 85%)

Start solving! Statistics will be shown on exit.
Press Ctrl+C to stop and see summary.

─────────────────────────────────────────────────
[  1] R    TPS: 0.00 (recent: 0.00)
[  2] U    TPS: 2.50 (recent: 2.50)
[  3] R'   TPS: 2.85 (recent: 3.12)
[  4] U'   TPS: 3.10 (recent: 3.45)

  ▶ Phase: white_cross (at move 12)

[  5] F    TPS: 3.20 (recent: 3.80)
...

  ╔═══════════════════════════════════════════╗
  ║         🎉 CUBE SOLVED! 🎉                ║
  ╚═══════════════════════════════════════════╝

  Time: 45.230s | Moves: 58 | TPS: 1.28

─────────────────────────────────────────────────

═══════════════════════════════════════
           SOLVE STATISTICS
═══════════════════════════════════════

Total moves:     58
Duration:        45.230s
Average TPS:     1.28

Face Distribution:
  R:  15 ( 25.9%) █████
  U:  12 ( 20.7%) ████
  F:   9 ( 15.5%) ███
  L:   8 ( 13.8%) ██
  D:   8 ( 13.8%) ██
  B:   6 ( 10.3%) ██

Moves: R U R' U' F B2 L' D R2 U' F' L2 B D' R U2 F'...

Final phase: solved
```

## Code Highlights

### SolveStats Structure

The example includes a `SolveStats` struct that demonstrates how to build statistics on top of the move stream:

```go
type SolveStats struct {
    StartTime    time.Time
    Moves        []gocube.Move
    PhaseTimes   map[gocube.Phase]time.Time
    FaceCounts   map[gocube.Face]int
    LastMoveTime time.Time
}
```

### Calculating TPS

```go
// Overall TPS (average)
func (s *SolveStats) TPS() float64 {
    duration := s.Duration()
    if duration == 0 {
        return 0
    }
    return float64(len(s.Moves)) / duration.Seconds()
}

// Recent TPS (last N moves)
func (s *SolveStats) RecentTPS(n int) float64 {
    // ... gets TPS over last N moves for more responsive feedback
}
```

### Real-time Move Callback

```go
cube.OnMove(func(m gocube.Move) {
    stats.RecordMove(m)

    fmt.Printf("[%3d] %-3s  TPS: %.2f (recent: %.2f)\n",
        len(stats.Moves),
        m.Notation(),
        stats.TPS(),
        stats.RecentTPS(5),
    )
})
```

## Understanding TPS

**TPS (Turns Per Second)** is a common metric in speedcubing:

| TPS Range | Skill Level |
|-----------|-------------|
| < 1.0 | Beginner |
| 1.0 - 2.0 | Intermediate |
| 2.0 - 3.0 | Advanced |
| 3.0 - 4.0 | Expert |
| > 4.0 | World-class |

Note: TPS varies by solving phase. Cross and F2L typically have lower TPS than last layer algorithms.

## Building On This Example

You could extend this example to:

- **Save solve data**: Write moves to a file for later analysis
- **Compare solves**: Track multiple solves and show improvement
- **Detect patterns**: Find repeated move sequences
- **Generate reports**: Create detailed solve analysis

## Related Examples

- [connect](../connect/) - Basic connection without statistics
- [simulate](../simulate/) - Test algorithms without hardware
