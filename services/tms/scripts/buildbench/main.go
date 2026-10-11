package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var errUsage = errors.New(
	"usage: buildbench <probes|analyze|report|fanin|deps-of> [flags]",
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:])
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "buildbench:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errUsage
	}

	switch args[0] {
	case "probes":
		return runProbes(ctx, args[1:])
	case "analyze":
		return runAnalyze(args[1:])
	case "report":
		return runReport(args[1:])
	case "fanin":
		return runFanin(ctx, args[1:])
	case "deps-of":
		return runDepsOf(ctx, args[1:])
	default:
		return fmt.Errorf("%w: unknown command %q", errUsage, args[0])
	}
}
