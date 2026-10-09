package agentruntime

import (
	"errors"
	"sort"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/stringutils"
)

// contractArguments is what a tool receives: the model's arguments with a
// misnamed required parameter renamed to what it plainly meant, the owner
// key set aside for the runtime to stamp, and everything else held to the
// tool's whole schema.
//
// The schema used to be checked only on an approver's changes. A model that
// sent moves[].type, a stop type the domain does not have, or a window as a
// string had the field dropped or the tool refuse it in its own words, after
// the call was already a proposal on somebody's card. Now a call that does
// not fit is refused to the model with each problem named, while it can
// still fix the call; only an argument that fits reaches a tool, a preview
// or a card.
//
// The model's own call is never changed: the result is a copy where anything
// was renamed or removed, so the thread records what was sent.
func contractArguments(
	validator *toolschema.Validator,
	name string,
	schema, args map[string]any,
) (map[string]any, []argumentAlias, error) {
	contract, err := contractCall(validator, name, schema, args)
	if err != nil {
		return nil, nil, err
	}

	return contract.args, contract.aliases, nil
}

// argumentContract is what a tool receives and what was done to get there.
type argumentContract struct {
	args      map[string]any
	aliases   []argumentAlias
	coercions []argumentCoercion
}

// note is what the result tells the model about the call it sent, so the
// next one is sent as the tool declares it.
func (c argumentContract) note() string {
	return argumentNote(c.aliases, c.coercions)
}

func contractCall(
	validator *toolschema.Validator,
	name string,
	schema, args map[string]any,
	opts ...toolschema.CoerceOption,
) (argumentContract, error) {
	aliased, aliases := aliasArguments(schema, args)
	if _, owned := aliased[serviceports.SelfScopeOwnerParam]; owned {
		stripped := make(map[string]any, len(aliased)-1)
		for key, value := range aliased {
			if key != serviceports.SelfScopeOwnerParam {
				stripped[key] = value
			}
		}
		aliased = stripped
	}
	coerced, coercions := coerceArguments(schema, aliased, opts...)

	if err := validator.ValidateFor(name, schema, coerced); err != nil {
		return argumentContract{}, err
	}
	if problems := idShapeProblems(schema, coerced); len(problems) > 0 {
		multiErr := errortypes.NewMultiError()
		for _, problem := range problems {
			multiErr.Add(problem.Path, errortypes.ErrInvalid, problem.Message)
		}

		return argumentContract{}, multiErr
	}

	return argumentContract{args: coerced, aliases: aliases, coercions: coercions}, nil
}

type argumentAlias struct {
	From string
	To   string
}

func aliasNote(aliases []argumentAlias) string {
	if len(aliases) == 0 {
		return ""
	}

	renamed := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		renamed = append(renamed, alias.From+" was read as "+alias.To)
	}

	return "Arguments renamed to the tool's parameters: " + strings.Join(renamed, "; ") +
		". Use those names from now on."
}

// argumentProblems lists each thing wrong with a call as "path: message",
// in path order, so the model reads the same line a person would beside a
// form value.
func argumentProblems(multiErr *errortypes.MultiError) []string {
	lines := make([]string, 0, len(multiErr.Errors))
	for _, entry := range multiErr.Errors {
		lines = append(lines, entry.Field+": "+entry.Message)
	}
	sort.Strings(lines)

	return lines
}

// describedProblems is argumentProblems with what the tool wanted at each
// path, read from the schema: the type, the allowed values and the
// parameter's own description, which says where an id comes from. A
// refusal that only said "got string, want integer" left the model to
// guess what the parameter was for; one that names the parameter's shape
// and source is answered by one corrected call.
func describedProblems(multiErr *errortypes.MultiError, schema map[string]any) []string {
	lines := make([]string, 0, len(multiErr.Errors))
	for _, entry := range multiErr.Errors {
		line := entry.Field + ": " + entry.Message
		if expected := expectedAt(schema, entry.Field, entry.Message); expected != "" {
			line += " (" + expected + ")"
		}
		lines = append(lines, line)
	}
	sort.Strings(lines)

	return lines
}

// expectedAt says what the schema declares at a path. For a key the tool
// does not take, it names the parameters it does, so the model picks one
// instead of inventing another.
func expectedAt(schema map[string]any, path, message string) string {
	parent, property := propertyAt(schema, path)
	if message == "This tool does not take this value" {
		if names := propertyNames(parent); len(names) > 0 {
			return "this tool takes: " + strings.Join(names, ", ")
		}

		return ""
	}
	if property == nil {
		return ""
	}

	parts := make([]string, 0, 3)
	if kind := toolschema.DeclaredType(property); kind != "" {
		parts = append(parts, kind)
	}
	if values := toolschema.EnumValues(property); len(values) > 0 {
		parts = append(parts, "one of "+strings.Join(values, ", "))
	}
	if description, _ := property["description"].(string); description != "" {
		parts = append(parts, strings.TrimSpace(stringutils.FirstSentence(description)))
	}

	return strings.Join(parts, "; ")
}

// propertyAt walks a schema to a field path written as the validator writes
// it, names joined by dots and indices in brackets, and returns the object
// schema the field sits in and the field's own schema.
func propertyAt(schema map[string]any, path string) (parent, property map[string]any) {
	current := schema
	segments := strings.Split(path, ".")
	for idx, segment := range segments {
		name := segment
		indexed := false
		if at := strings.Index(segment, "["); at >= 0 {
			name = segment[:at]
			indexed = true
		}
		properties, _ := current["properties"].(map[string]any)
		next, ok := properties[name].(map[string]any)
		if !ok {
			return current, nil
		}
		if indexed {
			items, _ := next["items"].(map[string]any)
			next = items
		}
		if idx == len(segments)-1 {
			return current, next
		}
		current = next
	}

	return current, nil
}

const maxNamedParameters = 16

func propertyNames(schema map[string]any) []string {
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) == 0 {
		return nil
	}
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > maxNamedParameters {
		names = append(names[:maxNamedParameters], "…")
	}

	return names
}

// argumentOutcome is the refusal a call gets when its arguments do not fit
// the tool's schema. A schema that could not be compiled is the tool's
// fault, not the model's, and is reported as a failure rather than a call
// to fix.
func argumentOutcome(name string, schema map[string]any, err error) toolOutcome {
	var multiErr *errortypes.MultiError
	if !errors.As(err, &multiErr) {
		return refusedOutcome(
			aitrace.OutcomeFailed,
			"its arguments could not be checked",
			"Tool %q could not be run: its arguments could not be checked (%s). Try again later.",
			name, err.Error(),
		)
	}

	return refusedOutcome(
		aitrace.OutcomeInvalid,
		strings.Join(argumentProblems(multiErr), "; "),
		"Tool %q was not run or proposed: its arguments do not fit the tool.\n- %s\n"+
			"Fix the call and send it again, changing only what is named.",
		name, strings.Join(describedProblems(multiErr, schema), "\n- "),
	)
}

// aliasedArguments renames an argument the model sent under the wrong name to
// the required parameter it plainly meant, and leaves everything else alone.
//
// A small model asked for a shipment by {"id": ...} when get_shipment takes
// shipmentId, and the call failed on a missing parameter it had in fact
// supplied. Two readings are safe enough to act on: the same name spelled
// another way (shipment_id, ShipmentID), and a bare "id" when exactly one
// required *Id parameter is missing. Anything less certain is left for the
// schema to refuse, which names the parameter it wanted.
//
// The model's own call is not changed: the result is a copy, so the thread
// records what was sent and the tool reads what was meant.
func aliasedArguments(schema, args map[string]any) map[string]any {
	aliased, _ := aliasArguments(schema, args)

	return aliased
}

func aliasArguments(schema, args map[string]any) (map[string]any, []argumentAlias) {
	if len(args) == 0 {
		return args, nil
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok || len(properties) == 0 {
		return args, nil
	}

	missing := missingRequired(schema, properties, args)
	if len(missing) == 0 {
		return args, nil
	}

	renamer := &argumentRenamer{args: args}
	byShape := make(map[string]string, len(missing))
	for _, name := range missing {
		byShape[argumentShape(name)] = name
	}
	for key := range args {
		if _, declared := properties[key]; declared {
			continue
		}
		if target, found := byShape[argumentShape(key)]; found {
			renamer.rename(key, target)
			delete(byShape, argumentShape(key))
		}
	}

	if _, declared := properties["id"]; !declared {
		if value, sent := args["id"]; sent && value != nil {
			if target, lone := loneIDTarget(byShape); lone && renamer.holds("id") {
				renamer.rename("id", target)
			}
		}
	}

	return renamer.result()
}

type argumentRenamer struct {
	args    map[string]any
	aliased map[string]any
	aliases []argumentAlias
}

func (r *argumentRenamer) rename(from, to string) {
	r.aliases = append(r.aliases, argumentAlias{From: from, To: to})
	if r.aliased == nil {
		r.aliased = make(map[string]any, len(r.args))
		for key, value := range r.args {
			r.aliased[key] = value
		}
	}
	r.aliased[to] = r.aliased[from]
	delete(r.aliased, from)
}

func (r *argumentRenamer) holds(key string) bool {
	return r.aliased == nil || r.aliased[key] != nil
}

func (r *argumentRenamer) result() (map[string]any, []argumentAlias) {
	if r.aliased == nil {
		return r.args, nil
	}
	sort.Slice(r.aliases, func(i, j int) bool { return r.aliases[i].From < r.aliases[j].From })

	return r.aliased, r.aliases
}

func loneIDTarget(byShape map[string]string) (string, bool) {
	target := ""
	count := 0
	for _, name := range byShape {
		if len(name) > 2 && strings.HasSuffix(name, "Id") {
			target = name
			count++
		}
	}

	return target, count == 1
}

// missingRequired is the schema's required parameters the arguments lack.
func missingRequired(schema, properties, args map[string]any) []string {
	var required []string
	switch values := schema["required"].(type) {
	case []string:
		required = values
	case []any:
		required = make([]string, 0, len(values))
		for _, value := range values {
			if name, ok := value.(string); ok {
				required = append(required, name)
			}
		}
	}

	missing := make([]string, 0, len(required))
	for _, name := range required {
		if _, declared := properties[name]; !declared {
			continue
		}
		if value, sent := args[name]; !sent || value == nil {
			missing = append(missing, name)
		}
	}

	return missing
}

// argumentShape is a name with its spelling taken out: case, underscores and
// hyphens, so shipment_id, ShipmentID and shipmentId read alike.
func argumentShape(name string) string {
	return toolschema.NameShape(name)
}
