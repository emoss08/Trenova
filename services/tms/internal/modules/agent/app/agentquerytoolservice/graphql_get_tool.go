package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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

	read   graphQLRead
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

	return &graphQLGetTool{
		getTool: getTool{spec: *spec, access: newFieldAccess(permissions)},
		read:    read,
		reader:  reader,
	}, nil
}

func (t *graphQLGetTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	id, err := requirePulid(params.Params, t.spec.paramName)
	if err != nil {
		return nil, err
	}

	result, err := t.reader.Read(ctx, &agentports.GraphQLReadRequest{
		Actor:             params.Actor,
		Operation:         t.read.Operation,
		Variables:         map[string]any{t.read.keyVariable(): id.String()},
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
	if !ok {
		return nil, fmt.Errorf("%s %s: %w", t.spec.entity, id, errGraphQLRecordMissing)
	}
	record := compact(node)
	presentMinor(record, t.spec.minorUnits)
	if withheld := relativePaths(result.Withheld, t.read.RootField+"."); len(withheld) > 0 {
		record[outcomeWithheld] = withheld
		record[outcomeWithheldNote] = withheldNote
	}
	if outside := outsideRecords(t.spec.resource, t.spec.outside, record); len(outside) > 0 {
		return &markedRecord{record: record, outside: outside}, nil
	}

	return record, nil
}

// keyVariable is the variable a get's operation takes its key in: the only
// id it requires when that is not named id (a driver's earnings by
// $workerId), as agentread/gen binds it; otherwise id.
func (r *graphQLRead) keyVariable() string {
	key := ""
	for _, variable := range r.Variables {
		if !variable.NonNull {
			continue
		}
		if key != "" || variable.Type != graphQLTypeID || variable.List {
			return graphQLIDVariable
		}
		key = variable.Name
	}
	if key == "" {
		return graphQLIDVariable
	}

	return key
}

// markedRecord is a get's record that shows text written outside the
// organization, encoded as the record and naming itself so the run marks
// what it read.
type markedRecord struct {
	record  map[string]any
	outside []agent.RecordRef
}

func (r *markedRecord) MarshalJSON() ([]byte, error) { return sonic.Marshal(r.record) }

func (r *markedRecord) TaintedRecords() []agent.RecordRef { return r.outside }
