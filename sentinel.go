package errx

// Sentinel is an immutable error value intended for use as a package-level sentinel.
// It carries a fixed message and code, and supports errors.Is matching by identity.
type Sentinel struct {
	msg  string
	code Code
}

// NewSentinel creates a new sentinel error with the given message and code.
func NewSentinel(msg string, code Code) *Sentinel {
	return &Sentinel{msg: msg, code: code}
}

// Error implements the error interface.
func (s *Sentinel) Error() string { return s.msg }

// Code implements the Coder interface.
func (s *Sentinel) Code() Code { return s.code }
