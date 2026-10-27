package reportjobs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"

	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/reportvalue"
)

var errSidecarStopped = errors.New("report rows sidecar stopped")

type runReader struct {
	src       table.Reader
	heartbeat func()
	every     int
	seen      int
	digest    *digestCapture

	side     chan *table.Batch
	sideDone chan struct{}
	sideExit chan struct{}
	stop     chan struct{}
	pending  bool
	closed   bool
	srcErr   error
	stopOnce sync.Once
}

func newRunReader(src table.Reader, heartbeat func(), every int, digest *digestCapture) *runReader {
	return &runReader{
		src:       src,
		heartbeat: heartbeat,
		every:     every,
		digest:    digest,
		side:      make(chan *table.Batch),
		sideDone:  make(chan struct{}),
		sideExit:  make(chan struct{}),
		stop:      make(chan struct{}),
	}
}

func (r *runReader) Columns() []table.Column { return r.src.Columns() }

func (r *runReader) Truncated() bool { return r.src.Truncated() }

func (r *runReader) Next(ctx context.Context) (*table.Batch, error) {
	if r.pending {
		select {
		case <-r.sideDone:
		case <-r.sideExit:
		}
		r.pending = false
	}

	batch, err := r.src.Next(ctx)
	if err != nil {
		r.srcErr = err
		r.closeSide()
		return nil, err
	}

	if r.digest != nil {
		r.digest.capture(r.src.Columns(), batch)
	}
	r.seen += batch.Len
	if r.heartbeat != nil && r.every > 0 && r.seen >= r.every {
		r.seen = 0
		r.heartbeat()
	}

	select {
	case r.side <- batch:
		r.pending = true
	case <-r.sideExit:
	}
	return batch, nil
}

func (r *runReader) closeSide() {
	if !r.closed {
		r.closed = true
		close(r.side)
	}
}

func (r *runReader) Stop() {
	r.stopOnce.Do(func() { close(r.stop) })
}

func (r *runReader) Sidecar() table.Reader {
	return &sideReader{run: r}
}

type sideReader struct {
	run     *runReader
	started bool
}

func (s *sideReader) Columns() []table.Column { return s.run.src.Columns() }

func (s *sideReader) Truncated() bool { return s.run.src.Truncated() }

func (s *sideReader) Next(ctx context.Context) (*table.Batch, error) {
	if s.started {
		select {
		case s.run.sideDone <- struct{}{}:
		case <-s.run.stop:
			return nil, errSidecarStopped
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	s.started = true

	select {
	case batch, ok := <-s.run.side:
		if !ok {
			if s.run.srcErr != nil && !errors.Is(s.run.srcErr, io.EOF) {
				return nil, s.run.srcErr
			}
			return nil, io.EOF
		}
		return batch, nil
	case <-s.run.stop:
		return nil, errSidecarStopped
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type digestCapture struct {
	limit   int
	builder *table.Builder
	rows    int
	dropped bool
	total   []any
	err     error
}

func newDigestCapture(limit int) *digestCapture {
	return &digestCapture{limit: min(limit, reportports.MaxDigestRows)}
}

func (d *digestCapture) capture(columns []table.Column, b *table.Batch) {
	if d.builder == nil {
		d.builder = table.NewBuilder(columns)
	}
	for i := range b.Len {
		level := b.Level(i)
		switch {
		case level == table.Total:
			d.total = reportvalue.NativeRow(columns, b, i)
		case level != table.Detail:
		case d.rows < d.limit:
			if err := d.builder.Append(reportvalue.NativeRow(columns, b, i)...); err != nil &&
				d.err == nil {
				d.err = err
			}
			d.rows++
		default:
			d.dropped = true
		}
	}
}

func (d *digestCapture) digest(
	ctx context.Context,
	columns []table.Column,
) (*reportports.ReportDigest, error) {
	if d.err != nil {
		return nil, d.err
	}
	if d.builder == nil {
		d.builder = table.NewBuilder(columns)
	}
	if d.total != nil {
		if err := d.builder.AppendLevel(table.Total, d.total...); err != nil {
			return nil, err
		}
	}
	result := d.builder.Table()
	result.Truncated = d.dropped
	var buf bytes.Buffer
	if _, err := export.JSON(ctx, &buf, result.Reader(), export.Options{}); err != nil {
		return nil, err
	}
	return &reportports.ReportDigest{Envelope: buf.Bytes(), Truncated: d.dropped}, nil
}
