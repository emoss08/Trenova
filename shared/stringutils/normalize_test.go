package stringutils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeIdentifier(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ABC123", NormalizeIdentifier("Abc-123"))
	assert.Equal(t, "INV2024001", NormalizeIdentifier("INV 2024/001"))
	assert.Equal(t, "", NormalizeIdentifier(" -_/ "))
}

func TestTruncateRunes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		maxLength int
		want      string
	}{
		{name: "shorter than max", value: "ABC", maxLength: 5, want: "ABC"},
		{name: "equal to max", value: "ABC", maxLength: 3, want: "ABC"},
		{name: "longer than max", value: "ABCDEFG", maxLength: 3, want: "ABC"},
		{name: "unicode safe", value: "ÅBCDEF", maxLength: 3, want: "ÅBC"},
		{name: "zero max", value: "ABC", maxLength: 0, want: ""},
		{name: "negative max", value: "ABC", maxLength: -1, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, TruncateRunes(tt.value, tt.maxLength))
		})
	}
}

func TestFirstSentence(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Look up one shipment.", FirstSentence("Look up one shipment. Returns stops."))
	assert.Equal(t, "No stop here", FirstSentence("No stop here"))
	assert.Equal(
		t,
		".x",
		FirstSentence(".x"),
		"a leading stop is not a sentence and the text is returned whole",
	)
}

func TestCollapseWhitespace(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Shipments for Peak", CollapseWhitespace("  Shipments\nfor\t\tPeak \r\n"))
	assert.Empty(t, CollapseWhitespace(" \n\t "))
}

func TestIsCompactToken(t *testing.T) {
	t.Parallel()

	for value, want := range map[string]bool{
		"7f1c2a-9":             true,
		"a":                    true,
		"":                     false,
		"has space":            false,
		"tab\there":            false,
		"line\nbreak":          false,
		"bell\a":               false,
		strings.Repeat("x", 8): true,
		strings.Repeat("x", 9): false,
	} {
		assert.Equal(t, want, IsCompactToken(value, 8), value)
	}
	assert.False(t, IsCompactToken("abc", 0))
}
