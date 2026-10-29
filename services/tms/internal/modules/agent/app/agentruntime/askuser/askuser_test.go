package askuser

import (
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var askNow = time.Date(2026, time.September, 29, 10, 0, 0, 0, time.UTC)

func askContent(now time.Time, arguments map[string]any) string {
	content, _ := Resolve(now, arguments)

	return content
}

func askBody(t *testing.T, content string) string {
	t.Helper()

	open := strings.Index(content, toolresult.UntrustedOpenTag)
	close := strings.LastIndex(content, toolresult.UntrustedCloseTag)
	require.Positive(t, open, "an ask is fenced like any other tool result")
	require.Greater(t, close, open)

	return strings.TrimSpace(content[open+len(toolresult.UntrustedOpenTag) : close])
}

func decodeAsk(t *testing.T, content string) askRequest {
	t.Helper()

	var request askRequest
	require.NoError(t, sonic.Unmarshal([]byte(askBody(t, content)), &request))

	return request
}

func TestResolveAsk_CarriesTheQuestionAndItsChoices(t *testing.T) {
	t.Parallel()

	content := askContent(askNow, map[string]any{
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

	request := decodeAsk(t, askContent(askNow, map[string]any{
		"question": "Which customer?",
		"options":  []any{map[string]any{"value": "acme", "label": "Acme"}},
	}))

	assert.Contains(t, request.Note, "End your turn now")
	assert.Contains(t, request.Note, "Do not choose for them")
}

func TestResolveAsk_HonoursAClosedSetOfChoices(t *testing.T) {
	t.Parallel()

	request := decodeAsk(t, askContent(askNow, map[string]any{
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

	content := askContent(askNow, map[string]any{
		"question":   "Which one?",
		"options":    []any{},
		"allowOther": false,
	})

	assert.Contains(t, content, "either options to choose from or allowOther")
	assert.NotContains(t, content, toolresult.UntrustedOpenTag, "there is no question to render")
}

func TestResolveAsk_RefusesAnEmptyQuestion(t *testing.T) {
	t.Parallel()

	assert.Contains(t, askContent(askNow, map[string]any{"question": "   "}), "needs a question")
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

	request := decodeAsk(t, askContent(askNow, map[string]any{
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

	request := decodeAsk(t, askContent(askNow, map[string]any{
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

	request := decodeAsk(t, askContent(askNow, map[string]any{
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

	spec := Spec()
	properties, ok := spec.Parameters["properties"].(map[string]any)
	require.True(t, ok)
	options, ok := properties["options"].(map[string]any)
	require.True(t, ok)
	items, ok := options["items"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, []string{"label"}, items["required"])
}

func TestResolveAsk_SaysHowFarAwayEachDateIs(t *testing.T) {
	t.Parallel()

	request := decodeAsk(t, askContent(askNow, map[string]any{
		"question": "Which day should the pickup be scheduled?",
		"options": []any{
			map[string]any{"value": "2026-10-06", "label": "2026-10-06"},
			map[string]any{"label": "Oct 1"},
			map[string]any{"value": "2026-09-30T14:30", "label": "Wednesday, 2026-09-30 14:30"},
			map[string]any{"label": "Today"},
			map[string]any{"label": "Next available"},
		},
	}))

	require.Len(t, request.Options, 5)
	assert.Equal(t, "Tue Oct 6 (in 7 days)", request.Options[0].Label)
	assert.Equal(t, "2026-10-06", request.Options[0].Value, "the value is what the tool needs")
	assert.Equal(t, "Thu Oct 1 (in 2 days)", request.Options[1].Label)
	assert.Equal(t, "Oct 1", request.Options[1].Value,
		"a label that answers for itself keeps its own words as the value")
	assert.Equal(t, "Wed Sep 30, 14:30 (tomorrow)", request.Options[2].Label)
	assert.Equal(t, "Today", request.Options[3].Label)
	assert.Equal(t, "Next available", request.Options[4].Label)
	assert.NotContains(t, request.Note, "in the past")
}

func TestResolveAsk_ReadsTheDatesInThePersonsZone(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	lateEvening := time.Date(2026, time.September, 29, 23, 30, 0, 0, chicago)

	request := decodeAsk(t, askContent(lateEvening, map[string]any{
		"question": "When should it deliver?",
		"options":  []any{map[string]any{"label": "2026-09-30"}},
	}))

	assert.Equal(t, "Wed Sep 30 (tomorrow)", request.Options[0].Label)
}

func TestResolveAsk_NamesTheYearOfADateInAnotherYear(t *testing.T) {
	t.Parallel()

	request := decodeAsk(t, askContent(askNow, map[string]any{
		"question": "Which appointment?",
		"options":  []any{map[string]any{"label": "2027-01-05"}},
	}))

	assert.Equal(t, "Tue Jan 5, 2027 (in 98 days)", request.Options[0].Label)
}

func TestResolveAsk_TellsTheModelWhenASchedulingChoiceIsInThePast(t *testing.T) {
	t.Parallel()

	request := decodeAsk(t, askContent(askNow, map[string]any{
		"question": "Which day should I schedule the delivery appointment for?",
		"options": []any{
			map[string]any{"label": "2026-09-25"},
			map[string]any{"label": "2026-10-02"},
		},
	}))

	assert.Equal(t, "Fri Sep 25 (4 days ago)", request.Options[0].Label)
	assert.Contains(t, request.Note, "Fri Sep 25 (4 days ago) is in the past")
	assert.NotContains(t, request.Note, "Oct 2")
}

func TestResolveAsk_APastDateOnAnotherKindOfQuestionIsNoConcern(t *testing.T) {
	t.Parallel()

	request := decodeAsk(t, askContent(askNow, map[string]any{
		"question": "Which window should the report start from?",
		"options":  []any{map[string]any{"label": "2026-09-01"}},
	}))

	assert.Equal(t, "Tue Sep 1 (28 days ago)", request.Options[0].Label)
	assert.NotContains(t, request.Note, "in the past")
}

// The thread reads one shape. A question answered only by typing used to
// carry "options": null, which every reader had to special-case.
func TestResolveAsk_OptionsAreAnEmptyListWhenThereAreNone(t *testing.T) {
	t.Parallel()

	content, shown := Resolve(askNow, map[string]any{
		"question": "Which PO number should I use?",
	})

	require.True(t, shown)
	body := askBody(t, content)
	assert.Contains(t, body, `"options":[]`)
	assert.NotContains(t, body, `"options":null`)
	assert.NotContains(t, body, `"input"`, "plain text has no input type")
	assert.NotContains(t, body, `"fields"`)

	request := decodeAsk(t, content)
	assert.NotNil(t, request.Options)
	assert.Empty(t, request.Options)
	assert.True(t, request.AllowOther)
}

func TestResolveAsk_TypesThePersonsOwnValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      any
		allowOther any
		want       *askInput
	}{
		{
			name:  "a customer picked from a search",
			input: map[string]any{"type": "record", "recordKind": "customer"},
			want:  &askInput{Type: askTypeRecord, RecordKind: "customer"},
		},
		{
			name:  "a location, written loosely",
			input: map[string]any{"type": " Record ", "recordKind": " Location "},
			want:  &askInput{Type: askTypeRecord, RecordKind: "location"},
		},
		{
			name:  "a kind of record the registry does not know is typed as text",
			input: map[string]any{"type": "record", "recordKind": "spaceship"},
		},
		{
			name:  "a record with no kind is typed as text",
			input: map[string]any{"type": "record"},
		},
		{
			name:  "the conversation itself is never a record to pick",
			input: map[string]any{"type": "record", "recordKind": "assistant_thread"},
		},
		{
			name:  "a date and time",
			input: map[string]any{"type": "DateTime"},
			want:  &askInput{Type: askTypeDateTime},
		},
		{
			name:  "a kind only a record takes is dropped from a number",
			input: map[string]any{"type": "number", "recordKind": "customer"},
			want:  &askInput{Type: askTypeNumber},
		},
		{
			name:  "a type the thread has no box for is typed as text",
			input: map[string]any{"type": "colour"},
		},
		{
			name:  "plain text says nothing",
			input: map[string]any{"type": "text"},
		},
		{
			name:  "not an object",
			input: "date",
		},
		{
			name:       "nothing is typed when nothing may be",
			input:      map[string]any{"type": "date"},
			allowOther: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			arguments := map[string]any{
				"question": "Which one should I use?",
				"options":  []any{map[string]any{"value": "a", "label": "A"}},
				"input":    tt.input,
			}
			if tt.allowOther != nil {
				arguments["allowOther"] = tt.allowOther
			}

			content, shown := Resolve(askNow, arguments)
			require.True(t, shown)
			assert.Equal(t, tt.want, decodeAsk(t, content).Input)
		})
	}
}

// A write refused for missing values is completed by one form, and the form
// stands alone: no box of its own beside it, and no input type for one.
func TestResolveAsk_AsksForTheValuesAWriteReportedMissing(t *testing.T) {
	t.Parallel()

	content, shown := Resolve(askNow, map[string]any{
		"question":   "What does the shipment still need?",
		"fields":     runtimetest.AskFieldArgs(),
		"allowOther": true,
		"otherHint":  "Anything else",
		"input":      map[string]any{"type": "date"},
	})

	require.True(t, shown)
	request := decodeAsk(t, content)
	assert.Equal(t, []askField{
		{Key: "customerId", Label: "Customer", Type: askTypeRecord, RecordKind: "customer"},
		{
			Key:   "pickupWindow",
			Label: "Pickup window",
			Type:  askTypeDateTime,
			Hint:  "When the truck can load",
		},
		{Key: "weight", Label: "Weight", Type: askTypeNumber, Optional: true},
	}, request.Fields)
	assert.NotNil(t, request.Options)
	assert.Empty(t, request.Options)
	assert.False(t, request.AllowOther, "a form has no box of its own")
	assert.Nil(t, request.Input)
	assert.Empty(t, request.OtherHint)
	assert.Contains(t, request.Note, "a form with 3 fields")
	assert.Contains(t, request.Note, "End your turn now and write nothing more")
	assert.NotContains(t, request.Note, "their own value")
}

func TestResolveAsk_NamesASingleFieldInTheNote(t *testing.T) {
	t.Parallel()

	request := decodeAsk(t, askContent(askNow, map[string]any{
		"question": "Which customer is this load for?",
		"fields": []any{map[string]any{
			"key": "customerId", "label": "Customer", "type": "record", "recordKind": "customer",
		}},
	}))

	assert.Contains(t, request.Note, "a form with 1 field for this question")
}

func TestResolveAsk_RefusesChoicesAndAFormTogether(t *testing.T) {
	t.Parallel()

	content, shown := Resolve(askNow, map[string]any{
		"question": "What does the shipment still need?",
		"options":  []any{map[string]any{"value": "a", "label": "A"}},
		"fields":   runtimetest.AskFieldArgs(),
	})

	assert.False(t, shown)
	assert.Equal(t, mixedAskRefusal, content)
	assert.NotContains(t, content, toolresult.UntrustedOpenTag, "there is no question to render")
}

// An empty list beside a form is no choice offered, and a model that sends
// one is still asking for the form.
func TestResolveAsk_AnEmptyListOfChoicesBesideAFormIsNoChoice(t *testing.T) {
	t.Parallel()

	_, shown := Resolve(askNow, map[string]any{
		"question": "What does the shipment still need?",
		"options":  []any{},
		"fields":   runtimetest.AskFieldArgs(),
	})

	assert.True(t, shown)
}

func TestResolveAsk_DropsTheFieldsItCannotShow(t *testing.T) {
	t.Parallel()

	fields := []any{
		map[string]any{"key": "keep", "label": "Keep", "type": "text"},
		map[string]any{"label": "No key", "type": "text"},
		map[string]any{"key": "nolabel", "type": "text"},
		map[string]any{"key": "notype", "label": "No type"},
		map[string]any{"key": "colour", "label": "Colour", "type": "colour"},
		map[string]any{"key": "kindless", "label": "Kindless", "type": "record"},
		map[string]any{
			"key": "unknown", "label": "Unknown", "type": "record", "recordKind": "spaceship",
		},
		map[string]any{"key": "keep", "label": "Duplicate", "type": "number"},
		"not a field at all",
		map[string]any{"key": "long", "label": strings.Repeat("x", 200), "type": "date"},
	}
	for idx := range maxAskFields {
		fields = append(fields, map[string]any{
			"key": strings.Repeat("k", idx+1), "label": "Extra", "type": "text",
		})
	}

	request := decodeAsk(t, askContent(askNow, map[string]any{
		"question": "What does the shipment still need?",
		"fields":   fields,
	}))

	require.Len(t, request.Fields, maxAskFields)
	keys := make([]string, 0, len(request.Fields))
	for _, field := range request.Fields {
		keys = append(keys, field.Key)
		assert.LessOrEqual(t, len([]rune(field.Label)), maxAskLabelChars+1)
	}
	assert.Equal(t, []string{"keep", "long", "k"}, keys[:3])
	assert.Equal(t, "Keep", request.Fields[0].Label, "the first of a key is the one kept")
	assert.Equal(t, askTypeDate, request.Fields[1].Type)
}

func TestResolveAsk_RefusesAFormWithNothingToShow(t *testing.T) {
	t.Parallel()

	content, shown := Resolve(askNow, map[string]any{
		"question": "What does the shipment still need?",
		"fields": []any{
			map[string]any{"key": "customerId", "label": "Customer", "type": "record"},
			map[string]any{"label": "Pickup window", "type": "datetime"},
		},
	})

	assert.False(t, shown)
	assert.Equal(t, unshowableFieldsRefusal, content)
}

// A model asked for a whole shipment in one free-text question and was shown
// a box to retype a load sheet into. A question answered only by typing asks
// for one value.
func TestResolveAsk_RefusesAFormDisguisedAsOneQuestion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		question string
	}{
		{
			name: "the shipment that was asked for in one box",
			question: "What shipment details should I book? Please include the customer, " +
				"pickup and delivery locations with date/time windows, service type, shipment " +
				"type, and any freight details (weight, pieces, commodity) or price/reference.",
		},
		{
			name:     "a short list",
			question: "Which customer, pickup and delivery?",
		},
		{
			name:     "a list joined by and",
			question: "What are the pickup and delivery windows, and the weight?",
		},
		{
			name:     "a list split by semicolons",
			question: "What is the weight; the piece count; the commodity?",
		},
		{
			name:     "a question too long to be about one value",
			question: "What " + strings.Repeat("very ", maxOpenQuestionRunes/5) + "long question?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content, shown := Resolve(askNow, map[string]any{"question": tt.question})

			assert.False(t, shown)
			assert.Equal(t, multiPartAskRefusal, content)
			assert.Contains(t, content, "with fields for exactly those values")
		})
	}
}

func TestResolveAsk_ShowsAQuestionAboutOneValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		arguments map[string]any
	}{
		{
			name:      "a short question",
			arguments: map[string]any{"question": "Which PO number should I use?"},
		},
		{
			name:      "one aside",
			arguments: map[string]any{"question": "What weight should I use, in pounds?"},
		},
		{
			name:      "a figure with a thousands separator",
			arguments: map[string]any{"question": "Should I use $2,450 or something else?"},
		},
		{
			name: "two things joined by a bare and",
			arguments: map[string]any{
				"question": "Which date should I use for pickup and delivery?",
			},
		},
		{
			name: "a list of choices is a choice, not a form",
			arguments: map[string]any{
				"question": "Which customer, pickup and delivery?",
				"options":  []any{map[string]any{"value": "a", "label": "Acme, LA to Chicago"}},
			},
		},
		{
			name: "a typed value of its own type",
			arguments: map[string]any{
				"question": "When, roughly, and at which dock, should pickup start?",
				"input":    map[string]any{"type": "datetime"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content, shown := Resolve(askNow, tt.arguments)

			assert.True(t, shown, content)
			assert.Contains(t, content, toolresult.UntrustedOpenTag)
		})
	}
}

// A typed answer the model can work out, such as a partner code from the
// partner's name, waits in the person's box. Before, it was only an example
// in the placeholder, and the person retyped it.
func TestResolveAsk_OffersTheModelsSuggestionForATypedValue(t *testing.T) {
	t.Parallel()

	request := decodeAsk(t, askContent(askNow, map[string]any{
		"question":   "What partner code should Meritor have in Trenova?",
		"otherHint":  "A short code, such as MERITOR",
		"suggestion": "  MERITOR\n",
	}))

	assert.Equal(t, "MERITOR", request.Suggestion)
	assert.Equal(t, "A short code, such as MERITOR", request.OtherHint)
	assert.Contains(t, request.Note, "holding your suggestion for them to accept or replace")
	assert.Contains(t, request.Note, "Do not choose for them")
}

func TestResolveAsk_KeepsASuggestionOnlyForATypedTextValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		arguments map[string]any
		want      string
	}{
		{
			name: "spacing inside it is folded onto one line",
			arguments: map[string]any{
				"question":   "What should the lane be called?",
				"suggestion": "Chicago\n  to   Dallas",
			},
			want: "Chicago to Dallas",
		},
		{
			name: "choices are already best first",
			arguments: map[string]any{
				"question":   "Which window should the report cover?",
				"options":    []any{map[string]any{"value": "7", "label": "7 days"}},
				"suggestion": "30",
			},
		},
		{
			name: "a value of another type has its own box",
			arguments: map[string]any{
				"question":   "How many days should the report cover?",
				"input":      map[string]any{"type": "number"},
				"suggestion": "30",
			},
		},
		{
			name: "a form has no box of its own",
			arguments: map[string]any{
				"question": "What else does the shipment need?",
				"fields": []any{
					map[string]any{"key": "weight", "label": "Weight", "type": "number"},
				},
				"suggestion": "42000",
			},
		},
		{
			name: "one too long to take at a glance is dropped, not cut",
			arguments: map[string]any{
				"question":   "What should the note say?",
				"suggestion": strings.Repeat("a", maxAskSuggestionRunes+1),
			},
		},
		{
			name: "one of exactly the bound is kept",
			arguments: map[string]any{
				"question":   "What should the note say?",
				"suggestion": strings.Repeat("é", maxAskSuggestionRunes),
			},
			want: strings.Repeat("é", maxAskSuggestionRunes),
		},
		{
			name: "only spacing is no suggestion",
			arguments: map[string]any{
				"question":   "What partner code should Meritor have?",
				"suggestion": " \n ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content, shown := Resolve(askNow, tt.arguments)
			require.True(t, shown, content)

			request := decodeAsk(t, content)
			assert.Equal(t, tt.want, request.Suggestion)
			if tt.want == "" {
				assert.NotContains(t, askBody(t, content), `"suggestion"`)
				assert.NotContains(t, request.Note, "suggestion")
			}
		})
	}
}

// Options are no longer required: a question answered only by typing sends
// none, and a strict provider must not demand them.
func TestAskUserSpec_RequiresOnlyTheQuestion(t *testing.T) {
	t.Parallel()

	spec := Spec()
	assert.Equal(t, []string{"question"}, spec.Parameters["required"])

	properties, ok := spec.Parameters["properties"].(map[string]any)
	require.True(t, ok)

	fields, ok := properties["fields"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, maxAskFields, fields["maxItems"])
	items, ok := fields["items"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []string{"key", "label", "type"}, items["required"])
	itemProperties, ok := items["properties"].(map[string]any)
	require.True(t, ok)
	fieldType, ok := itemProperties["type"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, askValueTypes, fieldType["enum"])

	input, ok := properties["input"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []string{"type"}, input["required"])
}

// A question nobody saw is not one the conversation asked, so asking it
// again properly is not repeating it.
func TestAskedQuestions_LeavesOutAQuestionThatWasRefused(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "book a load"},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{
			{ID: "c1", Name: Name, Arguments: map[string]any{
				"question": "Which customer, pickup and delivery?",
			}},
		}},
		{
			Role:       conversation.RoleTool,
			ToolCallID: "c1",
			ToolName:   Name,
			ToolFailed: true,
			Content:    multiPartAskRefusal,
		},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{
			{ID: "c2", Name: Name, Arguments: map[string]any{
				"question": "Which customer is it for?",
			}},
		}},
		{Role: conversation.RoleTool, ToolCallID: "c2", ToolName: Name, Content: "shown"},
	}

	asked := AskedQuestions(history)

	assert.Equal(t, map[string]struct{}{"which customer is it for": {}}, asked)
}
