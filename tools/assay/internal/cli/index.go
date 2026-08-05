package cli

import (
	"fmt"
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
	)

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Build the line-to-test coverage index",
		Long: "index runs each test in isolation with coverage enabled and records which source\n" +
			"lines it executed. The result is pinned to the current commit: narrowing only\n" +
			"engages when a selection's base commit matches the commit the index was built at.\n\n" +
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

			progress := func(pkg string, done, total int) {
				if quiet {
					return
				}
				cmd.PrintErrf("\r\033[2Kindexing %d/%d %s", done, total, shortenTail(pkg, 60))
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
				Progress:    progress,
			})
			if err != nil {
				return err
			}
			if !quiet {
				cmd.PrintErrln()
			}

			return printIndexStats(cmd, stats, commit)
		},
	}

	cmd.Flags().IntVar(&jobs, "jobs", 0, "packages to index concurrently (default: GOMAXPROCS)")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "per-test timeout (default: 60s)")
	cmd.Flags().StringSliceVar(&packages, "packages", nil, "limit indexing to these import paths")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress progress output")
	cmd.Flags().BoolVar(&allowDirty, "allow-dirty", false,
		"index a modified tree; the records cannot then be used for narrowing")

	return cmd
}

func resolveIndexTargets(s *session, requested []string) ([]string, error) {
	if len(requested) == 0 {
		return nil, nil
	}

	testable := make(map[string]struct{})
	for _, pkg := range s.graph.TestablePackages() {
		testable[pkg] = struct{}{}
	}

	var targets []string
	for _, want := range requested {
		trimmed := strings.TrimSuffix(strings.TrimSuffix(want, "..."), "/")
		matched := false
		for pkg := range testable {
			if pkg == want || strings.HasPrefix(pkg, trimmed) {
				targets = append(targets, pkg)
				matched = true
			}
		}
		if !matched {
			return nil, fmt.Errorf("no testable package matches %q", want)
		}
	}

	return targets, nil
}

func printIndexStats(cmd *cobra.Command, stats index.Stats, commit string) error {
	cmd.Printf("indexed %d, reused %d, of %d packages at %s in %s\n",
		stats.Indexed, stats.Reused, stats.Total, shortSHA(commit), stats.Duration.Round(time.Millisecond))
	cmd.Printf("%d tests recorded, %d always-run\n", stats.Tests, stats.AlwaysRun)

	if len(stats.Failures) == 0 {
		return nil
	}

	cmd.PrintErrf("%d packages failed to index:\n", len(stats.Failures))
	for _, failure := range stats.Failures {
		cmd.PrintErrf("  %s: %v\n", failure.Package, failure.Err)
	}

	return &ExitError{Code: 1}
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	if sha == "" {
		return "unknown"
	}

	return sha
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
