package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/mickamy/errx"
	"github.com/mickamy/errx/cerr"
)

// validationError implements errx.Localizable.
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

	// Create a Connect interceptor that converts errx errors.
	interceptor := cerr.NewInterceptor()

	// Start an HTTP server with a simple Connect-style handler.
	mux := http.NewServeMux()
	mux.HandleFunc("/greet", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")

		// Simulate handler logic that returns errx errors.
		err := greet(name)
		if err != nil {
			// Use the interceptor's conversion logic via ToConnectError.
			header := r.Header
			locale := header.Get("Accept-Language")

			// Build a connect error with localized details.
			var l errx.Localizable
			if errors.As(err, &l) && locale != "" {
				msg := l.Localize(locale)
				if msg != "" {
					var ex *errx.Error
					if errors.As(err, &ex) {
						err = ex.WithDetails(&errdetails.LocalizedMessage{
							Locale:  locale,
							Message: msg,
						})
					}
				}
			}

			ce := cerr.ToConnectError(err)
			http.Error(w, ce.Error(), http.StatusBadRequest)
			return
		}

		fmt.Fprintf(w, "Hello %s!\n", name) //nolint:gosec // example code
	})

	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	srv := &http.Server{Handler: mux} //nolint:gosec // example code

	go func() {
		if err := srv.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
		}
	}()
	defer func() { _ = srv.Close() }()

	addr := lis.Addr().String()

	// Demonstrate the interceptor and error conversion.
	_ = interceptor // interceptor is used with Connect RPC handlers

	// 1. ToConnectError with code + details.
	logger.Info("=== 1. InvalidArgument + FieldViolation ===")
	err1 := errx.New("name is required").
		WithCode(errx.InvalidArgument).
		WithDetails(&errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{Field: "name", Description: "must not be empty"},
			},
		})
	ce1 := cerr.ToConnectError(err1)
	logConnectError(logger, ce1)

	// 2. Localizable error conversion.
	logger.Info("=== 2. Localizable error ===")
	err2 := errx.Wrap(&validationError{
		field: "name",
		messages: map[string]string{
			"en": "Name is required",
			"ja": "名前は必須です", //nolint:gosmopolitan // example i18n
		},
	}).WithCode(errx.InvalidArgument)
	ce2 := cerr.ToConnectError(err2)
	logConnectError(logger, ce2)

	// 3. Round-trip: errx → connect → errx.
	logger.Info("=== 3. Round-trip ===")
	original := errx.New("not found").
		WithCode(errx.NotFound).
		WithDetails(&errdetails.ResourceInfo{
			ResourceType: "User",
			ResourceName: "alice",
		})
	ce3 := cerr.ToConnectError(original)
	recovered := cerr.FromConnectError(ce3)
	logger.Info("recovered",
		slog.String("code", string(recovered.Code())),
		slog.String("message", recovered.Error()),
		slog.Int("details_count", len(errx.DetailsOf(recovered))),
	)

	// 4. HTTP call to the server.
	logger.Info("=== 4. HTTP call (validation error) ===")
	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://"+addr+"/greet?name=validate",
		nil,
	)
	req.Header.Set("Accept-Language", "ja")
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // example code
	if err != nil {
		return fmt.Errorf("http call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	logger.Info("response", slog.Int("status", resp.StatusCode))

	return nil
}

func greet(name string) error {
	switch name {
	case "":
		return errx.New("name is required").
			WithCode(errx.InvalidArgument).
			WithDetails(&errdetails.BadRequest{
				FieldViolations: []*errdetails.BadRequest_FieldViolation{
					{Field: "name", Description: "must not be empty"},
				},
			})
	case "validate":
		return errx.Wrap(&validationError{
			field: "name",
			messages: map[string]string{
				"en": "Name is required",
				"ja": "名前は必須です", //nolint:gosmopolitan // example i18n
			},
		}).WithCode(errx.InvalidArgument)
	}
	return nil
}

func logConnectError(logger *slog.Logger, ce *connect.Error) {
	attrs := []any{
		slog.String("code", ce.Code().String()),
		slog.String("message", ce.Message()),
	}

	for _, detail := range ce.Details() {
		v, err := detail.Value()
		if err != nil {
			continue
		}
		switch d := v.(type) {
		case *errdetails.BadRequest:
			for _, fv := range d.GetFieldViolations() {
				attrs = append(attrs, slog.String("field_violation", fv.GetField()+"="+fv.GetDescription()))
			}
		case *errdetails.ResourceInfo:
			attrs = append(attrs, slog.String("resource", d.GetResourceType()+"/"+d.GetResourceName()))
		case *errdetails.LocalizedMessage:
			attrs = append(attrs, slog.String("localized_"+d.GetLocale(), d.GetMessage()))
		}
	}

	logger.Error("connect error", attrs...)
}
