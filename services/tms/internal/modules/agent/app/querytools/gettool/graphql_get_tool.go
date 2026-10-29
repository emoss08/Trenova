package gettool

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"golang.org/x/sync/errgroup"
)

const (
	// GraphQLAtOnce is how many reads a get for several ids runs at
	// once, each its own read under its own span.
	GraphQLAtOnce = 10
)

var errGraphQLRecordMissing = errors.New("the read returned no record")

// GraphQLTool reads one record through the operation its card sends, by
// the id its parameter names. Its name, description, policy and id
// parameter are the get spec's, as every get tool's are; the record is the
// operation's selection, shaped as a list's items are, with what access
// withheld named beside it. A spec with a batch takes a list of ids and
// reads each one, at most graphQLGetsAtOnce at once.
type GraphQLTool struct {
	Tool

	read graphqlread.Read
	// params are the operation's variables other than the id, and required
	// those of them it requires.
	Params   []graphqlread.Param
	Required []string
	// byID and idRequired say whether the operation takes the record's id,
	// and needs it.
	ByID       bool
	idRequired bool
	// key is the variable the id is sent in: $id, or the one id a record
	// read by the record it belongs to requires (a driver's hours by
	// $workerId).
	key    string
	keyRef *readspec.Reference
	beside graphqlread.BesideReads
	reader graphqlreadports.GraphQLReader
}

func NewGraphQL(
	spec *readspec.Get,
	reader graphqlreadports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (*GraphQLTool, error) {
	read, ok := graphqlread.Reads[spec.Name]
	if !ok {
		return nil, fmt.Errorf("%s: %w", spec.Name, graphqlread.ErrUnbound)
	}
	if read.Verb != graphqlread.VerbGet {
		return nil, fmt.Errorf("%s reads %s, which is not a get", spec.Name, read.Operation)
	}
	if spec.Fetch != nil {
		return nil, fmt.Errorf("%s reads through GraphQL; its spec fetches nothing itself",
			spec.Name)
	}
	tool := &GraphQLTool{
		Tool:   Tool{Spec: *spec, Access: fieldgate.NewAccess(permissions)},
		read:   read,
		reader: reader,
	}
	tool.key = read.GetKey()
	for _, variable := range read.Variables {
		if variable.Name == tool.key {
			tool.ByID, tool.idRequired = true, variable.NonNull
		}
	}
	if tool.ByID && spec.ParamName == "" {
		return nil, fmt.Errorf("%s reads by id; its spec names no id parameter", spec.Name)
	}
	if spec.Batch > 0 && (!tool.ByID || !tool.idRequired) {
		return nil, fmt.Errorf("%s reads several ids; its operation requires no id", spec.Name)
	}
	params, required, _, err := read.RootParams(spec.Name,
		map[string]struct{}{tool.key: {}})
	if err != nil {
		return nil, err
	}
	if params, required, err = graphqlread.Renamed(spec.Name, params, spec.Names, required); err != nil {
		return nil, err
	}
	references := spec.References
	if ref, keyed := references[spec.ParamName]; keyed && tool.ByID {
		tool.keyRef = &ref
		references = maps.Clone(references)
		delete(references, spec.ParamName)
	}
	if params, err = graphqlread.Referenced(spec.Name, params, references); err != nil {
		return nil, err
	}
	if tool.Params, err = graphqlread.Annotate(spec.Name, params, spec.Notes); err != nil {
		return nil, err
	}
	tool.Required = required
	bound, err := graphqlread.MatchAlongside(spec.Name, "alongside", spec.Alongside, read.Alongside)
	if err != nil {
		return nil, err
	}
	tool.beside = graphqlread.BesideReads{
		Specs: spec.Alongside, Bound: bound, Reader: reader, Permissions: permissions,
	}

	return tool, nil
}

// ParamSchema is the id the get's spec names, as every get tool's is, and
// any other argument its operation takes.
func (t *GraphQLTool) ParamSchema() map[string]any {
	if len(t.Params) == 0 && t.ByID && t.idRequired && t.keyRef == nil && t.Spec.Batch == 0 {
		return t.Tool.ParamSchema()
	}
	properties := make(map[string]any, len(t.Params)+1)
	required := slices.Clone(t.Required)
	if t.ByID {
		properties[t.Spec.ParamName] = t.idProperty()
		if t.idRequired {
			required = append([]string{t.Spec.ParamName}, required...)
		}
	}
	for _, param := range t.Params {
		properties[param.Name] = param.Schema
	}

	return graphqlread.ArgumentSchema(properties, required)
}

// idProperty is the id parameter's schema: a list of ids for a spec with a
// batch, the reference a keyed get names, or the get's own id.
func (t *GraphQLTool) idProperty() any {
	switch {
	case t.Spec.Batch > 0:
		return agenttoolschema.RecordIDs(t.Spec.Resource,
			fmt.Sprintf("The %ss' ids, %s.", t.Spec.Entity, t.Spec.IDSource), t.Spec.Batch)
	case t.keyRef != nil:
		return t.keyRef.Schema(false)
	}
	own, _ := t.Tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)

	return own[t.Spec.ParamName]
}

func (t *GraphQLTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}
	if t.Spec.Batch > 0 {
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
func (t *GraphQLTool) one(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	variables map[string]any,
	named string,
	several bool,
) (any, error) {
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
	node, found := result.Data[t.read.RootField].(map[string]any)
	if !found && missingRecord(result) {
		if t.Spec.Absent != nil {
			return t.Spec.Absent(pulid.ID(named)), nil
		}
		return nil, fmt.Errorf("%s %s: %w", t.Spec.Entity, named, errGraphQLRecordMissing)
	}
	if err = graphqlread.ReadFailure(result, t.Spec.Resource); err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("%s %s: %w", t.Spec.Entity, named, errGraphQLRecordMissing)
	}
	record := graphqlread.Compact(node)
	withheld := graphqlread.RelativePaths(result.Withheld, t.read.RootField+".")
	if t.Spec.Shape != nil {
		hidden := make(map[string]struct{}, len(withheld))
		for _, path := range withheld {
			hidden[path] = struct{}{}
		}
		t.Spec.Shape(record, hidden, querytool.ClockFor(params))
	}
	if several && t.Spec.ShapeEach != nil {
		t.Spec.ShapeEach(record)
	}
	if t.Spec.Pages != nil {
		t.Spec.Pages.Mark(record, variables)
	}
	if t.Spec.Amounts {
		graphqlread.PresentAmounts(record)
		graphqlread.PresentMinor(record, t.Spec.MinorUnits)
		withheld = graphqlread.PresentedPaths(withheld)
	}
	beside, besideWithheld, notes, err := t.beside.Read(ctx, params, variables, false)
	if err != nil {
		return nil, err
	}
	for key, value := range beside {
		record[key] = value
	}
	if withheld = append(withheld, besideWithheld...); len(withheld) > 0 {
		record[graphqlread.OutcomeWithheld] = withheld
		record[graphqlread.OutcomeWithheldNote] = fieldgate.WithheldNote
	}
	if len(notes) > 0 {
		record[graphqlread.OutcomeNote] = strings.Join(notes, " ")
	}
	if outside := graphqlread.OutsideRecords(t.Spec.Resource, t.Spec.Outside, record); len(
		outside,
	) > 0 {
		return &graphqlread.MarkedRecord{Record: record, Outside: outside}, nil
	}

	return record, nil
}

// missingRecord reports whether a read found no record: its root came back
// empty, or its only error says the record does not exist.
func missingRecord(result *graphqlreadports.GraphQLReadResult) bool {
	if len(result.Errors) == 0 {
		return true
	}

	return len(result.Errors) == 1 && result.Errors[0].Code == graphqlreadports.GraphQLCodeNotFound
}

// several reads each id the call names, one read per id and at most
// graphQLGetsAtOnce at once. One id reads as the single get does; several
// come back as items in the order named, with what access withheld from any
// of them named once, and the ids that are no record of this organization
// named in the note. A refusal or a failed read fails the call.
func (t *GraphQLTool) several(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	ids, err := querytool.RequireIDList(params.Params, t.Spec.ParamName, t.Spec.Batch)
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
	group.SetLimit(GraphQLAtOnce)
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
func (t *GraphQLTool) gathered(ids []pulid.ID, records []any) any {
	items := make([]map[string]any, 0, len(ids))
	var missing, withheld []string
	var outside []agentrecord.RecordRef
	for idx, read := range records {
		record, found := read.(map[string]any)
		if mark, ok := read.(*graphqlread.MarkedRecord); ok {
			record, found = mark.Record, true
			outside = append(outside, mark.Outside...)
		}
		if !found || record == nil {
			missing = append(missing, ids[idx].String())
			continue
		}
		if paths, ok := record[graphqlread.OutcomeWithheld].([]string); ok {
			withheld = append(withheld, paths...)
		}
		delete(record, graphqlread.OutcomeWithheld)
		delete(record, graphqlread.OutcomeWithheldNote)
		items = append(items, record)
	}

	outcome := map[string]any{
		graphqlread.OutcomeCount:       len(items),
		graphqlread.OutcomeSearchedFor: []string{t.Spec.Entity + "s by id"},
		graphqlread.OutcomeItems:       items,
	}
	if columns := graphqlread.SelectedColumns(t.read.Fields, items); len(columns) > 0 {
		outcome[graphqlread.OutcomeColumns] = columns
	}
	if len(missing) > 0 {
		outcome[graphqlread.OutcomeNote] = fmt.Sprintf(
			"No %s of this organization has these ids: %s",
			t.Spec.Entity,
			strings.Join(missing, ", "),
		)
	}
	if len(withheld) > 0 {
		slices.Sort(withheld)
		outcome[graphqlread.OutcomeWithheld] = slices.Compact(withheld)
		outcome[graphqlread.OutcomeWithheldNote] = fieldgate.WithheldNote
	}
	if len(outside) > 0 {
		return &graphqlread.MarkedRecord{Record: outcome, Outside: outside}
	}

	return outcome
}

// variables are what the read sends: the id the call names, parsed so a
// paraphrased id is refused rather than read as a missing record, and the
// operation's other arguments; and how a missing record is named.
func (t *GraphQLTool) variables(
	params *agenttoolports.QueryToolParams,
) (variables map[string]any, named string, err error) {
	if variables, err = t.arguments(params); err != nil {
		return nil, "", err
	}
	if !t.ByID {
		return variables, "for those arguments", nil
	}
	var id pulid.ID
	if t.idRequired {
		id, err = querytool.RequirePulid(params.Params, t.Spec.ParamName)
	} else {
		id, err = querytool.OptionalID(params.Params, t.Spec.ParamName)
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
func (t *GraphQLTool) arguments(params *agenttoolports.QueryToolParams) (map[string]any, error) {
	if err := graphqlread.RequireArguments(t.Required, params.Params); err != nil {
		return nil, err
	}
	if err := graphqlread.CheckReferences(t.Params, params.Params); err != nil {
		return nil, err
	}

	args, err := graphqlread.CompileArgs(t.Params, params.Params, querytool.ClockFor(params))
	if err != nil {
		return nil, err
	}
	for _, variable := range t.read.Variables {
		if _, set := args[variable.Name]; !set && variable.NonNull &&
			variable.Kind == graphqlread.KindInput {
			args[variable.Name] = map[string]any{}
		}
	}

	return args, nil
}
