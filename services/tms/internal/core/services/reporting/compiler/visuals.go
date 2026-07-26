package compiler

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/floatutils"
)

const (
	pivotIDSeparator = ":"
	pivotOtherValue  = "__other__"
)

// outputRole records what a chart may do with an output column: dimensions
// position a series, value columns are what gets drawn.
type outputRole struct {
	isDim bool
}

// outputRoles enumerates the columns the query will actually return, expanding
// pivoted measures into the one column per bucket a chart can address.
func outputRoles(v *validatedDef) map[string]outputRole {
	pivot := v.def.Pivot
	pivoted := pivotedMeasureIDs(pivot)

	roles := make(map[string]outputRole, len(v.columns))
	for i := range v.columns {
		col := &v.columns[i]
		if col.spec.Kind == report.ColumnKindDimension {
			roles[col.spec.ID] = outputRole{isDim: true}
			continue
		}
		if !pivoted[col.spec.ID] {
			roles[col.spec.ID] = outputRole{}
			continue
		}
		for _, value := range pivot.Values {
			roles[col.spec.ID+pivotIDSeparator+value] = outputRole{}
		}
		if pivot.IncludeOther {
			roles[col.spec.ID+pivotIDSeparator+pivotOtherValue] = outputRole{}
		}
	}
	return roles
}

func validateTotals(v *validatedDef, multiErr *errortypes.MultiError) {
	if !v.def.Totals {
		return
	}
	if !v.def.HasMeasures() {
		multiErr.Add("definition.totals", errortypes.ErrInvalid,
			"A total row needs at least one measure to total")
		return
	}
	// The total is computed from the underlying rows so averages stay averages;
	// a measure filter removes whole groups after aggregation, and a total that
	// silently re-included them would not match the rows on screen.
	if !v.def.Having.IsEmpty() {
		multiErr.Add(
			"definition.totals",
			errortypes.ErrInvalid,
			"A total row cannot be combined with measure filters — move the condition to the row filters, or turn totals off",
		)
	}
}

func validateCharts(v *validatedDef, multiErr *errortypes.MultiError) {
	charts := v.def.Charts
	if len(charts) == 0 {
		return
	}
	if len(charts) > report.MaxCharts {
		multiErr.Add("definition.charts", errortypes.ErrInvalid,
			fmt.Sprintf("A report can carry at most %d charts", report.MaxCharts))
		return
	}

	roles := outputRoles(v)
	seen := make(map[string]bool, len(charts))

	for i := range charts {
		chart := &charts[i]
		fieldPath := fmt.Sprintf("definition.charts[%d]", i)

		if chart.ID == "" {
			multiErr.Add(fieldPath+".id", errortypes.ErrRequired, "Chart ID is required")
			continue
		}
		if seen[chart.ID] {
			multiErr.Add(fieldPath+".id", errortypes.ErrInvalid, "Duplicate chart ID")
			continue
		}
		seen[chart.ID] = true

		if !chart.Type.IsValid() {
			multiErr.Add(fieldPath+".type", errortypes.ErrInvalid,
				fmt.Sprintf("Unknown chart type %q", chart.Type))
			continue
		}

		validateChartAxes(multiErr, chart, roles, fieldPath)
		validateChartSeries(multiErr, chart, roles, fieldPath)
		validateChartOptions(multiErr, chart, roles, fieldPath)
	}
}

func validateChartAxes(
	multiErr *errortypes.MultiError,
	chart *report.ChartSpec,
	roles map[string]outputRole,
	fieldPath string,
) {
	//nolint:exhaustive // every other encoding shares the dimension-axis rule
	switch chart.Type {
	case report.ChartKPI:
		if chart.XColumnID != "" {
			multiErr.Add(fieldPath+".xColumnId", errortypes.ErrInvalid,
				"A KPI tile shows a single number and has no category axis")
		}
	case report.ChartMap:
		if chart.XColumnID != "" {
			multiErr.Add(fieldPath+".xColumnId", errortypes.ErrInvalid,
				"A map positions rows by coordinate and has no category axis")
		}
		validateChartCoordinates(multiErr, chart, roles, fieldPath)
	case report.ChartScatter:
		role, ok := roles[chart.XColumnID]
		switch {
		case chart.XColumnID == "":
			multiErr.Add(fieldPath+".xColumnId", errortypes.ErrRequired,
				"A scatter plot needs a measure on the horizontal axis")
		case !ok:
			multiErr.Add(fieldPath+".xColumnId", errortypes.ErrInvalid,
				fmt.Sprintf("Chart references unknown column %q", chart.XColumnID))
		case role.isDim:
			multiErr.Add(fieldPath+".xColumnId", errortypes.ErrInvalid,
				"A scatter plot needs a measure on the horizontal axis, not a grouping column")
		}
	default:
		role, ok := roles[chart.XColumnID]
		switch {
		case chart.XColumnID == "":
			multiErr.Add(fieldPath+".xColumnId", errortypes.ErrRequired,
				"Choose the column that groups this chart")
		case !ok:
			multiErr.Add(fieldPath+".xColumnId", errortypes.ErrInvalid,
				fmt.Sprintf("Chart references unknown column %q", chart.XColumnID))
		case !role.isDim:
			multiErr.Add(fieldPath+".xColumnId", errortypes.ErrInvalid,
				"Charts group by a dimension column, not by a measure")
		}
	}
}

// validateChartCoordinates insists on a real pair. A map with one coordinate
// is not a partial map — every row would land on a meridian, which reads as
// data rather than as the mistake it is.
func validateChartCoordinates(
	multiErr *errortypes.MultiError,
	chart *report.ChartSpec,
	roles map[string]outputRole,
	fieldPath string,
) {
	coordinates := []struct {
		id    string
		field string
		label string
	}{
		{chart.LatColumnID, ".latColumnId", "latitude"},
		{chart.LngColumnID, ".lngColumnId", "longitude"},
	}

	for _, coordinate := range coordinates {
		role, ok := roles[coordinate.id]
		switch {
		case coordinate.id == "":
			multiErr.Add(fieldPath+coordinate.field, errortypes.ErrRequired,
				fmt.Sprintf("A map needs a %s column", coordinate.label))
		case !ok:
			multiErr.Add(fieldPath+coordinate.field, errortypes.ErrInvalid,
				fmt.Sprintf("Chart references unknown column %q", coordinate.id))
		case role.isDim:
			// A coordinate grouped as a dimension still carries a usable
			// number, which is how "one pin per stop" is expressed.
			continue
		}
	}

	if chart.LatColumnID != "" && chart.LatColumnID == chart.LngColumnID {
		multiErr.Add(fieldPath+".lngColumnId", errortypes.ErrInvalid,
			"Latitude and longitude cannot be the same column")
	}

	if chart.LabelColumnID != "" {
		if _, ok := roles[chart.LabelColumnID]; !ok {
			multiErr.Add(fieldPath+".labelColumnId", errortypes.ErrInvalid,
				fmt.Sprintf("Chart references unknown column %q", chart.LabelColumnID))
		}
	}
}

func validateChartSeries(
	multiErr *errortypes.MultiError,
	chart *report.ChartSpec,
	roles map[string]outputRole,
	fieldPath string,
) {
	if len(chart.SeriesIDs) == 0 {
		// A map is positioned by its coordinates; a measure is optional there
		// and only scales the marker.
		if chart.Type == report.ChartMap {
			return
		}
		multiErr.Add(fieldPath+".seriesIds", errortypes.ErrRequired,
			"Choose at least one measure to plot")
		return
	}
	if len(chart.SeriesIDs) > report.MaxChartSeries {
		multiErr.Add(fieldPath+".seriesIds", errortypes.ErrInvalid,
			fmt.Sprintf("A chart can plot at most %d measures", report.MaxChartSeries))
		return
	}
	if chart.Type.SingleSeries() && len(chart.SeriesIDs) > 1 {
		multiErr.Add(fieldPath+".seriesIds", errortypes.ErrInvalid,
			fmt.Sprintf("A %s chart plots exactly one measure", chart.Type))
		return
	}

	seen := make(map[string]bool, len(chart.SeriesIDs))
	for _, id := range chart.SeriesIDs {
		if seen[id] {
			multiErr.Add(fieldPath+".seriesIds", errortypes.ErrInvalid,
				fmt.Sprintf("Measure %q is plotted twice", id))
			continue
		}
		seen[id] = true

		role, ok := roles[id]
		if !ok {
			multiErr.Add(fieldPath+".seriesIds", errortypes.ErrInvalid,
				fmt.Sprintf("Chart references unknown column %q", id))
			continue
		}
		if role.isDim {
			multiErr.Add(fieldPath+".seriesIds", errortypes.ErrInvalid,
				fmt.Sprintf("%q is a grouping column and cannot be plotted as a measure", id))
		}
	}
}

func validateChartOptions(
	multiErr *errortypes.MultiError,
	chart *report.ChartSpec,
	roles map[string]outputRole,
	fieldPath string,
) {
	if chart.Limit < 0 || chart.Limit > report.MaxChartCategory {
		multiErr.Add(
			fieldPath+".limit",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Chart category limit must be between 0 and %d",
				report.MaxChartCategory,
			),
		)
	}

	if chart.CompareID != "" {
		if chart.Type != report.ChartKPI {
			multiErr.Add(fieldPath+".compareId", errortypes.ErrInvalid,
				"Only KPI tiles show a comparison value")
		} else if role, ok := roles[chart.CompareID]; !ok || role.isDim {
			multiErr.Add(fieldPath+".compareId", errortypes.ErrInvalid,
				fmt.Sprintf("Comparison column %q is not a measure this report returns",
					chart.CompareID))
		}
	}

	validateChartGoal(multiErr, chart.Goal, roles, fieldPath+".goal")
}

func validateChartGoal(
	multiErr *errortypes.MultiError,
	goal *report.ChartGoal,
	roles map[string]outputRole,
	fieldPath string,
) {
	if goal.IsEmpty() {
		return
	}
	if goal.Value != nil && goal.ColumnID != "" {
		multiErr.Add(fieldPath, errortypes.ErrInvalid,
			"A target is either a fixed number or a column, not both")
		return
	}
	if goal.Value != nil {
		if !floatutils.IsFinite(*goal.Value) {
			multiErr.Add(fieldPath+".value", errortypes.ErrInvalid,
				"The target must be a finite number")
		}
		return
	}
	if role, ok := roles[goal.ColumnID]; !ok || role.isDim {
		multiErr.Add(fieldPath+".columnId", errortypes.ErrInvalid,
			fmt.Sprintf("Target column %q is not a measure this report returns", goal.ColumnID))
	}
}
