package cli

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/mattn/go-isatty"

	"github.com/emoss08/assay/internal/ui"
)

const (
	plainProgressInterval = 5 * time.Second

	// ttyRedrawInterval caps redraw frequency. Progress callbacks fire once per
	// completed unit, which for fast phases can be thousands per second; a
	// capped redraw keeps the bar smooth without making rendering measurable.
	ttyRedrawInterval = 50 * time.Millisecond

	progressBarWidth = 24
)

// newProgress returns a progress reporter suited to where its output lands. On
// a terminal it rewrites one line with a live bar, percentage, throughput and
// the item in flight; anywhere else it prints an occasional plain line,
// because carriage returns and erase-line escapes turn CI logs into noise.
func newProgress(out io.Writer, quiet bool, paint ui.Painter) func(item string, done, total int) {
	if quiet {
		return nil
	}

	if isTerminal(out) {
		var (
			mu    sync.Mutex
			start time.Time
			last  time.Time
		)

		return func(item string, done, total int) {
			mu.Lock()
			defer mu.Unlock()

			now := time.Now()
			if start.IsZero() {
				start = now
			}
			if done < total && now.Sub(last) < ttyRedrawInterval {
				return
			}
			last = now

			fraction := 0.0
			if total > 0 {
				fraction = float64(done) / float64(total)
			}

			elapsed := now.Sub(start).Round(time.Second)
			counts := fmt.Sprintf("%d/%d", done, total)
			percent := fmt.Sprintf("%3.0f%%", fraction*100)

			fmt.Fprintf(out, "\r\033[2K%s %s %s %s %s",
				paint.Bar(fraction, progressBarWidth),
				paint.Bold(percent),
				counts,
				paint.Muted(elapsed.String()),
				paint.Muted(shortenTail(item, 48)),
			)
		}
	}

	var (
		mu    sync.Mutex
		last  time.Time
		phase string
	)

	return func(item string, done, total int) {
		mu.Lock()
		defer mu.Unlock()

		// A new phase always announces itself; otherwise the first line of a short
		// phase can be swallowed by the previous phase's interval.
		if item == phase && done < total && time.Since(last) < plainProgressInterval {
			return
		}
		last = time.Now()
		phase = item
		fmt.Fprintf(out, "%s %d/%d\n", item, done, total)
	}
}

func finishProgress(out io.Writer, quiet bool) {
	if quiet || !isTerminal(out) {
		return
	}
	fmt.Fprint(out, "\r\033[2K")
}

func isTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	fd := file.Fd()

	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
