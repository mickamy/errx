package gerr

import (
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/types/known/durationpb"
)

// FieldViolation creates a BadRequest with a single field violation.
func FieldViolation(field, description string) *errdetails.BadRequest {
	return &errdetails.BadRequest{
		FieldViolations: []*errdetails.BadRequest_FieldViolation{
			{Field: field, Description: description},
		},
	}
}

// BadRequest creates a BadRequest with the given field violations.
func BadRequest(violations ...*errdetails.BadRequest_FieldViolation) *errdetails.BadRequest {
	return &errdetails.BadRequest{
		FieldViolations: violations,
	}
}

// NewFieldViolation creates a single BadRequest_FieldViolation.
func NewFieldViolation(field, description string) *errdetails.BadRequest_FieldViolation {
	return &errdetails.BadRequest_FieldViolation{
		Field:       field,
		Description: description,
	}
}

// PreconditionFailure creates a PreconditionFailure with the given violations.
func PreconditionFailure(violations ...*errdetails.PreconditionFailure_Violation) *errdetails.PreconditionFailure {
	return &errdetails.PreconditionFailure{
		Violations: violations,
	}
}

// NewPreconditionViolation creates a single PreconditionFailure_Violation.
func NewPreconditionViolation(typ, subject, description string) *errdetails.PreconditionFailure_Violation {
	return &errdetails.PreconditionFailure_Violation{
		Type:        typ,
		Subject:     subject,
		Description: description,
	}
}

// QuotaFailure creates a QuotaFailure with the given violations.
func QuotaFailure(violations ...*errdetails.QuotaFailure_Violation) *errdetails.QuotaFailure {
	return &errdetails.QuotaFailure{
		Violations: violations,
	}
}

// NewQuotaViolation creates a single QuotaFailure_Violation.
func NewQuotaViolation(subject, description string) *errdetails.QuotaFailure_Violation {
	return &errdetails.QuotaFailure_Violation{
		Subject:     subject,
		Description: description,
	}
}

// ResourceInfo creates a ResourceInfo detail.
func ResourceInfo(resourceType, resourceName, owner, description string) *errdetails.ResourceInfo {
	return &errdetails.ResourceInfo{
		ResourceType: resourceType,
		ResourceName: resourceName,
		Owner:        owner,
		Description:  description,
	}
}

// ErrorInfo creates an ErrorInfo detail.
func ErrorInfo(reason, domain string, metadata map[string]string) *errdetails.ErrorInfo {
	return &errdetails.ErrorInfo{
		Reason:   reason,
		Domain:   domain,
		Metadata: metadata,
	}
}

// RetryInfo creates a RetryInfo detail with the given retry delay.
func RetryInfo(retryDelay time.Duration) *errdetails.RetryInfo {
	return &errdetails.RetryInfo{
		RetryDelay: durationpb.New(retryDelay),
	}
}

// DebugInfo creates a DebugInfo detail.
func DebugInfo(stackEntries []string, detail string) *errdetails.DebugInfo {
	return &errdetails.DebugInfo{
		StackEntries: stackEntries,
		Detail:       detail,
	}
}

// LocalizedMessage creates a LocalizedMessage detail.
func LocalizedMessage(locale, message string) *errdetails.LocalizedMessage {
	return &errdetails.LocalizedMessage{
		Locale:  locale,
		Message: message,
	}
}
