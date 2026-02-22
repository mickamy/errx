package errx

// SentinelError is an immutable error value intended for use as a package-level sentinel.
// It carries a fixed message and code, and supports errors.Is matching by identity.
type SentinelError struct {
	msg  string
	code Code
}

// NewSentinel creates a new sentinel error with the given message and code.
func NewSentinel(msg string, code Code) *SentinelError {
	return &SentinelError{msg: msg, code: code}
}

// Error implements the error interface.
func (s *SentinelError) Error() string { return s.msg }

// Code implements the Coder interface.
func (s *SentinelError) Code() Code { return s.code }
