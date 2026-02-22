package errx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mickamy/errx"
)

func TestCode_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code errx.Code
		want string
	}{
		{errx.NotFound, "not_found"},
		{errx.Internal, "internal"},
		{errx.Code("custom_code"), "custom_code"},
		{errx.Code(""), ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.code.String(); got != tt.want {
				t.Errorf("Code.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCode_Code(t *testing.T) {
	t.Parallel()

	code := errx.NotFound
	if got := code.Code(); got != errx.NotFound {
		t.Errorf("Code.Code() = %q, want %q", got, errx.NotFound)
	}
}

// coderError is a minimal Coder implementation for testing CodeOf/IsCode.
type coderError struct {
	code errx.Code
	msg  string
}

func (e *coderError) Error() string   { return e.msg }
func (e *coderError) Code() errx.Code { return e.code }

func TestCodeOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want errx.Code
	}{
		{
			name: "direct coder",
			err:  &coderError{code: errx.NotFound, msg: "not found"},
			want: errx.NotFound,
		},
		{
			name: "wrapped coder",
			err:  fmt.Errorf("outer: %w", &coderError{code: errx.Internal, msg: "boom"}),
			want: errx.Internal,
		},
		{
			name: "no coder in chain",
			err:  errors.New("plain error"),
			want: "",
		},
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
		{
			name: "custom code",
			err:  &coderError{code: errx.Code("payment_required"), msg: "pay up"},
			want: errx.Code("payment_required"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := errx.CodeOf(tt.err); got != tt.want {
				t.Errorf("CodeOf() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsCode(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("wrap: %w", &coderError{code: errx.PermissionDenied, msg: "denied"})

	if !errx.IsCode(err, errx.PermissionDenied) {
		t.Error("IsCode should return true for matching code")
	}
	if errx.IsCode(err, errx.NotFound) {
		t.Error("IsCode should return false for non-matching code")
	}
	if errx.IsCode(nil, errx.NotFound) {
		t.Error("IsCode should return false for nil error")
	}
}
