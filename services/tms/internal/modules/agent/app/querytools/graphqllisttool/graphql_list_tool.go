package graphqllisttool

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/textcase"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	connectionCursor     = "cursor"
	connectionTotalCount = "totalCount"
	connectionTotalExact = "totalCountExact"
	pageEndCursor        = "endCursor"

	IncludeTotalCount = "includeTotalCount"

	// maxNearMissProbes bounds the reads an empty first page may spend asking
	// which condition to loosen; each is charged to the caller's budget.
	maxNearMissProbes = 4
	nearMissSample    = 5
)

type Tool struct {
	Spec  readspec.List
	Read  graphqlread.Read
	Input string
	// search is the filter input variable and field a list paged by its
	// root's own first and after takes its free text in.
	search [2]string
	// searched are the fields the query matches: the record's search
	// configuration's, which the repository searches with, and the spec's.
	Searched []string
	// pageIn is the input a list paged inside its own input (first and
	// after on InboundMessagesInput) takes its page in.
	pageIn string
	Params []graphqlread.Param
	// required are the arguments a call that lists records must name, and
	// demanded those the schema marks required: an argument a read made
	// instead can do without is required only when listing.
	Required    []string
	demanded    []string
	sortable    agenttoolschema.EnumSource[string]
	beside      graphqlread.BesideReads
	instead     []insteadRead
	reader      graphqlreadports.GraphQLReader
	permissions permissionports.PermissionEngine
	registry    *permissioncatalog.Registry
	rowFields   []graphqlread.Field
	columns     agenttoolschema.EnumSource[string]
}

// insteadRead is a read a list makes in place of its own, and the
// parameters of its own a call chooses it by.
type insteadRead struct {
	spec   *readspec.Alongside
	bound  graphqlread.Alongside
	params []graphqlread.Param
	// shared are the list's own parameters the read is keyed by too.
	shared []graphqlread.Param
	// derived are its parameters' names before the spec named them.
	derived []string
}

func New(
	spec *readspec.List,
	reader graphqlreadports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (*Tool, error) {
	read, ok := graphqlread.Reads[spec.Name]
	if !ok {
		return nil, fmt.Errorf("%s: %w", spec.Name, graphqlread.ErrUnbound)
	}
	if read.Verb != graphqlread.VerbList {
		return nil, fmt.Errorf("%s reads %s, which is not a list", spec.Name, read.Operation)
	}
	input, inputType, tabled := read.TableInput()
	skip, err := pageSkips(spec.Name, &read, input, tabled)
	if err != nil {
		return nil, err
	}
	if err = searchedFields(spec.Name, read.Fields, spec.Searches); err != nil {
		return nil, err
	}
	params, err := offeredParams(spec, &read, inputType, tabled)
	if err != nil {
		return nil, err
	}
	args, required, search, err := read.RootParams(spec.Name, skip)
	if err != nil {
		return nil, err
	}
	params = graphqlread.WithRootArgs(params, args)
	slices.SortFunc(
		params,
		func(a, b graphqlread.Param) int { return strings.Compare(a.Name, b.Name) },
	)
	instead, err := insteadReads(spec, &read)
	if err != nil {
		return nil, err
	}
	if params, required, err = namedParams(
		spec,
		params,
		required,
		insteadNames(instead),
	); err != nil {
		return nil, err
	}
	shareInsteadKeys(instead, params)
	bound, err := graphqlread.MatchAlongside(spec.Name, "alongside", spec.Alongside, read.Alongside)
	if err != nil {
		return nil, err
	}

	if err = checkConditions(spec, tabled, params); err != nil {
		return nil, err
	}

	tool := &Tool{
		Spec:     *spec,
		Read:     read,
		Input:    input,
		search:   search,
		pageIn:   read.PagedInput(),
		Params:   params,
		Required: required,
		demanded: undefaulted(requiredUnlessInstead(required, instead), spec.Defaults),
		sortable: agenttoolschema.Derived(spec.Name+".graphqlSortBy", read.Sortable()),
		beside: graphqlread.BesideReads{
			Specs: spec.Alongside, Bound: bound, Reader: reader, Permissions: permissions,
		},
		instead:     instead,
		reader:      reader,
		permissions: permissions,
		registry:    fieldsensitivity.DefaultRegistry(),
		rowFields:   presentedFields(read.Fields, spec.Presents),
	}
	tool.columns = agenttoolschema.Derived(spec.Name+".column",
		append(graphqlread.ListColumns(tool.rowFields, spec.Amounts), spec.Derived...))
	tool.Params = slices.DeleteFunc(params, func(param graphqlread.Param) bool {
		return tool.confidential(&param)
	})
	tool.Searched = searchedOf(&read, input, spec.Searches)

	return tool, nil
}

func pageSkips(
	name string,
	read *graphqlread.Read,
	input string,
	tabled bool,
) (map[string]struct{}, error) {
	skip := map[string]struct{}{IncludeTotalCount: {}}
	switch {
	case tabled:
		skip[input] = struct{}{}
	case read.PagesByCursor():
		skip[graphqlread.InputFirst], skip[graphqlread.InputAfter] = struct{}{}, struct{}{}
	case read.PagedInput() != "":
	default:
		return nil, fmt.Errorf("%s: %s takes no table input and no page of its own",
			name, read.Operation)
	}

	return skip, nil
}

func offeredParams(
	spec *readspec.List,
	read *graphqlread.Read,
	inputType string,
	tabled bool,
) ([]graphqlread.Param, error) {
	params, err := Offered(spec.Name, read.TypedParams(), spec.Unoffered)
	if err != nil {
		return nil, err
	}
	if params, err = graphqlread.InstantParams(spec.Name, params, spec.Instants); err != nil {
		return nil, err
	}
	if spec.Amounts {
		params = graphqlread.AmountParams(params)
	}
	if tabled {
		params = graphqlread.WithInputParams(params, graphqlread.InputParams(inputType))
	}

	return params, nil
}

func namedParams(
	spec *readspec.List,
	params []graphqlread.Param,
	required []string,
	theirs map[string]struct{},
) ([]graphqlread.Param, []string, error) {
	params, required, err := graphqlread.Renamed(
		spec.Name,
		params,
		without(spec.Names, theirs),
		required,
	)
	if err != nil {
		return nil, nil, err
	}
	if params, err = graphqlread.Referenced(spec.Name, params,
		without(spec.References, theirs)); err != nil {
		return nil, nil, err
	}
	if params, err = graphqlread.Annotate(
		spec.Name,
		params,
		without(spec.Notes, theirs),
	); err != nil {
		return nil, nil, err
	}

	return params, required, nil
}

func shareInsteadKeys(instead []insteadRead, params []graphqlread.Param) {
	for idx := range instead {
		for _, param := range params {
			if param.ArgField == "" && slices.Contains(instead[idx].bound.Keys, param.Arg) {
				instead[idx].shared = append(instead[idx].shared, param)
			}
		}
	}
}

func searchedOf(read *graphqlread.Read, input string, searches []string) []string {
	var searched []string
	if input != "" {
		searched = read.ConfiguredSearch()
	}
	for _, name := range searches {
		if !slices.Contains(searched, name) {
			searched = append(searched, name)
		}
	}

	return searched
}

func (t *Tool) conditionFilters(
	args map[string]any,
) (filters, groups []map[string]any, err error) {
	for idx := range t.Spec.Conditions {
		condition := &t.Spec.Conditions[idx]
		raw, present := args[condition.Name]
		if !present || raw == nil {
			continue
		}
		yes, ok := raw.(bool)
		if !ok {
			return nil, nil, fmt.Errorf("parameter %q must be true or false", condition.Name)
		}
		all, oneOf := condition.Compile(yes)
		filters = append(filters, all...)
		groups = append(groups, oneOf...)
	}

	return filters, groups, nil
}

func checkConditions(spec *readspec.List, tabled bool, params []graphqlread.Param) error {
	if len(spec.Conditions) == 0 {
		return nil
	}
	if !tabled {
		return fmt.Errorf("%s: conditions need a table input to filter", spec.Name)
	}
	for idx := range spec.Conditions {
		name := spec.Conditions[idx].Name
		if slices.ContainsFunc(params, func(p graphqlread.Param) bool { return p.Name == name }) {
			return fmt.Errorf("%s: condition %s is already a parameter", spec.Name, name)
		}
		if len(spec.Conditions[idx].When) == 0 || len(spec.Conditions[idx].Unless) == 0 {
			return fmt.Errorf("%s: condition %s says neither what yes nor no keeps",
				spec.Name, name)
		}
	}

	return nil
}

// confidential marks a parameter on a Confidential field, which no reader's
// data access reaches (a card provider's own card id): offering it would
// only lead to a refusal.
func (t *Tool) confidential(param *graphqlread.Param) bool {
	if param.OnInput {
		return false
	}
	resource, name := t.FieldOwner(param.Field)

	return !fieldsensitivity.Visible(t.registry, resource, name, permission.SensitivityRestricted)
}

// insteadReads pairs each read a spec makes instead of its list's with the
// one the generator proved, and derives the parameters of its own a call
// chooses it by, named and typed by the spec as the list's are.
func insteadReads(spec *readspec.List, read *graphqlread.Read) ([]insteadRead, error) {
	bound, err := graphqlread.MatchAlongside(spec.Name, "instead", spec.Instead, read.Instead)
	if err != nil {
		return nil, err
	}
	out := make([]insteadRead, 0, len(bound))
	for idx := range bound {
		instead := insteadRead{spec: &spec.Instead[idx], bound: bound[idx]}
		for _, variable := range bound[idx].Variables {
			if !slices.Contains(bound[idx].Own, variable.Name) {
				continue
			}
			derived, deriveErr := graphqlread.VariableParams(
				spec.Name,
				bound[idx].Operation,
				&variable,
			)
			if deriveErr != nil {
				return nil, deriveErr
			}
			instead.params = append(instead.params, derived...)
		}
		for _, param := range instead.params {
			instead.derived = append(instead.derived, param.Name)
		}
		if instead.params, _, err = graphqlread.Renamed(spec.Name, instead.params,
			only(spec.Names, instead.params), nil); err != nil {
			return nil, err
		}
		if instead.params, err = graphqlread.Referenced(spec.Name, instead.params,
			only(spec.References, instead.params)); err != nil {
			return nil, err
		}
		if instead.params, err = graphqlread.Annotate(spec.Name, instead.params,
			only(spec.Notes, instead.params)); err != nil {
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
func only[V any](m map[string]V, params []graphqlread.Param) map[string]V {
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
			every = every && slices.ContainsFunc(read.shared, func(p graphqlread.Param) bool {
				return p.Name == name && slices.ContainsFunc(read.bound.Variables,
					func(v graphqlread.Variable) bool { return v.Name == p.Arg && v.NonNull })
			})
		}
		if every {
			out = append(out, name)
		}
	}

	return out
}

// Offered are a record's parameters less those of the fields a spec leaves
// out, refusing a field that is none of its filters.
func Offered(
	tool string,
	params []graphqlread.Param,
	unoffered []string,
) ([]graphqlread.Param, error) {
	for _, field := range unoffered {
		if !slices.ContainsFunc(
			params,
			func(p graphqlread.Param) bool { return p.Field == field },
		) {
			return nil, fmt.Errorf("%s leaves out %q, which is not one of its filters",
				tool, field)
		}
	}

	return slices.DeleteFunc(params, func(param graphqlread.Param) bool {
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

func (t *Tool) Name() string { return t.Spec.Name }

// Description is the summary and every typed parameter an agent at the
// default data access may use, by name, so a model choosing a tool sees what
// it can narrow the list by. A range is named once,
// by its lower end (createdAtFrom/To, amountMin/Max), and a nested record's
// parameters are named once under it.
func (t *Tool) Description() string {
	own := make([]string, 0, len(t.Params))
	nested := make(map[string][]string)
	var holders []string
	for _, param := range t.Params {
		name, ok := DescribedName(&param)
		if !ok || !t.Describable(&param) {
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
			textcase.LowerFirst(strings.TrimPrefix(name, parent)))
	}
	for idx := range t.Spec.Conditions {
		own = append(own, t.Spec.Conditions[idx].Name)
	}
	if len(own) == 0 && len(holders) == 0 {
		return t.Spec.Summary
	}

	var b strings.Builder
	b.WriteString(t.Spec.Summary)
	b.WriteString(" Filters: ")
	b.WriteString(strings.Join(own, ", "))
	for _, parent := range holders {
		fmt.Fprintf(&b, "; %s fields, each named %s and the field (%s%s): %s",
			parent, parent, parent, textcase.CapitalizeFirst(nested[parent][0]),
			strings.Join(nested[parent], ", "))
	}
	b.WriteString(".")

	return b.String()
}

// applyDefaults is a call's arguments with the spec's defaults for those it
// omits, and its day-dependent defaults when it omits all of them.
func applyDefaults(
	spec *readspec.List,
	args map[string]any,
	clk filtercatalog.Clock,
) map[string]any {
	var dated map[string]any
	if spec.DefaultsAt != nil {
		dated = spec.DefaultsAt(clk)
		for name := range dated {
			if raw, given := args[name]; given && raw != nil {
				dated = nil
				break
			}
		}
	}
	var unscoped map[string]any
	if !namesReference(spec, args) {
		unscoped = spec.Unscoped
	}
	if len(spec.Defaults) == 0 && len(dated) == 0 && len(unscoped) == 0 {
		return args
	}
	out := make(map[string]any, len(args)+len(spec.Defaults)+len(dated)+len(unscoped))
	for name, value := range spec.Defaults {
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
func namesReference(spec *readspec.List, args map[string]any) bool {
	for name := range spec.References {
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
func (t *Tool) Describable(param *graphqlread.Param) bool {
	if param.OnInput {
		return true
	}
	resource, name := t.FieldOwner(param.Field)

	return fieldsensitivity.Visible(t.registry, resource, name, permission.SensitivityInternal)
}

// DescribedName is how a description names a parameter: a range once, by
// its lower end.
func DescribedName(param *graphqlread.Param) (string, bool) {
	switch param.Kind {
	case graphqlread.ParamDayTo, graphqlread.ParamAtMost, graphqlread.ParamInstantTo:
		return "", false
	case graphqlread.ParamDayFrom, graphqlread.ParamInstantFrom:
		return param.Name + "/" + graphqlread.SuffixTo, true
	case graphqlread.ParamAtLeast:
		return param.Name + "/" + graphqlread.SuffixMax, true
	case graphqlread.ParamIn,
		graphqlread.ParamEquals,
		graphqlread.ParamContains,
		graphqlread.ParamPresent,
		graphqlread.ParamReference,
		graphqlread.ParamReferences,
		graphqlread.ParamDayText:
		return param.Name, true
	}

	return param.Name, true
}

func (t *Tool) SearchTerms() []string { return t.Spec.SearchTerms }

func (t *Tool) Policy() agenttoolports.ToolPolicy {
	spec := t.Spec.Reads
	spec.Resource = t.Spec.Resource

	return querytool.ReadPolicy(t.Name(), &spec)
}

func (t *Tool) ParamSchema() map[string]any {
	properties := make(map[string]any, len(t.Params)+5)
	if t.Input != "" || t.search[0] != "" {
		properties[graphqlread.ParamQuery] = map[string]any{
			toolschema.KeyType:        toolschema.TypeString,
			toolschema.KeyDescription: t.queryDescription(),
		}
	}
	for _, param := range t.Params {
		properties[param.Name] = param.Schema
	}
	for idx := range t.Spec.Conditions {
		condition := &t.Spec.Conditions[idx]
		properties[condition.Name] = map[string]any{
			toolschema.KeyType:        toolschema.TypeBoolean,
			toolschema.KeyDescription: condition.Description,
		}
	}
	for idx := range t.instead {
		for _, param := range t.instead[idx].params {
			properties[param.Name] = param.Schema
		}
	}
	if len(t.sortable.Names()) > 0 {
		properties[listtool.ParamSortBy] = agenttoolschema.Enum(
			"Field to order by. Defaults to most recently created.", t.sortable,
		)
		properties[listtool.ParamSortDirection] = agenttoolschema.Enum(
			"asc for oldest or smallest first, desc for newest or largest.",
			listtool.SortDirections,
		)
	}
	for name, property := range querytool.CursorPageSchema(t.pageBounds()) {
		properties[name] = property
	}
	if len(t.columns.Names()) > 1 {
		properties[agenttoolports.ListFieldsParam] = listtool.FieldsSchema(t.columns)
	}

	return graphqlread.ArgumentSchema(properties, t.demanded)
}

// pageBounds are the rows a call reads unless it asks, and the most it may.
func (t *Tool) pageBounds() (limit, most int) {
	if t.Spec.Page[0] > 0 {
		return t.Spec.Page[0], t.Spec.Page[1]
	}

	return querytool.GraphQLListLimit, querytool.GraphQLListMost
}

func (t *Tool) queryDescription() string {
	if t.Spec.QueryNote != "" {
		return t.Spec.QueryNote
	}

	return "Optional free text matched across the searchable columns. Omit it when you are " +
		"narrowing by a field."
}

// chooseInstead is the read made in place of the list's own that a call
// chooses, by naming one of its own parameters.
func (t *Tool) chooseInstead(args map[string]any) (*insteadRead, bool) {
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
func (t *Tool) readInstead(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	instead *insteadRead,
	clk filtercatalog.Clock,
) (any, error) {
	keys := slices.Concat(instead.params, instead.shared)
	chosen := instead.params[0].Name
	for name, raw := range params.Params {
		if raw == nil || raw == "" || name == querytool.ParamLimit ||
			name == agenttoolports.ListFieldsParam ||
			slices.ContainsFunc(keys, func(p graphqlread.Param) bool { return p.Name == name }) {
			continue
		}
		return nil, fmt.Errorf("parameter %q lists records and does not apply to %q, which "+
			"reads %s instead; leave one of them out", name, chosen, instead.spec.What)
	}
	if err := graphqlread.CheckReferences(keys, params.Params); err != nil {
		return nil, err
	}
	variables, err := graphqlread.CompileArgs(keys, params.Params, clk)
	if err != nil {
		return nil, err
	}
	resource := permission.Resource(instead.bound.Resource)
	result, err := t.reader.Read(ctx, &graphqlreadports.GraphQLReadRequest{
		Actor:             params.Actor,
		Operation:         instead.bound.Operation,
		Variables:         variables,
		DataAccessCeiling: params.DataAccessCeiling,
		RequestID:         params.RunID.String(),
	})
	if err != nil {
		return nil, err
	}
	if err = graphqlread.ReadFailure(result, resource); err != nil {
		return nil, err
	}

	value, prefix, _ := graphqlread.BesideValue(&instead.bound, instead.spec, result.Data)
	outcome := map[string]any{instead.spec.Key: value}
	if rows, ok := value.([]map[string]any); ok {
		outcome[graphqlread.OutcomeCount] = len(rows)
	}
	if instead.spec.Note != "" {
		outcome[graphqlread.OutcomeNote] = instead.spec.Note
	}
	if withheld := graphqlread.RelativePaths(result.Withheld, prefix); len(withheld) > 0 {
		outcome[graphqlread.OutcomeWithheld] = withheld
		outcome[graphqlread.OutcomeWithheldNote] = fieldgate.WithheldNote
	}

	return outcome, nil
}

// Outcome is one page of a list read through GraphQL, as the model
// reads it.
type Outcome struct {
	Count            int              `json:"count"`
	SearchedFor      []string         `json:"searchedFor"`
	Items            []map[string]any `json:"items"`
	Columns          []string         `json:"columns,omitempty"`
	Note             string           `json:"note,omitempty"`
	HasMore          bool             `json:"hasMore"`
	NextCursor       string           `json:"nextCursor,omitempty"`
	Total            *int             `json:"total,omitempty"`
	TotalEstimated   bool             `json:"totalEstimated,omitempty"`
	WithheldByAccess []string         `json:"withheldByAccess,omitempty"`
	WithheldNote     string           `json:"withheldNote,omitempty"`

	Outside []agentrecord.RecordRef `json:"-"`
	// cursors are the edge cursors of items, in order, and window the call's
	// cursor scope: what a page cut short resumes from. They are nil when the
	// rows are not in the list's own order, so no cut can resume.
	cursors []string
	window  querytool.CursorWindow
}

//nolint:gocritic // the outcome is passed by value wherever a list returns it
func (o Outcome) PageRows() int { return len(o.Items) }

// CutPage is the page ending after its first rows rows, resuming at the edge
// after them.
//
//nolint:gocritic // the outcome is passed by value wherever a list returns it
func (o Outcome) CutPage(rows int) (any, bool) {
	if rows <= 0 || rows >= len(o.Items) || len(o.cursors) != len(o.Items) ||
		o.cursors[rows-1] == "" {
		return nil, false
	}
	cut := o
	cut.Items = o.Items[:rows]
	cut.cursors = o.cursors[:rows]
	cut.Count = rows
	cut.HasMore = true
	cut.NextCursor = o.window.Next(o.cursors[rows-1])
	cut.Outside = graphqlread.KeptRecords(o.Outside, cut.Items)

	return cut, true
}

// TaintedRecords is on the value: a list returns its outcome by value, as
// its callers and the runtime's taint carrier read it.
//
//nolint:gocritic // the outcome is passed by value wherever a list returns it
func (o Outcome) TaintedRecords() []agentrecord.RecordRef { return o.Outside }

func markedRows(spec *readspec.List, items []map[string]any) []agentrecord.RecordRef {
	out := graphqlread.OutsideRecords(spec.Resource, spec.Outside, items...)
	if spec.Marks == nil {
		return out
	}
	for _, item := range items {
		if ref, marked := spec.Marks(item); marked {
			out = append(out, ref)
		}
	}

	return out
}

func (t *Tool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}
	clk := querytool.ClockFor(params)
	if instead, chosen := t.chooseInstead(params.Params); chosen {
		return t.readInstead(ctx, params, instead, clk)
	}
	if err := t.refuseWithheld(ctx, params); err != nil {
		return nil, err
	}

	args := applyDefaults(&t.Spec, params.Params, clk)
	req, err := t.compileRequest(args, clk)
	if err != nil {
		return nil, err
	}
	fewest, most := t.pageBounds()
	window, err := querytool.ReadCursorWindow(t.Name(), params.Params, fewest, most)
	if err != nil {
		return nil, err
	}
	req.query = strings.TrimSpace(querytool.OptionalString(params.Params, graphqlread.ParamQuery))
	req.sort = t.sort(params.Params)
	req.limit = window.Limit
	req.after = window.Info.After
	req.total = params.IncludeTotal
	criteria := t.criteria(args, req.query)

	page, err := t.page(ctx, params, req)
	if err != nil {
		return nil, err
	}
	beside, besideWithheld, besideNotes, err := t.beside.Read(
		ctx, params, req.args, namesReference(&t.Spec, args),
	)
	if err != nil {
		return nil, err
	}
	if t.Spec.Combine != nil {
		t.Spec.Combine(page.Items, beside)
	}
	page.withheld = append(page.withheld, besideWithheld...)

	outcome := t.outcome(params, page, criteria, &window, req.query)
	notes := besideNotes
	if len(page.Items) == 0 && page.first {
		note := criteria.EmptyNote()
		if near := t.nearMiss(ctx, params, &pageRequest{
			query:   req.query,
			inputs:  req.inputs,
			args:    req.args,
			filters: req.filters,
			groups:  req.groups,
		}); near != "" {
			note = near
		}
		notes = append([]string{note}, notes...)
	}
	outcome.Note = strings.Join(notes, " ")
	if len(beside) == 0 {
		return outcome, nil
	}

	return WithBeside(&outcome, beside)
}

func (t *Tool) compileRequest(
	args map[string]any,
	clk filtercatalog.Clock,
) (*pageRequest, error) {
	if err := graphqlread.RequireArguments(t.Required, args); err != nil {
		return nil, err
	}
	if err := graphqlread.CheckReferences(t.Params, args); err != nil {
		return nil, err
	}
	filters, err := graphqlread.CompileFilters(t.Params, args, clk)
	if err != nil {
		return nil, err
	}
	conditioned, groups, err := t.conditionFilters(args)
	if err != nil {
		return nil, err
	}
	inputs, err := graphqlread.CompileInputs(t.Params, args, clk)
	if err != nil {
		return nil, err
	}
	rootArgs, err := graphqlread.CompileArgs(t.Params, args, clk)
	if err != nil {
		return nil, err
	}

	return &pageRequest{
		inputs:  inputs,
		args:    rootArgs,
		filters: append(filters, conditioned...),
		groups:  groups,
	}, nil
}

func (t *Tool) criteria(args map[string]any, query string) *filtercatalog.Criteria {
	criteria := DescribeCriteria(t.Spec.EntityPlural, t.Params, args, query)
	for idx := range t.Spec.Conditions {
		if raw, present := args[t.Spec.Conditions[idx].Name]; present && raw != nil {
			criteria.Field(t.Spec.Conditions[idx].Name, fmt.Sprint(raw))
		}
	}

	return criteria
}

func (t *Tool) outcome(
	params *agenttoolports.QueryToolParams,
	page *Page,
	criteria *filtercatalog.Criteria,
	window *querytool.CursorWindow,
	query string,
) Outcome {
	cursors := page.cursors
	if t.Spec.Rank != nil && query != "" &&
		querytool.OptionalString(params.Params, listtool.ParamSortBy) == "" {
		page.Items = t.Spec.Rank(query, page.Items)
		cursors = nil
	}

	outcome := Outcome{
		Count:            len(page.Items),
		SearchedFor:      criteria.Describe(),
		Items:            page.Items,
		Columns:          graphqlread.SelectedColumns(t.rowFields, page.Items),
		HasMore:          page.HasMore,
		Total:            page.total,
		TotalEstimated:   page.total != nil && !page.totalExact,
		WithheldByAccess: page.withheld,
		Outside:          markedRows(&t.Spec, page.Items),
		cursors:          cursors,
		window:           *window,
	}
	if page.HasMore {
		outcome.NextCursor = window.Next(page.endCursor)
	}
	if len(page.withheld) > 0 {
		outcome.WithheldNote = fieldgate.WithheldNote
	}

	return outcome
}

// WithBeside is a page's outcome with the records read alongside it, each
// under its key.
func WithBeside(outcome *Outcome, beside map[string]any) (any, error) {
	return &BesidePage{Page: *outcome, beside: beside}, nil
}

// BesidePage is a page with the records read alongside it, encoded as one
// object, which cuts as its page does and keeps what was read beside it.
type BesidePage struct {
	Page   Outcome
	beside map[string]any
}

func (p *BesidePage) MarshalJSON() ([]byte, error) {
	record, err := p.record()
	if err != nil {
		return nil, err
	}

	return sonic.Marshal(record)
}

// record is the page and what was read beside it as one object.
func (p *BesidePage) record() (map[string]any, error) {
	out := make(map[string]any, len(p.beside)+10)
	encoded, err := sonic.Marshal(p.Page)
	if err != nil {
		return nil, fmt.Errorf("encode the page: %w", err)
	}
	if err = sonic.Unmarshal(encoded, &out); err != nil {
		return nil, fmt.Errorf("decode the page: %w", err)
	}
	for key, value := range p.beside {
		out[key] = value
	}

	return out, nil
}

func (p *BesidePage) TaintedRecords() []agentrecord.RecordRef { return p.Page.Outside }

func (p *BesidePage) PageRows() int { return p.Page.PageRows() }

func (p *BesidePage) CutPage(rows int) (any, bool) {
	cut, ok := p.Page.CutPage(rows)
	if !ok {
		return nil, false
	}
	page, _ := cut.(Outcome)

	return &BesidePage{Page: page, beside: p.beside}, true
}

type pageRequest struct {
	query  string
	inputs map[string]any
	// args are the root variables the call's argument parameters set.
	args    map[string]any
	filters []map[string]any
	groups  []map[string]any
	sort    []map[string]any
	limit   int
	after   string
	total   bool
}

type Page struct {
	Items      []map[string]any
	cursors    []string
	HasMore    bool
	endCursor  string
	withheld   []string
	first      bool
	total      *int
	totalExact bool
}

func (t *Tool) page(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	req *pageRequest,
) (*Page, error) {
	result, err := t.reader.Read(ctx, &graphqlreadports.GraphQLReadRequest{
		Actor:             params.Actor,
		Operation:         t.Read.Operation,
		Variables:         t.variables(req),
		DataAccessCeiling: params.DataAccessCeiling,
		RequestID:         params.RunID.String(),
	})
	if err != nil {
		return nil, err
	}
	if err = graphqlread.ReadFailure(result, t.Spec.Resource); err != nil {
		return nil, err
	}

	connection, _ := result.Data[t.Read.RootField].(map[string]any)
	page := &Page{first: req.after == "", withheld: t.relativeWithheld(result.Withheld)}
	edges, _ := connection[graphqlread.ConnectionEdges].([]any)
	page.Items = make([]map[string]any, 0, len(edges))
	page.cursors = make([]string, 0, len(edges))
	for _, edge := range edges {
		object, _ := edge.(map[string]any)
		if node, ok := object[graphqlread.ConnectionNode].(map[string]any); ok {
			page.Items = append(page.Items, presented(graphqlread.Compact(node), t.Spec.Presents))
			cursor, _ := object[connectionCursor].(string)
			page.cursors = append(page.cursors, cursor)
		}
	}
	if t.Spec.Shape != nil {
		withheld := make(map[string]struct{}, len(page.withheld))
		for _, path := range page.withheld {
			withheld[path] = struct{}{}
		}
		clk := querytool.ClockFor(params)
		for _, item := range page.Items {
			t.Spec.Shape(item, withheld, clk)
		}
	}
	if t.Spec.Amounts {
		for _, item := range page.Items {
			graphqlread.PresentAmounts(item)
		}
		page.withheld = graphqlread.PresentedPaths(page.withheld)
	}
	if info, ok := connection[graphqlread.ConnectionPageInfo].(map[string]any); ok {
		page.HasMore, _ = info[graphqlread.PageHasNext].(bool)
		page.endCursor, _ = info[pageEndCursor].(string)
	}
	if req.total {
		if total, counted := typeutils.IntOf(connection[connectionTotalCount]); counted {
			page.total = &total
			page.totalExact, _ = connection[connectionTotalExact].(bool)
		}
	}

	return page, nil
}

// variables are what one page's read sends: the root arguments, and the
// table input or the root's own page, with the free text where the list
// takes it.
func (t *Tool) variables(req *pageRequest) map[string]any {
	variables := make(map[string]any, len(req.args)+4)
	for name, value := range req.args {
		if filter, ok := value.(map[string]any); ok {
			value = maps.Clone(filter)
		}
		variables[name] = value
	}
	if t.Input == "" {
		t.rootPage(variables, req)
		return variables
	}

	input := make(map[string]any, len(t.Spec.Inputs)+len(req.inputs)+5)
	for name, value := range t.Spec.Inputs {
		input[name] = value
	}
	for name, value := range req.inputs {
		input[name] = value
	}
	input[graphqlread.InputFirst] = req.limit
	if req.after != "" {
		input[graphqlread.InputAfter] = req.after
	}
	if req.query != "" {
		input[graphqlread.InputQuery] = req.query
	}
	if len(req.filters) > 0 {
		input[graphqlread.InputFieldFilters] = req.filters
	}
	if len(req.groups) > 0 {
		input[graphqlread.InputFilterGroups] = req.groups
	}
	if len(req.sort) > 0 {
		input[graphqlread.InputSort] = req.sort
	}
	variables[t.Input] = input
	variables[IncludeTotalCount] = req.total

	return variables
}

// rootPage sets the page of a list paged by its root's own first and after,
// or inside the input that holds them, and its free text inside the filter
// input that takes it.
func (t *Tool) rootPage(variables map[string]any, req *pageRequest) {
	page := variables
	if t.pageIn != "" {
		in, _ := variables[t.pageIn].(map[string]any)
		if in == nil {
			in = make(map[string]any, 3)
		}
		variables[t.pageIn] = in
		page = in
	}
	page[graphqlread.InputFirst] = req.limit
	if req.after != "" {
		page[graphqlread.InputAfter] = req.after
	}
	if req.query != "" {
		filter, _ := variables[t.search[0]].(map[string]any)
		if filter == nil {
			filter = make(map[string]any, 1)
		}
		filter[t.search[1]] = req.query
		variables[t.search[0]] = filter
	}
	if t.Read.Declares(IncludeTotalCount) {
		variables[IncludeTotalCount] = req.total
	}
}

// relativeWithheld names withheld fields from the record, as a model reads
// them: customers.edges.node.billingProfile.creditLimit is
// billingProfile.creditLimit.
func (t *Tool) relativeWithheld(paths []string) []string {
	return graphqlread.RelativePaths(paths,
		t.Read.RootField+"."+graphqlread.ConnectionEdges+"."+graphqlread.ConnectionNode+".")
}

// selectedColumns are the fields the operation selects on the record, in
// schema order, that the rows carry.
func presentedFields(fields []graphqlread.Field, presents map[string]string) []graphqlread.Field {
	if len(presents) == 0 {
		return fields
	}
	out := slices.Clone(fields)
	for idx := range out {
		if name, ok := presents[out[idx].Name]; ok {
			out[idx].Name = name
		}
	}

	return out
}

func presented(row map[string]any, presents map[string]string) map[string]any {
	for field, name := range presents {
		if value, ok := row[field]; ok {
			delete(row, field)
			row[name] = value
		}
	}

	return row
}

func (t *Tool) sort(args map[string]any) []map[string]any {
	field := querytool.OptionalString(args, listtool.ParamSortBy)
	if field == "" || !slices.Contains(t.sortable.Names(), field) {
		return nil
	}
	direction := querytool.OptionalString(args, listtool.ParamSortDirection)
	if direction == "" {
		direction = string(dbtype.SortDirectionAsc)
	}

	return []map[string]any{{querytool.ParamField: field, querytool.ParamDirection: direction}}
}

// refuseWithheld refuses a filter or an order on a field the caller may not
// be shown, before anything is read: filtering on a hidden value would leak
// it one row at a time.
func (t *Tool) refuseWithheld(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) error {
	fields := make([]string, 0, len(t.Params)+1)
	for _, param := range t.Params {
		if raw, present := params.Params[param.Name]; present && raw != nil && !param.OnInput {
			fields = append(fields, param.Field)
		}
	}
	if sortBy := querytool.OptionalString(params.Params, listtool.ParamSortBy); sortBy != "" {
		fields = append(fields, sortBy)
	}
	for idx := range t.Spec.Conditions {
		condition := &t.Spec.Conditions[idx]
		if raw, present := params.Params[condition.Name]; !present || raw == nil {
			continue
		}
		for _, filter := range condition.When {
			fields = append(fields, filter.Field)
		}
	}
	if err := t.refuseFields(ctx, params, fields); err != nil {
		return err
	}
	if query := strings.TrimSpace(
		querytool.OptionalString(params.Params, querytool.ParamQuery),
	); query != "" {
		return t.RefuseSearch(ctx, params)
	}

	return nil
}

// searchedFields refuses a searched field the record does not have, which
// would leave a query unguarded.
func searchedFields(tool string, fields []graphqlread.Field, searches []string) error {
	for _, name := range searches {
		if !slices.ContainsFunc(fields, func(field graphqlread.Field) bool {
			return field.Name == name
		}) {
			return fmt.Errorf("%s searches %q, which its record does not have", tool, name)
		}
	}

	return nil
}

// refuseSearch refuses free text when the fields it matches include one
// the caller may not be shown: a match on a hidden name would leak it as a
// filter on it would.
func (t *Tool) RefuseSearch(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) error {
	if err := t.refuseFields(ctx, params, t.Searched); err != nil {
		return fmt.Errorf(
			"%s cannot search by query: it matches values hidden at your access; narrow by "+
				"another parameter instead", t.Spec.Name)
	}

	return nil
}

// refuseFields refuses the first of fields the caller may not be shown.
func (t *Tool) refuseFields(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	fields []string,
) error {
	if len(fields) == 0 {
		return nil
	}

	ceilings := fieldsensitivity.NewAgentCeilings(
		t.permissions, params.Actor, params.DataAccessCeiling,
	)
	for _, field := range fields {
		resource, name := t.FieldOwner(field)
		hidden := resource != t.Spec.Resource && !ceilings.Reads(ctx, resource)
		if hidden || !fieldsensitivity.Visible(t.registry, resource, name,
			ceilings.For(ctx, resource)) {
			return fmt.Errorf(
				"%s cannot be filtered or ordered by %s: its values are hidden at your access",
				t.Spec.Name, field)
		}
	}

	return nil
}

// fieldOwner is the resource whose sensitivities a filtered field is judged
// by, and the field's name there, as the field guard judges it: a nested
// record of a registered kind (a worker's profile) by its own kind, any
// other by the list's.
func (t *Tool) FieldOwner(field string) (resource permission.Resource, name string) {
	parent, child, ok := t.Read.NestedField(field)
	if !ok {
		return t.Spec.Resource, field
	}
	kind := textcase.ConvertCamelToSnake(parent.Type)
	if t.registry.HasResource(kind) {
		return permission.Resource(kind), child.Name
	}

	return t.Spec.Resource, child.Name
}

// nearMiss reads again with one condition dropped at a time, and says which
// one, alone, kept every row out.
func (t *Tool) nearMiss(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	asked *pageRequest,
) string {
	probe := &nearMissProbe{tool: t, ctx: ctx, params: params, asked: asked}
	for _, loosen := range []func() string{
		probe.withoutQuery,
		probe.withoutFilter,
		probe.withoutInput,
		probe.withoutArgField,
	} {
		if note := loosen(); note != "" {
			return note
		}
	}
	if probe.conditions == 1 && probe.probes == 1 && len(t.Spec.Inputs) == 0 {
		return fmt.Sprintf("No %s matched, and without that condition none do either: "+
			"there are no %s here for you to find, so another filter will not find any.",
			t.Spec.EntityPlural, t.Spec.EntityPlural)
	}

	return ""
}

type nearMissProbe struct {
	tool       *Tool
	ctx        context.Context
	params     *agenttoolports.QueryToolParams
	asked      *pageRequest
	probes     int
	conditions int
}

func (p *nearMissProbe) read(req *pageRequest) *Page {
	req.groups, req.limit = p.asked.groups, nearMissSample
	page, err := p.tool.page(p.ctx, p.params, req)
	if err != nil || len(page.Items) == 0 {
		return nil
	}

	return page
}

func (p *nearMissProbe) spent() bool {
	return p.probes >= maxNearMissProbes
}

func (p *nearMissProbe) withoutQuery() string {
	asked := p.asked
	if asked.query == "" {
		return ""
	}
	p.probes++
	p.conditions++
	if p.read(&pageRequest{inputs: asked.inputs, args: asked.args, filters: asked.filters}) == nil {
		return ""
	}

	return fmt.Sprintf("Nothing matched the text %q, but %s do match the other "+
		"conditions. Try different words, or drop the text.", asked.query, p.tool.Spec.EntityPlural)
}

func (p *nearMissProbe) withoutFilter() string {
	asked := p.asked
	for idx := range asked.filters {
		p.conditions++
		if p.spent() {
			continue
		}
		p.probes++
		rest := slices.Delete(slices.Clone(asked.filters), idx, idx+1)
		page := p.read(&pageRequest{
			query:   asked.query,
			inputs:  asked.inputs,
			args:    asked.args,
			filters: rest,
		})
		if page != nil {
			return p.tool.KeptOut(asked.filters[idx][querytool.ParamField], page)
		}
	}

	return ""
}

func (p *nearMissProbe) withoutInput() string {
	asked := p.asked
	for _, name := range slices.Sorted(maps.Keys(asked.inputs)) {
		p.conditions++
		if _, toggle := asked.inputs[name].(bool); toggle || p.spent() {
			continue
		}
		p.probes++
		rest := maps.Clone(asked.inputs)
		delete(rest, name)
		page := p.read(&pageRequest{
			query:   asked.query,
			inputs:  rest,
			args:    asked.args,
			filters: asked.filters,
		})
		if page != nil {
			return p.tool.KeptOut(name, page)
		}
	}

	return ""
}

func (p *nearMissProbe) withoutArgField() string {
	asked := p.asked
	for idx := range p.tool.Params {
		param := &p.tool.Params[idx]
		if _, scoped := asked.args[param.Arg]; scoped && param.ArgField == "" {
			p.conditions++
			continue
		}
		filter, _ := asked.args[param.Arg].(map[string]any)
		value, set := filter[param.ArgField]
		if param.ArgField == "" || !set {
			continue
		}
		p.conditions++
		if _, toggle := value.(bool); toggle || p.spent() {
			continue
		}
		p.probes++
		rest := maps.Clone(asked.args)
		narrower := maps.Clone(filter)
		delete(narrower, param.ArgField)
		rest[param.Arg] = narrower
		page := p.read(&pageRequest{
			query:   asked.query,
			inputs:  asked.inputs,
			args:    rest,
			filters: asked.filters,
		})
		if page != nil {
			return p.tool.KeptOut(param.Name, page)
		}
	}

	return ""
}

func (t *Tool) KeptOut(condition any, page *Page) string {
	field := fmt.Sprint(condition)
	named := field
	if t.Spec.Amounts {
		named = graphqlread.PresentedName(named)
	}
	note := fmt.Sprintf("No %s matched every condition. Without the condition on %s",
		t.Spec.EntityPlural, named)
	if held := KeptOutRows(page, field, named); held != "" {
		count, verb := strconv.Itoa(len(page.Items)), "do"
		switch {
		case page.HasMore:
			count = "at least " + count
		case len(page.Items) == 1:
			verb = "does"
		}
		return fmt.Sprintf("%s, %s %s: %s. That condition alone keeps them out.",
			note, count, verb, held)
	}

	return note + " some do, so that one alone keeps them out."
}

func KeptOutRows(page *Page, field, named string) string {
	labels := make([]string, 0, len(page.Items))
	for _, row := range page.Items {
		label := rowLabel(row)
		value, shown := scalarText(row[field])
		if !shown {
			value, shown = scalarText(row[named])
		}
		switch {
		case label != "" && shown:
			labels = append(labels, fmt.Sprintf("%s (%s %s)", label, named, value))
		case shown:
			labels = append(labels, named+" "+value)
		case label != "":
			labels = append(labels, label)
		}
	}

	return strings.Join(slices.Compact(labels), "; ")
}

var rowLabelKeys = []string{
	"proNumber", "invoiceNumber", "settlementNumber", "number", "referenceNumber",
	"unitNumber", "code", "name", "wholeName", "title",
}

func rowLabel(row map[string]any) string {
	for _, key := range rowLabelKeys {
		if text, ok := scalarText(row[key]); ok {
			return text
		}
	}

	return ""
}

func scalarText(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		return text, text != ""
	case bool, int, int64, float64:
		return fmt.Sprint(typed), true
	default:
		return "", false
	}
}

// runView reads a composed table view through the same operation, so the
// count and rows a view previews are what listing the table shows the same
// person, and refuses a view that filters or orders by a field that person
// may not be shown, as a call is refused.
func (t *Tool) RunView(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	query string,
	filters []domaintypes.FieldFilter,
	sorting []domaintypes.SortField,
	limit int,
) (*querytool.ViewRun, error) {
	if t.Input == "" || len(t.Required) > 0 {
		return nil, fmt.Errorf("%s reads by its own arguments; a table view cannot be "+
			"composed over it", t.Spec.Name)
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
			querytool.ParamField:     filter.Field,
			querytool.FilterOperator: string(filter.Operator),
			querytool.FilterValue:    filter.Value,
		})
	}
	for _, field := range sorting {
		req.sort = append(req.sort, map[string]any{
			querytool.ParamField: field.Field, querytool.ParamDirection: string(field.Direction),
		})
	}

	page, err := t.page(ctx, params, req)
	if err != nil {
		return nil, err
	}
	criteria := filtercatalog.NewCriteria(t.Spec.EntityPlural)
	criteria.Text(query)
	items := make([]any, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, item)
	}

	return &querytool.ViewRun{
		Outcome: querytool.SearchOutcome{
			Count:       len(items),
			SearchedFor: criteria.Describe(),
			Items:       items,
			Columns:     graphqlread.SelectedColumns(t.rowFields, page.Items),
		},
		Withheld: page.withheld,
		More:     page.HasMore,
	}, nil
}
