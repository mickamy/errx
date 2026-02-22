package errx

import "log/slog"

// LogValue implements slog.LogValuer, allowing *Error to be logged directly as a structured value.
// Fields are collected from the entire error chain (outermost first).
func (e *Error) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, 4)
	attrs = append(attrs, slog.String("msg", e.Error()))
	if c := e.Code(); c != "" {
		attrs = append(attrs, slog.String("code", c.String()))
	}
	attrs = append(attrs, Fields(e)...)
	if s := StackOf(e); s != nil {
		frames := s.Frames()
		if len(frames) > 0 {
			f := frames[0]
			attrs = append(attrs, slog.Group("caller",
				slog.String("function", f.Function),
				slog.String("file", f.File),
				slog.Int("line", f.Line),
			))
		}
	}
	return slog.GroupValue(attrs...)
}

// SlogAttr builds a slog.Attr from the entire error chain.
// Fields are collected outermost-first; code is taken from the first Coder in the chain.
func SlogAttr(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}

	attrs := make([]slog.Attr, 0, 4)
	attrs = append(attrs, slog.String("msg", err.Error()))

	if c := CodeOf(err); c != "" {
		attrs = append(attrs, slog.String("code", c.String()))
	}

	attrs = append(attrs, Fields(err)...)

	if s := StackOf(err); s != nil {
		frames := s.Frames()
		if len(frames) > 0 {
			f := frames[0]
			attrs = append(attrs, slog.Group("caller",
				slog.String("function", f.Function),
				slog.String("file", f.File),
				slog.Int("line", f.Line),
			))
		}
	}

	return slog.Attr{Key: "error", Value: slog.GroupValue(attrs...)}
}

// Ensure *Error implements slog.LogValuer at compile time.
var _ slog.LogValuer = (*Error)(nil)
