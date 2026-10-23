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
func (r *graphQLRead) rootParams(
	tool string,
	skip map[string]struct{},
) (params []graphQLParam, required []string, search [2]string, err error) {
	for _, variable := range r.Variables {
		if _, own := skip[variable.Name]; own {
			continue
		}
		if variable.Kind == graphQLKindInput {
			filter, filterSearchIn := filterInputParams(&variable)
			params = append(params, filter...)
			if filterSearchIn != "" {
				search = [2]string{variable.Name, filterSearchIn}
			}
			continue
		}
		derived, argErr := variableParams(tool, r.Operation, &variable)
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

// variableParams is one root variable's parameter, sent as that variable.
func variableParams(tool, operation string, variable *graphQLVariable) ([]graphQLParam, error) {
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
		return nil, fmt.Errorf("%s: %s takes $%s, which no parameter can be derived from; "+
			"describe the argument in the schema", tool, operation, variable.Name)
	}
	for idx := range derived {
		derived[idx].OnInput = true
		derived[idx].Arg = variable.Name
		if said := variable.Description; said != "" && (variable.Kind == graphQLKindEnum ||
			variable.Type == graphQLTypeString) && !variable.List {
			derived[idx].Schema = withDescription(derived[idx].Schema, said)
		}
	}

	return derived, nil
}

// filterInputParams are a filter input's fields as parameters, each sent inside
// the variable, and the field it takes free text in, which is the tool's
// query rather than a parameter of its own.
func filterInputParams(variable *graphQLVariable) (params []graphQLParam, search string) {
	for idx := range graphQLInputs[variable.Type] {
		field := graphQLInputs[variable.Type][idx]
		if field.Kind == graphQLKindInput {
			continue
		}
		if field.Name == filterSearch && field.Type == graphQLTypeString {
			search = field.Name
			continue
		}
		for _, param := range argumentParam(&field) {
			param.OnInput = true
			param.Arg = variable.Name
			param.ArgField = field.Name
			params = append(params, param)
		}
	}

	return params, search
}

// renamed gives each derived parameter the name the tool takes it by
// (integrationType is system, as the accounting writes call it), and refuses
// a name for a parameter the tool does not derive.
func renamed(
	tool string,
	params []graphQLParam,
	names map[string]string,
	required []string,
) ([]graphQLParam, []string, error) {
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
	slices.SortFunc(params, func(a, b graphQLParam) int { return strings.Compare(a.Name, b.Name) })

	return params, out, nil
}

// checkReferences refuses a call that names a record by something other
// than its id, before anything is read.
func checkReferences(params []graphQLParam, args map[string]any) error {
	for _, param := range params {
		raw, present := args[param.Name]
		if !present || raw == nil {
			continue
		}
		switch param.Kind {
		case paramReference:
			if _, err := requirePulid(args, param.Name); err != nil {
				return err
			}
		case paramReferences:
			if err := validIDList(args, param.Name); err != nil {
				return err
			}
		case paramIn, paramEquals, paramContains, paramDayFrom, paramDayTo, paramAtLeast,
			paramAtMost, paramPresent, paramInstantFrom, paramInstantTo:
		}
	}

	return nil
}

// validIDList refuses a list of ids any of which is not one.
func validIDList(args map[string]any, name string) error {
	for _, value := range optionalStrings(args, name) {
		if _, err := pulid.Parse(value); err != nil {
			return fmt.Errorf("%q in %q is not a valid id: %w", value, name, err)
		}
	}

	return nil
}

// requireArguments refuses a call that leaves out an argument the read
// requires.
func requireArguments(required []string, args map[string]any) error {
	for _, name := range required {
		if raw, present := args[name]; !present || raw == nil || raw == "" {
			return fmt.Errorf("parameter %q is required", name)
		}
	}

	return nil
}

// matchAlongside pairs each read a spec makes beside or instead of its own
// with the one the generator proved, and refuses either without the other.
func matchAlongside(
	tool, how string,
	specs []alongsideSpec,
	bound []graphQLAlongside,
) ([]graphQLAlongside, error) {
	out := make([]graphQLAlongside, 0, len(specs))
	for _, beside := range specs {
		idx := slices.IndexFunc(bound, func(b graphQLAlongside) bool {
			return b.Operation == beside.operation
		})
		if idx < 0 {
			return nil, fmt.Errorf("%s reads %s %s, which reads.yml does not list for it",
				tool, beside.operation, how)
		}
		if len(bound[idx].Keys) > 0 && len(beside.inputs) > 0 {
			return nil, fmt.Errorf("%s reads %s %s by the call's own %s; it takes "+
				"no table input", tool, beside.operation, how,
				strings.Join(bound[idx].Keys, ", "))
		}
		out = append(out, bound[idx])
	}
	if len(out) != len(bound) {
		return nil, fmt.Errorf("%s: reads.yml lists %d reads %s it, its spec %d",
			tool, len(bound), how, len(out))
	}

	return out, nil
}

// besideReads are the reads a tool makes beside its own, with the specs
// that say what each is for.
type besideReads struct {
	specs       []alongsideSpec
	bound       []graphQLAlongside
	reader      agentports.GraphQLReader
	permissions permissionports.PermissionEngine
}

// read makes each read beside the tool's own, where the reader may read its
// kind and the call names what a keyed read is keyed by, and returns the
// records by key, what it withheld, and the notes it leaves: a read cut
// short, or one the reader may not make. scoped says whether the call named
// one of the tool's references.
func (b *besideReads) read(
	ctx context.Context,
	params *agentports.QueryToolParams,
	sent map[string]any,
	scoped bool,
) (records map[string]any, withheld, notes []string, err error) {
	if len(b.bound) == 0 {
		return nil, nil, nil, nil
	}
	ceilings := fieldsensitivity.NewAgentCeilings(
		b.permissions, params.Actor, params.DataAccessCeiling,
	)
	records = make(map[string]any, len(b.bound))
	for idx := range b.bound {
		bound := &b.bound[idx]
		spec := &b.specs[idx]
		resource := permission.Resource(bound.Resource)
		if spec.scoped && !scoped {
			continue
		}
		if !ceilings.Reads(ctx, resource) {
			if spec.withheld != "" {
				notes = append(notes, spec.withheld)
			}
			continue
		}
		variables, keyed := alongsideVariables(bound, spec, sent)
		if !keyed {
			continue
		}

		var result *agentports.GraphQLReadResult
		result, err = b.reader.Read(ctx, &agentports.GraphQLReadRequest{
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
				if spec.withheld != "" {
					notes = append(notes, spec.withheld)
				}
				continue
			}
			return nil, nil, nil, err
		}

		value, prefix, note := besideValue(bound, spec, result.Data)
		if value == nil {
			continue
		}
		records[spec.key] = value
		for _, path := range relativePaths(result.Withheld, prefix) {
			withheld = append(withheld, spec.key+"."+path)
		}
		if note != "" {
			notes = append(notes, note)
		}
	}

	return records, withheld, notes, nil
}

// besideValue is what a read beside the tool's own returned: one record, or
// its rows, each shaped by its spec, with the prefix its withheld paths
// carry and the note a connection read cut short leaves.
func besideValue(
	bound *graphQLAlongside,
	spec *alongsideSpec,
	data map[string]any,
) (value any, prefix, note string) {
	if bound.Object {
		node, ok := data[bound.RootField].(map[string]any)
		if !ok {
			return nil, "", ""
		}
		record := compact(node)
		if spec.shape != nil {
			spec.shape(record)
		}

		return record, bound.RootField + ".", ""
	}
	rows, prefix, note := alongsideRows(bound, spec, data)
	if spec.shape != nil {
		for _, row := range rows {
			spec.shape(row)
		}
	}

	return rows, prefix, note
}

// argumentSchema is a parameter schema for the arguments a read takes from
// its root field, marking the required ones.
func argumentSchema(properties map[string]any, required []string) map[string]any {
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
