package agentquerytoolservice

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/dbtype"
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
		fmt.Fprintf(
			&builder,
			" Its fields: %s.",
			strings.Join(describedFieldKeys(entity), ", "),
		)
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
		return fmt.Sprintf(
			"The %s dataset has no edge %q and no edges at all.",
			entity.Key,
			edgeName,
		)
	}

	return fmt.Sprintf(
		"The %s dataset has no edge %q. Its edges: %s.",
		entity.Key,
		edgeName,
		strings.Join(names, ", "),
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
		if !describedField(field) {
			continue
		}
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

// bookkeepingFieldKeys are fields every dataset carries for the database
// rather than for a report: the tenant keys every row the person can see
// shares, and the counter that guards concurrent edits. Listed on every
// dataset and again on every edge's targetFields, they pushed the fields a
// report is built from past the first page of the description. They are
// hidden from what the tools describe and nothing else: the compiler still
// accepts them, so a saved report that names one keeps running.
var bookkeepingFieldKeys = map[string]struct{}{
	"businessUnitId": {},
	"organizationId": {},
	"version":        {},
}

// describedField reports whether the tools name a field to a model.
func describedField(field *reportcatalog.Field) bool {
	_, bookkeeping := bookkeepingFieldKeys[field.Key]

	return !bookkeeping
}

func describedFieldKeys(entity *reportcatalog.Entity) []string {
	keys := make([]string, 0, len(entity.Fields))
	for i := range entity.Fields {
		if describedField(&entity.Fields[i]) {
			keys = append(keys, entity.Fields[i].Key)
		}
	}

	return keys
}

func describedFieldCount(entity *reportcatalog.Entity) int {
	count := 0
	for i := range entity.Fields {
		if describedField(&entity.Fields[i]) {
			count++
		}
	}

	return count
}

// referenceKeyField names the field on source that holds the id of the
// record a to-one edge leads to — customerId for shipment's customer edge.
// An edge that picks one row of a to-many relationship, or joins through a
// table, has no such field.
func referenceKeyField(
	catalog *reportcatalog.Catalog,
	source *reportcatalog.Entity,
	edge *reportcatalog.Edge,
) (string, bool) {
	if edge.Cardinality != reportcatalog.CardinalityOne || edge.Through != nil ||
		edge.Pick != nil {
		return "", false
	}

	target, ok := catalog.Entity(edge.Target)
	if !ok {
		return "", false
	}
	targetID, ok := target.Field("id")
	if !ok {
		return "", false
	}

	for _, pair := range edge.Join {
		if pair.Remote != targetID.Column.Name {
			continue
		}
		for i := range source.Fields {
			field := &source.Fields[i]
			if field.Column.Name == pair.Local && field.Type == reportcatalog.FieldRef &&
				field.Filterable && describedField(field) {
				return field.Key, true
			}
		}
	}

	return "", false
}

// identifyingFields are the fields of a related record a model reaches for
// to name it — the words a person used — when the record's id says the same
// thing and does not change when the record is renamed.
var identifyingFields = map[string]struct{}{
	"name": {},
	"code": {},
}

// referenceKeyHint points a filter that names a related record by its name
// or code at the reference key that names it by id. It is advice, not an
// error: the filter compiled and may be exactly what was meant, but a name
// matches every record sharing it and stops matching when it is corrected,
// and a model that has just resolved the record to an id already holds the
// better key.
func referenceKeyHint(catalog *reportcatalog.Catalog, definition *report.Definition) string {
	if definition == nil || definition.Filters == nil {
		return ""
	}

	suggestions := make([]string, 0, 1)
	seen := make(map[string]struct{}, 1)
	err := definition.Filters.Walk(func(filter *report.FieldFilter) error {
		suggestion, ok := referenceKeySuggestion(catalog, definition.Entity, filter)
		if !ok {
			return nil
		}
		if _, done := seen[suggestion]; done {
			return nil
		}
		seen[suggestion] = struct{}{}
		suggestions = append(suggestions, suggestion)

		return nil
	})
	if err != nil || len(suggestions) == 0 {
		return ""
	}

	return fmt.Sprintf(
		" A filter matches a related record by its name or code, which two records "+
			"can share and which stops matching when the record is renamed. If you "+
			"already know the record's id, filter on its reference key with the id "+
			"as the value: %s.",
		strings.Join(suggestions, "; "),
	)
}

// referenceKeySuggestion reads one filter and, when it compares a related
// record's name or code, names the reference key to use instead.
func referenceKeySuggestion(
	catalog *reportcatalog.Catalog,
	entity string,
	filter *report.FieldFilter,
) (string, bool) {
	if filter.Operator != dbtype.OpEqual && filter.Operator != dbtype.OpIn {
		return "", false
	}
	if len(filter.Ref.Path) == 0 {
		return "", false
	}
	if _, identifying := identifyingFields[filter.Ref.Field]; !identifying {
		return "", false
	}

	base, resolved, err := catalog.ResolvePath(entity, filter.Ref.Path)
	if err != nil || len(resolved.Steps) == 0 {
		return "", false
	}
	steps := len(resolved.Steps)
	source := base
	if steps > 1 {
		source = resolved.Steps[steps-2].Entity
	}
	key, ok := referenceKeyField(catalog, source, resolved.Steps[steps-1].Edge)
	if !ok {
		return "", false
	}

	replacement := report.FieldRef{Path: filter.Ref.Path[:len(filter.Ref.Path)-1], Field: key}

	return fmt.Sprintf("%s instead of %s", replacement.String(), filter.Ref.String()), true
}
