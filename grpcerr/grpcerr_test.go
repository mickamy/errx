package grpcerr_test

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mickamy/errx"
	"github.com/mickamy/errx/grpcerr"
)

func TestToGRPCCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   errx.Code
		want codes.Code
	}{
		{errx.NotFound, codes.NotFound},
		{errx.Internal, codes.Internal},
		{errx.Canceled, codes.Canceled},
		{errx.InvalidArgument, codes.InvalidArgument},
		{errx.DeadlineExceeded, codes.DeadlineExceeded},
		{errx.AlreadyExists, codes.AlreadyExists},
		{errx.PermissionDenied, codes.PermissionDenied},
		{errx.Unauthenticated, codes.Unauthenticated},
		{errx.Unavailable, codes.Unavailable},
		{errx.Code("custom"), codes.Unknown},
		{errx.Code(""), codes.Unknown},
	}

	for _, tt := range tests {
		t.Run(string(tt.in), func(t *testing.T) {
			t.Parallel()
			if got := grpcerr.ToGRPCCode(tt.in); got != tt.want {
				t.Errorf("ToGRPCCode(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestToErrxCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   codes.Code
		want errx.Code
	}{
		{codes.OK, ""},
		{codes.NotFound, errx.NotFound},
		{codes.Internal, errx.Internal},
		{codes.Unauthenticated, errx.Unauthenticated},
	}

	for _, tt := range tests {
		t.Run(tt.in.String(), func(t *testing.T) {
			t.Parallel()
			if got := grpcerr.ToErrxCode(tt.in); got != tt.want {
				t.Errorf("ToErrxCode(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestToStatus(t *testing.T) {
	t.Parallel()

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		st := grpcerr.ToStatus(nil)
		if st.Code() != codes.OK {
			t.Errorf("ToStatus(nil) code = %v, want OK", st.Code())
		}
	})

	t.Run("errx error with code", func(t *testing.T) {
		t.Parallel()
		err := errx.New("not found").WithCode(errx.NotFound)
		st := grpcerr.ToStatus(err)
		if st.Code() != codes.NotFound {
			t.Errorf("code = %v, want NotFound", st.Code())
		}
		if st.Message() != "not found" {
			t.Errorf("message = %q, want %q", st.Message(), "not found")
		}
	})

	t.Run("plain error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("plain")
		st := grpcerr.ToStatus(err)
		if st.Code() != codes.Unknown {
			t.Errorf("code = %v, want Unknown", st.Code())
		}
	})

	t.Run("wrapped errx", func(t *testing.T) {
		t.Parallel()
		inner := errx.New("db error").WithCode(errx.Internal)
		outer := errx.Wrapf(inner, "query failed")
		st := grpcerr.ToStatus(outer)
		if st.Code() != codes.Internal {
			t.Errorf("code = %v, want Internal", st.Code())
		}
	})
}

func TestFromStatus(t *testing.T) {
	t.Parallel()

	t.Run("OK returns nil", func(t *testing.T) {
		t.Parallel()
		st := status.New(codes.OK, "")
		if grpcerr.FromStatus(st) != nil {
			t.Error("FromStatus(OK) should return nil")
		}
	})

	t.Run("error status", func(t *testing.T) {
		t.Parallel()
		st := status.New(codes.NotFound, "user not found")
		err := grpcerr.FromStatus(st)
		if err == nil {
			t.Fatal("FromStatus should return non-nil")
		}
		if err.Error() != "user not found" {
			t.Errorf("Error() = %q, want %q", err.Error(), "user not found")
		}
		if err.Code() != errx.NotFound {
			t.Errorf("Code() = %q, want %q", err.Code(), errx.NotFound)
		}
	})
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	original := errx.New("permission denied", "resource", "doc-123").
		WithCode(errx.PermissionDenied)

	st := grpcerr.ToStatus(original)
	recovered := grpcerr.FromStatus(st)

	if recovered.Code() != errx.PermissionDenied {
		t.Errorf("round-trip code = %q, want %q", recovered.Code(), errx.PermissionDenied)
	}
	if recovered.Error() != original.Error() {
		t.Errorf("round-trip message = %q, want %q", recovered.Error(), original.Error())
	}
}
