package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
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

// The filter vocabulary lives in filtercatalog, which the table composer
// behind the Ask input compiles through as well. Aliasing rather than
// redeclaring is what keeps "in transit" meaning one thing to an agent and the
// same thing to the grid.
type filterKind = filtercatalog.Kind

const (
	filterText   = filtercatalog.KindText
	filterEnum   = filtercatalog.KindEnum
	filterDate   = filtercatalog.KindDate
	filterNumber = filtercatalog.KindNumber
	filterBool   = filtercatalog.KindBoolean
)

type listField = filtercatalog.Field

// listSpec is one entry in the catalog: an entity, the fields a model may
// narrow it by, and the repository call that serves it.
//
// One tool is generated per entry rather than a single list_records(entity, …)
// tool, because a tool's Policy names exactly one permission resource
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
	// fetchIn is fetch with the caller's clock, for a row that renders a
	// time of day and needs the organization's zone to do it. One of the
	// two is set.
	fetchIn func(ctx context.Context, opts *pagination.QueryOptions, clk clock) ([]any, error)

	access     fieldAccess
	fetchGated func(ctx context.Context, opts *pagination.QueryOptions, gate *fieldGate) ([]any, error)
}

type listTool struct {
	spec        listSpec
	resource    filtercatalog.Resource
	operators   []string
	description string
}

func newListTool(spec listSpec) serviceports.AgentQueryTool {
	seen := make(map[dbtype.Operator]bool, len(spec.fields)*4)
	operators := make([]string, 0, len(spec.fields)*4)
	for _, field := range spec.fields {
		for _, operator := range filtercatalog.Operators(field.Kind) {
			if seen[operator] {
				continue
			}
			seen[operator] = true
			operators = append(operators, string(operator))
		}
	}

	return &listTool{
		spec:        spec,
		resource:    catalogResource(spec),
		operators:   operators,
		description: buildListDescription(spec),
	}
}

// catalogResource is the one place a list spec becomes a catalogued resource,
// so the tool and the table composer narrow the same entity by the same
// fields.
func catalogResource(spec listSpec) filtercatalog.Resource {
	return filtercatalog.Resource{
		Tool:     spec.name,
		Resource: spec.resource,
		Entity:   spec.entityPlural,
		Summary:  spec.summary,
		Fields:   spec.fields,
		Config:   spec.config,
	}.Prepare()
}

// listDateRule is said once per tool rather than per date field: the server
// resolves every relative window, so a model never does calendar arithmetic.
const listDateRule = " Dates: today, nextndays/lastndays with days, or YYYY-MM-DD; " +
	"never Unix time."

// buildListDescription renders the summary and every filterable field as
// compactly as a model still reads them. A field with values or a note is
// spelled out on its own; the rest are only names, grouped under their kind.
func buildListDescription(spec listSpec) string {
	var b strings.Builder
	b.WriteString(spec.summary)
	b.WriteString(" Filters: ")

	written := 0
	separate := func() {
		if written > 0 {
			b.WriteString("; ")
		}
		written++
	}

	hasDate := false
	kinds := make([]filterKind, 0, len(spec.fields))
	plain := make(map[filterKind][]string, len(spec.fields))
	for _, field := range spec.fields {
		hasDate = hasDate || field.Kind == filterDate
		if len(field.Values) == 0 && field.Note == "" {
			if _, seen := plain[field.Kind]; !seen {
				kinds = append(kinds, field.Kind)
			}
			plain[field.Kind] = append(plain[field.Kind], field.Name)
			continue
		}

		separate()
		b.WriteString(field.Name)
		b.WriteString(" (")
		if len(field.Values) > 0 {
			b.WriteString(strings.Join(field.Values, "|"))
		} else {
			b.WriteString(string(field.Kind))
		}
		b.WriteString(")")
		if field.Note != "" {
			b.WriteString(": ")
			b.WriteString(field.Note)
		}
	}

	for _, kind := range kinds {
		separate()
		b.WriteString(string(kind))
		b.WriteString(": ")
		b.WriteString(strings.Join(plain[kind], ", "))
	}
	b.WriteString(".")

	if hasDate {
		b.WriteString(listDateRule)
	}

	return b.String()
}

func (t *listTool) Name() string { return t.spec.name }

func (t *listTool) Description() string { return t.description }

func (t *listTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: t.spec.resource,
	})
}

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
						"field": map[string]any{
							"type": "string",
							"enum": t.resource.FieldNames(),
						},
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
				"enum":        t.resource.SortableNames(),
				"description": "Field to order by. Defaults to most recently created.",
			},
			"sortDirection": map[string]any{
				"type":        "string",
				"enum":        []string{"asc", "desc"},
				"description": "asc for oldest or smallest first, desc for newest or largest.",
			},
			"limit":  pageSchema(defaultListLimit, maxListLimit)["limit"],
			"offset": pageSchema(defaultListLimit, maxListLimit)["offset"],
		},
		"additionalProperties": false,
	}
}

func (t *listTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	var gate *fieldGate
	if t.spec.fetchGated != nil {
		gate = t.spec.access.gate(ctx, params, t.spec.resource)
		if err := refuseWithheldFields(params.Params, gate); err != nil {
			return nil, err
		}
	}

	criteria := filtercatalog.NewCriteria(t.spec.entityPlural).At(clockFor(params))

	query := optionalString(params.Params, "query")
	criteria.Text(query)

	filters, err := t.buildFilters(params.Params, criteria)
	if err != nil {
		return nil, err
	}

	sorting, err := t.buildSort(params.Params)
	if err != nil {
		return nil, err
	}

	window := readPage(params.Params, defaultListLimit, maxListLimit)

	opts := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		Pagination:   pagination.Info{Limit: window.fetch(), Offset: window.offset},
		Query:        query,
		FieldFilters: filters,
		Sort:         sorting,
	}

	var rows []any
	switch {
	case gate != nil:
		rows, err = t.spec.fetchGated(ctx, opts, gate)
	case t.spec.fetchIn != nil:
		rows, err = t.spec.fetchIn(ctx, opts, criteria.Clock)
	default:
		rows, err = t.spec.fetch(ctx, opts)
	}
	if err != nil {
		return nil, err
	}

	rows, more := trim(window, rows)
	outcome := searchResult(criteria, rows, len(rows)).paged(window, more)
	if gate != nil {
		return gatedResult(outcome, gate), nil
	}

	return outcome, nil
}

func refuseWithheldFields(params map[string]any, gate *fieldGate) error {
	fields := make([]string, 0, maxListFilters+1)
	if entries, ok := params["filters"].([]any); ok {
		for _, entry := range entries {
			if object, isObject := entry.(map[string]any); isObject {
				fields = append(fields, optionalString(object, "field"))
			}
		}
	}
	fields = append(fields, optionalString(params, "sortBy"))

	for _, field := range fields {
		if field != "" && !gate.shows(field) {
			return fmt.Errorf(
				"%s is withheld at this data access, so it cannot be filtered or sorted on",
				field,
			)
		}
	}

	return nil
}

func (t *listTool) buildFilters(
	params map[string]any,
	criteria *filtercatalog.Criteria,
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

	conditions := make([]filtercatalog.Condition, 0, len(entries))
	for _, entry := range entries {
		object, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf(
				`each entry in "filters" must be an object with field and operator`,
			)
		}
		conditions = append(conditions, toCondition(object))
	}

	return t.resource.Compile(conditions, criteria)
}

// toCondition reads one JSON filter. Every operand is taken as a string
// because that is what the schema asks for and what a model produces; reading
// it is the catalog's job, so nothing here needs to know that a date is stored
// as an epoch second.
func toCondition(object map[string]any) filtercatalog.Condition {
	condition := filtercatalog.Condition{
		Field:    optionalString(object, "field"),
		Operator: dbtype.Operator(optionalString(object, "operator")),
		Value:    optionalString(object, "value"),
		Days:     optionalInt(object, "days", 0),
	}

	if raw, ok := object["values"].([]any); ok {
		condition.Values = make([]string, 0, len(raw))
		for _, item := range raw {
			// A non-string entry becomes the empty string rather than being
			// dropped, so the catalog refuses the filter by name instead of
			// silently applying a shorter one than was asked for.
			text, _ := item.(string)
			condition.Values = append(condition.Values, text)
		}
	}

	return condition
}

func (t *listTool) buildSort(params map[string]any) ([]domaintypes.SortField, error) {
	return t.resource.Sort(
		optionalString(params, "sortBy"),
		optionalString(params, "sortDirection"),
	)
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
