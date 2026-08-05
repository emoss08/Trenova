package index

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc/pool"

	"github.com/emoss08/assay/internal/cache"
	"github.com/emoss08/assay/internal/cover"
	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/proc"
)

const defaultTestTimeout = 60 * time.Second

var testNamePattern = regexp.MustCompile(`^(Test|Example|Fuzz)[^a-z]?`)

type Options struct {
	Root        string
	Commit      string
	Graph       *graph.Graph
	TreeDigests map[string]string
	Store       *cache.Store
	Tags        []string
	Jobs        int
	Timeout     time.Duration
	Env         []string
	Packages    []string
	Progress    func(pkg string, done, total int)
}

type PackageOutcome struct {
	Package   string
	Tests     int
	AlwaysRun int
	Reused    bool
	Degraded  string
	Err       error
}

type Stats struct {
	Total     int
	Indexed   int
	Reused    int
	Tests     int
	AlwaysRun int
	Failures  []PackageOutcome
	Duration  time.Duration
}

func Build(ctx context.Context, opts Options) (Stats, error) {
	started := time.Now()

	targets := opts.Packages
	if len(targets) == 0 {
		targets = opts.Graph.TestablePackages()
	}
	sort.Strings(targets)

	fingerprinter := NewFingerprinter(opts.Graph, opts.TreeDigests, opts.Tags)
	coverPkg := coverPkgArgument(opts.Graph)

	cores := runtime.GOMAXPROCS(0)

	jobs := opts.Jobs
	if jobs <= 0 {
		jobs = max(1, cores/2)
	}
	jobs = min(jobs, max(1, len(targets)))

	// Each `go test -c` fans out its own build actions, so an unconstrained inner
	// parallelism multiplies with jobs and buries the machine. Divide the cores
	// between the workers instead.
	buildParallelism := max(1, cores/jobs)

	outcomes := make([]PackageOutcome, len(targets))
	var done atomic.Int64

	indexing := pool.New().
		WithMaxGoroutines(jobs).
		WithContext(ctx).
		WithFirstError().
		WithCancelOnError()

	for i := range targets {
		indexing.Go(func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}

			outcomes[i] = indexPackage(ctx, opts, fingerprinter, coverPkg, targets[i], buildParallelism)

			if opts.Progress != nil {
				opts.Progress(targets[i], int(done.Add(1)), len(targets))
			}

			return nil
		})
	}

	if err := indexing.Wait(); err != nil {
		return Stats{}, err
	}

	stats := Stats{Total: len(targets), Duration: time.Since(started)}
	for _, outcome := range outcomes {
		switch {
		case outcome.Err != nil:
			stats.Failures = append(stats.Failures, outcome)
		case outcome.Reused:
			stats.Reused++
			stats.Tests += outcome.Tests
			stats.AlwaysRun += outcome.AlwaysRun
		default:
			stats.Indexed++
			stats.Tests += outcome.Tests
			stats.AlwaysRun += outcome.AlwaysRun
		}
	}

	return stats, nil
}

func coverPkgArgument(g *graph.Graph) string {
	modules := g.Modules()
	patterns := make([]string, 0, len(modules))
	for _, module := range modules {
		patterns = append(patterns, module+"/...")
	}

	return strings.Join(patterns, ",")
}

func indexPackage(
	ctx context.Context,
	opts Options,
	fingerprinter *Fingerprinter,
	coverPkg, importPath string,
	buildParallelism int,
) PackageOutcome {
	key := fingerprinter.For(importPath)

	if opts.Store != nil {
		if payload, hit := opts.Store.Get(key); hit {
			if record, err := Decode(payload); err == nil {
				return PackageOutcome{
					Package:   importPath,
					Tests:     len(record.Tests),
					AlwaysRun: len(record.AlwaysRunTests()),
					Reused:    true,
					Degraded:  record.Degraded,
				}
			}
		}
	}

	record, err := collectPackage(ctx, opts, coverPkg, importPath, buildParallelism)
	if err != nil {
		return PackageOutcome{Package: importPath, Err: err}
	}

	if opts.Store != nil {
		if payload, encodeErr := record.Encode(); encodeErr == nil {
			_ = opts.Store.Put(key, payload)
		}
	}

	return PackageOutcome{
		Package:   importPath,
		Tests:     len(record.Tests),
		AlwaysRun: len(record.AlwaysRunTests()),
		Degraded:  record.Degraded,
	}
}

func collectPackage(ctx context.Context, opts Options, coverPkg, importPath string, buildParallelism int) (*Record, error) {
	pkg, ok := opts.Graph.Package(importPath)
	if !ok {
		return nil, fmt.Errorf("package %s is not in the graph", importPath)
	}

	workdir, err := os.MkdirTemp("", "assay-index-")
	if err != nil {
		return nil, fmt.Errorf("create work directory: %w", err)
	}
	defer os.RemoveAll(workdir)

	binary := filepath.Join(workdir, "test.bin")
	if buildErr := buildTestBinary(ctx, opts, coverPkg, importPath, binary, buildParallelism); buildErr != nil {
		return nil, buildErr
	}

	if _, statErr := os.Stat(binary); statErr != nil {
		return &Record{Version: recordVersion, Package: importPath, IndexedAt: opts.Commit}, nil
	}

	names, err := listTests(ctx, opts, binary, pkg.Dir)
	if err != nil {
		return nil, err
	}

	record := &Record{Version: recordVersion, Package: importPath, IndexedAt: opts.Commit}
	build := newBuilder()
	resolve := profileResolver(opts.Graph)

	for _, name := range names {
		test, unresolved := runTest(ctx, opts, binary, pkg.Dir, workdir, name, build, resolve)
		if unresolved != "" && record.Degraded == "" {
			record.Degraded = "unresolved coverage path: " + unresolved
		}
		record.Tests = append(record.Tests, test)
	}

	record.Files = build.order

	return record, nil
}

func buildTestBinary(
	ctx context.Context,
	opts Options,
	coverPkg, importPath, output string,
	buildParallelism int,
) error {
	args := []string{
		"test", "-c", "-covermode=atomic",
		"-coverpkg=" + coverPkg,
		"-p", strconv.Itoa(buildParallelism),
		"-o", output,
	}
	if len(opts.Tags) > 0 {
		args = append(args, "-tags", strings.Join(opts.Tags, ","))
	}
	args = append(args, importPath)

	if _, err := runCommand(ctx, opts.Root, opts.Env, 0, "go", args...); err != nil {
		return fmt.Errorf("compile test binary for %s: %w", importPath, err)
	}

	return nil
}

func listTests(ctx context.Context, opts Options, binary, dir string) ([]string, error) {
	out, err := runCommand(ctx, dir, opts.Env, opts.timeout(), binary, "-test.list", ".*")
	if err != nil {
		return nil, fmt.Errorf("list tests: %w", err)
	}

	var names []string
	for line := range strings.SplitSeq(string(out), "\n") {
		name := strings.TrimSpace(line)
		if name == "" || !testNamePattern.MatchString(name) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	return names, nil
}

func runTest(
	ctx context.Context,
	opts Options,
	binary, dir, workdir, name string,
	build *builder,
	resolve cover.Resolver,
) (TestCoverage, string) {
	profilePath := filepath.Join(workdir, "profile.out")
	_ = os.Remove(profilePath)

	started := time.Now()
	_, runErr := runCommand(ctx, dir, opts.Env, opts.timeout(),
		binary, "-test.run", "^"+regexp.QuoteMeta(name)+"$", "-test.coverprofile", profilePath)
	elapsed := time.Since(started)

	file, openErr := os.Open(profilePath)
	if openErr != nil {
		note := "no coverage profile produced"
		if runErr != nil {
			note += ": " + runErr.Error()
		}

		return TestCoverage{Name: name, AlwaysRun: true, Note: note, Duration: elapsed}, ""
	}
	defer file.Close()

	blocks, parseErr := cover.ParseProfile(file)
	if parseErr != nil {
		return TestCoverage{Name: name, AlwaysRun: true, Note: parseErr.Error(), Duration: elapsed}, ""
	}

	var ranges []Range
	var unresolved string
	for _, fileBlocks := range blocks {
		abs, ok := resolve(fileBlocks.ProfileName)
		if !ok {
			if unresolved == "" {
				unresolved = fileBlocks.ProfileName
			}

			continue
		}
		id := build.intern(abs)
		for _, block := range fileBlocks.Blocks {
			ranges = append(ranges, Range{
				FileID: id,
				Start:  uint32(block.StartLine),
				End:    uint32(block.EndLine),
			})
		}
	}

	if len(ranges) == 0 {
		return TestCoverage{
			Name:      name,
			AlwaysRun: true,
			Note:      "no executed statements attributed; the test may have skipped",
			Duration:  elapsed,
		}, unresolved
	}

	return TestCoverage{Name: name, Ranges: ranges, Duration: elapsed}, unresolved
}

func profileResolver(g *graph.Graph) cover.Resolver {
	return func(profileName string) (string, bool) {
		importPath, file := cover.SplitProfileName(profileName)
		if importPath == "" || file == "" {
			return "", false
		}
		pkg, ok := g.Package(importPath)
		if !ok {
			return "", false
		}

		return filepath.Join(pkg.Dir, file), true
	}
}

func (o Options) timeout() time.Duration {
	if o.Timeout > 0 {
		return o.Timeout
	}

	return defaultTestTimeout
}

func runCommand(
	ctx context.Context,
	dir string,
	env []string,
	timeout time.Duration,
	name string,
	args ...string,
) ([]byte, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = env
	}
	proc.Isolate(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return stdout.Bytes(), fmt.Errorf("timed out after %s", timeout)
		}

		return stdout.Bytes(), fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return stdout.Bytes(), nil
}
