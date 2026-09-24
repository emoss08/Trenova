package agentredteam

import (
	"context"
	"slices"
	"strings"
	"testing"
)

type AssertMode int

const (
	AssertScripted AssertMode = iota
	AssertRecorded
)

func Assert(ctx context.Context, t *testing.T, outcome *Outcome, mode AssertMode) Verdict {
	t.Helper()

	verdict := Check(ctx, outcome)
	if mode == AssertScripted {
		for _, problem := range verdict.Problems {
			t.Errorf("the case no longer exercises what it claims: %s", problem)
		}
	}

	gap := outcome.Case.KnownGap
	for _, invariant := range Invariants() {
		violations := verdict.Of(invariant)
		declared := gap != nil && slices.Contains(gap.Invariants, invariant)
		t.Run(invariant, func(t *testing.T) {
			switch {
			case len(violations) > 0 && declared:
				t.Skipf("KNOWN GAP %s: %s\n%s", gap.ID, strings.TrimSpace(gap.Reason),
					describe(violations))
			case len(violations) > 0:
				t.Errorf("%s does not hold:\n%s", invariant, describe(violations))
			case declared && mode == AssertScripted:
				t.Errorf("known gap %s no longer reproduces for %s: the gap is closed, so "+
					"remove %s from knownGap in %s", gap.ID, invariant, invariant,
					outcome.Case.Path)
			}
		})
	}

	return verdict
}

func (v Verdict) Undeclared(gap *KnownGap) []string {
	broken := v.Broken()
	if gap == nil {
		return broken
	}

	return slices.DeleteFunc(broken, func(invariant string) bool {
		return slices.Contains(gap.Invariants, invariant)
	})
}

func describe(violations []Violation) string {
	lines := make([]string, 0, len(violations))
	for _, violation := range violations {
		lines = append(lines, "  - "+violation.Detail)
	}

	return strings.Join(lines, "\n")
}

func (o *Outcome) Attempted(tools []string) []string {
	attempted := make([]string, 0, len(tools))
	if o.Result == nil {
		return attempted
	}
	for idx := range o.Result.Messages {
		for _, call := range o.Result.Messages[idx].ToolCalls {
			if slices.Contains(tools, call.Name) && !slices.Contains(attempted, call.Name) {
				attempted = append(attempted, call.Name)
			}
		}
	}

	return attempted
}
