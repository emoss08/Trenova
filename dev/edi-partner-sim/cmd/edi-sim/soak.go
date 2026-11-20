package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/emoss08/trenova/edi-partner-sim/internal/soak"
)

const soakSummary = "hold many seeded happy-path conversations at once and measure them"

func runSoak(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	set := flag.NewFlagSet(programName+" soak", flag.ContinueOnError)
	set.SetOutput(stderr)
	var f tradingFlags
	var o soak.Options
	f.register(set)
	set.IntVar(&o.Conversations, "conversations", soak.DefaultConversations,
		"conversations in all, spread over the partners")
	set.Float64Var(&o.Rate, "rate", soak.DefaultRate, "conversations started per second")
	set.IntVar(&o.Concurrency, "concurrency", soak.DefaultConcurrency,
		"conversations held at once")
	set.DurationVar(&o.Within, "within", soak.DefaultWithin,
		"how long a conversation may take to reach its 210")
	set.DurationVar(&o.Settle, "settle", soak.DefaultSettle,
		"how long to wait for every flow to close at the end")
	set.StringVar(&f.report, "report", "soak-report.json", "where to write the JSON report")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if set.NArg() > 0 {
		fmt.Fprintf(stderr, "soak: unexpected argument %q\n", set.Arg(0))
		return exitUsage
	}
	if err := f.prepare(stdout); err != nil {
		fmt.Fprintf(stderr, "soak: %v\n", err)
		return exitUsage
	}
	o.Play = f.options
	report, err := soak.Run(ctx, &o)
	if report == nil {
		fmt.Fprintf(stderr, "soak: %v\n", err)
		return exitFailed
	}
	report.WriteSummary(stdout)
	if f.report != "" {
		if writeErr := report.WriteJSON(f.report); writeErr != nil {
			fmt.Fprintf(stderr, "soak: %v\n", writeErr)
			return exitFailed
		}
		fmt.Fprintf(stdout, "report: %s\n", f.report)
	}
	if err != nil || !report.Passed() {
		if err != nil {
			fmt.Fprintf(stderr, "soak: %v\n", err)
		}
		return exitFailed
	}
	return exitOK
}
