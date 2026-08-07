package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/vcs"
)

func newIndexCommand(opts *options) *cobra.Command {
	var (
		allowDirty bool
		jobs       int
		timeout    time.Duration
		packages   []string
		quiet      bool
		legacy     bool
	)

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Build the line-to-test coverage index",
		Long: "index records which source lines each test executes, running every test in a\n" +
			"package inside one coverage-instrumented process. Records validate per package\n" +
			"by content: narrowing engages for any selection whose base tree matches the\n" +
			"indexed content of that package and its dependencies, even across commits.\n\n" +
			"Build it on your default branch, or on the commit you will diff against.",
		Args: cobra.NoArgs,
		Example: "  assay index\n" +
			"  assay index --jobs 8 --packages ./internal/...\n" +
			"  assay index --tags integration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			session, err := opts.open(cmd.Context())
			if err != nil {
				return err
			}
			if session.store == nil {
				return fmt.Errorf("indexing needs a cache directory; --no-cache is incompatible with index")
			}

			commit, err := session.requireIndexCommit(cmd.Context())
			if err != nil {
				return err
			}

			dirty, err := vcs.DirtyPaths(cmd.Context(), session.root)
			if err != nil {
				return err
			}
			if len(dirty) > 0 && !allowDirty {
				return dirtyTreeError(dirty)
			}
			if len(dirty) > 0 {
				commit = ""
				cmd.PrintErrln("warning: indexing a dirty tree; records will not be usable for narrowing")
			}

			digests, err := vcs.TreeDigests(cmd.Context(), session.root, "HEAD")
			if err != nil {
				return err
			}

			store, err := session.indexStore()
			if err != nil {
				return err
			}

			targets, err := resolveIndexTargets(session, packages)
			if err != nil {
				return err
			}

			stats, err := index.Build(cmd.Context(), index.Options{
				Root:        session.root,
				Commit:      commit,
				Graph:       session.graph,
				TreeDigests: digests,
				Store:       store,
				Tags:        opts.tags,
				Jobs:        jobs,
				Timeout:     timeout,
				Packages:    targets,
				Progress:    newProgress(cmd.ErrOrStderr(), quiet, opts.painter()),
				Legacy:      legacy,
			})
			if err != nil {
				return err
			}
			finishProgress(cmd.ErrOrStderr(), quiet)

			return printIndexStats(cmd, stats, commit)
		},
	}

	cmd.Flags().IntVar(&jobs, "jobs", 0, "packages to index concurrently (default: GOMAXPROCS)")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "per-test timeout (default: 60s)")
	cmd.Flags().StringSliceVar(&packages, "packages", nil, "limit indexing to these import paths")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress progress output")
	cmd.Flags().BoolVar(&allowDirty, "allow-dirty", false,
		"index a modified tree; the records cannot then be used for narrowing")
	cmd.Flags().BoolVar(&legacy, "legacy-collection", false,
		"run each test in its own process instead of one harness per package (much slower)")

	return cmd
}

// resolveIndexTargets expands --packages patterns against the graph. Relative
// patterns resolve against the caller's working directory the way go's own
// package patterns do — `./internal/...` typed inside a module must mean that
// module's packages, not a prefix of some import path. Matching is on whole path
// segments, so `.../cover` cannot quietly include `.../coverage`, and a package
// matched by several patterns is indexed once.
func resolveIndexTargets(s *session, requested []string) ([]string, error) {
	return resolvePackagePatterns(s, requested, s.graph.TestablePackages())
}

func resolvePackagePatterns(s *session, requested, candidates []string) ([]string, error) {
	if len(requested) == 0 {
		return nil, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve working directory: %w", err)
	}

	seen := make(map[string]struct{})
	var targets []string

	for _, want := range requested {
		matched := false
		for _, importPath := range candidates {
			pkg, ok := s.graph.Package(importPath)
			if !ok {
				continue
			}
			if !patternMatches(want, cwd, importPath, pkg.Dir) {
				continue
			}
			matched = true
			if _, dup := seen[importPath]; dup {
				continue
			}
			seen[importPath] = struct{}{}
			targets = append(targets, importPath)
		}
		if !matched {
			return nil, fmt.Errorf("no package in scope matches %q", want)
		}
	}
	sort.Strings(targets)

	return targets, nil
}

func patternMatches(pattern, cwd, importPath, dir string) bool {
	if pattern == "..." {
		return true
	}

	if isRelativePattern(pattern) {
		base, wild := cutWildcard(pattern)
		absBase := filepath.Clean(filepath.Join(cwd, base))
		if wild {
			return dir == absBase || strings.HasPrefix(dir, absBase+string(filepath.Separator))
		}

		return dir == absBase
	}

	base, wild := cutWildcard(pattern)
	if wild {
		return importPath == base || strings.HasPrefix(importPath, base+"/")
	}

	return importPath == pattern
}

func isRelativePattern(pattern string) bool {
	return pattern == "." || pattern == ".." ||
		strings.HasPrefix(pattern, "./") || strings.HasPrefix(pattern, "../")
}

func cutWildcard(pattern string) (string, bool) {
	if base, ok := strings.CutSuffix(pattern, "/..."); ok {
		return base, true
	}

	return pattern, false
}

func printIndexStats(cmd *cobra.Command, stats index.Stats, commit string) error {
	cmd.Printf("indexed %d, reused %d, of %d packages at %s in %s\n",
		stats.Indexed, stats.Reused, stats.Total, vcs.ShortSHA(commit), stats.Duration.Round(time.Millisecond))
	cmd.Printf("%d tests recorded, %d always-run\n", stats.Tests, stats.AlwaysRun)

	fallbacks := make([]string, 0, len(stats.Packages))
	fellBack := 0
	for _, outcome := range stats.Packages {
		fallbacks = append(fallbacks, outcome.Fallback)
		if outcome.Fallback != "" {
			fellBack++
		}
	}
	if fellBack > 0 {
		cmd.PrintErrf("%d packages fell back to per-process collection:\n", fellBack)
		for _, reason := range summariseReasons(fallbacks, 3) {
			cmd.PrintErrf("  %s\n", reason)
		}
	}

	if len(stats.Failures) == 0 {
		return nil
	}

	cmd.PrintErrf("%d packages failed to index:\n", len(stats.Failures))
	for _, failure := range stats.Failures {
		cmd.PrintErrf("  %s: %v\n", failure.Package, failure.Err)
	}

	return &ExitError{Code: 1}
}

func shortenTail(value string, width int) string {
	if len(value) <= width {
		return value
	}

	return "..." + value[len(value)-width+3:]
}

func dirtyTreeError(dirty []string) error {
	shown := dirty
	if len(shown) > 5 {
		shown = shown[:5]
	}

	return fmt.Errorf(
		"the working tree has %d uncommitted Go or module changes, so an index built now would\n"+
			"claim to describe HEAD while actually describing something else. Commit or stash them,\n"+
			"or pass --allow-dirty to build an index that narrowing will refuse to use.\n  %s",
		len(dirty), strings.Join(shown, "\n  "))
}
