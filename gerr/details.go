package gerr

import (
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/types/known/durationpb"
)

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
