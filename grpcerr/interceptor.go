package grpcerr

import (
	"context"

	"google.golang.org/grpc"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor that
// converts returned errors to gRPC status errors using ToStatus.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return nil, ToStatus(err).Err()
		}
		return resp, nil
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor that
// converts returned errors to gRPC status errors using ToStatus.
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		err := handler(srv, ss)
		if err != nil {
			return ToStatus(err).Err()
		}
		return nil
	}
}
