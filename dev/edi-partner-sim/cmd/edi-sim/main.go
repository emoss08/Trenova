package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"syscall"
)

const (
	exitOK      = 0
	exitFailed  = 1
	exitUsage   = 2
	programName = "edi-sim"
)

type command struct {
	name    string
	summary string
	run     func(ctx context.Context, args []string, stdout, stderr io.Writer) int
}

var commands = []command{
	{name: "serve", summary: serveSummary, run: runServe},
	{name: "conform", summary: conformSummary, run: runConform},
	{name: "run", summary: runSummary, run: runRun},
	{name: "soak", summary: soakSummary, run: runSoak},
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := dispatch(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func dispatch(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		usage(stderr)
		if len(args) == 0 {
			return exitUsage
		}
		return exitOK
	}
	index := slices.IndexFunc(commands, func(c command) bool { return c.name == args[0] })
	if index < 0 {
		fmt.Fprintf(stderr, "%s: unknown command %q\n\n", programName, args[0])
		usage(stderr)
		return exitUsage
	}
	return commands[index].run(ctx, args[1:], stdout, stderr)
}

func usage(w io.Writer) {
	fmt.Fprintf(w, "usage: %s <command> [flags]\n\ncommands:\n", programName)
	for _, c := range commands {
		fmt.Fprintf(w, "  %-10s %s\n", c.name, c.summary)
	}
	fmt.Fprintf(w, "\nrun %s <command> -h for its flags\n", programName)
}
