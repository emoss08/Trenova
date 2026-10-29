package graphqlread

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// filterSearch is the field a filter input takes free text in, which a
	// tool offers as its query.
	filterSearch = "search"
)

// rootParams are a paged list's or a get's parameters from its root field's
// own arguments: each scalar, enum or id variable, sent as that variable, and
// each field of a filter input, sent inside it. skip names the variables the
// tool sends itself (its table input, the page), and search is where a
// filter input takes the list's free text.
func (r *Read) RootParams(
	tool string,
	skip map[string]struct{},
) (params []Param, required []string, search [2]string, err error) {
	pageIn := r.PagedInput()
	for _, variable := range r.Variables {
		if _, own := skip[variable.Name]; own {
			continue
		}
		if variable.Kind == KindInput {
			filter, filterSearchIn := filterInputParams(
				&variable, variable.Name == pageIn, r.Verb == VerbGet,
			)
			params = append(params, filter...)
			if filterSearchIn != "" {
				search = [2]string{variable.Name, filterSearchIn}
			}
			continue
		}
		derived, argErr := VariableParams(tool, r.Operation, &variable)
		if argErr != nil {
			return nil, nil, search, argErr
		}
		params = append(params, derived...)
		if variable.NonNull {
			required = append(required, variable.Name)
		}
	}

	return params, required, search, nil
}

// WithRootArgs adds a list's root-argument parameters to its field
// parameters. Where both name the same thing (a service failure's
// shipmentId), the root argument the screen itself sends wins.
func WithRootArgs(params, args []Param) []Param {
	names := make(map[string]struct{}, len(args))
	for _, arg := range args {
		names[arg.Name] = struct{}{}
	}
	params = slices.DeleteFunc(params, func(param Param) bool {
		_, shadowed := names[param.Name]
		return shadowed
	})

	return append(params, args...)
}

// VariableParams is one root variable's parameter, sent as that variable.
func VariableParams(tool, operation string, variable *Variable) ([]Param, error) {
	field := Field{
		Name:        variable.Name,
		Type:        variable.Type,
		Kind:        variable.Kind,
		List:        variable.List,
		NonNull:     variable.NonNull,
		Enum:        variable.Enum,
		Description: variable.Description,
	}
	derived := ArgumentParam(&field)
	if len(derived) == 0 {
		return nil, fmt.Errorf("%s: %s takes $%s, which no parameter can be derived from; "+
			"describe the argument in the schema", tool, operation, variable.Name)
	}
	for idx := range derived {
		derived[idx].OnInput = true
		derived[idx].Arg = variable.Name
		if said := variable.Description; said != "" && (variable.Kind == graphQLKindEnum ||
			variable.Type == TypeString) && !variable.List {
			derived[idx].Schema = WithDescription(derived[idx].Schema, said)
		}
	}

	return derived, nil
}

// filterInputParams are a filter input's fields as parameters, each sent inside
// the variable, and the field it takes free text in, which is the tool's
// query rather than a parameter of its own. An input a list pages inside
// keeps its first and after for the page, and takes free text in its query.
// A get's input chooses the record rather than filtering rows (a briefing's
// role), so a described enum or text field of it says only what its own
// description says.
func filterInputParams(
	variable *Variable,
	paged, get bool,
) (params []Param, search string) {
	for idx := range Inputs[variable.Type] {
		field := Inputs[variable.Type][idx]
		if field.Kind == KindInput ||
			(paged && (field.Name == InputFirst || field.Name == InputAfter)) {
			continue
		}
		searched := field.Name == filterSearch || (paged && field.Name == InputQuery)
		if searched && field.Type == TypeString {
			search = field.Name
			continue
		}
		for _, param := range ArgumentParam(&field) {
			param.OnInput = true
			param.Arg = variable.Name
			param.ArgField = field.Name
			if get && field.Description != "" && !field.List &&
				(field.Kind == graphQLKindEnum || field.Type == TypeString) &&
				param.Kind != ParamDayText {
				param.Schema = WithDescription(param.Schema, field.Description)
			}
			params = append(params, param)
		}
	}

	return params, search
}

// Renamed gives each derived parameter the name the tool takes it by
// (integrationType is system, as the accounting writes call it), and refuses
// a name for a parameter the tool does not derive.
func Renamed(
	tool string,
	params []Param,
	names map[string]string,
	required []string,
) ([]Param, []string, error) {
	if len(names) == 0 {
		return params, required, nil
	}
	used := make(map[string]struct{}, len(names))
	for idx := range params {
		if name, ok := names[params[idx].Name]; ok {
			used[params[idx].Name] = struct{}{}
			params[idx].Name = name
		}
	}
	out := make([]string, 0, len(required))
	for _, name := range required {
		if renamedTo, ok := names[name]; ok {
			name = renamedTo
		}
		out = append(out, name)
	}
	for from := range names {
		if _, ok := used[from]; !ok {
			return nil, nil, fmt.Errorf("%s names %q, which is not one of its parameters",
				tool, from)
		}
	}
	slices.SortFunc(params, func(a, b Param) int { return strings.Compare(a.Name, b.Name) })

	return params, out, nil
}

// CheckReferences refuses a call that names a record by something other
// than its id, before anything is read.
func CheckReferences(params []Param, args map[string]any) error {
	for _, param := range params {
		raw, present := args[param.Name]
		if !present || raw == nil {
			continue
		}
		switch param.Kind {
		case ParamReference:
			if _, err := querytool.RequirePulid(args, param.Name); err != nil {
				return err
			}
		case ParamReferences:
			if err := validIDList(args, param.Name); err != nil {
				return err
			}
		case ParamIn, ParamEquals, ParamContains, ParamDayFrom, ParamDayTo, ParamAtLeast,
			ParamAtMost, ParamPresent, ParamInstantFrom, ParamInstantTo, ParamDayText:
		}
	}

	return nil
}

// validIDList refuses a list of ids any of which is not one.
func validIDList(args map[string]any, name string) error {
	for _, value := range querytool.OptionalStrings(args, name) {
		if _, err := pulid.Parse(value); err != nil {
			return fmt.Errorf("%q in %q is not a valid id: %w", value, name, err)
		}
	}

	return nil
}

// RequireArguments refuses a call that leaves out an argument the read
// requires.
func RequireArguments(required []string, args map[string]any) error {
	for _, name := range required {
		if raw, present := args[name]; !present || raw == nil || raw == "" {
			return fmt.Errorf("parameter %q is required", name)
		}
	}

	return nil
}

// MatchAlongside pairs each read a spec makes beside or instead of its own
// with the one the generator proved, and refuses either without the other.
func MatchAlongside(
	tool, how string,
	specs []readspec.Alongside,
	bound []Alongside,
) ([]Alongside, error) {
	out := make([]Alongside, 0, len(specs))
	for _, beside := range specs {
		idx := slices.IndexFunc(bound, func(b Alongside) bool {
			return b.Operation == beside.Operation
		})
		if idx < 0 {
			return nil, fmt.Errorf("%s reads %s %s, which reads.yml does not list for it",
				tool, beside.Operation, how)
		}
		for name := range beside.Inputs {
			if bound[idx].Input == "" && !slices.Contains(bound[idx].Set, name) {
				return nil, fmt.Errorf("%s reads %s %s by the call's own %s; it takes "+
					"no %s of the spec's", tool, beside.Operation, how,
					strings.Join(bound[idx].Keys, ", "), name)
			}
		}
		out = append(out, bound[idx])
	}
	if len(out) != len(bound) {
		return nil, fmt.Errorf("%s: reads.yml lists %d reads %s it, its spec %d",
			tool, len(bound), how, len(out))
	}

	return out, nil
}

// BesideReads are the reads a tool makes beside its own, with the specs
// that say what each is for.
type BesideReads struct {
	Specs       []readspec.Alongside
	Bound       []Alongside
	Reader      graphqlreadports.GraphQLReader
	Permissions permissionports.PermissionEngine
}

// read makes each read beside the tool's own, where the reader may read its
// kind and the call names what a keyed read is keyed by, and returns the
// records by key, what it withheld, and the notes it leaves: a read cut
// short, or one the reader may not make. scoped says whether the call named
// one of the tool's references.
func (b *BesideReads) Read(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	sent map[string]any,
	scoped bool,
) (records map[string]any, withheld, notes []string, err error) {
	if len(b.Bound) == 0 {
		return nil, nil, nil, nil
	}
	ceilings := fieldsensitivity.NewAgentCeilings(
		b.Permissions, params.Actor, params.DataAccessCeiling,
	)
	records = make(map[string]any, len(b.Bound))
	for idx := range b.Bound {
		bound := &b.Bound[idx]
		spec := &b.Specs[idx]
		resource := permission.Resource(bound.Resource)
		if (spec.Scoped && !scoped) || namesAny(sent, spec.Unless) {
			continue
		}
		if !ceilings.Reads(ctx, resource) {
			if spec.Withheld != "" {
				notes = append(notes, spec.Withheld)
			}
			continue
		}
		variables, keyed := alongsideVariables(bound, spec, sent)
		if !keyed {
			continue
		}

		var result *graphqlreadports.GraphQLReadResult
		result, err = b.Reader.Read(ctx, &graphqlreadports.GraphQLReadRequest{
			Actor:             params.Actor,
			Operation:         bound.Operation,
			Variables:         variables,
			DataAccessCeiling: params.DataAccessCeiling,
			RequestID:         params.RunID.String(),
		})
		if err != nil {
			return nil, nil, nil, err
		}
		if err = ReadFailure(result, resource); err != nil {
			var refused *graphqlreadports.ReadRefusedError
			if errors.As(err, &refused) {
				if spec.Withheld != "" {
					notes = append(notes, spec.Withheld)
				}
				continue
			}
			return nil, nil, nil, err
		}

		value, prefix, note := BesideValue(bound, spec, result.Data)
		if value == nil {
			continue
		}
		records[spec.Key] = value
		for _, path := range RelativePaths(result.Withheld, prefix) {
			withheld = append(withheld, spec.Key+"."+path)
		}
		if note != "" {
			notes = append(notes, note)
		}
	}

	return records, withheld, notes, nil
}

// BesideValue is what a read beside the tool's own returned: one record, or
// its rows, each shaped by its spec, with the prefix its withheld paths
// carry and the note a connection read cut short leaves.
func BesideValue(
	bound *Alongside,
	spec *readspec.Alongside,
	data map[string]any,
) (value any, prefix, note string) {
	if bound.Object {
		node, ok := data[bound.RootField].(map[string]any)
		if !ok {
			return nil, "", ""
		}
		record := Compact(node)
		if spec.Shape != nil {
			spec.Shape(record)
		}

		return record, bound.RootField + ".", ""
	}
	rows, prefix, note := alongsideRows(bound, spec, data)
	if spec.Shape != nil {
		for _, row := range rows {
			spec.Shape(row)
		}
	}

	return rows, prefix, note
}

// ArgumentSchema is a parameter schema for the arguments a read takes from
// its root field, marking the required ones.
func ArgumentSchema(properties map[string]any, required []string) map[string]any {
	schema := map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyAdditionalProperties: false,
	}
	if len(required) > 0 {
		schema[toolschema.KeyRequired] = slices.Clone(required)
	}

	return schema
}

// namesAny reports whether a read's variables give any of names a value.
func namesAny(sent map[string]any, names []string) bool {
	for _, name := range names {
		if raw, given := sent[name]; given && raw != nil && raw != "" {
			return true
		}
	}

	return false
}
