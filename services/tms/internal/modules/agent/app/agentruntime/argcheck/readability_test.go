package argcheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func requiredSchema(required []string, fields ...string) map[string]any {
	properties := make(map[string]any, len(fields))
	for _, field := range fields {
		properties[field] = map[string]any{"type": "string"}
	}

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

/*
A record asked for by "id" reaches the tool under the name it takes.

The Dispatch desk called shipment_get with {"id": "shp_..."} and was told the
shipmentId it had in fact supplied was missing.
*/
func TestAliasedArguments_ReadsABareIDAsTheOneMissingID(t *testing.T) {
	t.Parallel()

	schema := requiredSchema([]string{"shipmentId"}, "shipmentId")
	sent := map[string]any{"id": "shp_1"}

	got := AliasedArguments(schema, sent)

	assert.Equal(t, map[string]any{"shipmentId": "shp_1"}, got)
	assert.Equal(t, map[string]any{"id": "shp_1"}, sent, "the model's own call is not rewritten")
}

func TestAliasedArguments_ReadsTheSameNameSpelledAnotherWay(t *testing.T) {
	t.Parallel()

	schema := requiredSchema([]string{"shipmentId"}, "shipmentId", "includeStops")

	assert.Equal(t, map[string]any{"shipmentId": "shp_1"},
		AliasedArguments(schema, map[string]any{"shipment_id": "shp_1"}))
	assert.Equal(t, map[string]any{"shipmentId": "shp_1"},
		AliasedArguments(schema, map[string]any{"ShipmentID": "shp_1"}))
}

// Anything less than plain is left for the tool to refuse, with the name it
// wanted: a bare id when two ids are missing could mean either.
func TestAliasedArguments_LeavesAnAmbiguousIDAlone(t *testing.T) {
	t.Parallel()

	schema := requiredSchema([]string{"moveId", "workerId"}, "moveId", "workerId")
	sent := map[string]any{"id": "mv_1"}

	assert.Equal(t, sent, AliasedArguments(schema, sent))
}

// Arguments already complete are passed through untouched.
func TestAliasedArguments_LeavesACompleteCallAlone(t *testing.T) {
	t.Parallel()

	schema := requiredSchema([]string{"shipmentId"}, "shipmentId")
	sent := map[string]any{"shipmentId": "shp_1", "id": "shp_2"}

	assert.Equal(t, sent, AliasedArguments(schema, sent))
}
