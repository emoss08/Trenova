package agentruntime

import (
	"errors"
	"sort"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
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

	if err := validator.ValidateFor(name, schema, aliased); err != nil {
		return nil, nil, err
	}

	return aliased, aliases, nil
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

// argumentOutcome is the refusal a call gets when its arguments do not fit
// the tool's schema. A schema that could not be compiled is the tool's
// fault, not the model's, and is reported as a failure rather than a call
// to fix.
func argumentOutcome(name string, err error) toolOutcome {
	var multiErr *errortypes.MultiError
	if !errors.As(err, &multiErr) {
		return refusedOutcome(
			aitrace.OutcomeFailed,
			"its arguments could not be checked",
			"Tool %q could not be run: its arguments could not be checked (%s). Try again later.",
			name, err.Error(),
		)
	}

	problems := argumentProblems(multiErr)

	return refusedOutcome(
		aitrace.OutcomeInvalid,
		strings.Join(problems, "; "),
		"Tool %q was not run: its arguments do not fit the tool.\n- %s\n"+
			"Fix the call and send it again.",
		name, strings.Join(problems, "\n- "),
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
	return strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(name))
}
