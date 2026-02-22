package cerr_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"golang.org/x/text/language"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/mickamy/errx"
	"github.com/mickamy/errx/cerr"
)

// localizableError is a test error that implements errx.Localizable.
type localizableError struct {
	messages map[string]string
}

func (e *localizableError) Error() string { return "localizable error" }

func (e *localizableError) Localize(locale string) string {
	return e.messages[locale]
}

var _ errx.Localizable = (*localizableError)(nil)

func newTestRequest(header http.Header) connect.AnyRequest {
	req := connect.NewRequest[any](nil)
	for k, vs := range header {
		for _, v := range vs {
			req.Header().Set(k, v)
		}
	}
	return req
}

// fakeStreamingHandlerConn implements connect.StreamingHandlerConn for testing.
type fakeStreamingHandlerConn struct {
	header http.Header
}

func (c *fakeStreamingHandlerConn) Spec() connect.Spec          { return connect.Spec{} }
func (c *fakeStreamingHandlerConn) Peer() connect.Peer          { return connect.Peer{} }
func (c *fakeStreamingHandlerConn) Receive(_ any) error         { return nil }
func (c *fakeStreamingHandlerConn) RequestHeader() http.Header  { return c.header }
func (c *fakeStreamingHandlerConn) Send(_ any) error            { return nil }
func (c *fakeStreamingHandlerConn) ResponseHeader() http.Header { return http.Header{} }
func (c *fakeStreamingHandlerConn) ResponseTrailer() http.Header {
	return http.Header{}
}

func TestNewInterceptor_Unary(t *testing.T) {
	t.Parallel()

	i := cerr.NewInterceptor()

	t.Run("no error", func(t *testing.T) {
		t.Parallel()
		inner := i.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, nil //nolint:nilnil // testing no-error path
		})
		_, err := inner(t.Context(), newTestRequest(http.Header{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("errx error", func(t *testing.T) {
		t.Parallel()
		inner := i.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, errx.New("not found").WithCode(errx.NotFound)
		})
		_, err := inner(t.Context(), newTestRequest(http.Header{}))
		var ce *connect.Error
		if !errors.As(err, &ce) {
			t.Fatal("error should be a *connect.Error")
		}
		if ce.Code() != connect.CodeNotFound {
			t.Errorf("code = %v, want NotFound", ce.Code())
		}
	})

	t.Run("plain error", func(t *testing.T) {
		t.Parallel()
		inner := i.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, errors.New("boom")
		})
		_, err := inner(t.Context(), newTestRequest(http.Header{}))
		var ce *connect.Error
		if !errors.As(err, &ce) {
			t.Fatal("error should be a *connect.Error")
		}
		if ce.Code() != connect.CodeUnknown {
			t.Errorf("code = %v, want Unknown", ce.Code())
		}
	})
}

func TestNewInterceptor_Localizable(t *testing.T) {
	t.Parallel()

	t.Run("auto-appends LocalizedMessage from Accept-Language", func(t *testing.T) {
		t.Parallel()
		i := cerr.NewInterceptor()
		inner := i.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, errx.Wrap(&localizableError{
				messages: map[string]string{"ja": "名前は必須です"}, //nolint:gosmopolitan // test i18n
			}).WithCode(errx.InvalidArgument)
		})
		header := http.Header{}
		header.Set("Accept-Language", "ja")
		_, err := inner(t.Context(), newTestRequest(header))
		var ce *connect.Error
		if !errors.As(err, &ce) {
			t.Fatal("error should be a *connect.Error")
		}
		found := false
		for _, d := range ce.Details() {
			v, vErr := d.Value()
			if vErr != nil {
				continue
			}
			if lm, ok := v.(*errdetails.LocalizedMessage); ok {
				found = true
				if lm.GetLocale() != "ja" {
					t.Errorf("locale = %q, want %q", lm.GetLocale(), "ja")
				}
				if lm.GetMessage() != "名前は必須です" { //nolint:gosmopolitan // test i18n
					t.Errorf("message = %q", lm.GetMessage())
				}
			}
		}
		if !found {
			t.Error("LocalizedMessage detail not found")
		}
	})

	t.Run("no Accept-Language means no LocalizedMessage", func(t *testing.T) {
		t.Parallel()
		i := cerr.NewInterceptor()
		inner := i.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, errx.Wrap(&localizableError{
				messages: map[string]string{"en": "Name is required"},
			}).WithCode(errx.InvalidArgument)
		})
		_, err := inner(t.Context(), newTestRequest(http.Header{}))
		var ce *connect.Error
		if !errors.As(err, &ce) {
			t.Fatal("error should be a *connect.Error")
		}
		for _, d := range ce.Details() {
			v, vErr := d.Value()
			if vErr != nil {
				continue
			}
			if _, ok := v.(*errdetails.LocalizedMessage); ok {
				t.Error("LocalizedMessage should not be present without Accept-Language")
			}
		}
	})

	t.Run("quality value selects highest priority", func(t *testing.T) {
		t.Parallel()
		i := cerr.NewInterceptor()
		inner := i.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, errx.Wrap(&localizableError{
				messages: map[string]string{"ja": "名前は必須です"}, //nolint:gosmopolitan // test i18n
			}).WithCode(errx.InvalidArgument)
		})
		header := http.Header{}
		header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")
		_, err := inner(t.Context(), newTestRequest(header))
		var ce *connect.Error
		if !errors.As(err, &ce) {
			t.Fatal("error should be a *connect.Error")
		}
		found := false
		for _, d := range ce.Details() {
			v, vErr := d.Value()
			if vErr != nil {
				continue
			}
			if lm, ok := v.(*errdetails.LocalizedMessage); ok {
				found = true
				if lm.GetLocale() != "ja" {
					t.Errorf("locale = %q, want %q", lm.GetLocale(), "ja")
				}
				if lm.GetMessage() != "名前は必須です" { //nolint:gosmopolitan // test i18n
					t.Errorf("message = %q", lm.GetMessage())
				}
			}
		}
		if !found {
			t.Error("LocalizedMessage detail not found")
		}
	})

	t.Run("default locale fallback", func(t *testing.T) {
		t.Parallel()
		i := cerr.NewInterceptor(
			cerr.WithDefaultLocale(language.English),
		)
		inner := i.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, errx.Wrap(&localizableError{
				messages: map[string]string{"en": "Name is required"},
			}).WithCode(errx.InvalidArgument)
		})
		// No Accept-Language header.
		_, err := inner(t.Context(), newTestRequest(http.Header{}))
		var ce *connect.Error
		if !errors.As(err, &ce) {
			t.Fatal("error should be a *connect.Error")
		}
		found := false
		for _, d := range ce.Details() {
			v, vErr := d.Value()
			if vErr != nil {
				continue
			}
			if lm, ok := v.(*errdetails.LocalizedMessage); ok {
				found = true
				if lm.GetLocale() != "en" || lm.GetMessage() != "Name is required" {
					t.Errorf("got locale=%q message=%q", lm.GetLocale(), lm.GetMessage())
				}
			}
		}
		if !found {
			t.Error("LocalizedMessage detail not found")
		}
	})

	t.Run("custom locale func", func(t *testing.T) {
		t.Parallel()
		i := cerr.NewInterceptor(
			cerr.WithLocaleFunc(func(_ http.Header) string { return "fr" }),
		)
		inner := i.WrapUnary(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, errx.Wrap(&localizableError{
				messages: map[string]string{"fr": "Le nom est requis"},
			}).WithCode(errx.InvalidArgument)
		})
		_, err := inner(t.Context(), newTestRequest(http.Header{}))
		var ce *connect.Error
		if !errors.As(err, &ce) {
			t.Fatal("error should be a *connect.Error")
		}
		found := false
		for _, d := range ce.Details() {
			v, vErr := d.Value()
			if vErr != nil {
				continue
			}
			if lm, ok := v.(*errdetails.LocalizedMessage); ok {
				found = true
				if lm.GetLocale() != "fr" || lm.GetMessage() != "Le nom est requis" {
					t.Errorf("got locale=%q message=%q", lm.GetLocale(), lm.GetMessage())
				}
			}
		}
		if !found {
			t.Error("LocalizedMessage detail not found")
		}
	})
}

func TestNewInterceptor_StreamingHandler(t *testing.T) {
	t.Parallel()

	i := cerr.NewInterceptor()
	header := http.Header{}
	header.Set("Accept-Language", "en")
	conn := &fakeStreamingHandlerConn{header: header}

	wrapped := i.WrapStreamingHandler(func(_ context.Context, _ connect.StreamingHandlerConn) error {
		return errx.Wrap(&localizableError{
			messages: map[string]string{"en": "Name is required"},
		}).WithCode(errx.InvalidArgument)
	})

	err := wrapped(t.Context(), conn)
	var ce *connect.Error
	if !errors.As(err, &ce) {
		t.Fatal("error should be a *connect.Error")
	}
	found := false
	for _, d := range ce.Details() {
		v, vErr := d.Value()
		if vErr != nil {
			continue
		}
		if lm, ok := v.(*errdetails.LocalizedMessage); ok {
			found = true
			if lm.GetLocale() != "en" || lm.GetMessage() != "Name is required" {
				t.Errorf("got locale=%q message=%q", lm.GetLocale(), lm.GetMessage())
			}
		}
	}
	if !found {
		t.Error("LocalizedMessage detail not found")
	}
}

func TestNewInterceptor_StreamingClient_PassThrough(t *testing.T) {
	t.Parallel()

	i := cerr.NewInterceptor()
	called := false
	original := func(_ context.Context, _ connect.Spec) connect.StreamingClientConn {
		called = true
		return nil
	}
	wrapped := i.WrapStreamingClient(original)
	_ = wrapped(t.Context(), connect.Spec{})
	if !called {
		t.Error("StreamingClient should pass through")
	}
}
