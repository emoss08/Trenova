package toolschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func coercedSchema() map[string]any {
	return map[string]any{
		KeyType: TypeObject,
		KeyProperties: map[string]any{
			"limit":  map[string]any{KeyType: TypeInteger},
			"ids":    map[string]any{KeyType: TypeArray, KeyItems: map[string]any{KeyType: TypeString}},
			"shared": map[string]any{KeyType: TypeBoolean},
			"status": map[string]any{KeyType: TypeString, KeyEnum: []string{"Late", "OnTime"}},
			"note":   map[string]any{KeyType: TypeString},
			"day":    map[string]any{KeyType: TypeString, KeyFormat: FormatDate},
			"pickup": map[string]any{KeyType: TypeString, KeyFormat: FormatLocalDateTime},
		},
		KeyRequired: []string{"limit"},
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

	got, coercions := Coerce(coercedSchema(), sent)

	assert.Equal(t, map[string]any{
		"limit":  float64(30),
		"ids":    []any{"shp_1"},
		"shared": true,
		"status": "Late",
		"day":    "2026-10-01",
		"pickup": "2026-10-01T08:00",
	}, got)
	assert.Len(t, coercions, 6)
	require.NoError(t, Validate(coercedSchema(), got))
	assert.Equal(t, "30", sent["limit"], "what was sent is never changed")
}

// What is not certain is left for the schema to refuse: a word for a
// number, a value outside the enum, a number for a date.
func TestCoerce_LeavesUncertainValuesAlone(t *testing.T) {
	t.Parallel()

	sent := map[string]any{"limit": "thirty", "status": "Delayed"}

	got, coercions := Coerce(coercedSchema(), sent)

	assert.Equal(t, sent, got)
	assert.Empty(t, coercions)

	dated, _ := Coerce(coercedSchema(), map[string]any{"limit": 1, "day": float64(1790812800)})
	errs := fieldErrors(t, Validate(coercedSchema(), dated))
	assert.Contains(t, errs["day"], "is a Unix time")
}

func TestNameShape(t *testing.T) {
	t.Parallel()

	assert.Equal(t, NameShape("shipmentId"), NameShape("shipment_id"))
	assert.Equal(t, "ab.c", JoinPath("ab", "c"))
	assert.Equal(t, "c", JoinPath("", "c"))
}
