package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

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

	outcomeCount        = "count"
	outcomeSearchedFor  = "searchedFor"
	outcomeItems        = "items"
	outcomeColumns      = "columns"
	outcomeNote         = "note"
	outcomeWithheld     = "withheldByAccess"
	outcomeWithheldNote = "withheldNote"
)

var errUnpagedSpec = errors.New("an unpaged list reads by its root field's arguments alone")

// alongsideSpec is a read an unpaged list makes beside its own, for records
// of another kind its next call needs (the templates a checklist can be
// started from), carried in the result under key. It is made only when the
// reader may read that kind, as the repository tool read it only then.
type alongsideSpec struct {
	key       string
	operation string
	// what names the records, for the note a cut-short read leaves.
	what string
	// inputs are the table input's fields every read sends (active
	// templates, started by hand).
	inputs map[string]any
}

// graphQLUnpagedListTool is a list whose root field returns every record at
// once, keyed by its own arguments (a worker's checklists by workerId): its
// parameters are the operation's variables, sent as the screen sends them,
// and it neither pages nor filters by a record's fields.
type graphQLUnpagedListTool struct {
	spec        graphQLListSpec
	read        graphQLRead
	params      []graphQLParam
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
		out = append(out, bound[idx])
	}
	if len(out) != len(bound) {
		return nil, fmt.Errorf("%s: reads.yml lists %d reads alongside it, its spec %d",
			spec.name, len(bound), len(out))
	}

	return out, nil
}

// argumentParams are an unpaged list's parameters: each of its operation's
// variables, typed by the root field argument it is sent as and sent as
// that variable, and the names of those the operation requires.
func (r *graphQLRead) argumentParams(tool string) (params []graphQLParam, required []string,
	err error,
) {
	for _, variable := range r.Variables {
		if variable.Kind == graphQLKindInput {
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

	return inputFieldParams(field)
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
			t.spec.shape(item, hidden)
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
	beside, besideWithheld, besideNotes, err := t.readAlongside(ctx, params)
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
		if param.Kind != paramReference {
			continue
		}
		if raw, present := args[param.Name]; present && raw != nil {
			if _, err := requirePulid(args, param.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

// readAlongside makes each read the list makes beside its own, where the
// reader may read its kind, and returns the records by key, what it
// withheld, and a note for a read cut short.
func (t *graphQLUnpagedListTool) readAlongside(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (records map[string][]map[string]any, withheld, notes []string, err error) {
	if len(t.alongside) == 0 {
		return nil, nil, nil, nil
	}
	ceilings := fieldsensitivity.NewAgentCeilings(
		t.permissions, params.Actor, params.DataAccessCeiling,
	)
	records = make(map[string][]map[string]any, len(t.alongside))
	for idx, bound := range t.alongside {
		spec := t.spec.alongside[idx]
		resource := permission.Resource(bound.Resource)
		if !ceilings.Reads(ctx, resource) {
			continue
		}
		input := make(map[string]any, len(spec.inputs)+1)
		for name, value := range spec.inputs {
			input[name] = value
		}
		input[inputFirst] = alongsideMost

		var result *agentports.GraphQLReadResult
		result, err = t.reader.Read(ctx, &agentports.GraphQLReadRequest{
			Actor:             params.Actor,
			Operation:         bound.Operation,
			Variables:         map[string]any{bound.Input: input},
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

		connection, _ := result.Data[bound.RootField].(map[string]any)
		edges, _ := connection[connectionEdges].([]any)
		rows := make([]map[string]any, 0, len(edges))
		for _, edge := range edges {
			object, _ := edge.(map[string]any)
			if node, ok := object[connectionNode].(map[string]any); ok {
				rows = append(rows, compact(node))
			}
		}
		records[spec.key] = rows
		prefix := bound.RootField + "." + connectionEdges + "." + connectionNode + "."
		for _, path := range relativePaths(result.Withheld, prefix) {
			withheld = append(withheld, spec.key+"."+path)
		}
		if info, ok := connection[connectionPageInfo].(map[string]any); ok {
			if more, _ := info[pageHasNext].(bool); more {
				notes = append(notes, fmt.Sprintf("Only the first %d %s are shown in %s.",
					alongsideMost, spec.what, spec.key))
			}
		}
	}

	return records, withheld, notes, nil
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
