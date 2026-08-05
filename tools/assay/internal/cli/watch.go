package cli

import (
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/watch"
)

func newWatchCommand(opts *options) *cobra.Command {
	var (
		jobs     int
		timeout  time.Duration
		debounce time.Duration
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Re-run the affected tests on every save",
		Long: "watch keeps a session open: every time a Go file is saved, the tests that change\n" +
			"can actually affect are re-run — narrowed to individual tests where the coverage\n" +
			"index proves that safe, whole packages otherwise. Test binaries stay warm between\n" +
			"saves, so the loop is bounded by the tests themselves, not by go test start-up.\n\n" +
			"Failing tests are sticky: they re-run at the front of every following cycle until\n" +
			"they pass. An index built at HEAD (assay index) makes the loop narrow; without\n" +
			"one it still works at package precision.",
		Args:    cobra.NoArgs,
		Example: "  assay index && assay watch\n  assay watch --jobs 4",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			if err := opts.validate(); err != nil {
				return err
			}

			root, err := resolveRoot(ctx, opts.root)
			if err != nil {
				return err
			}

			binaries, err := watch.NewBinaryCache()
			if err != nil {
				return err
			}
			defer binaries.Close()

			watcher, err := watch.NewWatcher(root, debounce)
			if err != nil {
				return err
			}
			defer watcher.Close()

			planner := &watch.Planner{
				Root:  root,
				Tags:  opts.tags,
				Store: opts.store(),
			}

			cmd.PrintErrf("watching %s — save a Go file to run its tests (ctrl-c to stop)\n", root)

			go watcher.Run(ctx)

			err = watch.Loop(ctx, watch.LoopOptions{
				Planner:  planner,
				Binaries: binaries,
				Batches:  watcher.Batches(),
				Out:      cmd.OutOrStdout(),
				UseColor: opts.useColor(),
				Jobs:     jobs,
				Timeout:  timeout,
			})
			if err != nil && ctx.Err() == nil {
				return err
			}

			cmd.PrintErrln("\nwatch stopped")

			return nil
		},
	}

	cmd.Flags().IntVar(&jobs, "jobs", 0, "packages to run concurrently (default: half of GOMAXPROCS)")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "per-package run timeout (default: 2m)")
	cmd.Flags().DurationVar(&debounce, "debounce", 0,
		fmt.Sprintf("quiet period before a save triggers a cycle (default: %s)", 250*time.Millisecond))

	return cmd
}
