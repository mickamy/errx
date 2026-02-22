package gerr_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/mickamy/errx"
	"github.com/mickamy/errx/gerr"
)

func TestUnaryServerInterceptor(t *testing.T) {
	t.Parallel()

	interceptor := gerr.UnaryServerInterceptor()

	t.Run("no error", func(t *testing.T) {
		t.Parallel()
		resp, err := interceptor(
			t.Context(), "req", &grpc.UnaryServerInfo{},
			func(_ context.Context, _ any) (any, error) {
				return "ok", nil
			},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp != "ok" {
			t.Errorf("resp = %v, want %q", resp, "ok")
		}
	})

	t.Run("errx error", func(t *testing.T) {
		t.Parallel()
		resp, err := interceptor(
			t.Context(), "req", &grpc.UnaryServerInfo{},
			func(_ context.Context, _ any) (any, error) {
				return nil, errx.New("not found").WithCode(errx.NotFound)
			},
		)
		if resp != nil {
			t.Errorf("resp should be nil, got %v", resp)
		}
		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("error should be a gRPC status error")
		}
		if st.Code() != codes.NotFound {
			t.Errorf("code = %v, want NotFound", st.Code())
		}
	})

	t.Run("plain error", func(t *testing.T) {
		t.Parallel()
		_, err := interceptor(
			t.Context(), "req", &grpc.UnaryServerInfo{},
			func(_ context.Context, _ any) (any, error) {
				return nil, errors.New("boom")
			},
		)
		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("error should be a gRPC status error")
		}
		if st.Code() != codes.Unknown {
			t.Errorf("code = %v, want Unknown", st.Code())
		}
	})
}

// fakeServerStream is a minimal grpc.ServerStream for testing.
type fakeServerStream struct {
	grpc.ServerStream

	ctx context.Context //nolint:containedctx // test helper
}

func (f *fakeServerStream) Context() context.Context { return f.ctx }

func TestStreamServerInterceptor(t *testing.T) {
	t.Parallel()

	interceptor := gerr.StreamServerInterceptor()

	t.Run("no error", func(t *testing.T) {
		t.Parallel()
		ss := &fakeServerStream{ctx: t.Context()}
		err := interceptor(
			nil, ss, &grpc.StreamServerInfo{},
			func(_ any, _ grpc.ServerStream) error {
				return nil
			},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("errx error", func(t *testing.T) {
		t.Parallel()
		ss := &fakeServerStream{ctx: t.Context()}
		err := interceptor(
			nil, ss, &grpc.StreamServerInfo{},
			func(_ any, _ grpc.ServerStream) error {
				return errx.New("internal").WithCode(errx.Internal)
			},
		)
		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("error should be a gRPC status error")
		}
		if st.Code() != codes.Internal {
			t.Errorf("code = %v, want Internal", st.Code())
		}
	})
}

// localizableError is a test error that implements errx.Localizable.
type localizableError struct {
	messages map[string]string
}

func (e *localizableError) Error() string { return "localizable error" }

func (e *localizableError) Localize(locale string) string {
	return e.messages[locale]
}

func TestUnaryServerInterceptor_Localizable(t *testing.T) {
	t.Parallel()

	t.Run("auto-appends LocalizedMessage from metadata", func(t *testing.T) {
		t.Parallel()
		interceptor := gerr.UnaryServerInterceptor()
		ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("accept-language", "ja"))
		_, err := interceptor(
			ctx, "req", &grpc.UnaryServerInfo{},
			func(_ context.Context, _ any) (any, error) {
				return nil, errx.Wrap(&localizableError{
					messages: map[string]string{"ja": "名前は必須です"}, //nolint:gosmopolitan // test i18n
				}).WithCode(errx.InvalidArgument)
			},
		)
		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("error should be a gRPC status error")
		}
		found := false
		for _, d := range st.Details() {
			if lm, ok := d.(*errdetails.LocalizedMessage); ok {
				found = true
				if lm.GetLocale() != "ja" {
					t.Errorf("locale = %q, want %q", lm.GetLocale(), "ja")
				}
				if lm.GetMessage() != "名前は必須です" { //nolint:gosmopolitan // test i18n
					t.Errorf("message = %q, want %q", lm.GetMessage(), "名前は必須です") //nolint:gosmopolitan // test i18n
				}
			}
		}
		if !found {
			t.Error("LocalizedMessage detail not found")
		}
	})

	t.Run("no metadata means no LocalizedMessage", func(t *testing.T) {
		t.Parallel()
		interceptor := gerr.UnaryServerInterceptor()
		_, err := interceptor(
			t.Context(), "req", &grpc.UnaryServerInfo{},
			func(_ context.Context, _ any) (any, error) {
				return nil, errx.Wrap(&localizableError{
					messages: map[string]string{"en": "Name is required"},
				}).WithCode(errx.InvalidArgument)
			},
		)
		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("error should be a gRPC status error")
		}
		for _, d := range st.Details() {
			if _, ok := d.(*errdetails.LocalizedMessage); ok {
				t.Error("LocalizedMessage should not be present without locale")
			}
		}
	})

	t.Run("non-Localizable error is unchanged", func(t *testing.T) {
		t.Parallel()
		interceptor := gerr.UnaryServerInterceptor()
		ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("accept-language", "en"))
		_, err := interceptor(
			ctx, "req", &grpc.UnaryServerInfo{},
			func(_ context.Context, _ any) (any, error) {
				return nil, errx.New("plain error").WithCode(errx.Internal)
			},
		)
		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("error should be a gRPC status error")
		}
		for _, d := range st.Details() {
			if _, ok := d.(*errdetails.LocalizedMessage); ok {
				t.Error("LocalizedMessage should not be present for non-Localizable error")
			}
		}
	})

	t.Run("quality value selects highest priority", func(t *testing.T) {
		t.Parallel()
		interceptor := gerr.UnaryServerInterceptor()
		ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("accept-language", "ja,en-US;q=0.9,en;q=0.8"))
		_, err := interceptor(
			ctx, "req", &grpc.UnaryServerInfo{},
			func(_ context.Context, _ any) (any, error) {
				return nil, errx.Wrap(&localizableError{
					messages: map[string]string{"ja": "名前は必須です"}, //nolint:gosmopolitan // test i18n
				}).WithCode(errx.InvalidArgument)
			},
		)
		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("error should be a gRPC status error")
		}
		found := false
		for _, d := range st.Details() {
			if lm, ok := d.(*errdetails.LocalizedMessage); ok {
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

	t.Run("custom locale func", func(t *testing.T) {
		t.Parallel()
		interceptor := gerr.UnaryServerInterceptor(
			gerr.WithLocaleFunc(func(_ context.Context) string { return "fr" }),
		)
		_, err := interceptor(
			t.Context(), "req", &grpc.UnaryServerInfo{},
			func(_ context.Context, _ any) (any, error) {
				return nil, errx.Wrap(&localizableError{
					messages: map[string]string{"fr": "Le nom est requis"},
				}).WithCode(errx.InvalidArgument)
			},
		)
		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("error should be a gRPC status error")
		}
		found := false
		for _, d := range st.Details() {
			if lm, ok := d.(*errdetails.LocalizedMessage); ok {
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

func TestStreamServerInterceptor_Localizable(t *testing.T) {
	t.Parallel()

	interceptor := gerr.StreamServerInterceptor()
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("accept-language", "en"))
	ss := &fakeServerStream{ctx: ctx}

	err := interceptor(
		nil, ss, &grpc.StreamServerInfo{},
		func(_ any, _ grpc.ServerStream) error {
			return errx.Wrap(&localizableError{
				messages: map[string]string{"en": "Name is required"},
			}).WithCode(errx.InvalidArgument)
		},
	)
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("error should be a gRPC status error")
	}
	found := false
	for _, d := range st.Details() {
		if lm, ok := d.(*errdetails.LocalizedMessage); ok {
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

// Ensure localizableError implements errx.Localizable at compile time.
var _ errx.Localizable = (*localizableError)(nil)
