package reportcatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/buncolgen"
)

type FieldType string

const (
	FieldString  FieldType = "string"
	FieldInt     FieldType = "int"
	FieldDecimal FieldType = "decimal"
	FieldBool    FieldType = "bool"
	FieldEnum    FieldType = "enum"
	FieldEpoch   FieldType = "epoch"
	FieldRef     FieldType = "ref"
	FieldJSON    FieldType = "json"
)

func (t FieldType) IsValid() bool {
	switch t {
	case FieldString,
		FieldInt,
		FieldDecimal,
		FieldBool,
		FieldEnum,
		FieldEpoch,
		FieldRef,
		FieldJSON:
		return true
	default:
		return false
	}
}

type FormatHint string

const (
	FormatNone     FormatHint = ""
	FormatMoney    FormatHint = "money"
	FormatWeight   FormatHint = "weight"
	FormatPercent  FormatHint = "percent"
	FormatDuration FormatHint = "duration"
	FormatDistance FormatHint = "distance"
	FormatCount    FormatHint = "count"
)

func (f FormatHint) IsValid() bool {
	switch f {
	case FormatNone, FormatMoney, FormatWeight, FormatPercent,
		FormatDuration, FormatDistance, FormatCount:
		return true
	default:
		return false
	}
}

type Aggregation string

const (
	AggCount         Aggregation = "count"
	AggCountDistinct Aggregation = "count_distinct"
	AggSum           Aggregation = "sum"
	AggAvg           Aggregation = "avg"
	AggMin           Aggregation = "min"
	AggMax           Aggregation = "max"
)

func (a Aggregation) IsValid() bool {
	switch a {
	case AggCount, AggCountDistinct, AggSum, AggAvg, AggMin, AggMax:
		return true
	default:
		return false
	}
}

// FieldClass is what reading a field of a record reached through another
// record takes: a field in its kind's reference view is shown to anyone who
// may read the record that links to it, and every other field needs the
// kind's own read (docs/engineering/reference-views.md).
type FieldClass string

const (
	ClassReference FieldClass = "reference"
	ClassKindRead  FieldClass = "kind_read"
)

func (c FieldClass) IsValid() bool {
	return c == ClassReference || c == ClassKindRead
}

type EnumValue struct {
	Value string
	Label string
}

type Field struct {
	Key          string
	Column       buncolgen.Column
	Label        string
	Description  string
	Type         FieldType
	Format       FormatHint
	Nullable     bool
	EnumValues   []EnumValue
	Aggregations []Aggregation
	Filterable   bool
	Groupable    bool
	Class        FieldClass
	// Protected is the field group a reader also needs read on, as the
	// GraphQL schema's @protected marks it; empty for an unprotected field.
	Protected permission.FieldGroup
}

func (f *Field) SupportsAggregation(agg Aggregation) bool {
	for _, a := range f.Aggregations {
		if a == agg {
			return true
		}
	}
	return false
}

type TenantColumns struct {
	OrganizationID string
	BusinessUnitID string
}

func (t TenantColumns) IsTenanted() bool {
	return t.OrganizationID != "" && t.BusinessUnitID != ""
}

type Entity struct {
	Key             string
	Resource        permission.Resource
	Table           buncolgen.TableInfo
	Label           string
	PluralLabel     string
	Description     string
	Category        string
	Tenant          TenantColumns
	OwnershipColumn string
	// WithinRecord marks an entity whose rows belong to its resource's
	// record without being that record, such as a worker's profile or an
	// invoice's lines: none of its fields is in the resource's reference
	// view, so each needs the resource's read.
	WithinRecord bool
	Fields       []Field
	Edges        []Edge

	fieldsByKey map[string]int
	edgesByName map[string]int
}

func (e *Entity) Field(key string) (*Field, bool) {
	idx, ok := e.fieldsByKey[key]
	if !ok {
		return nil, false
	}
	return &e.Fields[idx], true
}

func (e *Entity) Edge(name string) (*Edge, bool) {
	idx, ok := e.edgesByName[name]
	if !ok {
		return nil, false
	}
	return &e.Edges[idx], true
}

func (e *Entity) GrainKey() []string {
	return e.Table.PrimaryKey
}

type Catalog struct {
	Version  string
	Entities []Entity

	byKey map[string]int
}

func (c *Catalog) Entity(key string) (*Entity, bool) {
	idx, ok := c.byKey[key]
	if !ok {
		return nil, false
	}
	return &c.Entities[idx], true
}

// indexed builds the lookup maps over a catalog literal so the generated
// Default can be a plain var initialiser rather than an init function.
func indexed(c Catalog) Catalog {
	c.index()
	return c
}

func (c *Catalog) index() {
	c.byKey = make(map[string]int, len(c.Entities))
	for i := range c.Entities {
		entity := &c.Entities[i]
		c.byKey[entity.Key] = i

		entity.fieldsByKey = make(map[string]int, len(entity.Fields))
		for j := range entity.Fields {
			entity.fieldsByKey[entity.Fields[j].Key] = j
		}

		entity.edgesByName = make(map[string]int, len(entity.Edges))
		for j := range entity.Edges {
			entity.edgesByName[entity.Edges[j].Name] = j
		}
	}
}
