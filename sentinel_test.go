package errx_test

import (
	"errors"
	"testing"

	"github.com/mickamy/errx"
)

func TestSentinel_Error(t *testing.T) {
	t.Parallel()

	s := errx.NewSentinel("not found", errx.NotFound)
	if s.Error() != "not found" {
		t.Errorf("Error() = %q, want %q", s.Error(), "not found")
	}
}

func TestSentinel_Code(t *testing.T) {
	t.Parallel()

	s := errx.NewSentinel("denied", errx.PermissionDenied)
	if s.Code() != errx.PermissionDenied {
		t.Errorf("Code() = %q, want %q", s.Code(), errx.PermissionDenied)
	}
}

func TestSentinel_ErrorsIs(t *testing.T) {
	t.Parallel()

	s := errx.NewSentinel("not found", errx.NotFound)

	t.Run("direct match", func(t *testing.T) {
		t.Parallel()
		if !errors.Is(s, s) {
			t.Error("errors.Is should match sentinel with itself")
		}
	})

	t.Run("wrapped match", func(t *testing.T) {
		t.Parallel()
		err := errx.Wrap(s, "user_id", 42)
		if !errors.Is(err, s) {
			t.Error("errors.Is should find sentinel through Wrap")
		}
	})

	t.Run("double wrapped match", func(t *testing.T) {
		t.Parallel()
		err := errx.Wrapf(errx.Wrap(s), "outer %s", "context")
		if !errors.Is(err, s) {
			t.Error("errors.Is should find sentinel through multiple wraps")
		}
	})
}

func TestSentinel_CodeOf(t *testing.T) {
	t.Parallel()

	s := errx.NewSentinel("conflict", errx.AlreadyExists)
	err := errx.Wrap(s, "table", "users")

	if errx.CodeOf(err) != errx.AlreadyExists {
		t.Errorf("CodeOf() = %q, want %q", errx.CodeOf(err), errx.AlreadyExists)
	}
}

func TestSentinel_CodeOverride(t *testing.T) {
	t.Parallel()

	s := errx.NewSentinel("base error", errx.Internal)
	err := errx.Wrap(s).WithCode(errx.Unavailable)

	if err.Code() != errx.Unavailable {
		t.Errorf("Code() = %q, want %q (outer should override)", err.Code(), errx.Unavailable)
	}
}
