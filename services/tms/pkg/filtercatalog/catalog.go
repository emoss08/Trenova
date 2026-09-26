// Package filtercatalog is the vocabulary a resource can be narrowed by: which
// fields exist, what kind each one is, which operators that kind accepts, and
// how a request in those terms compiles into a query.
//
// It is one implementation, not two. The list tools compile a model's filters
// through it, and so does the natural-language composer behind the Ask input on
// a data table — otherwise "which shipments are in transit" would mean one
// thing to an agent and another thing to the grid, and the second would be the
// one nobody tested.
//
// Everything here refuses rather than guesses. The query builder drops a field
// it cannot map and returns the page unfiltered, which reads as a filtered
// answer to whoever asked; an enum value outside the set matches nothing, which
// reads as "there are none". Both are wrong answers that look like right ones,
// so both are errors here, worded so the caller can fix them.
package filtercatalog

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

// Kind decides which operators a field accepts and how its value is read.
//
// It is declared per field rather than inferred from the Go type, because a Go
// string is a free-text name in one column and a closed enum in the next, and
// the difference is the whole of what keeps a caller from inventing a status
// that does not exist.
type Kind string

const (
	KindText    Kind = "text"
	KindEnum    Kind = "enum"
	KindDate    Kind = "date"
	KindNumber  Kind = "number"
	KindBoolean Kind = "boolean"
)

// MaxRelativeDays bounds a relative window. Ten years is past any question
// somebody means, and an unbounded one scans the table.
const MaxRelativeDays = 3650

// Field is one column a resource may be narrowed by.
type Field struct {
	Name string
	Kind Kind
	// Note explains the column in the words the business uses, when the name
	// alone does not.
	Note string
	// Values closes an enum. A value outside the set is refused rather than
	// passed through, because the query would match nothing and the caller
	// would report that as "there are none".
	Values         []string
	ValuesUnlisted bool
	Sortable       bool
}

// Resource is one entity and the fields it answers to.
type Resource struct {
	// Tool is the list tool that serves this resource, for a caller that wants
	// to name it.
	Tool     string
	Resource permission.Resource
	// Entity is the plural a person says: "shipments", "workers".
	Entity  string
	Summary string
	Fields  []Field
	// Config is the entity's computed field configuration. A curated field that
	// no longer maps to a column is dropped silently by the query builder, so
	// it is checked against this and refused loudly instead.
	Config *domaintypes.FieldConfiguration

	byName map[string]Field
}

// Prepare indexes a resource's fields. It is called once, when the resource is
// added to a catalog, so a lookup on a hot path is a map read.
func (r Resource) Prepare() Resource {
	r.byName = make(map[string]Field, len(r.Fields))
	for _, field := range r.Fields {
		r.byName[field.Name] = field
	}

	return r
}

// Field returns one field by name.
func (r Resource) Field(name string) (Field, bool) {
	field, ok := r.byName[name]

	return field, ok
}

// FieldNames is every field, in catalogue order, for a refusal that has to say
// what would have worked.
func (r Resource) FieldNames() []string {
	names := make([]string, 0, len(r.Fields))
	for _, field := range r.Fields {
		names = append(names, field.Name)
	}

	return names
}

// SortableNames is every field a caller may order by.
func (r Resource) SortableNames() []string {
	names := make([]string, 0, len(r.Fields))
	for _, field := range r.Fields {
		if field.Sortable {
			names = append(names, field.Name)
		}
	}

	return names
}

// Operators is what a field accepts, for a schema or a refusal.
func Operators(kind Kind) []dbtype.Operator {
	return operatorsByKind[kind]
}

// Catalog is every resource that answers to a filter, indexed for lookup.
type Catalog struct {
	byResource map[permission.Resource]Resource
	byEntity   map[string]Resource
	ordered    []Resource
}

func New(resources ...Resource) *Catalog {
	catalog := &Catalog{
		byResource: make(map[permission.Resource]Resource, len(resources)),
		byEntity:   make(map[string]Resource, len(resources)),
		ordered:    make([]Resource, 0, len(resources)),
	}
	for _, resource := range resources {
		prepared := resource.Prepare()
		catalog.byResource[prepared.Resource] = prepared
		catalog.byEntity[strings.ToLower(prepared.Entity)] = prepared
		catalog.ordered = append(catalog.ordered, prepared)
	}

	return catalog
}

// For looks a resource up by its permission resource, which is what a caller
// holding a page or a route has.
func (c *Catalog) For(resource permission.Resource) (Resource, bool) {
	entry, ok := c.byResource[resource]

	return entry, ok
}

// ByEntity looks a resource up by the plural a person says.
func (c *Catalog) ByEntity(entity string) (Resource, bool) {
	entry, ok := c.byEntity[strings.ToLower(strings.TrimSpace(entity))]

	return entry, ok
}

// Resources is every catalogued resource, in catalogue order.
func (c *Catalog) Resources() []Resource {
	return c.ordered
}

// Condition is one narrowing, stated in the caller's terms rather than the
// database's.
//
// Values arrive as strings because that is what both callers produce: a model
// filling a JSON schema, and a person typing into a box. Reading them is this
// package's job, so neither caller has to know that a date is stored as an
// epoch second or that an enum is case-sensitive.
type Condition struct {
	Field    string
	Operator dbtype.Operator
	// Value is the single operand, for an operator that takes one.
	Value string
	// Values is the operand list, for in and notin.
	Values []string
	// Days is the window width, for lastndays and nextndays.
	Days int
}

// Compile turns conditions into the filters a repository runs, recording what
// was applied into criteria along the way.
//
// criteria may be nil when the caller does not need the terms read back.
func (r Resource) Compile(
	conditions []Condition,
	criteria *Criteria,
) ([]domaintypes.FieldFilter, error) {
	if criteria == nil {
		criteria = NewCriteria(r.Entity)
	}

	built := make([]domaintypes.FieldFilter, 0, len(conditions)*2)
	for _, condition := range conditions {
		next, err := r.compileOne(condition, criteria)
		if err != nil {
			return nil, err
		}
		built = append(built, next...)
	}

	return built, nil
}

func (r Resource) compileOne(
	condition Condition,
	criteria *Criteria,
) ([]domaintypes.FieldFilter, error) {
	field, err := r.resolve(condition.Field)
	if err != nil {
		return nil, err
	}

	operator := dbtype.Operator(strings.ToLower(string(condition.Operator)))
	if !operatorAllowed(field.Kind, operator) {
		return nil, fmt.Errorf(
			"%q is not an operator for the %s field %q; use one of: %s",
			operator, field.Kind, field.Name, joinOperators(operatorsByKind[field.Kind]),
		)
	}

	switch operatorArities[operator] {
	case arityNone:
		criteria.Field(field.Name, OperatorPhrase(operator))

		return []domaintypes.FieldFilter{{Field: field.Name, Operator: operator}}, nil
	case arityDays:
		return window(field, operator, condition.Days, criteria)
	case arityList:
		return listFilter(field, operator, condition.Values, criteria)
	default:
		return valueFilter(field, operator, condition.Value, criteria)
	}
}

func (r Resource) resolve(name string) (Field, error) {
	field, ok := r.byName[name]
	if !ok {
		return Field{}, fmt.Errorf(
			"cannot filter %s on %q; the filterable fields are: %s",
			r.Entity, name, strings.Join(r.FieldNames(), ", "),
		)
	}

	// The query builder skips a field it cannot map and returns the page
	// unfiltered, which the caller would present as a filtered answer. Catching
	// the drift here turns a wrong answer into a plain error.
	if r.Config != nil && !r.Config.FilterableFields[field.Name] {
		return Field{}, fmt.Errorf(
			"%q can no longer be queried on %s; the filterable fields are: %s",
			field.Name, r.Entity, strings.Join(r.FieldNames(), ", "),
		)
	}

	return field, nil
}

// Sort resolves an ordering, refusing a field the resource cannot order by.
// An empty name is no ordering, not an error.
func (r Resource) Sort(name, direction string) ([]domaintypes.SortField, error) {
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}

	field, ok := r.byName[name]
	if !ok || !field.Sortable {
		return nil, fmt.Errorf(
			"cannot sort %s by %q; the sortable fields are: %s",
			r.Entity, name, strings.Join(r.SortableNames(), ", "),
		)
	}

	resolved := dbtype.SortDirectionDesc
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "asc":
		resolved = dbtype.SortDirectionAsc
	case "", "desc":
	default:
		return nil, fmt.Errorf(`"sortDirection" must be "asc" or "desc"`)
	}

	return []domaintypes.SortField{{Field: field.Name, Direction: resolved}}, nil
}
