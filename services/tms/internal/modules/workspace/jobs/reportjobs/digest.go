package reportjobs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/trenova/internal/infrastructure/reporting/render"
	"github.com/emoss08/trenova/internal/modules/documents/domain/reporttemplate"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/reportvalue"
	"github.com/emoss08/trenova/shared/i18n"
)

func readDigest(digest *reportports.ReportDigest) (*export.Envelope, error) {
	if digest.IsEmpty() {
		return nil, nil //nolint:nilnil // an empty digest has no envelope to read
	}
	return export.ReadJSON(bytes.NewReader(digest.Envelope))
}

func digestNote(
	ctx context.Context,
	digest *reportports.ReportDigest,
	shown int,
	rowCount int64,
) string {
	if digest.IsEmpty() || shown == 0 || !digest.Truncated {
		return ""
	}
	return i18n.Translate(
		i18n.FromContext(ctx),
		"Showing the first {0} of {1} rows — the full result is in the report file.",
		shown, rowCount,
	)
}

func digestTotal(digest *reportports.ReportDigest, columnID string) *float64 {
	envelope, err := readDigest(digest)
	if err != nil || envelope == nil {
		return nil
	}
	reader := envelope.Table.Reader()
	columns := reader.Columns()
	index := -1
	for i := range columns {
		if columns[i].ID == columnID {
			index = i
		}
	}
	if index < 0 {
		return nil
	}
	var total *float64
	for {
		batch, nextErr := reader.Next(context.Background())
		if nextErr != nil {
			return total
		}
		for i := range batch.Len {
			if batch.Level(i) != table.Total {
				continue
			}
			if value, ok := reportvalue.Float(columns[index].Type, &batch.Columns[index], i); ok {
				total = &value
			}
		}
	}
}

func digestContext(
	ctx context.Context,
	digest *reportports.ReportDigest,
	loc *time.Location,
	locale string,
	rowCount int64,
) (reporttemplate.ReportDigest, error) {
	envelope, err := readDigest(digest)
	if err != nil || envelope == nil {
		return reporttemplate.ReportDigest{}, err
	}

	reader := envelope.Table.Reader()
	columns := reader.Columns()
	formatters := render.Formatters(columns, format.Options{
		Locale:   render.Language(locale),
		Location: loc,
	})
	totalLabel := i18n.Translate(i18n.FromContext(ctx), "Total")

	out := reporttemplate.ReportDigest{
		Headers: make([]string, 0, len(columns)),
		Rows:    make([]reporttemplate.ReportRow, 0, reportports.MaxDigestRows),
	}
	for i := range columns {
		out.Headers = append(out.Headers, columns[i].Label)
	}

	for {
		batch, nextErr := reader.Next(ctx)
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return reporttemplate.ReportDigest{}, nextErr
		}
		for i := range batch.Len {
			level := batch.Level(i)
			if level == table.Total {
				cells := render.FormatRow(formatters, batch, i, totalLabel)
				out.Totals = make([]string, 0, len(cells))
				for _, cell := range cells {
					out.Totals = append(out.Totals, cell.Text)
				}
				continue
			}
			cells := render.FormatRow(formatters, batch, i, "")
			row := reporttemplate.ReportRow{
				Cells: make([]reporttemplate.ReportCell, 0, len(cells)),
			}
			for _, cell := range cells {
				row.Cells = append(
					row.Cells,
					reporttemplate.ReportCell{Text: cell.Text, Tone: cell.Tone},
				)
			}
			out.Rows = append(out.Rows, row)
		}
	}
	if len(out.Rows) == 0 {
		return reporttemplate.ReportDigest{}, nil
	}
	out.Note = digestNote(ctx, digest, len(out.Rows), rowCount)
	return out, nil
}
