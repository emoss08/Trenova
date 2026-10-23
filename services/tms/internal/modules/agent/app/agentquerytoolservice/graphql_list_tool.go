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
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	graphQLVerbList   = "list"
	graphQLParamQuery = "query"
	graphQLListLimit  = 25
	graphQLListMost   = 50

	connectionEdges      = "edges"
	connectionNode       = "node"
	connectionPageInfo   = "pageInfo"
	connectionTotalCount = "totalCount"
	pageHasNext          = "hasNextPage"
	pageEndCursor        = "endCursor"
	typenameField        = "__typename"

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
	// notes add what the schema does not say to a derived parameter's
	// description, keyed by the parameter's name.
	notes map[string]string
	// references name, for each id field, the resource its ids belong to
	// and the tool that supplies one. An id field with none is offered only
	// as its has<Thing> parameter: a model has no honest way to get an id no
	// tool lists.
	references map[string]idReference
	// rank orders a page that matched a free-text query when the call chose
	// no order, as the screen's search cannot (closest name first).
	rank func(query string, rows []map[string]any) []map[string]any
	// shape applies to each row what the screen's resolver does not, given
	// the fields withheld from it and the call's clock.
	shape func(row map[string]any, withheld map[string]struct{}, clk clock)
	// defaults are parameter values a call that omits them reads with (the
	// active courses, not the retired ones), named in their notes.
	defaults map[string]any
	// defaultsAt are defaults that depend on the day (time off from today),
	// applied together only when a call gives none of them.
	defaultsAt func(clk clock) map[string]any
	// unscoped are defaults a call that names none of its references reads
	// with (the open safety events across the organization, where one
	// worker's are read whole).
	unscoped map[string]any
	// inputs are input fields every call sends, as the screen sets them for
	// its own rendering (each time-off request's worker).
	inputs map[string]any
	// alongside are the reads a list makes beside its own: for records of
	// another kind its next call needs, or one keyed by the call that says
	// what its rows mean (the tolerance a drift is judged by).
	alongside []alongsideSpec
	// combine applies to a page what the records read alongside it say
	// about each row (a drift within the tolerance the overview carries).
	combine func(items []map[string]any, beside map[string]any)
	// instead are the reads a list makes in place of its own when a call
	// names their own parameters (the sync state of documents named by id).
	instead []alongsideSpec
	// names are the names the tool takes derived parameters by, keyed by
	// the derived name (integrationType is system).
	names map[string]string
	// instants are the Timestamp fields a call bounds to the minute rather
	// than the day (when an EDI message was created).
	instants []string
	// unoffered are the record's fields no one narrows the list by (the
	// template an EDI message was rendered from, the path it was dropped
	// at), left out of the parameters so the ones people use stay readable.
	unoffered []string
	resource  permission.Resource
	// reads is what the tool's records carry from outside the organization,
	// declared as every read tool declares it (readSpec).
	reads readSpec
}

// idReference is where a list's id parameter points: the resource, what the
// id is, and the tool a model gets one from. An id of several kinds (a sync
// record's document) names the kinds instead of a resource, and a list of
// ids names the most one call takes.
type idReference struct {
	resource permission.Resource
	kinds    []permission.RecordKind
	what     string
	source   string
	most     int
}

// schema is the parameter an id or a list of ids becomes.
func (ref *idReference) schema(list bool) map[string]any {
	description := agenttoolschema.IDDescription(ref.what, ref.source)
	if ref.most == 0 {
		ref.most = unpagedMost
	}
	switch {
	case list && len(ref.kinds) > 0:
		return agenttoolschema.KindIDs(description, ref.most, ref.kinds...)
	case list:
		return agenttoolschema.RecordIDs(ref.resource, description, ref.most)
	case len(ref.kinds) > 0:
		return agenttoolschema.KindID(description, ref.kinds...)
	}

	return agenttoolschema.RecordID(ref.resource, ref.what, ref.source)
}

type graphQLListTool struct {
	spec  graphQLListSpec
	read  graphQLRead
	input string
	// search is the filter input variable and field a list paged by its
	// root's own first and after takes its free text in.
	search [2]string
	params []graphQLParam
	// required are the arguments a call that lists records must name, and
	// demanded those the schema marks required: an argument a read made
	// instead can do without is required only when listing.
	required    []string
	demanded    []string
	sortable    agenttoolschema.EnumSource[string]
	beside      besideReads
	instead     []insteadRead
	reader      agentports.GraphQLReader
	permissions permissionports.PermissionEngine
	registry    *permission.Registry
}

// insteadRead is a read a list makes in place of its own, and the
// parameters of its own a call chooses it by.
type insteadRead struct {
	spec   *alongsideSpec
	bound  graphQLAlongside
	params []graphQLParam
	// shared are the list's own parameters the read is keyed by too.
	shared []graphQLParam
	// derived are its parameters' names before the spec named them.
	derived []string
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
	input, inputType, tabled := read.tableInput()
	skip := map[string]struct{}{includeTotalCount: {}}
	switch {
	case tabled:
		skip[input] = struct{}{}
	case read.pagesByCursor():
		skip[inputFirst], skip[inputAfter] = struct{}{}, struct{}{}
	default:
		return nil, fmt.Errorf("%s: %s takes no table input and no page of its own",
			spec.name, read.Operation)
	}
	params, err := offered(spec.name, read.typedParams(), spec.unoffered)
	if err != nil {
		return nil, err
	}
	if params, err = instantParams(spec.name, params, spec.instants); err != nil {
		return nil, err
	}
	if tabled {
		params = withInputParams(params, inputParams(inputType))
	}
	args, required, search, err := read.rootParams(spec.name, skip)
	if err != nil {
		return nil, err
	}
	params = append(params, args...)
	slices.SortFunc(params, func(a, b graphQLParam) int { return strings.Compare(a.Name, b.Name) })
	instead, err := insteadReads(spec, &read)
	if err != nil {
		return nil, err
	}
	theirs := insteadNames(instead)
	if params, required, err = renamed(spec.name, params, without(spec.names, theirs),
		required); err != nil {
		return nil, err
	}
	if params, err = referenced(spec.name, params,
		without(spec.references, theirs)); err != nil {
		return nil, err
	}
	if params, err = annotate(spec.name, params, without(spec.notes, theirs)); err != nil {
		return nil, err
	}
	for idx := range instead {
		for _, param := range params {
			if param.ArgField == "" && slices.Contains(instead[idx].bound.Keys, param.Arg) {
				instead[idx].shared = append(instead[idx].shared, param)
			}
		}
	}
	bound, err := matchAlongside(spec.name, "alongside", spec.alongside, read.Alongside)
	if err != nil {
		return nil, err
	}

	return &graphQLListTool{
		spec:     *spec,
		read:     read,
		input:    input,
		search:   search,
		params:   params,
		required: required,
		demanded: undefaulted(requiredUnlessInstead(required, instead), spec.defaults),
		sortable: agenttoolschema.Derived(spec.name+".graphqlSortBy", read.sortable()),
		beside: besideReads{
			specs: spec.alongside, bound: bound, reader: reader, permissions: permissions,
		},
		instead:     instead,
		reader:      reader,
		permissions: permissions,
		registry:    fieldsensitivity.DefaultRegistry(),
	}, nil
}

// pagesByCursor reports whether a list's root takes its own first and after,
// as a connection paged without a table input does.
func (r *graphQLRead) pagesByCursor() bool {
	var first, after bool
	for _, variable := range r.Variables {
		first = first || (variable.Name == inputFirst && variable.Type == graphQLTypeInt)
		after = after || (variable.Name == inputAfter && variable.Type == graphQLTypeString)
	}

	return first && after
}

// insteadReads pairs each read a spec makes instead of its list's with the
// one the generator proved, and derives the parameters of its own a call
// chooses it by, named and typed by the spec as the list's are.
func insteadReads(spec *graphQLListSpec, read *graphQLRead) ([]insteadRead, error) {
	bound, err := matchAlongside(spec.name, "instead", spec.instead, read.Instead)
	if err != nil {
		return nil, err
	}
	out := make([]insteadRead, 0, len(bound))
	for idx := range bound {
		instead := insteadRead{spec: &spec.instead[idx], bound: bound[idx]}
		for _, variable := range bound[idx].Variables {
			if !slices.Contains(bound[idx].Own, variable.Name) {
				continue
			}
			derived, deriveErr := variableParams(spec.name, bound[idx].Operation, &variable)
			if deriveErr != nil {
				return nil, deriveErr
			}
			instead.params = append(instead.params, derived...)
		}
		for _, param := range instead.params {
			instead.derived = append(instead.derived, param.Name)
		}
		if instead.params, _, err = renamed(spec.name, instead.params,
			only(spec.names, instead.params), nil); err != nil {
			return nil, err
		}
		if instead.params, err = referenced(spec.name, instead.params,
			only(spec.references, instead.params)); err != nil {
			return nil, err
		}
		if instead.params, err = annotate(spec.name, instead.params,
			only(spec.notes, instead.params)); err != nil {
			return nil, err
		}
		out = append(out, instead)
	}

	return out, nil
}

// insteadNames are every name the reads made instead of a list's take their
// own parameters by, derived or given.
func insteadNames(instead []insteadRead) map[string]struct{} {
	out := make(map[string]struct{})
	for idx := range instead {
		read := &instead[idx]
		for _, param := range read.params {
			out[param.Name] = struct{}{}
		}
		for _, name := range read.derived {
			out[name] = struct{}{}
		}
	}

	return out
}

// without is a spec's map less the keys another read's parameters take.
func without[V any](m map[string]V, taken map[string]struct{}) map[string]V {
	out := make(map[string]V, len(m))
	for key, value := range m {
		if _, theirs := taken[key]; !theirs {
			out[key] = value
		}
	}

	return out
}

// only is a spec's map narrowed to the keys a read's parameters take.
func only[V any](m map[string]V, params []graphQLParam) map[string]V {
	out := make(map[string]V)
	for _, param := range params {
		if value, ok := m[param.Name]; ok {
			out[param.Name] = value
		}
	}

	return out
}

// requiredUnlessInstead are the list's required arguments that a read made
// instead of it requires too: a call that chooses the other read need not
// name one the other read can do without.
func requiredUnlessInstead(required []string, instead []insteadRead) []string {
	if len(instead) == 0 {
		return required
	}
	out := make([]string, 0, len(required))
	for _, name := range required {
		every := true
		for idx := range instead {
			read := &instead[idx]
			every = every && slices.ContainsFunc(read.shared, func(p graphQLParam) bool {
				return p.Name == name && slices.ContainsFunc(read.bound.Variables,
					func(v graphQLVariable) bool { return v.Name == p.Arg && v.NonNull })
			})
		}
		if every {
			out = append(out, name)
		}
	}

	return out
}

// offered are a record's parameters less those of the fields a spec leaves
// out, refusing a field that is none of its filters.
func offered(tool string, params []graphQLParam, unoffered []string) ([]graphQLParam, error) {
	for _, field := range unoffered {
		if !slices.ContainsFunc(params, func(p graphQLParam) bool { return p.Field == field }) {
			return nil, fmt.Errorf("%s leaves out %q, which is not one of its filters",
				tool, field)
		}
	}

	return slices.DeleteFunc(params, func(param graphQLParam) bool {
		return slices.Contains(unoffered, param.Field)
	}), nil
}

// undefaulted are the required arguments a call must name itself: one the
// spec defaults (a transfer's direction, Inbound unless named) is sent for a
// call that leaves it out, so the schema does not require it.
func undefaulted(required []string, defaults map[string]any) []string {
	if len(defaults) == 0 {
		return required
	}

	return slices.DeleteFunc(slices.Clone(required), func(name string) bool {
		_, defaulted := defaults[name]
		return defaulted
	})
}

// referenced types each id parameter by the resource its spec names, drops
// the ones it names none for, and refuses a reference to no id parameter.
func referenced(
	tool string,
	params []graphQLParam,
	references map[string]idReference,
) ([]graphQLParam, error) {
	out := make([]graphQLParam, 0, len(params))
	used := 0
	for _, param := range params {
		if param.Kind != paramReference && param.Kind != paramReferences {
			out = append(out, param)
			continue
		}
		ref, ok := references[param.Name]
		if !ok {
			continue
		}
		param.Schema = ref.schema(param.Kind == paramReferences)
		out = append(out, param)
		used++
	}
	if used == len(references) {
		return out, nil
	}
	for name := range references {
		if !slices.ContainsFunc(out, func(p graphQLParam) bool { return p.Name == name }) {
			return nil, fmt.Errorf("%s references %q, which is not one of its id parameters",
				tool, name)
		}
	}

	return out, nil
}

// annotate appends each note to its parameter's description, and refuses a
// note for a parameter the tool does not take.
func annotate(
	tool string,
	params []graphQLParam,
	notes map[string]string,
) ([]graphQLParam, error) {
	matched := 0
	for idx := range params {
		note, ok := notes[params[idx].Name]
		if !ok {
			continue
		}
		description, _ := params[idx].Schema[toolschema.KeyDescription].(string)
		params[idx].Schema = withDescription(params[idx].Schema, description+" "+note)
		matched++
	}
	if matched == len(notes) {
		return params, nil
	}
	for name := range notes {
		if !slices.ContainsFunc(params, func(p graphQLParam) bool { return p.Name == name }) {
			return nil, fmt.Errorf("%s notes %q, which is not one of its parameters", tool, name)
		}
	}

	return params, nil
}

// graphQLListSpecs are every list tool, module by module, that reads
// through GraphQL by a spec alone.
func graphQLListSpecs() []graphQLListSpec {
	return slices.Concat(
		referenceReadSpecs(), partyReadSpecs(), equipmentReadSpecs(), workerReadSpecs(),
		trainingReadSpecs(), timeOffReadSpecs(), workerRecordReadSpecs(), accountingReadSpecs(),
	)
}

// graphQLListProviders registers each spec as its own list tool.
func graphQLListProviders() []any {
	specs := graphQLListSpecs()
	providers := make([]any, 0, len(specs))
	for idx := range specs {
		spec := &specs[idx]
		providers = append(providers, func(
			reader agentports.GraphQLReader,
			permissions permissionports.PermissionEngine,
		) (agentports.AgentQueryTool, error) {
			return newGraphQLReadTool(spec, reader, permissions)
		})
	}

	return providers
}

func (t *graphQLListTool) Name() string { return t.spec.name }

// Description is the summary and every typed parameter an agent at the
// default data access may use, by name, so a model choosing a tool sees what
// it can narrow the list by. A range is named once,
// by its lower end (createdAtFrom/To, amountMin/Max), and a nested record's
// parameters are named once under it.
func (t *graphQLListTool) Description() string {
	own := make([]string, 0, len(t.params))
	nested := make(map[string][]string)
	var holders []string
	for _, param := range t.params {
		name, ok := describedName(param)
		if !ok || !t.describable(param) {
			continue
		}
		parent, _, isNested := strings.Cut(param.Field, ".")
		if !isNested {
			own = append(own, name)
			continue
		}
		if _, seen := nested[parent]; !seen {
			holders = append(holders, parent)
		}
		nested[parent] = append(nested[parent],
			stringutils.LowerFirst(strings.TrimPrefix(name, parent)))
	}
	if len(own) == 0 && len(holders) == 0 {
		return t.spec.summary
	}

	var b strings.Builder
	b.WriteString(t.spec.summary)
	b.WriteString(" Filters: ")
	b.WriteString(strings.Join(own, ", "))
	for _, parent := range holders {
		fmt.Fprintf(&b, "; %s fields, each named %s and the field (%s%s): %s",
			parent, parent, parent, stringutils.CapitalizeFirst(nested[parent][0]),
			strings.Join(nested[parent], ", "))
	}
	b.WriteString(".")

	return b.String()
}

// applyDefaults is a call's arguments with the spec's defaults for those it
// omits, and its day-dependent defaults when it omits all of them.
func applyDefaults(spec *graphQLListSpec, args map[string]any, clk clock) map[string]any {
	var dated map[string]any
	if spec.defaultsAt != nil {
		dated = spec.defaultsAt(clk)
		for name := range dated {
			if raw, given := args[name]; given && raw != nil {
				dated = nil
				break
			}
		}
	}
	var unscoped map[string]any
	if !namesReference(spec, args) {
		unscoped = spec.unscoped
	}
	if len(spec.defaults) == 0 && len(dated) == 0 && len(unscoped) == 0 {
		return args
	}
	out := make(map[string]any, len(args)+len(spec.defaults)+len(dated)+len(unscoped))
	for name, value := range spec.defaults {
		out[name] = value
	}
	for name, value := range unscoped {
		out[name] = value
	}
	for name, value := range dated {
		out[name] = value
	}
	for name, value := range args {
		out[name] = value
	}

	return out
}

// namesReference reports whether a call names any record its spec's
// references point to (the worker whose events are read).
func namesReference(spec *graphQLListSpec, args map[string]any) bool {
	for name := range spec.references {
		if raw, given := args[name]; given && raw != nil && raw != "" {
			return true
		}
	}

	return false
}

// describable reports whether a parameter's field can be filtered at the
// default agent data access. A description names only those: the schema
// still offers every parameter, and a reader above the default may use one
// the description leaves out.
func (t *graphQLListTool) describable(param graphQLParam) bool {
	if param.OnInput {
		return true
	}
	resource, name := t.fieldOwner(param.Field)

	return fieldsensitivity.Visible(t.registry, resource, name, permission.SensitivityInternal)
}

// describedName is how a description names a parameter: a range once, by
// its lower end.
func describedName(param graphQLParam) (string, bool) {
	switch param.Kind {
	case paramDayTo, paramAtMost, paramInstantTo:
		return "", false
	case paramDayFrom, paramInstantFrom:
		return param.Name + "/" + suffixTo, true
	case paramAtLeast:
		return param.Name + "/" + suffixMax, true
	case paramIn, paramEquals, paramContains, paramPresent, paramReference, paramReferences:
		return param.Name, true
	}

	return param.Name, true
}

func (t *graphQLListTool) SearchTerms() []string { return t.spec.searchTerms }

func (t *graphQLListTool) Policy() agentports.ToolPolicy {
	spec := t.spec.reads
	spec.resource = t.spec.resource

	return readPolicy(t.Name(), spec)
}

func (t *graphQLListTool) ParamSchema() map[string]any {
	properties := make(map[string]any, len(t.params)+5)
	if t.input != "" || t.search[0] != "" {
		properties[graphQLParamQuery] = map[string]any{
			toolschema.KeyType:        toolschema.TypeString,
			toolschema.KeyDescription: t.queryDescription(),
		}
	}
	for _, param := range t.params {
		properties[param.Name] = param.Schema
	}
	for idx := range t.instead {
		for _, param := range t.instead[idx].params {
			properties[param.Name] = param.Schema
		}
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

	return argumentSchema(properties, t.demanded)
}

func (t *graphQLListTool) queryDescription() string {
	if t.spec.queryNote != "" {
		return t.spec.queryNote
	}

	return "Optional free text matched across the searchable columns. Omit it when you are " +
		"narrowing by a field."
}

// chooseInstead is the read made in place of the list's own that a call
// chooses, by naming one of its own parameters.
func (t *graphQLListTool) chooseInstead(args map[string]any) (*insteadRead, bool) {
	for idx := range t.instead {
		for _, param := range t.instead[idx].params {
			if raw, given := args[param.Name]; given && raw != nil && raw != "" {
				return &t.instead[idx], true
			}
		}
	}

	return nil, false
}

// readInstead reads what the call chose in place of the list's own records,
// refusing any of the list's filters, which do not apply to it.
func (t *graphQLListTool) readInstead(
	ctx context.Context,
	params *agentports.QueryToolParams,
	instead *insteadRead,
	clk clock,
) (any, error) {
	keys := slices.Concat(instead.params, instead.shared)
	chosen := instead.params[0].Name
	for name, raw := range params.Params {
		if raw == nil || raw == "" || name == paramLimit ||
			slices.ContainsFunc(keys, func(p graphQLParam) bool { return p.Name == name }) {
			continue
		}
		return nil, fmt.Errorf("parameter %q lists records and does not apply to %q, which "+
			"reads %s instead; leave one of them out", name, chosen, instead.spec.what)
	}
	if err := checkReferences(keys, params.Params); err != nil {
		return nil, err
	}
	variables, err := compileArgs(keys, params.Params, clk)
	if err != nil {
		return nil, err
	}
	resource := permission.Resource(instead.bound.Resource)
	result, err := t.reader.Read(ctx, &agentports.GraphQLReadRequest{
		Actor:             params.Actor,
		Operation:         instead.bound.Operation,
		Variables:         variables,
		DataAccessCeiling: params.DataAccessCeiling,
		RequestID:         params.RunID.String(),
	})
	if err != nil {
		return nil, err
	}
	if err = readFailure(result, resource); err != nil {
		return nil, err
	}

	value, prefix, _ := besideValue(&instead.bound, instead.spec, result.Data)
	outcome := map[string]any{instead.spec.key: value}
	if rows, ok := value.([]map[string]any); ok {
		outcome[outcomeCount] = len(rows)
	}
	if instead.spec.note != "" {
		outcome[outcomeNote] = instead.spec.note
	}
	if withheld := relativePaths(result.Withheld, prefix); len(withheld) > 0 {
		outcome[outcomeWithheld] = withheld
		outcome[outcomeWithheldNote] = withheldNote
	}

	return outcome, nil
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
	Total            *int             `json:"total,omitempty"`
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
	if instead, chosen := t.chooseInstead(params.Params); chosen {
		return t.readInstead(ctx, params, instead, clk)
	}
	if err := t.refuseWithheld(ctx, params); err != nil {
		return nil, err
	}

	args := applyDefaults(&t.spec, params.Params, clk)
	if err := requireArguments(t.required, args); err != nil {
		return nil, err
	}
	if err := checkReferences(t.params, args); err != nil {
		return nil, err
	}
	filters, err := compileFilters(t.params, args, clk)
	if err != nil {
		return nil, err
	}
	inputs, err := compileInputs(t.params, args, clk)
	if err != nil {
		return nil, err
	}
	rootArgs, err := compileArgs(t.params, args, clk)
	if err != nil {
		return nil, err
	}
	window, err := readCursorWindow(t.Name(), params.Params, graphQLListLimit, graphQLListMost)
	if err != nil {
		return nil, err
	}
	query := strings.TrimSpace(optionalString(params.Params, graphQLParamQuery))
	criteria := describeCriteria(t.spec.entityPlural, t.params, args, query)

	page, err := t.page(ctx, params, &pageRequest{
		query:   query,
		inputs:  inputs,
		args:    rootArgs,
		filters: filters,
		sort:    t.sort(params.Params),
		limit:   window.limit,
		after:   window.info.After,
		total:   params.IncludeTotal,
	})
	if err != nil {
		return nil, err
	}
	beside, besideWithheld, besideNotes, err := t.beside.read(
		ctx, params, rootArgs, namesReference(&t.spec, args),
	)
	if err != nil {
		return nil, err
	}
	if t.spec.combine != nil {
		t.spec.combine(page.items, beside)
	}
	page.withheld = append(page.withheld, besideWithheld...)

	if t.spec.rank != nil && query != "" && optionalString(params.Params, paramSortBy) == "" {
		page.items = t.spec.rank(query, page.items)
	}

	outcome := graphQLListOutcome{
		Count:            len(page.items),
		SearchedFor:      criteria.Describe(),
		Items:            page.items,
		Columns:          selectedColumns(t.read.Fields, page.items),
		HasMore:          page.hasMore,
		Total:            page.total,
		WithheldByAccess: page.withheld,
	}
	if page.hasMore {
		outcome.NextCursor = window.next(page.endCursor)
	}
	if len(page.withheld) > 0 {
		outcome.WithheldNote = withheldNote
	}
	notes := besideNotes
	if len(page.items) == 0 && page.first {
		note := criteria.EmptyNote()
		if near := t.nearMiss(ctx, params, query, inputs, rootArgs, filters); near != "" {
			note = near
		}
		notes = append([]string{note}, notes...)
	}
	outcome.Note = strings.Join(notes, " ")
	if len(beside) == 0 {
		return outcome, nil
	}

	return withBeside(&outcome, beside)
}

// withBeside is a page's outcome with the records read alongside it, each
// under its key.
func withBeside(outcome *graphQLListOutcome, beside map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(beside)+10)
	encoded, err := sonic.Marshal(outcome)
	if err != nil {
		return nil, fmt.Errorf("encode the page: %w", err)
	}
	if err = sonic.Unmarshal(encoded, &out); err != nil {
		return nil, fmt.Errorf("decode the page: %w", err)
	}
	for key, value := range beside {
		out[key] = value
	}

	return out, nil
}

type pageRequest struct {
	query  string
	inputs map[string]any
	// args are the root variables the call's argument parameters set.
	args    map[string]any
	filters []map[string]any
	sort    []map[string]any
	limit   int
	after   string
	total   bool
}

type graphQLPage struct {
	items     []map[string]any
	hasMore   bool
	endCursor string
	withheld  []string
	first     bool
	total     *int
}

func (t *graphQLListTool) page(
	ctx context.Context,
	params *agentports.QueryToolParams,
	req *pageRequest,
) (*graphQLPage, error) {
	result, err := t.reader.Read(ctx, &agentports.GraphQLReadRequest{
		Actor:             params.Actor,
		Operation:         t.read.Operation,
		Variables:         t.variables(req),
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
	if t.spec.shape != nil {
		withheld := make(map[string]struct{}, len(page.withheld))
		for _, path := range page.withheld {
			withheld[path] = struct{}{}
		}
		clk := clockFor(params)
		for _, item := range page.items {
			t.spec.shape(item, withheld, clk)
		}
	}
	if info, ok := connection[connectionPageInfo].(map[string]any); ok {
		page.hasMore, _ = info[pageHasNext].(bool)
		page.endCursor, _ = info[pageEndCursor].(string)
	}
	if req.total {
		if total, counted := typeutils.IntOf(connection[connectionTotalCount]); counted {
			page.total = &total
		}
	}

	return page, nil
}

// variables are what one page's read sends: the root arguments, and the
// table input or the root's own page, with the free text where the list
// takes it.
func (t *graphQLListTool) variables(req *pageRequest) map[string]any {
	variables := make(map[string]any, len(req.args)+4)
	for name, value := range req.args {
		if filter, ok := value.(map[string]any); ok {
			value = maps.Clone(filter)
		}
		variables[name] = value
	}
	if t.input == "" {
		t.rootPage(variables, req)
		return variables
	}

	input := make(map[string]any, len(t.spec.inputs)+len(req.inputs)+5)
	for name, value := range t.spec.inputs {
		input[name] = value
	}
	for name, value := range req.inputs {
		input[name] = value
	}
	input[inputFirst] = req.limit
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
	variables[t.input] = input
	variables[includeTotalCount] = req.total

	return variables
}

// rootPage sets the page of a list paged by its root's own first and after,
// and its free text inside the filter input that takes it.
func (t *graphQLListTool) rootPage(variables map[string]any, req *pageRequest) {
	variables[inputFirst] = req.limit
	if req.after != "" {
		variables[inputAfter] = req.after
	}
	if req.query != "" {
		filter, _ := variables[t.search[0]].(map[string]any)
		if filter == nil {
			filter = make(map[string]any, 1)
		}
		filter[t.search[1]] = req.query
		variables[t.search[0]] = filter
	}
	if t.read.declares(includeTotalCount) {
		variables[includeTotalCount] = req.total
	}
}

// declares reports whether the read's operation takes a variable.
func (r *graphQLRead) declares(name string) bool {
	return slices.ContainsFunc(r.Variables, func(v graphQLVariable) bool { return v.Name == name })
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
	return relativePaths(paths,
		t.read.RootField+"."+connectionEdges+"."+connectionNode+".")
}

// compact drops what tells a model nothing: nulls, empty lists and type
// names, in the record and in each record it carries.
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
			for idx, element := range v {
				if object, ok := element.(map[string]any); ok {
					v[idx] = compact(object)
				}
			}
		case map[string]any:
			value = compact(v)
		}
		out[key] = value
	}

	return out
}

// selectedColumns are the fields the operation selects on the record, in
// schema order, that the rows carry.
func selectedColumns(fields []graphQLField, items []map[string]any) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
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

// refuseWithheld refuses a filter or an order on a field the caller may not
// be shown, before anything is read: filtering on a hidden value would leak
// it one row at a time.
func (t *graphQLListTool) refuseWithheld(
	ctx context.Context,
	params *agentports.QueryToolParams,
) error {
	fields := make([]string, 0, len(t.params)+1)
	for _, param := range t.params {
		if raw, present := params.Params[param.Name]; present && raw != nil && !param.OnInput {
			fields = append(fields, param.Field)
		}
	}
	if sortBy := optionalString(params.Params, paramSortBy); sortBy != "" {
		fields = append(fields, sortBy)
	}

	return t.refuseFields(ctx, params, fields)
}

// refuseFields refuses the first of fields the caller may not be shown.
func (t *graphQLListTool) refuseFields(
	ctx context.Context,
	params *agentports.QueryToolParams,
	fields []string,
) error {
	if len(fields) == 0 {
		return nil
	}

	ceilings := fieldsensitivity.NewAgentCeilings(
		t.permissions, params.Actor, params.DataAccessCeiling,
	)
	for _, field := range fields {
		resource, name := t.fieldOwner(field)
		hidden := resource != t.spec.resource && !ceilings.Reads(ctx, resource)
		if hidden || !fieldsensitivity.Visible(t.registry, resource, name,
			ceilings.For(ctx, resource)) {
			return fmt.Errorf(
				"%s cannot be filtered or ordered by %s: its values are hidden at your access",
				t.spec.name, field)
		}
	}

	return nil
}

// fieldOwner is the resource whose sensitivities a filtered field is judged
// by, and the field's name there, as the field guard judges it: a nested
// record of a registered kind (a worker's profile) by its own kind, any
// other by the list's.
func (t *graphQLListTool) fieldOwner(field string) (resource permission.Resource, name string) {
	parent, child, ok := t.read.nestedField(field)
	if !ok {
		return t.spec.resource, field
	}
	kind := stringutils.ConvertCamelToSnake(parent.Type)
	if t.registry.HasResource(kind) {
		return permission.Resource(kind), child.Name
	}

	return t.spec.resource, child.Name
}

// nearMiss reads again with one condition dropped at a time, and says which
// one, alone, kept every row out.
func (t *graphQLListTool) nearMiss(
	ctx context.Context,
	params *agentports.QueryToolParams,
	query string,
	inputs, args map[string]any,
	filters []map[string]any,
) string {
	probes := 0
	if query != "" {
		probes++
		if page, err := t.page(ctx, params, &pageRequest{
			inputs: inputs, args: args, filters: filters, limit: 1,
		}); err == nil &&
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
		page, err := t.page(ctx, params, &pageRequest{
			query: query, inputs: inputs, args: args, filters: rest, limit: 1,
		})
		if err == nil && len(page.items) > 0 {
			return t.keptOut(filters[idx][paramField])
		}
	}
	for _, name := range slices.Sorted(maps.Keys(inputs)) {
		if _, toggle := inputs[name].(bool); toggle {
			continue
		}
		if probes >= maxNearMissProbes {
			break
		}
		probes++
		rest := maps.Clone(inputs)
		delete(rest, name)
		page, err := t.page(ctx, params, &pageRequest{
			query: query, inputs: rest, args: args, filters: filters, limit: 1,
		})
		if err == nil && len(page.items) > 0 {
			return t.keptOut(name)
		}
	}
	for _, param := range t.params {
		filter, _ := args[param.Arg].(map[string]any)
		value, set := filter[param.ArgField]
		if _, toggle := value.(bool); param.ArgField == "" || !set || toggle {
			continue
		}
		if probes >= maxNearMissProbes {
			break
		}
		probes++
		rest := maps.Clone(args)
		narrower := maps.Clone(filter)
		delete(narrower, param.ArgField)
		rest[param.Arg] = narrower
		page, err := t.page(ctx, params, &pageRequest{
			query: query, inputs: inputs, args: rest, filters: filters, limit: 1,
		})
		if err == nil && len(page.items) > 0 {
			return t.keptOut(param.Name)
		}
	}

	return ""
}

func (t *graphQLListTool) keptOut(condition any) string {
	return fmt.Sprintf("No %s matched every condition. Without the condition on %v "+
		"some do, so that one alone keeps them out.", t.spec.entityPlural, condition)
}

// runView reads a composed table view through the same operation, so the
// count and rows a view previews are what listing the table shows the same
// person, and refuses a view that filters or orders by a field that person
// may not be shown, as a call is refused.
func (t *graphQLListTool) runView(
	ctx context.Context,
	params *agentports.QueryToolParams,
	query string,
	filters []domaintypes.FieldFilter,
	sorting []domaintypes.SortField,
	limit int,
) (*viewRun, error) {
	if t.input == "" || len(t.required) > 0 {
		return nil, fmt.Errorf("%s reads by its own arguments; a table view cannot be "+
			"composed over it", t.spec.name)
	}
	fields := make([]string, 0, len(filters)+len(sorting))
	for _, filter := range filters {
		fields = append(fields, filter.Field)
	}
	for _, field := range sorting {
		fields = append(fields, field.Field)
	}
	if err := t.refuseFields(ctx, params, fields); err != nil {
		return nil, err
	}

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
			Columns:     selectedColumns(t.read.Fields, page.items),
		},
		withheld: page.withheld,
		more:     page.hasMore,
	}, nil
}
