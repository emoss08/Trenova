package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"golang.org/x/sync/errgroup"
)

const (
	graphQLVerbGet = "get"
	// graphQLIDVariable is the variable a get's operation takes the id in.
	graphQLIDVariable = "id"
	// graphQLGetsAtOnce is how many reads a get for several ids runs at
	// once, each its own read under its own span.
	graphQLGetsAtOnce = 10
)

var errGraphQLRecordMissing = errors.New("the read returned no record")

// graphQLGetTool reads one record through the operation its card sends, by
// the id its parameter names. Its name, description, policy and id
// parameter are the get spec's, as every get tool's are; the record is the
// operation's selection, shaped as a list's items are, with what access
// withheld named beside it. A spec with a batch takes a list of ids and
// reads each one, at most graphQLGetsAtOnce at once.
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

// ParamSchema is the get's own id, or, for a spec with a batch, a list of
// them.
func (t *graphQLGetTool) ParamSchema() map[string]any {
	if t.spec.batch == 0 {
		return t.getTool.ParamSchema()
	}

	return idSchema(t.spec.paramName, agenttoolschema.RecordIDs(t.spec.resource,
		fmt.Sprintf("The %ss' ids, %s.", t.spec.entity, t.spec.idSource), t.spec.batch))
}

func (t *graphQLGetTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	if t.spec.batch > 0 {
		return t.several(ctx, params)
	}
	id, err := requirePulid(params.Params, t.spec.paramName)
	if err != nil {
		return nil, err
	}

	return t.one(ctx, params, id, false)
}

func (t *graphQLGetTool) one(
	ctx context.Context,
	params *agentports.QueryToolParams,
	id pulid.ID,
	several bool,
) (map[string]any, error) {
	result, err := t.reader.Read(ctx, &agentports.GraphQLReadRequest{
		Actor:             params.Actor,
		Operation:         t.read.Operation,
		Variables:         map[string]any{graphQLIDVariable: id.String()},
		DataAccessCeiling: params.DataAccessCeiling,
		RequestID:         params.RunID.String(),
	})
	if err != nil {
		return nil, err
	}
	node, found := result.Data[t.read.RootField].(map[string]any)
	if !found && missingRecord(result) {
		return nil, fmt.Errorf("%s %s: %w", t.spec.entity, id, errGraphQLRecordMissing)
	}
	if err = readFailure(result, t.spec.resource); err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("%s %s: %w", t.spec.entity, id, errGraphQLRecordMissing)
	}
	record := compact(node)
	if t.spec.shape != nil {
		t.spec.shape(record, several)
	}
	if withheld := relativePaths(result.Withheld, t.read.RootField+"."); len(withheld) > 0 {
		record[outcomeWithheld] = withheld
		record[outcomeWithheldNote] = withheldNote
	}

	return record, nil
}

// missingRecord reports whether a read found no record: its root came back
// empty, or its only error says the record does not exist.
func missingRecord(result *agentports.GraphQLReadResult) bool {
	if len(result.Errors) == 0 {
		return true
	}

	return len(result.Errors) == 1 && result.Errors[0].Code == agentports.GraphQLCodeNotFound
}

// several reads each id the call names, one read per id and at most
// graphQLGetsAtOnce at once. One id reads as the single get does; several
// come back as items in the order named, with what access withheld from any
// of them named once, and the ids that are no record of this organization
// named in the note. A refusal or a failed read fails the call.
func (t *graphQLGetTool) several(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	ids, err := requireIDList(params.Params, t.spec.paramName, t.spec.batch)
	if err != nil {
		return nil, err
	}
	ids = sliceutils.Dedupe(ids)
	if len(ids) == 1 {
		return t.one(ctx, params, ids[0], false)
	}

	records := make([]map[string]any, len(ids))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(graphQLGetsAtOnce)
	for idx, id := range ids {
		group.Go(func() error {
			record, readErr := t.one(groupCtx, params, id, true)
			if errors.Is(readErr, errGraphQLRecordMissing) {
				return nil
			}
			records[idx] = record
			return readErr
		})
	}
	if err = group.Wait(); err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0, len(ids))
	var missing, withheld []string
	for idx, record := range records {
		if record == nil {
			missing = append(missing, ids[idx].String())
			continue
		}
		if paths, ok := record[outcomeWithheld].([]string); ok {
			withheld = append(withheld, paths...)
		}
		delete(record, outcomeWithheld)
		delete(record, outcomeWithheldNote)
		items = append(items, record)
	}

	outcome := map[string]any{
		outcomeCount:       len(items),
		outcomeSearchedFor: []string{t.spec.entity + "s by id"},
		outcomeItems:       items,
	}
	if columns := selectedColumns(t.read.Fields, items); len(columns) > 0 {
		outcome[outcomeColumns] = columns
	}
	if len(missing) > 0 {
		outcome[outcomeNote] = fmt.Sprintf("No %s of this organization has these ids: %s",
			t.spec.entity, strings.Join(missing, ", "))
	}
	if len(withheld) > 0 {
		slices.Sort(withheld)
		outcome[outcomeWithheld] = slices.Compact(withheld)
		outcome[outcomeWithheldNote] = withheldNote
	}

	return outcome, nil
}
