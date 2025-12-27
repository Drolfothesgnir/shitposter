package scum

// Warning describes the problem occured during the tokenizing or the parsing processes.
type Warning struct {

	// Issue defines the type of the problem.
	Issue Issue

	// Pos defines the byte position in the input string at which the problem occured.
	Pos int

	// Description is a human-readable story of what went wrong.
	Description string
}
