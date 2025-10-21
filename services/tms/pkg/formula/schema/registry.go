package schema

import (
	"bytes"
	"fmt"
	"maps"
	"slices"
	"sync"

	goErrors "errors"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/formula/errors"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var ErrMissingID = goErrors.New("missing $id field in schema")

type Registry struct {
	mu       sync.RWMutex
	compiler *jsonschema.Compiler
	schemas  map[string]*Definition
}

type Definition struct {
	ID             string                    `json:"$id"`
	Schema         string                    `json:"$schema"`
	Title          string                    `json:"title"`
	Description    string                    `json:"description"`
	Type           string                    `json:"type"`
	Properties     map[string]PropertySchema `json:"properties"`
	Required       []string                  `json:"required,omitempty"`
	Version        string                    `json:"version"`
	FormulaContext FormulaContextExtension   `json:"x-formula-context"` //nolint:tagliatelle // this is fine
	DataSource     DataSource                `json:"x-data-source"`     //nolint:tagliatelle // this is fine
	FieldSources   map[string]*FieldSource
	compiled       *jsonschema.Schema
}

type PropertySchema struct {
	Type        any                       `json:"type"`
	Description string                    `json:"description"`
	Enum        []string                  `json:"enum,omitempty"`
	Minimum     *float64                  `json:"minimum,omitempty"`
	Maximum     *float64                  `json:"maximum,omitempty"`
	MinItems    *int                      `json:"minItems,omitempty"`
	MaxItems    *int                      `json:"maxItems,omitempty"`
	Properties  map[string]PropertySchema `json:"properties,omitempty"`
	Items       *PropertySchema           `json:"items,omitempty"`
	Source      FieldSource               `json:"x-source"`
}

func NewRegistry() *Registry {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)

	return &Registry{
		compiler: compiler,
		schemas:  make(map[string]*Definition),
	}
}

func (r *Registry) RegisterSchema(id string, schemaJSON []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var schemaDef Definition
	if err := sonic.Unmarshal(schemaJSON, &schemaDef); err != nil {
		return errors.NewSchemaError(id, "unmarshal", err)
	}

	if schemaDef.ID == "" {
		return errors.NewSchemaError(id, "validation", ErrMissingID)
	}

	schema, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return errors.NewSchemaError(schemaDef.ID, "parse", err)
	}

	if err = r.compiler.AddResource(schemaDef.ID, schema); err != nil {
		return errors.NewSchemaError(schemaDef.ID, "add to compiler", err)
	}

	compiled, err := r.compiler.Compile(schemaDef.ID)
	if err != nil {
		return errors.NewSchemaError(schemaDef.ID, "compile", err)
	}
	schemaDef.compiled = compiled

	schemaDef.FieldSources = r.extractFieldSources(schemaDef.Properties, "")

	r.schemas[id] = &schemaDef
	return nil
}

func (r *Registry) GetSchema(id string) (*Definition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, exists := r.schemas[id]
	if !exists {
		return nil, fmt.Errorf("schema not found: %s", id)
	}
	return schema, nil
}

func (r *Registry) ValidateData(schemaID string, data any) error {
	schema, err := r.GetSchema(schemaID)
	if err != nil {
		return err
	}

	if err = schema.compiled.Validate(data); err != nil {
		return errors.NewSchemaError(schemaID, "validate", err)
	}
	return nil
}

func (r *Registry) ValidateJSON(schemaID string, jsonData []byte) error {
	var data any
	if err := sonic.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return r.ValidateData(schemaID, data)
}

func (r *Registry) ListSchemas() []*Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]*Definition, 0, len(r.schemas))
	for _, schema := range r.schemas {
		schemas = append(schemas, schema)
	}
	return schemas
}

func (r *Registry) GetSchemasForEntity(entity string) []*Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var schemas []*Definition
	for _, schema := range r.schemas {
		if slices.Contains(schema.FormulaContext.Entities, entity) {
			schemas = append(schemas, schema)
		}
	}
	return schemas
}

func (r *Registry) extractFieldSources(
	properties map[string]PropertySchema,
	prefix string,
) map[string]*FieldSource {
	sources := make(map[string]*FieldSource)

	for name := range properties {
		prop := properties[name]
		fieldPath := name
		if prefix != "" {
			fieldPath = prefix + "." + name
		}

		if prop.Source.Path != "" || prop.Source.Computed {
			source := prop.Source
			sources[fieldPath] = &source
		}

		if prop.Properties != nil {
			nestedSources := r.extractFieldSources(prop.Properties, fieldPath)
			maps.Copy(sources, nestedSources)
		}

		if prop.Type == "array" && prop.Items != nil {
			arrayPrefix := fieldPath + "[]"
			if prop.Items.Source.Path != "" || prop.Items.Source.Computed {
				source := prop.Items.Source
				sources[arrayPrefix] = &source
			}
			if prop.Items.Properties != nil {
				nestedSources := r.extractFieldSources(prop.Items.Properties, arrayPrefix)
				maps.Copy(sources, nestedSources)
			}
		}
	}

	return sources
}
