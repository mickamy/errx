package main

import (
	"fmt"
	"log/slog"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))

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

	// 2. Localizable error — ToConnectError preserves the error as-is;
	// the interceptor (cerr.NewInterceptor) auto-appends LocalizedMessage
	// when used with a Connect handler.
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

	// 4. Interceptor usage (with a real Connect handler):
	//
	//   interceptor := cerr.NewInterceptor()
	//   _, handler := foov1connect.NewFooServiceHandler(svc,
	//       connect.WithInterceptors(interceptor),
	//   )
	//
	// The interceptor automatically:
	//   - Converts errx errors to *connect.Error with the correct code
	//   - Attaches proto.Message details from errx.WithDetails
	//   - Appends LocalizedMessage for errx.Localizable errors
	//     based on the Accept-Language header
	fmt.Println("See cerr.NewInterceptor() for automatic error conversion in Connect handlers.")
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
