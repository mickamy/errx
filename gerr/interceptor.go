package gerr

import (
	"context"
	"errors"

	"golang.org/x/text/language"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/mickamy/errx"
)

// InterceptorOption configures the gRPC server interceptors.
type InterceptorOption func(*interceptorConfig)

type interceptorConfig struct {
	localeFunc    func(context.Context) string
	defaultLocale language.Tag
}

// WithLocaleFunc sets a custom function to extract locale from context.
// The default parses the "accept-language" gRPC metadata value and returns
// the highest-priority language tag as a BCP 47 string.
func WithLocaleFunc(f func(context.Context) string) InterceptorOption {
	return func(cfg *interceptorConfig) {
		if f == nil {
			return
		}
		cfg.localeFunc = f
	}
}

// WithDefaultLocale sets a fallback locale used when the locale function
// returns an empty string (e.g. no accept-language metadata).
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

// defaultLocaleFunc extracts locale from the "accept-language" gRPC metadata key.
func defaultLocaleFunc(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get("accept-language")
	if len(vals) == 0 {
		return ""
	}
	return errx.ParseAcceptLanguage(vals[0])
}

// UnaryServerInterceptor returns a gRPC unary server interceptor that
// converts returned errors to gRPC status errors using ToStatus.
// If the error implements errx.Localizable, a LocalizedMessage detail
// is automatically appended.
func UnaryServerInterceptor(opts ...InterceptorOption) grpc.UnaryServerInterceptor {
	cfg := newInterceptorConfig(opts)
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return nil, cfg.toStatusError(ctx, err)
		}
		return resp, nil
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor that
// converts returned errors to gRPC status errors using ToStatus.
// If the error implements errx.Localizable, a LocalizedMessage detail
// is automatically appended.
func StreamServerInterceptor(opts ...InterceptorOption) grpc.StreamServerInterceptor {
	cfg := newInterceptorConfig(opts)
	return func(
		srv any,
		ss grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		err := handler(srv, ss)
		if err != nil {
			return cfg.toStatusError(ss.Context(), err)
		}
		return nil
	}
}

// toStatusError converts an error to a gRPC status error, automatically
// appending a LocalizedMessage detail if the error implements errx.Localizable.
func (cfg *interceptorConfig) toStatusError(ctx context.Context, err error) error {
	err = appendLocalizedDetail(ctx, err, cfg.localeFunc, cfg.defaultLocale)
	return ToStatus(err).Err() //nolint:wrapcheck // intentionally returns gRPC status error
}

// appendLocalizedDetail checks if the error (or any error in its chain)
// implements errx.Localizable. If so and a locale is available, it wraps
// the error with a LocalizedMessage detail.
func appendLocalizedDetail(
	ctx context.Context, err error, localeFunc func(context.Context) string, defaultLocale language.Tag,
) error {
	var l errx.Localizable
	if !errors.As(err, &l) {
		return err
	}
	locale := localeFunc(ctx)
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
		return ex.WithDetails(LocalizedMessage(locale, msg))
	}
	return errx.Wrap(err).WithDetails(LocalizedMessage(locale, msg))
}
