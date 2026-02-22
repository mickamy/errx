package errx

import (
	"errors"
	"fmt"
	"log/slog"
)

// Error is a structured error that carries a message, optional cause,
// classification code, structured fields, and an optional stack trace.
type Error struct {
	msg    string
	cause  error
	code   Code
	fields []slog.Attr
	stack  *Stack
}

// New creates a new Error with the given message and optional structured fields.
// Fields follow the same convention as slog: alternating key-value pairs or slog.Attr values.
func New(msg string, args ...any) *Error {
	return &Error{
		msg:    msg,
		fields: argsToAttrs(args),
	}
}

// Wrap wraps an existing error with optional structured fields.
// Returns nil if err is nil.
func Wrap(err error, args ...any) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		cause:  err,
		fields: argsToAttrs(args),
	}
}

// Wrapf wraps an existing error with a formatted message.
// Additional args beyond the format arguments are not supported;
// use [Wrap] followed by [Error.With] for structured fields.
// Returns nil if err is nil.
func Wrapf(err error, format string, fmtArgs ...any) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		msg:   fmt.Sprintf(format, fmtArgs...),
		cause: err,
	}
}

// With returns a copy of the error with additional structured fields appended.
func (e *Error) With(args ...any) *Error {
	cp := *e
	cp.fields = append(append([]slog.Attr(nil), e.fields...), argsToAttrs(args)...)
	return &cp
}

// WithCode returns a copy of the error with the given code set.
func (e *Error) WithCode(c Code) *Error {
	cp := *e
	cp.code = c
	return &cp
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.msg == "" && e.cause != nil {
		return e.cause.Error()
	}
	if e.cause == nil {
		return e.msg
	}
	return e.msg + ": " + e.cause.Error()
}

// Unwrap returns the underlying cause, enabling errors.Is/errors.As.
func (e *Error) Unwrap() error {
	return e.cause
}

// Code returns the code of this error.
// If this error has no code set, it walks the cause chain.
func (e *Error) Code() Code {
	if e.code != "" {
		return e.code
	}
	return CodeOf(e.cause)
}

// Fields collects all structured fields from the error chain (outermost first).
// Duplicate keys are preserved, matching slog behavior.
func Fields(err error) []slog.Attr {
	var attrs []slog.Attr
	for err != nil {
		var ex *Error
		if errors.As(err, &ex) {
			attrs = append(attrs, ex.fields...)
			err = ex.cause
		} else {
			break
		}
	}
	return attrs
}

// argsToAttrs converts slog-style args (alternating key/value or slog.Attr) into []slog.Attr.
// Follows the same conventions as slog: a lone key without a value gets the key "!BADKEY".
func argsToAttrs(args []any) []slog.Attr {
	var attrs []slog.Attr
	for i := 0; i < len(args); {
		switch v := args[i].(type) {
		case slog.Attr:
			attrs = append(attrs, v)
			i++
		case string:
			if i+1 < len(args) {
				attrs = append(attrs, slog.Any(v, args[i+1]))
				i += 2
			} else {
				attrs = append(attrs, slog.String("!BADKEY", v))
				i++
			}
		default:
			attrs = append(attrs, slog.Any("!BADKEY", v))
			i++
		}
	}
	return attrs
}
