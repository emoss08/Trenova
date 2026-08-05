package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sourcegraph/conc/pool"

	"github.com/emoss08/assay/internal/cover"
)

// assembleWindows converts the completed counter windows to text profiles and
// builds the record from them.
//
// The init window — everything the package executed before the first test — is
// merged into every test's ranges, and merged *before* the empty-ranges check
// that marks a test always-run. That ordering is parity, not taste: a
// per-process profile always contained init execution, so a skipped test in a
// package with init-time statements had non-empty ranges and was attributable.
// Merging after the check would silently reclassify such tests.
func assembleWindows(
	ctx context.Context,
	opts Options,
	importPath, counterDir string,
	completed []harnessProgress,
) (*Record, error) {
	windows := make([]string, 0, len(completed)+1)
	windows = append(windows, initWindowName)
	for _, progress := range completed {
		windows = append(windows, progress.name)
	}

	if err := convertWindows(ctx, opts, counterDir, windows); err != nil {
		return nil, err
	}

	record := &Record{Version: recordVersion, Package: importPath, IndexedAt: opts.Commit}
	build := newBuilder()
	resolve := profileResolver(opts.Graph)

	degrade := func(unresolved string) {
		if unresolved != "" && record.Degraded == "" {
			record.Degraded = "unresolved coverage path: " + unresolved
		}
	}

	initRanges, unresolved, err := parseWindow(counterDir, initWindowName, build, resolve)
	if err != nil {
		return nil, err
	}
	degrade(unresolved)

	for _, progress := range completed {
		ranges, unresolvedTest, parseErr := parseWindow(counterDir, progress.name, build, resolve)
		if parseErr != nil {
			return nil, parseErr
		}
		degrade(unresolvedTest)

		ranges = append(ranges, initRanges...)

		test := TestCoverage{Name: progress.name, Ranges: ranges, Duration: progress.duration}
		if len(ranges) == 0 {
			test.AlwaysRun = true
			test.Note = "no executed statements attributed; the test may have skipped"
		}
		record.Tests = append(record.Tests, test)
	}

	record.Files = build.order

	return record, nil
}

// convertWindows runs `go tool covdata textfmt` for each window. The conversions
// are independent and each costs a process spawn, so they share a small pool —
// bounded well below the package workers' own parallelism.
func convertWindows(ctx context.Context, opts Options, counterDir string, windows []string) error {
	workers := min(max(1, runtime.GOMAXPROCS(0)/2), len(windows))

	converting := pool.New().
		WithMaxGoroutines(workers).
		WithContext(ctx).
		WithFirstError().
		WithCancelOnError()

	for _, window := range windows {
		converting.Go(func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}

			input := filepath.Join(counterDir, window)
			output := input + ".txt"
			if _, err := runCommand(ctx, counterDir, opts.Env, 0,
				"go", "tool", "covdata", "textfmt", "-i="+input, "-o="+output); err != nil {
				return fmt.Errorf("convert coverage window %s: %w", window, err)
			}

			return nil
		})
	}

	return converting.Wait()
}

func parseWindow(
	counterDir, window string,
	build *builder,
	resolve cover.Resolver,
) ([]Range, string, error) {
	file, err := os.Open(filepath.Join(counterDir, window) + ".txt")
	if err != nil {
		return nil, "", fmt.Errorf("open converted window %s: %w", window, err)
	}
	defer file.Close()

	blocks, err := cover.ParseProfile(file)
	if err != nil {
		return nil, "", fmt.Errorf("parse coverage window %s: %w", window, err)
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

	return ranges, unresolved, nil
}

// rebuilderFrom continues a record's file interning where assembly left it, so
// salvaged tests share file ids with the harness-collected ones.
func rebuilderFrom(record *Record) *builder {
	build := newBuilder()
	for _, file := range record.Files {
		build.intern(file)
	}

	return build
}
