package htmlutils

import (
	"strings"
	"unicode"
)

const (
	divergentShare    = 0.2
	divergentMinWords = 5
	minComparedLength = 3
)

// PreferredText is the text of a message that came with both a text/plain
// and an HTML part. The plain part is used while it says what the HTML
// shows. When a substantial share of its words appear nowhere in the HTML a
// reader sees, the plain part is carrying something the reader never saw, so
// the text read from the HTML is used instead and divergent is true.
func PreferredText(plain, document string) (text string, divergent bool) {
	if strings.TrimSpace(document) == "" {
		return plain, false
	}
	shown := ToText(document)
	if strings.TrimSpace(plain) == "" {
		return shown, false
	}

	visible := wordSet(shown)
	unseen, compared := 0, 0
	for _, word := range words(plain) {
		compared++
		if _, ok := visible[word]; !ok {
			unseen++
		}
	}
	if unseen >= divergentMinWords && float64(unseen) > divergentShare*float64(compared) {
		return shown, true
	}
	return plain, false
}

func words(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := fields[:0]
	for _, field := range fields {
		if len([]rune(field)) >= minComparedLength {
			out = append(out, field)
		}
	}
	return out
}

func wordSet(text string) map[string]struct{} {
	list := words(text)
	set := make(map[string]struct{}, len(list))
	for _, word := range list {
		set[word] = struct{}{}
	}
	return set
}
