package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/sourcegraph/conc/pool"

	"github.com/emoss08/assay/internal/proc"
)

const maxPackagesPerInvocation = 400

type Group struct {
	Packages []string
	Run      string
}

type Options struct {
	Root      string
	Groups    []Group
	Tags      []string
	ExtraArgs []string
	Jobs      int
	Stdout    io.Writer
	Stderr    io.Writer
}

type invocation struct {
	run      string
	packages []string
}

// Run executes every group, running invocations concurrently.
//
// Narrowed groups almost never share a -run pattern, so each becomes its own
// `go test` invocation — executed one after another, they forfeit exactly the
// parallelism narrowing was supposed to buy. Output stays readable by promotion:
// the front invocation streams live, later ones buffer, and each is flushed in
// submission order as the front finishes.
func Run(ctx context.Context, opts Options) (int, error) {
	var invocations []invocation
	for _, group := range opts.Groups {
		if len(group.Packages) == 0 {
			continue
		}
		for chunk := range chunks(group.Packages, maxPackagesPerInvocation) {
			invocations = append(invocations, invocation{run: group.Run, packages: chunk})
		}
	}
	if len(invocations) == 0 {
		return 0, nil
	}

	cores := runtime.GOMAXPROCS(0)
	jobs := opts.Jobs
	if jobs <= 0 {
		jobs = max(1, cores/2)
	}
	jobs = min(jobs, len(invocations))

	// Each `go test` schedules its own builds and test binaries, so concurrent
	// invocations must split the cores between them or the load multiplies.
	buildParallelism := max(1, cores/jobs)

	stdouts := make([]*promotedWriter, len(invocations))
	stderrs := make([]*promotedWriter, len(invocations))
	for i := range invocations {
		stdouts[i] = &promotedWriter{out: opts.Stdout}
		stderrs[i] = &promotedWriter{out: opts.Stderr}
	}
	stdouts[0].promote()
	stderrs[0].promote()

	codes := make([]int, len(invocations))
	var flushing sync.Mutex
	flushed := 0

	running := pool.New().
		WithMaxGoroutines(jobs).
		WithContext(ctx).
		WithFirstError().
		WithCancelOnError()

	done := make([]chan struct{}, len(invocations))
	for i := range done {
		done[i] = make(chan struct{})
	}

	for i := range invocations {
		running.Go(func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				close(done[i])

				return err
			}

			code, err := runInvocation(ctx, opts, invocations[i], buildParallelism, stdouts[i], stderrs[i])
			if err != nil {
				close(done[i])

				return err
			}
			codes[i] = code
			close(done[i])

			flushing.Lock()
			defer flushing.Unlock()
			for flushed < len(invocations) {
				select {
				case <-done[flushed]:
					flushed++
					if flushed < len(invocations) {
						stdouts[flushed].promote()
						stderrs[flushed].promote()
					}
				default:
					return nil
				}
			}

			return nil
		})
	}

	if err := running.Wait(); err != nil {
		return 1, err
	}

	// Anything still buffered belongs to invocations that finished out of order
	// behind a slower front; flush them in order now that everything is done.
	for i := range invocations {
		stdouts[i].promote()
		stderrs[i].promote()
	}

	worst := 0
	for _, code := range codes {
		if code > worst {
			worst = code
		}
	}

	return worst, nil
}

func GroupByRun(full []string, narrowed map[string]string) []Group {
	groups := make([]Group, 0, 1+len(narrowed))
	if len(full) > 0 {
		packages := append([]string(nil), full...)
		sort.Strings(packages)
		groups = append(groups, Group{Packages: packages})
	}

	byPattern := make(map[string][]string, len(narrowed))
	for pkg, pattern := range narrowed {
		byPattern[pattern] = append(byPattern[pattern], pkg)
	}

	patterns := make([]string, 0, len(byPattern))
	for pattern := range byPattern {
		patterns = append(patterns, pattern)
	}
	sort.Strings(patterns)

	for _, pattern := range patterns {
		packages := byPattern[pattern]
		sort.Strings(packages)
		groups = append(groups, Group{Packages: packages, Run: pattern})
	}

	return groups
}

func runInvocation(
	ctx context.Context,
	opts Options,
	inv invocation,
	buildParallelism int,
	stdout, stderr io.Writer,
) (int, error) {
	args := []string{"test", "-p", strconv.Itoa(buildParallelism)}
	if len(opts.Tags) > 0 {
		args = append(args, "-tags", strings.Join(opts.Tags, ","))
	}
	if inv.run != "" {
		args = append(args, "-run", inv.run)
	}
	args = append(args, opts.ExtraArgs...)
	args = append(args, inv.packages...)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = opts.Root
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	proc.Isolate(cmd)

	return exitVerdict(cmd.Run())
}

// exitVerdict separates a test verdict from an aborted process. A negative code
// means go test was killed by a signal, not that a test failed — reading it as a
// verdict made Ctrl-C look like success to `assay run` and like unsound narrowing
// to `assay verify`.
func exitVerdict(err error) (int, error) {
	if err == nil {
		return 0, nil
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return 0, err
	}

	if code := exitErr.ExitCode(); code >= 0 {
		return code, nil
	}

	return 0, fmt.Errorf("go test was terminated before reporting a verdict: %w", err)
}

// promotedWriter buffers until promoted, then streams. Promotion happens in
// invocation order, so interleaved concurrent runs still read as one sequential
// log.
type promotedWriter struct {
	mu   sync.Mutex
	live bool
	out  io.Writer
	buf  bytes.Buffer
}

func (w *promotedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.live && w.out != nil {
		return w.out.Write(p)
	}

	return w.buf.Write(p)
}

func (w *promotedWriter) promote() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.live && w.out != nil && w.buf.Len() > 0 {
		_, _ = w.out.Write(w.buf.Bytes())
	}
	w.buf.Reset()
	w.live = true
}

func chunks(in []string, size int) func(func([]string) bool) {
	return func(yield func([]string) bool) {
		for start := 0; start < len(in); start += size {
			end := min(start+size, len(in))
			if !yield(in[start:end]) {
				return
			}
		}
	}
}
