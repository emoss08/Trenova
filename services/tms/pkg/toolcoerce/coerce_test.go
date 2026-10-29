package toolcoerce

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/pkg/toolschematest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func coercedSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"limit": map[string]any{toolschema.KeyType: toolschema.TypeInteger},
			"ids": map[string]any{
				toolschema.KeyType:  toolschema.TypeArray,
				toolschema.KeyItems: map[string]any{toolschema.KeyType: toolschema.TypeString},
			},
			"shared": map[string]any{toolschema.KeyType: toolschema.TypeBoolean},
			"status": map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyEnum: []string{"Late", "OnTime"},
			},
			"note": map[string]any{toolschema.KeyType: toolschema.TypeString},
			"day": map[string]any{
				toolschema.KeyType:   toolschema.TypeString,
				toolschema.KeyFormat: toolschema.FormatDate,
			},
			"pickup": map[string]any{
				toolschema.KeyType:   toolschema.TypeString,
				toolschema.KeyFormat: toolschema.FormatLocalDateTime,
			},
		},
		toolschema.KeyRequired: []string{"limit"},
	}
}

// Every reading that is certain is made, each told back by path, and the
// result fits the schema it was read against.
func TestCoerce_ReadsValuesTheWayTheSchemaDeclaresThem(t *testing.T) {
	t.Parallel()

	sent := map[string]any{
		"limit":  "30",
		"ids":    "shp_1",
		"shared": "yes",
		"status": "late",
		"note":   nil,
		"day":    "2026-10-01T00:00:00Z",
		"pickup": "2026-10-01 08:00",
	}

	got, coercions := Apply(coercedSchema(), sent)

	assert.Equal(t, map[string]any{
		"limit":  float64(30),
		"ids":    []any{"shp_1"},
		"shared": true,
		"status": "Late",
		"day":    "2026-10-01",
		"pickup": "2026-10-01T08:00",
	}, got)
	assert.Len(t, coercions, 6)
	require.NoError(t, toolschema.Validate(coercedSchema(), got))
	assert.Equal(t, "30", sent["limit"], "what was sent is never changed")
}

// What is not certain is left for the schema to refuse: a word for a
// number, a value outside the enum, a number for a date.
func TestCoerce_LeavesUncertainValuesAlone(t *testing.T) {
	t.Parallel()

	sent := map[string]any{"limit": "thirty", "status": "Delayed"}

	got, coercions := Apply(coercedSchema(), sent)

	assert.Equal(t, sent, got)
	assert.Empty(t, coercions)

	dated, _ := Apply(coercedSchema(), map[string]any{"limit": 1, "day": float64(1790812800)})
	errs := toolschematest.FieldErrors(t, toolschema.Validate(coercedSchema(), dated))
	assert.Contains(t, errs["day"], "is a Unix time")
}

/*
A person asks what delivered today and a model passes the word on. A few
reads parsed "today" themselves and the rest refused it; the runtime now
reads today, yesterday and tomorrow once, as days in the organization's
timezone, for every day parameter. Without a day to read them by, the word
is left for the format to refuse.
*/
func TestCoerce_ReadsRelativeDaysInTheOrganizationsTimezone(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"asOf":  map[string]any{"type": "string", "format": toolschema.FormatDate},
			"from":  map[string]any{"type": "string", "format": toolschema.FormatDate},
			"until": map[string]any{"type": "string", "format": toolschema.FormatDate},
		},
	}
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	lateEvening := time.Date(2026, 10, 9, 23, 30, 0, 0, chicago)

	got, notes := Apply(schema, map[string]any{
		"asOf": "Today", "from": "yesterday", "until": "tomorrow",
	}, WithToday(lateEvening))

	assert.Equal(t, map[string]any{
		"asOf": "2026-10-09", "from": "2026-10-08", "until": "2026-10-10",
	}, got, "the organization's day, not UTC's, which is already the 10th")
	assert.Len(t, notes, 3)

	unread, _ := Apply(schema, map[string]any{"asOf": "today"})
	assert.Equal(t, "today", unread["asOf"])
}

func TestCoerce_ReadsJSONTextAsTheListOrObjectItHolds(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filters": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"field": map[string]any{"type": "string"},
						"value": map[string]any{"type": "string"},
					},
				},
			},
			"window": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"days": map[string]any{"type": "integer"},
				},
			},
		},
	}

	got, coercions := Apply(schema, map[string]any{
		"filters": ` [{"field": "status", "value": "Available"}]`,
		"window":  `{"days": "3"}`,
	})

	assert.Equal(t, map[string]any{
		"filters": []any{map[string]any{"field": "status", "value": "Available"}},
		"window":  map[string]any{"days": float64(3)},
	}, got)
	assert.Len(t, coercions, 3)
	require.NoError(t, toolschema.Validate(schema, got))

	left, none := Apply(schema, map[string]any{"filters": "[not json"})
	assert.Equal(t, []any{"[not json"}, left["filters"])
	assert.Len(t, none, 1)
}

// A report definition is a whole schema of its own, its root a $ref into its
// own $defs. Sent as JSON text, it is read as the object it holds, as any
// object parameter is.
func TestCoerce_ReadsJSONTextAsTheObjectALocalRefNames(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"definition": map[string]any{
				"$id":  "https://trenova.test/definition.schema.json",
				"$ref": "#/$defs/Definition",
				"$defs": map[string]any{
					"Definition": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"entity": map[string]any{"type": "string"},
						},
						"required": []any{"entity"},
					},
				},
			},
		},
	}

	got, coercions := Apply(schema, map[string]any{"definition": `{"entity": "invoice"}`})

	assert.Equal(t, map[string]any{"definition": map[string]any{"entity": "invoice"}}, got)
	assert.Len(t, coercions, 1)
	require.NoError(t, toolschema.Validate(schema, got))
}

func TestCoerce_ReadsAClampedNumberPastItsBoundsAsTheBound(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"limit": toolschema.Clamped(
				map[string]any{
					toolschema.KeyType:    toolschema.TypeInteger,
					toolschema.KeyMinimum: 1,
					toolschema.KeyMaximum: 50,
				},
			),
			"days": map[string]any{
				toolschema.KeyType:    toolschema.TypeInteger,
				toolschema.KeyMaximum: 30,
			},
		},
	}

	got, coercions := Apply(schema, map[string]any{"limit": float64(500), "days": float64(90)})
	assert.Equal(t, map[string]any{"limit": float64(50), "days": float64(90)}, got,
		"only a clamped number is read as its bound; any other is left for the schema")
	require.Len(t, coercions, 1)
	assert.Contains(t, coercions[0].Note, "read as 50")

	got, _ = Apply(schema, map[string]any{"limit": "0"})
	assert.Equal(t, float64(1), got["limit"])
	require.NoError(t, toolschema.Validate(schema, map[string]any{"limit": got["limit"]}))
	assert.NotContains(
		t,
		toolschema.ForModel(
			schema[toolschema.KeyProperties].(map[string]any)["limit"].(map[string]any),
		),
		toolschema.KeyClamp,
	)
}
