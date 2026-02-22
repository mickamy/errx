package errx

import "errors"

// Code is a string-based error classification.
// Users can define custom codes with plain const declarations; no registration required.
type Code string

// String returns the string representation of the Code.
func (c Code) String() string { return string(c) }

// Code implements the Coder interface.
func (c Code) Code() Code { return c }

// Built-in codes that map naturally to gRPC/HTTP status codes.
const (
	Canceled           Code = "canceled"
	Unknown            Code = "unknown"
	InvalidArgument    Code = "invalid_argument"
	DeadlineExceeded   Code = "deadline_exceeded"
	NotFound           Code = "not_found"
	AlreadyExists      Code = "already_exists"
	PermissionDenied   Code = "permission_denied"
	ResourceExhausted  Code = "resource_exhausted"
	FailedPrecondition Code = "failed_precondition"
	Aborted            Code = "aborted"
	OutOfRange         Code = "out_of_range"
	Unimplemented      Code = "unimplemented"
	Internal           Code = "internal"
	Unavailable        Code = "unavailable"
	DataLoss           Code = "data_loss"
	Unauthenticated    Code = "unauthenticated"
)

// Coder is implemented by errors that carry a [Code].
type Coder interface {
	Code() Code
}

// CodeOf extracts the first Code found in the error chain.
// Returns the zero value ("") if no Coder is found.
func CodeOf(err error) Code {
	var c Coder
	if errors.As(err, &c) {
		return c.Code()
	}
	return ""
}

// IsCode reports whether any error in the chain carries the given code.
func IsCode(err error, code Code) bool {
	return CodeOf(err) == code
}
