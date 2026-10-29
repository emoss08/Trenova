package graphqllisttool

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	OutcomeHasMore    = "hasMore"
	OutcomeNextCursor = "nextCursor"
)

var errUnpagedSpec = errors.New("an unpaged list reads by its root field's arguments alone")

// UnpagedTool is a list whose root field returns every record at
// once, keyed by its own arguments (a worker's checklists by workerId): its
// parameters are the operation's variables, sent as the screen sends them,
// and it neither pages nor filters by a record's fields.
type UnpagedTool struct {
	spec   readspec.List
	read   graphqlread.Read
	Params []graphqlread.Param
	// paged marks a root that takes a limit and an offset, which the tool
	// pages by cursor: the offset stays inside the nextCursor it hands out.
	paged       bool
	Required    []string
	beside      graphqlread.BesideReads
	reader      graphqlreadports.GraphQLReader
	permissions permissionports.PermissionEngine
}

// NewRead builds a spec's list tool in the shape its root field
// reads: paged through a table input, or unpaged by its own arguments.
func NewRead(
	spec *readspec.List,
	reader graphqlreadports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (agenttoolports.AgentQueryTool, error) {
	if read, ok := graphqlread.Reads[spec.Name]; ok && read.Unpaged {
		return NewUnpaged(spec, reader, permissions)
	}
	if len(spec.Matches) > 0 {
		return nil, fmt.Errorf("%s pages; its query is the server's, not matched in the rows",
			spec.Name)
	}

	return New(spec, reader, permissions)
}

func NewUnpaged(
	spec *readspec.List,
	reader graphqlreadports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (*UnpagedTool, error) {
	read, ok := graphqlread.Reads[spec.Name]
	if !ok {
		return nil, fmt.Errorf("%s: %w", spec.Name, graphqlread.ErrUnbound)
	}
	if read.Verb != graphqlread.VerbList || !read.Unpaged {
		return nil, fmt.Errorf("%s reads %s, which is not an unpaged list", spec.Name,
			read.Operation)
	}
	if spec.Rank != nil || spec.DefaultsAt != nil || len(spec.Instead) > 0 ||
		len(spec.Conditions) > 0 || (spec.QueryNote == "") != (len(spec.Matches) == 0) {
		return nil, fmt.Errorf("%s: %w", spec.Name, errUnpagedSpec)
	}

	params, required, err := read.ArgumentParams(spec.Name)
	if err != nil {
		return nil, err
	}
	if params, required, err = graphqlread.Renamed(spec.Name, params, spec.Names, required); err != nil {
		return nil, err
	}
	if params, err = graphqlread.Referenced(spec.Name, params, spec.References); err != nil {
		return nil, err
	}
	if params, err = graphqlread.Annotate(spec.Name, params, spec.Notes); err != nil {
		return nil, err
	}
	if spec.Amounts && slices.ContainsFunc(params, func(p graphqlread.Param) bool {
		_, minor := graphqlread.MoneyName(p.Name)
		return minor
	}) {
		return nil, fmt.Errorf("%s: %w", spec.Name, errUnpagedSpec)
	}
	for _, name := range required {
		if !slices.ContainsFunc(params, func(p graphqlread.Param) bool { return p.Name == name }) {
			return nil, fmt.Errorf("%s requires %s, which no tool supplies; name its reference",
				spec.Name, name)
		}
	}
	alongside, err := graphqlread.MatchAlongside(
		spec.Name,
		"alongside",
		spec.Alongside,
		read.Alongside,
	)
	if err != nil {
		return nil, err
	}

	return &UnpagedTool{
		spec:     *spec,
		read:     read,
		Params:   params,
		paged:    read.PagesByOffset(),
		Required: required,
		beside: graphqlread.BesideReads{
			Specs: spec.Alongside, Bound: alongside, Reader: reader, Permissions: permissions,
		},
		reader:      reader,
		permissions: permissions,
	}, nil
}

// cutShort is the note a list whose limit it filled leaves, since an unpaged
// list cannot hand out the next page.
func cutShort(args map[string]any, count int, plural string) string {
	limit := querytool.OptionalInt(args, querytool.ParamLimit, 0)
	if limit <= 0 || count < limit {
		return ""
	}

	return fmt.Sprintf("These are the first %d %s; more may exist. Narrow the call, or raise "+
		"limit to at most %d.", count, plural, querytool.UnpagedMost)
}

func (t *UnpagedTool) Name() string { return t.spec.Name }

// Description is the summary alone: the parameters are the few arguments the
// screen sends, each described in the schema, not a record's filterable
// fields.
func (t *UnpagedTool) Description() string { return t.spec.Summary }

func (t *UnpagedTool) SearchTerms() []string { return t.spec.SearchTerms }

func (t *UnpagedTool) Policy() agenttoolports.ToolPolicy {
	spec := t.spec.Reads
	spec.Resource = t.spec.Resource

	return querytool.ReadPolicy(t.Name(), spec)
}

func (t *UnpagedTool) ParamSchema() map[string]any {
	properties := make(map[string]any, len(t.Params))
	for _, param := range t.Params {
		properties[param.Name] = param.Schema
	}
	if t.paged {
		properties[querytool.ParamAfter] = querytool.CursorPageSchema(querytool.GraphQLListLimit, querytool.UnpagedMost)[querytool.ParamAfter]
	}
	if len(t.spec.Matches) > 0 {
		properties[graphqlread.ParamQuery] = map[string]any{
			toolschema.KeyType:        toolschema.TypeString,
			toolschema.KeyDescription: t.spec.QueryNote,
		}
	}
	schema := map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyAdditionalProperties: false,
	}
	if len(t.Required) > 0 {
		schema[toolschema.KeyRequired] = slices.Clone(t.Required)
	}

	return schema
}

func (t *UnpagedTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}
	clk := querytool.ClockFor(params)
	args := applyDefaults(&t.spec, params.Params, clk)
	if err := t.checkArguments(args); err != nil {
		return nil, err
	}
	variables, err := graphqlread.CompileInputs(t.Params, args, clk)
	if err != nil {
		return nil, err
	}
	for name, value := range t.spec.Inputs {
		variables[name] = value
	}
	var window querytool.OffsetWindow
	if t.paged {
		if window, err = querytool.ReadOffsetWindow(t.spec.Name, args, querytool.GraphQLListLimit,
			querytool.UnpagedMost); err != nil {
			return nil, err
		}
		variables[querytool.ParamLimit] = window.Fetch(querytool.UnpagedMost)
		variables[querytool.ParamOffset] = window.Offset
	}

	result, err := t.reader.Read(ctx, &graphqlreadports.GraphQLReadRequest{
		Actor:             params.Actor,
		Operation:         t.read.Operation,
		Variables:         variables,
		DataAccessCeiling: params.DataAccessCeiling,
		RequestID:         params.RunID.String(),
	})
	if err != nil {
		return nil, err
	}
	if err = graphqlread.ReadFailure(result, t.spec.Resource); err != nil {
		return nil, err
	}

	rows, prefix := t.rows(result.Data)
	more := t.paged && window.More(len(rows), querytool.UnpagedMost)
	if t.paged && len(rows) > window.Limit {
		rows = rows[:window.Limit]
	}
	query := strings.ToLower(
		strings.TrimSpace(querytool.OptionalString(params.Params, graphqlread.ParamQuery)),
	)
	items := make([]map[string]any, 0, len(rows))
	positions := make([]int, 0, len(rows))
	for idx, row := range rows {
		if node, ok := row.(map[string]any); ok && matchesQuery(node, t.spec.Matches, query) {
			items = append(items, graphqlread.Compact(node))
			positions = append(positions, idx)
		}
	}
	withheld := graphqlread.RelativePaths(result.Withheld, prefix)
	if t.spec.Shape != nil {
		hidden := make(map[string]struct{}, len(withheld))
		for _, path := range withheld {
			hidden[path] = struct{}{}
		}
		for _, item := range items {
			t.spec.Shape(item, hidden, clk)
		}
	}
	if t.spec.Amounts {
		for _, item := range items {
			graphqlread.PresentAmounts(item)
		}
		withheld = graphqlread.PresentedPaths(withheld)
	}

	criteria := DescribeCriteria(t.spec.EntityPlural, t.Params, args, query)
	outcome := map[string]any{
		graphqlread.OutcomeCount:       len(items),
		graphqlread.OutcomeSearchedFor: criteria.Describe(),
		graphqlread.OutcomeItems:       items,
	}
	if columns := graphqlread.SelectedColumns(t.read.Fields, items); len(columns) > 0 {
		outcome[graphqlread.OutcomeColumns] = columns
	}
	var notes []string
	if len(items) == 0 {
		notes = append(notes, criteria.EmptyNote())
	}
	switch {
	case more:
		outcome[OutcomeHasMore] = true
		outcome[OutcomeNextCursor] = window.Next()
	case !t.paged:
		if note := cutShort(args, len(items), t.spec.EntityPlural); note != "" {
			notes = append(notes, note)
		}
	}
	beside, besideWithheld, besideNotes, err := t.beside.Read(
		ctx, params, variables, namesReference(&t.spec, variables),
	)
	if err != nil {
		return nil, err
	}
	if t.spec.Combine != nil {
		t.spec.Combine(items, beside)
	}
	for key, rows := range beside {
		outcome[key] = rows
	}
	withheld = append(withheld, besideWithheld...)
	notes = append(notes, besideNotes...)
	if len(notes) > 0 {
		outcome[graphqlread.OutcomeNote] = strings.Join(notes, " ")
	}
	if len(withheld) > 0 {
		outcome[graphqlread.OutcomeWithheld] = withheld
		outcome[graphqlread.OutcomeWithheldNote] = fieldgate.WithheldNote
	}
	marked := markedRows(&t.spec, items)
	if t.paged {
		return &OffsetPage{
			Record: outcome, Outside: marked, Positions: positions, Window: window,
		}, nil
	}
	if len(marked) > 0 {
		return &graphqlread.MarkedRecord{Record: outcome, Outside: marked}, nil
	}

	return outcome, nil
}

// OffsetPage is a page of a list that pages by offset under the cursor
// contract, which a cut resumes at the offset after the last row it kept.
// positions are each item's place among the rows read, since a query the
// list matches itself keeps only some of them.
type OffsetPage struct {
	Record    map[string]any
	Outside   []agentrecord.RecordRef
	Positions []int
	Window    querytool.OffsetWindow
}

func (p *OffsetPage) MarshalJSON() ([]byte, error) { return sonic.Marshal(p.Record) }

func (p *OffsetPage) TaintedRecords() []agentrecord.RecordRef { return p.Outside }

func (p *OffsetPage) Items() []map[string]any {
	items, _ := p.Record[graphqlread.OutcomeItems].([]map[string]any)

	return items
}

func (p *OffsetPage) PageRows() int { return len(p.Items()) }

func (p *OffsetPage) CutPage(rows int) (any, bool) {
	items := p.Items()
	if rows <= 0 || rows >= len(items) || len(p.Positions) != len(items) {
		return nil, false
	}
	record := maps.Clone(p.Record)
	record[graphqlread.OutcomeItems] = items[:rows]
	record[graphqlread.OutcomeCount] = rows
	record[OutcomeHasMore] = true
	record[OutcomeNextCursor] = pagination.EncodeKeyCursor(
		p.Window.Scope, strconv.Itoa(p.Window.Offset+p.Positions[rows-1]+1))

	return &OffsetPage{
		Record:    record,
		Outside:   graphqlread.KeptRecords(p.Outside, items[:rows]),
		Positions: p.Positions[:rows],
		Window:    p.Window,
	}, true
}

// rows are the records the root returned, as a list or under items, and the
// prefix their withheld paths carry.
func (t *UnpagedTool) rows(data map[string]any) (rows []any, prefix string) {
	if !t.read.Items {
		rows, _ = data[t.read.RootField].([]any)
		return rows, t.read.RootField + "."
	}
	holder, _ := data[t.read.RootField].(map[string]any)
	rows, _ = holder[graphqlread.OutcomeItems].([]any)

	return rows, t.read.RootField + "." + graphqlread.OutcomeItems + "."
}

// matchesQuery reports whether a row holds the query's text, ignoring case,
// in any of the fields a spec matches it against; no query matches every row.
func matchesQuery(row map[string]any, paths []string, query string) bool {
	if query == "" {
		return true
	}
	for _, path := range paths {
		var value any = row
		for part := range strings.SplitSeq(path, ".") {
			object, ok := value.(map[string]any)
			if !ok {
				value = nil
				break
			}
			value = object[part]
		}
		if text, ok := value.(string); ok && strings.Contains(strings.ToLower(text), query) {
			return true
		}
	}

	return false
}

// checkArguments refuses a call that leaves out a required argument or names
// a record by something other than its id, before anything is read.
func (t *UnpagedTool) checkArguments(args map[string]any) error {
	for _, name := range t.Required {
		if raw, present := args[name]; !present || raw == nil {
			return fmt.Errorf("parameter %q is required", name)
		}
	}
	for _, param := range t.Params {
		raw, present := args[param.Name]
		if !present || raw == nil {
			continue
		}
		switch param.Kind { //nolint:exhaustive // only ids are checked here
		case graphqlread.ParamReference:
			if _, err := querytool.RequirePulid(args, param.Name); err != nil {
				return err
			}
		case graphqlread.ParamReferences:
			if _, err := querytool.PulidList(raw, param.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

// DescribeCriteria is what a call asked a list for, as its outcome reports it
// and its empty note words it.
func DescribeCriteria(
	plural string,
	params []graphqlread.Param,
	args map[string]any,
	query string,
) *filtercatalog.Criteria {
	criteria := filtercatalog.NewCriteria(plural)
	criteria.Text(query)
	for _, param := range params {
		raw, present := args[param.Name]
		if !present || raw == nil {
			continue
		}
		criteria.Field(param.Name, fmt.Sprint(raw))
	}

	return criteria
}
