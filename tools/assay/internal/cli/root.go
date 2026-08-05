package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/report"
	"github.com/emoss08/assay/internal/selection"
	"github.com/emoss08/assay/internal/vcs"
)

const Version = "0.1.0-m0"

type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

type options struct {
	since   string
	files   string
	root    string
	tags    []string
	all     bool
	asJSON  bool
	verbose bool
	noColor bool
}

func (o *options) useColor() bool {
	return !o.noColor && !color.NoColor
}

func NewRootCommand() *cobra.Command {
	opts := &options{}

	root := &cobra.Command{
		Use:   "assay",
		Short: "Test intelligence for Go",
		Long: "assay runs the tests a change can actually affect, instead of all of them.\n\n" +
			"It builds the workspace package graph, attributes changed files to packages, and\n" +
			"walks reverse dependency edges to find every test package a diff can reach.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	flags := root.PersistentFlags()
	flags.StringVar(&opts.since, "since", "", "diff against this ref's merge-base with HEAD")
	flags.StringVar(&opts.files, "files", "", "read changed paths from a file, or - for stdin")
	flags.StringVar(&opts.root, "root", "", "workspace root (default: git repository root)")
	flags.StringSliceVar(&opts.tags, "tags", nil, "build tags")
	flags.BoolVar(&opts.all, "all", false, "skip selection and use every testable package")
	flags.BoolVarP(&opts.verbose, "verbose", "v", false, "list every selected package and ignored file")
	flags.BoolVar(&opts.noColor, "no-color", false, "disable colored output")

	root.AddCommand(newSelectCommand(opts), newRunCommand(opts), newVersionCommand())

	return root
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the assay version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Println("assay " + Version)
		},
	}
}

func (o *options) summarize(ctx context.Context) (report.Summary, selection.Result, error) {
	root, err := resolveRoot(ctx, o.root)
	if err != nil {
		return report.Summary{}, selection.Result{}, err
	}

	g, err := graph.Load(ctx, graph.LoadOptions{Root: root, Tags: o.tags})
	if err != nil {
		return report.Summary{}, selection.Result{}, err
	}

	result, err := o.selectPackages(ctx, g, root)
	if err != nil {
		return report.Summary{}, selection.Result{}, err
	}

	return report.NewSummary(result, g.Len(), len(g.TestablePackages())), result, nil
}

func (o *options) selectPackages(ctx context.Context, g *graph.Graph, root string) (selection.Result, error) {
	if o.all {
		return selection.All(g, "--all requested"), nil
	}

	if o.files != "" {
		changes, err := readFileList(o.files, root, os.Stdin)
		if err != nil {
			return selection.Result{}, err
		}

		return selection.Select(selection.Options{Graph: g, Changes: changes}), nil
	}

	changes, err := vcs.Changes(ctx, vcs.Options{
		Root:             root,
		Base:             o.since,
		IncludeUntracked: true,
	})
	if err != nil {
		return selection.Result{}, err
	}

	return selection.Select(selection.Options{Graph: g, Changes: changes}), nil
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
