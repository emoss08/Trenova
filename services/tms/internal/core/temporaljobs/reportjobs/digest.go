package reportjobs

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/reportfmt"
)

// Inline styles throughout: email clients strip <style> blocks and support no
// classes worth relying on, so every rule rides on the element.
const (
	digestCellStyle   = "padding:6px 10px;border-bottom:1px solid #ececec;white-space:nowrap;"
	digestHeaderStyle = "padding:6px 10px;border-bottom:2px solid #ddd;color:#666;" +
		"font-weight:600;text-align:left;white-space:nowrap;"
	digestTotalsStyle = "padding:6px 10px;border-top:2px solid #ddd;font-weight:600;" +
		"white-space:nowrap;"
)

// digestToneColors mirrors the grid and the XLSX palette. A tone is a meaning,
// not a colour, so each surface picks its own — these are chosen to stay legible
// on the white background email clients force.
var digestToneColors = map[reportfmt.Tone]string{
	// ToneNone is listed for completeness; it is filtered before the lookup.
	reportfmt.ToneNone:     "",
	reportfmt.TonePositive: "#15803d",
	reportfmt.ToneWarning:  "#b45309",
	reportfmt.ToneNegative: "#b91c1c",
	reportfmt.ToneNeutral:  "#1a1a1a",
}

// renderDigestHTML draws the sampled rows as a table for the email body. Every
// cell goes through the column's own display contract, so a currency, a
// duration, a banded range, and a conditional-format tone all read exactly as
// they do in the grid and the exports.
func renderDigestHTML(digest *services.ReportDigest, loc *time.Location) string {
	if digest.IsEmpty() || len(digest.Rows) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(
		`<table style="border-collapse:collapse;font-size:12px;margin:0 0 16px;width:100%;">`,
	)

	sb.WriteString(`<thead><tr>`)
	for i := range digest.Columns {
		column := &digest.Columns[i]
		fmt.Fprintf(&sb, `<th style="%s%s">%s</th>`,
			digestHeaderStyle, digestAlign(column), html.EscapeString(column.Label))
	}
	sb.WriteString(`</tr></thead><tbody>`)

	for _, row := range digest.Rows {
		sb.WriteString(`<tr>`)
		for i := range digest.Columns {
			column := &digest.Columns[i]
			var value any
			if i < len(row) {
				value = row[i]
			}
			fmt.Fprintf(&sb, `<td style="%s%s%s">%s</td>`,
				digestCellStyle,
				digestAlign(column),
				digestToneStyle(column, value),
				html.EscapeString(column.Display.String(value, loc)),
			)
		}
		sb.WriteString(`</tr>`)
	}
	sb.WriteString(`</tbody>`)

	if len(digest.Totals) > 0 {
		sb.WriteString(`<tfoot><tr>`)
		for i := range digest.Columns {
			column := &digest.Columns[i]
			var value any
			if i < len(digest.Totals) {
				value = digest.Totals[i]
			}
			// A dimension has no total; the first one carries the label instead.
			text := column.Display.String(value, loc)
			if value == nil && i == 0 {
				text = "Total"
			}
			fmt.Fprintf(&sb, `<td style="%s%s">%s</td>`,
				digestTotalsStyle, digestAlign(column), html.EscapeString(text))
		}
		sb.WriteString(`</tr></tfoot>`)
	}

	sb.WriteString(`</table>`)
	return sb.String()
}

// renderDigestText is the plain-text counterpart. It stays deliberately simple —
// a fixed-width table in an email that may be read on a phone is worse than
// labelled values.
func renderDigestText(digest *services.ReportDigest, loc *time.Location) string {
	if digest.IsEmpty() || len(digest.Rows) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, row := range digest.Rows {
		cells := make([]string, 0, len(digest.Columns))
		for i := range digest.Columns {
			var value any
			if i < len(row) {
				value = row[i]
			}
			cells = append(cells, digest.Columns[i].Label+": "+
				digest.Columns[i].Display.String(value, loc))
		}
		sb.WriteString(strings.Join(cells, " | "))
		sb.WriteString("\n")
	}
	return sb.String()
}

// digestNote tells the reader the table is a sample rather than the whole
// answer, so nobody reconciles against 20 rows of a 4,000-row report.
func digestNote(digest *services.ReportDigest, rowCount int64) string {
	if digest.IsEmpty() || len(digest.Rows) == 0 {
		return ""
	}
	if !digest.Truncated {
		return ""
	}
	return fmt.Sprintf(
		"Showing the first %d of %d rows — the full result is in the report file.",
		len(digest.Rows), rowCount,
	)
}

func digestAlign(column *services.ReportResultColumn) string {
	if column.Display.IsNumeric() {
		return "text-align:right;"
	}
	return "text-align:left;"
}

func digestToneStyle(column *services.ReportResultColumn, value any) string {
	tone := column.Display.ToneFor(value)
	if tone == reportfmt.ToneNone {
		return ""
	}
	colour, ok := digestToneColors[tone]
	if !ok {
		return ""
	}
	return "color:" + colour + ";font-weight:600;"
}
