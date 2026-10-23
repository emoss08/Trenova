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
	"github.com/emoss08/trenova/pkg/filtercatalog"
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
	alongside   []graphQLAlongside
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
	if len(spec.alongside) > 0 {
		return nil, fmt.Errorf("%s pages; only an unpaged list reads alongside its own",
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
	if spec.rank != nil || spec.defaultsAt != nil || spec.queryNote != "" {
		return nil, fmt.Errorf("%s: %w", spec.name, errUnpagedSpec)
	}

	params, required, err := read.argumentParams(spec.name)
	if err != nil {
		return nil, err
	}
	if params, err = referenced(spec.name, params, spec.references); err != nil {
		return nil, err
	}
	if params, err = annotate(spec.name, params, spec.notes); err != nil {
		return nil, err
	}
	for _, name := range required {
		if !slices.ContainsFunc(params, func(p graphQLParam) bool { return p.Name == name }) {
			return nil, fmt.Errorf("%s requires %s, which no tool supplies; name its reference",
				spec.name, name)
		}
	}
	alongside, err := matchAlongside(spec, read.Alongside)
	if err != nil {
		return nil, err
	}

	return &graphQLUnpagedListTool{
		spec:        *spec,
		read:        read,
		params:      params,
		paged:       read.pagesByOffset(),
		required:    required,
		alongside:   alongside,
		reader:      reader,
		permissions: permissions,
	}, nil
}

// matchAlongside pairs each read the spec makes alongside with the one the
// generator proved, and refuses either without the other.
func matchAlongside(
	spec *graphQLListSpec,
	bound []graphQLAlongside,
) ([]graphQLAlongside, error) {
	out := make([]graphQLAlongside, 0, len(spec.alongside))
	for _, beside := range spec.alongside {
		idx := slices.IndexFunc(bound, func(b graphQLAlongside) bool {
			return b.Operation == beside.operation
		})
		if idx < 0 {
			return nil, fmt.Errorf("%s reads %s alongside, which reads.yml does not list for it",
				spec.name, beside.operation)
		}
		for name := range beside.inputs {
			if len(bound[idx].Keys) > 0 && !slices.Contains(bound[idx].Set, name) {
				return nil, fmt.Errorf("%s reads %s alongside by the call's own %s; it "+
					"takes no %s of the spec's", spec.name, beside.operation,
					strings.Join(bound[idx].Keys, ", "), name)
			}
		}
		out = append(out, bound[idx])
	}
	if len(out) != len(bound) {
		return nil, fmt.Errorf("%s: reads.yml lists %d reads alongside it, its spec %d",
			spec.name, len(bound), len(out))
	}

	return out, nil
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

	rows, _ := result.Data[t.read.RootField].([]any)
	more := t.paged && window.more(len(rows), unpagedMost)
	if t.paged && len(rows) > window.limit {
		rows = rows[:window.limit]
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if node, ok := row.(map[string]any); ok {
			items = append(items, compact(node))
		}
	}
	withheld := relativePaths(result.Withheld, t.read.RootField+".")
	if t.spec.shape != nil {
		hidden := make(map[string]struct{}, len(withheld))
		for _, path := range withheld {
			hidden[path] = struct{}{}
		}
		for _, item := range items {
			t.spec.shape(item, hidden, clk)
		}
	}

	criteria := describeCriteria(t.spec.entityPlural, t.params, args, "")
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
	beside, besideWithheld, besideNotes, err := t.readAlongside(ctx, params, variables)
	if err != nil {
		return nil, err
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

	return outcome, nil
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

// readAlongside makes each read the list makes beside its own, where the
// reader may read its kind and the call names what a keyed read is keyed by,
// and returns the records by key, what it withheld, and a note for a read
// cut short.
func (t *graphQLUnpagedListTool) readAlongside(
	ctx context.Context,
	params *agentports.QueryToolParams,
	sent map[string]any,
) (records map[string][]map[string]any, withheld, notes []string, err error) {
	if len(t.alongside) == 0 {
		return nil, nil, nil, nil
	}
	ceilings := fieldsensitivity.NewAgentCeilings(
		t.permissions, params.Actor, params.DataAccessCeiling,
	)
	records = make(map[string][]map[string]any, len(t.alongside))
	for idx := range t.alongside {
		bound := &t.alongside[idx]
		spec := &t.spec.alongside[idx]
		resource := permission.Resource(bound.Resource)
		if !ceilings.Reads(ctx, resource) || (spec.scoped && !namesReference(&t.spec, sent)) {
			continue
		}
		variables, keyed := alongsideVariables(bound, spec, sent)
		if !keyed {
			continue
		}

		var result *agentports.GraphQLReadResult
		result, err = t.reader.Read(ctx, &agentports.GraphQLReadRequest{
			Actor:             params.Actor,
			Operation:         bound.Operation,
			Variables:         variables,
			DataAccessCeiling: params.DataAccessCeiling,
			RequestID:         params.RunID.String(),
		})
		if err != nil {
			return nil, nil, nil, err
		}
		if err = readFailure(result, resource); err != nil {
			var refused *agentports.ReadRefusedError
			if errors.As(err, &refused) {
				continue
			}
			return nil, nil, nil, err
		}

		rows, prefix, note := alongsideRows(bound, spec, result.Data)
		records[spec.key] = rows
		for _, path := range relativePaths(result.Withheld, prefix) {
			withheld = append(withheld, spec.key+"."+path)
		}
		if note != "" {
			notes = append(notes, note)
		}
	}

	return records, withheld, notes, nil
}

// alongsideVariables are what a read alongside sends: a keyed read the
// values the list's own call sent for its keys, and whether the call named
// them all; a connection read its table input.
func alongsideVariables(
	bound *graphQLAlongside,
	spec *alongsideSpec,
	sent map[string]any,
) (variables map[string]any, keyed bool) {
	if len(bound.Keys) > 0 {
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
	if len(bound.Keys) > 0 {
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
