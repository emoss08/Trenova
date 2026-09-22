package render

import (
	"context"
	"io"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

var _ serviceports.ReportDatasetReader = (*RowsSidecar)(nil)

/*
RowsSidecar is the machine-readable copy of a run, written whatever format the
run was asked for.

It exists because only the JSON renderer preserves raw values: CSV and XLSX put
every cell through reportfmt, so they hold "$2,840.00" — where a currency or
precision change reads exactly like a price change — and a PDF cannot be read
back as rows at all. Comparing two runs of those is guesswork.

It is an observer rather than a second renderer because ReportDatasetReader is
a forward-only cursor and the chosen renderer drives it. Two renderers cannot
share one, and re-running the query would give two results free to disagree —
the one thing a diff may not rest on. So the rows are encoded as they go past,
in the same pass, and the sidecar holds exactly the rows the run produced.
*/
type RowsSidecar struct {
	inner    serviceports.ReportDatasetReader
	envelope *rowsEnvelope

	// err is sticky. The first write failure stops the sidecar and is reported
	// from Finish; the run's own artifact is unaffected, and a run without a
	// sidecar is already a state the reader understands.
	err error

	// totals is memoized because Finish needs it whether or not the renderer
	// asked for it, and asking the reader twice is a second aggregate query.
	totals      serviceports.ReportRow
	totalsErr   error
	totalsTaken bool
}

// NewRowsSidecar wraps inner so every row pulled from it is also encoded into
// sink. The envelope head is written immediately, so sink receives bytes even
// for a run that yields no rows.
func NewRowsSidecar(
	inner serviceports.ReportDatasetReader,
	sink io.Writer,
	meta serviceports.ReportRunMeta,
) *RowsSidecar {
	sidecar := &RowsSidecar{
		inner:    inner,
		envelope: newRowsEnvelope(sink, inner.Schema()),
	}
	sidecar.err = sidecar.envelope.WriteHead(&meta)

	return sidecar
}

func (s *RowsSidecar) Schema() []serviceports.ReportResultColumn { return s.inner.Schema() }

func (s *RowsSidecar) Next(ctx context.Context) (serviceports.ReportRow, error) {
	row, err := s.inner.Next(ctx)
	if err != nil {
		return row, err
	}

	if s.err == nil {
		s.err = s.envelope.WriteRow(row)
	}

	return row, nil
}

func (s *RowsSidecar) Totals(ctx context.Context) (serviceports.ReportRow, error) {
	if !s.totalsTaken {
		s.totalsTaken = true
		s.totals, s.totalsErr = s.inner.Totals(ctx)
	}

	return s.totals, s.totalsErr
}

func (s *RowsSidecar) RowCount() int64 { return s.inner.RowCount() }

func (s *RowsSidecar) Truncated() bool { return s.inner.Truncated() }

func (s *RowsSidecar) Close() error { return s.inner.Close() }

// Finish closes the envelope and flushes it.
//
// It is the caller's job rather than Close's, and it takes its own totals,
// because a renderer is not obliged to ask for them — CSV does, PDF does not —
// and the sidecar's shape may not depend on which format a person picked.
func (s *RowsSidecar) Finish(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	totals, err := s.Totals(ctx)
	if err != nil {
		s.err = err

		return err
	}

	s.err = s.envelope.WriteTail(totals, s.inner.RowCount(), s.inner.Truncated())

	return s.err
}
