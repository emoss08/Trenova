package recordnumbers

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/agent/ports/recordnumberports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramQuery      = "query"
	paramFilters    = "filters"
	paramField      = "field"
	paramOperator   = "operator"
	paramValues     = "values"
	operatorIn      = "in"
	maxListedNumber = 25
	maxNumberRunes  = 100
)

var queryJoiners = []string{"and", "or", "&", "+"}

type ListQuery struct {
	Actor      *permissionports.RequestActor
	DataAccess permission.FieldSensitivity
	Resource   permission.Resource
	Schema     map[string]any
	Args       map[string]any
}

type ListedNumbers struct {
	Args map[string]any
	Note string
}

func (r *Resolver) ListQuery(ctx context.Context, q *ListQuery) (ListedNumbers, error) {
	unchanged := ListedNumbers{Args: q.Args}
	query, _ := q.Args[paramQuery].(string)
	numbers := queryNumbers(query)
	if len(numbers) == 0 {
		return unchanged, nil
	}
	kind := recordkind.Kind(q.Resource)
	numbering, readable := r.readable(
		ctx, fieldsensitivity.NewAgentCeilings(r.permissions, q.Actor, q.DataAccess), kind,
	)
	if !readable || !filtersIn(q.Schema, numbering.Field) {
		return unchanged, nil
	}

	asked := make(map[recordkind.Kind][]string, 1)
	asked[kind] = numbers
	found, err := r.records.FindNumbers(ctx, q.Actor.TenantInfo(), asked)
	if err != nil {
		return ListedNumbers{}, fmt.Errorf("read the records these numbers name: %w", err)
	}
	stored := make([]any, 0, len(numbers))
	for _, number := range numbers {
		records := found[kind][number]
		if len(records) != 1 {
			return unchanged, nil
		}
		stored = append(stored, records[0].Number)
	}

	return ListedNumbers{
		Args: numberedArgs(q.Args, numbering.Field, stored),
		Note: listedNote(kind, numbering, stored),
	}, nil
}

func (r *Resolver) readable(
	ctx context.Context,
	ceilings *fieldsensitivity.AgentCeilings,
	kind recordkind.Kind,
) (recordnumberports.Numbering, bool) {
	numbering, ok := r.records.Numbering(kind)
	if !ok {
		return recordnumberports.Numbering{}, false
	}
	resource := kind.Resource()
	if !ceilings.Reads(ctx, resource) ||
		!fieldsensitivity.Visible(
			r.registry,
			resource,
			numbering.Field,
			ceilings.For(ctx, resource),
		) {
		return recordnumberports.Numbering{}, false
	}

	return numbering, true
}

func queryNumbers(query string) []string {
	words := strings.FieldsFunc(query, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == ';'
	})
	numbers := make([]string, 0, len(words))
	for _, word := range words {
		if slices.ContainsFunc(queryJoiners, func(joiner string) bool {
			return strings.EqualFold(word, joiner)
		}) {
			continue
		}
		if len([]rune(word)) > maxNumberRunes || !strings.ContainsFunc(word, unicode.IsDigit) ||
			pulid.LooksLike(word) {
			return nil
		}
		if !slices.Contains(numbers, word) {
			numbers = append(numbers, word)
		}
	}
	if len(numbers) > maxListedNumber {
		return nil
	}

	return numbers
}

func filtersIn(schema map[string]any, field string) bool {
	properties, _ := schema[toolschema.KeyProperties].(map[string]any)
	if _, takesQuery := properties[paramQuery].(map[string]any); !takesQuery {
		return false
	}
	filters, _ := properties[paramFilters].(map[string]any)
	items, _ := filters[toolschema.KeyItems].(map[string]any)
	fields, _ := items[toolschema.KeyProperties].(map[string]any)
	fieldSchema, _ := fields[paramField].(map[string]any)
	operatorSchema, _ := fields[paramOperator].(map[string]any)
	if _, takesValues := fields[paramValues]; !takesValues {
		return false
	}

	return enumHolds(fieldSchema, field) && enumHolds(operatorSchema, operatorIn)
}

func enumHolds(property map[string]any, value string) bool {
	switch values := property[toolschema.KeyEnum].(type) {
	case []string:
		return slices.Contains(values, value)
	case []any:
		return slices.Contains(values, any(value))
	}

	return false
}

func numberedArgs(args map[string]any, field string, numbers []any) map[string]any {
	out := make(map[string]any, len(args))
	for key, value := range args {
		if key != paramQuery {
			out[key] = value
		}
	}
	filters, _ := args[paramFilters].([]any)
	out[paramFilters] = append(slices.Clone(filters), map[string]any{
		paramField:    field,
		paramOperator: operatorIn,
		paramValues:   numbers,
	})

	return out
}

func listedNote(kind recordkind.Kind, numbering recordnumberports.Numbering, numbers []any) string {
	written := make([]string, 0, len(numbers))
	for _, number := range numbers {
		written = append(written, fmt.Sprint(number))
	}
	noun := kind.Noun()
	if len(numbers) > 1 {
		noun = inflect.PluralNoun(noun)
	}

	return fmt.Sprintf(
		"The words were %s, so this lists exactly the %s they name: %s.",
		inflect.PluralNoun(numbering.Name), noun, strings.Join(written, ", "),
	)
}
