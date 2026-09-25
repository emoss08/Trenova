package uploadpipe

import (
	"context"
	"errors"
	"io"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
)

// ErrTooLarge is returned by a pipe's writer once what was written would pass
// the pipe's size limit.
var ErrTooLarge = errors.New("upload exceeds the maximum size")

type Params struct {
	Key         string
	ContentType string
	Metadata    map[string]string
	// MaxBytes bounds what may be written; zero or less means no bound.
	MaxBytes int64
}

// Pipe streams what is written to it straight into object storage, so an
// artifact rendered row by row never has to fit in memory to learn its length
// before it is uploaded.
type Pipe struct {
	sink   io.Writer
	writer *io.PipeWriter
	done   chan error
	bytes  int64
}

func Open(ctx context.Context, client storage.Client, params Params) *Pipe {
	pipeReader, pipeWriter := io.Pipe()

	var sink io.Writer = pipeWriter
	if params.MaxBytes > 0 {
		sink = NewLimitWriter(pipeWriter, params.MaxBytes)
	}

	pipe := &Pipe{
		sink:   sink,
		writer: pipeWriter,
		done:   make(chan error, 1),
	}

	go func() {
		info, err := client.Upload(ctx, &storage.UploadParams{
			Key:         params.Key,
			ContentType: params.ContentType,
			Size:        -1,
			Body:        pipeReader,
			Metadata:    params.Metadata,
		})
		if err != nil {
			pipeReader.CloseWithError(err)
			pipe.done <- err

			return
		}
		pipe.bytes = info.Size
		pipe.done <- nil
	}()

	return pipe
}

// Writer is where the artifact is written.
func (p *Pipe) Writer() io.Writer {
	return p.sink
}

// Close ends the stream and waits for the upload, returning the stored size.
func (p *Pipe) Close() (int64, error) {
	if err := p.writer.Close(); err != nil {
		<-p.done

		return 0, err
	}
	if err := <-p.done; err != nil {
		return 0, err
	}

	return p.bytes, nil
}

// Abort tears the stream down and waits for the upload goroutine, so a failed
// render never leaves one running past its caller.
func (p *Pipe) Abort(cause error) {
	_ = p.writer.CloseWithError(cause)
	<-p.done
}

// LimitWriter refuses a write that would take it past its limit, rather than
// writing part of it.
type LimitWriter struct {
	inner     io.Writer
	remaining int64
}

func NewLimitWriter(inner io.Writer, limit int64) *LimitWriter {
	return &LimitWriter{inner: inner, remaining: limit}
}

func (w *LimitWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, ErrTooLarge
	}
	n, err := w.inner.Write(p)
	w.remaining -= int64(n)

	return n, err
}

// Heartbeat reports progress to whatever is watching the work, such as
// activity.RecordHeartbeat. A nil heartbeat reports nothing.
type Heartbeat func(ctx context.Context, details ...any)

// RowHeartbeat beats once every so many rows, so a long render or export is
// not mistaken for a lost worker.
type RowHeartbeat struct {
	ctx   context.Context
	beat  Heartbeat
	every int64
	rows  int64
}

func NewRowHeartbeat(ctx context.Context, every int64, beat Heartbeat) *RowHeartbeat {
	if every <= 0 {
		every = 1
	}

	return &RowHeartbeat{ctx: ctx, beat: beat, every: every}
}

// Tick counts one row and beats when the count reaches a multiple of every.
func (h *RowHeartbeat) Tick() {
	h.rows++
	if h.beat != nil && h.rows%h.every == 0 {
		h.beat(h.ctx, h.rows)
	}
}

// Rows is how many rows have been counted.
func (h *RowHeartbeat) Rows() int64 {
	return h.rows
}

// HeartbeatingReader beats as rows are read from a report dataset.
type HeartbeatingReader struct {
	inner     services.ReportDatasetReader
	heartbeat *RowHeartbeat
}

func NewHeartbeatingReader(
	ctx context.Context,
	inner services.ReportDatasetReader,
	every int64,
	beat Heartbeat,
) *HeartbeatingReader {
	return &HeartbeatingReader{inner: inner, heartbeat: NewRowHeartbeat(ctx, every, beat)}
}

func (r *HeartbeatingReader) Schema() []services.ReportResultColumn { return r.inner.Schema() }

func (r *HeartbeatingReader) Next(ctx context.Context) (services.ReportRow, error) {
	row, err := r.inner.Next(ctx)
	if err == nil {
		r.heartbeat.Tick()
	}

	return row, err
}

func (r *HeartbeatingReader) Totals(ctx context.Context) (services.ReportRow, error) {
	return r.inner.Totals(ctx)
}

func (r *HeartbeatingReader) RowCount() int64 { return r.inner.RowCount() }

func (r *HeartbeatingReader) Truncated() bool { return r.inner.Truncated() }

func (r *HeartbeatingReader) Close() error { return r.inner.Close() }
