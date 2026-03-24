package schema

import (
	"bytes"
	"fmt"
	"maps"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/services/formula/errors"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type Registry struct {
	mu       sync.RWMutex
	compiler *jsonschema.Compiler
	schemas  map[string]*formulatypes.Defintion
}

func NewRegistry() *Registry {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)

	return &Registry{
		compiler: compiler,
		schemas:  make(map[string]*formulatypes.Defintion),
	}
}

func (r *Registry) Register(id string, schemaJSON []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var definition formulatypes.Defintion

	if err := sonic.Unmarshal(schemaJSON, &definition); err != nil {
		return errors.NewSchemaError(id, "unmarshal", err)
	}

	if definition.ID == "" {
		return errors.NewSchemaError(id, "missing id", nil)
	}

	schema, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return errors.NewSchemaError(definition.ID, "parse", err)
	}

	if err = r.compiler.AddResource(definition.ID, schema); err != nil {
		return errors.NewSchemaError(definition.ID, "add to compiler", err)
	}

	compiled, err := r.compiler.Compile(definition.ID)
	if err != nil {
		return errors.NewSchemaError(definition.ID, "compile", err)
	}

	definition.CompiledSchema = compiled
	definition.FieldSources = r.extractFieldSource(definition.Properties, "")

	r.schemas[id] = &definition
	return nil
}

func (r *Registry) Get(schemaID string) (*formulatypes.Defintion, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, exists := r.schemas[schemaID]
	if !exists {
		return nil, false
	}

	return schema, true
}

func (r *Registry) ValidateData(schemaID string, data any) error {
	schema, exists := r.Get(schemaID)
	if !exists {
		return errors.NewSchemaError(schemaID, "not found", nil)
	}

	if err := schema.CompiledSchema.Validate(data); err != nil {
		return errors.NewSchemaError(schemaID, "validate", err)
	}

	return nil
}

func (r *Registry) ListSchemas() []*formulatypes.Defintion {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]*formulatypes.Defintion, 0, len(r.schemas))
	for _, schema := range r.schemas {
		schemas = append(schemas, schema)
	}

	return schemas
}

func (r *Registry) extractFieldSource(
	properties map[string]formulatypes.Property,
	prefix string,
) map[string]*formulatypes.FieldSource {
	sources := make(map[string]*formulatypes.FieldSource)

	for name := range properties {
		prop := properties[name]
		fieldPath := name

		if prefix != "" {
			fieldPath = fmt.Sprintf("%s.%s", prefix, fieldPath)
		}

		if prop.Source.Field != "" || prop.Source.Computed {
			source := prop.Source
			sources[fieldPath] = &source
		}

		if prop.Properties != nil {
			nestedSources := r.extractFieldSource(prop.Properties, fieldPath)
			maps.Copy(sources, nestedSources)
		}

		if prop.Type == "array" && prop.Items != nil {
			arrayPrefix := fmt.Sprintf("%s[]", fieldPath)
			if prop.Items.Source.Path != "" || prop.Items.Source.Computed {
				source := prop.Items.Source
				sources[arrayPrefix] = &source
			}

			if prop.Items.Properties != nil {
				nestedSources := r.extractFieldSource(prop.Items.Properties, arrayPrefix)
				maps.Copy(sources, nestedSources)
			}
		}
	}

	return sources
}
