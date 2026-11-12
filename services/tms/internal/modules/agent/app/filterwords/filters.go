// Package filterwords reads what a list tool says it was narrowed by, which is
// written for the model in the tool's own field names and the ids it was
// given, as words a person reads: a field's label, its values, and every
// record by its name.
package filterwords

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/recordids"
	"github.com/emoss08/trenova/shared/textcase"
)

// MaxRecords bounds how many records one call's filters are named from. A
// filter over more says how many it left out.
const MaxRecords = 50

// lineSeparator stands between the filters of one line.
const lineSeparator = " · "

var (
	codedValue  = regexp.MustCompile(`\b[A-Z][a-z0-9]+(?:[A-Z][a-z0-9]+)+\b`)
	idSuffixes  = []string{"Ids", "IDs", "Id", "ID"}
	impliedLead = []string{
		filtercatalog.OperatorPhrase(dbtype.OpIn) + " ",
		filtercatalog.OperatorPhrase(dbtype.OpEqual) + " ",
	}
)

// Filter is one thing a list was narrowed by: "Customer" and the customers'
// names, "Status" and the statuses in words. A filter the tool worded as a
// phrase of its own ("running late") has no label.
type Filter struct {
	Label string `json:"label,omitempty"`
	Value string `json:"value"`

	text  string
	coded bool
}

// resultKey is where a list tool's result names what it applied, in the
// model's terms.
const resultKey = "searchedFor"

// OfResult reads what a list tool's result says it applied against the
// arguments it was called with. A result that is not a list's has none.
func OfResult(result any, arguments map[string]any) []Filter {
	fields, ok := result.(map[string]any)
	if !ok {
		return nil
	}

	var terms []string
	switch typed := fields[resultKey].(type) {
	case []string:
		terms = typed
	case []any:
		terms = make([]string, 0, len(typed))
		for _, entry := range typed {
			if term, isText := entry.(string); isText {
				terms = append(terms, term)
			}
		}
	}

	return Of(terms, arguments)
}

// Of reads the terms a list tool reported against the arguments it was
// called with. A term that opens with one of the call's own fields is that
// field's filter; any other is a phrase. Every record id is taken out: named
// by Name when its record has a name, counted when it has none.
func Of(terms []string, arguments map[string]any) []Filter {
	fields := filterFields(arguments)
	filters := make([]Filter, 0, len(terms))
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" || filtercatalog.Unfiltered(term) {
			continue
		}

		filter := Filter{text: term}
		if name, rest, found := strings.Cut(term, " "); found {
			if _, known := fields[name]; known {
				filter.Label = filterLabel(name)
				filter.text = withoutImpliedLead(strings.TrimSpace(rest))
				filter.coded = assistantartifact.CodedKey(name)
			}
		}
		filter.Value = filter.render(nil)
		if filter.Value == "" {
			continue
		}
		filters = append(filters, filter)
	}

	return filters
}

// Records is the ids of the records the filters name, each once, at most
// MaxRecords of them.
func Records(filters []Filter) []string {
	texts := make([]string, 0, len(filters))
	for idx := range filters {
		texts = append(texts, filters[idx].text)
	}

	return recordids.Find(strings.Join(texts, "\n"), MaxRecords)
}

// Name writes each filter with its records' names in place of their ids. A
// record name answers "" for has no name.
func Name(filters []Filter, name func(id string) string) {
	for idx := range filters {
		filters[idx].Value = filters[idx].render(name)
	}
}

// Line is the filters on one line, each label before its value: "Customer
// Granite Textiles · Status New, Delayed".
func Line(filters []Filter) string {
	parts := make([]string, 0, len(filters))
	for idx := range filters {
		part := strings.TrimSpace(filters[idx].Label + " " + filters[idx].Value)
		if part != "" {
			parts = append(parts, part)
		}
	}

	return strings.Join(parts, lineSeparator)
}

func (f *Filter) render(name func(id string) string) string {
	text := f.text
	ids := recordids.Find(text, math.MaxInt)
	unnamed := 0
	for _, id := range ids {
		label := ""
		if name != nil {
			label = strings.TrimSpace(name(id))
		}
		if label == "" {
			unnamed++
			continue
		}
		text = strings.ReplaceAll(text, id, label)
	}
	if unnamed > 0 {
		text = strings.Trim(recordids.Strip(text), ", ")
		switch {
		case text == "":
			return strconv.Itoa(unnamed) + " selected"
		case unnamed == len(ids):
			text += " (" + strconv.Itoa(unnamed) + " selected)"
		default:
			text += " and " + strconv.Itoa(unnamed) + " more"
		}
	}
	if f.coded {
		text = codedValue.ReplaceAllStringFunc(text, textcase.HumanizeCamelCaseSentence)
	}

	return text
}

func filterFields(arguments map[string]any) map[string]struct{} {
	fields := make(map[string]struct{}, len(arguments))
	for key, value := range arguments {
		fields[key] = struct{}{}
		entries, _ := value.([]any)
		for _, entry := range entries {
			record, _ := entry.(map[string]any)
			if field, _ := record["field"].(string); field != "" {
				fields[field] = struct{}{}
			}
		}
	}

	return fields
}

func filterLabel(name string) string {
	for _, suffix := range idSuffixes {
		trimmed, ok := strings.CutSuffix(name, suffix)
		if ok && trimmed != "" && assistantartifact.EndsWithWord(name, []string{suffix}) {
			name = trimmed
			break
		}
	}

	return assistantartifact.DisplayLabel(name, assistantartifact.DisplayText)
}

func withoutImpliedLead(value string) string {
	for _, lead := range impliedLead {
		if rest, ok := strings.CutPrefix(value, lead); ok {
			return rest
		}
	}

	return value
}
