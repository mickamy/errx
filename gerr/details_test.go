package gerr_test

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/mickamy/errx/gerr"
)

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

	messages := []proto.Message{
		gerr.QuotaFailure(),
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
