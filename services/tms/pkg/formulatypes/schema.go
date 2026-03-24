package formulatypes

import (
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type DataSource struct {
	Table    string                `json:"table"`
	Entity   string                `json:"entity"`
	Preloads []string              `json:"preloads,omitempty"`
	Joins    map[string]JoinConfig `json:"joins,omitempty"`
	Filters  []FilterConfig        `json:"filters,omitempty"`
	OrderBy  string                `json:"orderBy,omitempty"`
}

type JoinConfig struct {
	Table                string          `json:"table"`
	On                   string          `json:"on"`
	Type                 dbtype.JoinType `json:"type,omitempty"`                 // Default: LEFT
	AdditionalConditions string          `json:"additionalConditions,omitempty"` // Additional WHERE conditions
}

type FilterConfig struct {
	Field    string          `json:"field"`
	Operator dbtype.Operator `json:"operator"`
	Value    any             `json:"value"`
}

type Defintion struct {
	ID             string                  `json:"$id"`
	Schema         string                  `json:"$schema"`
	Title          string                  `json:"title"`
	Description    string                  `json:"description"`
	Type           string                  `json:"type"`
	Properties     map[string]Property     `json:"properties"`
	Required       []string                `json:"required"`
	Version        string                  `json:"version"`
	FormulaContext FormulaContextExtension `json:"x-formula-context"`
	DataSource     DataSource              `json:"x-data-source"`
	FieldSources   map[string]*FieldSource
	CompiledSchema *jsonschema.Schema
}

type FieldSource struct {
	Field     string   `json:"field,omitempty"`
	Path      string   `json:"path,omitempty"`
	Computed  bool     `json:"computed,omitempty"`
	Function  string   `json:"function,omitempty"`
	Requires  []string `json:"requires,omitempty"`
	Preload   []string `json:"preload,omitempty"`
	Relation  string   `json:"relation,omitempty"`
	Table     string   `json:"table,omitempty"`
	Nullable  bool     `json:"nullable,omitempty"`
	Transform string   `json:"transform,omitempty"`
	Type      string   `json:"type,omitempty"`
}

type Property struct {
	Type        any                 `json:"type"`
	Description string              `json:"description"`
	Enum        []string            `json:"enum,omitempty"`
	Minimum     *float64            `json:"minimum,omitempty"`
	Maximum     *float64            `json:"maximum,omitempty"`
	MinItems    *int                `json:"minItems,omitempty"`
	MaxItems    *int                `json:"maxItems,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
	Items       *Property           `json:"items,omitempty"`
	Source      FieldSource         `json:"x-source"`
}

type FormulaContextExtension struct {
	Category    string   `json:"category"`
	Entities    []string `json:"entities"`
	Permissions []string `json:"permissions"`
	Tags        []string `json:"tags"`
}
