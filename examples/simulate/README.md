# Simulate Example

Standalone cube simulation without any BLE hardware required.

## What This Example Does

Demonstrates the full cube simulation API:

1. Creating and manipulating a virtual cube
2. Applying moves using predefined constants
3. Parsing moves from notation strings
4. Checking solving phases and progress
5. Cloning cube state
6. ASCII visualization

## Why Use Simulation?

- **No hardware needed**: Test algorithms anywhere
- **Faster iteration**: Instant state changes
- **Reproducible**: Same scramble gives same result
- **Visualization**: Build cube display applications
- **Algorithm development**: Test solving algorithms

## Running the Example

```bash
# From the examples/simulate directory
go run main.go

# Or from the repository root
go run ./examples/simulate
```

## Example Output

```
GoCube Simulation Example
=========================

PART 1: Basic Operations
────────────────────────

New cube created (solved state)
  IsSolved: true
  Phase: solved

Applying moves: R U R' U'
  IsSolved: false
  Phase: scrambled

PART 2: The Sexy Move (R U R' U')
──────────────────────────────────

Applying (R U R' U') x 6 should return to solved...
  After 1 iteration(s): IsSolved=false
  After 2 iteration(s): IsSolved=false
  After 3 iteration(s): IsSolved=false
  After 4 iteration(s): IsSolved=false
  After 5 iteration(s): IsSolved=false
  After 6 iteration(s): IsSolved=true

...
```

## Code Examples

### Creating a Cube

```go
// Create a solved cube
cube := gocube.NewCube()

// Check state
fmt.Println(cube.IsSolved()) // true
fmt.Println(cube.Phase())    // solved
```

### Applying Moves

```go
// Using predefined constants (recommended)
cube.Apply(gocube.R, gocube.U, gocube.RPrime, gocube.UPrime)

// Using notation string
cube.ApplyNotation("F R U' R' U' R U R' F'")

// Multiple moves at once
cube.Apply(gocube.R, gocube.R, gocube.R, gocube.R) // Returns to solved
```

### Move Constants

All 18 standard moves are available:

```go
// Right face
gocube.R       // R  (clockwise)
gocube.RPrime  // R' (counter-clockwise)
gocube.R2      // R2 (half turn)

// Left face
gocube.L, gocube.LPrime, gocube.L2

// Up face
gocube.U, gocube.UPrime, gocube.U2

// Down face
gocube.D, gocube.DPrime, gocube.D2

// Front face
gocube.F, gocube.FPrime, gocube.F2

// Back face
gocube.B, gocube.BPrime, gocube.B2
```

### Parsing Moves

```go
// Parse single move
move, err := gocube.ParseMove("R'")
if err == nil {
    fmt.Println(move.Notation()) // R'
    fmt.Println(move.Face)       // R
    fmt.Println(move.Turn)       // -1 (CCW)
}

// Parse sequence
moves, err := gocube.ParseMoves("R U R' U'")
if err == nil {
    for _, m := range moves {
        fmt.Println(m.Notation())
    }
}
```

### Getting Inverse Moves

```go
move := gocube.R
inverse := move.Inverse()

fmt.Println(move.Notation())    // R
fmt.Println(inverse.Notation()) // R'

// R2 is its own inverse
move2 := gocube.R2
fmt.Println(move2.Inverse().Notation()) // R2
```

### Checking Progress

```go
progress := cube.GetProgress()

fmt.Println("White Cross:", progress.WhiteCross)
fmt.Println("First Layer:", progress.FirstLayer)
fmt.Println("Second Layer:", progress.SecondLayer)
fmt.Println("Yellow Cross:", progress.YellowCross)
fmt.Println("Yellow Corners:", progress.YellowCorners)
fmt.Println("Yellow Oriented:", progress.YellowOriented)
fmt.Println("Solved:", progress.Solved)
```

### Cloning

```go
original := gocube.NewCube()
original.Apply(gocube.R, gocube.U)

// Create independent copy
clone := original.Clone()

// Modifying clone doesn't affect original
clone.Reset()
fmt.Println(original.IsSolved()) // false
fmt.Println(clone.IsSolved())    // true
```

### Visualization

```go
cube := gocube.NewCube()
cube.ApplyNotation("R U R' U'")

// ASCII art visualization
fmt.Println(cube.String())

// Output:
//       W W W
//       W W W
//       W W W
// O O O G G G R R R B B B
// O O O G G G R R R B B B
// O O O G G G R R R B B B
//       Y Y Y
//       Y Y Y
//       Y Y Y
```

## Use Cases

### Testing Algorithms

```go
func testAlgorithm(alg string) bool {
    cube := gocube.NewCube()

    // Apply algorithm 6 times - should return to solved for valid triggers
    for i := 0; i < 6; i++ {
        cube.ApplyNotation(alg)
    }

    return cube.IsSolved()
}

// Test sexy move
fmt.Println(testAlgorithm("R U R' U'")) // true
```

### Scramble Verification

```go
func verifyScramble(scramble string) bool {
    cube := gocube.NewCube()

    // Apply scramble
    cube.ApplyNotation(scramble)
    if cube.IsSolved() {
        return false // Invalid scramble
    }

    // Reverse should solve
    moves, _ := gocube.ParseMoves(scramble)
    for i := len(moves) - 1; i >= 0; i-- {
        cube.Apply(moves[i].Inverse())
    }

    return cube.IsSolved()
}
```

### Building a Solver

```go
func solve(cube *gocube.Cube) []gocube.Move {
    // Clone to avoid modifying original
    work := cube.Clone()

    var solution []gocube.Move

    // Your solving logic here...
    // Check work.Phase() to track progress
    // Use work.GetProgress() for detailed state

    return solution
}
```

## Related Examples

- [connect](../connect/) - Connect to a real GoCube
- [track-moves](../track-moves/) - Real-time move tracking
