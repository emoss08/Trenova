package templateengine

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// ErrOutputTooLarge means rendering exceeded the byte budget.
var ErrOutputTooLarge = errors.New("template output exceeds the size limit")

// limitedWriter bounds a render by output size and honours cancellation.
//
// Both checks live in Write because that is the only place execution reliably
// yields. text/template has no unbounded loop construct — {{range}} iterates
// finite data — so a runaway template is always a runaway *writer*: a nested range
// over large collections, or a range that emits a lot per element. Failing the
// write aborts Execute from the inside, on the caller's goroutine, with no
// separate watchdog to leak.
type limitedWriter struct {
	ctx       context.Context
	dst       io.Writer
	remaining int64
	written   int64
}

func newLimitedWriter(ctx context.Context, dst io.Writer, limit int64) *limitedWriter {
	return &limitedWriter{ctx: ctx, dst: dst, remaining: limit}
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, fmt.Errorf("template render canceled: %w", err)
	}

	if w.remaining >= 0 && int64(len(p)) > w.remaining {
		return 0, fmt.Errorf("%w: over %d bytes", ErrOutputTooLarge, w.written+w.remaining)
	}

	n, err := w.dst.Write(p)
	if w.remaining >= 0 {
		w.remaining -= int64(n)
	}
	w.written += int64(n)

	return n, err
}

// Written reports how many bytes made it through.
func (w *limitedWriter) Written() int64 { return w.written }
