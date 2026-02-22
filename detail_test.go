package errx_test

import (
	"testing"

	"github.com/mickamy/errx"
)

func TestFieldViolation(t *testing.T) {
	t.Parallel()

	d := errx.FieldViolation("email", "invalid format")
	if len(d.Violations) != 1 {
		t.Fatalf("violations length = %d, want 1", len(d.Violations))
	}
	v := d.Violations[0]
	if v.Field != "email" {
		t.Errorf("field = %q, want %q", v.Field, "email")
	}
	if v.Description != "invalid format" {
		t.Errorf("description = %q, want %q", v.Description, "invalid format")
	}
}

func TestBadRequest(t *testing.T) {
	t.Parallel()

	d := errx.BadRequest(
		errx.BadRequestFieldViolation{Field: "email", Description: "invalid"},
		errx.BadRequestFieldViolation{Field: "name", Description: "required"},
	)
	if len(d.Violations) != 2 {
		t.Fatalf("violations length = %d, want 2", len(d.Violations))
	}
	if d.Violations[0].Field != "email" {
		t.Errorf("violations[0].field = %q, want %q", d.Violations[0].Field, "email")
	}
	if d.Violations[1].Field != "name" {
		t.Errorf("violations[1].field = %q, want %q", d.Violations[1].Field, "name")
	}
}

func TestPreconditionFailure(t *testing.T) {
	t.Parallel()

	d := errx.PreconditionFailure(
		errx.PreconditionViolation{Type: "TOS", Subject: "user", Description: "not accepted"},
	)
	if len(d.Violations) != 1 {
		t.Fatalf("violations length = %d, want 1", len(d.Violations))
	}
	v := d.Violations[0]
	if v.Type != "TOS" || v.Subject != "user" || v.Description != "not accepted" {
		t.Errorf("got %+v", v)
	}
}

func TestResourceInfo(t *testing.T) {
	t.Parallel()

	d := errx.ResourceInfo("User", "123", "admin", "not found")
	if d.ResourceType != "User" {
		t.Errorf("resource_type = %q, want %q", d.ResourceType, "User")
	}
	if d.ResourceName != "123" {
		t.Errorf("resource_name = %q, want %q", d.ResourceName, "123")
	}
	if d.Owner != "admin" {
		t.Errorf("owner = %q, want %q", d.Owner, "admin")
	}
	if d.Description != "not found" {
		t.Errorf("description = %q, want %q", d.Description, "not found")
	}
}

func TestErrorInfo(t *testing.T) {
	t.Parallel()

	meta := map[string]string{"limit": "100"}
	d := errx.ErrorInfo("QUOTA_EXCEEDED", "example.com", meta)
	if d.Reason != "QUOTA_EXCEEDED" {
		t.Errorf("reason = %q, want %q", d.Reason, "QUOTA_EXCEEDED")
	}
	if d.Domain != "example.com" {
		t.Errorf("domain = %q, want %q", d.Domain, "example.com")
	}
	if d.Metadata["limit"] != "100" {
		t.Errorf("metadata[limit] = %q, want %q", d.Metadata["limit"], "100")
	}
}

func TestDetailWithError(t *testing.T) {
	t.Parallel()

	err := errx.New("bad request").
		WithCode(errx.InvalidArgument).
		WithDetails(
			errx.FieldViolation("email", "invalid"),
			errx.ResourceInfo("User", "42", "", "not found"),
		)

	details := errx.DetailsOf(err)
	if len(details) != 2 {
		t.Fatalf("details length = %d, want 2", len(details))
	}
	if _, ok := details[0].(*errx.BadRequestDetail); !ok {
		t.Errorf("details[0] type = %T, want *errx.BadRequestDetail", details[0])
	}
	if _, ok := details[1].(*errx.ResourceInfoDetail); !ok {
		t.Errorf("details[1] type = %T, want *errx.ResourceInfoDetail", details[1])
	}
}
