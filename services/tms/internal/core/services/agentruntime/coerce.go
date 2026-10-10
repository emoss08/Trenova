package agentruntime

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// argumentCoercion is one value the runtime read the way the tool declares
// it rather than the way the model sent it.
type argumentCoercion = toolschema.Coercion

// maxCoercionsNoted bounds how many adjustments a result names. The point of
// naming them is to teach the next call; a list of twenty teaches nothing.
const maxCoercionsNoted = 5

// coerceArguments reads a call's arguments as the tool declares them,
// wherever the reading is certain (toolschema.Coerce). The proposal executor
// applies the same readings to what an approver changed, so a value the
// runtime would have read is read the same way when it runs.
func coerceArguments(
	schema, args map[string]any,
	opts ...toolschema.CoerceOption,
) (map[string]any, []argumentCoercion) {
	return toolschema.Coerce(schema, args, opts...)
}

// coercionNote tells the model what was read differently from how it was
// sent, so the next call is sent right and the note stops appearing.
//
// Parameters left out for being empty are named together on a line of their
// own: a model that fills every parameter sends five of them at once, and a
// note per parameter would push the readings that matter past the cap.
func coercionNote(coercions []argumentCoercion) string {
	if len(coercions) == 0 {
		return ""
	}
	omitted := make([]string, 0, len(coercions))
	read := make([]argumentCoercion, 0, len(coercions))
	for _, coercion := range coercions {
		if coercion.Omitted {
			omitted = append(omitted, coercion.Path)

			continue
		}
		read = append(read, coercion)
	}

	parts := make([]string, 0, 2)
	if len(read) > 0 {
		notes := make([]string, 0, min(len(read), maxCoercionsNoted))
		for _, coercion := range read[:min(len(read), maxCoercionsNoted)] {
			notes = append(notes, coercion.Note)
		}
		text := "Arguments were read as the tool declares them: " + strings.Join(notes, "; ")
		if len(read) > maxCoercionsNoted {
			text += fmt.Sprintf("; and %d more", len(read)-maxCoercionsNoted)
		}
		parts = append(parts, text+". Send them that way from now on.")
	}
	if len(omitted) > 0 {
		parts = append(parts, "Left out because they were empty: "+strings.Join(omitted, ", ")+
			". Leave out a parameter you have no value for instead of sending it empty.")
	}

	return strings.Join(parts, " ")
}

// argumentNote is everything the runtime changed about a call's arguments
// before the tool read them: parameters renamed and values read as declared.
func argumentNote(aliases []argumentAlias, coercions []argumentCoercion) string {
	parts := make([]string, 0, 2)
	if note := aliasNote(aliases); note != "" {
		parts = append(parts, note)
	}
	if note := coercionNote(coercions); note != "" {
		parts = append(parts, note)
	}

	return strings.Join(parts, " ")
}

// idShapeProblems names each record id parameter whose value is not an id of
// the record it takes: a pro number, a name or a count sent where the tool
// wanted the id a lookup hands out, or a carrier's id sent for a customer.
// The schema cannot say so, since an id is a string to it, and the service
// answers "not found" for a value that was never the right id, which a model
// reads as the record not existing.
//
// A parameter marked with the resource it takes (toolschema.KeyRecordOf) is
// held to that resource's prefix exactly, and one marked with the kinds of
// record it takes (toolschema.KeyRecordKinds) to one of those kinds' prefixes.
// An unmarked one is checked only for
// the shape of an id, and only when its description names where the id comes
// from, so a key that is not a PULID (a report's chart id, an external
// reference) is left alone.
func idShapeProblems(schema, args map[string]any) []argumentProblem {
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) == 0 {
		return nil
	}
	var problems []argumentProblem
	walkIDs(properties, args, "", &problems)
	sort.Slice(problems, func(i, j int) bool { return problems[i].Path < problems[j].Path })

	return problems
}

type argumentProblem struct {
	Path    string
	Message string
}

// idCheck is what one id parameter is held to: the kinds of record whose
// prefix it must carry, when it is marked with a resource or with kinds, and
// the description naming where its ids come from.
type idCheck struct {
	kinds       []permission.RecordKind
	description string
}

// markedKinds is the kinds of record a property's mark names: the one kind of
// the resource it is marked with, or the kinds it lists.
func markedKinds(property map[string]any) []permission.RecordKind {
	if resource := toolschema.RecordOf(property); resource != "" {
		return []permission.RecordKind{permission.RecordKind(resource)}
	}
	names := toolschema.RecordKinds(property)
	if len(names) == 0 {
		return nil
	}
	kinds := make([]permission.RecordKind, 0, len(names))
	for _, name := range names {
		kinds = append(kinds, permission.RecordKind(name))
	}

	return kinds
}

func walkIDs(properties, args map[string]any, path string, problems *[]argumentProblem) {
	for key, value := range args {
		property, ok := properties[key].(map[string]any)
		if !ok || value == nil {
			continue
		}
		at := toolschema.JoinPath(path, key)
		switch toolschema.DeclaredType(property) {
		case "object":
			if object, isObject := value.(map[string]any); isObject {
				if nested, has := property["properties"].(map[string]any); has {
					walkIDs(nested, object, at, problems)
				}
			}
		case "array":
			items, _ := property["items"].(map[string]any)
			list, isList := value.([]any)
			if !isList || len(items) == 0 {
				continue
			}
			check, checked := listIDCheck(key, property, items)
			for idx, element := range list {
				elementPath := at + "[" + strconv.Itoa(idx) + "]"
				switch toolschema.DeclaredType(items) {
				case "object":
					if object, isObject := element.(map[string]any); isObject {
						if nested, has := items["properties"].(map[string]any); has {
							walkIDs(nested, object, elementPath, problems)
						}
					}
				case "string":
					if text, isText := element.(string); isText && checked {
						if problem, bad := idProblem(elementPath, text, check); bad {
							*problems = append(*problems, problem)
						}
					}
				}
			}
		case "string":
			text, isText := value.(string)
			if !isText {
				continue
			}
			if check, checked := singleIDCheck(key, property); checked {
				if problem, bad := idProblem(at, text, check); bad {
					*problems = append(*problems, problem)
				}
			}
		}
	}
}

func singleIDCheck(name string, property map[string]any) (idCheck, bool) {
	description, _ := property["description"].(string)
	if kinds := markedKinds(property); len(kinds) > 0 {
		return idCheck{kinds: kinds, description: description}, true
	}
	if !namesRecordID(name, property) {
		return idCheck{}, false
	}

	return idCheck{description: description}, true
}

func listIDCheck(name string, property, items map[string]any) (idCheck, bool) {
	description, _ := property["description"].(string)
	if kinds := markedKinds(items); len(kinds) > 0 {
		return idCheck{kinds: kinds, description: description}, true
	}
	if !namesRecordIDs(name, property) {
		return idCheck{}, false
	}

	return idCheck{description: description}, true
}

// idExemptNames are parameters named like ids that hold something a person
// or another system assigned, never a PULID.
var idExemptNames = map[string]struct{}{
	"idempotencykey": {},
	"taxid":          {},
	"externalid":     {},
	"chartid":        {},
	"columnid":       {},
	"tileid":         {},
	"widgetid":       {},
	"lineid":         {},
}

func namesRecordID(name string, property map[string]any) bool {
	if !strings.HasSuffix(name, "Id") || len(name) <= 2 {
		return false
	}

	return idSourced(name, property)
}

func namesRecordIDs(name string, property map[string]any) bool {
	if !strings.HasSuffix(name, "Ids") || len(name) <= 3 {
		return false
	}

	return idSourced(name, property)
}

func idSourced(name string, property map[string]any) bool {
	if _, exempt := idExemptNames[toolschema.NameShape(name)]; exempt {
		return false
	}
	description, _ := property["description"].(string)

	return strings.Contains(description, " from ")
}

func idProblem(path, value string, check idCheck) (argumentProblem, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return argumentProblem{}, false
	}
	if prefix, length, misCopied := pulid.MisCopied(trimmed); misCopied {
		return argumentProblem{Path: path, Message: fmt.Sprintf(
			"%q is not a whole id: after %q it has %d characters where an id has 26, so one "+
				"was lost or added in copying it. Send the id exactly as the result that gave it "+
				"wrote it.",
			trimmed, prefix, length,
		)}, true
	}
	if !pulid.LooksLike(trimmed) {
		message := fmt.Sprintf(
			"%q is not a record id. A record id looks like shp_01J… and comes from a lookup, "+
				"never from a number, a name or a guess. %s",
			trimmed, strings.TrimSpace(stringutils.FirstSentence(check.description)),
		)

		return argumentProblem{Path: path, Message: strings.TrimSpace(message)}, true
	}
	if len(check.kinds) == 0 {
		return argumentProblem{}, false
	}

	return wrongKindProblem(path, pulid.ID(trimmed), check)
}

// wrongKindProblem refuses an id of a kind of record the parameter does not
// take, named by what its prefix says it is, when the parameter is marked with
// the resource or the kinds it takes and those have prefixes of their own. A
// kind with no prefix in permission's table cannot be told apart, so a mark
// naming only such kinds checks nothing beyond the shape of an id.
func wrongKindProblem(path string, id pulid.ID, check idCheck) (argumentProblem, bool) {
	got := id.Prefix()
	nouns := make([]string, 0, len(check.kinds))
	prefixes := make([]string, 0, len(check.kinds))
	checked := make([]permission.RecordKind, 0, len(check.kinds))
	for _, kind := range check.kinds {
		prefix, known := kind.IDPrefix()
		if !known {
			continue
		}
		if prefix == got {
			return argumentProblem{}, false
		}
		nouns = append(nouns, stringutils.WithArticle(kind.Noun()))
		prefixes = append(prefixes, strconv.Quote(prefix))
		checked = append(checked, kind)
	}
	if len(nouns) == 0 {
		return argumentProblem{}, false
	}

	takes := stringutils.JoinAlternatives(nouns)
	from := ""
	if source := idSource(check.description); source != "" {
		from = ", from " + source
	}
	if actual, found := permission.RecordKindOfIDPrefix(got); found {
		return argumentProblem{Path: path, Message: fmt.Sprintf(
			"%q is %s id; this parameter takes %s id%s.",
			id, stringutils.WithArticle(actual.Noun()), takes, from,
		)}, true
	}
	if len(nouns) == 1 {
		return argumentProblem{Path: path, Message: fmt.Sprintf(
			"%q is not %s id: %s ids start with %s. This parameter takes %s id%s.",
			id, takes, checked[0].Noun(), prefixes[0], takes, from,
		)}, true
	}

	return argumentProblem{Path: path, Message: fmt.Sprintf(
		"%q is not %s id: their ids start with %s. This parameter takes %s id%s.",
		id, takes, stringutils.JoinAlternatives(prefixes), takes, from,
	)}, true
}

// idSource is where a parameter's description says its ids come from: the
// words after "from" up to the end of that sentence.
func idSource(description string) string {
	_, after, found := strings.Cut(description, " from ")
	if !found {
		return ""
	}
	if end := strings.Index(after, ". "); end >= 0 {
		after = after[:end]
	}

	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(after), "."))
}
