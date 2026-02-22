package grpcerr_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mickamy/errx"
	"github.com/mickamy/errx/grpcerr"
)

func TestUnaryServerInterceptor(t *testing.T) {
	t.Parallel()

	interceptor := grpcerr.UnaryServerInterceptor()

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

func TestStreamServerInterceptor(t *testing.T) {
	t.Parallel()

	interceptor := grpcerr.StreamServerInterceptor()

	t.Run("no error", func(t *testing.T) {
		t.Parallel()
		err := interceptor(
			nil, nil, &grpc.StreamServerInfo{},
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
		err := interceptor(
			nil, nil, &grpc.StreamServerInfo{},
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
