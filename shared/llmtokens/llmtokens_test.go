package llmtokens_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/llmtokens"
	"github.com/stretchr/testify/assert"
)

func TestFromRunes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		runes int
		want  int
	}{
		{name: "negative counts nothing", runes: -3, want: 0},
		{name: "zero counts nothing", runes: 0, want: 0},
		{name: "one rune is a whole token", runes: 1, want: 1},
		{name: "exactly one token", runes: 4, want: 1},
		{name: "rounds up past a boundary", runes: 5, want: 2},
		{name: "large count", runes: 4000, want: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, llmtokens.FromRunes(tt.runes))
		})
	}
}

func TestEstimateCountsRunesNotBytes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, llmtokens.Estimate("日本語"))
	assert.Equal(t, 2, llmtokens.Estimate("日本語です"))
	assert.Zero(t, llmtokens.Estimate(""))
}

func TestEstimateAllSumsEachTextRoundedUp(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 4, llmtokens.EstimateAll([]string{"a", "bb", "ccccc"}))
	assert.Zero(t, llmtokens.EstimateAll(nil))
}
