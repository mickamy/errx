package errx

import "golang.org/x/text/language"

// ParseAcceptLanguage parses an Accept-Language header value and returns
// the highest-priority language tag as a BCP 47 string.
// It returns an empty string if the input is empty or cannot be parsed.
func ParseAcceptLanguage(s string) string {
	tags, qs, err := language.ParseAcceptLanguage(s)
	if err != nil || len(tags) == 0 {
		return ""
	}
	best := 0
	for i := 1; i < len(tags); i++ {
		if qs[i] > qs[best] {
			best = i
		}
	}
	return tags[best].String()
}
