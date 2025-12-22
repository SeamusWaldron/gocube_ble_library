# Connect Example

Basic example demonstrating how to connect to a GoCube smart cube via Bluetooth Low Energy.

## What This Example Does

1. Scans for nearby GoCube devices
2. Connects to the first device found
3. Sets up event handlers for:
   - Moves (every face rotation)
   - Phase changes (solving progress)
   - Solved state
   - Battery updates
   - Disconnection
4. Displays real-time move information
5. Gracefully handles Ctrl+C shutdown

## Prerequisites

- macOS (BLE functionality is currently macOS-only)
- Go 1.22+
- A GoCube smart cube (tested with GoCube Edge)

## Before Running

1. **Disconnect from phone**: Go to your phone's Bluetooth settings and "Forget" the GoCube device
2. **Wake the cube**: Rotate any face to wake it from sleep
3. **Stay in range**: Keep the cube within Bluetooth range (~10m)

## Running the Example

```bash
# From the examples/connect directory
go run main.go

# Or from the repository root
go run ./examples/connect
```

## Expected Output

```
GoCube Connect Example
======================

Scanning for GoCube devices...
(Make sure your cube is awake and not connected to another device)

Connected to: GoCube_XXXX
Battery: 85%

Cube is ready! Try making some moves...
Press Ctrl+C to disconnect and exit.

  Move: R
  Move: U
  Move: R'
  Move: U'
  Phase: scrambled
  Move: F
  ...
```

## Code Walkthrough

### Connecting to a Cube

The simplest way to connect is using `ConnectFirst`:

```go
cube, err := gocube.ConnectFirst(ctx)
if err != nil {
    // Handle error
}
defer cube.Close()
```

For more control, scan and connect separately:

```go
// Scan with timeout
devices, err := gocube.Scan(ctx, 5*time.Second)
if err != nil {
    // Handle error
}

// Connect to a specific device
cube, err := gocube.Connect(ctx, devices[0])
```

### Event Callbacks

Set up callbacks to react to cube events:

```go
// Called on every move
cube.OnMove(func(m gocube.Move) {
    fmt.Printf("Move: %s\n", m.Notation())
})

// Called when solving phase changes
cube.OnPhaseChange(func(p gocube.Phase) {
    fmt.Printf("Phase: %s\n", p.String())
})

// Called when cube is solved
cube.OnSolved(func() {
    fmt.Println("Solved!")
})
```

### Accessing Cube State

Query the current state at any time:

```go
cube.IsSolved()     // true if cube is solved
cube.Phase()        // Current solving phase
cube.Battery()      // Battery percentage
cube.Moves()        // All moves since connection
cube.Cube()         // Access the internal Cube state
```

## Troubleshooting

### "No GoCube devices found"

1. Make sure the cube is disconnected from your phone
2. Wake the cube by rotating it
3. Try running the scan again (macOS BLE sometimes needs multiple attempts)
4. Move closer to the cube

### Connection drops frequently

- Check the cube's battery level
- Reduce distance between cube and computer
- Avoid Bluetooth interference from other devices

## Next Steps

- See [track-moves](../track-moves/) for detailed move tracking
- See [simulate](../simulate/) for cube simulation without BLE
