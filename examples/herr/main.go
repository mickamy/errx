package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/mickamy/errx"
	"github.com/mickamy/errx/herr"
)

var ErrUserNotFound = errx.NewSentinel("user not found", errx.NotFound)

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

	mux := http.NewServeMux()

	// Register handlers using herr.Handler.
	mux.Handle("GET /hello", herr.Handler(handleHello))

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// 1. Successful call.
	logger.Info("=== 1. Successful call ===")
	doRequest(logger, srv, "/hello?name=Alice", "")

	// 2. InvalidArgument with FieldViolation.
	logger.Info("=== 2. InvalidArgument + FieldViolation ===")
	doRequest(logger, srv, "/hello?name=", "")

	// 3. NotFound with ResourceInfo.
	logger.Info("=== 3. NotFound + ResourceInfo ===")
	doRequest(logger, srv, "/hello?name=unknown", "")

	// 4. PermissionDenied.
	logger.Info("=== 4. PermissionDenied ===")
	doRequest(logger, srv, "/hello?name=admin", "")

	// 5. Localizable error with Accept-Language.
	logger.Info("=== 5. Localizable + FieldViolation (ja) ===")
	doRequest(logger, srv, "/hello?name=validate", "ja")

	// 6. Round-trip: HTTP response → errx.
	logger.Info("=== 6. Round-trip (HTTP → errx) ===")
	resp := doRequestRaw(srv, "/hello?name=unknown", "")
	var p herr.ProblemDetail
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		logger.Error("decode error", "error", err)
		return
	}
	_ = resp.Body.Close()
	recovered := herr.FromProblemDetail(&p)
	logger.Info("recovered errx error",
		"code", recovered.Code(),
		"message", recovered.Error(),
	)
}

func handleHello(w http.ResponseWriter, r *http.Request) error {
	name := r.URL.Query().Get("name")

	switch name {
	case "":
		return errx.New("name is required").
			WithCode(errx.InvalidArgument).
			WithDetails(errx.FieldViolation("name", "must not be empty"))
	case "unknown":
		return errx.Wrap(ErrUserNotFound).
			WithDetails(errx.ResourceInfo("User", name, "", "user not found"))
	case "admin":
		return errx.New("admin access denied", "name", name).
			WithCode(errx.PermissionDenied)
	case "validate":
		return errx.Wrap(&validationError{
			field: "name",
			messages: map[string]string{
				"en": "Name is required",
				"ja": "名前は必須です", //nolint:gosmopolitan // example i18n
			},
		}).WithCode(errx.InvalidArgument).
			WithDetails(errx.FieldViolation("name", "must not be empty"))
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "Hello %s\n", name) //nolint:gosec // example code
	return nil
}

func doRequest(logger *slog.Logger, srv *httptest.Server, path, lang string) {
	resp := doRequestRaw(srv, path, lang)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		logger.Info("response", "status", resp.StatusCode)
		return
	}

	var p herr.ProblemDetail
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		logger.Error("decode error", "error", err)
		return
	}

	attrs := []any{
		"status", p.Status,
		"code", p.Code,
		"detail", p.Detail,
	}
	if p.LocalizedMessage != nil {
		attrs = append(attrs, "localized_"+p.LocalizedMessage.Locale, p.LocalizedMessage.Message)
	}
	for _, e := range p.Errors {
		attrs = append(attrs, "error_type", e["type"])
	}
	logger.Error("HTTP error", attrs...)
}

func doRequestRaw(srv *httptest.Server, path, lang string) *http.Response {
	req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil) //nolint:noctx // example code
	if err != nil {
		panic(err)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // example code
	if err != nil {
		panic(err)
	}
	return resp
}
