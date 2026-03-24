package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validSchemaJSON = `{
	"$id": "test-schema",
	"$schema": "https://json-schema.org/draft/2020-12/schema",
	"title": "Test Schema",
	"type": "object",
	"properties": {
		"name": {
			"type": "string",
			"x-source": {"field": "name"}
		},
		"age": {
			"type": "integer",
			"minimum": 0,
			"x-source": {"field": "age"}
		}
	},
	"required": ["name"]
}`

const nestedSchemaJSON = `{
	"$id": "nested-schema",
	"$schema": "https://json-schema.org/draft/2020-12/schema",
	"title": "Nested Schema",
	"type": "object",
	"properties": {
		"address": {
			"type": "object",
			"properties": {
				"city": {
					"type": "string",
					"x-source": {"field": "city", "table": "addresses"}
				},
				"zip": {
					"type": "string",
					"x-source": {"field": "zip_code"}
				}
			}
		},
		"tags": {
			"type": "array",
			"items": {
				"type": "string",
				"x-source": {"path": "tag_values", "computed": true}
			}
		}
	}
}`

func TestNewRegistry(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	assert.NotNil(t, r)
	assert.NotNil(t, r.schemas)
	assert.NotNil(t, r.compiler)
	assert.Empty(t, r.schemas)
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		err := r.Register("test", []byte(validSchemaJSON))
		require.NoError(t, err)

		def, exists := r.schemas["test"]
		require.True(t, exists)
		assert.Equal(t, "test-schema", def.ID)
		assert.Equal(t, "Test Schema", def.Title)
		assert.NotNil(t, def.CompiledSchema)
		assert.NotNil(t, def.FieldSources)
		assert.Contains(t, def.FieldSources, "name")
		assert.Contains(t, def.FieldSources, "age")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		err := r.Register("bad", []byte(`{invalid`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal")
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		err := r.Register("noid", []byte(`{"type": "object", "properties": {}}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing id")
	})

	t.Run("extracts nested field sources", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		err := r.Register("nested", []byte(nestedSchemaJSON))
		require.NoError(t, err)

		def, exists := r.schemas["nested"]
		require.True(t, exists)
		assert.Contains(t, def.FieldSources, "address.city")
		assert.Contains(t, def.FieldSources, "address.zip")
		assert.Equal(t, "addresses", def.FieldSources["address.city"].Table)
	})

	t.Run("extracts array item sources", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		err := r.Register("nested", []byte(nestedSchemaJSON))
		require.NoError(t, err)

		def := r.schemas["nested"]
		assert.Contains(t, def.FieldSources, "tags[]")
		assert.True(t, def.FieldSources["tags[]"].Computed)
		assert.Equal(t, "tag_values", def.FieldSources["tags[]"].Path)
	})
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		require.NoError(t, r.Register("test", []byte(validSchemaJSON)))

		def, exists := r.Get("test")
		assert.True(t, exists)
		assert.Equal(t, "test-schema", def.ID)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()

		def, exists := r.Get("missing")
		assert.False(t, exists)
		assert.Nil(t, def)
	})
}

func TestRegistry_ValidateData(t *testing.T) {
	t.Parallel()

	t.Run("valid data", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		require.NoError(t, r.Register("test", []byte(validSchemaJSON)))

		err := r.ValidateData("test", map[string]any{
			"name": "John",
			"age":  float64(30),
		})
		assert.NoError(t, err)
	})

	t.Run("schema not found", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()

		err := r.ValidateData("missing", map[string]any{"name": "test"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("invalid data missing required", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		require.NoError(t, r.Register("test", []byte(validSchemaJSON)))

		err := r.ValidateData("test", map[string]any{
			"age": float64(30),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validate")
	})

	t.Run("invalid data wrong type", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		require.NoError(t, r.Register("test", []byte(validSchemaJSON)))

		err := r.ValidateData("test", map[string]any{
			"name": 123,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validate")
	})

	t.Run("invalid data negative age", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		require.NoError(t, r.Register("test", []byte(validSchemaJSON)))

		err := r.ValidateData("test", map[string]any{
			"name": "John",
			"age":  float64(-1),
		})
		require.Error(t, err)
	})
}

func TestRegistry_ListSchemas(t *testing.T) {
	t.Parallel()

	t.Run("empty registry", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		schemas := r.ListSchemas()
		assert.Empty(t, schemas)
	})

	t.Run("multiple schemas", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		require.NoError(t, r.Register("test1", []byte(validSchemaJSON)))
		require.NoError(t, r.Register("test2", []byte(nestedSchemaJSON)))

		schemas := r.ListSchemas()
		assert.Len(t, schemas, 2)
	})
}
