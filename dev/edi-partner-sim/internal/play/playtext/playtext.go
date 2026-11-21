package playtext

import (
	"cmp"
	"fmt"
	"io"
	"strings"

	"github.com/emoss08/trenova/edi-partner-sim/internal/play"
)

const textWidth = 60

func Summary(w io.Writer, r *play.Report) {
	fmt.Fprintf(w, "run %d against %s (simulator %s), %s\n", r.Run, r.API, r.Sim, r.Took)
	for _, p := range r.Partners {
		partner(w, p)
	}
	waiting(w, r)
	if r.Passed() {
		fmt.Fprintln(w, "\nPASS")
		return
	}
	fmt.Fprintf(w, "\nFAIL: %d failures\n", r.Failures)
}

func waiting(w io.Writer, r *play.Report) {
	waited := r.Waiting
	fmt.Fprintf(w, "\nwaited %s on %d expectations and %s on %d Trenova actions",
		waited.Expectations, waited.Expected, waited.Actions, waited.Acted)
	if waited.Failed > 0 {
		fmt.Fprintf(w, "; %s of it in steps that failed", waited.Failed)
	}
	if r.Scale != 0 && r.Scale != 1 {
		fmt.Fprintf(w, " (every deadline ×%g)", r.Scale)
	}
	fmt.Fprintln(w)
}

func partner(w io.Writer, p *play.PartnerReport) {
	passed, failed := 0, 0
	for _, s := range p.Scenarios {
		switch s.Status {
		case play.StatusPassed:
			passed++
		case play.StatusFailed:
			failed++
		}
	}
	verdict := "pass"
	if failed > 0 || len(p.Failures) > 0 {
		verdict = "FAIL"
	}
	fmt.Fprintf(w, "\n%s (%s): %s\n", p.Name, p.Role, verdict)
	if setup := p.Provision; setup != nil {
		fmt.Fprintf(w, "  provisioned %s over %s", setup.Code, cmp.Or(setup.Transport, "?"))
		if setup.VersionNumber > 0 {
			fmt.Fprintf(w, ", Lading version %d", setup.VersionNumber)
		}
		if setup.Unchanged {
			fmt.Fprint(w, " (unchanged)")
		}
		fmt.Fprintln(w)
		for _, test := range setup.FailingTests {
			fmt.Fprintf(w, "    failing test: %s\n", test)
		}
	}
	for _, f := range p.Failures {
		fmt.Fprintf(w, "  %s: %s\n", f.Source, f.Message)
	}
	if len(p.Scenarios) > 0 {
		fmt.Fprintf(w, "  scenarios: %d run, %d passed, %d failed\n",
			passed+failed, passed, failed)
	}
	for _, s := range p.Scenarios {
		Line(w, s)
	}
}

func Line(w io.Writer, s *play.ScenarioReport) {
	mark := "ok  "
	switch s.Status {
	case play.StatusFailed:
		mark = "FAIL"
	case play.StatusSkipped:
		mark = "skip"
	}
	fmt.Fprintf(w, "  %s %-40s %8s  seed %d\n", mark, s.Slug, s.Took, s.Seed)
	for _, f := range s.Failures {
		text := ""
		if f.Step >= 0 && f.Step < len(s.Steps) {
			text = s.Steps[f.Step].Text
		}
		fmt.Fprintf(w, "       step %d %s\n       %s %s: %s\n", f.Step+1, text, f.Source,
			f.Rule, f.Message)
	}
	if s.Status == play.StatusFailed {
		fmt.Fprintf(w, "       replay: %s\n", s.Replay)
	}
}

func Scenario(w io.Writer, s *play.ScenarioReport) {
	fmt.Fprintf(w, "\n== %s (%s), seed %d: %s\n", s.Name, s.Slug, s.Seed, s.Status)
	for _, step := range s.Steps {
		fmt.Fprintf(w, "  %2d %-7s %-60s %-7s %8s", step.Index+1, step.Kind,
			truncate(step.Text, textWidth), step.Status, step.Took)
		if step.Note != "" {
			fmt.Fprintf(w, "  %s", step.Note)
		}
		fmt.Fprintln(w)
	}
	for _, c := range s.Conversations {
		fmt.Fprintf(w, "  -- %s key %s\n", c.Name, cmp.Or(c.Key, "(none)"))
		for _, entry := range c.Wire {
			accepted := ""
			if entry.Accepted != nil && !*entry.Accepted {
				accepted = " REJECTED BY ORACLE"
			}
			fmt.Fprintf(w, "     %s %-8s %-8s %-10s %-9s ack %s %s%s\n",
				entry.At.Format("15:04:05.000"), entry.From, strings.Join(entry.Sets, ","),
				entry.Purpose, entry.State, cmp.Or(entry.Ack, "-"), entry.AckCode, accepted)
			for index := range entry.Findings {
				f := &entry.Findings[index]
				fmt.Fprintf(w, "        oracle %s %s %s: %s\n", f.Severity, f.SegmentID, f.Rule,
					f.Message)
			}
		}
	}
}

func truncate(text string, width int) string {
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	return string(runes[:width-1]) + "…"
}
