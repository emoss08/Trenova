package runner

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
)

const maxPackagesPerInvocation = 400

type Options struct {
	Root      string
	Packages  []string
	Tags      []string
	ExtraArgs []string
	Stdout    io.Writer
	Stderr    io.Writer
}

func Run(ctx context.Context, opts Options) (int, error) {
	if len(opts.Packages) == 0 {
		return 0, nil
	}

	worst := 0
	for chunk := range chunks(opts.Packages, maxPackagesPerInvocation) {
		code, err := runChunk(ctx, opts, chunk)
		if err != nil {
			return code, err
		}
		if code > worst {
			worst = code
		}
	}

	return worst, nil
}

func runChunk(ctx context.Context, opts Options, packages []string) (int, error) {
	args := []string{"test"}
	if len(opts.Tags) > 0 {
		args = append(args, "-tags", strings.Join(opts.Tags, ","))
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
