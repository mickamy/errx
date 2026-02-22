package errx_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/mickamy/errx"
)

func TestNew(t *testing.T) {
	t.Parallel()

	err := errx.New("something failed", "user_id", 42)
	if err.Error() != "something failed" {
		t.Errorf("Error() = %q, want %q", err.Error(), "something failed")
	}

	fields := errx.Fields(err)
	if len(fields) != 1 {
		t.Fatalf("Fields length = %d, want 1", len(fields))
	}
	if fields[0].Key != "user_id" {
		t.Errorf("field key = %q, want %q", fields[0].Key, "user_id")
	}
}

func TestWrap(t *testing.T) {
	t.Parallel()

	t.Run("wraps error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("db timeout")
		err := errx.Wrap(cause, "query", "SELECT 1")
		if err.Error() != "db timeout" {
			t.Errorf("Error() = %q, want %q", err.Error(), "db timeout")
		}
		if !errors.Is(err, cause) {
			t.Error("errors.Is should find cause")
		}
		fields := errx.Fields(err)
		if len(fields) != 1 || fields[0].Key != "query" {
			t.Errorf("unexpected fields: %v", fields)
		}
	})

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		if errx.Wrap(nil) != nil {
			t.Error("Wrap(nil) should return nil")
		}
	})
}

func TestWrapf(t *testing.T) {
	t.Parallel()

	t.Run("formats message", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("connection refused")
		err := errx.Wrapf(cause, "connect to %s", "localhost:5432")
		want := "connect to localhost:5432: connection refused"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
		if !errors.Is(err, cause) {
			t.Error("errors.Is should find cause")
		}
	})

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		if errx.Wrapf(nil, "msg") != nil {
			t.Error("Wrapf(nil) should return nil")
		}
	})
}

func TestWith(t *testing.T) {
	t.Parallel()

	original := errx.New("fail", "a", 1)
	extended := original.With("b", 2)

	origFields := errx.Fields(original)
	extFields := errx.Fields(extended)

	if len(origFields) != 1 {
		t.Errorf("original should have 1 field, got %d", len(origFields))
	}
	if len(extFields) != 2 {
		t.Errorf("extended should have 2 fields, got %d", len(extFields))
	}
}

func TestWithCode(t *testing.T) {
	t.Parallel()

	err := errx.New("fail").WithCode(errx.NotFound)
	if err.Code() != errx.NotFound {
		t.Errorf("Code() = %q, want %q", err.Code(), errx.NotFound)
	}
}

func TestError_ErrorString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *errx.Error
		want string
	}{
		{
			name: "message only",
			err:  errx.New("boom"),
			want: "boom",
		},
		{
			name: "cause only",
			err:  errx.Wrap(errors.New("root")),
			want: "root",
		},
		{
			name: "message and cause",
			err:  errx.Wrapf(errors.New("root"), "context"),
			want: "context: root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("root cause")
	err := errx.Wrap(cause)
	if !errors.Is(err.Unwrap(), cause) {
		t.Error("Unwrap should return the cause")
	}
}

func TestErrorsIs(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("sentinel")
	err := errx.Wrapf(sentinel, "wrapping")
	if !errors.Is(err, sentinel) {
		t.Error("errors.Is should find sentinel through errx.Error")
	}
}

func TestErrorsAs(t *testing.T) {
	t.Parallel()

	inner := errx.New("inner").WithCode(errx.Internal)
	outer := errx.Wrapf(inner, "outer")

	var target *errx.Error
	if !errors.As(outer, &target) {
		t.Fatal("errors.As should find *errx.Error")
	}
	if target.Error() != "outer: inner" {
		t.Errorf("target.Error() = %q", target.Error())
	}
}

func TestCode_ChainLookup(t *testing.T) {
	t.Parallel()

	t.Run("outer code wins", func(t *testing.T) {
		t.Parallel()
		inner := errx.New("inner").WithCode(errx.Internal)
		outer := errx.Wrap(inner).WithCode(errx.NotFound)
		if outer.Code() != errx.NotFound {
			t.Errorf("Code() = %q, want %q", outer.Code(), errx.NotFound)
		}
	})

	t.Run("falls through to inner", func(t *testing.T) {
		t.Parallel()
		inner := errx.New("inner").WithCode(errx.Internal)
		outer := errx.Wrap(inner)
		if outer.Code() != errx.Internal {
			t.Errorf("Code() = %q, want %q", outer.Code(), errx.Internal)
		}
	})

	t.Run("no code returns zero", func(t *testing.T) {
		t.Parallel()
		err := errx.New("plain")
		if err.Code() != "" {
			t.Errorf("Code() = %q, want empty", err.Code())
		}
	})
}

func TestFields_Chain(t *testing.T) {
	t.Parallel()

	inner := errx.New("inner", "key1", "val1")
	outer := errx.Wrap(inner, "key2", "val2")

	fields := errx.Fields(outer)
	if len(fields) != 2 {
		t.Fatalf("Fields length = %d, want 2", len(fields))
	}
	if fields[0].Key != "key2" {
		t.Errorf("first field key = %q, want %q", fields[0].Key, "key2")
	}
	if fields[1].Key != "key1" {
		t.Errorf("second field key = %q, want %q", fields[1].Key, "key1")
	}
}

func TestFields_NilError(t *testing.T) {
	t.Parallel()

	fields := errx.Fields(nil)
	if len(fields) != 0 {
		t.Errorf("Fields(nil) should be empty, got %d", len(fields))
	}
}

func TestFields_PlainError(t *testing.T) {
	t.Parallel()

	fields := errx.Fields(errors.New("plain"))
	if len(fields) != 0 {
		t.Errorf("Fields on plain error should be empty, got %d", len(fields))
	}
}

func TestArgsToAttrs_SlogAttr(t *testing.T) {
	t.Parallel()

	err := errx.New("msg", slog.String("key", "value"))
	fields := errx.Fields(err)
	if len(fields) != 1 || fields[0].Key != "key" {
		t.Errorf("unexpected fields: %v", fields)
	}
}

func TestArgsToAttrs_BadKey(t *testing.T) {
	t.Parallel()

	t.Run("lone string key", func(t *testing.T) {
		t.Parallel()
		err := errx.New("msg", "orphan")
		fields := errx.Fields(err)
		if len(fields) != 1 || fields[0].Key != "!BADKEY" {
			t.Errorf("unexpected fields: %v", fields)
		}
	})

	t.Run("non-string key", func(t *testing.T) {
		t.Parallel()
		err := errx.New("msg", 42)
		fields := errx.Fields(err)
		if len(fields) != 1 || fields[0].Key != "!BADKEY" {
			t.Errorf("unexpected fields: %v", fields)
		}
	})
}
