package conform

import (
	"fmt"
	"slices"

	"github.com/emoss08/trenova/edi-partner-sim/internal/acknowledge"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

type acked struct {
	ack    *Ack
	notes  []acknowledge.SegmentNote
	group  *oracle.Group
	report *oracle.Report
}

func (r *runner) acknowledge(s *subject, c *Case, step *Step, j *oracle.Judgement) {
	rules := &s.def.Partner.Acknowledgment
	for _, ic := range j.Interchanges {
		for _, g := range ic.Groups {
			for _, rep := range g.Reports {
				decision := acknowledge.Decide(rules, ic, g, rep)
				step.Ack = &Ack{
					Set:  string(rules.Sends),
					Code: decision.Code,
					Expected: expectedAck(
						rules.Reject,
						rep.Set,
						decision.Errors,
						decision.Warnings,
					),
					Errors:   decision.Errors,
					Warnings: decision.Warnings,
					Notes:    ackNotesOf(decision.Notes),
				}
				checkAck(c, step, rules, acked{
					ack:    step.Ack,
					notes:  decision.Notes,
					group:  g,
					report: rep,
				})
			}
		}
	}
}

func checkAck(c *Case, step *Step, rules *partner.Acknowledgment, a acked) {
	if a.ack.Code != a.ack.Expected {
		c.fail(step, &Failure{
			Source: SourceAck,
			Rule:   "acknowledgment.reject",
			Message: fmt.Sprintf(
				"Decide gave %s for %d errors and %d warnings; partner.yaml's reject rules say %s",
				a.ack.Code, a.ack.Errors, a.ack.Warnings, a.ack.Expected),
		})
	}
	if a.ack.Code == acknowledge.Rejected && a.ack.Errors == 0 {
		c.fail(step, &Failure{
			Source:  SourceAck,
			Rule:    "acknowledgment.reject",
			X12Code: "AK501 R",
			Message: fmt.Sprintf(
				"the partner would reject %s %s with no error the oracle found",
				a.report.Set, a.report.ControlNumber),
		})
	}
	if rules.Sends == partner.AckNone {
		return
	}
	built, err := acknowledge.Build(rules, acknowledge.Input{
		GroupID:      a.group.FunctionalID,
		GroupControl: a.group.ControlNumber,
		Version:      a.group.Version,
		Transactions: []acknowledge.InputTransaction{{
			Set:     a.report.Set,
			Control: a.report.ControlNumber,
			Code:    a.ack.Code,
			Notes:   a.notes,
		}},
	})
	if err != nil {
		c.fail(step, &Failure{Source: SourceAck, Rule: "build", Message: err.Error()})
		return
	}
	if got := acknowledge.TransactionCode(built); got != a.ack.Code {
		c.fail(step, &Failure{
			Source: SourceAck,
			Rule:   "build",
			Message: fmt.Sprintf(
				"the built %s says %s, Decide said %s", built.Set, got, a.ack.Code),
		})
	}
}

func expectedAck(rules partner.RejectRules, set string, errs, warns int) string {
	if slices.Contains(rules.Sets, set) {
		return acknowledge.Rejected
	}
	switch rules.When {
	case partner.RejectNever:
		if errs+warns > 0 {
			return acknowledge.AcceptedWithErrors
		}
	case partner.RejectAny:
		if errs+warns > 0 {
			return acknowledge.Rejected
		}
	case partner.RejectErrors:
		if errs > 0 {
			return acknowledge.Rejected
		}
		if warns > 0 {
			return acknowledge.AcceptedWithErrors
		}
	default:
		if errs+warns > 0 {
			return acknowledge.AcceptedWithErrors
		}
	}
	return acknowledge.Accepted
}
