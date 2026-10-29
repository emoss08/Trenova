package listtool

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	ParamFilters       = "filters"
	ParamOperator      = "operator"
	paramValue         = "value"
	ParamValues        = "values"
	paramDays          = "days"
	ParamSortBy        = "sortBy"
	ParamSortDirection = "sortDirection"
	ParamProNumber     = "proNumber"
)

const (
	DefaultLimit = 25
	MaxLimit     = 50
	// maxListFilters bounds how much a single call can ask for. Six is more
	// than any question a person phrases in one sentence, and the query builder
	// truncates past its own cap silently, which would filter on less than the
	// model was told it filtered on.
	maxListFilters  = 6
	maxRelativeDays = 3650
)

// Spec is one entry in the catalog: an entity, the fields a model may
// narrow it by, and the repository call that serves it.
//
// One tool is generated per entry rather than a single list_records(entity, …)
// tool, because a tool's Policy names exactly one permission resource
// and the agent builder groups the picker by it. A single tool would have to
// resolve its resource at call time, and an organization could not grant an
// agent "list workers" without also granting "list customers".
type Spec struct {
	Name         string
	EntityPlural string
	Summary      string
	SearchTerms  []string
	QueryNote    string
	Resource     permission.Resource
	Fields       []filtercatalog.Field
	// row is a zero value of the row the list returns, whose columns a call
	// may name in fields for what the model reads. A list without one takes
	// no fields.
	Row any
	// config is the entity's computed field configuration. A curated field that
	// no longer maps to a column is dropped silently by the query builder, so it
	// is checked here and refused loudly instead.
	Config        *domaintypes.FieldConfiguration
	PagesByCursor bool
	Fetch         func(
		ctx context.Context,
		opts *pagination.QueryOptions,
		cursor pagination.CursorInfo,
	) (Page, error)
	Count func(ctx context.Context, opts *pagination.QueryOptions) (int, bool, error)
	// fetchIn is fetch with the caller's clock, for a row that renders a
	// time of day and needs the organization's zone to do it. One of the
	// two is set.
	fetchIn func(
		ctx context.Context,
		opts *pagination.QueryOptions,
		cursor pagination.CursorInfo,
		clk filtercatalog.Clock,
	) (Page, error)

	Access     fieldgate.Access
	FetchGated func(
		ctx context.Context,
		opts *pagination.QueryOptions,
		cursor pagination.CursorInfo,
		gate *fieldgate.Gate,
	) (Page, error)
	GatedField  func(field string) string
	FetchJoined func(
		ctx context.Context,
		opts *pagination.QueryOptions,
		cursor pagination.CursorInfo,
		mayRead func(permission.Resource) bool,
	) (Page, error)
}

type Page struct {
	Rows       []any
	hasMore    bool
	nextCursor string
	// cursors are each row's edge cursor, in order, when the page has them.
	cursors  []string
	cursored bool
	fuzzy    bool
}

type Tool struct {
	Spec        Spec
	resource    filtercatalog.Resource
	Fields      agenttoolschema.EnumSource[string]
	Operators   agenttoolschema.EnumSource[string]
	sortable    agenttoolschema.EnumSource[string]
	Columns     agenttoolschema.EnumSource[string]
	description string
}

func New(spec Spec) agenttoolports.AgentQueryTool {
	return Build(&spec)
}

func Build(spec *Spec) *Tool {
	seen := make(map[dbtype.Operator]bool, len(spec.Fields)*4)
	operators := make([]string, 0, len(spec.Fields)*4)
	for _, field := range spec.Fields {
		for _, operator := range filtercatalog.Operators(field.Kind) {
			if seen[operator] {
				continue
			}
			seen[operator] = true
			operators = append(operators, string(operator))
		}
	}

	resource := CatalogResource(*spec)

	return &Tool{
		Spec:        *spec,
		resource:    resource,
		Fields:      agenttoolschema.Derived(spec.Name+".field", resource.FieldNames()),
		Operators:   agenttoolschema.Derived(spec.Name+".operator", operators),
		sortable:    agenttoolschema.Derived(spec.Name+".sortBy", resource.SortableNames()),
		Columns:     agenttoolschema.Derived(spec.Name+".column", rowColumns(spec.Row)),
		description: buildListDescription(*spec),
	}
}

// rowColumns are the columns of a list's row but the id, which every row
// keeps: what a call may name in fields.
func rowColumns(row any) []string {
	if row == nil {
		return nil
	}

	return slices.DeleteFunc(querytool.ColumnsOf([]any{row}), func(name string) bool {
		return name == querytool.RecordIDField
	})
}

// CatalogResource is the one place a list spec becomes a catalogued resource,
// so the tool and the table composer narrow the same entity by the same
// fields.
func CatalogResource(spec Spec) filtercatalog.Resource {
	return filtercatalog.Resource{
		Tool:     spec.Name,
		Resource: spec.Resource,
		Entity:   spec.EntityPlural,
		Summary:  spec.Summary,
		Fields:   spec.Fields,
		Config:   spec.Config,
	}.Prepare()
}

// DateRule is said once per tool rather than per date field: the server
// resolves every relative window, so a model never does calendar arithmetic.
const DateRule = " Dates: today, nextndays/lastndays with days, or YYYY-MM-DD; " +
	"never Unix time."

// buildListDescription renders the summary and every filterable field as
// compactly as a model still reads them. A field with values or a note is
// spelled out on its own; the rest are only names, grouped under their kind.
func buildListDescription(spec Spec) string {
	var b strings.Builder
	b.WriteString(spec.Summary)
	b.WriteString(" Filters: ")

	written := 0
	separate := func() {
		if written > 0 {
			b.WriteString("; ")
		}
		written++
	}

	hasDate := false
	kinds := make([]filtercatalog.Kind, 0, len(spec.Fields))
	plain := make(map[filtercatalog.Kind][]string, len(spec.Fields))
	for _, field := range spec.Fields {
		hasDate = hasDate || field.Kind == filtercatalog.KindDate
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
		b.WriteString(DateRule)
	}

	return b.String()
}

func (t *Tool) Name() string { return t.Spec.Name }

func (t *Tool) Description() string { return t.description }

func (t *Tool) SearchTerms() []string { return t.Spec.SearchTerms }

func (t *Tool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), querytool.ReadSpec{
		Resource: t.Spec.Resource,
	})
}

func (t *Tool) ParamSchema() map[string]any {
	properties := map[string]any{
		querytool.ParamQuery: map[string]any{
			toolschema.KeyType:        toolschema.TypeString,
			toolschema.KeyDescription: t.queryDescription(),
		},
		ParamFilters:       FiltersSchema(t.Fields, t.Operators),
		ParamSortBy:        t.SortSchema(),
		ParamSortDirection: SortDirectionSchema(),
	}
	for name, property := range t.pagingSchema() {
		properties[name] = property
	}
	if len(t.Columns.Names()) > 1 {
		properties[agenttoolports.ListFieldsParam] = FieldsSchema(t.Columns)
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *Tool) pagingSchema() map[string]any {
	if t.Spec.PagesByCursor {
		return querytool.CursorPageSchema(DefaultLimit, MaxLimit)
	}

	return querytool.PageSchema(DefaultLimit, MaxLimit)
}

func (t *Tool) queryDescription() string {
	if t.Spec.QueryNote != "" {
		return t.Spec.QueryNote
	}

	return "Optional free text matched across the searchable columns. Omit it when you are " +
		"filtering on a field."
}

func FiltersSchema(
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
				querytool.ParamField: agenttoolschema.Enum("", fields),
				ParamOperator:        agenttoolschema.Enum("", operators),
				paramValue:           map[string]any{toolschema.KeyType: toolschema.TypeString},
				ParamValues: map[string]any{
					toolschema.KeyType:  toolschema.TypeArray,
					toolschema.KeyItems: map[string]any{toolschema.KeyType: toolschema.TypeString},
				},
				paramDays: map[string]any{
					toolschema.KeyType:    toolschema.TypeInteger,
					toolschema.KeyMinimum: 1,
					toolschema.KeyMaximum: maxRelativeDays,
				},
			},
			toolschema.KeyRequired:             []string{querytool.ParamField, ParamOperator},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func (t *Tool) SortSchema() map[string]any {
	return agenttoolschema.Enum("Field to order by. Defaults to most recently created.", t.sortable)
}

func SortDirectionSchema() map[string]any {
	return agenttoolschema.Enum(
		"asc for oldest or smallest first, desc for newest or largest.",
		SortDirections,
	)
}

func (t *Tool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	var gate *fieldgate.Gate
	if t.Spec.FetchGated != nil {
		gate = t.Spec.Access.Gate(ctx, params, t.Spec.Resource)
		if err := refuseWithheldFields(params.Params, gate, t.Spec.GatedField); err != nil {
			return nil, err
		}
	}

	criteria := filtercatalog.NewCriteria(t.Spec.EntityPlural).At(querytool.ClockFor(params))

	query := querytool.OptionalString(params.Params, "query")
	criteria.Text(query)

	filters, err := t.BuildFilters(params.Params, criteria)
	if err != nil {
		return nil, err
	}

	sorting, err := t.BuildSort(params.Params)
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
		cursor:   window.cursor.Info,
		gate:     gate,
		criteria: criteria,
	})
	if err != nil {
		return nil, err
	}

	outcome, more := window.outcome(criteria, fetched)
	if fetched.fuzzy && outcome.Count > 0 {
		outcome.Note = closeSpellingNote(t.Spec.EntityPlural, query)
	}
	if more && t.Spec.Count != nil {
		if total, exact, countErr := t.Spec.Count(ctx, opts); countErr == nil {
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
		return fieldgate.Result(&outcome, gate), nil
	}

	return outcome, nil
}

type listWindow struct {
	offset   querytool.Page
	cursor   querytool.CursorWindow
	byCursor bool
}

func (t *Tool) readWindow(params map[string]any) (listWindow, error) {
	if !t.Spec.PagesByCursor {
		return listWindow{offset: querytool.ReadPage(params, DefaultLimit, MaxLimit)}, nil
	}

	cursor, err := querytool.ReadCursorWindow(t.Name(), params, DefaultLimit, MaxLimit)
	if err != nil {
		return listWindow{}, err
	}

	return listWindow{cursor: cursor, byCursor: true}, nil
}

func (t *Tool) firstWindow(limit int) listWindow {
	if t.Spec.PagesByCursor {
		return listWindow{cursor: querytool.FirstCursorWindow(limit), byCursor: true}
	}

	return listWindow{offset: querytool.Page{Limit: limit}}
}

func (w *listWindow) pagination() pagination.Info {
	if w.byCursor {
		return pagination.Info{Limit: w.cursor.Limit}
	}

	return pagination.Info{Limit: w.offset.Fetch(), Offset: w.offset.Offset}
}

func (w *listWindow) first() bool {
	return w.offset.Offset == 0 && w.cursor.First()
}

func (w *listWindow) rows(fetched Page) ([]any, bool) {
	if w.byCursor {
		return fetched.Rows, fetched.hasMore
	}

	return querytool.Trim(w.offset, fetched.Rows)
}

func (w *listWindow) outcome(
	criteria *filtercatalog.Criteria,
	fetched Page,
) (querytool.SearchOutcome, bool) {
	rows, more := w.rows(fetched)
	outcome := querytool.SearchResult(criteria, rows, len(rows))
	if w.byCursor {
		outcome.CursorPaged(&w.cursor, more, fetched.nextCursor)
		outcome.Resume = &querytool.PageResume{Scope: w.cursor.Scope, Cursors: fetched.cursors}

		return outcome, more
	}
	outcome = outcome.Paged(w.offset, more)
	outcome.Resume = &querytool.PageResume{ByOffset: true}

	return outcome, more
}

type listFetch struct {
	params   *agenttoolports.QueryToolParams
	opts     *pagination.QueryOptions
	cursor   pagination.CursorInfo
	gate     *fieldgate.Gate
	criteria *filtercatalog.Criteria
}

// fetchRows reads one window of rows the way the list's spec says to: through
// the field gate, joined with what the reader may see, on the caller's clock,
// or plainly.
func (t *Tool) fetchRows(ctx context.Context, req *listFetch) (Page, error) {
	fetched, err := t.fetchPage(ctx, req)
	if err != nil {
		return Page{}, querytool.RefuseStaleCursor(err)
	}
	if fetched.cursored != t.Spec.PagesByCursor {
		return Page{}, fmt.Errorf(
			"%s pages its repository the wrong way: its spec and its fetch disagree on "+
				"cursor paging",
			t.Spec.Name,
		)
	}

	return fetched, nil
}

func (t *Tool) fetchPage(ctx context.Context, req *listFetch) (Page, error) {
	switch {
	case req.gate != nil:
		return t.Spec.FetchGated(ctx, req.opts, req.cursor, req.gate)
	case t.Spec.FetchJoined != nil:
		return t.Spec.FetchJoined(
			ctx,
			req.opts,
			req.cursor,
			func(resource permission.Resource) bool {
				return t.Spec.Access.MayRead(ctx, req.params, resource)
			},
		)
	case t.Spec.fetchIn != nil:
		return t.Spec.fetchIn(ctx, req.opts, req.cursor, req.criteria.Clock)
	default:
		return t.Spec.Fetch(ctx, req.opts, req.cursor)
	}
}

// runView reads a composed view through this list's own fetch, so the count
// and the rows a view previews are what its page shows the same person: the
// same filters, the same tenant and the same withheld fields. It reads one
// row past limit to say whether there are more.
func (t *Tool) RunView(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	query string,
	filters []domaintypes.FieldFilter,
	sorting []domaintypes.SortField,
	limit int,
) (*querytool.ViewRun, error) {
	var gate *fieldgate.Gate
	if t.Spec.FetchGated != nil {
		gate = t.Spec.Access.Gate(ctx, params, t.Spec.Resource)
	}
	criteria := filtercatalog.NewCriteria(t.Spec.EntityPlural).At(querytool.ClockFor(params))
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
		cursor:   window.cursor.Info,
		gate:     gate,
		criteria: criteria,
	})
	if err != nil {
		return nil, err
	}

	rows, more := window.rows(fetched)
	run := &querytool.ViewRun{
		Outcome: querytool.SearchResult(criteria, rows, len(rows)),
		More:    more,
	}
	if gate != nil {
		run.Withheld = gate.Withheld()
	}

	return run, nil
}

func refuseWithheldFields(
	params map[string]any,
	gate *fieldgate.Gate,
	gatedField func(string) string,
) error {
	fields := make([]string, 0, maxListFilters+1)
	if entries, ok := params["filters"].([]any); ok {
		for _, entry := range entries {
			if object, isObject := entry.(map[string]any); isObject {
				fields = append(fields, querytool.OptionalString(object, "field"))
			}
		}
	}
	fields = append(fields, querytool.OptionalString(params, "sortBy"))

	for _, field := range fields {
		if field == "" {
			continue
		}
		judged := field
		if gatedField != nil {
			judged = gatedField(field)
		}
		if !gate.Shows(judged) {
			return fmt.Errorf(
				"%s is withheld at this data access, so it cannot be filtered or sorted on",
				field,
			)
		}
	}

	return nil
}

func (t *Tool) BuildFilters(
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
		Field:    querytool.OptionalString(object, "field"),
		Operator: dbtype.Operator(querytool.OptionalString(object, "operator")),
		Value:    querytool.OptionalString(object, "value"),
		Days:     querytool.OptionalInt(object, "days", 0),
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

func (t *Tool) BuildSort(params map[string]any) ([]domaintypes.SortField, error) {
	return t.resource.Sort(
		querytool.OptionalString(params, "sortBy"),
		querytool.OptionalString(params, "sortDirection"),
	)
}

func CursorPage[T any](
	result *pagination.CursorListResult[T],
	row func(T) any,
) (Page, error) {
	next, err := result.NextCursor()
	if err != nil {
		return Page{}, fmt.Errorf("encode the next page's cursor: %w", err)
	}

	return Page{
		Rows:       listRows(result.Items, row),
		hasMore:    result.HasNextPage,
		nextCursor: next,
		cursors:    result.EdgeCursors,
		cursored:   true,
		fuzzy:      result.Fuzzy,
	}, nil
}

func OffsetPage[T any](items []T, row func(T) any) Page {
	return Page{Rows: listRows(items, row)}
}

func ResultPage[T any](result *pagination.ListResult[T], row func(T) any) Page {
	return Page{Rows: listRows(result.Items, row), fuzzy: result.Fuzzy}
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
