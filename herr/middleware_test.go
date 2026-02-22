package herr_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mickamy/errx"
	"github.com/mickamy/errx/herr"
)

func TestHandler_Success(t *testing.T) {
	t.Parallel()

	h := herr.Handler(func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return nil
	})

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", w.Body.String(), "ok")
	}
}

func TestHandler_ErrxError(t *testing.T) {
	t.Parallel()

	h := herr.Handler(func(_ http.ResponseWriter, _ *http.Request) error {
		return errx.New("user not found").WithCode(errx.NotFound)
	})

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q", ct)
	}

	var p herr.ProblemDetail
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.Code != "not_found" {
		t.Errorf("Code = %q, want %q", p.Code, "not_found")
	}
	if p.Detail != "user not found" {
		t.Errorf("Detail = %q, want %q", p.Detail, "user not found")
	}
}

func TestHandler_WithDetails(t *testing.T) {
	t.Parallel()

	h := herr.Handler(func(_ http.ResponseWriter, _ *http.Request) error {
		return errx.New("bad request").
			WithCode(errx.InvalidArgument).
			WithDetails(errx.FieldViolation("email", "invalid"))
	})

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var p herr.ProblemDetail
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if len(p.Errors) != 1 {
		t.Fatalf("errors length = %d, want 1", len(p.Errors))
	}
	if p.Errors[0]["type"] != "BadRequest" {
		t.Errorf("error type = %v, want BadRequest", p.Errors[0]["type"])
	}
}

type localizableError struct {
	messages map[string]string
}

func (e *localizableError) Error() string { return "validation error" }

func (e *localizableError) Localize(locale string) string {
	if msg, ok := e.messages[locale]; ok {
		return msg
	}
	return e.messages["en"]
}

func TestHandler_Localizable(t *testing.T) {
	t.Parallel()

	h := herr.Handler(func(_ http.ResponseWriter, _ *http.Request) error {
		return errx.Wrap(&localizableError{
			messages: map[string]string{
				"en": "Name is required",
				"ja": "名前は必須です", //nolint:gosmopolitan // test i18n
			},
		}).WithCode(errx.InvalidArgument)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "ja")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var p herr.ProblemDetail
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.LocalizedMessage == nil {
		t.Fatal("localized_message should not be nil")
	}
	if p.LocalizedMessage.Locale != "ja" {
		t.Errorf("locale = %q, want %q", p.LocalizedMessage.Locale, "ja")
	}
	if p.LocalizedMessage.Message != "名前は必須です" { //nolint:gosmopolitan // test i18n
		t.Errorf("message = %q", p.LocalizedMessage.Message)
	}
}

func TestHandler_Localizable_NoHeader(t *testing.T) {
	t.Parallel()

	h := herr.Handler(func(_ http.ResponseWriter, _ *http.Request) error {
		return errx.Wrap(&localizableError{
			messages: map[string]string{"en": "Name is required"},
		}).WithCode(errx.InvalidArgument)
	})

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	var p herr.ProblemDetail
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.LocalizedMessage != nil {
		t.Error("localized_message should be nil when no Accept-Language")
	}
}

func TestHandler_PlainError(t *testing.T) {
	t.Parallel()

	h := herr.Handler(func(_ http.ResponseWriter, _ *http.Request) error {
		return errors.New("something broke")
	})

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var p herr.ProblemDetail
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.Code != "unknown" {
		t.Errorf("Code = %q, want %q", p.Code, "unknown")
	}
}

func TestHandler_WithLocaleFunc(t *testing.T) {
	t.Parallel()

	h := herr.Handler(
		func(_ http.ResponseWriter, _ *http.Request) error {
			return errx.Wrap(&localizableError{
				messages: map[string]string{
					"en":    "Name is required",
					"ja-JP": "名前は必須です", //nolint:gosmopolitan // test i18n
				},
			}).WithCode(errx.InvalidArgument)
		},
		herr.WithLocaleFunc(func(h http.Header) string {
			return h.Get("X-Locale")
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Locale", "ja-JP")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var p herr.ProblemDetail
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.LocalizedMessage == nil {
		t.Fatal("localized_message should not be nil")
	}
	if p.LocalizedMessage.Locale != "ja-JP" {
		t.Errorf("locale = %q, want %q", p.LocalizedMessage.Locale, "ja-JP")
	}
}
