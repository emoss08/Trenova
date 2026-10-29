package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextUsage_CompactsItselfOnlyPastTheShareAndWithEnoughToFree(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		usage *ContextUsage
		want  bool
	}{
		{"nothing measured", nil, false},
		{
			"below the share",
			&ContextUsage{
				Instructions: 6000,
				Messages:     100_000,
				Compactable:  90_000,
				Window:       200_000,
			},
			false,
		},
		{
			"past the share with plenty behind the latest turns",
			&ContextUsage{
				Instructions: 6000, Messages: 40_000, ToolResults: 130_000,
				Compactable: 150_000, Window: 200_000,
			},
			true,
		},
		{
			"past the share but all of it in the latest turns",
			&ContextUsage{
				Instructions: 6000,
				ToolResults:  170_000,
				Compactable:  3000,
				Window:       200_000,
			},
			false,
		},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, tc.usage.NeedsCompaction(), tc.name)
	}
}

func TestContextUsage_FreesTheStretchLessItsSummary(t *testing.T) {
	t.Parallel()

	usage := &ContextUsage{Compactable: 100_000}
	assert.Equal(t, 100_000-SummaryTokenBudget, usage.Frees())

	small := &ContextUsage{Compactable: 5000}
	assert.Equal(t, 4000, small.Frees(), "a short stretch is expected to summarize to a fifth")
	assert.False(
		t,
		small.WorthCompacting(),
		"freeing no more than the minimum is not worth a model call",
	)
}
