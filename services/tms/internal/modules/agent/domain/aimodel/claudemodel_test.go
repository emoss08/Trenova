package aimodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseClaudeModel_ReadsEveryFormAnEndpointUses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		model string
		want  ClaudeModel
	}{
		{"claude-opus-5-5", ClaudeModel{Family: ClaudeFamilyOpus, Version: 55}},
		{"claude-sonnet-5", ClaudeModel{Family: ClaudeFamilySonnet, Version: 50}},
		{"claude-haiku-4-5-20251001", ClaudeModel{Family: ClaudeFamilyHaiku, Version: 45}},
		{"anthropic.claude-opus-5-5", ClaudeModel{Family: ClaudeFamilyOpus, Version: 55}},
		{
			"us.anthropic.claude-sonnet-4-6-v1:0",
			ClaudeModel{Family: ClaudeFamilySonnet, Version: 46},
		},
		{"claude-opus-4-6@20260205", ClaudeModel{Family: ClaudeFamilyOpus, Version: 46}},
		{"anthropic/claude-fable-5-1", ClaudeModel{Family: ClaudeFamilyFable, Version: 51}},
		{"  Claude-Mythos-5-1  ", ClaudeModel{Family: ClaudeFamilyMythos, Version: 51}},
	}
	for _, tc := range cases {
		got, ok := ParseClaudeModel(tc.model)
		assert.True(t, ok, tc.model)
		assert.Equal(t, tc.want, got, tc.model)
	}
}

func TestParseClaudeModel_RefusesWhatItCannotRead(t *testing.T) {
	t.Parallel()

	for _, model := range []string{
		"",
		"claude-3-7-sonnet-20250219",
		"claude-latest",
		"gpt-5-mini",
		"house-claude",
	} {
		_, ok := ParseClaudeModel(model)
		assert.False(t, ok, model)
	}
}

/*
The models from Opus and Sonnet 4.6, Haiku 5.5 and every Fable and Mythos take
a million tokens. Read as 200,000, a Desk thread on them was compacted at about
170,000 tokens with five times that room left.
*/
func TestContextWindowFor_GivesClaudeModelsTheirOwnWindow(t *testing.T) {
	t.Parallel()

	cases := []struct {
		model string
		want  int
	}{
		{"claude-opus-5-5", 1_000_000},
		{"claude-opus-5", 1_000_000},
		{"claude-opus-4-6", 1_000_000},
		{"claude-sonnet-5-5", 1_000_000},
		{"claude-sonnet-4-6", 1_000_000},
		{"claude-haiku-5-5", 1_000_000},
		{"claude-fable-5", 1_000_000},
		{"claude-mythos-5-1", 1_000_000},
		{"anthropic.claude-opus-5-5", 1_000_000},
		{"us.anthropic.claude-sonnet-4-6-v1:0", 1_000_000},
		{"claude-opus-4-6@20260205", 1_000_000},
		{"claude-opus-4-5", 200_000},
		{"claude-sonnet-4-5", 200_000},
		{"claude-haiku-4-5-20251001", 200_000},
		{"claude-opus-4-1", 200_000},
		{"claude-3-7-sonnet-20250219", 200_000},
		{"claude-latest", 200_000},
		{"claude-sketch-6", 200_000},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, ContextWindowFor(tc.model), tc.model)
	}
}
