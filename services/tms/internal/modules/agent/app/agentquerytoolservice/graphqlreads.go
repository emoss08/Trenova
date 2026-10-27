package agentquerytoolservice

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
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
	// SelfScoped marks a read whose resolver acts only on the caller's own
	// record and checks no resource; the tool's policy, scoped to the
	// caller, is its check.
	SelfScoped bool
	Entity     func() domaintypes.PostgresSearchable
	// Unpaged marks a list whose root field returns every record at once,
	// keyed by its own arguments (a worker's checklists by workerId).
	Unpaged bool
	// Items marks an unpaged list whose root holds its records under items
	// (a carrier invoice list read by limit and offset).
	Items     bool
	Variables []graphQLVariable
	Fields    []graphQLField
	// Alongside are the reads a tool makes beside its own, for records its
	// next call needs or that say what its own records mean.
	Alongside []graphQLAlongside
	// Instead are the reads a list makes in place of its own when a call
	// names their own variables.
	Instead []graphQLAlongside
}

// graphQLVariable is one of the operation's variables, described by the
// root field argument it is sent as.
type graphQLVariable struct {
	Name        string
	Type        string
	NonNull     bool
	List        bool
	Kind        string
	Enum        []string
	Description string
}

// graphQLAlongside is a read a tool makes beside or instead of its own,
// checked by its own resolver for its own resource: a connection read
// through its table input, or an unpaged list or one record keyed by the
// tool's own arguments.
type graphQLAlongside struct {
	Operation string
	RootField string
	Resource  string
	// Input is the variable a connection read takes its table input in.
	Input string
	// Keys are the variables a keyed read takes from the tool's own call.
	Keys []string
	// Object marks a keyed read whose root returns one record.
	Object bool
	// Own are the variables of a read made instead of a list's that the
	// list does not take: naming one chooses the read.
	Own []string
	// Variables are a read made instead's variables, described by the root
	// field arguments they are sent as.
	Variables []graphQLVariable
	// Set are the optional variables of a keyed read that the list's call
	// does not declare, which the spec sets as the screen does.
	Set []string
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
// the Google Places id an address was geocoded to, which nothing a model
// reads supplies, and the words of a cancelled record's reason, which the
// conventions keep for the reason a cancel is given.
var unfilteredFields = map[string]struct{}{
	recordIDField:        {},
	tenantOrganizationID: {},
	tenantBusinessUnitID: {},
	rowVersion:           {},
	geocodedPlaceID:      {},
	cancelReason:         {},
}

const (
	tenantOrganizationID = "organizationId"
	tenantBusinessUnitID = "businessUnitId"
	rowVersion           = "version"
	geocodedPlaceID      = "placeId"
	cancelReason         = "cancelReason"
	fieldNotes           = "notes"
	fieldCancelReason    = "cancelReason"
	paramReason          = "reason"
	paramNote            = "note"
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
	// paramReferences is a list of ids a root argument takes (the documents
	// whose sync state is read).
	paramReferences
	// paramInstantFrom and paramInstantTo bound an instant people name to
	// the minute (when an EDI message was created), where a day is too
	// coarse.
	paramInstantFrom
	paramInstantTo
	// paramDayText is a day sent as the text a String argument takes (a fuel
	// index price's day), not as a timestamp.
	paramDayText
)

const (
	// dayTextFormat is what the description of a String argument that takes
	// a day says it is written as, and dayTextLayout how it is parsed.
	dayTextFormat = "YYYY-MM-DD"
	dayTextLayout = "2006-01-02"
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
	// (WorkerPTOEntriesInput.workerId), or one of the root field's own
	// arguments, sent as that field rather than as a field filter. Like any
	// argument the screen sends, it is not judged by a record field's
	// sensitivity.
	OnInput bool
	// Minor marks an amount filter on a field held in minor units, which the
	// call sends as a decimal (amountMin on amountMinor).
	Minor bool
	// Arg is the root variable an argument parameter is sent as (a drift
	// list's integrationType), and ArgField the field of that variable's
	// filter input it sets (statuses), when it is one.
	Arg      string
	ArgField string
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

// withInputParams adds the input's own parameters to the record's. A range
// end on the input takes the place of the record's of the same name, as the
// resolver reads it with the screen's meaning (time off's startDateFrom with
// overlapping); any other name the record already takes keeps the record's,
// which can match several values.
func withInputParams(record, onInput []graphQLParam) []graphQLParam {
	ranges := make(map[string]struct{}, len(onInput))
	for _, param := range onInput {
		if param.Kind == paramDayFrom || param.Kind == paramDayTo {
			ranges[param.Name] = struct{}{}
		}
	}
	out := slices.DeleteFunc(slices.Clone(record), func(param graphQLParam) bool {
		_, shadowed := ranges[param.Name]
		return shadowed
	})
	taken := make(map[string]struct{}, len(out))
	for _, param := range out {
		taken[param.Name] = struct{}{}
	}
	for _, param := range onInput {
		if _, dup := taken[param.Name]; !dup {
			out = append(out, param)
		}
	}

	return out
}

// inputParams are a table-shaped input's own fields as parameters, sent on
// the input: the screen's filters (a PTO status, a worker) beside the
// table's. An include* toggle the screen sets for its own rendering is left
// out.
func inputParams(typ string) []graphQLParam {
	var out []graphQLParam
	for idx := range graphQLInputs[typ] {
		field := &graphQLInputs[typ][idx]
		if _, core := tableInputFields[field.Name]; core ||
			strings.HasPrefix(field.Name, includePrefix) || field.Kind == graphQLKindInput {
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
// range end (startDateFrom) is a day, read in the organization's zone, and a
// described Boolean is a switch its own description explains (overlapping).
func inputFieldParams(field *graphQLField) []graphQLParam {
	words := stringutils.HumanizeSnakeCase(stringutils.ConvertCamelToSnake(field.Name))
	describe := func(fallback string) string {
		if field.Description != "" {
			return field.Description + " " + fallback
		}
		return fallback
	}
	if field.Type == graphQLTypeTimestamp {
		words = stringutils.HumanizeSnakeCase(stringutils.ConvertCamelToSnake(
			strings.TrimSuffix(strings.TrimSuffix(field.Name, suffixFrom), suffixTo)))
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
	if field.Type == graphQLTypeBoolean && field.Description != "" {
		return []graphQLParam{{Name: field.Name, Field: field.Name, Kind: paramEquals,
			Schema: map[string]any{
				toolschema.KeyType:        toolschema.TypeBoolean,
				toolschema.KeyDescription: field.Description,
			}}}
	}
	if field.Kind == graphQLKindEnum && !field.List {
		what := "Only records whose " + words + " is this."
		if field.Name == inputSort {
			what = "How the list is ordered."
		}
		return []graphQLParam{{Name: field.Name, Field: field.Name, Kind: paramEquals,
			Schema: agenttoolschema.Enum(describe(what),
				agenttoolschema.Derived(graphQLEnumSource+field.Type, field.Enum))}}
	}

	return paramsFor(field)
}

// compileInputs is the input fields a call's on-input parameters set.
func compileInputs(params []graphQLParam, args map[string]any, clk clock) (map[string]any, error) {
	out := make(map[string]any)
	for _, param := range params {
		raw, present := args[param.Name]
		if !param.OnInput || param.Arg != "" || !present || raw == nil {
			continue
		}
		value, err := inputValue(&param, args, raw, clk)
		if err != nil {
			return nil, err
		}
		out[param.Field] = value
	}

	return out, nil
}

// compileArgs is the root variables a call's argument parameters set, each
// filter input built from the fields its parameters name.
func compileArgs(params []graphQLParam, args map[string]any, clk clock) (map[string]any, error) {
	out := make(map[string]any)
	for _, param := range params {
		raw, present := args[param.Name]
		if param.Arg == "" || !present || raw == nil {
			continue
		}
		value, err := inputValue(&param, args, raw, clk)
		if err != nil {
			return nil, err
		}
		if param.ArgField == "" {
			out[param.Arg] = value
			continue
		}
		filter, _ := out[param.Arg].(map[string]any)
		if filter == nil {
			filter = make(map[string]any)
			out[param.Arg] = filter
		}
		filter[param.ArgField] = value
	}

	return out, nil
}

// inputValue is the value an on-input or argument parameter sends: a day in
// the organization's zone for a range end, the call's own value otherwise.
func inputValue(param *graphQLParam, args map[string]any, raw any, clk clock) (any, error) {
	switch param.Kind {
	case paramDayFrom:
		day, err := readDay(args, param.Name, clk)
		if err != nil {
			return nil, err
		}
		return day, nil
	case paramDayTo:
		day, err := readDay(args, param.Name, clk)
		if err != nil {
			return nil, err
		}
		return endOfDay(clk, day), nil
	case paramDayText:
		return readDayText(args, param.Name)
	case paramIn, paramEquals, paramContains, paramAtLeast, paramAtMost, paramPresent,
		paramReference, paramReferences, paramInstantFrom, paramInstantTo:
	}

	return raw, nil
}

// typedParams derives a list tool's parameters from its record type: every
// field the entity's own filter configuration (the one the resolver compiles
// fieldFilters with) marks filterable, typed by its schema type. The
// operation's selection decides only what a result carries; a field the
// table does not show can still narrow it, as the resolver accepts.
func (r *graphQLRead) typedParams() []graphQLParam {
	if r.Entity == nil {
		return nil
	}
	config := entityFields(r.Entity())
	params := make([]graphQLParam, 0, len(r.Fields))
	for idx := range r.Fields {
		field := &r.Fields[idx]
		if _, skip := unfilteredFields[field.Name]; skip || field.List {
			continue
		}
		if !config.FilterableFields[field.Name] {
			continue
		}
		derived := paramsFor(field)
		if field.Type == graphQLTypeString && config.EnumMap[field.Name] {
			derived = exactParams(derived)
		}
		params = append(params, derived...)
	}
	params = append(params, r.nestedParams(config.FilterableFields)...)
	slices.SortFunc(params, func(a, b graphQLParam) int { return strings.Compare(a.Name, b.Name) })

	return params
}

// configuredSearch are the record fields a table input's free-text query
// matches, read from the record's search configuration as the repository's
// list query reads it (listquery): its searchable columns, or, when it names
// none, every filterable field of the record itself that is not an enum. A column is named by the record field it is stored in, which is how
// its sensitivity is declared, not by the column's own name, which an enum
// column is also filed under. A column of a joined record (customer.name)
// is never matched by the configuration's search, so it is not read.
func (r *graphQLRead) configuredSearch() []string {
	if r.Entity == nil {
		return nil
	}
	entity := r.Entity()
	config := entityFields(entity)
	searchable := entity.GetPostgresSearchConfig().SearchableFields
	if len(searchable) == 0 {
		fields := make([]string, 0, len(config.FieldMap))
		for field := range config.FieldMap {
			if !config.EnumMap[field] {
				fields = append(fields, field)
			}
		}
		slices.Sort(fields)

		return fields
	}
	byColumn := make(map[string]string, len(config.FieldMap))
	for field, column := range config.FieldMap {
		if _, named := byColumn[column]; named && field == column {
			continue
		}
		byColumn[column] = field
	}
	fields := make([]string, 0, len(searchable))
	for _, column := range searchable {
		if strings.Contains(column.Name, ".") {
			continue
		}
		field, mapped := byColumn[column.Name]
		if !mapped {
			field = stringutils.LowerCamel(column.Name)
		}
		if !slices.Contains(fields, field) {
			fields = append(fields, field)
		}
	}

	return fields
}

// exactParams turns a text parameter into an equality one, for a String field
// whose column is a database enum (a journal entry's status): the database
// cannot match part of an enum value, and a model names the whole value.
func exactParams(params []graphQLParam) []graphQLParam {
	out := slices.Clone(params)
	for idx := range out {
		if out[idx].Kind != paramContains {
			continue
		}
		words := stringutils.HumanizeSnakeCase(stringutils.ConvertCamelToSnake(out[idx].Field))
		out[idx].Kind = paramEquals
		out[idx].Schema = map[string]any{
			toolschema.KeyType:        toolschema.TypeString,
			toolschema.KeyDescription: "Only records whose " + words + " is exactly this.",
		}
	}

	return out
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
	if r.Entity == nil {
		return nil
	}
	config := entityFields(r.Entity())
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
	fieldNotes:        paramNote,
	fieldCancelReason: paramReason,
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

// instantParams are a list's parameters with each Timestamp the spec names as
// an instant bounded to the minute rather than the day: createdAtFrom/To
// take a date and time with its offset.
func instantParams(tool string, params []graphQLParam, instants []string) ([]graphQLParam, error) {
	for _, field := range instants {
		matched := false
		for idx := range params {
			param := &params[idx]
			if param.Field != field || param.OnInput ||
				(param.Kind != paramDayFrom && param.Kind != paramDayTo) {
				continue
			}
			matched = true
			words := stringutils.HumanizeSnakeCase(stringutils.ConvertCamelToSnake(field))
			if param.Kind == paramDayFrom {
				param.Kind = paramInstantFrom
				param.Schema = agenttoolschema.DateTime("Only records whose " + words +
					" is at or after this instant.")
				continue
			}
			param.Kind = paramInstantTo
			param.Schema = agenttoolschema.DateTime("Only records whose " + words +
				" is at or before this instant.")
		}
		if !matched {
			return nil, fmt.Errorf("%s names %q an instant, which is not one of its dated "+
				"fields", tool, field)
		}
	}

	return params, nil
}

// readInstant is an instant parameter as Unix seconds, refusing anything
// but a date and time with its offset.
func readInstant(args map[string]any, name string) (int64, error) {
	raw := strings.TrimSpace(optionalString(args, name))
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return 0, fmt.Errorf("parameter %q must be a date and time with its offset, such as "+
			"2026-03-14T09:30:00-05:00", name)
	}

	return at.Unix(), nil
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
	case paramIn, paramReferences:
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
	case paramInstantFrom, paramInstantTo:
		at, atErr := readInstant(args, param.Name)
		if atErr != nil {
			return nil, false, atErr
		}
		if param.Kind == paramInstantFrom {
			return compare(dbtype.OpGreaterThanOrEqual, at), true, nil
		}
		return compare(dbtype.OpLessThanOrEqual, at), true, nil
	case paramAtLeast, paramAtMost:
		value := args[param.Name]
		if param.Minor {
			minor, minorErr := minorUnits(value, param.Name)
			if minorErr != nil {
				return nil, false, minorErr
			}
			value = minor
		}
		if param.Kind == paramAtLeast {
			return compare(dbtype.OpGreaterThanOrEqual, value), true, nil
		}
		return compare(dbtype.OpLessThanOrEqual, value), true, nil
	case paramPresent:
		has, ok := args[param.Name].(bool)
		if !ok {
			return nil, false, fmt.Errorf("%s must be true or false", param.Name)
		}
		if has {
			return compare(dbtype.OpIsNotNull, nil), true, nil
		}
		return compare(dbtype.OpIsNull, nil), true, nil
	case paramDayText:
	}

	return nil, false, fmt.Errorf("parameter %q has no filter", param.Name)
}

// readDayText is a day a String argument takes, checked as a day and sent as
// the text the argument's description names.
func readDayText(args map[string]any, name string) (string, error) {
	raw := strings.TrimSpace(optionalString(args, name))
	if _, err := time.Parse(dayTextLayout, raw); err != nil {
		return "", fmt.Errorf("parameter %q must be a date as %s", name, dayTextFormat)
	}

	return raw, nil
}
