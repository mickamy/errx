package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/mickamy/errx"
	"github.com/mickamy/errx/grpcerr"
)

var ErrUserNotFound = errx.NewSentinel("user not found", errx.NotFound)

// validationError is an error that implements errx.Localizable.
type validationError struct {
	field    string
	messages map[string]string
}

func (e *validationError) Error() string {
	return e.field + " is invalid"
}

func (e *validationError) Localize(locale string) string {
	if msg, ok := e.messages[locale]; ok {
		return msg
	}
	return e.messages["en"]
}

// server implements the Greeter service.
type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(_ context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	name := req.GetName()

	switch name {
	case "":
		// WithDetails: attach a FieldViolation detail.
		return nil, errx.New("name is required").
			WithCode(errx.InvalidArgument).
			WithDetails(grpcerr.FieldViolation("name", "must not be empty"))
	case "unknown":
		// WithDetails: attach a ResourceInfo detail.
		return nil, errx.Wrap(ErrUserNotFound).
			WithDetails(grpcerr.ResourceInfo("User", name, "", "user not found"))
	case "admin":
		return nil, errx.New("admin access denied", "name", name).
			WithCode(errx.PermissionDenied)
	case "validate":
		// Localizable: the interceptor auto-appends LocalizedMessage.
		return nil, errx.Wrap(&validationError{
			field: "name",
			messages: map[string]string{
				"en": "Name is required",
				"ja": "名前は必須です", //nolint:gosmopolitan // example i18n
			},
		}).WithCode(errx.InvalidArgument).
			WithDetails(grpcerr.FieldViolation("name", "must not be empty"))
	}

	return &pb.HelloReply{Message: "Hello " + name}, nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	// Start gRPC server with the errx interceptor.
	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
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
		return fmt.Errorf("connect: %w", err)
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

	// 2. InvalidArgument with FieldViolation detail.
	logger.Info("=== 2. InvalidArgument + FieldViolation ===")
	_, err = client.SayHello(ctx, &pb.HelloRequest{Name: ""})
	logGRPCError(logger, err)

	// 3. NotFound with ResourceInfo detail.
	logger.Info("=== 3. NotFound + ResourceInfo ===")
	_, err = client.SayHello(ctx, &pb.HelloRequest{Name: "unknown"})
	logGRPCError(logger, err)

	// 4. PermissionDenied (no details).
	logger.Info("=== 4. PermissionDenied ===")
	_, err = client.SayHello(ctx, &pb.HelloRequest{Name: "admin"})
	logGRPCError(logger, err)

	// 5. Localizable error — sends accept-language metadata.
	logger.Info("=== 5. Localizable + FieldViolation (ja) ===")
	jaCtx := metadata.AppendToOutgoingContext(ctx, "accept-language", "ja")
	_, err = client.SayHello(jaCtx, &pb.HelloRequest{Name: "validate"})
	logGRPCError(logger, err)

	// 6. Round-trip: convert gRPC status back to errx with details.
	logger.Info("=== 6. Round-trip (gRPC → errx with details) ===")
	_, err = client.SayHello(ctx, &pb.HelloRequest{Name: ""})
	st, _ := status.FromError(err)
	recovered := grpcerr.FromStatus(st)
	logger.Error("recovered errx error",
		"code", recovered.Code(),
		"message", recovered.Error(),
		"details_count", len(errx.DetailsOf(recovered)),
	)

	return nil
}

func logGRPCError(logger *slog.Logger, err error) {
	st, ok := status.FromError(err)
	if !ok {
		logger.Error("non-gRPC error", "error", err)
		return
	}

	attrs := []any{
		"grpc_code", st.Code().String(),
		"message", st.Message(),
		"is_not_found", st.Code() == codes.NotFound,
	}

	for _, detail := range st.Details() {
		switch d := detail.(type) {
		case *errdetails.BadRequest:
			for _, v := range d.GetFieldViolations() {
				attrs = append(attrs, "field_violation", v.GetField()+"="+v.GetDescription())
			}
		case *errdetails.ResourceInfo:
			attrs = append(attrs, "resource", d.GetResourceType()+"/"+d.GetResourceName())
		case *errdetails.LocalizedMessage:
			attrs = append(attrs, "localized_"+d.GetLocale(), d.GetMessage())
		}
	}

	logger.Error("gRPC error", attrs...)
}
