package agenttoolcatalog_test

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/require"
)

// recordIDParameter is a parameter named as holding the id of a record, or a
// list of them.
var recordIDParameter = regexp.MustCompile(`(Id|Ids)$`)

// ownKeys are id parameters that hold a key the tool or its caller makes up,
// never a record's id: a chart, column, tile, widget or line of the record
// being built.
var ownKeys = map[string]struct{}{
	"chartId":  {},
	"columnId": {},
	"tileId":   {},
	"widgetId": {},
	"lineId":   {},
}

// untypedIDs are id parameters, by tool and path, that no mark could hold:
// values that are not PULIDs at all, and an id whose kind is chosen by another
// parameter from a list the registry does not follow. Each says why.
var untypedIDs = map[string]string{
	"describe_formula_schema.schemaId": "the name of a formula schema, such as shipment",
	"test_formula_expression.schemaId": "the name of a formula schema, such as shipment",
	"propose_formula.schemaId":         "the name of a formula schema, such as shipment",
	"arrange_home_layout.widgetIds": "home layout widget ids are names NextWidgetID " +
		"gives a widget, not PULIDs",
	"open_page.recordId": "its kind follows the entity parameter, drawn from the " +
		"generated product-guide catalog; the id only builds a link and is never read",
}

/*
Every parameter that takes a record's id says which kind of record, wherever it
sits: at the top, inside an object, or inside the objects a list holds.

An id parameter left untyped is checked only for looking like some id, so a
carrier's id sent where a customer's belonged passed, the service answered
"not found", and the model told the person the record did not exist. A mark
names the resource (x-recordOf) or the kinds of record (x-recordKinds) the id
must be, and the runtime refuses any other prefix before the tool runs.
*/
func TestEveryRecordIDParameterNamesItsKind(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, len(untypedIDs))
	var problems []string
	for _, tool := range buildTools(t) {
		properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
		problems = append(problems, idTypingProblems(tool.Name(), properties, "", seen)...)
	}
	for key := range untypedIDs {
		if _, ok := seen[key]; !ok {
			problems = append(problems, fmt.Sprintf(
				"%s: exempt from typing, but no tool has that parameter any more", key))
		}
	}

	sort.Strings(problems)
	require.Emptyf(t, problems, "%d id parameters do not say which record they take:\n%s",
		len(problems), strings.Join(problems, "\n"))
}

func idTypingProblems(
	tool string,
	properties map[string]any,
	path string,
	seen map[string]struct{},
) []string {
	var problems []string
	for param, raw := range properties {
		property, _ := raw.(map[string]any)
		at := param
		if path != "" {
			at = path + "." + param
		}
		problems = append(problems, kindMarkProblems(tool, at, property)...)
		if problem, bad := untypedID(tool, param, at, property, seen); bad {
			problems = append(problems, problem)
		}
		for nestedPath, nested := range nestedProperties(property, at) {
			problems = append(problems, idTypingProblems(tool, nested, nestedPath, seen)...)
		}
	}

	return problems
}

func untypedID(
	tool, param, at string,
	property map[string]any,
	seen map[string]struct{},
) (string, bool) {
	if !recordIDParameter.MatchString(param) {
		return "", false
	}
	if _, ok := selfEvident[param]; ok {
		return "", false
	}
	if _, ok := ownKeys[param]; ok {
		return "", false
	}
	key := tool + "." + at
	if _, exempt := untypedIDs[key]; exempt {
		seen[key] = struct{}{}

		return "", false
	}
	if idMarked(property) {
		return "", false
	}

	return fmt.Sprintf("%s: takes a record id without naming its resource or kinds "+
		"(agenttoolschema.RecordID, KindID, OfResource or OfKinds); if no mark can hold it, "+
		"add it to untypedIDs with the reason", key), true
}

// idMarked reports whether a property, or the items of a list, carries a mark
// the runtime checks a prefix against.
func idMarked(property map[string]any) bool {
	if marked(property) {
		return true
	}
	items, ok := property[toolschema.KeyItems].(map[string]any)

	return ok && marked(items)
}

func marked(property map[string]any) bool {
	return toolschema.RecordOf(property) != "" || len(toolschema.RecordKinds(property)) > 0
}

// kindMarkProblems holds a kinds mark, on a property or its list's items, to
// kinds whose prefix the runtime knows. A kind it does not know would let any
// id through, and the mark would promise a check that never happens.
func kindMarkProblems(tool, at string, property map[string]any) []string {
	kinds := make([]string, 0, 4)
	kinds = append(kinds, toolschema.RecordKinds(property)...)
	if items, ok := property[toolschema.KeyItems].(map[string]any); ok {
		kinds = append(kinds, toolschema.RecordKinds(items)...)
	}

	problems := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		if !permission.RecordKind(kind).Known() {
			problems = append(problems, fmt.Sprintf(
				"%s.%s: marked as taking a %s id, which has no prefix in permission's tables",
				tool, at, kind))
		}
	}

	return problems
}
