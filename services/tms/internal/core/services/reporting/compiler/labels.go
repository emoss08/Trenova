package compiler

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/i18n"
)

const labelSlot = "\uE000"

type labeler struct {
	locale i18n.Locale
}

func (l labeler) text(message string, args ...any) string {
	return i18n.Translate(l.locale, message, args...)
}

// defaultLabel derives a self-describing header for a column that carries no
// author-supplied label. Related fields reached through different edges — two
// `code` fields, say — are qualified by the edge that reached them so the
// header identifies which one it is.
func (l labeler) defaultLabel(col *validatedColumn) string {
	if col.spec.Label != "" {
		return col.spec.Label
	}
	if col.ref == nil {
		return i18n.Translate(l.locale, "Calculation")
	}

	base := l.qualifiedFieldLabel(col.ref, 1)
	if col.spec.Kind == report.ColumnKindMeasure {
		return l.aggregatedLabel(col.spec.Agg, col.ref, base)
	}
	if col.spec.Kind == report.ColumnKindDimension {
		return l.groupedLabel(col.spec, base)
	}
	return base
}

// qualifiedFieldLabel joins the last `depth` edge labels of the path with the
// field label, dropping segments that already repeat one another.
func (l labeler) qualifiedFieldLabel(ref *resolvedRef, depth int) string {
	steps := ref.path.Steps
	if len(steps) == 0 || depth <= 0 {
		return l.text(ref.field.Label)
	}

	start := max(len(steps)-depth, 0)

	segments := make([]string, 0, len(steps)-start+1)
	for i := start; i < len(steps); i++ {
		segments = append(segments, l.text(edgeLabel(steps[i].Edge)))
	}
	segments = append(segments, l.text(ref.field.Label))

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

func (l labeler) aggregatedLabel(
	agg reportcatalog.Aggregation,
	ref *resolvedRef,
	base string,
) string {
	chronological := ref.field.Type == reportcatalog.FieldEpoch

	switch agg {
	case reportcatalog.AggSum:
		return l.unlessRepeated(
			base,
			i18n.Translate(l.locale, "Total {0}", base),
			i18n.Translate(l.locale, "Total {0}", labelSlot),
		)
	case reportcatalog.AggAvg:
		return l.unlessRepeated(
			base,
			i18n.Translate(l.locale, "Average {0}", base),
			i18n.Translate(l.locale, "Average {0}", labelSlot),
		)
	case reportcatalog.AggMin:
		if chronological {
			return l.unlessRepeated(
				base,
				i18n.Translate(l.locale, "Earliest {0}", base),
				i18n.Translate(l.locale, "Earliest {0}", labelSlot),
			)
		}
		return l.unlessRepeated(
			base,
			i18n.Translate(l.locale, "Minimum {0}", base),
			i18n.Translate(l.locale, "Minimum {0}", labelSlot),
		)
	case reportcatalog.AggMax:
		if chronological {
			return l.unlessRepeated(
				base,
				i18n.Translate(l.locale, "Latest {0}", base),
				i18n.Translate(l.locale, "Latest {0}", labelSlot),
			)
		}
		return l.unlessRepeated(
			base,
			i18n.Translate(l.locale, "Maximum {0}", base),
			i18n.Translate(l.locale, "Maximum {0}", labelSlot),
		)
	case reportcatalog.AggCount:
		return l.countLabel(ref, base, false)
	case reportcatalog.AggCountDistinct:
		return l.countLabel(ref, base, true)
	default:
		return base
	}
}

func (l labeler) unlessRepeated(base, rendered, shape string) string {
	before, after, found := strings.Cut(shape, labelSlot)
	if !found {
		return rendered
	}
	before = strings.ToLower(strings.TrimSpace(before))
	after = strings.ToLower(strings.TrimSpace(after))
	words := before
	if words == "" {
		words = after
	} else if after != "" {
		return rendered
	}
	if words != "" && containsWords(strings.ToLower(base), words) {
		return base
	}
	return rendered
}

func containsWords(label, words string) bool {
	return label == words ||
		strings.HasPrefix(label, words+" ") ||
		strings.HasSuffix(label, " "+words) ||
		strings.Contains(label, " "+words+" ")
}

func (l labeler) countLabel(ref *resolvedRef, base string, distinct bool) string {
	subject := base
	if isIdentityField(ref) {
		subject = l.text(ref.entity.PluralLabel)
	}
	if distinct {
		return i18n.Translate(l.locale, "Distinct {0}", subject)
	}
	return i18n.Translate(l.locale, "{0} Count", subject)
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
func (l labeler) groupedLabel(spec *report.ColumnSpec, base string) string {
	if !spec.Band.IsEmpty() {
		return i18n.Translate(l.locale, "{0} (Range)", base)
	}
	return l.bucketedLabel(spec.Bucket, base)
}

func (l labeler) bucketedLabel(bucket report.DateBucket, base string) string {
	switch bucket {
	case report.DateBucketDay:
		return i18n.Translate(l.locale, "{0} (Day)", base)
	case report.DateBucketWeek:
		return i18n.Translate(l.locale, "{0} (Week)", base)
	case report.DateBucketMonth:
		return i18n.Translate(l.locale, "{0} (Month)", base)
	case report.DateBucketQuarter:
		return i18n.Translate(l.locale, "{0} (Quarter)", base)
	case report.DateBucketYear:
		return i18n.Translate(l.locale, "{0} (Year)", base)
	case report.DateBucketNone:
		return base
	default:
		return base
	}
}

func (l labeler) pivotLabel(base, value string) string {
	return i18n.Translate(l.locale, "{0} ({1})", base, value)
}

// disambiguateLabels guarantees every exported header is unique: colliding
// columns are re-qualified with more of their path, then fall back to the
// column id so a spreadsheet never carries two identically named columns.
func (l labeler) disambiguateLabels(v *validatedDef, outputs []outputColumn) {
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
			candidate := l.requalify(col, out, depth)
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
		outputs[i].column.Label = l.pivotLabel(label, outputs[i].id)
		seen[outputs[i].column.Label] = true
	}
}

func (l labeler) requalify(col *validatedColumn, out *outputColumn, depth int) string {
	base := l.qualifiedFieldLabel(col.ref, depth)
	switch col.spec.Kind {
	case report.ColumnKindMeasure:
		base = l.aggregatedLabel(col.spec.Agg, col.ref, base)
	case report.ColumnKindDimension:
		base = l.groupedLabel(col.spec, base)
	case report.ColumnKindComputed:
	}
	if out.pivotSuffix != "" {
		return l.pivotLabel(base, out.pivotSuffix)
	}
	return base
}
