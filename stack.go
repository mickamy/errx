package errx

import (
	"errors"
	"runtime"
	"strings"
)

// Stack holds captured stack frames.
type Stack struct {
	frames []Frame
}

// Frames returns the captured stack frames.
func (s *Stack) Frames() []Frame {
	if s == nil {
		return nil
	}
	cp := make([]Frame, len(s.frames))
	copy(cp, s.frames)
	return cp
}

// Frame represents a single stack frame.
type Frame struct {
	Function string
	File     string
	Line     int
}

// WithStack returns a copy of the error with a captured stack trace.
func (e *Error) WithStack() *Error {
	cp := *e
	cp.stack = captureStack(2) // skip captureStack and WithStack
	return &cp
}

// StackOf walks the error chain and returns the first Stack found, or nil.
func StackOf(err error) *Stack {
	for err != nil {
		var ex *Error
		if errors.As(err, &ex) {
			if ex.stack != nil {
				return ex.stack
			}
			err = ex.cause
		} else {
			break
		}
	}
	return nil
}

// captureStack captures the call stack, skipping the given number of frames
// (callers above captureStack itself).
func captureStack(skip int) *Stack {
	var pcs [32]uintptr
	n := runtime.Callers(skip+1, pcs[:]) // +1 for runtime.Callers itself
	if n == 0 {
		return &Stack{}
	}

	rframes := runtime.CallersFrames(pcs[:n])
	frames := make([]Frame, 0, n)
	for {
		f, more := rframes.Next()
		// Skip runtime internals.
		if strings.HasPrefix(f.Function, "runtime.") {
			if !more {
				break
			}
			continue
		}
		frames = append(frames, Frame{
			Function: f.Function,
			File:     f.File,
			Line:     f.Line,
		})
		if !more {
			break
		}
	}
	return &Stack{frames: frames}
}
