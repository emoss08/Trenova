package conform

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	directionInbound  = "inbound"
	directionOutbound = "outbound"
	maxLookupRounds   = 3
	ruleRun           = "lading run"
)

type answerFunc func(need lading.Need) *lading.Found

type namedDocument struct {
	name     string
	text     string
	set      string
	document any
	now      time.Time
}

type message struct {
	text   string
	now    time.Time
	answer answerFunc
	want   string
	flows  bool
	flow   *flowInstance
}

func instant(at time.Time) string {
	return at.UTC().Format(time.RFC3339)
}

func (r *runner) read(
	ctx context.Context,
	s *subject,
	c *Case,
	step *Step,
	m message,
) *lading.TransactionResult {
	step.X12 = m.text
	step.Map = s.inboundMap(step.Set)
	if step.Map == "" {
		c.fail(step, &Failure{
			Source:  SourceLading,
			Rule:    "receive " + step.Set,
			Message: fmt.Sprintf("the project has no map that reads a %s", step.Set),
		})
		return nil
	}
	request := &lading.RunRequest{
		Map:   step.Map,
		Input: lading.X12Input(m.text),
		Now:   instant(m.now),
		Rules: true,
	}
	result, err := r.resolve(ctx, s, step, request, m.answer)
	if err != nil {
		c.fail(step, &Failure{Source: SourceLading, Rule: ruleRun, Message: err.Error()})
		return nil
	}
	if result.Error != nil {
		c.fail(step, &Failure{
			Source:  SourceLading,
			Rule:    ruleRun,
			Message: result.Error.String(),
		})
		return nil
	}
	if len(result.Transactions) != 1 {
		c.fail(step, &Failure{
			Source:  SourceLading,
			Rule:    "one transaction",
			Message: fmt.Sprintf("read %d transactions, want 1", len(result.Transactions)),
		})
		return nil
	}
	transaction := &result.Transactions[0]
	step.Document = transaction.Document
	step.Status = StatusRead
	for i := range transaction.Findings {
		f := &transaction.Findings[i]
		step.Lading = append(step.Lading, LadingFinding{
			Kind:     "finding",
			Severity: string(f.Severity),
			Code:     f.Code,
			Path:     f.SegmentID,
			Message:  f.Message.String(),
		})
		if f.IsError() {
			c.fail(step, &Failure{
				Source:  SourceLading,
				Rule:    "x12 " + f.Code,
				Segment: f.SegmentID,
				Message: f.Message.String(),
			})
		}
	}
	if m.flows {
		r.received(ctx, s, c, step, m.flow, transaction, m.now)
	}
	if m.want == WantRejected {
		expectRejected(c, step, transaction)
		return transaction
	}
	r.outcomes(c, step, transaction.Outcomes)
	if transaction.Rejected() && step.Status != StatusFailed {
		c.fail(step, &Failure{
			Source:  SourceLading,
			Rule:    "transaction status",
			Message: "the transaction was rejected",
		})
	}
	expectOutcome(c, step, m.want)
	return transaction
}

func expectRejected(c *Case, step *Step, transaction *lading.TransactionResult) {
	for _, o := range transaction.Outcomes {
		step.Lading = append(step.Lading, LadingFinding{
			Kind:    "outcome " + o.Kind,
			Code:    o.Code,
			Rule:    o.Rule,
			Path:    o.Path,
			Message: o.Message,
		})
	}
	if transaction.Rejected() {
		step.Status = StatusRejected
		return
	}
	c.fail(step, &Failure{
		Source:  SourceLading,
		Rule:    "want rejected",
		Message: "the partner sent this to be rejected, and the project took it",
	})
}

func expectOutcome(c *Case, step *Step, want string) {
	switch {
	case step.Status == StatusFailed:
	case want == WantHeld && step.Status != StatusHeld:
		c.fail(step, &Failure{
			Source:  SourceLading,
			Rule:    "want held",
			Message: "the partner sent this to be held for a person, and the project took it",
		})
	case want == WantApplied && step.Status != StatusRead:
		c.fail(step, &Failure{
			Source: SourceLading,
			Rule:   "want applied",
			Message: fmt.Sprintf("the partner sent this to be applied, and the project %s it",
				step.Status),
		})
	}
}

func (r *runner) resolve(
	ctx context.Context,
	s *subject,
	step *Step,
	request *lading.RunRequest,
	answer answerFunc,
) (*lading.RunResult, error) {
	answered := map[lading.Need]bool{}
	for round := 0; ; round++ {
		result, err := r.client.Run(ctx, s.project, request)
		if err != nil {
			return nil, err
		}
		fresh := 0
		for _, need := range result.Needs {
			if answered[need] {
				continue
			}
			answered[need] = true
			fresh++
			found := answer(need)
			if found == nil {
				step.Lading = append(step.Lading, LadingFinding{
					Kind:     "lookup",
					Severity: lading.SeverityWarning,
					Code:     need.Kind,
					Message: fmt.Sprintf(
						"the map looked up %s %q, which the document does not carry",
						need.Kind, need.Code),
				})
			}
			request.Lookups = append(request.Lookups, lading.LookupAnswer{
				Kind:  need.Kind,
				Code:  need.Code,
				Found: found,
			})
		}
		if fresh == 0 || round == maxLookupRounds {
			return result, nil
		}
	}
}

func (r *runner) outcomes(c *Case, step *Step, outcomes []lading.Outcome) {
	for _, o := range outcomes {
		step.Lading = append(step.Lading, LadingFinding{
			Kind:    "outcome " + o.Kind,
			Code:    o.Code,
			Rule:    o.Rule,
			Path:    o.Path,
			Message: o.Message,
		})
		switch o.Kind {
		case lading.OutcomeReject:
			c.fail(step, &Failure{
				Source:  SourceLading,
				Rule:    "rule " + o.Rule,
				Message: "rejected: " + o.Message,
			})
		case lading.OutcomeHold:
			if step.Status != StatusFailed {
				step.Status = StatusHeld
			}
		case lading.OutcomeDecline:
			if step.Status != StatusFailed && step.Status != StatusHeld {
				step.Status = StatusDeclined
			}
		}
	}
}

func (r *runner) write(
	ctx context.Context,
	s *subject,
	c *Case,
	doc namedDocument,
	control lading.Control,
) *Step {
	step := c.step(doc.name, doc.text, directionOutbound, doc.set)
	step.Map = s.outboundMap(doc.set)
	if step.Map == "" {
		c.fail(step, &Failure{
			Source:  SourceLading,
			Rule:    "send " + doc.set,
			Message: fmt.Sprintf("the project has no map that writes a %s", doc.set),
		})
		return step
	}
	encoded, err := sonic.Marshal(doc.document)
	if err != nil {
		c.fail(step, &Failure{Source: SourceHarness, Rule: "document", Message: err.Error()})
		return step
	}
	step.Document = encoded
	result, err := r.resolve(ctx, s, step, &lading.RunRequest{
		Map:     step.Map,
		Input:   lading.DocumentInput(encoded),
		Now:     instant(doc.now),
		Control: control,
		Rules:   true,
	}, func(lading.Need) *lading.Found { return nil })
	switch {
	case err != nil:
		c.fail(step, &Failure{Source: SourceLading, Rule: ruleRun, Message: err.Error()})
		return step
	case result.Error != nil:
		c.fail(step, &Failure{
			Source:  SourceLading,
			Rule:    ruleRun,
			Message: result.Error.String(),
		})
		return step
	case result.NotSent:
		step.Status = StatusNotSent
		step.Note = "the map declares this not sent: nothing goes out and no acknowledgment is awaited"
		return step
	case result.Output == nil:
		c.fail(step, &Failure{
			Source:  SourceLading,
			Rule:    "map " + step.Map,
			Message: "the map wrote no message",
		})
		return step
	}
	output := result.Output
	step.X12 = output.Text
	step.Status = StatusSent
	r.outcomes(c, step, output.Outcomes)
	held := step.Status == StatusHeld || step.Status == StatusDeclined
	r.writeFindings(c, step, output, held)
	if held {
		step.Note = "Trenova would not send it: a rule held it"
		return step
	}
	r.judge(s, c, step)
	return step
}

func (r *runner) writeFindings(c *Case, step *Step, output *lading.WriteResult, held bool) {
	for i := range output.Findings {
		f := &output.Findings[i]
		step.Lading = append(step.Lading, LadingFinding{
			Kind:     "finding",
			Severity: f.Severity,
			Code:     f.Code,
			Path:     f.Path,
			Message:  f.Message.String(),
		})
		if f.Severity == lading.SeverityError && !held {
			c.fail(step, &Failure{
				Source:  SourceLading,
				Rule:    "write " + f.Code,
				Message: f.Message.String(),
			})
		}
	}
	for i := range output.Validation {
		f := &output.Validation[i]
		step.Lading = append(step.Lading, LadingFinding{
			Kind:     "validation",
			Severity: string(f.Severity),
			Code:     f.Code,
			Path:     f.ElementRef,
			Message:  f.Message.String(),
		})
		if f.IsError() && !held {
			ref := stringutils.FirstNonEmpty(f.ElementRef, f.SegmentID, f.Code)
			c.fail(step, &Failure{
				Source:  SourceLading,
				Rule:    "crate guide " + ref,
				Segment: f.SegmentID,
				Message: "Trenova would not send it: " + f.Message.String(),
			})
		}
	}
}

func (r *runner) judge(s *subject, c *Case, step *Step) {
	j, err := s.guides.Judge([]byte(step.X12))
	if err != nil {
		c.fail(step, &Failure{Source: SourceOracle, Rule: "read", Message: err.Error()})
		return
	}
	findings := judgementFindings(j)
	for i := range findings {
		f := &findings[i]
		step.Oracle = append(step.Oracle, *f)
		if f.Severity != oracle.SeverityError {
			continue
		}
		c.fail(step, &Failure{
			Source:  SourceOracle,
			Rule:    f.Rule,
			X12Code: f.X12Code(),
			Segment: segmentOf(f),
			Message: f.Message,
		})
	}
	r.envelope(s, c, step, j)
	r.acknowledge(s, c, step, j)
}

func judgementFindings(j *oracle.Judgement) []oracle.Finding {
	out := append([]oracle.Finding(nil), j.Findings...)
	for _, ic := range j.Interchanges {
		out = append(out, ic.Findings...)
		for _, g := range ic.Groups {
			out = append(out, g.Findings...)
			for _, rep := range g.Reports {
				out = append(out, rep.Findings...)
			}
		}
	}
	return out
}

func (r *runner) envelope(s *subject, c *Case, step *Step, j *oracle.Judgement) {
	p := &s.def.Partner
	check := func(rule, got, want string) {
		if got != want {
			c.fail(step, &Failure{
				Source:  SourceEnvelope,
				Rule:    rule,
				Message: fmt.Sprintf("is %q, partner.yaml says %q", got, want),
			})
		}
	}
	if len(j.Interchanges) != 1 {
		c.fail(step, &Failure{
			Source:  SourceEnvelope,
			Rule:    "one interchange",
			Message: fmt.Sprintf("Trenova wrote %d interchanges", len(j.Interchanges)),
		})
		return
	}
	ic := j.Interchanges[0]
	check("ISA05 trenova.qualifier", ic.SenderQualifier, p.Trenova.Qualifier)
	check("ISA06 trenova.id", ic.SenderID, p.Trenova.ID)
	check("ISA07 them.qualifier", ic.ReceiverQualifier, p.Them.Qualifier)
	check("ISA08 them.id", ic.ReceiverID, p.Them.ID)
	for _, g := range ic.Groups {
		check("GS02 trenova.group", g.SenderCode, p.Trenova.Group)
		check("GS03 them.group", g.ReceiverCode, p.Them.Group)
		check("GS08 envelope.version", g.Version, p.Envelope.Version)
	}
}
