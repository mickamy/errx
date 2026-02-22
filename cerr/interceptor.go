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
	localeFunc func(http.Header) string
}

// WithLocaleFunc sets a custom function to extract locale from request headers.
// The default extracts the "Accept-Language" header value.
func WithLocaleFunc(f func(http.Header) string) InterceptorOption {
	return func(cfg *interceptorConfig) {
		if f == nil {
			return
		}
		cfg.localeFunc = f
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
	err = appendLocalizedDetail(header, err, cfg.localeFunc)
	return ToConnectError(err)
}

func appendLocalizedDetail(header http.Header, err error, localeFunc func(http.Header) string) error {
	var l errx.Localizable
	if !errors.As(err, &l) {
		return err
	}
	locale := localeFunc(header)
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
