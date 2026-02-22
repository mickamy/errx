package errx_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/mickamy/errx"
)

func TestLogValue(t *testing.T) {
	t.Parallel()

	err := errx.New("db error", "table", "users").WithCode(errx.Internal)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		// Remove time for deterministic output.
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	logger.Info("operation failed", "error", err)

	var m map[string]any
	if jsonErr := json.Unmarshal(buf.Bytes(), &m); jsonErr != nil {
		t.Fatalf("failed to parse JSON: %v\nbody: %s", jsonErr, buf.String())
	}

	errObj, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error to be an object, got: %v", m["error"])
	}
	if errObj["msg"] != "db error" {
		t.Errorf("msg = %v, want %q", errObj["msg"], "db error")
	}
	if errObj["code"] != "internal" {
		t.Errorf("code = %v, want %q", errObj["code"], "internal")
	}
	if errObj["table"] != "users" {
		t.Errorf("table = %v, want %q", errObj["table"], "users")
	}
}

func TestLogValue_NoCode(t *testing.T) {
	t.Parallel()

	err := errx.New("plain error")

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Info("msg", "error", err)

	var m map[string]any
	if jsonErr := json.Unmarshal(buf.Bytes(), &m); jsonErr != nil {
		t.Fatalf("failed to parse JSON: %v", jsonErr)
	}

	errObj := m["error"].(map[string]any)
	if _, exists := errObj["code"]; exists {
		t.Error("code should not be present when unset")
	}
}

func TestSlogAttr(t *testing.T) {
	t.Parallel()

	inner := errx.New("inner", "key1", "val1").WithCode(errx.NotFound)
	outer := errx.Wrap(inner, "key2", "val2")

	attr := errx.SlogAttr(outer)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	logger.Info("test", attr)

	var m map[string]any
	if jsonErr := json.Unmarshal(buf.Bytes(), &m); jsonErr != nil {
		t.Fatalf("failed to parse JSON: %v\nbody: %s", jsonErr, buf.String())
	}

	errObj, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error group, got: %v", m["error"])
	}
	if errObj["msg"] != "inner" {
		t.Errorf("msg = %v, want %q", errObj["msg"], "inner")
	}
	if errObj["code"] != "not_found" {
		t.Errorf("code = %v, want %q", errObj["code"], "not_found")
	}
	if errObj["key2"] != "val2" {
		t.Errorf("key2 = %v, want %q", errObj["key2"], "val2")
	}
	if errObj["key1"] != "val1" {
		t.Errorf("key1 = %v, want %q", errObj["key1"], "val1")
	}
}

func TestSlogAttr_Nil(t *testing.T) {
	t.Parallel()

	attr := errx.SlogAttr(nil)
	if attr.Key != "" {
		t.Errorf("SlogAttr(nil) key = %q, want empty", attr.Key)
	}
}

func TestLogValue_WithStack(t *testing.T) {
	t.Parallel()

	err := errx.New("fail").WithStack()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Info("msg", "error", err)

	var m map[string]any
	if jsonErr := json.Unmarshal(buf.Bytes(), &m); jsonErr != nil {
		t.Fatalf("failed to parse JSON: %v", jsonErr)
	}

	errObj := m["error"].(map[string]any)
	caller, ok := errObj["caller"].(map[string]any)
	if !ok {
		t.Fatalf("expected caller group, got: %v", errObj["caller"])
	}
	if caller["function"] == nil || caller["function"] == "" {
		t.Error("caller.function should be populated")
	}
}
