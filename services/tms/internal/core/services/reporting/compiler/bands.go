package compiler

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"github.com/shopspring/decimal"
)

// dimensionGrouping carries the two ways a dimension can be collapsed before
// grouping: a calendar bucket for instants and a numeric band for magnitudes.
// They are mutually exclusive, which validation enforces.
type dimensionGrouping struct {
	bucket report.DateBucket
	band   *reportfmt.Band
}

func columnGrouping(spec *report.ColumnSpec) dimensionGrouping {
	return dimensionGrouping{bucket: spec.Bucket, band: spec.Band}
}

// bandExpr collapses a numeric column onto the lower bound of the range it
// falls in. Emitting the bound rather than a text label is what keeps GROUP BY,
// ORDER BY and drill-through working on exact numbers — the range a reader sees
// is produced by the column's display contract from the same boundaries.
//
// Boundaries are inlined as decimal literals rather than bound parameters: the
// expression is repeated for every band, and duplicating placeholders would put
// the argument list out of step with the SQL. They are validated finite numbers
// rendered by decimal, so there is nothing to inject.
func bandExpr(column string, band *reportfmt.Band, fieldType reportcatalog.FieldType) string {
	if !band.IsUsable() {
		return column
	}

	numeric := "(" + column + ")::numeric"

	var text string
	if band.IsFixedWidth() {
		width := decimal.NewFromFloat(band.Width).String()
		text = "FLOOR(" + numeric + " / " + width + ") * " + width
	} else {
		text = bandCase(numeric, band.Edges)
	}

	// A whole-number column stays a whole number, so the scanned value keeps the
	// type the rest of the pipeline resolved for it.
	if fieldType == reportcatalog.FieldInt {
		return "(" + text + ")::bigint"
	}
	return text
}

// bandCase walks the boundaries from the top down so the first match is the
// highest boundary the value clears. Anything below the lowest boundary falls
// into the first range, which is what an author means by "0–2h" when the data
// can dip below zero.
func bandCase(numeric string, edges []float64) string {
	var sb strings.Builder

	sb.WriteString("CASE WHEN ")
	sb.WriteString(numeric)
	sb.WriteString(" IS NULL THEN NULL")

	for i := len(edges) - 1; i >= 1; i-- {
		edge := decimal.NewFromFloat(edges[i]).String()
		sb.WriteString(" WHEN ")
		sb.WriteString(numeric)
		sb.WriteString(" >= ")
		sb.WriteString(edge)
		sb.WriteString(" THEN ")
		sb.WriteString(edge)
	}

	sb.WriteString(" ELSE ")
	sb.WriteString(decimal.NewFromFloat(edges[0]).String())
	sb.WriteString(" END")

	return sb.String()
}

func bandableType(fieldType reportcatalog.FieldType) bool {
	return fieldType == reportcatalog.FieldInt || fieldType == reportcatalog.FieldDecimal
}
