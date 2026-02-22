package gerr_test

import (
	"errors"
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mickamy/errx"
	"github.com/mickamy/errx/gerr"
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
			if got := gerr.ToGRPCCode(tt.in); got != tt.want {
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
			if got := gerr.ToErrxCode(tt.in); got != tt.want {
				t.Errorf("ToErrxCode(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestToStatus(t *testing.T) {
	t.Parallel()

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		st := gerr.ToStatus(nil)
		if st.Code() != codes.OK {
			t.Errorf("ToStatus(nil) code = %v, want OK", st.Code())
		}
	})

	t.Run("errx error with code", func(t *testing.T) {
		t.Parallel()
		err := errx.New("not found").WithCode(errx.NotFound)
		st := gerr.ToStatus(err)
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
		st := gerr.ToStatus(err)
		if st.Code() != codes.Unknown {
			t.Errorf("code = %v, want Unknown", st.Code())
		}
	})

	t.Run("wrapped errx", func(t *testing.T) {
		t.Parallel()
		inner := errx.New("db error").WithCode(errx.Internal)
		outer := errx.Wrapf(inner, "query failed")
		st := gerr.ToStatus(outer)
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
		if gerr.FromStatus(st) != nil {
			t.Error("FromStatus(OK) should return nil")
		}
	})

	t.Run("error status", func(t *testing.T) {
		t.Parallel()
		st := status.New(codes.NotFound, "user not found")
		err := gerr.FromStatus(st)
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

func TestToStatus_WithDetails(t *testing.T) {
	t.Parallel()

	t.Run("proto.Message details are included", func(t *testing.T) {
		t.Parallel()
		err := errx.New("bad request").
			WithCode(errx.InvalidArgument).
			WithDetails(gerr.FieldViolation("email", "invalid"))
		st := gerr.ToStatus(err)
		if st.Code() != codes.InvalidArgument {
			t.Errorf("code = %v, want InvalidArgument", st.Code())
		}
		details := st.Details()
		if len(details) != 1 {
			t.Fatalf("details length = %d, want 1", len(details))
		}
		br, ok := details[0].(*errdetails.BadRequest)
		if !ok {
			t.Fatalf("detail type = %T, want *errdetails.BadRequest", details[0])
		}
		if len(br.GetFieldViolations()) != 1 || br.GetFieldViolations()[0].GetField() != "email" {
			t.Errorf("unexpected field violation: %v", br.GetFieldViolations())
		}
	})

	t.Run("non-proto details are ignored", func(t *testing.T) {
		t.Parallel()
		err := errx.New("fail").
			WithCode(errx.Internal).
			WithDetails("not a proto message")
		st := gerr.ToStatus(err)
		if len(st.Details()) != 0 {
			t.Errorf("details length = %d, want 0", len(st.Details()))
		}
	})

	t.Run("multiple details from chain", func(t *testing.T) {
		t.Parallel()
		inner := errx.New("inner").
			WithCode(errx.InvalidArgument).
			WithDetails(gerr.FieldViolation("name", "required"))
		outer := errx.Wrap(inner).
			WithDetails(gerr.FieldViolation("email", "invalid"))
		st := gerr.ToStatus(outer)
		if len(st.Details()) != 2 {
			t.Fatalf("details length = %d, want 2", len(st.Details()))
		}
	})
}

func TestFromStatus_WithDetails(t *testing.T) {
	t.Parallel()

	t.Run("details are restored", func(t *testing.T) {
		t.Parallel()
		st, err := status.New(codes.InvalidArgument, "bad request").
			WithDetails(gerr.FieldViolation("email", "invalid"))
		if err != nil {
			t.Fatal(err)
		}
		ex := gerr.FromStatus(st)
		details := errx.DetailsOf(ex)
		if len(details) != 1 {
			t.Fatalf("details length = %d, want 1", len(details))
		}
		br, ok := details[0].(*errdetails.BadRequest)
		if !ok {
			t.Fatalf("detail type = %T, want *errdetails.BadRequest", details[0])
		}
		if br.GetFieldViolations()[0].GetField() != "email" {
			t.Errorf("field = %q, want %q", br.GetFieldViolations()[0].GetField(), "email")
		}
	})

	t.Run("no details", func(t *testing.T) {
		t.Parallel()
		st := status.New(codes.NotFound, "not found")
		ex := gerr.FromStatus(st)
		details := errx.DetailsOf(ex)
		if len(details) != 0 {
			t.Errorf("details length = %d, want 0", len(details))
		}
	})
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	original := errx.New("permission denied", "resource", "doc-123").
		WithCode(errx.PermissionDenied)

	st := gerr.ToStatus(original)
	recovered := gerr.FromStatus(st)

	if recovered.Code() != errx.PermissionDenied {
		t.Errorf("round-trip code = %q, want %q", recovered.Code(), errx.PermissionDenied)
	}
	if recovered.Error() != original.Error() {
		t.Errorf("round-trip message = %q, want %q", recovered.Error(), original.Error())
	}
}

func TestRoundTrip_WithDetails(t *testing.T) {
	t.Parallel()

	original := errx.New("bad request").
		WithCode(errx.InvalidArgument).
		WithDetails(
			gerr.FieldViolation("email", "invalid"),
			gerr.LocalizedMessage("en", "Email is invalid"),
		)

	st := gerr.ToStatus(original)
	recovered := gerr.FromStatus(st)

	if recovered.Code() != errx.InvalidArgument {
		t.Errorf("code = %q, want %q", recovered.Code(), errx.InvalidArgument)
	}

	details := errx.DetailsOf(recovered)
	if len(details) != 2 {
		t.Fatalf("details length = %d, want 2", len(details))
	}

	if _, ok := details[0].(*errdetails.BadRequest); !ok {
		t.Errorf("detail[0] type = %T, want *errdetails.BadRequest", details[0])
	}
	if lm, ok := details[1].(*errdetails.LocalizedMessage); !ok {
		t.Errorf("detail[1] type = %T, want *errdetails.LocalizedMessage", details[1])
	} else if lm.GetMessage() != "Email is invalid" {
		t.Errorf("localized message = %q, want %q", lm.GetMessage(), "Email is invalid")
	}
}
