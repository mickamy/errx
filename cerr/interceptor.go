package cerr

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/text/language"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/mickamy/errx"
)

// InterceptorOption configures the Connect server interceptor.
type InterceptorOption func(*interceptorConfig)

type interceptorConfig struct {
	localeFunc    func(http.Header) string
	defaultLocale language.Tag
}

// WithLocaleFunc sets a custom function to extract locale from request headers.
// The default parses the "Accept-Language" header and returns the highest-priority
// language tag as a BCP 47 string.
func WithLocaleFunc(f func(http.Header) string) InterceptorOption {
	return func(cfg *interceptorConfig) {
		if f == nil {
			return
		}
		cfg.localeFunc = f
	}
}

// WithDefaultLocale sets a fallback locale used when the locale function
// returns an empty string (e.g. no Accept-Language header).
func WithDefaultLocale(tag language.Tag) InterceptorOption {
	return func(cfg *interceptorConfig) {
		cfg.defaultLocale = tag
	}
}

func newInterceptorConfig(opts []InterceptorOption) *interceptorConfig {
	cfg := &interceptorConfig{
		localeFunc: defaultLocaleFunc,
	}
	for _, o := range opts {
		o(cfg)
	}
	return cfg
}

func defaultLocaleFunc(h http.Header) string {
	return errx.ParseAcceptLanguage(h.Get("Accept-Language"))
}

// NewInterceptor returns a Connect interceptor that converts returned errors
// to Connect errors using ToConnectError.
// If the error implements errx.Localizable, a LocalizedMessage detail
// is automatically appended based on the request's Accept-Language header.
func NewInterceptor(opts ...InterceptorOption) connect.Interceptor {
	cfg := newInterceptorConfig(opts)
	return &interceptor{cfg: cfg}
}

var _ connect.Interceptor = (*interceptor)(nil)

type interceptor struct {
	cfg *interceptorConfig
}

func (i *interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp, err := next(ctx, req)
		if err != nil {
			return nil, i.cfg.toConnectError(req.Header(), err)
		}
		return resp, nil
	}
}

func (i *interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		err := next(ctx, conn)
		if err != nil {
			return i.cfg.toConnectError(conn.RequestHeader(), err)
		}
		return nil
	}
}

func (cfg *interceptorConfig) toConnectError(header http.Header, err error) error {
	err = appendLocalizedDetail(header, err, cfg.localeFunc, cfg.defaultLocale)
	return ToConnectError(err)
}

func appendLocalizedDetail(
	header http.Header, err error, localeFunc func(http.Header) string, defaultLocale language.Tag,
) error {
	var l errx.Localizable
	if !errors.As(err, &l) {
		return err
	}
	locale := localeFunc(header)
	if locale == "" && defaultLocale != language.Und {
		locale = defaultLocale.String()
	}
	if locale == "" {
		return err
	}
	msg := l.Localize(locale)
	if msg == "" {
		return err
	}
	var ex *errx.Error
	if errors.As(err, &ex) {
		return ex.WithDetails(&errdetails.LocalizedMessage{Locale: locale, Message: msg})
	}
	return errx.Wrap(err).WithDetails(&errdetails.LocalizedMessage{Locale: locale, Message: msg})
}
