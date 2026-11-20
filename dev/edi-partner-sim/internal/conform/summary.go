package conform

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

const exampleSeeds = 3

type ruleTally struct {
	key     string
	failure Failure
	seeds   []uint64
}

func (r *Report) WriteSummary(w io.Writer) {
	last := r.Seed + r.Count - 1
	fmt.Fprintf(w, "conform: seeds %d..%d, %d workers\n", r.Seed, last, r.Workers)
	for _, p := range r.Partners {
		writePartner(w, p)
	}
	if r.Passed() {
		fmt.Fprintln(w, "PASS")
		return
	}
	fmt.Fprintf(w, "FAIL: %d failures\n", r.Failures)
}

func writePartner(w io.Writer, p *PartnerReport) {
	failed, held, notSent := 0, 0, 0
	for _, c := range p.Cases {
		if len(c.Failures) > 0 {
			failed++
		}
		if c.held() {
			held++
		}
		notSent += c.notSent()
	}
	verdict := "pass"
	if failed > 0 || len(p.Failures) > 0 {
		verdict = "FAIL"
	}
	fmt.Fprintf(w, "\n%s (%s): %s\n", p.Name, p.Role, verdict)
	fmt.Fprintf(w, "  project: %d errors, %d warnings, %d Lading tests, %d failing\n",
		p.Project.Errors, p.Project.Warnings, p.Project.Tests, len(p.Project.FailedTests))
	for i := range p.Failures {
		fmt.Fprintf(w, "    %s: %s\n", p.Failures[i].Rule, p.Failures[i].Message)
	}
	fmt.Fprintf(
		w,
		"  cases: %d run, %d passed, %d failed, %d with a held message, %d messages not sent\n",
		len(p.Cases), len(p.Cases)-failed, failed, held, notSent,
	)
	tallies := tallyRules(p.Cases)
	if len(tallies) == 0 {
		return
	}
	fmt.Fprintln(w, "  failing rules:")
	for _, t := range tallies {
		f := t.failure
		code := ""
		if f.X12Code != "" {
			code = " [" + f.X12Code + "]"
		}
		fmt.Fprintf(w, "    %-4s %s %s%s: %d seeds, e.g. %s (%s)\n",
			stringutils.FirstNonEmpty(f.Set, f.Step), f.Source, f.Rule, code, len(t.seeds),
			seedList(t.seeds), f.Step)
		fmt.Fprintf(w, "         %s\n", f.Message)
		fmt.Fprintf(w, "         replay: edi-sim conform --replay %s\n", f.Replay)
	}
}

func (c *Case) held() bool {
	return slices.ContainsFunc(c.Steps, func(s *Step) bool { return s.Status == StatusHeld })
}

func (c *Case) notSent() int {
	n := 0
	for _, s := range c.Steps {
		if s.Status == StatusNotSent {
			n++
		}
	}
	return n
}

func tallyRules(cases []*Case) []*ruleTally {
	byKey := map[string]*ruleTally{}
	var order []*ruleTally
	for _, c := range cases {
		seen := map[string]bool{}
		for i := range c.Failures {
			f := &c.Failures[i]
			key := stringutils.FirstNonEmpty(f.Set, f.Step) + "\x00" + f.Source + "\x00" + f.Rule
			if seen[key] {
				continue
			}
			seen[key] = true
			t := byKey[key]
			if t == nil {
				t = &ruleTally{key: key, failure: *f}
				byKey[key] = t
				order = append(order, t)
			}
			t.seeds = append(t.seeds, c.Seed)
		}
	}
	slices.SortStableFunc(order, func(a, b *ruleTally) int {
		return cmp.Compare(len(b.seeds), len(a.seeds))
	})
	return order
}

func seedList(seeds []uint64) string {
	parts := make([]string, 0, exampleSeeds)
	for _, seed := range seeds[:min(len(seeds), exampleSeeds)] {
		parts = append(parts, strconv.FormatUint(seed, 10))
	}
	return strings.Join(parts, ", ")
}

func (r *Report) WriteCases(w io.Writer) {
	for _, p := range r.Partners {
		for _, c := range p.Cases {
			c.Write(w)
		}
	}
}

func (c *Case) Write(w io.Writer) {
	fmt.Fprintf(w, "\n== %s seed %d: %s\n", c.Partner, c.Seed, c.Variation)
	for _, s := range c.Steps {
		fmt.Fprintf(w, "\n-- %s (%s %s", s.Name, s.Direction, s.Set)
		if s.Map != "" {
			fmt.Fprintf(w, " via %s", s.Map)
		}
		fmt.Fprintf(w, "): %s\n", s.Status)
		if s.Note != "" {
			fmt.Fprintf(w, "   %s\n", s.Note)
		}
		if len(s.Document) > 0 {
			fmt.Fprintf(w, "   document: %s\n", s.Document)
		}
		if s.X12 != "" {
			fmt.Fprintf(w, "%s\n", strings.TrimRight(s.X12, "\r\n"))
		}
		for _, f := range s.Lading {
			fmt.Fprintf(
				w,
				"   lading %s %s %s %s: %s\n",
				f.Kind,
				f.Severity,
				f.Code,
				f.Rule,
				f.Message,
			)
		}
		for i := range s.Oracle {
			f := &s.Oracle[i]
			fmt.Fprintf(w, "   oracle %s %s [%s] %s: %s\n",
				f.Severity, segmentOf(f), f.X12Code(), f.Rule, f.Message)
		}
		if s.Ack != nil {
			fmt.Fprintf(w, "   partner's %s: %s (rules say %s), %d errors, %d warnings\n",
				s.Ack.Set, s.Ack.Code, s.Ack.Expected, s.Ack.Errors, s.Ack.Warnings)
		}
	}
	for i := range c.Failures {
		f := &c.Failures[i]
		fmt.Fprintf(w, "FAIL %s: %s %s: %s\n", f.Step, f.Source, f.Rule, f.Message)
	}
}
