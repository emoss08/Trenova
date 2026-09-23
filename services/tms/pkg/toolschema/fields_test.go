package toolschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func notifyDriverSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workerId": map[string]any{"type": "string", "description": "The driver's id."},
			"title":    map[string]any{"type": "string", "maxLength": 80},
			"message":  map[string]any{"type": "string", "maxLength": 500},
			"priority": map[string]any{
				"type": "string", "enum": []string{"low", "medium", "high", "critical"},
			},
			"withinDays": map[string]any{"type": "integer", "minimum": 1, "maximum": 365},
			"urgent":     map[string]any{"type": "boolean"},
			"codes": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
			"extra": map[string]any{"type": "object"},
		},
		"required":             []string{"workerId", "title", "message"},
		"additionalProperties": false,
	}
}

// Every field a tool declares becomes something a person can edit, typed
// by the schema so the control fits the value: prose gets a text area, a
// bounded integer gets its bounds, an enum gets its choices.
func TestFields_DerivesEditableFieldsFromTheSchema(t *testing.T) {
	t.Parallel()

	fields := Fields(notifyDriverSchema())
	byName := make(map[string]Field, len(fields))
	for _, field := range fields {
		byName[field.Name] = field
	}

	require.Len(t, fields, 8)
	assert.Equal(
		t,
		[]string{"message", "title", "workerId"},
		[]string{fields[0].Name, fields[1].Name, fields[2].Name},
		"required fields come first, by name",
	)

	assert.Equal(t, KindText, byName["workerId"].Kind)
	assert.Equal(t, "Worker ID", byName["workerId"].Label)
	assert.Equal(t, "The driver's id.", byName["workerId"].Description)
	assert.True(t, byName["workerId"].Required)

	assert.Equal(t, KindMultiline, byName["message"].Kind, "a message is prose")
	assert.Equal(t, 500, byName["message"].MaxLength)
	assert.Equal(t, KindText, byName["title"].Kind)

	assert.Equal(t, KindChoice, byName["priority"].Kind)
	assert.Equal(t, []string{"low", "medium", "high", "critical"}, byName["priority"].Options)
	assert.False(t, byName["priority"].Required)

	assert.Equal(t, KindInteger, byName["withinDays"].Kind)
	require.NotNil(t, byName["withinDays"].Minimum)
	assert.InDelta(t, 1, *byName["withinDays"].Minimum, 0)
	assert.InDelta(t, 365, *byName["withinDays"].Maximum, 0)

	assert.Equal(t, KindBoolean, byName["urgent"].Kind)
	assert.Equal(t, KindList, byName["codes"].Kind)
	assert.Equal(t, KindJSON, byName["extra"].Kind)
	assert.Empty(t, byName["extra"].Options, "options are a list, never null on the wire")
}

func TestFields_ReadsATypeListAndAnEnumList(t *testing.T) {
	t.Parallel()

	fields := Fields(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"note": map[string]any{"type": []any{"string", "null"}},
			"kinds": map[string]any{
				"type":  "array",
				"items": map[string]any{"enum": []any{"A", "B"}},
			},
		},
	})

	require.Len(t, fields, 2)
	assert.Equal(t, KindList, fields[0].Kind)
	assert.Equal(t, []string{"A", "B"}, fields[0].Options)
	assert.Equal(t, KindMultiline, fields[1].Kind)
}

func TestFields_IsEmptyForASchemaWithoutProperties(t *testing.T) {
	t.Parallel()

	assert.Empty(t, Fields(map[string]any{"type": "object"}))
	assert.Empty(t, Fields(nil))
}
