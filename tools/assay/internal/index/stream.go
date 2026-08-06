package index

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/assay/internal/proc"
)

type harnessProgress struct {
	name     string
	exit     int
	duration time.Duration
}

type harnessResult struct {
	completed []harnessProgress
	// stalled names the test that was in flight when the run died — by deadline,
	// panic, or harness failure. Empty means every listed test finished.
	stalled string
	err     error
}

// runHarness executes the harness binary and follows its progress line by line.
//
// The deadline is rolling: each completed test resets it, so one hanging test
// cannot be confused with a package that simply has many tests. When it fires,
// the whole process group is killed and the caller salvages — every window
// reported before the stall is complete on disk, because the harness prints a
// test's line only after its snapshot is written.
func runHarness(
	ctx context.Context,
	opts Options,
	binary, dir, listFile, counterDir string,
	names []string,
) harnessResult {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir = dir

	env := opts.Env
	if len(env) == 0 {
		env = os.Environ()
	}
	cmd.Env = append(append([]string(nil), env...),
		EnvTestList+"="+listFile,
		EnvCounterDir+"="+counterDir,
	)
	proc.Isolate(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return harnessResult{err: fmt.Errorf("open harness stdout: %w", err)}
	}
	stderr := newTailBuffer(32 << 10)
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return harnessResult{err: fmt.Errorf("start harness: %w", err)}
	}

	timeout := opts.timeout()
	deadline := time.AfterFunc(timeout, cancel)
	// The timer must outlive the scanner: if the reader dies first — an
	// oversized line, a closed descriptor — the child can fill the pipe and
	// block forever, and this deadline is the only thing left that can kill it.
	// Stopping it before Wait was a proven deadlock.
	defer deadline.Stop()

	var completed []harnessProgress
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		if len(completed) >= len(names) {
			continue
		}
		progress, ok := parseHarnessLine(scanner.Text(), names[len(completed)])
		if !ok {
			continue
		}
		completed = append(completed, progress)
		deadline.Reset(timeout)
	}

	scanErr := scanner.Err()
	if scanErr != nil {
		// The reader is done but the child may not be; without a drain the child
		// blocks on a full pipe and Wait never returns.
		cancel()
		_, _ = io.Copy(io.Discard, stdout)
	}

	waitErr := cmd.Wait()

	result := harnessResult{completed: completed}
	if len(completed) < len(names) {
		result.stalled = names[len(completed)]
		detail := strings.TrimSpace(stderr.String())
		if detail == "" && scanErr != nil {
			detail = "harness output unreadable: " + scanErr.Error()
		}
		if detail == "" && waitErr != nil {
			detail = waitErr.Error()
		}
		result.err = fmt.Errorf("harness stopped at %s after %d of %d tests: %s",
			result.stalled, len(completed), len(names), detail)
	}

	return result
}

// tailBuffer keeps only the last cap bytes written. The harness holds one
// stderr buffer for a whole package's tests; unbounded, a chatty suite times
// the resident cost by jobs, and only the tail ever reaches an error message.
type tailBuffer struct {
	limit int
	data  []byte
}

func newTailBuffer(limit int) *tailBuffer {
	return &tailBuffer{limit: limit}
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	if len(b.data) > b.limit {
		b.data = b.data[len(b.data)-b.limit:]
	}

	return len(p), nil
}

func (b *tailBuffer) String() string {
	return string(b.data)
}

// parseHarnessLine accepts only the line announcing the next expected test, so a
// test that itself prints something shaped like a progress line cannot desync
// the collector.
//
// The prefix is searched anywhere in the line, not only at its start: a test
// that prints without a trailing newline glues the harness's own output onto
// its fragment, and requiring a line boundary silently pushed such packages
// onto the salvage path. Forgery stays impossible — the name after the prefix
// must equal the next expected test exactly.
func parseHarnessLine(line, expected string) (harnessProgress, bool) {
	at := strings.LastIndex(line, harnessLinePrefix)
	if at < 0 {
		return harnessProgress{}, false
	}
	rest := line[at+len(harnessLinePrefix):]

	fields := strings.Fields(rest)
	if len(fields) != 3 || fields[0] != expected {
		return harnessProgress{}, false
	}

	exitField, ok := strings.CutPrefix(fields[1], "exit=")
	if !ok {
		return harnessProgress{}, false
	}
	exit, err := strconv.Atoi(exitField)
	if err != nil {
		return harnessProgress{}, false
	}

	nsField, ok := strings.CutPrefix(fields[2], "ns=")
	if !ok {
		return harnessProgress{}, false
	}
	ns, err := strconv.ParseInt(nsField, 10, 64)
	if err != nil || ns < 0 {
		return harnessProgress{}, false
	}

	return harnessProgress{name: fields[0], exit: exit, duration: time.Duration(ns)}, true
}
