package herr_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mickamy/errx"
	"github.com/mickamy/errx/herr"
)

func TestRegisterCode(t *testing.T) {
	t.Parallel()

	custom := errx.Code("payment_required")
	herr.RegisterCode(custom, http.StatusPaymentRequired)

	t.Run("forward lookup", func(t *testing.T) {
		t.Parallel()
		if got := herr.ToHTTPStatus(custom); got != http.StatusPaymentRequired {
			t.Errorf("ToHTTPStatus(%q) = %d, want %d", custom, got, http.StatusPaymentRequired)
		}
	})

	t.Run("reverse lookup", func(t *testing.T) {
		t.Parallel()
		if got := herr.ToErrxCode(http.StatusPaymentRequired); got != custom {
			t.Errorf("ToErrxCode(%d) = %q, want %q", http.StatusPaymentRequired, got, custom)
		}
	})
}

func TestToHTTPStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   errx.Code
		want int
	}{
		{errx.InvalidArgument, http.StatusBadRequest},
		{errx.OutOfRange, http.StatusBadRequest},
		{errx.Unauthenticated, http.StatusUnauthorized},
		{errx.PermissionDenied, http.StatusForbidden},
		{errx.NotFound, http.StatusNotFound},
		{errx.AlreadyExists, http.StatusConflict},
		{errx.Aborted, http.StatusConflict},
		{errx.FailedPrecondition, http.StatusPreconditionFailed},
		{errx.ResourceExhausted, http.StatusTooManyRequests},
		{errx.Canceled, 499},
		{errx.Internal, http.StatusInternalServerError},
		{errx.Unknown, http.StatusInternalServerError},
		{errx.DataLoss, http.StatusInternalServerError},
		{errx.Unimplemented, http.StatusNotImplemented},
		{errx.Unavailable, http.StatusServiceUnavailable},
		{errx.DeadlineExceeded, http.StatusGatewayTimeout},
		{errx.Code("custom"), http.StatusInternalServerError},
		{errx.Code(""), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(string(tt.in), func(t *testing.T) {
			t.Parallel()
			if got := herr.ToHTTPStatus(tt.in); got != tt.want {
				t.Errorf("ToHTTPStatus(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestToErrxCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   int
		want errx.Code
	}{
		{http.StatusBadRequest, errx.InvalidArgument},
		{http.StatusUnauthorized, errx.Unauthenticated},
		{http.StatusForbidden, errx.PermissionDenied},
		{http.StatusNotFound, errx.NotFound},
		{http.StatusConflict, errx.AlreadyExists},
		{http.StatusPreconditionFailed, errx.FailedPrecondition},
		{http.StatusTooManyRequests, errx.ResourceExhausted},
		{499, errx.Canceled},
		{http.StatusInternalServerError, errx.Internal},
		{http.StatusNotImplemented, errx.Unimplemented},
		{http.StatusServiceUnavailable, errx.Unavailable},
		{http.StatusGatewayTimeout, errx.DeadlineExceeded},
		{http.StatusTeapot, errx.Unknown},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.in), func(t *testing.T) {
			t.Parallel()
			if got := herr.ToErrxCode(tt.in); got != tt.want {
				t.Errorf("ToErrxCode(%d) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestToProblemDetail(t *testing.T) {
	t.Parallel()

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		if herr.ToProblemDetail(nil) != nil {
			t.Error("ToProblemDetail(nil) should return nil")
		}
	})

	t.Run("errx error with code", func(t *testing.T) {
		t.Parallel()
		err := errx.New("user not found").WithCode(errx.NotFound)
		p := herr.ToProblemDetail(err)
		if p.Type != "about:blank" {
			t.Errorf("Type = %q, want %q", p.Type, "about:blank")
		}
		if p.Title != "Not Found" {
			t.Errorf("Title = %q, want %q", p.Title, "Not Found")
		}
		if p.Status != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", p.Status, http.StatusNotFound)
		}
		if p.Detail != "user not found" {
			t.Errorf("Detail = %q, want %q", p.Detail, "user not found")
		}
		if p.Code != "not_found" {
			t.Errorf("Code = %q, want %q", p.Code, "not_found")
		}
	})

	t.Run("non-standard status code falls back to code string for title", func(t *testing.T) {
		t.Parallel()
		err := errx.New("client closed").WithCode(errx.Canceled)
		p := herr.ToProblemDetail(err)
		if p.Status != 499 {
			t.Errorf("Status = %d, want 499", p.Status)
		}
		if p.Title != "canceled" {
			t.Errorf("Title = %q, want %q", p.Title, "canceled")
		}
	})

	t.Run("plain error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("something went wrong")
		p := herr.ToProblemDetail(err)
		if p.Code != "unknown" {
			t.Errorf("Code = %q, want %q", p.Code, "unknown")
		}
		if p.Status != http.StatusInternalServerError {
			t.Errorf("Status = %d, want %d", p.Status, http.StatusInternalServerError)
		}
	})

	t.Run("with WithType option", func(t *testing.T) {
		t.Parallel()
		err := errx.New("not found").WithCode(errx.NotFound)
		p := herr.ToProblemDetail(err, herr.WithType("https://example.com/not-found"))
		if p.Type != "https://example.com/not-found" {
			t.Errorf("Type = %q, want custom URI", p.Type)
		}
	})

	t.Run("with WithInstance option", func(t *testing.T) {
		t.Parallel()
		err := errx.New("not found").WithCode(errx.NotFound)
		p := herr.ToProblemDetail(err, herr.WithInstance("/users/alice"))
		if p.Instance != "/users/alice" {
			t.Errorf("Instance = %q, want %q", p.Instance, "/users/alice")
		}
	})

	t.Run("with BadRequestDetail", func(t *testing.T) {
		t.Parallel()
		err := errx.New("bad request").
			WithCode(errx.InvalidArgument).
			WithDetails(errx.FieldViolation("email", "invalid"))
		p := herr.ToProblemDetail(err)
		if len(p.Errors) != 1 {
			t.Fatalf("errors length = %d, want 1", len(p.Errors))
		}
		if p.Errors[0]["type"] != "BadRequest" {
			t.Errorf("error type = %v, want BadRequest", p.Errors[0]["type"])
		}
	})

	t.Run("with ResourceInfoDetail", func(t *testing.T) {
		t.Parallel()
		err := errx.New("not found").
			WithCode(errx.NotFound).
			WithDetails(errx.ResourceInfo("User", "123", "", "not found"))
		p := herr.ToProblemDetail(err)
		if len(p.Errors) != 1 {
			t.Fatalf("errors length = %d, want 1", len(p.Errors))
		}
		if p.Errors[0]["type"] != "ResourceInfo" {
			t.Errorf("error type = %v, want ResourceInfo", p.Errors[0]["type"])
		}
		if p.Errors[0]["resource_type"] != "User" {
			t.Errorf("resource_type = %v, want User", p.Errors[0]["resource_type"])
		}
	})

	t.Run("with PreconditionFailureDetail", func(t *testing.T) {
		t.Parallel()
		err := errx.New("precondition failed").
			WithCode(errx.FailedPrecondition).
			WithDetails(errx.PreconditionFailure(errx.PreconditionViolation{
				Type: "TOS", Subject: "user", Description: "not accepted",
			}))
		p := herr.ToProblemDetail(err)
		if len(p.Errors) != 1 {
			t.Fatalf("errors length = %d, want 1", len(p.Errors))
		}
		if p.Errors[0]["type"] != "PreconditionFailure" {
			t.Errorf("error type = %v, want PreconditionFailure", p.Errors[0]["type"])
		}
	})

	t.Run("with ErrorInfoDetail", func(t *testing.T) {
		t.Parallel()
		err := errx.New("quota exceeded").
			WithCode(errx.ResourceExhausted).
			WithDetails(errx.ErrorInfo("QUOTA_EXCEEDED", "example.com", map[string]string{"limit": "100"}))
		p := herr.ToProblemDetail(err)
		if len(p.Errors) != 1 {
			t.Fatalf("errors length = %d, want 1", len(p.Errors))
		}
		if p.Errors[0]["type"] != "ErrorInfo" {
			t.Errorf("error type = %v, want ErrorInfo", p.Errors[0]["type"])
		}
		if p.Errors[0]["reason"] != "QUOTA_EXCEEDED" {
			t.Errorf("reason = %v, want QUOTA_EXCEEDED", p.Errors[0]["reason"])
		}
	})

	t.Run("non-errx details are ignored", func(t *testing.T) {
		t.Parallel()
		err := errx.New("fail").
			WithCode(errx.Internal).
			WithDetails("not a detail type")
		p := herr.ToProblemDetail(err)
		if len(p.Errors) != 0 {
			t.Errorf("errors length = %d, want 0", len(p.Errors))
		}
	})
}

func TestWriteError(t *testing.T) {
	t.Parallel()

	t.Run("nil error does nothing", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		herr.WriteError(w, nil)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if w.Body.Len() != 0 {
			t.Errorf("body = %q, want empty", w.Body.String())
		}
	})

	t.Run("errx error", func(t *testing.T) {
		t.Parallel()
		err := errx.New("user not found").WithCode(errx.NotFound)
		w := httptest.NewRecorder()
		herr.WriteError(w, err)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
			t.Errorf("Content-Type = %q, want application/problem+json", ct)
		}

		var p herr.ProblemDetail
		if jsonErr := json.NewDecoder(w.Body).Decode(&p); jsonErr != nil {
			t.Fatal(jsonErr)
		}
		if p.Type != "about:blank" {
			t.Errorf("Type = %q, want %q", p.Type, "about:blank")
		}
		if p.Title != "Not Found" {
			t.Errorf("Title = %q, want %q", p.Title, "Not Found")
		}
		if p.Status != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", p.Status, http.StatusNotFound)
		}
		if p.Detail != "user not found" {
			t.Errorf("Detail = %q, want %q", p.Detail, "user not found")
		}
		if p.Code != "not_found" {
			t.Errorf("Code = %q, want %q", p.Code, "not_found")
		}
	})

	t.Run("with details", func(t *testing.T) {
		t.Parallel()
		err := errx.New("bad request").
			WithCode(errx.InvalidArgument).
			WithDetails(errx.FieldViolation("email", "invalid"))
		w := httptest.NewRecorder()
		herr.WriteError(w, err)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}

		var p herr.ProblemDetail
		if jsonErr := json.NewDecoder(w.Body).Decode(&p); jsonErr != nil {
			t.Fatal(jsonErr)
		}
		if len(p.Errors) != 1 {
			t.Fatalf("errors length = %d, want 1", len(p.Errors))
		}
	})

	t.Run("with options", func(t *testing.T) {
		t.Parallel()
		err := errx.New("not found").WithCode(errx.NotFound)
		w := httptest.NewRecorder()
		herr.WriteError(w, err, herr.WithInstance("/users/alice"))

		var p herr.ProblemDetail
		if jsonErr := json.NewDecoder(w.Body).Decode(&p); jsonErr != nil {
			t.Fatal(jsonErr)
		}
		if p.Instance != "/users/alice" {
			t.Errorf("Instance = %q, want %q", p.Instance, "/users/alice")
		}
	})
}

func TestFromProblemDetail(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		if herr.FromProblemDetail(nil) != nil {
			t.Error("FromProblemDetail(nil) should return nil")
		}
	})

	t.Run("restores code and message", func(t *testing.T) {
		t.Parallel()
		p := &herr.ProblemDetail{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: "user not found",
			Code:   "not_found",
		}
		err := herr.FromProblemDetail(p)
		if err == nil {
			t.Fatal("FromProblemDetail should return non-nil")
		}
		if err.Code() != errx.NotFound {
			t.Errorf("Code() = %q, want %q", err.Code(), errx.NotFound)
		}
		if err.Error() != "user not found" {
			t.Errorf("Error() = %q, want %q", err.Error(), "user not found")
		}
	})

	t.Run("falls back to status code", func(t *testing.T) {
		t.Parallel()
		p := &herr.ProblemDetail{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: "user not found",
		}
		err := herr.FromProblemDetail(p)
		if err.Code() != errx.NotFound {
			t.Errorf("Code() = %q, want %q", err.Code(), errx.NotFound)
		}
	})

	t.Run("round-trip", func(t *testing.T) {
		t.Parallel()
		original := errx.New("permission denied").WithCode(errx.PermissionDenied)
		p := herr.ToProblemDetail(original)
		recovered := herr.FromProblemDetail(p)
		if recovered.Code() != errx.PermissionDenied {
			t.Errorf("Code() = %q, want %q", recovered.Code(), errx.PermissionDenied)
		}
		if recovered.Error() != original.Error() {
			t.Errorf("Error() = %q, want %q", recovered.Error(), original.Error())
		}
	})
}
