package compiler

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

func (e *emitter) buildOutputColumns() ([]outputColumn, error) {
	pivot := e.v.def.Pivot
	pivotedMeasures := make(map[string]bool)
	if pivot != nil {
		for _, id := range pivot.MeasureIDs {
			pivotedMeasures[id] = true
		}
	}

	outputs := make([]outputColumn, 0, len(e.v.columns))

	for i := range e.v.columns {
		col := &e.v.columns[i]

		if col.spec.Kind == report.ColumnKindDimension {
			expr, err := e.dimensionExpr(col.ref, columnGrouping(col.spec))
			if err != nil {
				return nil, err
			}
			outputs = append(outputs, outputColumn{
				id:            col.spec.ID,
				sourceID:      col.spec.ID,
				explicitLabel: col.spec.Label != "",
				expr:          applyTransform(expr, col.spec.Transform),
				isDim:         true,
				column:        e.resultColumn(col, "", ""),
			})
			continue
		}

		if pivotedMeasures[col.spec.ID] {
			expanded, err := e.pivotExpansions(col)
			if err != nil {
				return nil, err
			}
			outputs = append(outputs, expanded...)
			continue
		}

		expr, err := e.columnValueExpr(col, sqlExpr{})
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, outputColumn{
			id:            col.spec.ID,
			sourceID:      col.spec.ID,
			explicitLabel: col.spec.Label != "",
			expr:          applyTransform(expr, col.spec.Transform),
			column:        e.outputResultColumn(col, "", ""),
		})
	}

	disambiguateLabels(e.v, outputs)

	return outputs, nil
}

// columnValueExpr emits the aggregate-level value expression for a measure or
// computed column, applying the optional FILTER predicate to every underlying
// aggregate (pivot cells thread their bucket predicate through here).
func (e *emitter) columnValueExpr(col *validatedColumn, filter sqlExpr) (sqlExpr, error) {
	if col.spec.Kind == report.ColumnKindComputed {
		return e.computedExpr(col, filter)
	}
	return e.measureExpr(col, filter)
}

func (e *emitter) computedExpr(col *validatedColumn, filter sqlExpr) (sqlExpr, error) {
	comp := col.spec.Computed

	leftExpr, err := e.computedOperandExpr(comp.Left(), col.spec.ID, filter)
	if err != nil {
		return sqlExpr{}, err
	}
	rightExpr, err := e.computedOperandExpr(comp.Right(), col.spec.ID, filter)
	if err != nil {
		return sqlExpr{}, err
	}

	args := appendArgs(leftExpr.args, rightExpr.args...)
	if comp.Op == report.ComputedOpDivide {
		return sqlExpr{
			text: "(" + leftExpr.text + ")::numeric / NULLIF((" + rightExpr.text + ")::numeric, 0)",
			args: args,
		}, nil
	}

	return sqlExpr{
		text: "(" + leftExpr.text + ") " + computedOpSQL(comp.Op) + " (" + rightExpr.text + ")",
		args: args,
	}, nil
}

// computedOperandExpr resolves one side of a calculation. A constant is
// inlined as a decimal literal rather than bound as an argument: the whole
// expression is re-emitted per pivot cell and for the totals pass, and a
// placeholder would desync the argument list from the SQL it belongs to.
func (e *emitter) computedOperandExpr(
	operand report.ComputedOperand,
	ownerID string,
	filter sqlExpr,
) (sqlExpr, error) {
	if operand.IsLiteral() {
		return sqlExpr{text: decimal.NewFromFloat(operand.ValueOrZero()).String()}, nil
	}

	col := e.v.columnByID(operand.ColumnID)
	if col == nil {
		return sqlExpr{}, fmt.Errorf(
			"computed column %q references an unknown operand", ownerID,
		)
	}
	return e.measureExpr(col, filter)
}

func computedOpSQL(op report.ComputedOp) string {
	//nolint:exhaustive // divide is emitted separately with a NULLIF guard
	switch op {
	case report.ComputedOpAdd:
		return "+"
	case report.ComputedOpSubtract:
		return "-"
	default:
		return "*"
	}
}

func (e *emitter) dimensionExpr(
	ref *resolvedRef,
	grouping dimensionGrouping,
) (sqlExpr, error) {
	alias := e.plan.aliasFor(ref.pathKey)
	if alias == "" {
		return sqlExpr{}, fmt.Errorf("no alias planned for path %q", ref.pathKey)
	}
	col := alias + "." + ref.field.Column.Name

	if !grouping.band.IsEmpty() {
		return sqlExpr{text: bandExpr(col, grouping.band, ref.field.Type)}, nil
	}

	if grouping.bucket == report.DateBucketNone {
		return sqlExpr{text: col}, nil
	}

	return sqlExpr{
		text: "EXTRACT(EPOCH FROM date_trunc(?, to_timestamp(" + col + ") AT TIME ZONE ?))::bigint",
		args: []any{string(grouping.bucket), e.loc.String()},
	}, nil
}

func (e *emitter) measureExpr(col *validatedColumn, filter sqlExpr) (sqlExpr, error) {
	if !col.ref.toMany {
		scoped, err := e.scopedMeasureFilter(col, filter)
		if err != nil {
			return sqlExpr{}, err
		}
		return e.aggregateExpr(col.ref, col.spec.Agg, scoped)
	}

	lateral := e.plan.laterals[col.ref.pathKey]
	if lateral == nil {
		return sqlExpr{}, fmt.Errorf("no lateral planned for path %q", col.ref.pathKey)
	}

	var measure *lateralMeasure
	for _, m := range lateral.measures {
		if m.column.spec.ID == col.spec.ID {
			measure = m
			break
		}
	}
	if measure == nil {
		return sqlExpr{}, fmt.Errorf("measure %q missing from lateral plan", col.spec.ID)
	}

	inner := lateral.alias + "." + measure.innerName
	filterSQL := ""
	var filterArgs []any
	if filter.text != "" {
		filterSQL = " FILTER (WHERE " + filter.text + ")"
		filterArgs = filter.args
	}

	//nolint:exhaustive // count_distinct is rejected at validation for to-many paths
	switch col.spec.Agg {
	case reportcatalog.AggSum:
		return sqlExpr{text: "SUM(" + inner + ")" + filterSQL, args: filterArgs}, nil
	case reportcatalog.AggCount:
		return sqlExpr{
			text: "COALESCE(SUM(" + inner + ")" + filterSQL + ", 0)::bigint",
			args: filterArgs,
		}, nil
	case reportcatalog.AggMin:
		return sqlExpr{text: "MIN(" + inner + ")" + filterSQL, args: filterArgs}, nil
	case reportcatalog.AggMax:
		return sqlExpr{text: "MAX(" + inner + ")" + filterSQL, args: filterArgs}, nil
	case reportcatalog.AggAvg:
		sumExpr := "SUM(" + lateral.alias + "." + measure.innerName + "_sum)" + filterSQL
		cntExpr := "SUM(" + lateral.alias + "." + measure.innerName + "_cnt)" + filterSQL
		args := appendArgs(filterArgs, filterArgs...)
		if len(filterArgs) == 0 {
			args = nil
		}
		return sqlExpr{text: sumExpr + " / NULLIF(" + cntExpr + ", 0)", args: args}, nil
	default:
		return sqlExpr{}, fmt.Errorf(
			"aggregation %q is not supported across to-many relationships", col.spec.Agg,
		)
	}
}

// scopedMeasureFilter conjoins the caller's bucket predicate (a pivot cell)
// with the column's own filter so a pivoted, filtered measure narrows on both.
func (e *emitter) scopedMeasureFilter(
	col *validatedColumn,
	filter sqlExpr,
) (sqlExpr, error) {
	if col.spec.Filter.IsEmpty() {
		return filter, nil
	}
	own, err := e.filterGroupExpr(col.spec.Filter, false)
	if err != nil {
		return sqlExpr{}, err
	}
	return andExpr(filter, own), nil
}

func (e *emitter) aggregateExpr(
	ref *resolvedRef,
	agg reportcatalog.Aggregation,
	filter sqlExpr,
) (sqlExpr, error) {
	if ref.toMany {
		return e.havingLateralExpr(ref, agg)
	}

	alias := e.plan.aliasFor(ref.pathKey)
	if alias == "" {
		return sqlExpr{}, fmt.Errorf("no alias planned for path %q", ref.pathKey)
	}
	col := alias + "." + ref.field.Column.Name

	var text string
	switch agg {
	case reportcatalog.AggCount:
		text = "COUNT(" + col + ")"
	case reportcatalog.AggCountDistinct:
		text = "COUNT(DISTINCT " + col + ")"
	case reportcatalog.AggSum:
		text = "SUM(" + col + ")"
	case reportcatalog.AggAvg:
		text = "AVG(" + col + ")"
	case reportcatalog.AggMin:
		text = "MIN(" + col + ")"
	case reportcatalog.AggMax:
		text = "MAX(" + col + ")"
	default:
		return sqlExpr{}, fmt.Errorf("unknown aggregation %q", agg)
	}

	if filter.text != "" {
		return sqlExpr{
			text: text + " FILTER (WHERE " + filter.text + ")",
			args: filter.args,
		}, nil
	}
	return sqlExpr{text: text}, nil
}

func (e *emitter) havingLateralExpr(
	ref *resolvedRef,
	agg reportcatalog.Aggregation,
) (sqlExpr, error) {
	lateral := e.plan.laterals[ref.pathKey]
	if lateral == nil || len(lateral.measures) == 0 {
		return sqlExpr{}, fmt.Errorf(
			"measure filters across to-many paths require a matching measure column on %q",
			ref.ref.String(),
		)
	}

	// A measure column carrying its own filter aggregates a subset, so it
	// cannot stand in for the unfiltered measure a HAVING clause asks about.
	for _, m := range lateral.measures {
		if m.column.spec.Agg == agg && m.column.ref.ref.String() == ref.ref.String() &&
			m.column.spec.Filter.IsEmpty() {
			return e.measureExpr(m.column, sqlExpr{})
		}
	}

	return sqlExpr{}, fmt.Errorf(
		"measure filter on %q requires a matching unfiltered measure column with aggregation %q",
		ref.ref.String(), agg,
	)
}

func (e *emitter) pivotExpansions(col *validatedColumn) ([]outputColumn, error) {
	pivot := e.v.def.Pivot
	pivotExpr, err := e.dimensionExpr(e.v.pivotRef, dimensionGrouping{})
	if err != nil {
		return nil, err
	}

	outputs := make([]outputColumn, 0, len(pivot.Values)+1)

	for valueIndex, value := range pivot.Values {
		coerced, cErr := coerceScalar(e.v.pivotRef.field.Type, e.v.pivotRef.field, value)
		if cErr != nil {
			return nil, fmt.Errorf("pivot value %q: %w", value, cErr)
		}

		filter := sqlExpr{
			text: pivotExpr.text + " = ?",
			args: appendArgs(pivotExpr.args, coerced),
		}
		expr, mErr := e.columnValueExpr(col, filter)
		if mErr != nil {
			return nil, mErr
		}

		suffix := pivot.LabelFor(valueIndex, e.pivotValueLabel(value))
		outputs = append(outputs, outputColumn{
			id:            col.spec.ID + pivotIDSeparator + value,
			sourceID:      col.spec.ID,
			pivotSuffix:   suffix,
			explicitLabel: col.spec.Label != "",
			expr:          applyTransform(expr, col.spec.Transform),
			column:        e.outputResultColumn(col, value, defaultLabel(col)+" ("+suffix+")"),
		})
	}

	if pivot.IncludeOther {
		values := make([]any, 0, len(pivot.Values))
		for _, value := range pivot.Values {
			coerced, cErr := coerceScalar(e.v.pivotRef.field.Type, e.v.pivotRef.field, value)
			if cErr != nil {
				return nil, cErr
			}
			values = append(values, coerced)
		}

		filter := sqlExpr{
			text: "(" + pivotExpr.text + " IS NULL OR " + pivotExpr.text + " NOT IN (?))",
			args: appendArgs(appendArgs(pivotExpr.args, pivotExpr.args...), bun.List(values)),
		}
		expr, mErr := e.columnValueExpr(col, filter)
		if mErr != nil {
			return nil, mErr
		}

		outputs = append(outputs, outputColumn{
			id:            col.spec.ID + pivotIDSeparator + pivotOtherValue,
			sourceID:      col.spec.ID,
			pivotSuffix:   "Other",
			explicitLabel: col.spec.Label != "",
			expr:          applyTransform(expr, col.spec.Transform),
			column:        e.outputResultColumn(col, pivotOtherValue, defaultLabel(col)+" (Other)"),
		})
	}

	return outputs, nil
}

func (e *emitter) pivotValueLabel(value string) string {
	for i := range e.v.pivotRef.field.EnumValues {
		if e.v.pivotRef.field.EnumValues[i].Value == value {
			return e.v.pivotRef.field.EnumValues[i].Label
		}
	}
	if e.v.pivotRef.field.Type == reportcatalog.FieldBool {
		switch value {
		case "true":
			return "Yes"
		case "false":
			return "No"
		}
	}
	return value
}

func (e *emitter) outputResultColumn(
	col *validatedColumn,
	pivotValue string,
	labelOverride string,
) services.ReportResultColumn {
	if col.spec.Kind == report.ColumnKindComputed {
		return e.computedResultColumn(col, pivotValue, labelOverride)
	}
	return e.resultColumn(col, pivotValue, labelOverride)
}

func measureResultType(
	agg reportcatalog.Aggregation,
	fieldType reportcatalog.FieldType,
) reportcatalog.FieldType {
	//nolint:exhaustive // min/max keep the underlying field type
	switch agg {
	case reportcatalog.AggCount, reportcatalog.AggCountDistinct:
		return reportcatalog.FieldInt
	case reportcatalog.AggAvg:
		return reportcatalog.FieldDecimal
	case reportcatalog.AggSum:
		if fieldType == reportcatalog.FieldInt {
			return reportcatalog.FieldInt
		}
		return reportcatalog.FieldDecimal
	default:
		return fieldType
	}
}

func (e *emitter) fieldSensitivity(col *validatedColumn) permission.FieldSensitivity {
	return e.c.permissionRegistry.GetFieldSensitivity(
		col.ref.entity.Resource.String(), col.ref.field.Key,
	)
}

func (e *emitter) computedResultColumn(
	col *validatedColumn,
	pivotValue string,
	labelOverride string,
) services.ReportResultColumn {
	// A constant carries no field, so it contributes no sensitivity; the
	// calculation is only ever as sensitive as the columns it reads.
	sensitivity := permission.SensitivityPublic
	for _, operand := range col.spec.Computed.Operands() {
		source := e.v.columnByID(operand.ColumnID)
		if operand.IsLiteral() || source == nil {
			continue
		}
		if operandSens := e.fieldSensitivity(source); operandSens.Level() > sensitivity.Level() {
			sensitivity = operandSens
		}
	}

	label := labelOverride
	if label == "" {
		label = defaultLabel(col)
	}

	resultType := columnValueType(e.v, col)

	return services.ReportResultColumn{
		ID:          outputColumnID(col, pivotValue),
		Label:       label,
		Type:        resultType,
		Format:      col.spec.Computed.Format,
		Display:     resolveDisplay(col, resultType, col.spec.Computed.Format),
		Sensitivity: sensitivity,
	}
}

func (e *emitter) resultColumn(
	col *validatedColumn,
	pivotValue string,
	labelOverride string,
) services.ReportResultColumn {
	label := labelOverride
	if label == "" {
		label = defaultLabel(col)
	}

	format := col.ref.field.Format
	if col.spec.Kind == report.ColumnKindMeasure &&
		(col.spec.Agg == reportcatalog.AggCount ||
			col.spec.Agg == reportcatalog.AggCountDistinct) {
		format = reportcatalog.FormatCount
	}

	resultType := columnValueType(e.v, col)

	return services.ReportResultColumn{
		ID:          outputColumnID(col, pivotValue),
		Label:       label,
		Type:        resultType,
		Format:      format,
		Display:     resolveDisplay(col, resultType, format),
		Sensitivity: e.fieldSensitivity(col),
	}
}

func outputColumnID(col *validatedColumn, pivotValue string) string {
	if pivotValue == "" {
		return col.spec.ID
	}
	return col.spec.ID + pivotIDSeparator + pivotValue
}

func resolveDisplay(
	col *validatedColumn,
	resultType reportcatalog.FieldType,
	hint reportcatalog.FormatHint,
) reportfmt.Resolved {
	return reportfmt.Resolve(reportfmt.ResolveInput{
		Type:      resultType,
		Hint:      hint,
		DateStyle: bucketDateStyle(col.spec.Bucket),
		Band:      col.spec.Band,
		Spec:      col.spec.Display,
	})
}

func bucketDateStyle(bucket report.DateBucket) reportfmt.DateStyle {
	switch bucket {
	case report.DateBucketDay, report.DateBucketWeek:
		return reportfmt.DateStyleDate
	case report.DateBucketMonth:
		return reportfmt.DateStyleMonthYear
	case report.DateBucketQuarter:
		return reportfmt.DateStyleQuarter
	case report.DateBucketYear:
		return reportfmt.DateStyleYear
	case report.DateBucketNone:
		return reportfmt.DateStyleAuto
	default:
		return reportfmt.DateStyleAuto
	}
}
