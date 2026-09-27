package stringutils

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestIndexFold(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 9, IndexFold("Shipper: JUNIPER Mfg", "juniper"))
	assert.Equal(t, -1, IndexFold("Shipper: Juniper", "granite"))
	assert.Equal(t, -1, IndexFold("anything", ""))
	assert.Equal(t, 0, IndexFold("Ⱥbc", "Ⱥbc"), "a length-changing fold still matches exactly")
}

func TestExcerptAround(t *testing.T) {
	t.Parallel()

	text := "Load # 8393\n\nShipper:   Juniper Manufacturing, Hartwell CA"
	start := IndexFold(text, "juniper")
	end := start + len("juniper")

	assert.Equal(t, "3 Shipper: Juniper Manufacturing", ExcerptAround(text, start, end, 14))
	assert.Equal(t, CollapseWhitespace(text), ExcerptAround(text, start, end, 500))
	assert.Empty(t, ExcerptAround(text, 10, 5, 3))
	assert.Empty(t, ExcerptAround(text, -1, 5, 3))
	assert.Empty(t, ExcerptAround(text, 0, len(text)+1, 3))
}

func TestExcerptAroundKeepsWholeRunes(t *testing.T) {
	t.Parallel()

	text := "ééé rate 100 ééé"
	start := IndexFold(text, "100")

	for context := range 8 {
		excerpt := ExcerptAround(text, start, start+3, context)
		assert.Contains(t, excerpt, "100")
		assert.True(t, utf8.ValidString(excerpt), "context %d cut a rune: %q", context, excerpt)
	}
}
