package errx

// Stack holds captured stack frames. Full implementation will be added later.
type Stack struct {
	frames []Frame
}

// Frame represents a single stack frame.
type Frame struct {
	Function string
	File     string
	Line     int
}
