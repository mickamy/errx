package errx_test

import (
	"testing"

	"github.com/mickamy/errx"
)

func TestParseAcceptLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "single language", input: "ja", want: "ja"},
		{name: "single language with region", input: "en-US", want: "en-US"},
		{name: "multiple with quality values", input: "ja,en-US;q=0.9,en;q=0.8", want: "ja"},
		{name: "highest quality not first", input: "en;q=0.8,ja", want: "ja"},
		{name: "explicit quality 1", input: "fr;q=1.0,de;q=0.9", want: "fr"},
		{name: "malformed input", input: "not a valid header!!!", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := errx.ParseAcceptLanguage(tt.input)
			if got != tt.want {
				t.Errorf("ParseAcceptLanguage(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
