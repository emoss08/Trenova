package runner

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sort"
	"strings"
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
	Stdout    io.Writer
	Stderr    io.Writer
}

func Run(ctx context.Context, opts Options) (int, error) {
	worst := 0
	for _, group := range opts.Groups {
		if len(group.Packages) == 0 {
			continue
		}
		for chunk := range chunks(group.Packages, maxPackagesPerInvocation) {
			code, err := runChunk(ctx, opts, group.Run, chunk)
			if err != nil {
				return code, err
			}
			if code > worst {
				worst = code
			}
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

func runChunk(ctx context.Context, opts Options, run string, packages []string) (int, error) {
	args := []string{"test"}
	if len(opts.Tags) > 0 {
		args = append(args, "-tags", strings.Join(opts.Tags, ","))
	}
	if run != "" {
		args = append(args, "-run", run)
	}
	args = append(args, opts.ExtraArgs...)
	args = append(args, packages...)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = opts.Root
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}

		return 1, err
	}

	return 0, nil
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
