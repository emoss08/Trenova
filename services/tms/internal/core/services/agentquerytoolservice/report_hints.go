package agentquerytoolservice

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/pkg/reportcatalog"
)

// The compiler names what it could not find and nothing else — "unknown
// field "locationName" on entity "stop"" — which is right for the builder,
// where the field picker is beside the message, and a dead end for a model,
// which answered it by trying the same field again. The hint reads the
// catalog back for each such error: the fields the entity does have that look
// like the one asked for, and the edges that lead to a dataset that has it.
var (
	unknownFieldPattern = regexp.MustCompile(`unknown field "([^"]+)" on entity "([^"]+)"`)
	unknownEdgePattern  = regexp.MustCompile(`unknown edge: "([^"]+)" on entity "([^"]+)"`)
)

// maxHintMatches bounds each list in a hint; a hint that names every field is
// the dataset description, which the model can ask for.
const maxHintMatches = 8

// compileErrorHint appends what the catalog knows to a compiler error. An
// error it does not recognize is returned as it was.
func compileErrorHint(catalog *reportcatalog.Catalog, message string) string {
	hints := make([]string, 0, 2)
	seen := make(map[string]bool, 4)

	for _, match := range unknownFieldPattern.FindAllStringSubmatch(message, -1) {
		key := "field:" + match[2] + ":" + match[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		if hint := unknownFieldHint(catalog, match[2], match[1]); hint != "" {
			hints = append(hints, hint)
		}
	}
	for _, match := range unknownEdgePattern.FindAllStringSubmatch(message, -1) {
		key := "edge:" + match[2] + ":" + match[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		if hint := unknownEdgeHint(catalog, match[2], match[1]); hint != "" {
			hints = append(hints, hint)
		}
	}

	if len(hints) == 0 {
		return message
	}

	return message + " " + strings.Join(hints, " ")
}

func unknownFieldHint(catalog *reportcatalog.Catalog, entityKey, fieldKey string) string {
	entity, ok := catalog.Entity(entityKey)
	if !ok {
		return ""
	}

	tokens := fieldTokens(fieldKey)
	own := matchingFields(entity, tokens)
	through := make([]string, 0, len(entity.Edges))
	for i := range entity.Edges {
		edge := &entity.Edges[i]
		if !edge.Traversable {
			continue
		}
		target, found := catalog.Entity(edge.Target)
		if !found {
			continue
		}
		for _, key := range matchingFields(target, tokens) {
			through = append(through, fmt.Sprintf("%s.%s", edge.Name, key))
			if len(through) >= maxHintMatches {
				break
			}
		}
		if len(through) >= maxHintMatches {
			break
		}
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "The %s dataset has no field %q.", entity.Key, fieldKey)
	if len(own) > 0 {
		fmt.Fprintf(&builder, " Its closest fields: %s.", strings.Join(own, ", "))
	}
	if len(through) > 0 {
		fmt.Fprintf(
			&builder,
			" Reachable through its edges: %s (add the edge to the ref's path, "+
				"so a field on stop's location edge is "+
				"{\"path\": [..., \"location\"], \"field\": \"name\"}).",
			strings.Join(through, ", "),
		)
	}
	if len(own) == 0 && len(through) == 0 {
		fmt.Fprintf(&builder, " Its fields: %s.", strings.Join(fieldKeys(entity), ", "))
	}
	fmt.Fprintf(
		&builder,
		" Call describe_report_dataset with dataset %q for the full list.",
		entity.Key,
	)

	return builder.String()
}

func unknownEdgeHint(catalog *reportcatalog.Catalog, entityKey, edgeName string) string {
	entity, ok := catalog.Entity(entityKey)
	if !ok {
		return ""
	}

	names := make([]string, 0, len(entity.Edges))
	for i := range entity.Edges {
		if entity.Edges[i].Traversable {
			names = append(names, entity.Edges[i].Name)
		}
	}
	if len(names) == 0 {
		return fmt.Sprintf("The %s dataset has no edge %q and no edges at all.", entity.Key, edgeName)
	}

	return fmt.Sprintf(
		"The %s dataset has no edge %q. Its edges: %s.", entity.Key, edgeName, strings.Join(names, ", "),
	)
}

// fieldTokens splits a camelCase or snake_case key into lower-case words, so
// "locationName" looks for "location" and "name" and "PRONumber" for "pro"
// and "number": an upper-case run is one word until a lower-case letter
// starts the next.
func fieldTokens(key string) []string {
	runes := []rune(key)
	tokens := make([]string, 0, 3)
	var current strings.Builder
	flush := func() {
		if current.Len() > 1 {
			tokens = append(tokens, strings.ToLower(current.String()))
		}
		current.Reset()
	}

	for i, r := range runes {
		switch {
		case r == '_' || r == '-' || r == '.' || r == ' ':
			flush()
		case unicode.IsUpper(r) && i > 0 && startsWord(runes, i):
			flush()
			current.WriteRune(r)
		default:
			current.WriteRune(r)
		}
	}
	flush()

	return tokens
}

// startsWord says whether the upper-case rune at i begins a new word: it
// follows a lower-case rune or a digit, or it ends an upper-case run that a
// lower-case rune continues ("PRONumber" breaks before the N).
func startsWord(runes []rune, i int) bool {
	previous := runes[i-1]
	if !unicode.IsUpper(previous) {
		return true
	}

	return i+1 < len(runes) && unicode.IsLower(runes[i+1])
}

// matchingFields names the entity's fields that share a word with the one
// asked for, most shared words first.
func matchingFields(entity *reportcatalog.Entity, tokens []string) []string {
	if len(tokens) == 0 {
		return nil
	}

	type scored struct {
		key   string
		score int
	}
	matches := make([]scored, 0, 4)
	for i := range entity.Fields {
		field := &entity.Fields[i]
		haystack := strings.ToLower(field.Key + " " + field.Label)
		score := 0
		for _, token := range tokens {
			if strings.Contains(haystack, token) {
				score++
			}
		}
		if score > 0 {
			matches = append(matches, scored{key: field.Key, score: score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].score > matches[j].score })

	keys := make([]string, 0, len(matches))
	for _, match := range matches {
		keys = append(keys, match.key)
		if len(keys) >= maxHintMatches {
			break
		}
	}

	return keys
}

func fieldKeys(entity *reportcatalog.Entity) []string {
	keys := make([]string, 0, len(entity.Fields))
	for i := range entity.Fields {
		keys = append(keys, entity.Fields[i].Key)
	}

	return keys
}
