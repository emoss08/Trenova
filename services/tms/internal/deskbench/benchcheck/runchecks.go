package benchcheck

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/deskbench/benchresult"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/deskbench/benchsql"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
)

type RunOutcome struct {
	ID       pulid.ID
	Subject  pulid.ID
	Status   agent.RunStatus
	Flagged  int
	Stopping bool
	Stopped  bool
}

func (o *RunOutcome) substitute(statement string) string {
	if o == nil {
		return statement
	}
	statement = strings.ReplaceAll(statement, benchscenario.PlaceholderRun, benchsql.Quote(o.ID))

	return strings.ReplaceAll(
		statement,
		benchscenario.PlaceholderSubject,
		benchsql.Quote(o.Subject),
	)
}

func RunChecks(expect *benchscenario.Expect, outcome *RunOutcome) []benchresult.CheckResult {
	checks := make([]benchresult.CheckResult, 0, 3)
	if outcome == nil {
		return checks
	}
	if outcome.Stopping {
		detail := "stopped while " + string(agent.RunStatusAwaitingDecision)
		if !outcome.Stopped {
			detail = "it was " + string(outcome.Status) + ", not waiting for a decision"
		}
		checks = append(checks, benchresult.CheckResult{
			Name:   "the run was stopped while it waited for a decision",
			Passed: outcome.Stopped,
			Detail: detail,
		})
	}
	if expect.Run != "" {
		checks = append(checks, benchresult.CheckResult{
			Name:   "the run ended " + expect.Run,
			Passed: strings.EqualFold(string(outcome.Status), expect.Run),
			Detail: "ended " + string(outcome.Status),
		})
	}
	if expect.Flagged != nil {
		name := "the run flagged it for a person"
		if !*expect.Flagged {
			name = "the run flagged nothing"
		}
		checks = append(checks, benchresult.CheckResult{
			Name:   name,
			Passed: (outcome.Flagged > 0) == *expect.Flagged,
			Detail: fmt.Sprintf("%d flagged", outcome.Flagged),
		})
	}

	return checks
}
