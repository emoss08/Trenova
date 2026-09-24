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
