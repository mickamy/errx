package errx_test

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/mickamy/errx"
)

func Example() {
	// Create a structured error with fields.
	err := errx.New("user not found", "user_id", 42).WithCode(errx.NotFound)
	fmt.Println(err)
	fmt.Println("code:", errx.CodeOf(err))
	// Output:
	// user not found
	// code: not_found
}

func Example_wrap() {
	cause := errors.New("connection refused")
	err := errx.Wrapf(cause, "connect to %s", "db:5432").
		With("retry", 3).
		WithCode(errx.Unavailable)

	fmt.Println(err)
	fmt.Println("code:", errx.CodeOf(err))
	fmt.Println("is cause:", errors.Is(err, cause))
	// Output:
	// connect to db:5432: connection refused
	// code: unavailable
	// is cause: true
}

func Example_sentinel() {
	var ErrPaymentRequired = errx.NewSentinel("payment required", errx.Code("payment_required"))

	err := errx.Wrap(ErrPaymentRequired, "plan", "premium")

	fmt.Println(errors.Is(err, ErrPaymentRequired))
	fmt.Println("code:", errx.CodeOf(err))
	// Output:
	// true
	// code: payment_required
}

func Example_slogAttr() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			// Remove time for deterministic output.
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	err := errx.New("db timeout", "query", "SELECT 1").WithCode(errx.Internal)
	logger.Error("operation failed", errx.SlogAttr(err))
	// Output:
	// {"level":"ERROR","msg":"operation failed","error":{"msg":"db timeout","code":"internal","query":"SELECT 1"}}
}
