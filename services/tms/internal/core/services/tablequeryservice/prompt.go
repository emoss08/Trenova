package tablequeryservice

import (
	"fmt"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

const systemPrompt = `You turn a question about a table into filters on that table.

You are given one entity, the fields it can be narrowed by, and what each field
accepts. Name only those fields and only the operators listed for each one.

Rules:
- Never invent a field, an operator or an enum value. If the question asks for
  something the fields cannot express, leave it out of "filters" and put it in
  "unresolved" with the phrase and why.
- Dates are calendar days: send YYYY-MM-DD, or today, tomorrow, yesterday.
  For a window use lastndays or nextndays with "days". Never compute an epoch.
- Put free text — a name, a code, part of a number — in "query" rather than
  inventing a contains filter, unless a text field is clearly the one meant.
- "sortBy" is only a field marked sortable.
- If the question narrows a view that already has filters, return the whole
  intended state, not just the change.

Return only the object the schema describes.`

// buildContext hands the model the catalogue and the question, fenced apart.
func buildContext(
	resource filtercatalog.Resource,
	prompt string,
	current CurrentView,
) serviceports.DelimitedContext {
	sections := []serviceports.ContextSection{
		{Title: "entity", Trusted: true, Content: describeResource(resource)},
		{Title: "question", Content: prompt},
	}

	if described := describeCurrent(current); described != "" {
		sections = append(sections, serviceports.ContextSection{
			Title:   "current_view",
			Trusted: true,
			Content: described,
		})
	}

	return serviceports.DelimitedContext{Sections: sections}
}

// describeResource is the catalogue as lines, which costs a fraction of the
// same thing as JSON and reads the same to a model.
func describeResource(resource filtercatalog.Resource) string {
	var b strings.Builder
	b.WriteString(resource.Entity)
	if resource.Summary != "" {
		b.WriteString(": ")
		b.WriteString(resource.Summary)
	}
	b.WriteString("\n\nfields:\n")

	for _, field := range resource.Fields {
		fmt.Fprintf(&b, "- %s (%s)", field.Name, field.Kind)
		if len(field.Values) > 0 {
			fmt.Fprintf(&b, " one of: %s", strings.Join(field.Values, ", "))
		}
		if field.Sortable {
			b.WriteString(" [sortable]")
		}
		if field.Note != "" {
			fmt.Fprintf(&b, " — %s", field.Note)
		}
		b.WriteString("\n  operators: ")
		b.WriteString(operatorList(field))
		b.WriteString("\n")
	}

	return b.String()
}

func operatorList(field filtercatalog.Field) string {
	operators := filtercatalog.Operators(field.Kind)
	names := make([]string, 0, len(operators))
	for _, operator := range operators {
		names = append(names, string(operator))
	}

	return strings.Join(names, ", ")
}

// describeCurrent states the filters already applied, so "and only Acme's"
// keeps the rest rather than replacing it.
func describeCurrent(current CurrentView) string {
	lines := make([]string, 0, len(current.FieldFilters)+2)
	if current.Query != "" {
		lines = append(lines, "text: "+current.Query)
	}
	for _, filter := range current.FieldFilters {
		lines = append(lines, describeFilter(filter))
	}
	for _, sort := range current.Sort {
		lines = append(lines, fmt.Sprintf("sorted by %s %s", sort.Field, sort.Direction))
	}

	return strings.Join(lines, "\n")
}

func describeFilter(filter domaintypes.FieldFilter) string {
	phrase := filtercatalog.OperatorPhrase(filter.Operator)
	if filter.Value == nil {
		return fmt.Sprintf("%s %s", filter.Field, phrase)
	}

	return fmt.Sprintf("%s %s %v", filter.Field, phrase, filter.Value)
}

// outputSchema is built from the resource, so the field names and enum values
// are constrained by the schema itself rather than only by the prompt. A
// provider that enforces the schema then cannot return a column that does not
// exist, and one that does not enforce it is caught by the compiler anyway.
func outputSchema(resource filtercatalog.Resource) map[string]any {
	operators := make([]string, 0, 16)
	seen := make(map[string]bool, 16)
	for _, field := range resource.Fields {
		for _, operator := range filtercatalog.Operators(field.Kind) {
			if seen[string(operator)] {
				continue
			}
			seen[string(operator)] = true
			operators = append(operators, string(operator))
		}
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"filters"},
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Free text to match, when the question names something rather than a field value.",
			},
			"filters": map[string]any{
				"type":     "array",
				"maxItems": maxComposedFilters,
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"field", "operator"},
					"properties": map[string]any{
						"field":    map[string]any{"type": "string", "enum": resource.FieldNames()},
						"operator": map[string]any{"type": "string", "enum": operators},
						"value":    map[string]any{"type": "string"},
						"values": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
						"days": map[string]any{
							"type":    "integer",
							"minimum": 1,
							"maximum": filtercatalog.MaxRelativeDays,
						},
					},
				},
			},
			"sortBy": map[string]any{
				"type": "string",
				"enum": resource.SortableNames(),
			},
			"sortDirection": map[string]any{"type": "string", "enum": []string{"asc", "desc"}},
			"unresolved": map[string]any{
				"type":        "array",
				"description": "Anything the question asked for that these fields cannot express.",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"phrase", "reason"},
					"properties": map[string]any{
						"phrase": map[string]any{"type": "string"},
						"reason": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}
