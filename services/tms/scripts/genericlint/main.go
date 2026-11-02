package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
)

const (
	defaultAllowPath  = "scripts/genericlint/allow.txt"
	defaultMaxMethods = 2
	guideLink         = "docs/engineering/module-architecture.md#generics-and-build-time"
	exitViolation     = 1
)

var (
	errViolations = errors.New("exported API names generic types every importer compiles again")
	errUsage      = errors.New("invalid flags")
)

type options struct {
	maxMethods int
	allowPath  string
	writeAllow bool
}

func main() {
	var opts options
	flag.IntVar(
		&opts.maxMethods,
		"max-methods",
		defaultMaxMethods,
		"report an instantiated non-interface generic type with more methods than this",
	)
	flag.StringVar(&opts.allowPath, "allow", defaultAllowPath, "allowlist file")
	flag.BoolVar(
		&opts.writeAllow,
		"write-allow",
		false,
		"write every current finding into the allowlist, keeping the reasons already there",
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, opts)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "genericlint:", err)
		os.Exit(exitViolation)
	}
}

func run(ctx context.Context, opts options) error {
	if opts.maxMethods < 0 {
		return fmt.Errorf("%w: -max-methods must not be negative", errUsage)
	}
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("working directory: %w", err)
	}
	pkgs, err := load(ctx, dir)
	if err != nil {
		return err
	}
	findings := check(pkgs, opts.maxMethods, dir)

	allow, err := readAllow(opts.allowPath)
	if err != nil {
		return err
	}
	if opts.writeAllow {
		count, writeErr := allow.write(findings)
		if writeErr != nil {
			return writeErr
		}
		fmt.Printf("genericlint: wrote %d entries to %s\n", count, opts.allowPath)
		return nil
	}

	problems := allow.filter(findings)
	if len(problems) == 0 {
		return nil
	}
	for _, p := range problems {
		fmt.Println(p)
	}

	return fmt.Errorf("%w: %d (see %s)", errViolations, len(problems), guideLink)
}
