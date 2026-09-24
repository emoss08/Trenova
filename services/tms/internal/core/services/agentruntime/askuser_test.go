package agentruntime

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeAsk(t *testing.T, content string) askRequest {
	t.Helper()

	open := strings.Index(content, untrustedOpenTag)
	close := strings.LastIndex(content, untrustedCloseTag)
	require.Positive(t, open, "an ask is fenced like any other tool result")
	require.Greater(t, close, open)

	body := strings.TrimSpace(content[open+len(untrustedOpenTag) : close])

	var request askRequest
	require.NoError(t, sonic.Unmarshal([]byte(body), &request))

	return request
}

func TestResolveAsk_CarriesTheQuestionAndItsChoices(t *testing.T) {
	t.Parallel()

	content := resolveAsk(map[string]any{
		"question": "Which window should the report cover?",
		"options": []any{
			map[string]any{"value": "7", "label": "7 days", "detail": "the last week"},
			map[string]any{"value": "30", "label": "30 days"},
		},
		"otherHint": "Number of days",
	})

	request := decodeAsk(t, content)
	assert.Equal(t, "Which window should the report cover?", request.Question)
	require.Len(t, request.Options, 2)
	assert.Equal(
		t,
		askOption{Value: "7", Label: "7 days", Detail: "the last week"},
		request.Options[0],
	)
	assert.Equal(t, "Number of days", request.OtherHint)
	assert.True(t, request.AllowOther, "a person can type their own value unless told otherwise")
}

// The whole point of asking is that the model does not know the answer. A note
// that did not say to stop would let it ask and then answer itself, which is
// what it did before the tool existed.
func TestResolveAsk_TellsTheModelToStopAndWait(t *testing.T) {
	t.Parallel()

	request := decodeAsk(t, resolveAsk(map[string]any{
		"question": "Which customer?",
		"options":  []any{map[string]any{"value": "acme", "label": "Acme"}},
	}))

	assert.Contains(t, request.Note, "End your turn now")
	assert.Contains(t, request.Note, "Do not choose for them")
}

func TestResolveAsk_HonoursAClosedSetOfChoices(t *testing.T) {
	t.Parallel()

	request := decodeAsk(t, resolveAsk(map[string]any{
		"question":   "Which status?",
		"allowOther": false,
		"options": []any{
			map[string]any{"value": "Active", "label": "Active"},
			map[string]any{"value": "Inactive", "label": "Inactive"},
		},
	}))

	assert.False(t, request.AllowOther)
	assert.NotContains(t, request.Note, "their own value")
}

func TestResolveAsk_RefusesAQuestionWithNothingToAnswerIt(t *testing.T) {
	t.Parallel()

	content := resolveAsk(map[string]any{
		"question":   "Which one?",
		"options":    []any{},
		"allowOther": false,
	})

	assert.Contains(t, content, "either options to choose from or allowOther")
	assert.NotContains(t, content, untrustedOpenTag, "there is no question to render")
}

func TestResolveAsk_RefusesAnEmptyQuestion(t *testing.T) {
	t.Parallel()

	assert.Contains(t, resolveAsk(map[string]any{"question": "   "}), "needs a question")
}

// A model that offers twenty choices has made a search, not a decision, and a
// malformed entry must not cost the question its usable options.
func TestResolveAsk_BoundsAndCleansTheOptions(t *testing.T) {
	t.Parallel()

	options := []any{
		map[string]any{"value": "keep", "label": "Keep"},
		map[string]any{"value": "keep", "label": "Duplicate"},
		map[string]any{"value": "  ", "label": "  "},
		map[string]any{"detail": "Neither a value nor a label"},
		"not an option at all",
		map[string]any{"value": "bare"},
		map[string]any{"value": "long", "label": strings.Repeat("x", 200)},
	}
	for range maxAskOptions {
		options = append(options, map[string]any{"value": strings.Repeat("a", len(options))})
	}

	request := decodeAsk(t, resolveAsk(map[string]any{
		"question": "Which one?",
		"options":  options,
	}))

	assert.Len(t, request.Options, maxAskOptions)
	values := make([]string, 0, len(request.Options))
	for _, option := range request.Options {
		values = append(values, option.Value)
		assert.NotEmpty(t, option.Label, "an option with no label is unclickable")
		assert.LessOrEqual(t, len([]rune(option.Label)), maxAskLabelChars+1)
	}
	assert.Equal(t, []string{"keep", "bare", "long"}, values[:3])
	assert.Equal(t, "bare", request.Options[1].Label, "a bare value labels itself")
}

// The question is rendered from the payload, so a number in it must not be
// rewritten into a date by the tool-result date pass.
func TestResolveAsk_DoesNotRewriteAnOptionValueAsADate(t *testing.T) {
	t.Parallel()

	request := decodeAsk(t, resolveAsk(map[string]any{
		"question": "Which cut-off date?",
		"options": []any{
			map[string]any{"value": "1791591001", "label": "10 October 2026"},
		},
	}))

	assert.Equal(t, "1791591001", request.Options[0].Value)
}

// A model offered three choices as label and detail with no value, and the
// person was shown an empty list and a text box. The label is what they read
// and what they would type, so it is what a pick answers with.
func TestResolveAsk_AnOptionWithoutAValueAnswersWithItsLabel(t *testing.T) {
	t.Parallel()

	request := decodeAsk(t, resolveAsk(map[string]any{
		"question": "Which lane should the report cover?",
		"options": []any{
			map[string]any{"label": "Chicago to Dallas", "detail": "the busiest lane"},
			map[string]any{"label": "Atlanta to Miami"},
			map[string]any{"label": " Chicago to Dallas ", "detail": "said twice"},
			map[string]any{"value": "sea-pdx", "label": "Seattle to Portland"},
		},
	}))

	require.Len(t, request.Options, 3)
	assert.Equal(
		t,
		askOption{
			Value:  "Chicago to Dallas",
			Label:  "Chicago to Dallas",
			Detail: "the busiest lane",
		},
		request.Options[0],
	)
	assert.Equal(t, "Atlanta to Miami", request.Options[1].Value)
	assert.Equal(
		t,
		askOption{Value: "sea-pdx", Label: "Seattle to Portland"},
		request.Options[2],
	)
	assert.Contains(t, request.Note, "with 3 options")
}

// The schema is what a strict provider validates against, so it must not
// demand the value the runtime fills in itself.
func TestAskUserSpec_RequiresOnlyTheLabel(t *testing.T) {
	t.Parallel()

	spec := askUserSpec()
	properties, ok := spec.Parameters["properties"].(map[string]any)
	require.True(t, ok)
	options, ok := properties["options"].(map[string]any)
	require.True(t, ok)
	items, ok := options["items"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, []string{"label"}, items["required"])
}
