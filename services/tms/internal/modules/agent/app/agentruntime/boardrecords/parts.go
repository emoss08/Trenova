package boardrecords

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardtyped"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const idKey = "id"

func partsIn(value any, depth int, parents map[string]string) {
	if depth > maxDepth {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		if id, _ := typed[idKey].(string); pulid.LooksLike(id) {
			if spec, ok := partKind(id); ok {
				if parent := parentIn(typed, spec.PartOf); parent != "" {
					parents[id] = parent
				}
			}
		}
		for _, name := range slices.Sorted(maps.Keys(typed)) {
			partsIn(typed[name], depth+1, parents)
		}
	case []any:
		for _, element := range typed {
			partsIn(element, depth+1, parents)
		}
	}
}

func partKind(id string) (boardcatalog.Kind, bool) {
	kind, known := recordkind.OfIDPrefix(pulid.ID(id).Prefix())
	if !known {
		return boardcatalog.Kind{}, false
	}
	spec, ok := boardregistry.Catalog().KindOf(kind)

	return spec, ok && spec.PartOf != ""
}

func parentIn(fields map[string]any, kind recordkind.Kind) string {
	for _, name := range slices.Sorted(maps.Keys(fields)) {
		candidate := ""
		switch typed := fields[name].(type) {
		case string:
			if name != idKey {
				candidate = typed
			}
		case map[string]any:
			candidate, _ = typed[idKey].(string)
		}
		if isOfKind(candidate, kind) {
			return candidate
		}
	}

	return ""
}

func isOfKind(id string, kind recordkind.Kind) bool {
	if !pulid.LooksLike(id) {
		return false
	}
	found, known := recordkind.OfIDPrefix(pulid.ID(id).Prefix())

	return known && found == kind
}

func (r Read) WholesOf(arguments map[string]any) (map[string]any, bool) {
	shown, _ := AnswerShown(arguments)
	wholes := make(map[string]any, 8)
	for kind, ids := range shown {
		if spec, ok := boardregistry.Catalog().KindOf(kind); !ok || spec.PartOf == "" {
			continue
		}
		for _, id := range ids {
			if parent := r.parents[id.String()]; parent != "" {
				wholes[id.String()] = parent
			}
		}
	}
	if len(wholes) == 0 {
		return arguments, false
	}
	out := maps.Clone(arguments)
	out[boardtyped.PartOfField] = wholes

	return out, true
}

func (r Read) PartsRefusal(
	tool string,
	shown map[recordkind.Kind][]pulid.ID,
	words string,
) string {
	if len(shown) != 1 {
		return ""
	}
	for whole, ids := range shown {
		for _, part := range boardregistry.Catalog().Kinds() {
			if part.PartOf != whole || !mentions(words, part.Kind) {
				continue
			}
			held := r.partsOf(part.Kind, ids)
			if len(held) == 0 {
				continue
			}
			return partsRefusal(tool, whole, part.Kind, held)
		}
	}

	return ""
}

type heldParts struct {
	label string
	ids   []string
}

func (r Read) partsOf(part recordkind.Kind, wholes []pulid.ID) []heldParts {
	held := make([]heldParts, 0, len(wholes))
	for _, whole := range wholes {
		entry := heldParts{label: whole.String()}
		if ref, known := r.ref(whole.String()); known && ref.Label != "" {
			entry.label = ref.Label
		}
		for _, id := range slices.Sorted(maps.Keys(r.parents)) {
			if r.parents[id] == whole.String() && isOfKind(id, part) {
				entry.ids = append(entry.ids, id)
			}
		}
		if len(entry.ids) > 0 {
			held = append(held, entry)
		}
	}

	return held
}

func mentions(words string, kind recordkind.Kind) bool {
	noun := kind.Noun()
	if cut := strings.LastIndexByte(noun, ' '); cut >= 0 {
		noun = noun[cut+1:]
	}
	noun = strings.ToLower(noun)
	lowered := strings.ToLower(words)

	return stringutils.ContainsWords(lowered, noun) ||
		stringutils.ContainsWords(lowered, inflect.PluralNoun(noun))
}

func partsRefusal(tool string, whole, part recordkind.Kind, held []heldParts) string {
	wholes := inflect.PluralNoun(whole.Noun())
	parts := inflect.PluralNoun(part.Noun())
	listed := make([]string, 0, min(len(held), maxNamedInNote)+1)
	for _, entry := range held[:min(len(held), maxNamedInNote)] {
		listed = append(listed, entry.label+": "+strings.Join(entry.ids, ", "))
	}
	if len(held) > maxNamedInNote {
		listed = append(listed, strconv.Itoa(len(held)-maxNamedInNote)+" more")
	}

	return fmt.Sprintf(
		"Tool %q was not run: its words are about %s, but its records are %s, so each row "+
			"would be a %s and nothing on the board would read the %s. Show the %s "+
			"themselves, the ones this conversation read (%s), so each row reads its own "+
			"measure; {count} then counts %s.",
		tool, parts, wholes, whole.Noun(), parts, parts, strings.Join(listed, "; "),
		parts,
	)
}
