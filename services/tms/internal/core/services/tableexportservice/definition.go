package tableexportservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
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
	Definition *report.Definition
	Skipped    []string
}

func entityFor(
	catalog *reportcatalog.Catalog,
	resource permission.Resource,
) (*reportcatalog.Entity, bool) {
	for idx := range catalog.Entities {
		if catalog.Entities[idx].Resource == resource {
			return &catalog.Entities[idx], true
		}
	}
	return nil, false
}

func resolveRef(
	catalog *reportcatalog.Catalog,
	entity *reportcatalog.Entity,
	apiField string,
) (report.FieldRef, *reportcatalog.Field, bool) {
	segments := strings.Split(apiField, ".")
	field := segments[len(segments)-1]
	path := segments[:len(segments)-1]

	target := entity
	if len(path) > 0 {
		resolved, _, err := catalog.ResolvePath(entity.Key, path)
		if err != nil || resolved == nil {
			return report.FieldRef{}, nil, false
		}
		target = resolved
	}

	spec, ok := target.Field(field)
	if !ok {
		return report.FieldRef{}, nil, false
	}
	return report.FieldRef{Path: path, Field: field}, spec, true
}

func Build(catalog *reportcatalog.Catalog, view *View) (*Built, error) {
	entity, ok := entityFor(catalog, view.Resource)
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
			fmt.Sprintf("Export between 1 and %d columns", MaxColumns),
		)
	}

	definition := &report.Definition{IRVersion: report.CurrentIRVersion, Entity: entity.Key}
	built := &Built{Definition: definition}
	columnIDs := make(map[string]string, len(view.Columns))
	for _, column := range view.Columns {
		ref, _, resolved := resolveRef(catalog, entity, column.Field)
		if !resolved {
			built.Skipped = append(built.Skipped, column.Label)
			continue
		}
		id := fmt.Sprintf("c%d", len(definition.Columns))
		columnIDs[column.Field] = id
		definition.Columns = append(definition.Columns, report.ColumnSpec{
			ID:    id,
			Ref:   ref,
			Kind:  report.ColumnKindDimension,
			Label: column.Label,
		})
	}
	if len(definition.Columns) == 0 {
		return nil, errortypes.NewValidationError(
			"columns",
			errortypes.ErrInvalid,
			"None of the shown columns can be exported as a report",
		)
	}

	filters, err := buildFilters(catalog, entity, view)
	if err != nil {
		return nil, err
	}
	definition.Filters = filters

	for _, sort := range view.Sort {
		id, shown := columnIDs[sort.Field]
		if !shown {
			continue
		}
		definition.Sort = append(
			definition.Sort,
			report.SortSpec{ColumnID: id, Direction: sort.Direction},
		)
	}

	return built, nil
}

func buildFilters(
	catalog *reportcatalog.Catalog,
	entity *reportcatalog.Entity,
	view *View,
) (*report.FilterGroup, error) {
	if len(view.FieldFilters) == 0 && len(view.FilterGroups) == 0 {
		return nil, nil //nolint:nilnil // an unfiltered view has no filter group
	}

	convert := func(field string, filter domaintypes.FieldFilter) (report.FieldFilter, error) {
		ref, spec, ok := resolveRef(catalog, entity, filter.Field)
		if !ok || !spec.Filterable {
			return report.FieldFilter{}, errortypes.NewValidationError(
				field,
				errortypes.ErrInvalid,
				fmt.Sprintf(
					"The filter on %q cannot be exported as a report; remove it or export from the browser",
					filter.Field,
				),
			)
		}
		return report.FieldFilter{Ref: ref, Operator: filter.Operator, Value: filter.Value}, nil
	}

	root := &report.FilterGroup{Op: report.BoolOpAnd}
	for idx, filter := range view.FieldFilters {
		converted, err := convert(fmt.Sprintf("fieldFilters[%d]", idx), filter)
		if err != nil {
			return nil, err
		}
		root.Filters = append(root.Filters, converted)
	}
	for groupIdx, group := range view.FilterGroups {
		either := report.FilterGroup{Op: report.BoolOpOr}
		for idx, filter := range group.Filters {
			converted, err := convert(
				fmt.Sprintf("filterGroups[%d].filters[%d]", groupIdx, idx),
				filter,
			)
			if err != nil {
				return nil, err
			}
			either.Filters = append(either.Filters, converted)
		}
		if len(either.Filters) > 0 {
			root.Groups = append(root.Groups, either)
		}
	}
	return root, nil
}
