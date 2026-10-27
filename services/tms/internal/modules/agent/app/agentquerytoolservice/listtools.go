package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	paramFilters       = "filters"
	paramOperator      = "operator"
	paramValue         = "value"
	paramValues        = "values"
	paramDays          = "days"
	paramSortBy        = "sortBy"
	paramSortDirection = "sortDirection"
	paramProNumber     = "proNumber"
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
	searchTerms  []string
	queryNote    string
	resource     permission.Resource
	fields       []listField
	// config is the entity's computed field configuration. A curated field that
	// no longer maps to a column is dropped silently by the query builder, so it
	// is checked here and refused loudly instead.
	config        *domaintypes.FieldConfiguration
	pagesByCursor bool
	fetch         func(
		ctx context.Context,
		opts *pagination.QueryOptions,
		cursor pagination.CursorInfo,
	) (listPage, error)
	count func(ctx context.Context, opts *pagination.QueryOptions) (int, bool, error)
	// fetchIn is fetch with the caller's clock, for a row that renders a
	// time of day and needs the organization's zone to do it. One of the
	// two is set.
	fetchIn func(
		ctx context.Context,
		opts *pagination.QueryOptions,
		cursor pagination.CursorInfo,
		clk clock,
	) (listPage, error)

	access     fieldAccess
	fetchGated func(
		ctx context.Context,
		opts *pagination.QueryOptions,
		cursor pagination.CursorInfo,
		gate *fieldGate,
	) (listPage, error)
	gatedField  func(field string) string
	fetchJoined func(
		ctx context.Context,
		opts *pagination.QueryOptions,
		cursor pagination.CursorInfo,
		mayRead func(permission.Resource) bool,
	) (listPage, error)
}

type listPage struct {
	rows       []any
	hasMore    bool
	nextCursor string
	cursored   bool
	fuzzy      bool
}

type listTool struct {
	spec        listSpec
	resource    filtercatalog.Resource
	fields      agenttoolschema.EnumSource[string]
	operators   agenttoolschema.EnumSource[string]
	sortable    agenttoolschema.EnumSource[string]
	description string
}

func newListTool(spec listSpec) agentports.AgentQueryTool {
	return buildListTool(&spec)
}

func buildListTool(spec *listSpec) *listTool {
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

	resource := catalogResource(*spec)

	return &listTool{
		spec:        *spec,
		resource:    resource,
		fields:      agenttoolschema.Derived(spec.name+".field", resource.FieldNames()),
		operators:   agenttoolschema.Derived(spec.name+".operator", operators),
		sortable:    agenttoolschema.Derived(spec.name+".sortBy", resource.SortableNames()),
		description: buildListDescription(*spec),
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
		if len(field.Values) > 0 && !field.ValuesUnlisted {
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

func (t *listTool) SearchTerms() []string { return t.spec.searchTerms }

func (t *listTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: t.spec.resource,
	})
}

func (t *listTool) ParamSchema() map[string]any {
	properties := map[string]any{
		paramQuery: map[string]any{
			toolschema.KeyType:        toolschema.TypeString,
			toolschema.KeyDescription: t.queryDescription(),
		},
		paramFilters:       filtersSchema(t.fields, t.operators),
		paramSortBy:        t.sortSchema(),
		paramSortDirection: sortDirectionSchema(),
	}
	for name, property := range t.pagingSchema() {
		properties[name] = property
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *listTool) pagingSchema() map[string]any {
	if t.spec.pagesByCursor {
		return cursorPageSchema(defaultListLimit, maxListLimit)
	}

	return pageSchema(defaultListLimit, maxListLimit)
}

func (t *listTool) queryDescription() string {
	if t.spec.queryNote != "" {
		return t.spec.queryNote
	}

	return "Optional free text matched across the searchable columns. Omit it when you are " +
		"filtering on a field."
}

func filtersSchema(
	fields agenttoolschema.EnumSource[string],
	operators agenttoolschema.EnumSource[string],
) map[string]any {
	return map[string]any{
		toolschema.KeyType:     toolschema.TypeArray,
		toolschema.KeyMaxItems: maxListFilters,
		toolschema.KeyDescription: "Conditions, combined with AND. Give value for a single " +
			"comparison, values for in and notin, days for nextndays and lastndays, " +
			"and nothing at all for isnull, isnotnull, today, yesterday and tomorrow.",
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				paramField:    agenttoolschema.Enum("", fields),
				paramOperator: agenttoolschema.Enum("", operators),
				paramValue:    map[string]any{toolschema.KeyType: toolschema.TypeString},
				paramValues: map[string]any{
					toolschema.KeyType:  toolschema.TypeArray,
					toolschema.KeyItems: map[string]any{toolschema.KeyType: toolschema.TypeString},
				},
				paramDays: map[string]any{
					toolschema.KeyType:    toolschema.TypeInteger,
					toolschema.KeyMinimum: 1,
					toolschema.KeyMaximum: maxRelativeDays,
				},
			},
			toolschema.KeyRequired:             []string{paramField, paramOperator},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func (t *listTool) sortSchema() map[string]any {
	return agenttoolschema.Enum("Field to order by. Defaults to most recently created.", t.sortable)
}

func sortDirectionSchema() map[string]any {
	return agenttoolschema.Enum(
		"asc for oldest or smallest first, desc for newest or largest.",
		listSortDirections,
	)
}

func (t *listTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	var gate *fieldGate
	if t.spec.fetchGated != nil {
		gate = t.spec.access.gate(ctx, params, t.spec.resource)
		if err := refuseWithheldFields(params.Params, gate, t.spec.gatedField); err != nil {
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

	window, err := t.readWindow(params.Params)
	if err != nil {
		return nil, err
	}

	opts := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		Pagination:   window.pagination(),
		Query:        query,
		FieldFilters: filters,
		Sort:         sorting,
	}

	fetched, err := t.fetchRows(ctx, &listFetch{
		params:   params,
		opts:     opts,
		cursor:   window.cursor.info,
		gate:     gate,
		criteria: criteria,
	})
	if err != nil {
		return nil, err
	}

	outcome, more := window.outcome(criteria, fetched)
	if fetched.fuzzy && outcome.Count > 0 {
		outcome.Note = closeSpellingNote(t.spec.entityPlural, query)
	}
	if more && t.spec.count != nil {
		if total, exact, countErr := t.spec.count(ctx, opts); countErr == nil {
			outcome.Total = &total
			outcome.TotalEstimated = !exact
		}
	}
	if outcome.Count == 0 && window.first() {
		if note := t.nearMiss(ctx, &nearMissRequest{
			params:   params,
			opts:     opts,
			gate:     gate,
			criteria: criteria,
			filters:  filters,
		}); note != "" {
			outcome.Note = note
		}
	}
	if gate != nil {
		return gatedResult(&outcome, gate), nil
	}

	return outcome, nil
}

type listWindow struct {
	offset   page
	cursor   cursorWindow
	byCursor bool
}

func (t *listTool) readWindow(params map[string]any) (listWindow, error) {
	if !t.spec.pagesByCursor {
		return listWindow{offset: readPage(params, defaultListLimit, maxListLimit)}, nil
	}

	cursor, err := readCursorWindow(t.Name(), params, defaultListLimit, maxListLimit)
	if err != nil {
		return listWindow{}, err
	}

	return listWindow{cursor: cursor, byCursor: true}, nil
}

func (t *listTool) firstWindow(limit int) listWindow {
	if t.spec.pagesByCursor {
		return listWindow{cursor: firstCursorWindow(limit), byCursor: true}
	}

	return listWindow{offset: page{limit: limit}}
}

func (w *listWindow) pagination() pagination.Info {
	if w.byCursor {
		return pagination.Info{Limit: w.cursor.limit}
	}

	return pagination.Info{Limit: w.offset.fetch(), Offset: w.offset.offset}
}

func (w *listWindow) first() bool {
	return w.offset.offset == 0 && w.cursor.first()
}

func (w *listWindow) rows(fetched listPage) ([]any, bool) {
	if w.byCursor {
		return fetched.rows, fetched.hasMore
	}

	return trim(w.offset, fetched.rows)
}

func (w *listWindow) outcome(
	criteria *filtercatalog.Criteria,
	fetched listPage,
) (searchOutcome, bool) {
	rows, more := w.rows(fetched)
	outcome := searchResult(criteria, rows, len(rows))
	if w.byCursor {
		outcome.cursorPaged(&w.cursor, more, fetched.nextCursor)

		return outcome, more
	}

	return outcome.paged(w.offset, more), more
}

type listFetch struct {
	params   *agentports.QueryToolParams
	opts     *pagination.QueryOptions
	cursor   pagination.CursorInfo
	gate     *fieldGate
	criteria *filtercatalog.Criteria
}

// fetchRows reads one window of rows the way the list's spec says to: through
// the field gate, joined with what the reader may see, on the caller's clock,
// or plainly.
func (t *listTool) fetchRows(ctx context.Context, req *listFetch) (listPage, error) {
	fetched, err := t.fetchPage(ctx, req)
	if err != nil {
		return listPage{}, refuseStaleCursor(err)
	}
	if fetched.cursored != t.spec.pagesByCursor {
		return listPage{}, fmt.Errorf(
			"%s pages its repository the wrong way: its spec and its fetch disagree on "+
				"cursor paging",
			t.spec.name,
		)
	}

	return fetched, nil
}

func (t *listTool) fetchPage(ctx context.Context, req *listFetch) (listPage, error) {
	switch {
	case req.gate != nil:
		return t.spec.fetchGated(ctx, req.opts, req.cursor, req.gate)
	case t.spec.fetchJoined != nil:
		return t.spec.fetchJoined(
			ctx,
			req.opts,
			req.cursor,
			func(resource permission.Resource) bool {
				return t.spec.access.mayRead(ctx, req.params, resource)
			},
		)
	case t.spec.fetchIn != nil:
		return t.spec.fetchIn(ctx, req.opts, req.cursor, req.criteria.Clock)
	default:
		return t.spec.fetch(ctx, req.opts, req.cursor)
	}
}

// viewRun is what a composed view holds when it is run: up to its counting
// limit of rows in the list's own shape, and whether there were more.
type viewRun struct {
	outcome  searchOutcome
	withheld []string
	more     bool
}

// runView reads a composed view through this list's own fetch, so the count
// and the rows a view previews are what its page shows the same person: the
// same filters, the same tenant and the same withheld fields. It reads one
// row past limit to say whether there are more.
func (t *listTool) runView(
	ctx context.Context,
	params *agentports.QueryToolParams,
	query string,
	filters []domaintypes.FieldFilter,
	sorting []domaintypes.SortField,
	limit int,
) (*viewRun, error) {
	var gate *fieldGate
	if t.spec.fetchGated != nil {
		gate = t.spec.access.gate(ctx, params, t.spec.resource)
	}
	criteria := filtercatalog.NewCriteria(t.spec.entityPlural).At(clockFor(params))
	criteria.Text(query)

	window := t.firstWindow(limit)
	fetched, err := t.fetchRows(ctx, &listFetch{
		params: params,
		opts: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID:  params.OrganizationID,
				BuID:   params.BusinessUnitID,
				UserID: params.Actor.UserID,
			},
			Pagination:   window.pagination(),
			Query:        query,
			FieldFilters: filters,
			Sort:         sorting,
		},
		cursor:   window.cursor.info,
		gate:     gate,
		criteria: criteria,
	})
	if err != nil {
		return nil, err
	}

	rows, more := window.rows(fetched)
	run := &viewRun{outcome: searchResult(criteria, rows, len(rows)), more: more}
	if gate != nil {
		run.withheld = gate.Withheld()
	}

	return run, nil
}

func refuseWithheldFields(
	params map[string]any,
	gate *fieldGate,
	gatedField func(string) string,
) error {
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
		if field == "" {
			continue
		}
		judged := field
		if gatedField != nil {
			judged = gatedField(field)
		}
		if !gate.shows(judged) {
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

func cursorListPage[T any](
	result *pagination.CursorListResult[T],
	row func(T) any,
) (listPage, error) {
	next, err := result.NextCursor()
	if err != nil {
		return listPage{}, fmt.Errorf("encode the next page's cursor: %w", err)
	}

	return listPage{
		rows:       listRows(result.Items, row),
		hasMore:    result.HasNextPage,
		nextCursor: next,
		cursored:   true,
		fuzzy:      result.Fuzzy,
	}, nil
}

func offsetListPage[T any](items []T, row func(T) any) listPage {
	return listPage{rows: listRows(items, row)}
}

func offsetResultPage[T any](result *pagination.ListResult[T], row func(T) any) listPage {
	return listPage{rows: listRows(result.Items, row), fuzzy: result.Fuzzy}
}

func closeSpellingNote(entityPlural, query string) string {
	return fmt.Sprintf(
		"No %s matched %q as written; these matched as a close spelling of it. Say which one "+
			"you took it to mean, or ask, before acting on one.",
		entityPlural,
		query,
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
