package modeladapter_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sample struct {
	Name  string   `json:"name"`
	Count int      `json:"count"`
	Tags  []string `json:"tags"`
}

func TestExtractJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want sample
	}{
		{
			name: "clean object from a schema-enforcing provider",
			raw:  `{"name":"alpha","count":2,"tags":["a","b"]}`,
			want: sample{Name: "alpha", Count: 2, Tags: []string{"a", "b"}},
		},
		{
			name: "markdown fence with language tag",
			raw:  "```json\n{\"name\":\"alpha\",\"count\":2}\n```",
			want: sample{Name: "alpha", Count: 2},
		},
		{
			name: "markdown fence without language tag",
			raw:  "```\n{\"name\":\"alpha\",\"count\":2}\n```",
			want: sample{Name: "alpha", Count: 2},
		},
		{
			name: "preamble before the object",
			raw:  "Here is the JSON you asked for:\n{\"name\":\"alpha\",\"count\":2}",
			want: sample{Name: "alpha", Count: 2},
		},
		{
			name: "commentary on both sides",
			raw:  "Sure!\n{\"name\":\"alpha\",\"count\":2}\nLet me know if you need anything else.",
			want: sample{Name: "alpha", Count: 2},
		},
		{
			name: "fence plus preamble",
			raw:  "Thinking done.\n\n```json\n{\"name\":\"alpha\",\"count\":2}\n```\n\nDone.",
			want: sample{Name: "alpha", Count: 2},
		},
		{
			name: "trailing comma before closing brace",
			raw:  `{"name":"alpha","count":2,}`,
			want: sample{Name: "alpha", Count: 2},
		},
		{
			name: "trailing comma in nested array",
			raw:  `{"name":"alpha","tags":["a","b",],}`,
			want: sample{Name: "alpha", Tags: []string{"a", "b"}},
		},
		{
			name: "braces inside string values do not end the scan",
			raw:  `prose {"name":"a } b {","count":3} more prose`,
			want: sample{Name: "a } b {", Count: 3},
		},
		{
			name: "escaped quotes inside string values",
			raw:  `{"name":"say \"hi\"","count":1}`,
			want: sample{Name: `say "hi"`, Count: 1},
		},
		{
			name: "comma inside a string is not treated as trailing",
			raw:  `{"name":"a,","count":1}`,
			want: sample{Name: "a,", Count: 1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got sample
			require.NoError(t, modeladapter.ExtractJSON(tc.raw, &got))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestExtractJSON_Failures(t *testing.T) {
	t.Parallel()

	t.Run("empty response", func(t *testing.T) {
		t.Parallel()
		var got sample
		assert.ErrorIs(t, modeladapter.ExtractJSON("", &got), modeladapter.ErrNoJSONObject)
	})

	t.Run("whitespace only", func(t *testing.T) {
		t.Parallel()
		var got sample
		assert.ErrorIs(t, modeladapter.ExtractJSON("   \n\t ", &got), modeladapter.ErrNoJSONObject)
	})

	t.Run("prose with no object", func(t *testing.T) {
		t.Parallel()
		var got sample
		assert.Error(t, modeladapter.ExtractJSON("I cannot help with that.", &got))
	})

	t.Run("unterminated object", func(t *testing.T) {
		t.Parallel()
		var got sample
		assert.Error(t, modeladapter.ExtractJSON(`{"name":"alpha"`, &got))
	})
}

func TestExtractJSON_ProposalPayloadWithTrailingCommas(t *testing.T) {
	t.Parallel()

	raw := "```json\n" + `{
  "proposals": [
    {
      "toolName": "correct_charge_code",
      "toolParams": {"chargeCode": "DET"},
      "confidence": 0.82,
      "rationale": "Detention was billed under the wrong code.",
      "evidence": [{"type": "document", "id": "doc_123"}],
    }
  ],
  "exceptions": [],
}` + "\n```"

	var payload struct {
		Proposals []struct {
			ToolName   string           `json:"toolName"`
			ToolParams map[string]any   `json:"toolParams"`
			Confidence float64          `json:"confidence"`
			Evidence   []map[string]any `json:"evidence"`
		} `json:"proposals"`
		Exceptions []map[string]any `json:"exceptions"`
	}
	require.NoError(t, modeladapter.ExtractJSON(raw, &payload))

	require.Len(t, payload.Proposals, 1)
	assert.Equal(t, "correct_charge_code", payload.Proposals[0].ToolName)
	assert.InDelta(t, 0.82, payload.Proposals[0].Confidence, 0.001)
	require.Len(t, payload.Proposals[0].Evidence, 1)
	assert.Empty(t, payload.Exceptions)
}
