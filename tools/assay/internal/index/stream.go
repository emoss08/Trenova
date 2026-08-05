package index

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
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
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return harnessResult{err: fmt.Errorf("start harness: %w", err)}
	}

	timeout := opts.timeout()
	deadline := time.AfterFunc(timeout, cancel)

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
	deadline.Stop()

	waitErr := cmd.Wait()

	result := harnessResult{completed: completed}
	if len(completed) < len(names) {
		result.stalled = names[len(completed)]
		detail := strings.TrimSpace(stderr.String())
		if detail == "" && waitErr != nil {
			detail = waitErr.Error()
		}
		result.err = fmt.Errorf("harness stopped at %s after %d of %d tests: %s",
			result.stalled, len(completed), len(names), detail)
	}

	return result
}

// parseHarnessLine accepts only the line announcing the next expected test, so a
// test that itself prints something shaped like a progress line cannot desync
// the collector.
func parseHarnessLine(line, expected string) (harnessProgress, bool) {
	rest, ok := strings.CutPrefix(line, harnessLinePrefix)
	if !ok {
		return harnessProgress{}, false
	}

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
