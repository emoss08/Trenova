package play

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/scenario"
	"github.com/emoss08/trenova/edi-partner-sim/internal/sim"
	"github.com/emoss08/trenova/edi-partner-sim/internal/simclient"
	"github.com/emoss08/trenova/edi-partner-sim/internal/trenova"
)

const (
	defaultTenderCents = 185000
	scenarioSeedStride = scenario.MaxSeedsPerCase
	MaxScenarios       = runStride / scenarioSeedStride
)

type stepError struct {
	source string
	rule   string
	err    error
}

func (s *stepError) Error() string {
	return s.err.Error()
}

func (s *stepError) Unwrap() error {
	return s.err
}

func failure(source, rule string, err error) error {
	return &stepError{source: source, rule: rule, err: err}
}

func failf(source, rule, format string, args ...any) error {
	return failure(source, rule, fmt.Errorf(format, args...))
}

type caseRun struct {
	env      *Env
	party    *Party
	scenario *scenario.Scenario
	base     uint64
	groups   map[string][]*Conversation
	order    []*Conversation
	report   *ScenarioReport
	started  time.Time
	trace    *Trace
}

func (e *Env) RunScenario(
	ctx context.Context,
	party *Party,
	s *scenario.Scenario,
	index int,
) *ScenarioReport {
	base := SeedOf(e.Run, uint64(index)*scenarioSeedStride)
	run := &caseRun{
		env:      e,
		party:    party,
		scenario: s,
		base:     base,
		groups:   map[string][]*Conversation{},
		started:  time.Now(),
		trace:    new(Trace),
		report: &ScenarioReport{
			Name:      s.Name,
			File:      s.File,
			Slug:      s.Slug,
			Seed:      base,
			Status:    StatusPassed,
			StartedAt: time.Now().UTC(),
			Steps:     []*StepReport{},
			Replay: fmt.Sprintf("edi-sim run --partner %s --scenario %s --run %d",
				party.Name(), s.Slug, e.Run),
		},
	}
	e.logf("%s: %s", party.Name(), s.Name)
	for index := range s.Steps {
		run.report.Steps = append(run.report.Steps, &StepReport{
			Index:  index,
			Kind:   s.Steps[index].Kind(),
			Text:   s.Steps[index].Describe(string(party.Def.Partner.Acknowledgment.Expects)),
			Status: runlog.StatusPending,
		})
	}
	run.post(ctx, runlog.StatusRunning)
	failed := false
	for index := range s.Steps {
		step := &s.Steps[index]
		report := run.report.Steps[index]
		report.At = Duration(time.Since(run.started))
		if failed {
			report.Status = StatusSkipped
			continue
		}
		began := time.Now()
		note, err := run.step(ctx, step)
		report.Took, report.Note = Duration(time.Since(began)), note
		if err == nil {
			report.Status = StatusPassed
			if e.verbose {
				e.logf("  ok   %s %s", report.Text, note)
			}
		} else {
			failed = true
			report.Status = StatusFailed
			run.fail(index, err)
			e.logf("  FAIL step %d %s: %v", index+1, report.Text, err)
		}
		run.record(ctx, step, report, err)
	}
	if failed {
		run.report.Status = StatusFailed
	}
	run.report.Took = Duration(time.Since(run.started))
	run.report.Conversations = run.conversations(ctx)
	for _, c := range run.order {
		e.fleet.park(c.shipment())
	}
	run.post(ctx, run.report.Status)
	return run.report
}

func (r *caseRun) fail(index int, err error) {
	f := Failure{Step: index, Source: SourceHarness, Rule: "step", Message: err.Error(),
		Replay: r.report.Replay}
	var stepErr *stepError
	if errors.As(err, &stepErr) {
		f.Source, f.Rule = stepErr.source, stepErr.rule
	}
	var problem *trenova.Problem
	if f.Source == SourceHarness && errors.As(err, &problem) {
		f.Source, f.Rule = SourceTrenova, fmt.Sprintf("%s %s", problem.Method, problem.Path)
	}
	if f.Source == SourceHarness && errors.Is(err, simclient.ErrSimulator) {
		f.Source = SourceSim
	}
	r.report.Failures = append(r.report.Failures, f)
}

func (r *caseRun) within(step time.Duration) time.Duration {
	return time.Duration(float64(r.scenario.WithinOf(step)) * r.env.scale)
}

func (r *caseRun) group(name string) []*Conversation {
	return r.groups[name]
}

func (r *caseRun) start(name string, seeds []uint64) []*Conversation {
	members := make([]*Conversation, 0, len(seeds))
	for index, seed := range seeds {
		member := name
		if len(seeds) > 1 {
			member = name + "#" + strconv.Itoa(index+1)
		}
		c := &Conversation{Name: member, Party: r.party, Seed: seed}
		c.Follow(r.trace)
		members = append(members, c)
		r.order = append(r.order, c)
	}
	r.groups[name] = members
	return members
}

func (r *caseRun) step(ctx context.Context, step *scenario.Step) (string, error) {
	switch step.Kind() {
	case scenario.KindSend:
		return r.send(ctx, step.Send)
	case scenario.KindExpect:
		return r.expect(ctx, step.Expect)
	case scenario.KindTrenova:
		return r.act(ctx, step.Trenova)
	case scenario.KindAdvance:
		return r.advance(ctx, step.Advance)
	case scenario.KindScript:
		return r.script(ctx, step.Script)
	case scenario.KindAcks:
		return r.acks(ctx, step.Acks)
	}
	return "", errors.New("the step does nothing")
}

func (r *caseRun) send(ctx context.Context, send *scenario.Send) (string, error) {
	if r.party.Carrier() {
		return "", failf(SourceHarness, "send", "%s is a carrier: Trenova tenders to it; "+
			"use trenova: { tender: … } and script", r.party.Name())
	}
	variation := partner.Variation{
		Kind:   partner.Kind(firstNonEmpty(send.Variation, string(partner.KindOriginal))),
		Sparse: send.Sparse,
		Delta: partner.Delta{
			RatePercent: send.Delta.RatePercent,
			AddStop:     send.Delta.AddStop,
		},
	}
	if send.Delta.Move != nil {
		variation.Delta.Move = &partner.Move{Stop: send.Delta.Move.Stop, By: send.Delta.Move.By}
	}
	var conversations []*Conversation
	if variation.Kind == partner.KindOriginal {
		count := max(send.Count, 1)
		seeds := make([]uint64, 0, count)
		for offset := range uint64(count) {
			seeds = append(seeds, r.base+send.Seed+offset)
		}
		conversations = r.start(send.As, seeds)
		for _, c := range conversations {
			c.Need = send.Load
		}
	} else {
		conversations = r.group(send.Of)
	}
	if len(conversations) > 0 {
		variation.Need = conversations[0].Need
	}
	messages, err := r.env.SendTender(ctx, r.party, conversations, variation)
	if err != nil {
		return "", failure(SourceSim, "send", err)
	}
	keys := make([]string, 0, len(conversations))
	for _, c := range conversations {
		keys = append(keys, c.Key)
	}
	return fmt.Sprintf("%d interchange(s), key %s", len(messages), summarize(keys)), nil
}

func (r *caseRun) act(ctx context.Context, action *scenario.Action) (string, error) {
	verb, about := action.Verb()
	within := r.within(0)
	if verb == "tender" {
		return r.tender(ctx, about, action, within)
	}
	var notes []string
	for _, c := range r.group(about) {
		note, err := r.actOn(ctx, verb, action, c, within)
		r.snapshot(ctx, c)
		if err != nil {
			return strings.Join(notes, "; "), err
		}
		if note != "" {
			notes = append(notes, note)
		}
	}
	return strings.Join(notes, "; "), nil
}

func (r *caseRun) actOn(
	ctx context.Context,
	verb string,
	action *scenario.Action,
	c *Conversation,
	within time.Duration,
) (string, error) {
	switch verb {
	case "decide":
		return "", r.env.Decide(ctx, c, action.Decision == "accept", action.Reason, within)
	case "dispatch":
		return "", r.env.Dispatch(ctx, c, within)
	case "arrive":
		return "", r.env.RecordStop(ctx, c, action.Stop, trenova.ActionArrive, within)
	case "depart":
		return "", r.env.RecordStop(ctx, c, action.Stop, trenova.ActionDepart, within)
	case "deliver":
		return "", r.env.Deliver(ctx, c, within)
	case "invoice":
		return "", r.env.Invoice(ctx, c, within)
	case "release":
		return "", r.env.Release(ctx, c, within)
	}
	return "", fmt.Errorf("unknown action %s", verb)
}

func (r *caseRun) tender(
	ctx context.Context,
	name string,
	action *scenario.Action,
	within time.Duration,
) (string, error) {
	if !r.party.Carrier() {
		return "", failf(SourceHarness, "tender", "%s tenders to Trenova; only a carrier "+
			"partner is tendered to", r.party.Name())
	}
	if r.party.Setup.CarrierID == "" {
		return "", failf(SourceSetup, "carrier", "no Trenova carrier is linked to %s",
			r.party.Name())
	}
	c := r.start(name, []uint64{r.base})[0]
	c.BOL = fmt.Sprintf("SIM%03d%02d%s", r.env.Run, r.base%runStride/scenarioSeedStride,
		strings.ToUpper(name))
	err := r.env.TenderOut(ctx, c, rateCents(action.Rate, defaultTenderCents), within)
	r.snapshot(ctx, c)
	if err != nil {
		return "", err
	}
	return "key " + c.Key, nil
}

func (r *caseRun) advance(ctx context.Context, advance *scenario.Advance) (string, error) {
	if err := r.env.CLI.Check(); err != nil {
		return "", failure(SourceHarness, "advance", err)
	}
	var notes []string
	for _, c := range r.group(advance.About) {
		report, err := r.env.Advance(ctx, c, advance.By, r.within(0))
		if err != nil {
			return strings.Join(notes, "; "), failure(SourceTrenova, "edi flows advance", err)
		}
		note := fmt.Sprintf("%s: %s", c.Name, report.Status)
		if report.Reason != "" {
			note += " (" + report.Reason + ")"
		}
		for _, sent := range report.Sent {
			note += " sent " + sent.TransactionSet
		}
		notes = append(notes, note)
	}
	r.trace.addGap("advanced " + scenario.ShortDuration(advance.By))
	return strings.Join(notes, "; "), nil
}

func (r *caseRun) script(ctx context.Context, script *scenario.Script) (string, error) {
	answers := make([]sim.ScriptAnswer, 0, len(script.Answers))
	for _, answer := range script.Answers {
		key := ""
		if answer.About != "" {
			for _, c := range r.group(answer.About) {
				key = c.Key
			}
		}
		answers = append(answers, sim.ScriptAnswer{
			Key:        key,
			Decline:    answer.Decline,
			Delay:      answer.Delay,
			DeltaCents: answer.DeltaCents,
			Misfooted:  answer.Misfooted,
			Stray:      answer.Stray,
			Silent:     answer.Silent,
		})
	}
	queued, err := r.env.Sim.Script(ctx, r.party.Name(), &sim.ScriptRequest{Answers: answers})
	if err != nil {
		return "", failure(SourceSim, "script", err)
	}
	return fmt.Sprintf("%d answers queued", len(queued)), nil
}

func (r *caseRun) acks(ctx context.Context, acks *scenario.Acks) (string, error) {
	var overrides []sim.AckOverride
	for _, c := range r.group(acks.About) {
		overrides = append(overrides, sim.AckOverride{
			Set:    acks.Set,
			Key:    c.Key,
			Action: acks.Action,
		})
	}
	queued, err := r.env.Sim.OverrideAcks(ctx, r.party.Name(),
		&sim.AckOverrideRequest{Overrides: overrides})
	if err != nil {
		return "", failure(SourceSim, "acks", err)
	}
	return fmt.Sprintf("%d overrides queued", len(queued)), nil
}

func (r *caseRun) snapshot(ctx context.Context, c *Conversation) {
	snap := Snapshot{Step: len(r.report.Steps) - 1, At: Duration(time.Since(r.started)).String()}
	if c.Key != "" {
		instances, err := r.env.Trenova.FlowInstances(ctx, r.party.Setup.PartnerID, c.Key)
		if err != nil {
			snap.Error = err.Error()
		} else if len(instances) > 0 {
			snap.FlowStatus, snap.FlowWhy = instances[0].Status, instances[0].Why()
		}
	}
	c.mu.Lock()
	shipmentID := c.shipmentID
	c.mu.Unlock()
	if shipmentID != "" {
		shipment, err := r.env.Trenova.Shipment(ctx, shipmentID)
		if err != nil {
			snap.Error = firstNonEmpty(snap.Error, err.Error())
		} else {
			snap.Shipment, snap.Status = shipment.ID, shipment.Status
			snap.Stops, snap.Total = len(shipment.Stops()), shipment.TotalChargeAmount
		}
	}
	c.mu.Lock()
	c.states = append(c.states, snap)
	c.mu.Unlock()
}

func (r *caseRun) conversations(ctx context.Context) []*ConversationReport {
	out := make([]*ConversationReport, 0, len(r.order))
	for _, c := range r.order {
		r.snapshot(ctx, c)
		report := &ConversationReport{Name: c.Name, Key: c.Key, Seed: c.Seed}
		c.mu.Lock()
		report.States = c.states
		c.mu.Unlock()
		if c.Key != "" {
			r.env.Watch.Read(func(view simclient.View) {
				messages := view.About(r.party.Name(), c.Key)
				report.Wire = wireLog(messages)
				report.Milestones = milestones(messages, c)
			})
		}
		out = append(out, report)
	}
	return out
}

func wireLog(messages []journal.Message) []WireEntry {
	out := make([]WireEntry, 0, len(messages))
	for index := range messages {
		m := &messages[index]
		entry := WireEntry{
			MessageID: m.ID,
			From:      "partner",
			Purpose:   string(m.Purpose),
			Sets:      m.Sets,
			State:     string(m.State),
			At:        m.At,
			Variation: m.Variation,
			Accepted:  m.Accepted,
			Error:     m.Error,
			X12:       m.Raw,
		}
		if m.Direction == journal.FromTrenova {
			entry.From = "trenova"
		}
		if m.Ack != nil {
			entry.Ack, entry.AckCode = string(m.Ack.Status), m.Ack.Code
		}
		if m.Judgement != nil {
			entry.Findings = append(entry.Findings, m.Judgement.Findings...)
			for _, ic := range m.Judgement.Interchanges {
				entry.Findings = append(entry.Findings, ic.Findings...)
				for _, g := range ic.Groups {
					entry.Findings = append(entry.Findings, g.Findings...)
					for _, rep := range g.Reports {
						entry.Findings = append(entry.Findings, rep.Findings...)
					}
				}
			}
		}
		out = append(out, entry)
	}
	return out
}

func milestones(messages []journal.Message, c *Conversation) Milestones {
	var out Milestones
	if c.StartedAt.IsZero() {
		return out
	}
	documents := journal.FromTrenova
	if c.Party.Carrier() {
		documents = journal.ToTrenova
	}
	first := map[string]time.Duration{}
	for index := range messages {
		m := &messages[index]
		since := m.At.Sub(c.StartedAt)
		for _, set := range m.Sets {
			name := set
			switch {
			case m.Direction == journal.FromTrenova && m.Purpose == journal.PurposeAck:
				name = "ack"
			case m.Direction != documents:
				continue
			}
			if _, seen := first[name]; !seen {
				first[name] = max(since, 0)
			}
		}
	}
	if d, ok := first["ack"]; ok {
		out.Ack = durationPtr(d)
	}
	if d, ok := first["990"]; ok {
		out.Answer = durationPtr(d)
	}
	if d, ok := first["214"]; ok {
		out.Status = durationPtr(d)
	}
	if d, ok := first["210"]; ok {
		out.Invoice = durationPtr(d)
	}
	return out
}

func summarize(keys []string) string {
	if len(keys) <= 3 {
		return strings.Join(keys, ", ")
	}
	return fmt.Sprintf("%s … %s (%d)", keys[0], keys[len(keys)-1], len(keys))
}
