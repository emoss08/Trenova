package toolschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func datedSchema() map[string]any {
	return map[string]any{
		KeyType: TypeObject,
		KeyProperties: map[string]any{
			"day":    map[string]any{KeyType: TypeString, KeyFormat: FormatDate},
			"at":     map[string]any{KeyType: TypeString, KeyFormat: FormatDateTime},
			"pickup": map[string]any{KeyType: TypeString, KeyFormat: FormatLocalDateTime},
		},
	}
}

// The three shapes are asserted, the custom one included, so a value of the
// wrong shape is refused before a tool parses it in its own words.
func TestValidate_AssertsTheDateFormats(t *testing.T) {
	t.Parallel()

	require.NoError(t, Validate(datedSchema(), map[string]any{
		"day":    "2026-10-01",
		"at":     "2026-10-01T08:00:00-05:00",
		"pickup": "2026-10-01T08:00",
	}))

	errs := fieldErrors(t, Validate(datedSchema(), map[string]any{
		"day":    "10/01/2026",
		"at":     "2026-10-01 08:00",
		"pickup": "2026-10-01T08:00:00Z",
	}))

	assert.Equal(t, `"10/01/2026" is not a date; send YYYY-MM-DD, such as 2026-10-01`, errs["day"])
	assert.Contains(t, errs["at"], "send YYYY-MM-DDTHH:MM:SS with its UTC offset")
	assert.Equal(t, `"2026-10-01T08:00:00Z" is not a local-date-time; send YYYY-MM-DDTHH:MM `+
		`with no UTC offset, such as 2026-10-01T08:00`, errs["pickup"])
}

// A string of digits is the Unix time a model reached for, and is named as
// one rather than as a malformed date.
func TestValidate_NamesAUnixTimeAsOne(t *testing.T) {
	t.Parallel()

	errs := fieldErrors(t, Validate(datedSchema(), map[string]any{"day": "1790812800"}))

	assert.Equal(t,
		"1790812800 is a Unix time; send a date as YYYY-MM-DD, such as 2026-10-01", errs["day"])
}

func TestDateFormatNamed_KnowsOnlyTheThree(t *testing.T) {
	t.Parallel()

	for _, name := range []string{FormatDate, FormatDateTime, FormatLocalDateTime} {
		format, ok := DateFormatNamed(name)
		require.True(t, ok, name)
		assert.NotEmpty(t, format.Example)
		assert.NotEmpty(t, format.Sentence)
	}

	_, ok := DateFormatNamed("email")
	assert.False(t, ok)
}

// Anthropic and OpenAI take a date's format and examples as JSON Schema, so
// they see them; a server that only speaks OpenAI's protocol is shown neither
// the examples nor this package's own local-date-time format, and keeps the
// standard date and date-time formats Gemini documents as accepted.
func TestForModel_KeepsDateKeywordsAndForPortableModelDropsTheUnportable(t *testing.T) {
	t.Parallel()

	schema := datedSchema()
	schema[KeyProperties].(map[string]any)["day"].(map[string]any)[KeyExamples] = []string{
		"2026-10-01",
	}

	full := ForModel(schema)[KeyProperties].(map[string]any)
	assert.Equal(t, FormatDate, FormatOf(full["day"].(map[string]any)))
	assert.Contains(t, full["day"], KeyExamples)
	assert.Equal(t, FormatLocalDateTime, FormatOf(full["pickup"].(map[string]any)))

	portable := ForPortableModel(schema)[KeyProperties].(map[string]any)
	assert.NotContains(t, portable["day"], KeyExamples)
	assert.Equal(t, FormatDate, FormatOf(portable["day"].(map[string]any)))
	assert.NotContains(t, portable["pickup"], KeyFormat)
	assert.Equal(t, FormatDateTime, FormatOf(portable["at"].(map[string]any)))
	assert.Contains(t, schema[KeyProperties].(map[string]any)["day"], KeyExamples,
		"the tool's own schema is untouched")
}

// An empty date is the tool's to read: several take it as "clear it".
func TestValidate_LeavesAnEmptyDateToTheTool(t *testing.T) {
	t.Parallel()

	require.NoError(t, Validate(datedSchema(), map[string]any{
		"day": "", "at": "", "pickup": " ",
	}))
}
