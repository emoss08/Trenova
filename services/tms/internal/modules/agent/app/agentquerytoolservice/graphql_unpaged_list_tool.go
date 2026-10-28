package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	// alongsideMost is how many records a read made alongside a list returns:
	// the server's most for one page, which the catalogs the repository tools
	// read beside a worker's records (checklist and review templates) never
	// approached.
	alongsideMost = 100
	// unpagedMost is the most rows a limit on an unpaged list may ask for: the
	// server's page bound, since such a list has no next page to hand out.
	unpagedMost = 100

	outcomeCount        = "count"
	outcomeSearchedFor  = "searchedFor"
	outcomeItems        = "items"
	outcomeColumns      = "columns"
	outcomeNote         = "note"
	outcomeWithheld     = "withheldByAccess"
	outcomeWithheldNote = "withheldNote"
	outcomeHasMore      = "hasMore"
	outcomeNextCursor   = "nextCursor"
)

var errUnpagedSpec = errors.New("an unpaged list reads by its root field's arguments alone")

// alongsideSpec is a read an unpaged list makes beside its own, for records
// of another kind its next call needs (the templates a checklist can be
// started from, a worker's recognitions), carried in the result under key.
// It is made only when the reader may read that kind, as the repository tool
// read it only then, and a read keyed by the call's own arguments only when
// the call names them.
type alongsideSpec struct {
	key       string
	operation string
	// what names the records, for the note a cut-short read leaves.
	what string
	// inputs are the table input's fields every connection read sends
	// (active templates, started by hand), or the variables a keyed read
	// sends beside its keys (the open swaps only, at most twenty).
	inputs map[string]any
	// scoped reads it only when the call names one of the list's references
	// (the credential types beside one worker's credentials, not beside what
	// falls due across the organization).
	scoped bool
	// unless are the list's parameters that, named, leave it unread: a call
	// that names a credential type already knows the types, and a dozen
	// workers checked for one each came back with the whole catalog again.
	unless []string
	// withheld is the note a result carries when the reader may not read
	// the kind, so the model knows the records exist and are hidden.
	withheld string
	// shape applies to each record what its resolver does not (what a sync
	// record's status means).
	shape func(row map[string]any)
	// note is what a read made instead of a list's always says beside its
	// records (a document with no state was never sent).
	note string
}

// graphQLUnpagedListTool is a list whose root field returns every record at
// once, keyed by its own arguments (a worker's checklists by workerId): its
// parameters are the operation's variables, sent as the screen sends them,
// and it neither pages nor filters by a record's fields.
type graphQLUnpagedListTool struct {
	spec   graphQLListSpec
	read   graphQLRead
	params []graphQLParam
	// paged marks a root that takes a limit and an offset, which the tool
	// pages by cursor: the offset stays inside the nextCursor it hands out.
	paged       bool
	required    []string
	beside      besideReads
	reader      agentports.GraphQLReader
	permissions permissionports.PermissionEngine
}

// newGraphQLReadTool builds a spec's list tool in the shape its root field
// reads: paged through a table input, or unpaged by its own arguments.
func newGraphQLReadTool(
	spec *graphQLListSpec,
	reader agentports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (agentports.AgentQueryTool, error) {
	if read, ok := graphQLReads[spec.name]; ok && read.Unpaged {
		return newGraphQLUnpagedListTool(spec, reader, permissions)
	}
	if len(spec.matches) > 0 {
		return nil, fmt.Errorf("%s pages; its query is the server's, not matched in the rows",
			spec.name)
	}

	return newGraphQLListTool(spec, reader, permissions)
}

func newGraphQLUnpagedListTool(
	spec *graphQLListSpec,
	reader agentports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (*graphQLUnpagedListTool, error) {
	read, ok := graphQLReads[spec.name]
	if !ok {
		return nil, fmt.Errorf("%s: %w", spec.name, errGraphQLReadUnbound)
	}
	if read.Verb != graphQLVerbList || !read.Unpaged {
		return nil, fmt.Errorf("%s reads %s, which is not an unpaged list", spec.name,
			read.Operation)
	}
	if spec.rank != nil || spec.defaultsAt != nil || len(spec.instead) > 0 ||
		len(spec.conditions) > 0 || (spec.queryNote == "") != (len(spec.matches) == 0) {
		return nil, fmt.Errorf("%s: %w", spec.name, errUnpagedSpec)
	}

	params, required, err := read.argumentParams(spec.name)
	if err != nil {
		return nil, err
	}
	if params, required, err = renamed(spec.name, params, spec.names, required); err != nil {
		return nil, err
	}
	if params, err = referenced(spec.name, params, spec.references); err != nil {
		return nil, err
	}
	if params, err = annotate(spec.name, params, spec.notes); err != nil {
		return nil, err
	}
	if spec.amounts && slices.ContainsFunc(params, func(p graphQLParam) bool {
		_, minor := moneyName(p.Name)
		return minor
	}) {
		return nil, fmt.Errorf("%s: %w", spec.name, errUnpagedSpec)
	}
	for _, name := range required {
		if !slices.ContainsFunc(params, func(p graphQLParam) bool { return p.Name == name }) {
			return nil, fmt.Errorf("%s requires %s, which no tool supplies; name its reference",
				spec.name, name)
		}
	}
	alongside, err := matchAlongside(spec.name, "alongside", spec.alongside, read.Alongside)
	if err != nil {
		return nil, err
	}

	return &graphQLUnpagedListTool{
		spec:     *spec,
		read:     read,
		params:   params,
		paged:    read.pagesByOffset(),
		required: required,
		beside: besideReads{
			specs: spec.alongside, bound: alongside, reader: reader, permissions: permissions,
		},
		reader:      reader,
		permissions: permissions,
	}, nil
}

// pagesByOffset reports whether an unpaged list's root takes a limit and an
// offset, so the tool can page it by cursor.
func (r *graphQLRead) pagesByOffset() bool {
	var limit, offset bool
	for _, variable := range r.Variables {
		limit = limit || (variable.Name == paramLimit && variable.Type == graphQLTypeInt)
		offset = offset || (variable.Name == paramOffset && variable.Type == graphQLTypeInt)
	}

	return limit && offset
}

// argumentParams are an unpaged list's parameters: each of its operation's
// variables, typed by the root field argument it is sent as and sent as
// that variable, and the names of those the operation requires. The offset
// of a list that pages by cursor is the tool's to send, never a parameter.
func (r *graphQLRead) argumentParams(tool string) (params []graphQLParam, required []string,
	err error,
) {
	paged := r.pagesByOffset()
	for _, variable := range r.Variables {
		if variable.Kind == graphQLKindInput || (paged && variable.Name == paramOffset) {
			continue
		}
		field := graphQLField{
			Name:        variable.Name,
			Type:        variable.Type,
			Kind:        variable.Kind,
			List:        variable.List,
			NonNull:     variable.NonNull,
			Enum:        variable.Enum,
			Description: variable.Description,
		}
		derived := argumentParam(&field)
		if len(derived) == 0 {
			return nil, nil, fmt.Errorf("%s: %s takes $%s, which no parameter can be derived "+
				"from; describe the argument in the schema", tool, r.Operation, variable.Name)
		}
		for _, param := range derived {
			param.OnInput = true
			if said := variable.Description; said != "" && !variable.List &&
				variable.Type == graphQLTypeString {
				param.Schema = withDescription(param.Schema, said)
			}
			params = append(params, param)
		}
		if variable.NonNull {
			required = append(required, variable.Name)
		}
	}
	slices.SortFunc(params, func(a, b graphQLParam) int { return strings.Compare(a.Name, b.Name) })

	return params, required, nil
}

// argumentParam is one argument's parameter. An id is a reference the spec
// types by its resource; anything else is derived as a table input's own
// field is.
func argumentParam(field *graphQLField) []graphQLParam {
	if field.Type == graphQLTypeID && field.List {
		words := stringutils.HumanizeSnakeCase(stringutils.ConvertCamelToSnake(
			strings.TrimSuffix(field.Name, "Ids")))
		return []graphQLParam{{
			Name: field.Name, Field: field.Name, Kind: paramReferences,
			Schema: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "The " + words + " records whose records are read.",
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeString,
				},
				toolschema.KeyMinItems: 1,
			},
		}}
	}
	if field.Type == graphQLTypeID && !field.List {
		words := stringutils.HumanizeSnakeCase(stringutils.ConvertCamelToSnake(
			strings.TrimSuffix(field.Name, suffixID)))
		return []graphQLParam{{
			Name: field.Name, Field: field.Name, Kind: paramReference,
			Schema: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: "The " + words + " whose records are read.",
			},
		}}
	}

	if field.Type == graphQLTypeID {
		description := field.Description
		if description == "" {
			description = "The " + stringutils.HumanizeSnakeCase(stringutils.ConvertCamelToSnake(
				field.Name)) + " whose records are read."
		}
		return []graphQLParam{{
			Name: field.Name, Field: field.Name, Kind: paramReferences,
			Schema: agenttoolschema.IDList(description, unpagedMost),
		}}
	}
	if field.Type == graphQLTypeInt && !field.List {
		return intArgumentParam(field)
	}
	if field.Type == graphQLTypeString && !field.List &&
		strings.Contains(field.Description, dayTextFormat) {
		return []graphQLParam{{
			Name: field.Name, Field: field.Name, Kind: paramDayText,
			Schema: agenttoolschema.Date(strings.TrimSpace(
				strings.ReplaceAll(field.Description, ", as "+dayTextFormat, ""))),
		}}
	}

	return inputFieldParams(field)
}

// intArgumentParam is an Int argument's parameter: a limit takes the list
// tools' bounds, and any other number is the value the schema describes (a
// case year), sent as it is.
func intArgumentParam(field *graphQLField) []graphQLParam {
	if field.Name == paramLimit {
		return []graphQLParam{{
			Name: field.Name, Field: field.Name, Kind: paramEquals,
			Schema: limitSchema(graphQLListLimit, unpagedMost),
		}}
	}
	if field.Description == "" {
		return nil
	}

	return []graphQLParam{{
		Name: field.Name, Field: field.Name, Kind: paramEquals,
		Schema: map[string]any{
			toolschema.KeyType:        toolschema.TypeInteger,
			toolschema.KeyDescription: field.Description,
		},
	}}
}

// cutShort is the note a list whose limit it filled leaves, since an unpaged
// list cannot hand out the next page.
func cutShort(args map[string]any, count int, plural string) string {
	limit := optionalInt(args, paramLimit, 0)
	if limit <= 0 || count < limit {
		return ""
	}

	return fmt.Sprintf("These are the first %d %s; more may exist. Narrow the call, or raise "+
		"limit to at most %d.", count, plural, unpagedMost)
}

func (t *graphQLUnpagedListTool) Name() string { return t.spec.name }

// Description is the summary alone: the parameters are the few arguments the
// screen sends, each described in the schema, not a record's filterable
// fields.
func (t *graphQLUnpagedListTool) Description() string { return t.spec.summary }

func (t *graphQLUnpagedListTool) SearchTerms() []string { return t.spec.searchTerms }

func (t *graphQLUnpagedListTool) Policy() agentports.ToolPolicy {
	spec := t.spec.reads
	spec.resource = t.spec.resource

	return readPolicy(t.Name(), spec)
}

func (t *graphQLUnpagedListTool) ParamSchema() map[string]any {
	properties := make(map[string]any, len(t.params))
	for _, param := range t.params {
		properties[param.Name] = param.Schema
	}
	if t.paged {
		properties[paramAfter] = cursorPageSchema(graphQLListLimit, unpagedMost)[paramAfter]
	}
	if len(t.spec.matches) > 0 {
		properties[graphQLParamQuery] = map[string]any{
			toolschema.KeyType:        toolschema.TypeString,
			toolschema.KeyDescription: t.spec.queryNote,
		}
	}
	schema := map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyAdditionalProperties: false,
	}
	if len(t.required) > 0 {
		schema[toolschema.KeyRequired] = slices.Clone(t.required)
	}

	return schema
}

func (t *graphQLUnpagedListTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	clk := clockFor(params)
	args := applyDefaults(&t.spec, params.Params, clk)
	if err := t.checkArguments(args); err != nil {
		return nil, err
	}
	variables, err := compileInputs(t.params, args, clk)
	if err != nil {
		return nil, err
	}
	for name, value := range t.spec.inputs {
		variables[name] = value
	}
	var window offsetWindow
	if t.paged {
		if window, err = readOffsetWindow(t.spec.name, args, graphQLListLimit,
			unpagedMost); err != nil {
			return nil, err
		}
		variables[paramLimit] = window.fetch(unpagedMost)
		variables[paramOffset] = window.offset
	}

	result, err := t.reader.Read(ctx, &agentports.GraphQLReadRequest{
		Actor:             params.Actor,
		Operation:         t.read.Operation,
		Variables:         variables,
		DataAccessCeiling: params.DataAccessCeiling,
		RequestID:         params.RunID.String(),
	})
	if err != nil {
		return nil, err
	}
	if err = readFailure(result, t.spec.resource); err != nil {
		return nil, err
	}

	rows, prefix := t.rows(result.Data)
	more := t.paged && window.more(len(rows), unpagedMost)
	if t.paged && len(rows) > window.limit {
		rows = rows[:window.limit]
	}
	query := strings.ToLower(strings.TrimSpace(optionalString(params.Params, graphQLParamQuery)))
	items := make([]map[string]any, 0, len(rows))
	positions := make([]int, 0, len(rows))
	for idx, row := range rows {
		if node, ok := row.(map[string]any); ok && matchesQuery(node, t.spec.matches, query) {
			items = append(items, compact(node))
			positions = append(positions, idx)
		}
	}
	withheld := relativePaths(result.Withheld, prefix)
	if t.spec.shape != nil {
		hidden := make(map[string]struct{}, len(withheld))
		for _, path := range withheld {
			hidden[path] = struct{}{}
		}
		for _, item := range items {
			t.spec.shape(item, hidden, clk)
		}
	}
	if t.spec.amounts {
		for _, item := range items {
			presentAmounts(item)
		}
		withheld = presentedPaths(withheld)
	}

	criteria := describeCriteria(t.spec.entityPlural, t.params, args, query)
	outcome := map[string]any{
		outcomeCount:       len(items),
		outcomeSearchedFor: criteria.Describe(),
		outcomeItems:       items,
	}
	if columns := selectedColumns(t.read.Fields, items); len(columns) > 0 {
		outcome[outcomeColumns] = columns
	}
	var notes []string
	if len(items) == 0 {
		notes = append(notes, criteria.EmptyNote())
	}
	switch {
	case more:
		outcome[outcomeHasMore] = true
		outcome[outcomeNextCursor] = window.next()
	case !t.paged:
		if note := cutShort(args, len(items), t.spec.entityPlural); note != "" {
			notes = append(notes, note)
		}
	}
	beside, besideWithheld, besideNotes, err := t.beside.read(
		ctx, params, variables, namesReference(&t.spec, variables),
	)
	if err != nil {
		return nil, err
	}
	if t.spec.combine != nil {
		t.spec.combine(items, beside)
	}
	for key, rows := range beside {
		outcome[key] = rows
	}
	withheld = append(withheld, besideWithheld...)
	notes = append(notes, besideNotes...)
	if len(notes) > 0 {
		outcome[outcomeNote] = strings.Join(notes, " ")
	}
	if len(withheld) > 0 {
		outcome[outcomeWithheld] = withheld
		outcome[outcomeWithheldNote] = withheldNote
	}
	marked := markedRows(&t.spec, items)
	if t.paged {
		return &offsetPage{
			record: outcome, outside: marked, positions: positions, window: window,
		}, nil
	}
	if len(marked) > 0 {
		return &markedRecord{record: outcome, outside: marked}, nil
	}

	return outcome, nil
}

// offsetPage is a page of a list that pages by offset under the cursor
// contract, which a cut resumes at the offset after the last row it kept.
// positions are each item's place among the rows read, since a query the
// list matches itself keeps only some of them.
type offsetPage struct {
	record    map[string]any
	outside   []agent.RecordRef
	positions []int
	window    offsetWindow
}

func (p *offsetPage) MarshalJSON() ([]byte, error) { return sonic.Marshal(p.record) }

func (p *offsetPage) TaintedRecords() []agent.RecordRef { return p.outside }

func (p *offsetPage) items() []map[string]any {
	items, _ := p.record[outcomeItems].([]map[string]any)

	return items
}

func (p *offsetPage) PageRows() int { return len(p.items()) }

func (p *offsetPage) CutPage(rows int) (any, bool) {
	items := p.items()
	if rows <= 0 || rows >= len(items) || len(p.positions) != len(items) {
		return nil, false
	}
	record := maps.Clone(p.record)
	record[outcomeItems] = items[:rows]
	record[outcomeCount] = rows
	record[outcomeHasMore] = true
	record[outcomeNextCursor] = pagination.EncodeKeyCursor(
		p.window.scope, strconv.Itoa(p.window.offset+p.positions[rows-1]+1))

	return &offsetPage{
		record:    record,
		outside:   keptRecords(p.outside, items[:rows]),
		positions: p.positions[:rows],
		window:    p.window,
	}, true
}

// rows are the records the root returned, as a list or under items, and the
// prefix their withheld paths carry.
func (t *graphQLUnpagedListTool) rows(data map[string]any) (rows []any, prefix string) {
	if !t.read.Items {
		rows, _ = data[t.read.RootField].([]any)
		return rows, t.read.RootField + "."
	}
	holder, _ := data[t.read.RootField].(map[string]any)
	rows, _ = holder[outcomeItems].([]any)

	return rows, t.read.RootField + "." + outcomeItems + "."
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
func (t *graphQLUnpagedListTool) checkArguments(args map[string]any) error {
	for _, name := range t.required {
		if raw, present := args[name]; !present || raw == nil {
			return fmt.Errorf("parameter %q is required", name)
		}
	}
	for _, param := range t.params {
		raw, present := args[param.Name]
		if !present || raw == nil {
			continue
		}
		switch param.Kind { //nolint:exhaustive // only ids are checked here
		case paramReference:
			if _, err := requirePulid(args, param.Name); err != nil {
				return err
			}
		case paramReferences:
			if _, err := pulidList(raw, param.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

// alongsideVariables are what a read alongside sends: a keyed read the
// values the list's own call sent for its keys, and whether the call named
// them all (a read keyed by nothing is made on every call); a connection
// read its table input.
func alongsideVariables(
	bound *graphQLAlongside,
	spec *alongsideSpec,
	sent map[string]any,
) (variables map[string]any, keyed bool) {
	if bound.Input == "" {
		variables = make(map[string]any, len(bound.Keys)+len(spec.inputs))
		for name, value := range spec.inputs {
			variables[name] = value
		}
		for _, key := range bound.Keys {
			value, present := sent[key]
			if !present || value == nil {
				return nil, false
			}
			variables[key] = value
		}

		return variables, true
	}
	input := make(map[string]any, len(spec.inputs)+1)
	for name, value := range spec.inputs {
		input[name] = value
	}
	input[inputFirst] = alongsideMost

	return map[string]any{bound.Input: input}, true
}

// alongsideRows are the records a read alongside returned, the prefix its
// withheld paths carry, and the note a connection read cut short leaves.
func alongsideRows(
	bound *graphQLAlongside,
	spec *alongsideSpec,
	data map[string]any,
) (rows []map[string]any, prefix, note string) {
	if bound.Input == "" {
		list, _ := data[bound.RootField].([]any)
		rows = make([]map[string]any, 0, len(list))
		for _, row := range list {
			if node, ok := row.(map[string]any); ok {
				rows = append(rows, compact(node))
			}
		}

		return rows, bound.RootField + ".", ""
	}
	connection, _ := data[bound.RootField].(map[string]any)
	edges, _ := connection[connectionEdges].([]any)
	rows = make([]map[string]any, 0, len(edges))
	for _, edge := range edges {
		object, _ := edge.(map[string]any)
		if node, ok := object[connectionNode].(map[string]any); ok {
			rows = append(rows, compact(node))
		}
	}
	if info, ok := connection[connectionPageInfo].(map[string]any); ok {
		if more, _ := info[pageHasNext].(bool); more {
			note = fmt.Sprintf("Only the first %d %s are shown in %s.",
				alongsideMost, spec.what, spec.key)
		}
	}

	return rows, bound.RootField + "." + connectionEdges + "." + connectionNode + ".", note
}

// relativePaths names withheld fields from the record, as a model reads
// them: workerChecklists.items.note is items.note.
func relativePaths(paths []string, prefix string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, strings.TrimPrefix(path, prefix))
	}

	return out
}

// describeCriteria is what a call asked a list for, as its outcome reports it
// and its empty note words it.
func describeCriteria(
	plural string,
	params []graphQLParam,
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
