package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/cache"
	"github.com/emoss08/assay/internal/ui"
	"github.com/emoss08/assay/internal/vcs"
)

// Version is stamped by the release build via -ldflags; a source build
// reports itself as a development version with the commit it was built from,
// so a bug report can always say exactly which code produced it.
var Version = ""

func resolveVersion() string {
	if Version != "" {
		return Version
	}

	revision, modified := "", ""
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.modified":
				if setting.Value == "true" {
					modified = "-dirty"
				}
			}
		}
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	if revision == "" {
		return "devel"
	}

	return "devel-" + revision + modified
}

type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

type options struct {
	since    string
	files    string
	root     string
	cacheDir string
	tags     []string
	all      bool
	noCache  bool
	noIndex  bool
	asJSON   bool
	verbose  bool
	noColor  bool
}

func (o *options) useColor() bool {
	return !o.noColor && os.Getenv("NO_COLOR") == "" && isTerminal(os.Stdout)
}

func (o *options) painter() ui.Painter {
	return ui.New(o.useColor())
}

func NewRootCommand() *cobra.Command {
	opts := &options{}

	root := &cobra.Command{
		Use:   "assay",
		Short: "Test intelligence for Go",
		Long: "assay runs the tests a change can actually affect, instead of all of them.\n\n" +
			"It builds the workspace package graph, attributes changed files to packages, and\n" +
			"narrows each affected package to the individual tests that cover the changed\n" +
			"lines — falling back to the whole package whenever that cannot be proven safe.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	flags := root.PersistentFlags()
	flags.StringVar(&opts.since, "since", "", "diff against this ref's merge-base with HEAD")
	flags.StringVar(&opts.files, "files", "", "read changed paths from a file, or - for stdin")
	flags.StringVar(&opts.root, "root", "", "workspace root (default: git repository root)")
	flags.StringSliceVar(&opts.tags, "tags", nil, "build tags")
	flags.BoolVar(&opts.all, "all", false, "skip selection and use every testable package")
	flags.BoolVar(&opts.noCache, "no-cache", false, "reload the package graph instead of reusing a cached one")
	flags.BoolVar(&opts.noIndex, "no-index", false, "skip line-level narrowing and select whole packages")
	flags.StringVar(&opts.cacheDir, "cache-dir", os.Getenv("ASSAY_CACHE"),
		"cache directory for the graph and coverage index (default: user cache dir)")
	flags.BoolVarP(&opts.verbose, "verbose", "v", false, "list every selected package and ignored file")
	flags.BoolVar(&opts.noColor, "no-color", false, "disable colored output")

	root.AddCommand(
		newSelectCommand(opts),
		newRunCommand(opts),
		newWatchCommand(opts),
		newIndexCommand(opts),
		newMutateCommand(opts),
		newExplainCommand(opts),
		newVerifyCommand(opts),
		newFlakesCommand(opts),
		newRiskCommand(opts),
		newVersionCommand(),
	)

	return root
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the assay version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Println("assay " + resolveVersion())
		},
	}
}

func (o *options) store() *cache.Store {
	if o.noCache {
		return nil
	}

	store, err := cache.NewStore(o.cacheDir)
	if err != nil {
		return nil
	}

	return store
}

func (o *options) validate() error {
	var sources []string
	if o.all {
		sources = append(sources, "--all")
	}
	if o.files != "" {
		sources = append(sources, "--files")
	}
	if o.since != "" {
		sources = append(sources, "--since")
	}
	if len(sources) > 1 {
		return fmt.Errorf("%s are mutually exclusive; pick one change source", strings.Join(sources, " and "))
	}

	return nil
}

func (o *options) changeSet(ctx context.Context, root string) (vcs.Result, error) {
	if o.files != "" {
		changes, err := readFileList(o.files, root, os.Stdin)
		if err != nil {
			return vcs.Result{}, err
		}

		return vcs.Result{Changes: changes, BaseMode: vcs.BaseWorkingTree}, nil
	}

	return vcs.Changes(ctx, vcs.Options{
		Root:             root,
		Base:             o.since,
		IncludeUntracked: true,
	})
}

func readFileList(source, root string, stdin io.Reader) ([]vcs.Change, error) {
	var raw []byte
	var err error
	if source == "-" {
		raw, err = io.ReadAll(stdin)
	} else {
		raw, err = os.ReadFile(source)
	}
	if err != nil {
		return nil, fmt.Errorf("read file list: %w", err)
	}

	lines := strings.Split(string(raw), "\n")
	changes := make([]vcs.Change, 0, len(lines))
	for _, line := range lines {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		changes = append(changes, vcs.Change{Path: filepath.Clean(path), Status: "M"})
	}

	return changes, nil
}

func resolveRoot(ctx context.Context, explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	root, err := vcs.RepoRoot(ctx, cwd)
	if err != nil {
		return "", errors.New("not inside a git repository; pass --root")
	}

	return root, nil
}

func goTestArgs(cmd *cobra.Command, args []string) ([]string, error) {
	dash := cmd.Flags().ArgsLenAtDash()
	if dash < 0 {
		if len(args) > 0 {
			return nil, fmt.Errorf("unexpected arguments %v; use -- to pass flags to go test", args)
		}

		return nil, nil
	}
	if dash > 0 {
		return nil, fmt.Errorf("unexpected arguments %v before --", args[:dash])
	}

	return args, nil
}
