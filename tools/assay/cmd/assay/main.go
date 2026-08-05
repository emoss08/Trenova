package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/report"
	"github.com/emoss08/assay/internal/runner"
	"github.com/emoss08/assay/internal/selection"
	"github.com/emoss08/assay/internal/vcs"
)

const version = "0.1.0-m0"

const usage = `assay - test intelligence for Go

Usage:
  assay select [flags]              Show which test packages a change affects
  assay run    [flags] [-- args]    Run only the affected test packages
  assay version

Flags:
  --since ref     Diff against this ref's merge-base (default: uncommitted vs HEAD)
  --files path    Read changed paths from a file, or - for stdin (skips git)
  --root dir      Workspace root (default: git repository root)
  --tags list     Comma-separated build tags
  --all           Skip selection and use every testable package
  --json          Emit machine-readable output (select only)
  -v              Verbose: list every selected package and ignored file

Arguments after -- are passed through to go test.

Examples:
  assay select --since origin/main -v
  assay run --since origin/main -- -count=1 -race
  assay run --tags integration --since HEAD~1
`

type config struct {
	since    string
	root     string
	files    string
	tags     []string
	all      bool
	asJSON   bool
	verbose  bool
	testArgs []string
}

func main() {
	code, err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "assay: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func run(argv []string) (int, error) {
	if len(argv) == 0 {
		fmt.Fprint(os.Stderr, usage)

		return 2, nil
	}

	command := argv[0]
	rest := argv[1:]

	switch command {
	case "version", "--version", "-version":
		fmt.Println("assay " + version)

		return 0, nil
	case "help", "--help", "-h":
		fmt.Print(usage)

		return 0, nil
	case "select", "run":
	default:
		fmt.Fprint(os.Stderr, usage)

		return 2, fmt.Errorf("unknown command %q", command)
	}

	cfg, err := parseFlags(command, rest)
	if err != nil {
		return 2, err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return execute(ctx, command, cfg)
}

func parseFlags(command string, argv []string) (config, error) {
	var cfg config

	flagArgs := argv
	for i, arg := range argv {
		if arg == "--" {
			flagArgs = argv[:i]
			cfg.testArgs = argv[i+1:]

			break
		}
	}

	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var tags string
	fs.StringVar(&cfg.since, "since", "", "diff against this ref's merge-base")
	fs.StringVar(&cfg.root, "root", "", "workspace root")
	fs.StringVar(&cfg.files, "files", "", "read changed paths from a file, or - for stdin")
	fs.StringVar(&tags, "tags", "", "comma-separated build tags")
	fs.BoolVar(&cfg.all, "all", false, "select every testable package")
	fs.BoolVar(&cfg.asJSON, "json", false, "emit machine-readable output")
	fs.BoolVar(&cfg.verbose, "v", false, "verbose output")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	if err := fs.Parse(flagArgs); err != nil {
		return cfg, err
	}

	if tags != "" {
		for _, t := range strings.Split(tags, ",") {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				cfg.tags = append(cfg.tags, trimmed)
			}
		}
	}

	if extra := fs.Args(); len(extra) > 0 && len(cfg.testArgs) == 0 {
		return cfg, fmt.Errorf("unexpected arguments %v (use -- to pass flags to go test)", extra)
	}

	return cfg, nil
}

func execute(ctx context.Context, command string, cfg config) (int, error) {
	root, err := resolveRoot(ctx, cfg.root)
	if err != nil {
		return 1, err
	}

	g, err := graph.Load(ctx, graph.LoadOptions{Root: root, Tags: cfg.tags})
	if err != nil {
		return 1, err
	}

	result, err := computeSelection(ctx, g, root, cfg)
	if err != nil {
		return 1, err
	}

	summary := report.NewSummary(result, g.Len(), len(g.TestablePackages()))

	if command == "select" {
		if cfg.asJSON {
			return 0, report.WriteJSON(os.Stdout, summary)
		}

		return 0, report.WriteText(os.Stdout, summary, cfg.verbose)
	}

	if err := report.WriteText(os.Stderr, summary, cfg.verbose); err != nil {
		return 1, err
	}

	if len(result.Packages) == 0 {
		fmt.Fprintln(os.Stderr, "no affected test packages")

		return 0, nil
	}

	return runner.Run(ctx, runner.Options{
		Root:      root,
		Packages:  result.Packages,
		Tags:      cfg.tags,
		ExtraArgs: cfg.testArgs,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	})
}

func computeSelection(ctx context.Context, g *graph.Graph, root string, cfg config) (selection.Result, error) {
	if cfg.all {
		return selection.All(g, "--all requested"), nil
	}

	if cfg.files != "" {
		changes, err := readFileList(cfg.files, root)
		if err != nil {
			return selection.Result{}, err
		}

		return selection.Select(selection.Options{Graph: g, Changes: changes}), nil
	}

	changes, err := vcs.Changes(ctx, vcs.Options{
		Root:             root,
		Base:             cfg.since,
		IncludeUntracked: true,
	})
	if err != nil {
		return selection.Result{}, err
	}

	return selection.Select(selection.Options{Graph: g, Changes: changes}), nil
}

func readFileList(source, root string) ([]vcs.Change, error) {
	var raw []byte
	var err error
	if source == "-" {
		raw, err = io.ReadAll(os.Stdin)
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
		return explicit, nil
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
