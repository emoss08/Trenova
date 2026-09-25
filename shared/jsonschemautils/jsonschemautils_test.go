package jsonschemautils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObjectClosesTheSchemaAndListsRequiredProperties(t *testing.T) {
	t.Parallel()

	properties := map[string]any{"system": Enum("The system.", "QuickBooksOnline")}

	assert.Equal(t, map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
		"required":             []string{"system"},
	}, Object(properties, "system"))
}

func TestObjectOmitsRequiredWhenNothingIsRequired(t *testing.T) {
	t.Parallel()

	schema := Object(map[string]any{})

	assert.NotContains(t, schema, "required")
	assert.Equal(t, false, schema["additionalProperties"])
}

func TestEnumKeepsEveryValueInOrder(t *testing.T) {
	t.Parallel()

	assert.Equal(t, map[string]any{
		"type":        "string",
		"enum":        []string{"b", "a"},
		"description": "Pick one.",
	}, Enum("Pick one.", "b", "a"))
}

func TestStringBoundsItsLengthOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	assert.Equal(t, map[string]any{"type": "string", "maxLength": 40}, String(40))
	assert.Equal(t, map[string]any{"type": "string"}, String(0))
}

func TestNumberCarriesItsRange(t *testing.T) {
	t.Parallel()

	assert.Equal(t, map[string]any{"type": "number", "minimum": 0.0, "maximum": 1.0}, Number(0, 1))
}

func TestArrayBoundsItsItemsOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	items := String(0)
	assert.Equal(t, map[string]any{"type": "array", "items": items, "maxItems": 5}, Array(items, 5))
	assert.Equal(t, map[string]any{"type": "array", "items": items}, Array(items, 0))
}

func TestDescribedScalarsCarryTheirTypeAndDescription(t *testing.T) {
	t.Parallel()

	assert.Equal(t, map[string]any{"type": "string", "description": "a"}, Text("a"))
	assert.Equal(t, map[string]any{"type": "boolean", "description": "b"}, Boolean("b"))
	assert.Equal(t, map[string]any{"type": "integer", "description": "c"}, Integer("c"))
}
