package argcheck

import (
	"maps"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type aliasCase struct {
	Name   string         `yaml:"name"`
	Schema map[string]any `yaml:"schema"`
	Args   map[string]any `yaml:"args"`
	Want   map[string]any `yaml:"want"`
}

func aliasedArguments(schema, args map[string]any) map[string]any {
	aliased, _ := aliasArguments(schema, args)

	return aliased
}

func TestAliasArguments_Cases(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../testdata/arguments/cases.yaml")
	require.NoError(t, err)
	var cases struct {
		Aliases []aliasCase `yaml:"aliases"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &cases))
	require.NotEmpty(t, cases.Aliases)

	for _, tc := range cases.Aliases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			sent := maps.Clone(tc.Args)
			got := aliasedArguments(tc.Schema, sent)

			assert.Equal(t, tc.Want, got)
			assert.Equal(t, tc.Args, sent, "the model's own call is never changed")
		})
	}
}

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
func TestAliasArguments_ReadsABareIDAsTheOneMissingID(t *testing.T) {
	t.Parallel()

	schema := requiredSchema([]string{"shipmentId"}, "shipmentId")
	sent := map[string]any{"id": "shp_1"}

	got := aliasedArguments(schema, sent)

	assert.Equal(t, map[string]any{"shipmentId": "shp_1"}, got)
	assert.Equal(t, map[string]any{"id": "shp_1"}, sent, "the model's own call is not rewritten")
}

func TestAliasArguments_ReadsTheSameNameSpelledAnotherWay(t *testing.T) {
	t.Parallel()

	schema := requiredSchema([]string{"shipmentId"}, "shipmentId", "includeStops")

	assert.Equal(t, map[string]any{"shipmentId": "shp_1"},
		aliasedArguments(schema, map[string]any{"shipment_id": "shp_1"}))
	assert.Equal(t, map[string]any{"shipmentId": "shp_1"},
		aliasedArguments(schema, map[string]any{"ShipmentID": "shp_1"}))
}

// Anything less than plain is left for the tool to refuse, with the name it
// wanted: a bare id when two ids are missing could mean either.
func TestAliasArguments_LeavesAnAmbiguousIDAlone(t *testing.T) {
	t.Parallel()

	schema := requiredSchema([]string{"moveId", "workerId"}, "moveId", "workerId")
	sent := map[string]any{"id": "mv_1"}

	assert.Equal(t, sent, aliasedArguments(schema, sent))
}

// Arguments already complete are passed through untouched.
func TestAliasArguments_LeavesACompleteCallAlone(t *testing.T) {
	t.Parallel()

	schema := requiredSchema([]string{"shipmentId"}, "shipmentId")
	sent := map[string]any{"shipmentId": "shp_1", "id": "shp_2"}

	assert.Equal(t, sent, aliasedArguments(schema, sent))
}
