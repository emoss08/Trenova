package agentquerytoolservice

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/stringutils"
)

// graphQLRead is a tool bound to the persisted GraphQL operation it reads
// through, as agentread/gen wrote it from the schema and the client's
// operations. Nothing here is kept by hand.
type graphQLRead struct {
	Verb      string
	Operation string
	Reason    string
	RootField string
	Node      string
	Resource  string
	Entity    func() domaintypes.PostgresSearchable
	Variables []graphQLVariable
	Fields    []graphQLField
}

type graphQLVariable struct {
	Name    string
	Type    string
	NonNull bool
	List    bool
}

type graphQLField struct {
	Name        string
	Type        string
	Kind        string
	List        bool
	NonNull     bool
	Selected    bool
	Enum        []string
	Description string
}

const (
	graphQLKindEnum   = "enum"
	graphQLKindScalar = "scalar"

	graphQLTypeTimestamp = "Timestamp"
	graphQLTypeBoolean   = "Boolean"
	graphQLTypeID        = "ID"
	graphQLTypeInt       = "Int"
	graphQLTypeFloat     = "Float"
	graphQLTypeDecimal   = "Decimal"
	graphQLTypeString    = "String"

	dataTableInput = "DataTableConnectionInput"

	suffixFrom = "From"
	suffixTo   = "To"
	suffixMin  = "Min"
	suffixMax  = "Max"
)

// unfilteredFields are selected by every operation but are never what a
// person narrows a list by: the tenant, the row's version, and its id, which
// a get reads.
var unfilteredFields = map[string]struct{}{
	recordIDField:        {},
	tenantOrganizationID: {},
	tenantBusinessUnitID: {},
	rowVersion:           {},
}

const (
	tenantOrganizationID = "organizationId"
	tenantBusinessUnitID = "businessUnitId"
	rowVersion           = "version"
	recordIDField        = "id"
	// graphQLEnumSource names an enum parameter's values by the schema's own
	// enum type, which is where they are declared.
	graphQLEnumSource = "graphql."
	filterOperator    = "operator"
	filterValue       = "value"
)

// paramKind is how a typed parameter compiles into the operation's filters.
type paramKind int

const (
	paramIn paramKind = iota
	paramEquals
	paramContains
	paramDayFrom
	paramDayTo
	paramAtLeast
	paramAtMost
)

// graphQLParam is one typed parameter of a tool that reads through GraphQL,
// derived from a field the operation selects and the server lets a table
// filter on.
type graphQLParam struct {
	Name   string
	Field  string
	Kind   paramKind
	Schema map[string]any
}

// typedParams derives a list tool's parameters from its record type: every
// field the entity's own filter configuration (the one the resolver compiles
// fieldFilters with) marks filterable, typed by its schema type. The
// operation's selection decides only what a result carries; a field the
// table does not show can still narrow it, as the resolver accepts.
func (r *graphQLRead) typedParams() []graphQLParam {
	config := querybuilder.GetFieldConfiguration(r.Entity())
	params := make([]graphQLParam, 0, len(r.Fields))
	for idx := range r.Fields {
		field := &r.Fields[idx]
		if _, skip := unfilteredFields[field.Name]; skip || field.List {
			continue
		}
		if !config.FilterableFields[field.Name] {
			continue
		}
		params = append(params, paramsFor(field)...)
	}
	slices.SortFunc(params, func(a, b graphQLParam) int { return strings.Compare(a.Name, b.Name) })

	return params
}

// sortable is every field of the record the server can order the list by.
func (r *graphQLRead) sortable() []string {
	config := querybuilder.GetFieldConfiguration(r.Entity())
	out := make([]string, 0, len(r.Fields))
	for _, field := range r.Fields {
		if _, skip := unfilteredFields[field.Name]; skip || field.List {
			continue
		}
		if config.SortableFields[field.Name] {
			out = append(out, field.Name)
		}
	}
	slices.Sort(out)

	return out
}

func paramsFor(field *graphQLField) []graphQLParam {
	words := stringutils.HumanizeSnakeCase(stringutils.ConvertCamelToSnake(field.Name))
	describe := func(fallback string) string {
		if field.Description != "" {
			return field.Description + " " + fallback
		}
		return fallback
	}

	switch {
	case field.Kind == graphQLKindEnum:
		return []graphQLParam{{
			Name:  field.Name,
			Field: field.Name,
			Kind:  paramIn,
			Schema: map[string]any{
				toolschema.KeyType: toolschema.TypeArray,
				toolschema.KeyDescription: describe(
					"Only records whose " + words + " is one of these.",
				),
				toolschema.KeyItems: agenttoolschema.Enum(
					"", agenttoolschema.Derived(graphQLEnumSource+field.Type, field.Enum),
				),
				toolschema.KeyMinItems: 1,
			},
		}}
	case field.Kind != graphQLKindScalar:
		return nil
	}

	return scalarParams(field, words, describe)
}

// scalarParams are the parameters a scalar field becomes, by its schema type.
func scalarParams(
	field *graphQLField,
	words string,
	describe func(string) string,
) []graphQLParam {
	switch field.Type {
	case graphQLTypeTimestamp:
		return []graphQLParam{
			{
				Name: field.Name + suffixFrom, Field: field.Name, Kind: paramDayFrom,
				Schema: agenttoolschema.Date(describe("Only records whose " + words +
					" is on or after this day, in the organization's timezone.")),
			},
			{
				Name: field.Name + suffixTo, Field: field.Name, Kind: paramDayTo,
				Schema: agenttoolschema.Date(describe("Only records whose " + words +
					" is on or before this day, in the organization's timezone.")),
			},
		}
	case graphQLTypeBoolean:
		return []graphQLParam{{
			Name: field.Name, Field: field.Name, Kind: paramEquals,
			Schema: map[string]any{
				toolschema.KeyType: toolschema.TypeBoolean,
				toolschema.KeyDescription: describe(
					"Only records that are, or are not, " + words + ".",
				),
			},
		}}
	case graphQLTypeID:
		return []graphQLParam{{
			Name: field.Name, Field: field.Name, Kind: paramEquals,
			Schema: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: describe("Only records with this " + words + "."),
			},
		}}
	case graphQLTypeInt, graphQLTypeFloat, graphQLTypeDecimal:
		number := map[string]any{toolschema.KeyType: toolschema.TypeInteger}
		unit := ""
		if field.Type != graphQLTypeInt {
			number = map[string]any{toolschema.KeyType: toolschema.TypeString}
			unit = " Write it as a decimal, such as 1250.00."
		}
		return []graphQLParam{
			{
				Name:  field.Name + suffixMin,
				Field: field.Name,
				Kind:  paramAtLeast,
				Schema: withDescription(
					number,
					describe("Only records whose "+words+" is at least this."+unit),
				),
			},
			{
				Name:  field.Name + suffixMax,
				Field: field.Name,
				Kind:  paramAtMost,
				Schema: withDescription(
					number,
					describe("Only records whose "+words+" is at most this."+unit),
				),
			},
		}
	case graphQLTypeString:
		return []graphQLParam{{
			Name: field.Name, Field: field.Name, Kind: paramContains,
			Schema: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: describe(
					"Only records whose " + words + " contains this text.",
				),
			},
		}}
	}

	return nil
}

func withDescription(schema map[string]any, description string) map[string]any {
	out := make(map[string]any, len(schema)+1)
	for k, v := range schema {
		out[k] = v
	}
	out[toolschema.KeyDescription] = description

	return out
}

// compileFilters turns the typed parameters a call sent into the field
// filters the operation's input takes, reading days in the organization's
// timezone.
func compileFilters(
	params []graphQLParam,
	args map[string]any,
	clk clock,
) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(params))
	for _, param := range params {
		raw, present := args[param.Name]
		if !present || raw == nil {
			continue
		}
		filter, sent, err := compileFilter(param, args, clk)
		if err != nil {
			return nil, err
		}
		if sent {
			out = append(out, filter)
		}
	}

	return out, nil
}

// compileFilter is one parameter's filter, and whether the call asked for
// one: an empty list or blank text narrows nothing.
func compileFilter(
	param graphQLParam,
	args map[string]any,
	clk clock,
) (filter map[string]any, sent bool, err error) {
	compare := func(operator dbtype.Operator, value any) map[string]any {
		return map[string]any{
			paramField:     param.Field,
			filterOperator: string(operator),
			filterValue:    value,
		}
	}

	switch param.Kind {
	case paramIn:
		values := optionalStrings(args, param.Name)
		return compare(dbtype.OpIn, values), len(values) > 0, nil
	case paramEquals:
		return compare(dbtype.OpEqual, args[param.Name]), true, nil
	case paramContains:
		text := strings.TrimSpace(optionalString(args, param.Name))
		return compare(dbtype.OpContains, text), text != "", nil
	case paramDayFrom:
		day, dayErr := readDay(args, param.Name, clk)
		if dayErr != nil {
			return nil, false, dayErr
		}
		return compare(dbtype.OpGreaterThanOrEqual, day), true, nil
	case paramDayTo:
		day, dayErr := readDay(args, param.Name, clk)
		if dayErr != nil {
			return nil, false, dayErr
		}
		return compare(dbtype.OpLessThanOrEqual, endOfDay(clk, day)), true, nil
	case paramAtLeast:
		return compare(dbtype.OpGreaterThanOrEqual, args[param.Name]), true, nil
	case paramAtMost:
		return compare(dbtype.OpLessThanOrEqual, args[param.Name]), true, nil
	}

	return nil, false, fmt.Errorf("parameter %q has no filter", param.Name)
}
