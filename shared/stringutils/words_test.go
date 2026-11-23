package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsWords_MatchesOnlyWholeWords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		text  string
		words string
		want  bool
	}{
		{text: "SEED-SHP-010 and SEED-SHP-008 compared", words: "SEED-SHP-008", want: true},
		{text: "SEED-SHP-0080 is another load", words: "SEED-SHP-008", want: false},
		{text: "XSEED-SHP-008 is another load", words: "SEED-SHP-008", want: false},
		{text: "Leon Parkinson, then Leon Park.", words: "Leon Park", want: true},
		{text: "anything", words: "", want: false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ContainsWords(tt.text, tt.words), tt.text)
	}
	assert.Equal(t, 4, IndexWords("the SEED-SHP-008 load", "SEED-SHP-008"))
}
