package compiler

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

// defaultLabel derives a self-describing header for a column that carries no
// author-supplied label. Related fields reached through different edges — two
// `code` fields, say — are qualified by the edge that reached them so the
// header identifies which one it is.
func defaultLabel(col *validatedColumn) string {
	if col.spec.Label != "" {
		return col.spec.Label
	}
	if col.ref == nil {
		return "Calculation"
	}

	base := qualifiedFieldLabel(col.ref, 1)
	if col.spec.Kind == report.ColumnKindMeasure {
		return aggregatedLabel(col.spec.Agg, col.ref, base)
	}
	if col.spec.Kind == report.ColumnKindDimension {
		return groupedLabel(col.spec, base)
	}
	return base
}

// qualifiedFieldLabel joins the last `depth` edge labels of the path with the
// field label, dropping segments that already repeat one another.
func qualifiedFieldLabel(ref *resolvedRef, depth int) string {
	steps := ref.path.Steps
	if len(steps) == 0 || depth <= 0 {
		return ref.field.Label
	}

	start := len(steps) - depth
	if start < 0 {
		start = 0
	}

	segments := make([]string, 0, len(steps)-start+1)
	for i := start; i < len(steps); i++ {
		segments = append(segments, edgeLabel(steps[i].Edge))
	}
	segments = append(segments, ref.field.Label)

	return joinLabelSegments(segments)
}

func edgeLabel(edge *reportcatalog.Edge) string {
	if edge.Label != "" {
		return edge.Label
	}
	return edge.Name
}

func joinLabelSegments(segments []string) string {
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		if len(parts) > 0 && absorbs(parts[len(parts)-1], segment) {
			continue
		}
		if len(parts) > 0 && absorbs(segment, parts[len(parts)-1]) {
			parts[len(parts)-1] = segment
			continue
		}
		parts = append(parts, segment)
	}
	return strings.Join(parts, " ")
}

// absorbs reports whether `outer` already conveys `inner` — "Fleet Code"
// absorbs "Code", so the pair renders as "Fleet Code" rather than
// "Fleet Code Code".
func absorbs(outer, inner string) bool {
	if outer == "" || inner == "" {
		return false
	}
	if strings.EqualFold(outer, inner) {
		return true
	}
	lowerOuter := strings.ToLower(outer)
	lowerInner := strings.ToLower(inner)
	return strings.HasSuffix(lowerOuter, " "+lowerInner) ||
		strings.HasPrefix(lowerOuter, lowerInner+" ")
}

func aggregatedLabel(
	agg reportcatalog.Aggregation,
	ref *resolvedRef,
	base string,
) string {
	chronological := ref.field.Type == reportcatalog.FieldEpoch

	switch agg {
	case reportcatalog.AggSum:
		return prefixed("Total", base)
	case reportcatalog.AggAvg:
		return prefixed("Average", base)
	case reportcatalog.AggMin:
		if chronological {
			return prefixed("Earliest", base)
		}
		return prefixed("Minimum", base)
	case reportcatalog.AggMax:
		if chronological {
			return prefixed("Latest", base)
		}
		return prefixed("Maximum", base)
	case reportcatalog.AggCount:
		return countLabel(ref, base, false)
	case reportcatalog.AggCountDistinct:
		return countLabel(ref, base, true)
	default:
		return base
	}
}

func prefixed(prefix, base string) string {
	if strings.HasPrefix(strings.ToLower(base), strings.ToLower(prefix)) {
		return base
	}
	return prefix + " " + base
}

func countLabel(ref *resolvedRef, base string, distinct bool) string {
	subject := base
	if isIdentityField(ref) {
		subject = ref.entity.PluralLabel
	}
	if distinct {
		return "Distinct " + subject
	}
	return subject + " Count"
}

func isIdentityField(ref *resolvedRef) bool {
	for _, key := range ref.entity.GrainKey() {
		if key == ref.field.Column.Name {
			return true
		}
	}
	return false
}

// groupedLabel names the collapsing a dimension carries, so a header says what
// its rows stand for rather than naming the raw field they came from.
func groupedLabel(spec *report.ColumnSpec, base string) string {
	if !spec.Band.IsEmpty() {
		return base + " (Range)"
	}
	return bucketedLabel(spec.Bucket, base)
}

func bucketedLabel(bucket report.DateBucket, base string) string {
	switch bucket {
	case report.DateBucketDay:
		return base + " (Day)"
	case report.DateBucketWeek:
		return base + " (Week)"
	case report.DateBucketMonth:
		return base + " (Month)"
	case report.DateBucketQuarter:
		return base + " (Quarter)"
	case report.DateBucketYear:
		return base + " (Year)"
	case report.DateBucketNone:
		return base
	default:
		return base
	}
}

// disambiguateLabels guarantees every exported header is unique: colliding
// columns are re-qualified with more of their path, then fall back to the
// column id so a spreadsheet never carries two identically named columns.
func disambiguateLabels(v *validatedDef, outputs []outputColumn) {
	counts := make(map[string]int, len(outputs))
	for i := range outputs {
		counts[outputs[i].column.Label]++
	}

	for i := range outputs {
		out := &outputs[i]
		if counts[out.column.Label] < 2 || out.explicitLabel {
			continue
		}
		col := v.columnByID(out.sourceID)
		if col == nil || col.ref == nil {
			continue
		}
		for depth := 2; depth <= len(col.ref.path.Steps); depth++ {
			candidate := requalify(col, out, depth)
			if counts[candidate] == 0 {
				counts[out.column.Label]--
				out.column.Label = candidate
				counts[candidate]++
				break
			}
		}
	}

	seen := make(map[string]bool, len(outputs))
	for i := range outputs {
		label := outputs[i].column.Label
		if !seen[label] {
			seen[label] = true
			continue
		}
		outputs[i].column.Label = label + " (" + outputs[i].id + ")"
		seen[outputs[i].column.Label] = true
	}
}

func requalify(col *validatedColumn, out *outputColumn, depth int) string {
	base := qualifiedFieldLabel(col.ref, depth)
	switch col.spec.Kind {
	case report.ColumnKindMeasure:
		base = aggregatedLabel(col.spec.Agg, col.ref, base)
	case report.ColumnKindDimension:
		base = groupedLabel(col.spec, base)
	case report.ColumnKindComputed:
	}
	if out.pivotSuffix != "" {
		return base + " (" + out.pivotSuffix + ")"
	}
	return base
}
