package gocube

// Predefined moves for convenience.
// Use these instead of constructing Move structs manually.
//
// Example:
//
//	cube.Apply(gocube.R, gocube.U, gocube.RPrime, gocube.UPrime)
var (
	// Right face moves
	R      = Move{Face: FaceR, Turn: CW}     // Right clockwise
	RPrime = Move{Face: FaceR, Turn: CCW}    // Right counter-clockwise
	R2     = Move{Face: FaceR, Turn: Double} // Right 180

	// Left face moves
	L      = Move{Face: FaceL, Turn: CW}     // Left clockwise
	LPrime = Move{Face: FaceL, Turn: CCW}    // Left counter-clockwise
	L2     = Move{Face: FaceL, Turn: Double} // Left 180

	// Up face moves
	U      = Move{Face: FaceU, Turn: CW}     // Up clockwise
	UPrime = Move{Face: FaceU, Turn: CCW}    // Up counter-clockwise
	U2     = Move{Face: FaceU, Turn: Double} // Up 180

	// Down face moves
	D      = Move{Face: FaceD, Turn: CW}     // Down clockwise
	DPrime = Move{Face: FaceD, Turn: CCW}    // Down counter-clockwise
	D2     = Move{Face: FaceD, Turn: Double} // Down 180

	// Front face moves
	F      = Move{Face: FaceF, Turn: CW}     // Front clockwise
	FPrime = Move{Face: FaceF, Turn: CCW}    // Front counter-clockwise
	F2     = Move{Face: FaceF, Turn: Double} // Front 180

	// Back face moves
	B      = Move{Face: FaceB, Turn: CW}     // Back clockwise
	BPrime = Move{Face: FaceB, Turn: CCW}    // Back counter-clockwise
	B2     = Move{Face: FaceB, Turn: Double} // Back 180
)

// Sexy move: R U R' U' - one of the most common algorithms
var SexyMove = []Move{R, U, RPrime, UPrime}

// Inverse sexy move: U R U' R'
var InverseSexyMove = []Move{U, R, UPrime, RPrime}

// T-perm algorithm
var TPerm = []Move{R, U, RPrime, UPrime, RPrime, F, R2, UPrime, RPrime, UPrime, R, U, RPrime, FPrime}
