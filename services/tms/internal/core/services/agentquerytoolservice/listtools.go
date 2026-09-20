package agentquerytoolservice

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	defaultListLimit = 25
	maxListLimit     = 50
	// maxListFilters bounds how much a single call can ask for. Six is more
	// than any question a person phrases in one sentence, and the query builder
	// truncates past its own cap silently, which would filter on less than the
	// model was told it filtered on.
	maxListFilters  = 6
	maxRelativeDays = 3650
)

// filterKind decides which operators a field accepts and how its value is read.
// It is declared per field in the catalog rather than inferred from the Go type,
// because a Go string is a free-text name in one column and a closed enum in the
// next, and the difference is the whole of what keeps a model from inventing a
// status that does not exist.
type filterKind string

const (
	filterText   filterKind = "text"
	filterEnum   filterKind = "enum"
	filterDate   filterKind = "date"
	filterNumber filterKind = "number"
	filterBool   filterKind = "boolean"
)

type listField struct {
	Name string
	Kind filterKind
	// Note explains the column in the words the business uses, when the name
	// alone does not.
	Note string
	// Values closes an enum. A value outside the set is refused rather than
	// passed through, because the query would match nothing and the model would
	// report that as "there are none".
	Values   []string
	Sortable bool
}

// listSpec is one entry in the catalog: an entity, the fields a model may
// narrow it by, and the repository call that serves it.
//
// One tool is generated per entry rather than a single list_records(entity, …)
// tool, because AgentQueryTool.PermissionResource returns exactly one resource
// and the agent builder groups the picker by it. A single tool would have to
// resolve its resource at call time, and an organization could not grant an
// agent "list workers" without also granting "list customers".
type listSpec struct {
	name         string
	entityPlural string
	summary      string
	resource     permission.Resource
	fields       []listField
	// config is the entity's computed field configuration. A curated field that
	// no longer maps to a column is dropped silently by the query builder, so it
	// is checked here and refused loudly instead.
	config *domaintypes.FieldConfiguration
	fetch  func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error)
}

type operatorArity uint8

const (
	arityValue operatorArity = iota
	arityNone
	arityDays
	arityList
)

var operatorArities = map[dbtype.Operator]operatorArity{
	dbtype.OpEqual:              arityValue,
	dbtype.OpNotEqual:           arityValue,
	dbtype.OpGreaterThan:        arityValue,
	dbtype.OpGreaterThanOrEqual: arityValue,
	dbtype.OpLessThan:           arityValue,
	dbtype.OpLessThanOrEqual:    arityValue,
	dbtype.OpContains:           arityValue,
	dbtype.OpStartsWith:         arityValue,
	dbtype.OpEndsWith:           arityValue,
	dbtype.OpIn:                 arityList,
	dbtype.OpNotIn:              arityList,
	dbtype.OpIsNull:             arityNone,
	dbtype.OpIsNotNull:          arityNone,
	dbtype.OpToday:              arityNone,
	dbtype.OpYesterday:          arityNone,
	dbtype.OpTomorrow:           arityNone,
	dbtype.OpLastNDays:          arityDays,
	dbtype.OpNextNDays:          arityDays,
}

var operatorsByKind = map[filterKind][]dbtype.Operator{
	filterText: {
		dbtype.OpEqual, dbtype.OpNotEqual, dbtype.OpContains, dbtype.OpStartsWith,
		dbtype.OpEndsWith, dbtype.OpIn, dbtype.OpNotIn, dbtype.OpIsNull, dbtype.OpIsNotNull,
	},
	filterEnum: {dbtype.OpEqual, dbtype.OpNotEqual, dbtype.OpIn, dbtype.OpNotIn},
	filterDate: {
		dbtype.OpGreaterThan, dbtype.OpGreaterThanOrEqual, dbtype.OpLessThan,
		dbtype.OpLessThanOrEqual, dbtype.OpLastNDays, dbtype.OpNextNDays, dbtype.OpToday,
		dbtype.OpYesterday, dbtype.OpTomorrow, dbtype.OpIsNull, dbtype.OpIsNotNull,
	},
	filterNumber: {
		dbtype.OpEqual, dbtype.OpNotEqual, dbtype.OpGreaterThan, dbtype.OpGreaterThanOrEqual,
		dbtype.OpLessThan, dbtype.OpLessThanOrEqual, dbtype.OpIsNull, dbtype.OpIsNotNull,
	},
	filterBool: {dbtype.OpEqual, dbtype.OpNotEqual},
}

type listTool struct {
	spec        listSpec
	byName      map[string]listField
	fieldNames  []string
	sortNames   []string
	operators   []string
	description string
}

func newListTool(spec listSpec) serviceports.AgentQueryTool {
	tool := &listTool{
		spec:       spec,
		byName:     make(map[string]listField, len(spec.fields)),
		fieldNames: make([]string, 0, len(spec.fields)),
		sortNames:  make([]string, 0, len(spec.fields)),
	}

	seenOperators := make(map[dbtype.Operator]bool, len(operatorArities))
	operators := make([]string, 0, len(operatorArities))

	for _, field := range spec.fields {
		tool.byName[field.Name] = field
		tool.fieldNames = append(tool.fieldNames, field.Name)
		if field.Sortable {
			tool.sortNames = append(tool.sortNames, field.Name)
		}
		for _, operator := range operatorsByKind[field.Kind] {
			if seenOperators[operator] {
				continue
			}
			seenOperators[operator] = true
			operators = append(operators, string(operator))
		}
	}

	tool.operators = operators
	tool.description = buildListDescription(spec)

	return tool
}

func buildListDescription(spec listSpec) string {
	var b strings.Builder
	b.WriteString(spec.summary)
	b.WriteString(" Filter on: ")

	for i, field := range spec.fields {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(field.Name)
		b.WriteString(" (")
		b.WriteString(string(field.Kind))
		if len(field.Values) > 0 {
			b.WriteString(": ")
			b.WriteString(strings.Join(field.Values, ", "))
		}
		b.WriteString(")")
		if field.Note != "" {
			b.WriteString(" — ")
			b.WriteString(field.Note)
		}
	}

	b.WriteString(
		". Dates are resolved on the server: use nextndays or lastndays with a day " +
			"count, or today, or a calendar date such as 2026-03-01. Never send a Unix " +
			"timestamp you worked out yourself.",
	)

	return b.String()
}

func (t *listTool) Name() string { return t.spec.name }

func (t *listTool) Description() string { return t.description }

func (t *listTool) PermissionResource() permission.Resource { return t.spec.resource }

func (t *listTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type": "string",
				"description": "Optional free text matched across the searchable columns. " +
					"Omit it when you are filtering on a field.",
			},
			"filters": map[string]any{
				"type":     "array",
				"maxItems": maxListFilters,
				"description": "Conditions, combined with AND. Give value for a single " +
					"comparison, values for in and notin, days for nextndays and lastndays, " +
					"and nothing at all for isnull, isnotnull, today, yesterday and tomorrow.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"field":    map[string]any{"type": "string", "enum": t.fieldNames},
						"operator": map[string]any{"type": "string", "enum": t.operators},
						"value":    map[string]any{"type": "string"},
						"values": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
						"days": map[string]any{"type": "integer"},
					},
					"required":             []string{"field", "operator"},
					"additionalProperties": false,
				},
			},
			"sortBy": map[string]any{
				"type":        "string",
				"enum":        t.sortNames,
				"description": "Field to order by. Defaults to most recently created.",
			},
			"sortDirection": map[string]any{
				"type": "string",
				"enum": []string{"asc", "desc"},
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("How many rows to return, at most %d.", maxListLimit),
			},
		},
		"additionalProperties": false,
	}
}

func (t *listTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	criteria := newSearchCriteria(t.spec.entityPlural)

	query := optionalString(params.Params, "query")
	criteria.text(query)

	filters, err := t.buildFilters(params.Params, criteria)
	if err != nil {
		return nil, err
	}

	sorting, err := t.buildSort(params.Params)
	if err != nil {
		return nil, err
	}

	limit := optionalInt(params.Params, "limit", defaultListLimit)
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	rows, err := t.spec.fetch(ctx, &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		Pagination:   pagination.Info{Limit: limit},
		Query:        query,
		FieldFilters: filters,
		Sort:         sorting,
	})
	if err != nil {
		return nil, err
	}

	return criteria.result(rows, len(rows)), nil
}

func (t *listTool) buildFilters(
	params map[string]any,
	criteria *searchCriteria,
) ([]domaintypes.FieldFilter, error) {
	raw, ok := params["filters"]
	if !ok || raw == nil {
		return nil, nil
	}

	entries, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf(
			`parameter "filters" must be an array of {field, operator, value} objects`,
		)
	}
	if len(entries) > maxListFilters {
		return nil, fmt.Errorf(
			"at most %d filters can be applied at once; narrow the question instead",
			maxListFilters,
		)
	}

	built := make([]domaintypes.FieldFilter, 0, len(entries)*2)
	for _, entry := range entries {
		condition, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf(
				`each entry in "filters" must be an object with field and operator`,
			)
		}

		next, err := t.buildFilter(condition, criteria)
		if err != nil {
			return nil, err
		}
		built = append(built, next...)
	}

	return built, nil
}

func (t *listTool) buildFilter(
	condition map[string]any,
	criteria *searchCriteria,
) ([]domaintypes.FieldFilter, error) {
	field, err := t.resolveField(optionalString(condition, "field"))
	if err != nil {
		return nil, err
	}

	operator := dbtype.Operator(strings.ToLower(optionalString(condition, "operator")))
	if !operatorAllowed(field.Kind, operator) {
		return nil, fmt.Errorf(
			"%q is not an operator for %q, which is a %s field; use one of: %s",
			operator, field.Name, field.Kind, joinOperators(operatorsByKind[field.Kind]),
		)
	}

	switch operatorArities[operator] {
	case arityNone:
		criteria.field(field.Name, operatorPhrase(operator))

		return []domaintypes.FieldFilter{{Field: field.Name, Operator: operator}}, nil
	case arityDays:
		return t.buildWindow(field, operator, condition, criteria)
	case arityList:
		return t.buildListFilter(field, operator, condition, criteria)
	default:
		return t.buildValueFilter(field, operator, condition, criteria)
	}
}

func (t *listTool) resolveField(name string) (listField, error) {
	field, ok := t.byName[name]
	if !ok {
		return listField{}, fmt.Errorf(
			"cannot filter %s on %q; the filterable fields are: %s",
			t.spec.entityPlural, name, strings.Join(t.fieldNames, ", "),
		)
	}

	// The query builder skips a field it cannot map and returns the page
	// unfiltered, which the model would present as a filtered answer. Catching
	// the drift here turns a wrong answer into a plain error.
	if t.spec.config != nil && !t.spec.config.FilterableFields[field.Name] {
		return listField{}, fmt.Errorf(
			"%q can no longer be queried on %s; the filterable fields are: %s",
			field.Name, t.spec.entityPlural, strings.Join(t.fieldNames, ", "),
		)
	}

	return field, nil
}

/*
buildWindow turns a relative day count into a bounded range.

OpNextNDays alone applies only `column <= now + N days`, so "expiring in the
next 30 days" would also match every row that lapsed years ago, and OpLastNDays
applies only the floor. Both are fine for a person reading a grid they can sort;
they are not fine for a model that will summarize the count in a sentence.
*/
func (t *listTool) buildWindow(
	field listField,
	operator dbtype.Operator,
	condition map[string]any,
	criteria *searchCriteria,
) ([]domaintypes.FieldFilter, error) {
	days := optionalInt(condition, "days", 0)
	if days <= 0 || days > maxRelativeDays {
		return nil, fmt.Errorf(
			"%q needs a whole number of days between 1 and %d in the \"days\" field",
			operator, maxRelativeDays,
		)
	}

	now := timeutils.NowUnix()
	span := int64(days) * secondsPerDay

	lower, upper := now, now+span
	phrase := fmt.Sprintf("within the next %d days", days)
	if operator == dbtype.OpLastNDays {
		lower, upper = now-span, now
		phrase = fmt.Sprintf("within the last %d days", days)
	}

	criteria.field(field.Name, phrase)

	return []domaintypes.FieldFilter{
		{Field: field.Name, Operator: dbtype.OpGreaterThanOrEqual, Value: lower},
		{Field: field.Name, Operator: dbtype.OpLessThanOrEqual, Value: upper},
	}, nil
}

func (t *listTool) buildListFilter(
	field listField,
	operator dbtype.Operator,
	condition map[string]any,
	criteria *searchCriteria,
) ([]domaintypes.FieldFilter, error) {
	raw, ok := condition["values"].([]any)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf(
			"%q on %q needs a non-empty \"values\" array", operator, field.Name,
		)
	}

	values := make([]any, 0, len(raw))
	labels := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("every entry in \"values\" for %q must be a string", field.Name)
		}

		value, err := coerceFilterValue(field, text)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
		labels = append(labels, text)
	}

	criteria.field(field.Name, operatorPhrase(operator)+" "+strings.Join(labels, ", "))

	return []domaintypes.FieldFilter{
		{Field: field.Name, Operator: operator, Value: values},
	}, nil
}

func (t *listTool) buildValueFilter(
	field listField,
	operator dbtype.Operator,
	condition map[string]any,
	criteria *searchCriteria,
) ([]domaintypes.FieldFilter, error) {
	raw := optionalString(condition, "value")
	if raw == "" {
		return nil, fmt.Errorf("%q on %q needs a \"value\"", operator, field.Name)
	}

	value, err := coerceFilterValue(field, raw)
	if err != nil {
		return nil, err
	}

	criteria.field(field.Name, operatorPhrase(operator)+" "+raw)

	return []domaintypes.FieldFilter{
		{Field: field.Name, Operator: operator, Value: value},
	}, nil
}

func (t *listTool) buildSort(params map[string]any) ([]domaintypes.SortField, error) {
	name := optionalString(params, "sortBy")
	if name == "" {
		return nil, nil
	}

	field, ok := t.byName[name]
	if !ok || !field.Sortable {
		return nil, fmt.Errorf(
			"cannot sort %s by %q; the sortable fields are: %s",
			t.spec.entityPlural, name, strings.Join(t.sortNames, ", "),
		)
	}

	direction := dbtype.SortDirectionDesc
	switch strings.ToLower(optionalString(params, "sortDirection")) {
	case "asc":
		direction = dbtype.SortDirectionAsc
	case "", "desc":
	default:
		return nil, fmt.Errorf(`"sortDirection" must be "asc" or "desc"`)
	}

	return []domaintypes.SortField{{Field: field.Name, Direction: direction}}, nil
}

func coerceFilterValue(field listField, raw string) (any, error) {
	switch field.Kind {
	case filterEnum:
		for _, candidate := range field.Values {
			if strings.EqualFold(candidate, raw) {
				return candidate, nil
			}
		}

		return nil, fmt.Errorf(
			"%q is not a value of %q; the values are: %s",
			raw, field.Name, strings.Join(field.Values, ", "),
		)
	case filterDate:
		return coerceDateValue(field.Name, raw)
	case filterNumber:
		if whole, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return whole, nil
		}
		fractional, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("%q on %q is not a number", raw, field.Name)
		}

		return fractional, nil
	case filterBool:
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("%q on %q must be true or false", raw, field.Name)
		}

		return parsed, nil
	case filterText:
		return raw, nil
	default:
		return raw, nil
	}
}

// coerceDateValue reads a calendar date rather than asking the model for an
// epoch. A fabricated timestamp is indistinguishable from a real one once it is
// in the SQL, and a model that has to do the arithmetic will sometimes get it
// wrong by a year.
func coerceDateValue(name, raw string) (int64, error) {
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return seconds, nil
	}

	if seconds, ok := namedDay(raw, timeutils.NowUnix()); ok {
		return seconds, nil
	}

	for _, layout := range []string{time.DateOnly, time.RFC3339} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC().Unix(), nil
		}
	}

	return 0, fmt.Errorf(
		"%q on %q is not a date; use YYYY-MM-DD, today, tomorrow or yesterday, "+
			"or nextndays/lastndays with a day count",
		raw, name,
	)
}

// namedDay resolves the words a person uses for a date.
//
// "today" is the obvious thing to send for "expiring from now on", and refusing
// it cost a whole round trip: the model sent it, read the correction, and asked
// again with the same question. That worked, but only because the refusal names
// what would have worked — on a weaker model the extra turn is where the
// conversation falls apart, and the retry is billed either way.
//
// Resolving it here rather than in the prompt also keeps the clock on the
// server. A model computing today's date is the arithmetic that put a medical
// card three months out of place.
func namedDay(raw string, now int64) (int64, bool) {
	const day = 86400

	midnight := func(seconds int64) int64 {
		t := time.Unix(seconds, 0).UTC()

		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
	}

	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "today", "now":
		return midnight(now), true
	case "tomorrow":
		return midnight(now + day), true
	case "yesterday":
		return midnight(now - day), true
	default:
		return 0, false
	}
}

func operatorAllowed(kind filterKind, operator dbtype.Operator) bool {
	for _, candidate := range operatorsByKind[kind] {
		if candidate == operator {
			return true
		}
	}

	return false
}

func joinOperators(operators []dbtype.Operator) string {
	names := make([]string, 0, len(operators))
	for _, operator := range operators {
		names = append(names, string(operator))
	}

	return strings.Join(names, ", ")
}

// operatorPhrase renders an operator the way the criteria read back to the
// model, so an empty result says "status equals Active" rather than "status eq".
func operatorPhrase(operator dbtype.Operator) string {
	switch operator { //nolint:exhaustive // the catalog exposes only these
	case dbtype.OpEqual:
		return "equals"
	case dbtype.OpNotEqual:
		return "is not"
	case dbtype.OpGreaterThan:
		return "after"
	case dbtype.OpGreaterThanOrEqual:
		return "on or after"
	case dbtype.OpLessThan:
		return "before"
	case dbtype.OpLessThanOrEqual:
		return "on or before"
	case dbtype.OpContains:
		return "contains"
	case dbtype.OpStartsWith:
		return "starts with"
	case dbtype.OpEndsWith:
		return "ends with"
	case dbtype.OpIn:
		return "one of"
	case dbtype.OpNotIn:
		return "not one of"
	case dbtype.OpIsNull:
		return "is empty"
	case dbtype.OpIsNotNull:
		return "is set"
	case dbtype.OpToday:
		return "is today"
	case dbtype.OpYesterday:
		return "was yesterday"
	case dbtype.OpTomorrow:
		return "is tomorrow"
	default:
		return string(operator)
	}
}

// listRows projects repository entities into the compact shapes the catalog
// declares. A raw entity carries relations, audit columns and custom fields, and
// twenty-five of them will crowd out the conversation they were fetched for.
func listRows[T any](items []T, row func(T) any) []any {
	rows := make([]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, row(item))
	}

	return rows
}
