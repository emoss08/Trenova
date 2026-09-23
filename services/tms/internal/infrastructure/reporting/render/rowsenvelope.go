package render

import (
	"bufio"
	"io"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/reportrows"
)

/*
The machine-readable shape a run's rows take.

It is one encoder rather than two because there are two writers of it: the JSON
renderer, when somebody asked for JSON, and the rows sidecar, which is written
on every run whatever format was asked for. A second copy of this envelope
would be a second thing to keep in step, and the day they drifted is the day a
diff reported a change nobody made.

Values are raw. Decimals are exact strings and times are unix seconds, because
the point of this envelope — unlike the CSV and XLSX renderers, which run every
cell through reportfmt — is that what comes back out is what went in. A reader
comparing "$2,840.00" to "$2,840.0" cannot tell a formatting change from a
price change.
*/

// rowsEnvelope writes {meta, schema, rows, totals, summary} incrementally, so
// a caller streams rows through it rather than holding them.
type rowsEnvelope struct {
	out    *bufio.Writer
	schema []serviceports.ReportResultColumn
	// first tracks the row separator, since a streamed array cannot know
	// whether it is at the start without being told.
	first bool
	// encoded is reused across rows: a report has tens of columns and
	// millions of rows, and allocating per row is the whole cost.
	encoded []any
}

func newRowsEnvelope(sink io.Writer, schema []serviceports.ReportResultColumn) *rowsEnvelope {
	return &rowsEnvelope{
		out:     bufio.NewWriterSize(sink, 64*1024),
		schema:  schema,
		first:   true,
		encoded: make([]any, len(schema)),
	}
}

func (e *rowsEnvelope) WriteHead(meta *serviceports.ReportRunMeta) error {
	metaJSON, err := sonic.Marshal(reportrows.Meta{
		Title:       meta.Title,
		Description: meta.Description,
		GeneratedAt: meta.GeneratedAtUnix,
		Timezone:    meta.Timezone,
		RequestedBy: meta.RequestedBy,
		Params:      meta.Params,
	})
	if err != nil {
		return err
	}

	columns := make([]reportrows.Column, len(e.schema))
	for i := range e.schema {
		columns[i] = reportrows.Column{
			ID:      e.schema[i].ID,
			Label:   e.schema[i].Label,
			Type:    e.schema[i].Type,
			Format:  e.schema[i].Format,
			Display: e.schema[i].Display,
		}
	}
	schemaJSON, err := sonic.Marshal(columns)
	if err != nil {
		return err
	}

	return writeAll(e.out,
		[]byte(`{"meta":`), metaJSON,
		[]byte(`,"schema":`), schemaJSON,
		[]byte(`,"rows":[`),
	)
}

func (e *rowsEnvelope) WriteRow(row serviceports.ReportRow) error {
	for i := range e.schema {
		if i < len(row) {
			e.encoded[i] = jsonValue(row[i])

			continue
		}
		e.encoded[i] = nil
	}

	rowJSON, err := sonic.Marshal(e.encoded)
	if err != nil {
		return err
	}

	if !e.first {
		if _, err = e.out.WriteString(","); err != nil {
			return err
		}
	}
	e.first = false
	_, err = e.out.Write(rowJSON)

	return err
}

// WriteTail closes the array and appends the grand totals and the summary.
//
// totals may be nil: not every report has a total row, and a report that does
// not is different from one whose totals are zero.
func (e *rowsEnvelope) WriteTail(
	totals serviceports.ReportRow,
	rowCount int64,
	truncated bool,
) error {
	if _, err := e.out.WriteString(`]`); err != nil {
		return err
	}

	if totals != nil {
		for i := range e.schema {
			if i < len(totals) {
				e.encoded[i] = jsonValue(totals[i])

				continue
			}
			e.encoded[i] = nil
		}
		totalsJSON, err := sonic.Marshal(e.encoded)
		if err != nil {
			return err
		}
		if err = writeAll(e.out, []byte(`,"totals":`), totalsJSON); err != nil {
			return err
		}
	}

	summaryJSON, err := sonic.Marshal(reportrows.Summary{
		RowCount:  rowCount,
		Truncated: truncated,
	})
	if err != nil {
		return err
	}
	if err = writeAll(e.out, []byte(`,"summary":`), summaryJSON, []byte(`}`)); err != nil {
		return err
	}

	return e.out.Flush()
}

func writeAll(out *bufio.Writer, chunks ...[]byte) error {
	for _, chunk := range chunks {
		if _, err := out.Write(chunk); err != nil {
			return err
		}
	}

	return nil
}
