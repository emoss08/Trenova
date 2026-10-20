package agentquerytoolservice

import (
	"fmt"
	"maps"
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
	// Fields are a selected record's own fields, one level down.
	Fields []graphQLField
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
	suffixID   = "Id"
	prefixHas  = "has"
)

// unfilteredFields are selected by operations but are never what a person
// narrows a list by: the tenant, the row's version, its id, which a get reads,
// and the Google Places id an address was geocoded to, which nothing a model
// reads supplies.
var unfilteredFields = map[string]struct{}{
	recordIDField:        {},
	tenantOrganizationID: {},
	tenantBusinessUnitID: {},
	rowVersion:           {},
	geocodedPlaceID:      {},
}

const (
	tenantOrganizationID = "organizationId"
	tenantBusinessUnitID = "businessUnitId"
	rowVersion           = "version"
	geocodedPlaceID      = "placeId"
	fieldNotes           = "notes"
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
	paramPresent
	paramReference
)

// graphQLParam is one typed parameter of a tool that reads through GraphQL,
// derived from a field the operation selects and the server lets a table
// filter on.
type graphQLParam struct {
	Name   string
	Field  string
	Kind   paramKind
	Schema map[string]any
	// Holder is the GraphQL type of a nested record a dotted Field belongs
	// to (WorkerProfile for profile.cdlClass); empty for the record's own.
	Holder string
	// OnInput marks a parameter that is one of the input's own fields
	// (WorkerPTOEntriesInput.workerId), sent as that field rather than as a
	// field filter. Like any argument the screen sends, it is not judged by
	// a record field's sensitivity.
	OnInput bool
}

// tableInputFields are what a table-shaped input holds for the table itself;
// its other fields are the screen's own filters.
var tableInputFields = map[string]struct{}{
	inputFirst: {}, inputAfter: {}, inputQuery: {}, inputFieldFilters: {}, inputSort: {},
	inputFilterGroups: {},
}

const (
	includePrefix     = "include"
	inputFilterGroups = "filterGroups"
	graphQLKindInput  = "input"
)

// tableShaped reports whether an input holds a table's fields, as
// DataTableConnectionInput does, under its own name.
func tableShaped(fields []graphQLField) bool {
	have := 0
	for _, field := range fields {
		if _, core := tableInputFields[field.Name]; core && field.Name != inputFilterGroups {
			have++
		}
	}

	return have == len(tableInputFields)-1
}

// tableInput is the variable a list's operation takes its table input in,
// and that input's type.
func (r *graphQLRead) tableInput() (name, typ string, ok bool) {
	for _, variable := range r.Variables {
		if variable.Type == dataTableInput || tableShaped(graphQLInputs[variable.Type]) {
			return variable.Name, variable.Type, true
		}
	}

	return "", "", false
}

// inputParams are a table-shaped input's own fields as parameters, sent on
// the input: the screen's filters (a PTO status, a worker) beside the
// table's. A field a record parameter already names, and an include*
// toggle the screen sets for its own rendering, is left out.
func inputParams(typ string, taken map[string]struct{}) []graphQLParam {
	var out []graphQLParam
	for idx := range graphQLInputs[typ] {
		field := &graphQLInputs[typ][idx]
		if _, core := tableInputFields[field.Name]; core ||
			strings.HasPrefix(field.Name, includePrefix) || field.Kind == graphQLKindInput {
			continue
		}
		if _, dup := taken[field.Name]; dup {
			continue
		}
		for _, param := range inputFieldParams(field) {
			param.OnInput = true
			out = append(out, param)
		}
	}

	return out
}

// inputFieldParams is one input field's parameter. A Timestamp named for a
// range end (startDateFrom) is a day, read in the organization's zone.
func inputFieldParams(field *graphQLField) []graphQLParam {
	words := stringutils.HumanizeSnakeCase(stringutils.ConvertCamelToSnake(field.Name))
	describe := func(fallback string) string {
		if field.Description != "" {
			return field.Description + " " + fallback
		}
		return fallback
	}
	if field.Type == graphQLTypeTimestamp {
		switch {
		case strings.HasSuffix(field.Name, suffixFrom):
			return []graphQLParam{{Name: field.Name, Field: field.Name, Kind: paramDayFrom,
				Schema: agenttoolschema.Date(describe("Only records whose " + words +
					" is this day or later, in the organization's timezone."))}}
		case strings.HasSuffix(field.Name, suffixTo):
			return []graphQLParam{{Name: field.Name, Field: field.Name, Kind: paramDayTo,
				Schema: agenttoolschema.Date(describe("Only records whose " + words +
					" is this day or earlier, in the organization's timezone."))}}
		}
		return nil
	}
	if field.Kind == graphQLKindEnum && !field.List {
		return []graphQLParam{{Name: field.Name, Field: field.Name, Kind: paramEquals,
			Schema: agenttoolschema.Enum(describe("Only records whose "+words+" is this."),
				agenttoolschema.Derived(graphQLEnumSource+field.Type, field.Enum))}}
	}

	return paramsFor(field)
}

// compileInputs is the input fields a call's on-input parameters set.
func compileInputs(params []graphQLParam, args map[string]any, clk clock) (map[string]any, error) {
	out := make(map[string]any)
	for _, param := range params {
		raw, present := args[param.Name]
		if !param.OnInput || !present || raw == nil {
			continue
		}
		switch param.Kind {
		case paramDayFrom:
			day, err := readDay(args, param.Name, clk)
			if err != nil {
				return nil, err
			}
			out[param.Field] = day
		case paramDayTo:
			day, err := readDay(args, param.Name, clk)
			if err != nil {
				return nil, err
			}
			out[param.Field] = endOfDay(clk, day)
		case paramIn, paramEquals, paramContains, paramAtLeast, paramAtMost, paramPresent,
			paramReference:
			out[param.Field] = raw
		}
	}

	return out, nil
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
	params = append(params, r.nestedParams(config.FilterableFields)...)
	slices.SortFunc(params, func(a, b graphQLParam) int { return strings.Compare(a.Name, b.Name) })

	return params
}

// nestedParams are the parameters of the filters the configuration allows
// one record down (profile.endorsement), typed by the nested record's field
// and named for the path (profileEndorsement).
func (r *graphQLRead) nestedParams(filterable map[string]bool) []graphQLParam {
	var out []graphQLParam
	for _, key := range slices.Sorted(maps.Keys(filterable)) {
		parent, child, ok := r.nestedField(key)
		if !ok || !filterable[key] {
			continue
		}
		for _, param := range paramsFor(child) {
			param.Name = parent.Name + stringutils.CapitalizeFirst(param.Name)
			param.Field = parent.Name + "." + param.Field
			param.Holder = parent.Type
			out = append(out, param)
		}
	}

	return out
}

// nestedField is the record field and its own field a dotted path names,
// when the operation reads that record and it is a part of the list's own
// record, named for it (a Worker's WorkerProfile): a linked record (a state,
// an equipment type) is filtered by its id, not by its fields.
func (r *graphQLRead) nestedField(path string) (parent, child *graphQLField, ok bool) {
	head, tail, dotted := strings.Cut(path, ".")
	if !dotted || strings.Contains(tail, ".") {
		return nil, nil, false
	}
	idx := slices.IndexFunc(r.Fields, func(f graphQLField) bool { return f.Name == head })
	if idx < 0 || len(r.Fields[idx].Fields) == 0 {
		return nil, nil, false
	}
	parent = &r.Fields[idx]
	if !strings.HasPrefix(parent.Type, r.Node) || parent.Type == r.Node {
		return nil, nil, false
	}
	at := slices.IndexFunc(parent.Fields, func(f graphQLField) bool { return f.Name == tail })
	if at < 0 || parent.Fields[at].List {
		return nil, nil, false
	}
	if _, skip := unfilteredFields[tail]; skip {
		return nil, nil, false
	}

	return parent, &parent.Fields[at], true
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
	for _, key := range slices.Sorted(maps.Keys(config.SortableFields)) {
		if _, _, ok := r.nestedField(key); ok && config.SortableFields[key] {
			out = append(out, key)
		}
	}
	slices.Sort(out)

	return out
}

// conventionNames are the names agent-tool-conventions.md gives a free-text
// field, which a parameter derived from it takes; its filter still names the
// field.
var conventionNames = map[string]string{
	fieldNotes: "note",
}

// paramsFor derives a field's parameters, named by the conventions.
func paramsFor(field *graphQLField) []graphQLParam {
	params := fieldParams(field)
	if name, renamed := conventionNames[field.Name]; renamed {
		for idx := range params {
			if params[idx].Name == field.Name {
				params[idx].Name = name
			}
		}
	}

	return params
}

func fieldParams(field *graphQLField) []graphQLParam {
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
					"true for only the records whose " + field.Name + " is true, false for " +
						"only those whose " + field.Name + " is false.",
				),
			},
		}}
	case graphQLTypeID:
		params := []graphQLParam{{
			Name: field.Name, Field: field.Name, Kind: paramReference,
			Schema: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: describe("Only records with this " + words + "."),
			},
		}}
		if present, ok := presenceParam(field, describe); ok {
			params = append(params, present)
		}
		return params
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

// presenceParam is the has<Thing> parameter a nullable reference becomes, so
// a call can ask for the records that have one, or lack one, without naming
// which.
func presenceParam(field *graphQLField, describe func(string) string) (graphQLParam, bool) {
	thing, ok := strings.CutSuffix(field.Name, suffixID)
	if field.NonNull || !ok || thing == "" {
		return graphQLParam{}, false
	}
	words := stringutils.HumanizeSnakeCase(stringutils.ConvertCamelToSnake(thing))

	return graphQLParam{
		Name:  prefixHas + strings.ToUpper(thing[:1]) + thing[1:],
		Field: field.Name,
		Kind:  paramPresent,
		Schema: map[string]any{
			toolschema.KeyType: toolschema.TypeBoolean,
			toolschema.KeyDescription: describe(
				"true for only records that have a " + words + ", false for only those " +
					"that have none.",
			),
		},
	}, true
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
		if param.OnInput || !present || raw == nil {
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
	case paramEquals, paramReference:
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
	case paramPresent:
		has, ok := args[param.Name].(bool)
		if !ok {
			return nil, false, fmt.Errorf("%s must be true or false", param.Name)
		}
		if has {
			return compare(dbtype.OpIsNotNull, nil), true, nil
		}
		return compare(dbtype.OpIsNull, nil), true, nil
	}

	return nil, false, fmt.Errorf("parameter %q has no filter", param.Name)
}
