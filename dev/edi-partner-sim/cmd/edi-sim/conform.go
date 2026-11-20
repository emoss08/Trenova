package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/edi-partner-sim/internal/conform"
)

const conformSummary = "prove every partner's Lading project against its guides, no Trenova needed"

type conformFlags struct {
	options conform.Options
	report  string
	replay  string
	verbose bool
}

func runConform(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	set := flag.NewFlagSet(programName+" conform", flag.ContinueOnError)
	set.SetOutput(stderr)
	var f conformFlags
	var partner string
	set.StringVar(&f.options.Root, "partners", "partners", "the folder holding the partners")
	set.StringVar(&partner, "partner", "", "run only these partners (comma separated)")
	set.Uint64Var(&f.options.Seed, "seed", 1, "the first seed")
	set.Uint64Var(&f.options.Count, "count", conform.DefaultCount, "how many seeds per partner")
	set.IntVar(&f.options.Workers, "workers", conform.DefaultWorkers,
		"cases run at once; also the Lading runtime's pool size")
	set.StringVar(
		&f.report,
		"report",
		"report.json",
		"where to write the JSON report (empty: none)",
	)
	set.StringVar(&f.replay, "replay", "", "rerun one case verbosely: <partner>:<seed>")
	set.BoolVar(&f.verbose, "v", false, "print every case's steps, wire and findings")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if set.NArg() > 0 {
		fmt.Fprintf(stderr, "conform: unexpected argument %q\n", set.Arg(0))
		return exitUsage
	}
	if partner != "" {
		f.options.Partners = strings.Split(partner, ",")
	}
	if f.replay != "" {
		name, seed, err := parseReplay(f.replay)
		if err != nil {
			fmt.Fprintf(stderr, "conform: %v\n", err)
			return exitUsage
		}
		f.options.Partners, f.options.Seed, f.options.Count = []string{name}, seed, 1
		f.verbose = true
	}
	report, err := conform.Run(ctx, &f.options)
	if err != nil {
		fmt.Fprintf(stderr, "conform: %v\n", err)
		return exitFailed
	}
	if f.verbose {
		report.WriteCases(stdout)
		fmt.Fprintln(stdout)
	}
	report.WriteSummary(stdout)
	if f.report != "" {
		if err := report.WriteJSON(f.report); err != nil {
			fmt.Fprintf(stderr, "conform: %v\n", err)
			return exitFailed
		}
		fmt.Fprintf(stdout, "report: %s\n", f.report)
	}
	if !report.Passed() {
		return exitFailed
	}
	return exitOK
}

func parseReplay(text string) (string, uint64, error) {
	name, seedText, ok := strings.Cut(text, ":")
	if !ok || name == "" {
		return "", 0, fmt.Errorf("--replay %q is not <partner>:<seed>", text)
	}
	seed, err := strconv.ParseUint(seedText, 10, 64)
	if err != nil || seed == 0 {
		return "", 0, fmt.Errorf("--replay %q: %q is not a seed", text, seedText)
	}
	return name, seed, nil
}
