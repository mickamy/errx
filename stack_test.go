package errx_test

import (
	"strings"
	"testing"

	"github.com/mickamy/errx"
)

func TestWithStack(t *testing.T) {
	t.Parallel()

	err := errx.New("fail").WithStack()
	s := errx.StackOf(err)
	if s == nil {
		t.Fatal("StackOf should return non-nil Stack")
	}

	frames := s.Frames()
	if len(frames) == 0 {
		t.Fatal("expected at least one frame")
	}

	// The top frame should be in this test function.
	top := frames[0]
	if !strings.Contains(top.Function, "TestWithStack") {
		t.Errorf("top frame function = %q, want containing %q", top.Function, "TestWithStack")
	}
	if !strings.HasSuffix(top.File, "stack_test.go") {
		t.Errorf("top frame file = %q, want ending with %q", top.File, "stack_test.go")
	}
	if top.Line == 0 {
		t.Error("top frame line should not be 0")
	}
}

func TestStackOf_Chain(t *testing.T) {
	t.Parallel()

	inner := errx.New("inner").WithStack()
	outer := errx.Wrap(inner, "key", "val")

	s := errx.StackOf(outer)
	if s == nil {
		t.Fatal("StackOf should find stack in the chain")
	}

	frames := s.Frames()
	if len(frames) == 0 {
		t.Fatal("expected at least one frame")
	}
}

func TestStackOf_Nil(t *testing.T) {
	t.Parallel()

	if errx.StackOf(nil) != nil {
		t.Error("StackOf(nil) should return nil")
	}
}

func TestStackOf_NoStack(t *testing.T) {
	t.Parallel()

	err := errx.New("no stack")
	if errx.StackOf(err) != nil {
		t.Error("StackOf should return nil when no stack is captured")
	}
}

func TestStack_Frames_Nil(t *testing.T) {
	t.Parallel()

	var s *errx.Stack
	if s.Frames() != nil {
		t.Error("nil Stack.Frames() should return nil")
	}
}
