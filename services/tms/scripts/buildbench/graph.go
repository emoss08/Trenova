package main

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"text/tabwriter"
)

var errNoRoot = errors.New("target has no main package")

func runFanin(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("fanin", flag.ContinueOnError)
	tags := fs.String("tags", "", "build tags")
	top := fs.Int("top", 30, "packages to print")
	if err := fs.Parse(args); err != nil {
		return err
	}

	l, err := goList(ctx, listRequest{tags: *tags, patterns: []string{"./..."}})
	if err != nil {
		return err
	}

	type ranked struct {
		path  string
		count int
	}
	counts := l.dependentCounts()
	rows := make([]ranked, 0, len(counts))
	for p, n := range counts {
		rows = append(rows, ranked{path: p, count: n})
	}
	slices.SortFunc(rows, func(a, b ranked) int {
		if c := cmp.Compare(b.count, a.count); c != 0 {
			return c
		}
		return strings.Compare(a.path, b.path)
	})

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintf(tw, "Rank\tDependents\t\tof %d first-party packages\n", len(counts))
	for i, r := range rows[:min(*top, len(rows))] {
		fmt.Fprintf(tw, "%d\t%d\t\t%s\n", i+1, r.count, l.trim(r.path))
	}

	if err = tw.Flush(); err != nil {
		return fmt.Errorf("write table: %w", err)
	}

	return nil
}

func runDepsOf(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("deps-of", flag.ContinueOnError)
	tags := fs.String("tags", "", "build tags")
	target := fs.String("target", "", "main package pattern, for example ./cmd/api")
	pattern := fs.String("pattern", "", "regular expression over first-party import paths")
	if err := fs.Parse(args); err != nil {
		return err
	}

	re, err := regexp.Compile(*pattern)
	if err != nil {
		return fmt.Errorf("compile pattern: %w", err)
	}

	l, err := goList(ctx, listRequest{tags: *tags, deps: true, patterns: []string{*target}})
	if err != nil {
		return err
	}
	roots := l.roots()
	if len(roots) != 1 {
		return fmt.Errorf("%w: %s resolves to %v", errNoRoot, *target, roots)
	}

	matched := 0
	for _, p := range l.firstPartyPaths() {
		if p == roots[0] || !re.MatchString(p) {
			continue
		}
		matched++
		chain := l.shortestChain(roots[0], p)
		trimmed := make([]string, len(chain))
		for i, c := range chain {
			trimmed[i] = l.trim(c)
		}
		fmt.Printf("%s\n    %s\n", l.trim(p), strings.Join(trimmed, " -> "))
	}

	fmt.Fprintf(
		os.Stderr,
		"%d first-party packages linked by %s match %q\n",
		matched,
		*target,
		*pattern,
	)

	return nil
}
