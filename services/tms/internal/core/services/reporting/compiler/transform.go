package compiler

import (
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/shopspring/decimal"
)

// preTransformType is the value type a column carries before its transform is
// applied: the field type for dimensions, the aggregate's result type for
// measures, and the arithmetic result type for calculations.
func preTransformType(v *validatedDef, col *validatedColumn) reportcatalog.FieldType {
	switch col.spec.Kind {
	case report.ColumnKindComputed:
		return computedValueType(v, col)
	case report.ColumnKindMeasure:
		return measureResultType(col.spec.Agg, col.ref.field.Type)
	case report.ColumnKindDimension:
		if col.spec.Bucket != report.DateBucketNone {
			return reportcatalog.FieldEpoch
		}
		return col.ref.field.Type
	default:
		return col.ref.field.Type
	}
}

func computedValueType(v *validatedDef, col *validatedColumn) reportcatalog.FieldType {
	comp := col.spec.Computed
	if comp == nil || comp.Op == report.ComputedOpDivide {
		return reportcatalog.FieldDecimal
	}

	for _, operand := range comp.Operands() {
		if computedOperandType(v, operand) != reportcatalog.FieldInt {
			return reportcatalog.FieldDecimal
		}
	}
	return reportcatalog.FieldInt
}

// computedOperandType mirrors what the emitter writes: a whole-numbered
// constant is inlined as an integer literal and keeps the expression in
// bigint, so the decoded type has to agree or the executor reads the wrong
// column shape.
func computedOperandType(
	v *validatedDef,
	operand report.ComputedOperand,
) reportcatalog.FieldType {
	if operand.IsLiteral() {
		if decimal.NewFromFloat(operand.ValueOrZero()).IsInteger() {
			return reportcatalog.FieldInt
		}
		return reportcatalog.FieldDecimal
	}

	source := v.columnByID(operand.ColumnID)
	if source == nil {
		return reportcatalog.FieldDecimal
	}
	return measureResultType(source.spec.Agg, source.ref.field.Type)
}

func columnValueType(v *validatedDef, col *validatedColumn) reportcatalog.FieldType {
	base := preTransformType(v, col)
	if col.spec.Transform == nil {
		return base
	}
	if col.spec.Transform.Op.ProducesDecimal() {
		return reportcatalog.FieldDecimal
	}
	return base
}

// applyTransform wraps an already-aggregated column expression. Dimensions are
// wrapped before GROUP BY is derived from the same expression, so grouping
// follows the transformed value.
func applyTransform(expr sqlExpr, transform *report.TransformSpec) sqlExpr {
	if transform == nil || transform.Op == report.TransformNone {
		return expr
	}

	inner := "(" + expr.text + ")"

	switch transform.Op {
	case report.TransformRound:
		return sqlExpr{
			text: "ROUND(" + inner + "::numeric, " + precisionSQL(transform) + ")",
			args: expr.args,
		}
	case report.TransformTruncate:
		return sqlExpr{
			text: "TRUNC(" + inner + "::numeric, " + precisionSQL(transform) + ")",
			args: expr.args,
		}
	case report.TransformCeil:
		return sqlExpr{text: roundingSQL("CEIL", inner, transform), args: expr.args}
	case report.TransformFloor:
		return sqlExpr{text: roundingSQL("FLOOR", inner, transform), args: expr.args}
	case report.TransformAbs:
		return sqlExpr{text: "ABS(" + inner + ")", args: expr.args}
	case report.TransformScale:
		factor := decimal.NewFromFloat(transform.FactorOrOne())
		return sqlExpr{
			text: inner + "::numeric * " + factor.String(),
			args: expr.args,
		}
	case report.TransformUpper:
		return sqlExpr{text: "UPPER(" + inner + ")", args: expr.args}
	case report.TransformLower:
		return sqlExpr{text: "LOWER(" + inner + ")", args: expr.args}
	case report.TransformTitle:
		return sqlExpr{text: "INITCAP(" + inner + ")", args: expr.args}
	case report.TransformTrim:
		return sqlExpr{text: "BTRIM(" + inner + ")", args: expr.args}
	case report.TransformNone:
		return expr
	default:
		return expr
	}
}

// roundingSQL emits directional rounding at a decimal precision: Postgres
// CEIL/FLOOR only round to whole units, so the value is shifted, rounded, and
// shifted back.
func roundingSQL(fn, inner string, transform *report.TransformSpec) string {
	precision := transform.PrecisionOrZero()
	if precision <= 0 {
		return fn + "(" + inner + "::numeric)"
	}
	scale := "1" + zeros(precision)
	return fn + "(" + inner + "::numeric * " + scale + ") / " + scale
}

func precisionSQL(transform *report.TransformSpec) string {
	return strconv.Itoa(transform.PrecisionOrZero())
}

func zeros(count int) string {
	buf := make([]byte, count)
	for i := range buf {
		buf[i] = '0'
	}
	return string(buf)
}
