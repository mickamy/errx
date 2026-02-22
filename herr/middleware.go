package herr

import (
	"errors"
	"net/http"

	"golang.org/x/text/language"

	"github.com/mickamy/errx"
)

// MiddlewareOption configures the HTTP error middleware.
type MiddlewareOption func(*middlewareConfig)

type middlewareConfig struct {
	localeFunc func(http.Header) string
}

// WithLocaleFunc sets a custom function to extract locale from request headers.
// The default extracts the "Accept-Language" header value.
func WithLocaleFunc(f func(http.Header) string) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		if f == nil {
			return
		}
		cfg.localeFunc = f
	}
}

func newMiddlewareConfig(opts []MiddlewareOption) *middlewareConfig {
	cfg := &middlewareConfig{
		localeFunc: defaultLocaleFunc,
	}
	for _, o := range opts {
		o(cfg)
	}
	return cfg
}

func defaultLocaleFunc(h http.Header) string {
	return parseAcceptLanguage(h.Get("Accept-Language"))
}

func parseAcceptLanguage(s string) string {
	tags, qs, err := language.ParseAcceptLanguage(s)
	if err != nil || len(tags) == 0 {
		return ""
	}
	best := 0
	for i := 1; i < len(tags); i++ {
		if qs[i] > qs[best] {
			best = i
		}
	}
	return tags[best].String()
}

// HandlerFunc is an HTTP handler that returns an error.
// If a non-nil error is returned, the middleware writes an RFC 9457 JSON response.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Handler wraps a [HandlerFunc] into an [http.Handler].
// If the handler returns an error, it is converted to an RFC 9457 problem detail response.
// If the error implements [errx.Localizable] and the request carries an Accept-Language header,
// a localized message is automatically included.
func Handler(h HandlerFunc, opts ...MiddlewareOption) http.Handler {
	cfg := newMiddlewareConfig(opts)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			cfg.writeErrorWithLocale(w, r.Header, err)
		}
	})
}

func (cfg *middlewareConfig) writeErrorWithLocale(w http.ResponseWriter, header http.Header, err error) {
	p := ToProblemDetail(err)

	var l errx.Localizable
	if errors.As(err, &l) {
		locale := cfg.localeFunc(header)
		if locale != "" {
			if msg := l.Localize(locale); msg != "" {
				p.LocalizedMessage = &LocalizedMsg{Locale: locale, Message: msg}
			}
		}
	}

	writeProblemDetail(w, p)
}
