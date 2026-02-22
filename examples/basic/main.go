package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/mickamy/errx"
)

// Domain-specific codes.
const (
	PaymentRequired errx.Code = "payment_required"
)

// Domain-specific sentinels.
var (
	ErrNotFound        = errx.NewSentinel("not found", errx.NotFound)
	ErrPaymentRequired = errx.NewSentinel("payment required", PaymentRequired)
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	logger.Info("=== 1. Basic error with fields ===")
	if err := createUser(""); err != nil {
		logger.Error("failed to create user", errx.SlogAttr(err))
	}

	logger.Info("=== 2. Wrapped sentinel ===")
	if err := getUser(999); err != nil {
		logger.Error("failed to get user",
			errx.SlogAttr(err),
			"recovered", errors.Is(err, ErrNotFound),
		)
	}

	logger.Info("=== 3. Code override ===")
	if err := processPayment("user-1", 5000); err != nil {
		logger.Error("payment failed",
			errx.SlogAttr(err),
			"code", errx.CodeOf(err),
		)
	}

	logger.Info("=== 4. Stack trace ===")
	if err := deepCall(); err != nil {
		stack := errx.StackOf(err)
		if stack != nil {
			frames := stack.Frames()
			logger.Error("deep failure",
				errx.SlogAttr(err),
				"top_frame", frames[0].Function,
			)
		}
	}

	logger.Info("=== 5. LogValuer (pass *Error directly) ===")
	err := errx.New("timeout", "endpoint", "/api/users", "latency_ms", 3200).
		WithCode(errx.DeadlineExceeded).
		WithStack()
	logger.Error("request failed", "error", err)
}

func createUser(name string) error {
	if name == "" {
		return errx.New("validation failed", "field", "name", "reason", "empty").
			WithCode(errx.InvalidArgument)
	}
	return nil
}

func getUser(id int) error {
	err := queryUser(id)
	if err != nil {
		return errx.Wrap(err, "user_id", id)
	}
	return nil
}

func queryUser(_ int) error {
	// Simulate a not-found from the data layer.
	return ErrNotFound
}

func processPayment(userID string, amount int) error {
	err := chargeCreditCard(userID, amount)
	if err != nil {
		// Override the inner code with a domain-specific one.
		return errx.Wrap(err, "user_id", userID, "amount", amount).
			WithCode(PaymentRequired)
	}
	return nil
}

func chargeCreditCard(_ string, _ int) error {
	return errx.New("card declined").WithCode(errx.FailedPrecondition)
}

func deepCall() error {
	return errx.Wrapf(innerCall(), "deep call failed").WithStack()
}

func innerCall() error {
	return errx.New("something broke").WithCode(errx.Internal)
}
