package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/status"

	"github.com/mickamy/errx"
	"github.com/mickamy/errx/grpcerr"
)

var ErrUserNotFound = errx.NewSentinel("user not found", errx.NotFound)

// server implements the Greeter service.
type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(_ context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	name := req.GetName()

	// Simulate various error scenarios.
	switch name {
	case "":
		return nil, errx.New("name is required", "field", "name").
			WithCode(errx.InvalidArgument)
	case "unknown":
		return nil, errx.Wrap(ErrUserNotFound, "name", name)
	case "admin":
		return nil, errx.New("admin access denied", "name", name).
			WithCode(errx.PermissionDenied)
	}

	return &pb.HelloReply{Message: "Hello " + name}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	// Start gRPC server with the errx interceptor.
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		logger.Error("failed to listen", "error", err)
		os.Exit(1)
	}
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(grpcerr.UnaryServerInterceptor()),
	)
	pb.RegisterGreeterServer(srv, &server{})

	go func() {
		if err := srv.Serve(lis); err != nil {
			logger.Error("server error", "error", err)
		}
	}()
	defer srv.Stop()

	// Connect client.
	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("failed to connect", "error", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	client := pb.NewGreeterClient(conn)
	ctx := context.Background()

	// 1. Successful call.
	logger.Info("=== 1. Successful call ===")
	resp, err := client.SayHello(ctx, &pb.HelloRequest{Name: "Alice"})
	if err != nil {
		logger.Error("unexpected error", "error", err)
	} else {
		logger.Info("response", "message", resp.GetMessage())
	}

	// 2. InvalidArgument — empty name.
	logger.Info("=== 2. InvalidArgument ===")
	_, err = client.SayHello(ctx, &pb.HelloRequest{Name: ""})
	logGRPCError(logger, err)

	// 3. NotFound — unknown user.
	logger.Info("=== 3. NotFound ===")
	_, err = client.SayHello(ctx, &pb.HelloRequest{Name: "unknown"})
	logGRPCError(logger, err)

	// 4. PermissionDenied — admin.
	logger.Info("=== 4. PermissionDenied ===")
	_, err = client.SayHello(ctx, &pb.HelloRequest{Name: "admin"})
	logGRPCError(logger, err)

	// 5. Round-trip: convert gRPC status back to errx.
	logger.Info("=== 5. Round-trip (gRPC → errx) ===")
	_, err = client.SayHello(ctx, &pb.HelloRequest{Name: "unknown"})
	st, _ := status.FromError(err)
	recovered := grpcerr.FromStatus(st)
	logger.Error("recovered errx error",
		"code", recovered.Code(),
		"message", recovered.Error(),
	)
}

func logGRPCError(logger *slog.Logger, err error) {
	st, ok := status.FromError(err)
	if !ok {
		logger.Error("non-gRPC error", "error", err)
		return
	}
	logger.Error("gRPC error",
		"grpc_code", st.Code().String(),
		"message", st.Message(),
		"is_not_found", st.Code() == codes.NotFound,
	)
}
