package recordquerytools

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const (
	opBetween    = "between"
	maxConds     = 50
	maxGroups    = 10
	maxSortField = 5
	dateLayout   = "2006-01-02"
	maxSuggested = 30
)

var ErrBadArgument = errors.New("invalid argument")

var operators = []string{
	"eq", "ne", "in", "notin", "contains", "startswith", "gt", "gte", "lt", "lte",
	opBetween, "isnull", "isnotnull", "lastndays", "nextndays", "today",
}

type condition struct {
	Field  string
	Op     string
	Value  any
	Values []any
	Days   int
}

type fieldSet struct {
	config  *domaintypes.FieldConfiguration
	extras  []querytool.Extra
	aliases map[string]string
}

func (f *fieldSet) resolve(field string) (name string, extra bool, ok bool) {
	if target, aliased := f.aliases[field]; aliased {
		field = target
	}
	if slices.ContainsFunc(f.extras, func(e querytool.Extra) bool { return e.Name == field }) {
		return field, true, true
	}
	if f.config != nil && f.config.FilterableFields[field] {
		return field, false, true
	}

	return field, false, false
}

func (f *fieldSet) sortable(field string) (string, bool) {
	if target, aliased := f.aliases[field]; aliased {
		field = target
	}

	return field, f.config != nil && f.config.SortableFields[field]
}

func (f *fieldSet) suggestions() []string {
	names := make([]string, 0, len(f.extras)+maxSuggested)
	for _, extra := range f.extras {
		names = append(names, extra.Name)
	}
	for alias := range f.aliases {
		names = append(names, alias)
	}
	for _, field := range f.publicFields() {
		if len(names) >= maxSuggested {
			break
		}
		names = append(names, field)
	}

	return names
}

func (f *fieldSet) publicFields() []string {
	if f.config == nil {
		return nil
	}
	out := make([]string, 0, len(f.config.FilterableFields))
	for field, on := range f.config.FilterableFields {
		if on && public(field) {
			out = append(out, field)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		di, dj := strings.Count(out[i], "."), strings.Count(out[j], ".")
		if di != dj {
			return di < dj
		}

		return out[i] < out[j]
	})

	return out
}

func public(field string) bool {
	last := field
	if idx := strings.LastIndex(field, "."); idx >= 0 {
		last = field[idx+1:]
	}
	switch last {
	case "businessUnitId", "organizationId", "version", "placeId", "searchVector":
		return false
	}

	return true
}

type compiled struct {
	where   querytool.Where
	applied []Applied
}

func compile(
	fields *fieldSet,
	and []condition,
	or [][]condition,
	sortBy []sortField,
	zone *time.Location,
) (*compiled, error) {
	if len(and) > maxConds {
		return nil, fmt.Errorf("%w: where takes at most %d conditions", ErrBadArgument, maxConds)
	}
	if len(or) > maxGroups {
		return nil, fmt.Errorf("%w: or takes at most %d groups", ErrBadArgument, maxGroups)
	}
	out := &compiled{where: querytool.Where{Extras: map[string]any{}}}
	for idx := range and {
		if err := out.add(fields, &and[idx], 0, zone); err != nil {
			return nil, err
		}
	}
	for groupIdx, group := range or {
		filters := make([]domaintypes.FieldFilter, 0, len(group))
		for idx := range group {
			cond := &group[idx]
			name, extra, ok := fields.resolve(cond.Field)
			if !ok {
				return nil, unknownField(fields, cond.Field)
			}
			if extra {
				return nil, fmt.Errorf("%w: %q cannot sit in an or group; put it in where",
					ErrBadArgument, cond.Field)
			}
			compiledFilters, err := filtersOf(name, cond, zone)
			if err != nil {
				return nil, err
			}
			filters = append(filters, compiledFilters...)
			out.applied = append(out.applied, Applied{Field: cond.Field, Operator: cond.Op,
				Value: shown(cond), Group: groupIdx + 1})
		}
		out.where.Groups = append(out.where.Groups, domaintypes.FilterGroup{Filters: filters})
	}
	if len(sortBy) > maxSortField {
		return nil, fmt.Errorf("%w: sort takes at most %d fields", ErrBadArgument, maxSortField)
	}
	for _, field := range sortBy {
		name, ok := fields.sortable(field.Field)
		if !ok {
			return nil, fmt.Errorf("%w: %q cannot be sorted by; describe_entity lists the sortable fields",
				ErrBadArgument, field.Field)
		}
		direction := dbtype.SortDirection(strings.ToLower(field.Direction))
		if direction == "" {
			direction = dbtype.SortDirectionAsc
		}
		out.where.Sort = append(out.where.Sort, domaintypes.SortField{Field: name, Direction: direction})
	}

	return out, nil
}

func (c *compiled) add(fields *fieldSet, cond *condition, group int, zone *time.Location) error {
	name, extra, ok := fields.resolve(cond.Field)
	if !ok {
		return unknownField(fields, cond.Field)
	}
	if !slices.Contains(operators, cond.Op) {
		return fmt.Errorf("%w: operator %q on %q is not one of %s", ErrBadArgument, cond.Op,
			cond.Field, strings.Join(operators, ", "))
	}
	c.applied = append(c.applied, Applied{Field: cond.Field, Operator: cond.Op, Value: shown(cond), Group: group})
	if extra {
		switch {
		case len(cond.Values) > 0:
			c.where.Extras[name] = cond.Values
		case cond.Op == "gte" || cond.Op == "gt":
			c.where.Extras[name] = number(cond.Value)
		default:
			c.where.Extras[name] = cond.Value
		}

		return nil
	}
	filters, err := filtersOf(name, cond, zone)
	if err != nil {
		return err
	}
	c.where.Filters = append(c.where.Filters, filters...)

	return nil
}

func filtersOf(name string, cond *condition, zone *time.Location) ([]domaintypes.FieldFilter, error) {
	switch cond.Op {
	case opBetween:
		if len(cond.Values) != 2 {
			return nil, fmt.Errorf("%w: between on %q takes values [from, to]", ErrBadArgument, cond.Field)
		}
		return []domaintypes.FieldFilter{
			{Field: name, Operator: dbtype.OpGreaterThanOrEqual, Value: dateValue(cond.Values[0], zone, false)},
			{Field: name, Operator: dbtype.OpLessThanOrEqual, Value: dateValue(cond.Values[1], zone, true)},
		}, nil
	case "in", "notin":
		if len(cond.Values) == 0 {
			return nil, fmt.Errorf("%w: %s on %q takes values", ErrBadArgument, cond.Op, cond.Field)
		}
		return []domaintypes.FieldFilter{{Field: name, Operator: dbtype.Operator(cond.Op), Value: cond.Values}}, nil
	case "lastndays", "nextndays":
		if cond.Days <= 0 {
			return nil, fmt.Errorf("%w: %s on %q takes days", ErrBadArgument, cond.Op, cond.Field)
		}
		return []domaintypes.FieldFilter{{Field: name, Operator: dbtype.Operator(cond.Op), Value: cond.Days}}, nil
	case "isnull", "isnotnull", "today":
		return []domaintypes.FieldFilter{{Field: name, Operator: dbtype.Operator(cond.Op)}}, nil
	case "gt", "gte":
		return []domaintypes.FieldFilter{{Field: name, Operator: dbtype.Operator(cond.Op),
			Value: dateValue(cond.Value, zone, false)}}, nil
	case "lt", "lte":
		return []domaintypes.FieldFilter{{Field: name, Operator: dbtype.Operator(cond.Op),
			Value: dateValue(cond.Value, zone, true)}}, nil
	}
	if cond.Value == nil {
		return nil, fmt.Errorf("%w: %s on %q takes a value", ErrBadArgument, cond.Op, cond.Field)
	}

	return []domaintypes.FieldFilter{{Field: name, Operator: dbtype.Operator(cond.Op), Value: cond.Value}}, nil
}

func dateValue(raw any, zone *time.Location, end bool) any {
	text, ok := raw.(string)
	if !ok {
		return raw
	}
	day, err := time.ParseInLocation(dateLayout, text, zone)
	if err != nil {
		return raw
	}
	if end {
		return day.Add(24*time.Hour - time.Second).Unix()
	}

	return day.Unix()
}

func number(raw any) any {
	if value, ok := raw.(float64); ok {
		return value
	}

	return raw
}

func shown(cond *condition) any {
	switch {
	case len(cond.Values) > 0:
		return cond.Values
	case cond.Days > 0:
		return cond.Days
	}

	return cond.Value
}

func unknownField(fields *fieldSet, field string) error {
	suggested := fields.suggestions()
	if len(suggested) == 0 {
		return fmt.Errorf("%w: %q is not a field this entity can be filtered by, and it takes no "+
			"where conditions at all; use query or the arguments describe_entity lists",
			ErrBadArgument, field)
	}

	return fmt.Errorf("%w: %q is not a field this entity can be filtered by; valid fields include %s "+
		"(describe_entity lists every one)", ErrBadArgument, field, strings.Join(suggested, ", "))
}
