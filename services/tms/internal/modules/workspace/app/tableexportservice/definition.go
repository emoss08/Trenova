package tableexportservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const MaxColumns = 100

type Column struct {
	Field string
	Label string
}

type View struct {
	Resource     permission.Resource
	Columns      []Column
	Query        string
	FieldFilters []domaintypes.FieldFilter
	FilterGroups []domaintypes.FilterGroup
	Sort         []domaintypes.SortField
}

type Built struct {
	Definition *ir.Definition
	Skipped    []string
}

//nolint:exhaustive // count operators filter relations, which a report export cannot express
var tableOperators = map[dbtype.Operator]ir.Op{
	dbtype.OpEqual:              ir.OpEq,
	dbtype.OpNotEqual:           ir.OpNe,
	dbtype.OpGreaterThan:        ir.OpGt,
	dbtype.OpGreaterThanOrEqual: ir.OpGte,
	dbtype.OpLessThan:           ir.OpLt,
	dbtype.OpLessThanOrEqual:    ir.OpLte,
	dbtype.OpContains:           ir.OpContains,
	dbtype.OpStartsWith:         ir.OpStartsWith,
	dbtype.OpEndsWith:           ir.OpEndsWith,
	dbtype.OpLike:               ir.OpLike,
	dbtype.OpILike:              ir.OpILike,
	dbtype.OpIn:                 ir.OpIn,
	dbtype.OpNotIn:              ir.OpNotIn,
	dbtype.OpIsNull:             ir.OpIsNull,
	dbtype.OpIsNotNull:          ir.OpIsNotNull,
	dbtype.OpDateRange:          ir.OpBetween,
	dbtype.OpLastNDays:          ir.OpLastNDays,
	dbtype.OpNextNDays:          ir.OpNextNDays,
	dbtype.OpToday:              ir.OpToday,
	dbtype.OpYesterday:          ir.OpYesterday,
	dbtype.OpTomorrow:           ir.OpTomorrow,
	dbtype.OpThisWeek:           ir.OpThisWeek,
	dbtype.OpLastWeek:           ir.OpLastWeek,
	dbtype.OpThisMonth:          ir.OpThisMonth,
	dbtype.OpLastMonth:          ir.OpLastMonth,
	dbtype.OpThisQuarter:        ir.OpThisQuarter,
	dbtype.OpLastQuarter:        ir.OpLastQuarter,
	dbtype.OpThisYear:           ir.OpThisYear,
	dbtype.OpLastYear:           ir.OpLastYear,
}

type resolvedField struct {
	path   string
	field  *analytics.Field
	toMany bool
}

func resolveField(entity *analytics.Entity, apiField string) (resolvedField, bool) {
	segments := strings.Split(apiField, ".")
	current := entity
	names := make([]string, 0, len(segments))
	toMany := false
	for _, segment := range segments[:len(segments)-1] {
		edge, ok := current.Edge(segment)
		if !ok {
			return resolvedField{}, false
		}
		names = append(names, edge.Name)
		toMany = toMany || edge.ToMany
		current = edge.Target
	}

	leaf := segments[len(segments)-1]
	field, key, ok := fieldByAPIName(current, leaf)
	if !ok {
		return resolvedField{}, false
	}
	if key != leaf && strings.Contains(key, ".") {
		edgeName, _, _ := strings.Cut(key, ".")
		edge, _ := current.Edge(edgeName)
		toMany = toMany || edge.ToMany
	}
	names = append(names, key)
	return resolvedField{path: strings.Join(names, "."), field: field, toMany: toMany}, true
}

func fieldByAPIName(entity *analytics.Entity, name string) (*analytics.Field, string, bool) {
	if field, ok := entity.Field(name); ok {
		return field, name, true
	}
	for _, field := range entity.Fields {
		if reportaccess.APIKey(entity.Table.Columns[field.Column].Name) == name {
			return field, field.Key, true
		}
	}
	if edgeName, isRef := strings.CutSuffix(name, "Id"); isRef {
		if edge, ok := entity.Edge(edgeName); ok && !edge.ToMany {
			if id, found := edge.Target.Field("id"); found {
				return id, edge.Name + ".id", true
			}
		}
	}
	return nil, "", false
}

func Build(kinds *reportaccess.Resolver, view *View) (*Built, error) {
	entity, ok := kinds.RootEntity(view.Resource)
	if !ok {
		return nil, errortypes.NewValidationError(
			"resource",
			errortypes.ErrInvalid,
			"This table cannot be exported as a report",
		)
	}
	if strings.TrimSpace(view.Query) != "" {
		return nil, errortypes.NewValidationError(
			"query",
			errortypes.ErrInvalid,
			"A search cannot be exported as a report; use filters instead, or export from the browser",
		)
	}
	if len(view.Columns) == 0 || len(view.Columns) > MaxColumns {
		return nil, errortypes.NewValidationError(
			"columns",
			errortypes.ErrInvalid,
			"Export between 1 and {0} columns",
			MaxColumns,
		)
	}

	definition := &ir.Definition{Version: ir.Version, Entity: entity.Key}
	built := &Built{Definition: definition}
	columnIDs := make(map[string]string, len(view.Columns))
	for _, column := range view.Columns {
		resolved, found := resolveField(entity, column.Field)
		if !found || resolved.toMany || !resolved.field.Groupable {
			built.Skipped = append(built.Skipped, column.Label)
			continue
		}
		id := fmt.Sprintf("c%d", len(definition.Columns))
		columnIDs[column.Field] = id
		definition.Columns = append(definition.Columns, ir.Column{
			ID:        id,
			Label:     column.Label,
			Dimension: &ir.Dimension{Field: resolved.path},
		})
	}
	if len(definition.Columns) == 0 {
		return nil, errortypes.NewValidationError(
			"columns",
			errortypes.ErrInvalid,
			"None of the shown columns can be exported as a report",
		)
	}

	filters, err := buildFilters(entity, view)
	if err != nil {
		return nil, err
	}
	definition.Filters = filters

	for _, sort := range view.Sort {
		id, shown := columnIDs[sort.Field]
		if !shown {
			continue
		}
		definition.Sort = append(definition.Sort, ir.Sort{
			Column: id,
			Desc:   strings.EqualFold(string(sort.Direction), "desc"),
		})
	}

	return built, nil
}

func buildFilters(entity *analytics.Entity, view *View) (*ir.Filter, error) {
	if len(view.FieldFilters) == 0 && len(view.FilterGroups) == 0 {
		return nil, nil //nolint:nilnil // an unfiltered view has no filter group
	}

	convert := func(at string, filter domaintypes.FieldFilter) (ir.Filter, error) {
		resolved, ok := resolveField(entity, filter.Field)
		op, known := tableOperators[filter.Operator]
		if !ok || !known || !resolved.field.Filterable {
			return ir.Filter{}, errortypes.NewValidationError(
				at,
				errortypes.ErrInvalid,
				"The filter on {0} cannot be exported as a report; remove it or export from the browser",
				filter.Field,
			)
		}
		return ir.Filter{Field: resolved.path, Op: op, Value: filter.Value}, nil
	}

	children := make([]ir.Filter, 0, len(view.FieldFilters)+len(view.FilterGroups))
	for idx, filter := range view.FieldFilters {
		converted, err := convert(fmt.Sprintf("fieldFilters[%d]", idx), filter)
		if err != nil {
			return nil, err
		}
		children = append(children, converted)
	}
	for groupIdx, group := range view.FilterGroups {
		either := make([]ir.Filter, 0, len(group.Filters))
		for idx, filter := range group.Filters {
			converted, err := convert(
				fmt.Sprintf("filterGroups[%d].filters[%d]", groupIdx, idx),
				filter,
			)
			if err != nil {
				return nil, err
			}
			either = append(either, converted)
		}
		if len(either) > 0 {
			children = append(children, *ir.Or(either...))
		}
	}
	return ir.And(children...), nil
}
