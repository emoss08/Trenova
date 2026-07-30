package reportjobs

import (
	"fmt"
	"html/template"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/fileutils"
)

// deliveryEmailContent groups what the message needs; the parameter list had
// outgrown a readable signature.
type deliveryEmailContent struct {
	run            *report.ReportRun
	schedule       *report.ReportSchedule
	title          string
	digest         *services.ReportDigest
	attached       bool
	attachTooLarge bool
}

// deliveryEmailContext maps a finished run onto the variables the report
// delivery template declares.
//
// The layout, the wording, and the digest table used to be two strings.Builders
// in delivery.go — one for text, one for inline-styled HTML — which meant an
// organization could not change a word of a message their own people receive on a
// schedule. The numbers still come from the column display contracts, so the
// email, the grid, and the exports still agree cell for cell.
func (a *Activities) deliveryEmailContext(
	content *deliveryEmailContent,
) documenttemplate.ReportDeliveryEmailContext {
	run, schedule := content.run, content.schedule
	loc := timezoneOrUTC(schedule.Timezone)

	out := documenttemplate.ReportDeliveryEmailContext{
		Title:        content.title,
		Format:       string(run.Format),
		CompletedAt:  formatInTimezone(run.CompletedAt, schedule.Timezone),
		RowCount:     run.RowCount,
		FileSize:     fileutils.HumanizeBytes(run.ByteSize),
		Truncated:    run.Truncated,
		Attached:     content.attached,
		AttachedNote: a.attachmentNote(content),
		Digest:       digestContext(content.digest, loc, run.RowCount),
	}

	if schedule.ReportDefinition != nil {
		out.Description = schedule.ReportDefinition.Description
	}
	if schedule.RunAs != nil {
		out.RequestedBy = schedule.RunAs.Name
	}
	if run.ArtifactExpiresAt > 0 {
		out.ExpiresAt = formatInTimezone(run.ArtifactExpiresAt, schedule.Timezone)
	}

	// The link is built from configured deployment settings, never from anything
	// a user typed, which is why the context types it as a template.URL. An
	// unconfigured base URL leaves it empty and the template drops the button
	// rather than linking nowhere.
	if base := a.cfg.GetDeliveryLinkBaseURL(); base != "" {
		//nolint:gosec // G203: configured base URL, not user input.
		out.DownloadURL = template.URL(base + "/reports/runs")
	}

	return out
}

// attachmentNote says where the file is, which is the one sentence a recipient
// actually acts on.
//
// A reader who is told to look for an attachment that was dropped for size will
// go looking for it, so the size limit is named rather than implied.
func (a *Activities) attachmentNote(content *deliveryEmailContent) string {
	switch {
	case content.attached:
		return "The report file is attached to this email."
	case content.attachTooLarge:
		return fmt.Sprintf(
			"The report file exceeds the %s attachment limit, so it was not attached.",
			fileutils.HumanizeBytes(a.cfg.GetEmailMaxAttachmentBytes()),
		)
	default:
		return "The report file is available from the run history in Trenova."
	}
}

// digestContext renders the sampled rows into the template's shape.
//
// Every cell goes through its column's own display contract, so a currency, a
// duration, a banded range, and a conditional-format tone read exactly as they do
// in the grid and the exports. The tone travels as a name rather than a colour:
// what a tone *means* is the report definition's decision, and how it looks
// belongs to whoever is editing the template.
func digestContext(
	digest *services.ReportDigest,
	loc *time.Location,
	rowCount int64,
) documenttemplate.ReportDigest {
	if digest.IsEmpty() || len(digest.Rows) == 0 {
		return documenttemplate.ReportDigest{}
	}

	out := documenttemplate.ReportDigest{
		Headers: make([]string, 0, len(digest.Columns)),
		Rows:    make([]documenttemplate.ReportRow, 0, len(digest.Rows)),
		Note:    digestNote(digest, rowCount),
	}

	for i := range digest.Columns {
		out.Headers = append(out.Headers, digest.Columns[i].Label)
	}

	for _, row := range digest.Rows {
		out.Rows = append(out.Rows, documenttemplate.ReportRow{
			Cells: digestCells(digest.Columns, row, loc),
		})
	}

	if len(digest.Totals) > 0 {
		out.Totals = digestTotals(digest, loc)
	}

	return out
}

func digestCells(
	columns []services.ReportResultColumn,
	row services.ReportRow,
	loc *time.Location,
) []documenttemplate.ReportCell {
	cells := make([]documenttemplate.ReportCell, 0, len(columns))
	for i := range columns {
		var value any
		if i < len(row) {
			value = row[i]
		}

		cells = append(cells, documenttemplate.ReportCell{
			Text: columns[i].Display.String(value, loc),
			Tone: string(columns[i].Display.ToneFor(value)),
		})
	}

	return cells
}

// digestTotals labels the leading dimension, which has no total of its own. A
// blank first cell reads as a missing number rather than as a row label.
func digestTotals(digest *services.ReportDigest, loc *time.Location) []string {
	totals := make([]string, 0, len(digest.Columns))
	for i := range digest.Columns {
		var value any
		if i < len(digest.Totals) {
			value = digest.Totals[i]
		}

		text := digest.Columns[i].Display.String(value, loc)
		if value == nil && i == 0 {
			text = digestTotalsLabel
		}
		totals = append(totals, text)
	}

	return totals
}

const digestTotalsLabel = "Total"
