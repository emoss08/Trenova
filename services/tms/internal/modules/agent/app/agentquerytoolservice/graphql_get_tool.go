package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/typeutils"
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
	keyRef *idReference
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
	if spec.batch > 0 && (!tool.byID || !tool.idRequired) {
		return nil, fmt.Errorf("%s reads several ids; its operation requires no id", spec.name)
	}
	params, required, _, err := read.rootParams(spec.name,
		map[string]struct{}{tool.key: {}})
	if err != nil {
		return nil, err
	}
	if params, required, err = renamed(spec.name, params, spec.names, required); err != nil {
		return nil, err
	}
	references := spec.references
	if ref, keyed := references[spec.paramName]; keyed && tool.byID {
		tool.keyRef = &ref
		references = maps.Clone(references)
		delete(references, spec.paramName)
	}
	if params, err = referenced(spec.name, params, references); err != nil {
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
	if len(t.params) == 0 && t.byID && t.idRequired && t.keyRef == nil && t.spec.batch == 0 {
		return t.getTool.ParamSchema()
	}
	properties := make(map[string]any, len(t.params)+1)
	required := slices.Clone(t.required)
	if t.byID {
		properties[t.spec.paramName] = t.idProperty()
		if t.idRequired {
			required = append([]string{t.spec.paramName}, required...)
		}
	}
	for _, param := range t.params {
		properties[param.Name] = param.Schema
	}

	return argumentSchema(properties, required)
}

// idProperty is the id parameter's schema: a list of ids for a spec with a
// batch, the reference a keyed get names, or the get's own id.
func (t *graphQLGetTool) idProperty() any {
	switch {
	case t.spec.batch > 0:
		return agenttoolschema.RecordIDs(t.spec.resource,
			fmt.Sprintf("The %ss' ids, %s.", t.spec.entity, t.spec.idSource), t.spec.batch)
	case t.keyRef != nil:
		return t.keyRef.schema(false)
	}
	own, _ := t.getTool.ParamSchema()[toolschema.KeyProperties].(map[string]any)

	return own[t.spec.paramName]
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
	variables, named, err := t.variables(params)
	if err != nil {
		return nil, err
	}

	return t.one(ctx, params, variables, named, false)
}

// one reads the record the variables name; several says whether it is one
// of a batch read together.
func (t *graphQLGetTool) one(
	ctx context.Context,
	params *agentports.QueryToolParams,
	variables map[string]any,
	named string,
	several bool,
) (any, error) {
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
	node, found := result.Data[t.read.RootField].(map[string]any)
	if !found && missingRecord(result) {
		if t.spec.absent != nil {
			return t.spec.absent(pulid.ID(named)), nil
		}
		return nil, fmt.Errorf("%s %s: %w", t.spec.entity, named, errGraphQLRecordMissing)
	}
	if err = readFailure(result, t.spec.resource); err != nil {
		return nil, err
	}
	if !found {
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
	if several && t.spec.shapeEach != nil {
		t.spec.shapeEach(record)
	}
	if t.spec.pages != nil {
		t.spec.pages.mark(record, variables)
	}
	if t.spec.amounts {
		presentAmounts(record)
		presentMinor(record, t.spec.minorUnits)
		withheld = presentedPaths(withheld)
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
	if outside := outsideRecords(t.spec.resource, t.spec.outside, record); len(outside) > 0 {
		return &markedRecord{record: record, outside: outside}, nil
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
	arguments, err := t.arguments(params)
	if err != nil {
		return nil, err
	}
	ids = sliceutils.Dedupe(ids)
	read := func(readCtx context.Context, id pulid.ID, several bool) (any, error) {
		variables := maps.Clone(arguments)
		variables[t.key] = id.String()
		return t.one(readCtx, params, variables, id.String(), several)
	}
	if len(ids) == 1 {
		return read(ctx, ids[0], false)
	}

	records := make([]any, len(ids))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(graphQLGetsAtOnce)
	for idx, id := range ids {
		group.Go(func() error {
			record, readErr := read(groupCtx, id, true)
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

	return t.gathered(ids, records), nil
}

// gathered is a batch's records as one outcome: the items in the order
// named, what access withheld from any of them named once, the ids that are
// no record of this organization named in the note, and every record whose
// text was written outside the organization marked.
func (t *graphQLGetTool) gathered(ids []pulid.ID, records []any) any {
	items := make([]map[string]any, 0, len(ids))
	var missing, withheld []string
	var outside []agent.RecordRef
	for idx, read := range records {
		record, found := read.(map[string]any)
		if mark, ok := read.(*markedRecord); ok {
			record, found = mark.record, true
			outside = append(outside, mark.outside...)
		}
		if !found || record == nil {
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
	if len(outside) > 0 {
		return &markedRecord{record: outcome, outside: outside}
	}

	return outcome
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

// variables are what the read sends: the id the call names, parsed so a
// paraphrased id is refused rather than read as a missing record, and the
// operation's other arguments; and how a missing record is named.
func (t *graphQLGetTool) variables(
	params *agentports.QueryToolParams,
) (variables map[string]any, named string, err error) {
	if variables, err = t.arguments(params); err != nil {
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

// arguments are the operation's variables other than the id, as the call
// names them. A required input the call names nothing of is sent empty, as
// the generator proved every field of it optional.
func (t *graphQLGetTool) arguments(params *agentports.QueryToolParams) (map[string]any, error) {
	if err := requireArguments(t.required, params.Params); err != nil {
		return nil, err
	}
	if err := checkReferences(t.params, params.Params); err != nil {
		return nil, err
	}

	args, err := compileArgs(t.params, params.Params, clockFor(params))
	if err != nil {
		return nil, err
	}
	for _, variable := range t.read.Variables {
		if _, set := args[variable.Name]; !set && variable.NonNull &&
			variable.Kind == graphQLKindInput {
			args[variable.Name] = map[string]any{}
		}
	}

	return args, nil
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

// mark says whether the held list continues past the page the variables
// read, and where the next page starts.
func (p *getPages) mark(record, variables map[string]any) {
	offset, _ := typeutils.IntOf(variables[p.offset])
	items, _ := record[p.items].([]any)
	total, counted := typeutils.IntOf(record[p.total])
	more := counted && offset+len(items) < total
	record[p.items+"HasMore"] = more
	if more {
		record["next"+strings.ToUpper(p.offset[:1])+p.offset[1:]] = offset + len(items)
	}
}
