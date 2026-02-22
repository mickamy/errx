package gerr_test

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/mickamy/errx/gerr"
)

func TestFieldViolation(t *testing.T) {
	t.Parallel()

	br := gerr.FieldViolation("email", "must be valid")
	if len(br.GetFieldViolations()) != 1 {
		t.Fatalf("violations length = %d, want 1", len(br.GetFieldViolations()))
	}
	v := br.GetFieldViolations()[0]
	if v.GetField() != "email" || v.GetDescription() != "must be valid" {
		t.Errorf("got field=%q desc=%q", v.GetField(), v.GetDescription())
	}
}

func TestBadRequest(t *testing.T) {
	t.Parallel()

	br := gerr.BadRequest(
		gerr.NewFieldViolation("name", "required"),
		gerr.NewFieldViolation("age", "must be positive"),
	)
	if len(br.GetFieldViolations()) != 2 {
		t.Fatalf("violations length = %d, want 2", len(br.GetFieldViolations()))
	}
}

func TestPreconditionFailure(t *testing.T) {
	t.Parallel()

	pf := gerr.PreconditionFailure(
		gerr.NewPreconditionViolation("TOS", "user/123", "Terms not accepted"),
	)
	if len(pf.GetViolations()) != 1 {
		t.Fatalf("violations length = %d, want 1", len(pf.GetViolations()))
	}
	v := pf.GetViolations()[0]
	if v.GetType() != "TOS" || v.GetSubject() != "user/123" || v.GetDescription() != "Terms not accepted" {
		t.Errorf("got type=%q subject=%q desc=%q", v.GetType(), v.GetSubject(), v.GetDescription())
	}
}

func TestQuotaFailure(t *testing.T) {
	t.Parallel()

	qf := gerr.QuotaFailure(
		gerr.NewQuotaViolation("project:abc", "RPM limit exceeded"),
	)
	if len(qf.GetViolations()) != 1 {
		t.Fatalf("violations length = %d, want 1", len(qf.GetViolations()))
	}
	v := qf.GetViolations()[0]
	if v.GetSubject() != "project:abc" || v.GetDescription() != "RPM limit exceeded" {
		t.Errorf("got subject=%q desc=%q", v.GetSubject(), v.GetDescription())
	}
}

func TestResourceInfo(t *testing.T) {
	t.Parallel()

	ri := gerr.ResourceInfo("user", "user/123", "admin", "not found")
	if ri.GetResourceType() != "user" || ri.GetResourceName() != "user/123" {
		t.Errorf("got type=%q name=%q", ri.GetResourceType(), ri.GetResourceName())
	}
	if ri.GetOwner() != "admin" || ri.GetDescription() != "not found" {
		t.Errorf("got owner=%q desc=%q", ri.GetOwner(), ri.GetDescription())
	}
}

func TestErrorInfo(t *testing.T) {
	t.Parallel()

	ei := gerr.ErrorInfo("RATE_LIMITED", "example.com", map[string]string{"limit": "100"})
	if ei.GetReason() != "RATE_LIMITED" || ei.GetDomain() != "example.com" {
		t.Errorf("got reason=%q domain=%q", ei.GetReason(), ei.GetDomain())
	}
	if ei.GetMetadata()["limit"] != "100" {
		t.Errorf("metadata = %v", ei.GetMetadata())
	}
}

func TestRetryInfo(t *testing.T) {
	t.Parallel()

	ri := gerr.RetryInfo(5 * time.Second)
	got := ri.GetRetryDelay().AsDuration()
	if got != 5*time.Second {
		t.Errorf("retry delay = %v, want 5s", got)
	}
}

func TestDebugInfo(t *testing.T) {
	t.Parallel()

	di := gerr.DebugInfo([]string{"main.go:42", "handler.go:10"}, "something broke")
	if len(di.GetStackEntries()) != 2 {
		t.Fatalf("stack entries length = %d, want 2", len(di.GetStackEntries()))
	}
	if di.GetDetail() != "something broke" {
		t.Errorf("detail = %q", di.GetDetail())
	}
}

func TestLocalizedMessage(t *testing.T) {
	t.Parallel()

	lm := gerr.LocalizedMessage("ja", "名前は必須です")                //nolint:gosmopolitan // test i18n
	if lm.GetLocale() != "ja" || lm.GetMessage() != "名前は必須です" { //nolint:gosmopolitan // test i18n
		t.Errorf("got locale=%q message=%q", lm.GetLocale(), lm.GetMessage())
	}
}

func TestHelpers_ReturnProtoMessage(t *testing.T) {
	t.Parallel()

	// Verify all helpers return types that implement proto.Message,
	// which is required for gRPC status details.
	messages := []proto.Message{
		gerr.FieldViolation("f", "d"),
		gerr.BadRequest(),
		gerr.PreconditionFailure(),
		gerr.QuotaFailure(),
		gerr.ResourceInfo("", "", "", ""),
		gerr.ErrorInfo("", "", nil),
		gerr.RetryInfo(0),
		gerr.DebugInfo(nil, ""),
		gerr.LocalizedMessage("", ""),
	}
	for i, m := range messages {
		if m == nil {
			t.Errorf("helper %d returned nil", i)
		}
	}
}
