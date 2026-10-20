package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	graphQLVerbList   = "list"
	graphQLParamQuery = "query"
	graphQLListLimit  = 25
	graphQLListMost   = 50

	connectionEdges    = "edges"
	connectionNode     = "node"
	connectionPageInfo = "pageInfo"
	pageHasNext        = "hasNextPage"
	pageEndCursor      = "endCursor"
	typenameField      = "__typename"

	inputFirst        = "first"
	inputAfter        = "after"
	inputQuery        = "query"
	inputFieldFilters = "fieldFilters"
	inputSort         = "sort"
	includeTotalCount = "includeTotalCount"

	// maxNearMissProbes bounds the reads an empty first page may spend asking
	// which condition to loosen; each is charged to the caller's budget.
	maxNearMissProbes = 4
)

var errGraphQLReadUnbound = errors.New("no GraphQL read is bound to this tool")

// graphQLListSpec is what a person writes for a list tool that reads through
// GraphQL: its name and the words a model reads. Everything else, from the
// operation to every parameter, is generated from the schema and the
// operation the screen sends (graphqlreads_gen.go).
type graphQLListSpec struct {
	name         string
	entityPlural string
	summary      string
	searchTerms  []string
	queryNote    string
	resource     permission.Resource
	// reads is what the tool's records carry from outside the organization,
	// declared as every read tool declares it (readSpec).
	reads readSpec
}

type graphQLListTool struct {
	spec        graphQLListSpec
	read        graphQLRead
	input       string
	params      []graphQLParam
	sortable    agenttoolschema.EnumSource[string]
	reader      agentports.GraphQLReader
	permissions permissionports.PermissionEngine
	registry    *permission.Registry
}

func newGraphQLListTool(
	spec *graphQLListSpec,
	reader agentports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (*graphQLListTool, error) {
	read, ok := graphQLReads[spec.name]
	if !ok {
		return nil, fmt.Errorf("%s: %w", spec.name, errGraphQLReadUnbound)
	}
	if read.Verb != graphQLVerbList {
		return nil, fmt.Errorf("%s reads %s, which is not a list", spec.name, read.Operation)
	}
	input := ""
	for _, variable := range read.Variables {
		if variable.Type == dataTableInput {
			input = variable.Name
		}
	}
	if input == "" {
		return nil, fmt.Errorf("%s: %s takes no %s", spec.name, read.Operation, dataTableInput)
	}
	params := read.typedParams()

	return &graphQLListTool{
		spec:        *spec,
		read:        read,
		input:       input,
		params:      params,
		sortable:    agenttoolschema.Derived(spec.name+".graphqlSortBy", read.sortable()),
		reader:      reader,
		permissions: permissions,
		registry:    fieldsensitivity.DefaultRegistry(),
	}, nil
}

func (t *graphQLListTool) Name() string { return t.spec.name }

func (t *graphQLListTool) Description() string { return t.spec.summary }

func (t *graphQLListTool) SearchTerms() []string { return t.spec.searchTerms }

func (t *graphQLListTool) Policy() agentports.ToolPolicy {
	spec := t.spec.reads
	spec.resource = t.spec.resource

	return readPolicy(t.Name(), spec)
}

func (t *graphQLListTool) ParamSchema() map[string]any {
	properties := make(map[string]any, len(t.params)+5)
	properties[graphQLParamQuery] = map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: t.queryDescription(),
	}
	for _, param := range t.params {
		properties[param.Name] = param.Schema
	}
	if len(t.sortable.Names()) > 0 {
		properties[paramSortBy] = agenttoolschema.Enum(
			"Field to order by. Defaults to most recently created.", t.sortable,
		)
		properties[paramSortDirection] = agenttoolschema.Enum(
			"asc for oldest or smallest first, desc for newest or largest.",
			listSortDirections,
		)
	}
	for name, property := range cursorPageSchema(graphQLListLimit, graphQLListMost) {
		properties[name] = property
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *graphQLListTool) queryDescription() string {
	if t.spec.queryNote != "" {
		return t.spec.queryNote
	}

	return "Optional free text matched across the searchable columns. Omit it when you are " +
		"narrowing by a field."
}

// graphQLListOutcome is one page of a list read through GraphQL, as the model
// reads it.
type graphQLListOutcome struct {
	Count            int              `json:"count"`
	SearchedFor      []string         `json:"searchedFor"`
	Items            []map[string]any `json:"items"`
	Columns          []string         `json:"columns,omitempty"`
	Note             string           `json:"note,omitempty"`
	HasMore          bool             `json:"hasMore"`
	NextCursor       string           `json:"nextCursor,omitempty"`
	WithheldByAccess []string         `json:"withheldByAccess,omitempty"`
	WithheldNote     string           `json:"withheldNote,omitempty"`
}

func (t *graphQLListTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	clk := clockFor(params)
	if err := t.refuseWithheld(ctx, params); err != nil {
		return nil, err
	}

	filters, err := compileFilters(t.params, params.Params, clk)
	if err != nil {
		return nil, err
	}
	window, err := readCursorWindow(t.Name(), params.Params, graphQLListLimit, graphQLListMost)
	if err != nil {
		return nil, err
	}
	query := strings.TrimSpace(optionalString(params.Params, graphQLParamQuery))
	criteria := t.criteria(params.Params, query)

	page, err := t.page(ctx, params, &pageRequest{
		query:   query,
		filters: filters,
		sort:    t.sort(params.Params),
		limit:   window.limit,
		after:   window.info.After,
	})
	if err != nil {
		return nil, err
	}

	outcome := graphQLListOutcome{
		Count:            len(page.items),
		SearchedFor:      criteria.Describe(),
		Items:            page.items,
		Columns:          t.columns(page.items),
		HasMore:          page.hasMore,
		WithheldByAccess: page.withheld,
	}
	if page.hasMore {
		outcome.NextCursor = window.next(page.endCursor)
	}
	if len(page.withheld) > 0 {
		outcome.WithheldNote = withheldNote
	}
	if len(page.items) == 0 && page.first {
		outcome.Note = criteria.EmptyNote()
		if near := t.nearMiss(ctx, params, query, filters); near != "" {
			outcome.Note = near
		}
	}

	return outcome, nil
}

type pageRequest struct {
	query   string
	filters []map[string]any
	sort    []map[string]any
	limit   int
	after   string
}

type graphQLPage struct {
	items     []map[string]any
	hasMore   bool
	endCursor string
	withheld  []string
	first     bool
}

func (t *graphQLListTool) page(
	ctx context.Context,
	params *agentports.QueryToolParams,
	req *pageRequest,
) (*graphQLPage, error) {
	input := map[string]any{inputFirst: req.limit}
	if req.after != "" {
		input[inputAfter] = req.after
	}
	if req.query != "" {
		input[inputQuery] = req.query
	}
	if len(req.filters) > 0 {
		input[inputFieldFilters] = req.filters
	}
	if len(req.sort) > 0 {
		input[inputSort] = req.sort
	}

	result, err := t.reader.Read(ctx, &agentports.GraphQLReadRequest{
		Actor:             params.Actor,
		Operation:         t.read.Operation,
		Variables:         map[string]any{t.input: input, includeTotalCount: false},
		DataAccessCeiling: params.DataAccessCeiling,
		RequestID:         params.RunID.String(),
	})
	if err != nil {
		return nil, err
	}
	if err = readFailure(result, t.spec.resource); err != nil {
		return nil, err
	}

	connection, _ := result.Data[t.read.RootField].(map[string]any)
	page := &graphQLPage{first: req.after == "", withheld: t.relativeWithheld(result.Withheld)}
	edges, _ := connection[connectionEdges].([]any)
	page.items = make([]map[string]any, 0, len(edges))
	for _, edge := range edges {
		object, _ := edge.(map[string]any)
		if node, ok := object[connectionNode].(map[string]any); ok {
			page.items = append(page.items, compact(node))
		}
	}
	if info, ok := connection[connectionPageInfo].(map[string]any); ok {
		page.hasMore, _ = info[pageHasNext].(bool)
		page.endCursor, _ = info[pageEndCursor].(string)
	}

	return page, nil
}

// readFailure turns the operation's errors into the tool's: a refusal the
// screen would have shown, a spent budget, or a read that failed.
func readFailure(result *agentports.GraphQLReadResult, resource permission.Resource) error {
	if len(result.Errors) == 0 {
		return nil
	}
	first := result.Errors[0]
	switch first.Code {
	case agentports.GraphQLCodeForbidden, agentports.GraphQLCodeUnauthorized:
		return &agentports.ReadRefusedError{Resource: resource, Message: first.Message}
	case agentports.GraphQLCodeBudgetExceeded:
		return fmt.Errorf("the read budget for this person is spent; try again in %s",
			result.RetryAfter.Round(1e9))
	}

	return fmt.Errorf("the read failed: %s", first.Message)
}

// relativeWithheld names withheld fields from the record, as a model reads
// them: customers.edges.node.billingProfile.creditLimit is
// billingProfile.creditLimit.
func (t *graphQLListTool) relativeWithheld(paths []string) []string {
	prefix := t.read.RootField + "." + connectionEdges + "." + connectionNode + "."
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, strings.TrimPrefix(path, prefix))
	}

	return out
}

// compact drops what tells a model nothing: nulls, empty lists and type
// names.
func compact(node map[string]any) map[string]any {
	out := make(map[string]any, len(node))
	for key, value := range node {
		if key == typenameField || value == nil {
			continue
		}
		switch v := value.(type) {
		case []any:
			if len(v) == 0 {
				continue
			}
		case map[string]any:
			value = compact(v)
		}
		out[key] = value
	}

	return out
}

// columns are the fields the operation selects on the record, in schema
// order, that the page's rows carry.
func (t *graphQLListTool) columns(items []map[string]any) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(t.read.Fields))
	for _, field := range t.read.Fields {
		if !field.Selected {
			continue
		}
		for _, item := range items {
			if _, ok := item[field.Name]; ok {
				out = append(out, field.Name)
				break
			}
		}
	}

	return out
}

func (t *graphQLListTool) sort(args map[string]any) []map[string]any {
	field := optionalString(args, paramSortBy)
	if field == "" || !slices.Contains(t.sortable.Names(), field) {
		return nil
	}
	direction := optionalString(args, paramSortDirection)
	if direction == "" {
		direction = string(dbtype.SortDirectionAsc)
	}

	return []map[string]any{{paramField: field, paramDirection: direction}}
}

func (t *graphQLListTool) criteria(args map[string]any, query string) *filtercatalog.Criteria {
	criteria := filtercatalog.NewCriteria(t.spec.entityPlural)
	criteria.Text(query)
	for _, param := range t.params {
		raw, present := args[param.Name]
		if !present || raw == nil {
			continue
		}
		criteria.Field(param.Name, fmt.Sprint(raw))
	}

	return criteria
}

// refuseWithheld refuses a filter or an order on a field the caller may not
// be shown, before anything is read: filtering on a hidden value would leak
// it one row at a time.
func (t *graphQLListTool) refuseWithheld(
	ctx context.Context,
	params *agentports.QueryToolParams,
) error {
	fields := make([]string, 0, len(t.params)+1)
	for _, param := range t.params {
		if raw, present := params.Params[param.Name]; present && raw != nil {
			fields = append(fields, param.Field)
		}
	}
	if sortBy := optionalString(params.Params, paramSortBy); sortBy != "" {
		fields = append(fields, sortBy)
	}
	if len(fields) == 0 {
		return nil
	}

	ceiling := fieldsensitivity.AgentCeiling(
		ctx, t.permissions, params.Actor, params.DataAccessCeiling, t.spec.resource,
	)
	for _, field := range fields {
		if !fieldsensitivity.Visible(t.registry, t.spec.resource, field, ceiling) {
			return fmt.Errorf(
				"%s cannot be filtered or ordered by %s: its values are hidden at your access",
				t.spec.name, field)
		}
	}

	return nil
}

// nearMiss reads again with one condition dropped at a time, and says which
// one, alone, kept every row out.
func (t *graphQLListTool) nearMiss(
	ctx context.Context,
	params *agentports.QueryToolParams,
	query string,
	filters []map[string]any,
) string {
	probes := 0
	if query != "" {
		probes++
		if page, err := t.page(ctx, params, &pageRequest{filters: filters, limit: 1}); err == nil &&
			len(page.items) > 0 {
			return fmt.Sprintf("Nothing matched the text %q, but %s do match the other "+
				"conditions. Try different words, or drop the text.", query, t.spec.entityPlural)
		}
	}
	for idx := range filters {
		if probes >= maxNearMissProbes {
			break
		}
		probes++
		rest := slices.Delete(slices.Clone(filters), idx, idx+1)
		page, err := t.page(ctx, params, &pageRequest{query: query, filters: rest, limit: 1})
		if err == nil && len(page.items) > 0 {
			return fmt.Sprintf("No %s matched every condition. Without the condition on %v "+
				"some do, so that one alone keeps them out.", t.spec.entityPlural,
				filters[idx][paramField])
		}
	}

	return ""
}

// runView reads a composed table view through the same operation, so the
// count and rows a view previews are what listing the table shows the same
// person.
func (t *graphQLListTool) runView(
	ctx context.Context,
	params *agentports.QueryToolParams,
	query string,
	filters []domaintypes.FieldFilter,
	sorting []domaintypes.SortField,
	limit int,
) (*viewRun, error) {
	req := &pageRequest{query: query, limit: limit}
	for _, filter := range filters {
		req.filters = append(req.filters, map[string]any{
			paramField:     filter.Field,
			filterOperator: string(filter.Operator),
			filterValue:    filter.Value,
		})
	}
	for _, field := range sorting {
		req.sort = append(req.sort, map[string]any{
			paramField: field.Field, paramDirection: string(field.Direction),
		})
	}

	page, err := t.page(ctx, params, req)
	if err != nil {
		return nil, err
	}
	criteria := filtercatalog.NewCriteria(t.spec.entityPlural)
	criteria.Text(query)
	items := make([]any, 0, len(page.items))
	for _, item := range page.items {
		items = append(items, item)
	}

	return &viewRun{
		outcome: searchOutcome{
			Count:       len(items),
			SearchedFor: criteria.Describe(),
			Items:       items,
			Columns:     t.columns(page.items),
		},
		withheld: page.withheld,
		more:     page.hasMore,
	}, nil
}
