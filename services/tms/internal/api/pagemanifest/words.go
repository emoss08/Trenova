package pagemanifest

import (
	"fmt"
	"go/format"
	"slices"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/api/graphql/gqlext"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/vektah/gqlparser/v2/ast"
)

// WordsPath is where the role editor's words for each reference view are
// written, relative to the repository root.
const WordsPath = "services/tms/internal/modules/identity/app/pageaccessservice/viewwords_gen.go"

// bookkeeping is in every view and says nothing about a record, so a sentence
// never names it.
var bookkeeping = []string{
	"id", "organizationId", "businessUnitId", "version", "createdAt", "updatedAt",
}

// addressParts are the fields a person reads together as a record's
// address, wherever a view carries them.
var addressParts = []string{
	"addressLine1", "addressLine2", "city", "state", "stateId", "postalCode",
	"country", "countryId", "latitude", "longitude", "isGeocoded", "placeId",
}

// initialisms are written in capitals inside a field's words.
var initialisms = map[string]string{
	"mc": "MC", "dot": "DOT", "scac": "SCAC", "edi": "EDI", "url": "URL", "pro": "PRO",
	"vin": "VIN", "ifta": "IFTA", "id": "ID", "ein": "EIN", "cdl": "CDL", "bol": "BOL",
	"po": "PO", "ach": "ACH", "gl": "GL", "ar": "AR", "ap": "AP", "eta": "ETA",
}

// ViewWord is one phrase a sentence names a view by, and the view fields it
// stands for.
type ViewWord struct {
	Label  string
	Fields []string
}

// ViewWords is a kind's reference view as a person reads it: bookkeeping
// left out, an address's parts as one address, a link's id beside the link
// as the link, each field's name in plain words, in the view's order.
func ViewWords(view []string, isKind func(field string) bool) []ViewWord {
	hasAddress := slices.ContainsFunc(view, func(field string) bool {
		return field == "addressLine1" || field == "city" || field == "postalCode"
	})
	var words []ViewWord
	address := -1
	for _, field := range view {
		switch {
		case slices.Contains(bookkeeping, field):
			continue
		case hasAddress && slices.Contains(addressParts, field):
			if address < 0 {
				address = len(words)
				words = append(words, ViewWord{Label: "address"})
			}
			words[address].Fields = append(words[address].Fields, field)
			continue
		}
		if base, ok := strings.CutSuffix(field, "Id"); ok && base != "" &&
			slices.Contains(view, base) {
			at := slices.IndexFunc(words, func(w ViewWord) bool {
				return slices.Contains(w.Fields, base)
			})
			if at >= 0 {
				words[at].Fields = append(words[at].Fields, field)
				continue
			}
		}
		if at := slices.IndexFunc(words, func(w ViewWord) bool {
			return slices.Contains(w.Fields, field+"Id")
		}); at >= 0 {
			words[at].Fields = append(words[at].Fields, field)
			continue
		}
		label := FieldWords(field, isKind)
		if base, ok := strings.CutSuffix(field, "Id"); ok && slices.Contains(view, base) {
			label = FieldWords(base, isKind)
		}
		words = append(words, ViewWord{Label: label, Fields: []string{field}})
	}
	return words
}

// FieldWords is a field's name as words in a sentence: mcNumber is "MC
// number", isSystem "system", and a link to another kind by its id the
// kind (invoiceId is "invoice").
func FieldWords(field string, isKind func(field string) bool) string {
	if base, ok := strings.CutSuffix(field, "Id"); ok && base != "" && isKind(base) {
		field = base
	}
	if rest, ok := strings.CutPrefix(
		field,
		"is",
	); ok && rest != "" &&
		unicode.IsUpper(rune(rest[0])) {
		field = rest
	}
	var words []string
	var current []rune
	runes := []rune(field)
	for i, r := range runes {
		boundary := unicode.IsUpper(r) && i > 0 &&
			(unicode.IsLower(runes[i-1]) ||
				(i+1 < len(runes) && unicode.IsLower(runes[i+1])))
		digit := unicode.IsDigit(r) && i > 0 && !unicode.IsDigit(runes[i-1])
		if (boundary || digit) && len(current) > 0 {
			words = append(words, string(current))
			current = nil
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	for i, word := range words {
		lower := strings.ToLower(word)
		if upper, ok := initialisms[lower]; ok {
			words[i] = upper
			continue
		}
		words[i] = lower
	}
	return strings.Join(words, " ")
}

// LinkedFrom is, for each kind, the kinds whose records link to one of it:
// a GraphQL type of the carrier kind with a field returning a type of the
// kind, where the kind is not part of the carrier.
func LinkedFrom(
	schema *ast.Schema,
	registry *permission.Registry,
) (map[string][]string, error) {
	kinds, err := gqlext.KindTypes(schema, registry)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string)
	for name, kind := range kinds {
		carrier := registry.Family(kind.String())
		for _, linked := range reachedKinds(schema, kinds, schema.Types[name]) {
			if linked.String() == carrier || registry.PartOf(linked.String(), carrier) {
				continue
			}
			if !slices.Contains(out[linked.String()], carrier) {
				out[linked.String()] = append(out[linked.String()], carrier)
			}
		}
	}
	for kind := range out {
		slices.Sort(out[kind])
	}
	return out, nil
}

// reachedKinds is every kind a type's fields return, directly or through
// types that are no kind of their own (a move's assignment holding its
// driver), stopping at each kind.
func reachedKinds(
	schema *ast.Schema,
	kinds map[string]permission.Resource,
	from *ast.Definition,
) []permission.Resource {
	var found []permission.Resource
	seen := map[string]bool{}
	var visit func(def *ast.Definition)
	visit = func(def *ast.Definition) {
		if def == nil || seen[def.Name] {
			return
		}
		seen[def.Name] = true
		for _, field := range def.Fields {
			child := schema.Types[field.Type.Name()]
			if child == nil || (child.Kind != ast.Object && child.Kind != ast.Interface) {
				continue
			}
			if kind, ok := kinds[child.Name]; ok {
				if !slices.Contains(found, kind) {
					found = append(found, kind)
				}
				continue
			}
			if strings.HasSuffix(child.Name, "Connection") ||
				strings.HasSuffix(child.Name, "Edge") {
				continue
			}
			visit(child)
		}
	}
	visit(from)
	return found
}

// RenderWords writes the role editor's words for every kind with a view, and
// which kinds link to each.
func RenderWords(schema *ast.Schema, registry *permission.Registry) (string, error) {
	linked, err := LinkedFrom(schema, registry)
	if err != nil {
		return "", err
	}
	defs := registry.All()
	slices.SortFunc(defs, func(a, b *permission.ResourceDefinition) int {
		return strings.Compare(a.Resource, b.Resource)
	})

	var b strings.Builder
	b.WriteString(
		"// Code generated by go run ./internal/api/graphql/gqlext/refgen. DO NOT EDIT.\n\n",
	)
	b.WriteString("package pageaccessservice\n\n")
	b.WriteString("var viewWords = map[string][]viewWord{\n")
	for _, def := range defs {
		words := ViewWords(def.ReferenceView, func(field string) bool {
			_, ok := registry.Get(stringutils.ConvertCamelToSnake(field))
			return ok
		})
		if len(words) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\t%q: {\n", def.Resource)
		for _, word := range words {
			quoted := make([]string, len(word.Fields))
			for i, field := range word.Fields {
				quoted[i] = fmt.Sprintf("%q", field)
			}
			fmt.Fprintf(&b, "\t\t{Label: %q, Fields: []string{%s}},\n",
				word.Label, strings.Join(quoted, ", "))
		}
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n\n")
	b.WriteString("var linkedFrom = map[string][]string{\n")
	kinds := make([]string, 0, len(linked))
	for kind := range linked {
		kinds = append(kinds, kind)
	}
	slices.Sort(kinds)
	for _, kind := range kinds {
		quoted := make([]string, len(linked[kind]))
		for i, carrier := range linked[kind] {
			quoted[i] = fmt.Sprintf("%q", carrier)
		}
		fmt.Fprintf(&b, "\t%q: {%s},\n", kind, strings.Join(quoted, ", "))
	}
	b.WriteString("}\n")

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}
