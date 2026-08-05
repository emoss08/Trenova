package cli

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
)

const plainProgressInterval = 5 * time.Second

// newProgress returns a progress reporter suited to where its output lands. On a
// terminal it rewrites one line; anywhere else it prints an occasional plain
// line, because carriage returns and erase-line escapes turn CI logs into noise.
func newProgress(out io.Writer, quiet bool) func(item string, done, total int) {
	if quiet {
		return nil
	}

	if isTerminal(out) {
		return func(item string, done, total int) {
			fmt.Fprintf(out, "\r\033[2K%d/%d %s", done, total, shortenTail(item, 60))
		}
	}

	var mu sync.Mutex
	last := time.Now()

	return func(_ string, done, total int) {
		mu.Lock()
		defer mu.Unlock()

		if done < total && time.Since(last) < plainProgressInterval {
			return
		}
		last = time.Now()
		fmt.Fprintf(out, "%d/%d\n", done, total)
	}
}

func finishProgress(out io.Writer, quiet bool) {
	if quiet || !isTerminal(out) {
		return
	}
	fmt.Fprintln(out)
}

func isTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	fd := file.Fd()

	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
