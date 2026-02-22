package cerr_test

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/mickamy/errx"
	"github.com/mickamy/errx/cerr"
)

func TestToConnectCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   errx.Code
		want connect.Code
	}{
		{errx.NotFound, connect.CodeNotFound},
		{errx.Internal, connect.CodeInternal},
		{errx.Canceled, connect.CodeCanceled},
		{errx.InvalidArgument, connect.CodeInvalidArgument},
		{errx.DeadlineExceeded, connect.CodeDeadlineExceeded},
		{errx.AlreadyExists, connect.CodeAlreadyExists},
		{errx.PermissionDenied, connect.CodePermissionDenied},
		{errx.Unauthenticated, connect.CodeUnauthenticated},
		{errx.Unavailable, connect.CodeUnavailable},
		{errx.Code("custom"), connect.CodeUnknown},
		{errx.Code(""), connect.CodeUnknown},
	}

	for _, tt := range tests {
		t.Run(string(tt.in), func(t *testing.T) {
			t.Parallel()
			if got := cerr.ToConnectCode(tt.in); got != tt.want {
				t.Errorf("ToConnectCode(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestToErrxCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   connect.Code
		want errx.Code
	}{
		{0, ""},
		{connect.CodeNotFound, errx.NotFound},
		{connect.CodeInternal, errx.Internal},
		{connect.CodeUnauthenticated, errx.Unauthenticated},
	}

	for _, tt := range tests {
		t.Run(tt.in.String(), func(t *testing.T) {
			t.Parallel()
			if got := cerr.ToErrxCode(tt.in); got != tt.want {
				t.Errorf("ToErrxCode(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestToConnectError(t *testing.T) {
	t.Parallel()

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		if cerr.ToConnectError(nil) != nil {
			t.Error("ToConnectError(nil) should return nil")
		}
	})

	t.Run("errx error with code", func(t *testing.T) {
		t.Parallel()
		err := errx.New("not found").WithCode(errx.NotFound)
		ce := cerr.ToConnectError(err)
		if ce.Code() != connect.CodeNotFound {
			t.Errorf("code = %v, want NotFound", ce.Code())
		}
	})

	t.Run("plain error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("plain")
		ce := cerr.ToConnectError(err)
		if ce.Code() != connect.CodeUnknown {
			t.Errorf("code = %v, want Unknown", ce.Code())
		}
	})

	t.Run("with proto details", func(t *testing.T) {
		t.Parallel()
		br := &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{Field: "email", Description: "invalid"},
			},
		}
		err := errx.New("bad request").
			WithCode(errx.InvalidArgument).
			WithDetails(br)
		ce := cerr.ToConnectError(err)
		if len(ce.Details()) != 1 {
			t.Fatalf("details length = %d, want 1", len(ce.Details()))
		}
	})

	t.Run("non-proto details are ignored", func(t *testing.T) {
		t.Parallel()
		err := errx.New("fail").
			WithCode(errx.Internal).
			WithDetails("not a proto message")
		ce := cerr.ToConnectError(err)
		if len(ce.Details()) != 0 {
			t.Errorf("details length = %d, want 0", len(ce.Details()))
		}
	})
}

func TestFromConnectError(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		if cerr.FromConnectError(nil) != nil {
			t.Error("FromConnectError(nil) should return nil")
		}
	})

	t.Run("error with code", func(t *testing.T) {
		t.Parallel()
		ce := connect.NewError(connect.CodeNotFound, errors.New("user not found"))
		ex := cerr.FromConnectError(ce)
		if ex == nil {
			t.Fatal("FromConnectError should return non-nil")
		}
		if ex.Error() != "user not found" {
			t.Errorf("Error() = %q, want %q", ex.Error(), "user not found")
		}
		if ex.Code() != errx.NotFound {
			t.Errorf("Code() = %q, want %q", ex.Code(), errx.NotFound)
		}
	})

	t.Run("with details", func(t *testing.T) {
		t.Parallel()
		ce := connect.NewError(connect.CodeInvalidArgument, errors.New("bad request"))
		br := &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{Field: "email", Description: "invalid"},
			},
		}
		detail, err := connect.NewErrorDetail(br)
		if err != nil {
			t.Fatal(err)
		}
		ce.AddDetail(detail)

		ex := cerr.FromConnectError(ce)
		details := errx.DetailsOf(ex)
		if len(details) != 1 {
			t.Fatalf("details length = %d, want 1", len(details))
		}
		got, ok := details[0].(*errdetails.BadRequest)
		if !ok {
			t.Fatalf("detail type = %T, want *errdetails.BadRequest", details[0])
		}
		if got.GetFieldViolations()[0].GetField() != "email" {
			t.Errorf("field = %q, want %q", got.GetFieldViolations()[0].GetField(), "email")
		}
	})
}

func TestToConnectError_WithErrxDetails(t *testing.T) {
	t.Parallel()

	t.Run("BadRequestDetail", func(t *testing.T) {
		t.Parallel()
		err := errx.New("bad request").
			WithCode(errx.InvalidArgument).
			WithDetails(errx.FieldViolation("email", "invalid"))
		ce := cerr.ToConnectError(err)
		if len(ce.Details()) != 1 {
			t.Fatalf("details length = %d, want 1", len(ce.Details()))
		}
		v, vErr := ce.Details()[0].Value()
		if vErr != nil {
			t.Fatal(vErr)
		}
		br, ok := v.(*errdetails.BadRequest)
		if !ok {
			t.Fatalf("detail type = %T, want *errdetails.BadRequest", v)
		}
		if len(br.GetFieldViolations()) != 1 || br.GetFieldViolations()[0].GetField() != "email" {
			t.Errorf("unexpected field violation: %v", br.GetFieldViolations())
		}
	})

	t.Run("PreconditionFailureDetail", func(t *testing.T) {
		t.Parallel()
		err := errx.New("precondition failed").
			WithCode(errx.FailedPrecondition).
			WithDetails(errx.PreconditionFailure(errx.PreconditionViolation{
				Type: "TOS", Subject: "user", Description: "not accepted",
			}))
		ce := cerr.ToConnectError(err)
		if len(ce.Details()) != 1 {
			t.Fatalf("details length = %d, want 1", len(ce.Details()))
		}
		v, vErr := ce.Details()[0].Value()
		if vErr != nil {
			t.Fatal(vErr)
		}
		pf, ok := v.(*errdetails.PreconditionFailure)
		if !ok {
			t.Fatalf("detail type = %T, want *errdetails.PreconditionFailure", v)
		}
		if pf.GetViolations()[0].GetType() != "TOS" {
			t.Errorf("type = %q, want %q", pf.GetViolations()[0].GetType(), "TOS")
		}
	})

	t.Run("ResourceInfoDetail", func(t *testing.T) {
		t.Parallel()
		err := errx.New("not found").
			WithCode(errx.NotFound).
			WithDetails(errx.ResourceInfo("User", "123", "", "not found"))
		ce := cerr.ToConnectError(err)
		if len(ce.Details()) != 1 {
			t.Fatalf("details length = %d, want 1", len(ce.Details()))
		}
		v, vErr := ce.Details()[0].Value()
		if vErr != nil {
			t.Fatal(vErr)
		}
		ri, ok := v.(*errdetails.ResourceInfo)
		if !ok {
			t.Fatalf("detail type = %T, want *errdetails.ResourceInfo", v)
		}
		if ri.GetResourceType() != "User" || ri.GetResourceName() != "123" {
			t.Errorf("got %v", ri)
		}
	})

	t.Run("ErrorInfoDetail", func(t *testing.T) {
		t.Parallel()
		err := errx.New("quota exceeded").
			WithCode(errx.ResourceExhausted).
			WithDetails(errx.ErrorInfo("QUOTA_EXCEEDED", "example.com", map[string]string{"limit": "100"}))
		ce := cerr.ToConnectError(err)
		if len(ce.Details()) != 1 {
			t.Fatalf("details length = %d, want 1", len(ce.Details()))
		}
		v, vErr := ce.Details()[0].Value()
		if vErr != nil {
			t.Fatal(vErr)
		}
		ei, ok := v.(*errdetails.ErrorInfo)
		if !ok {
			t.Fatalf("detail type = %T, want *errdetails.ErrorInfo", v)
		}
		if ei.GetReason() != "QUOTA_EXCEEDED" || ei.GetMetadata()["limit"] != "100" {
			t.Errorf("got %v", ei)
		}
	})

	t.Run("mixed errx and proto details", func(t *testing.T) {
		t.Parallel()
		err := errx.New("bad request").
			WithCode(errx.InvalidArgument).
			WithDetails(
				errx.FieldViolation("name", "required"),
				&errdetails.LocalizedMessage{Locale: "en", Message: "Name is required"},
			)
		ce := cerr.ToConnectError(err)
		if len(ce.Details()) != 2 {
			t.Fatalf("details length = %d, want 2", len(ce.Details()))
		}
	})
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	br := &errdetails.BadRequest{
		FieldViolations: []*errdetails.BadRequest_FieldViolation{
			{Field: "name", Description: "required"},
		},
	}
	original := errx.New("bad request").
		WithCode(errx.InvalidArgument).
		WithDetails(br)

	ce := cerr.ToConnectError(original)
	recovered := cerr.FromConnectError(ce)

	if recovered.Code() != errx.InvalidArgument {
		t.Errorf("code = %q, want %q", recovered.Code(), errx.InvalidArgument)
	}

	details := errx.DetailsOf(recovered)
	if len(details) != 1 {
		t.Fatalf("details length = %d, want 1", len(details))
	}
	got, ok := details[0].(*errdetails.BadRequest)
	if !ok {
		t.Fatalf("detail type = %T, want *errdetails.BadRequest", details[0])
	}
	if got.GetFieldViolations()[0].GetField() != "name" {
		t.Errorf("field = %q, want %q", got.GetFieldViolations()[0].GetField(), "name")
	}
}
