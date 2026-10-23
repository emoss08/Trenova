package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	graphQLVerbGet = "get"
	// graphQLIDVariable is the variable a get's operation takes the id in.
	graphQLIDVariable = "id"
)

var errGraphQLRecordMissing = errors.New("the read returned no record")

// graphQLGetTool reads one record through the operation its card sends, by
// the id its parameter names. Its name, description, policy and id
// parameter are the get spec's, as every get tool's are; the record is the
// operation's selection, shaped as a list's items are, with what access
// withheld named beside it.
type graphQLGetTool struct {
	getTool

	read graphQLRead
	// params are the operation's variables other than the id, and required
	// those of them it requires.
	params   []graphQLParam
	required []string
	// byID and idRequired say whether the operation takes the record's id,
	// and needs it.
	byID       bool
	idRequired bool
	// key is the variable the id is sent in: $id, or the one id a record
	// read by the record it belongs to requires (a driver's hours by
	// $workerId).
	key    string
	beside besideReads
	reader agentports.GraphQLReader
}

func newGraphQLGetTool(
	spec *getSpec,
	reader agentports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (*graphQLGetTool, error) {
	read, ok := graphQLReads[spec.name]
	if !ok {
		return nil, fmt.Errorf("%s: %w", spec.name, errGraphQLReadUnbound)
	}
	if read.Verb != graphQLVerbGet {
		return nil, fmt.Errorf("%s reads %s, which is not a get", spec.name, read.Operation)
	}
	if spec.fetch != nil {
		return nil, fmt.Errorf("%s reads through GraphQL; its spec fetches nothing itself",
			spec.name)
	}
	tool := &graphQLGetTool{
		getTool: getTool{spec: *spec, access: newFieldAccess(permissions)},
		read:    read,
		reader:  reader,
	}
	tool.key = read.getKey()
	for _, variable := range read.Variables {
		if variable.Name == tool.key {
			tool.byID, tool.idRequired = true, variable.NonNull
		}
	}
	if tool.byID && spec.paramName == "" {
		return nil, fmt.Errorf("%s reads by id; its spec names no id parameter", spec.name)
	}
	params, required, _, err := read.rootParams(spec.name,
		map[string]struct{}{tool.key: {}})
	if err != nil {
		return nil, err
	}
	if params, required, err = renamed(spec.name, params, spec.names, required); err != nil {
		return nil, err
	}
	if params, err = referenced(spec.name, params, spec.references); err != nil {
		return nil, err
	}
	if tool.params, err = annotate(spec.name, params, spec.notes); err != nil {
		return nil, err
	}
	tool.required = required
	bound, err := matchAlongside(spec.name, "alongside", spec.alongside, read.Alongside)
	if err != nil {
		return nil, err
	}
	tool.beside = besideReads{
		specs: spec.alongside, bound: bound, reader: reader, permissions: permissions,
	}

	return tool, nil
}

// ParamSchema is the id the get's spec names, as every get tool's is, and
// any other argument its operation takes.
func (t *graphQLGetTool) ParamSchema() map[string]any {
	if len(t.params) == 0 && t.byID && t.idRequired {
		return t.getTool.ParamSchema()
	}
	properties := make(map[string]any, len(t.params)+1)
	required := slices.Clone(t.required)
	if t.byID {
		own, _ := t.getTool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
		properties[t.spec.paramName] = own[t.spec.paramName]
		if t.idRequired {
			required = append([]string{t.spec.paramName}, required...)
		}
	}
	for _, param := range t.params {
		properties[param.Name] = param.Schema
	}

	return argumentSchema(properties, required)
}

func (t *graphQLGetTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	variables, named, err := t.variables(params)
	if err != nil {
		return nil, err
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

	node, ok := result.Data[t.read.RootField].(map[string]any)
	if !ok && t.spec.absent != nil {
		return t.spec.absent(pulid.ID(named)), nil
	}
	if !ok {
		return nil, fmt.Errorf("%s %s: %w", t.spec.entity, named, errGraphQLRecordMissing)
	}
	record := compact(node)
	withheld := relativePaths(result.Withheld, t.read.RootField+".")
	if t.spec.shape != nil {
		hidden := make(map[string]struct{}, len(withheld))
		for _, path := range withheld {
			hidden[path] = struct{}{}
		}
		t.spec.shape(record, hidden, clockFor(params))
	}
	beside, besideWithheld, notes, err := t.beside.read(ctx, params, variables, false)
	if err != nil {
		return nil, err
	}
	for key, value := range beside {
		record[key] = value
	}
	if withheld = append(withheld, besideWithheld...); len(withheld) > 0 {
		record[outcomeWithheld] = withheld
		record[outcomeWithheldNote] = withheldNote
	}
	if len(notes) > 0 {
		record[outcomeNote] = strings.Join(notes, " ")
	}

	return record, nil
}

// variables are what the read sends: the id the call names, parsed so a
// paraphrased id is refused rather than read as a missing record, and the
// operation's other arguments; and how a missing record is named.
func (t *graphQLGetTool) variables(
	params *agentports.QueryToolParams,
) (variables map[string]any, named string, err error) {
	if err = requireArguments(t.required, params.Params); err != nil {
		return nil, "", err
	}
	if err = checkReferences(t.params, params.Params); err != nil {
		return nil, "", err
	}
	if variables, err = compileArgs(t.params, params.Params, clockFor(params)); err != nil {
		return nil, "", err
	}
	if !t.byID {
		return variables, "for those arguments", nil
	}
	var id pulid.ID
	if t.idRequired {
		id, err = requirePulid(params.Params, t.spec.paramName)
	} else {
		id, err = optionalID(params.Params, t.spec.paramName)
	}
	if err != nil {
		return nil, "", err
	}
	if !id.IsNil() {
		variables[t.key] = id.String()
		named = id.String()
	}

	return variables, named, nil
}

// getKey is the variable a get's operation takes the record's id in: $id,
// or the one id it requires, for a record read by the id of the one it
// belongs to (a driver's hours by $workerId), as the generator proved.
func (r *graphQLRead) getKey() string {
	key := ""
	for _, variable := range r.Variables {
		if variable.Name == graphQLIDVariable {
			return graphQLIDVariable
		}
		if variable.NonNull && !variable.List && variable.Type == graphQLTypeID {
			key = variable.Name
		}
	}
	if key == "" {
		return graphQLIDVariable
	}

	return key
}
