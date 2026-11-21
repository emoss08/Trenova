package play

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/acknowledge"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/poll"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/scenario"
	"github.com/emoss08/trenova/edi-partner-sim/internal/simclient"
)

const invoiceTotal = "B3-07"

var ackSets = []string{"997", "999", "TA1"}

func (r *caseRun) expect(ctx context.Context, e *scenario.Expect) (string, error) {
	kind, about := e.Subject()
	within := r.within(e.Within)
	var notes []string
	for _, c := range r.group(about) {
		var note string
		var err error
		switch kind {
		case "wire":
			note, err = r.env.expectWire(ctx, c, e, within)
		case "ack":
			note, err = r.env.expectAck(ctx, c, e, within)
		case "flow":
			note, err = r.env.expectFlow(ctx, c, e, within)
		case "reconciled":
			note, err = r.env.expectReconciled(ctx, c, within)
		case "trenova":
			note, err = r.env.expectState(ctx, c, e.Trenova, within)
			r.snapshot(ctx, c)
		}
		if err != nil {
			if len(r.group(about)) > 1 {
				err = fmt.Errorf("%s: %w", c.Name, err)
			}
			return strings.Join(notes, "; "), err
		}
		if note != "" {
			notes = append(notes, note)
		}
	}
	if len(notes) > 3 {
		return fmt.Sprintf("%s … (%d conversations)", notes[0], len(notes)), nil
	}
	return strings.Join(notes, "; "), nil
}

func wireTransactions(view simclient.View, c *Conversation, set string) []transaction {
	messages := view.About(c.Party.Name(), c.Key)
	if !slices.Contains(ackSets, set) {
		return received(messages, set, c.Key)
	}
	var out []transaction
	for index := range messages {
		m := &messages[index]
		if m.Direction != journal.FromTrenova || !slices.Contains(m.Sets, set) {
			continue
		}
		for _, t := range transactionsOf(m, "") {
			if t.Set == set || set == "TA1" {
				out = append(out, t)
			}
		}
	}
	return out
}

func (e *Env) expectWire(
	ctx context.Context,
	c *Conversation,
	want *scenario.Expect,
	within time.Duration,
) (string, error) {
	if c.Key == "" {
		return "", failure(SourceHarness, "wire", ErrNoKey)
	}
	set := want.Wire
	count := max(want.Count, 1)
	var verdict error
	var picked []transaction
	var seen []transaction
	err := e.Watch.Until(ctx, time.Now().Add(within), func(view simclient.View) bool {
		all := wireTransactions(view, c, set)
		seen = all
		fresh := slices.DeleteFunc(slices.Clone(all), func(t transaction) bool {
			return c.isClaimed(t.id())
		})
		switch {
		case want.None:
			if len(fresh) > 0 {
				picked = fresh
				verdict = failf(SourceExpect, "none",
					"Trenova sent a %s about %s that the scenario says it must not", set, c.Key)
				return true
			}
			return false
		case want.Resent:
			for _, t := range fresh {
				if slices.ContainsFunc(all, func(earlier transaction) bool {
					return earlier.id() != t.id() && earlier.body == t.body && earlier.At <= t.At
				}) {
					picked = []transaction{t}
					return true
				}
			}
			return false
		case len(want.Statuses) > 0:
			var codes []string
			for index, t := range fresh {
				codes = append(codes, t.statusCodes()...)
				if covers(codes, want.Statuses) {
					picked = fresh[:index+1]
					verdict = checkTransactions(picked, want, c)
					return true
				}
			}
			return false
		case len(fresh) >= count:
			picked = fresh[:count]
			verdict = checkTransactions(picked, want, c)
			return true
		}
		return false
	})
	if want.None && errors.Is(err, simclient.ErrDeadline) {
		c.noteEvent(runlog.Event{Label: "No " + set + " sent",
			Detail: "within " + scenario.ShortDuration(within)})
		return fmt.Sprintf("no %s within %s", set, within), nil
	}
	if err != nil {
		c.noteEvent(runlog.Event{Label: "No " + set + " reached " + c.Party.Def.Partner.Display,
			Detail: "within " + scenario.ShortDuration(within), Tone: runlog.ToneBad})
		return "", failf(SourceExpect, "wire "+set, "no %s about %s reached the partner "+
			"within %s%s", set, c.Key, within, seenNote(seen, c, want))
	}
	ids := make([]string, 0, len(picked))
	for _, t := range picked {
		ids = append(ids, t.id())
		c.noteMatch(runlog.Match{MessageID: t.MessageID, GroupControl: t.Group,
			Control: t.Control})
	}
	c.claim(ids...)
	if verdict != nil {
		return "", verdict
	}
	return fmt.Sprintf("%s %s", set, strings.Join(controls(picked), ",")), nil
}

func seenNote(seen []transaction, c *Conversation, want *scenario.Expect) string {
	if len(seen) == 0 {
		return ""
	}
	var codes []string
	claimed := 0
	for _, t := range seen {
		if c.isClaimed(t.id()) {
			claimed++
		}
		codes = append(codes, t.statusCodes()...)
	}
	note := fmt.Sprintf(" (%d seen, %d matched by earlier steps", len(seen), claimed)
	if len(want.Statuses) > 0 {
		note += fmt.Sprintf("; statuses %v, wanted %v", codes, want.Statuses)
	}
	return note + ")"
}

func covers(codes, wanted []string) bool {
	left := map[string]int{}
	for _, code := range codes {
		left[strings.ToUpper(code)]++
	}
	for _, code := range wanted {
		code = strings.ToUpper(code)
		if left[code] == 0 {
			return false
		}
		left[code]--
	}
	return true
}

func controls(transactions []transaction) []string {
	out := make([]string, 0, len(transactions))
	for _, t := range transactions {
		out = append(out, t.Control)
	}
	return out
}

func checkTransactions(picked []transaction, want *scenario.Expect, c *Conversation) error {
	accepted := want.Accepted == nil || *want.Accepted
	for _, t := range picked {
		if t.Accepted != accepted {
			if accepted {
				return failf(SourceOracle, "partner guide",
					"the %s %s about %s breaks the partner's guide: %s", t.Set, t.Control, c.Key,
					describeFindings(t.Findings))
			}
			return failf(SourceOracle, "partner guide",
				"the %s %s about %s passed the partner's guide; the scenario expects it to "+
					"break it", t.Set, t.Control, c.Key)
		}
		for _, spec := range slices.Sorted(maps.Keys(want.Elements)) {
			wanted := want.Elements[spec]
			got, found := t.element(spec)
			if !found {
				return failf(SourceExpect, spec, "the %s %s has no %s segment", t.Set, t.Control,
					strings.SplitN(spec, "-", 2)[0])
			}
			if !strings.EqualFold(strings.TrimSpace(got), wanted) {
				return failf(SourceExpect, spec, "the %s %s has %s %q, wanted %q", t.Set,
					t.Control, spec, got, wanted)
			}
		}
		if want.Total != "" {
			if err := checkTotal(&t, want.Total, c); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkTotal(t *transaction, want string, c *Conversation) error {
	got, err := t.amountCents(invoiceTotal)
	if err != nil {
		return failure(SourceExpect, invoiceTotal, err)
	}
	expected, err := expectedCents(want, c)
	if err != nil {
		return failure(SourceExpect, invoiceTotal, err)
	}
	if got != expected {
		return failf(SourceExpect, invoiceTotal, "the 210 %s bills %s, wanted %s (%s)",
			t.Control, money(got), money(expected), want)
	}
	return nil
}

func expectedCents(want string, c *Conversation) (int64, error) {
	switch {
	case want == scenario.TotalTendered:
		return c.tenderedCents(), nil
	case want == scenario.TotalOriginal:
		return c.originalCents(), nil
	case strings.HasSuffix(want, "%"):
		percent, err := strconv.ParseFloat(strings.TrimSuffix(want, "%"), 64)
		if err != nil {
			return 0, err
		}
		return percentOf(c.originalCents(), percent), nil
	}
	return rateCents(want, -1), nil
}

func money(cents int64) string {
	return fmt.Sprintf("$%d.%02d", cents/100, cents%100)
}

func (e *Env) expectAck(
	ctx context.Context,
	c *Conversation,
	want *scenario.Expect,
	within time.Duration,
) (string, error) {
	if c.Party.Def.Partner.Acknowledgment.Expects == partner.AckNone {
		return "", failf(SourceHarness, "ack", "%s expects no acknowledgment", c.Party.Name())
	}
	accepted := want.Accepted == nil || *want.Accepted
	var verdict error
	var codes, acks []string
	err := e.Watch.Until(ctx, time.Now().Add(within), func(view simclient.View) bool {
		targets := ackTargets(view, c, want.Set)
		if len(targets) == 0 {
			return false
		}
		codes, acks = codes[:0], acks[:0]
		rejected := false
		for index := range targets {
			m := &targets[index]
			if m.Ack == nil || m.Ack.Status != journal.AckReceived {
				return false
			}
			codes = append(codes, m.Ack.Code)
			acks = append(acks, m.Ack.MessageID)
			ack, found := view.Message(m.Ack.MessageID)
			if found && ack.Accepted != nil && !*ack.Accepted {
				verdict = failf(SourceOracle, "partner guide",
					"Trenova's %s for %s breaks the partner's guide: %s", m.Ack.Set, c.Key,
					describeFindings(allFindings(&ack)))
				return true
			}
			if m.Ack.Code == acknowledge.Rejected || m.Ack.Code == acknowledge.Partial {
				rejected = true
			}
		}
		if accepted && rejected {
			verdict = failf(SourceExpect, "AK5", "Trenova rejected what the partner sent about "+
				"%s (%s); the scenario expects it accepted", c.Key, strings.Join(codes, ","))
		}
		if !accepted && !rejected {
			verdict = failf(SourceExpect, "AK5", "Trenova accepted everything the partner sent "+
				"about %s (%s); the scenario expects a rejection", c.Key, strings.Join(codes, ","))
		}
		return true
	})
	for _, id := range acks {
		c.noteMatch(runlog.Match{MessageID: id})
	}
	if err != nil {
		c.noteEvent(runlog.Event{
			Label:  "No " + string(c.Party.Def.Partner.Acknowledgment.Expects) + " from Trenova",
			Detail: "within " + scenario.ShortDuration(within),
			Tone:   runlog.ToneBad,
		})
		return "", failf(SourceExpect, "ack", "Trenova sent no %s for what the partner sent "+
			"about %s within %s", c.Party.Def.Partner.Acknowledgment.Expects,
			firstNonEmpty(c.Key, c.Name), within)
	}
	if verdict != nil {
		return "", verdict
	}
	return "ack " + strings.Join(codes, ","), nil
}

func ackTargets(view simclient.View, c *Conversation, set string) []journal.Message {
	if set == "" && len(c.LastSent) > 0 {
		out := make([]journal.Message, 0, len(c.LastSent))
		for _, id := range c.LastSent {
			if m, ok := view.Message(id); ok {
				out = append(out, m)
			}
		}
		return out
	}
	if c.Key == "" {
		return nil
	}
	var out []journal.Message
	for _, m := range view.About(c.Party.Name(), c.Key) {
		if m.Direction != journal.ToTrenova ||
			(m.Purpose != journal.PurposeDocument && m.Purpose != journal.PurposeAnswer) {
			continue
		}
		if set == "" || slices.Contains(m.Sets, set) {
			out = append(out, m)
		}
	}
	return out
}

func allFindings(m *journal.Message) []oracle.Finding {
	if m.Judgement == nil {
		return nil
	}
	out := slices.Clone(m.Judgement.Findings)
	for _, ic := range m.Judgement.Interchanges {
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

func (e *Env) expectFlow(
	ctx context.Context,
	c *Conversation,
	want *scenario.Expect,
	within time.Duration,
) (string, error) {
	if c.Key == "" {
		return "", failure(SourceHarness, "flow", ErrNoKey)
	}
	status, why := "", ""
	err := poll.Until(ctx, within, func(ctx context.Context) (bool, error) {
		instances, err := e.Trenova.FlowInstances(ctx, c.Party.Setup.PartnerID, c.Key)
		if err != nil || len(instances) == 0 {
			return false, err
		}
		if want.Count > 0 && len(instances) != want.Count {
			status, why = instances[0].Status, fmt.Sprintf("%d flow instances, wanted %d",
				len(instances), want.Count)
			return false, nil
		}
		instance := instances[0]
		c.mu.Lock()
		c.instanceID = instance.ID
		c.mu.Unlock()
		status, why = instance.Status, instance.Why()
		return (want.Status == "" || strings.EqualFold(status, want.Status)) &&
			(want.Why == "" || strings.Contains(strings.ToLower(why), strings.ToLower(want.Why))), nil
	})
	if err != nil {
		if status == "" {
			c.noteEvent(runlog.Event{Label: "No flow instance",
				Detail: "within " + scenario.ShortDuration(within), Tone: runlog.ToneBad})
			return "", failf(SourceTrenova, "flow", "no flow instance for %s within %s: %v",
				c.Key, within, err)
		}
		c.noteEvent(runlog.Event{Label: "Flow " + strings.ToLower(status), Detail: why,
			Tone: runlog.ToneBad})
		return "", failf(SourceExpect, "flow", "the flow for %s is %s (%s); wanted %s%s",
			c.Key, status, firstNonEmpty(why, "no reason"), firstNonEmpty(want.Status, "any"),
			whyNote(want.Why))
	}
	c.noteEvent(runlog.Event{Label: "Flow " + strings.ToLower(status), Detail: why,
		Tone: flowTone(status)})
	return firstNonEmpty(status+" "+quoted(why), status), nil
}

func flowTone(status string) string {
	switch strings.ToLower(status) {
	case "held", "escalated":
		return runlog.ToneHold
	}
	return runlog.ToneTrenova
}

func whyNote(why string) string {
	if why == "" {
		return ""
	}
	return fmt.Sprintf(" because %q", why)
}

func quoted(text string) string {
	if text == "" {
		return ""
	}
	return strconv.Quote(text)
}

func (e *Env) expectReconciled(
	ctx context.Context,
	c *Conversation,
	within time.Duration,
) (string, error) {
	if c.Key == "" {
		return "", failure(SourceHarness, "reconciled", ErrNoKey)
	}
	deadline := time.Now().Add(within)
	var documents []journal.Message
	err := e.Watch.Until(ctx, deadline, func(view simclient.View) bool {
		documents = documents[:0]
		for _, m := range view.About(c.Party.Name(), c.Key) {
			if m.Direction == journal.FromTrenova && m.Purpose == journal.PurposeDocument {
				documents = append(documents, m)
			}
		}
		if len(documents) == 0 {
			return false
		}
		for index := range documents {
			ack := documents[index].Ack
			if ack == nil || ack.Status != journal.AckSent {
				return false
			}
		}
		return true
	})
	if err != nil {
		c.noteEvent(runlog.Event{Label: "Not reconciled",
			Detail: "the partner has not acknowledged every document", Tone: runlog.ToneBad})
		return "", failf(SourceSim, "ack", "the partner has not acknowledged every document "+
			"Trenova sent about %s within %s (%d documents)", c.Key, within, len(documents))
	}
	var pending []string
	err = poll.Until(ctx, time.Until(deadline), func(ctx context.Context) (bool, error) {
		messages, err := e.Trenova.OutboundMessages(ctx, c.Party.Setup.PartnerID, "", 200)
		if err != nil {
			return false, err
		}
		pending = pending[:0]
		for index := range documents {
			d := &documents[index]
			for _, g := range d.Groups {
				for _, t := range g.Transactions {
					status := "missing"
					for _, m := range messages {
						if sameControl(m.InterchangeControlNumber, d.Interchange) &&
							sameControl(m.TransactionControlNumber, t.Control) {
							status = m.AckStatus
							break
						}
					}
					if status != "Accepted" && status != "AcceptedWithErrors" {
						pending = append(pending, fmt.Sprintf("%s %s/%s %s", t.Set,
							d.Interchange, t.Control, status))
					}
				}
			}
		}
		return len(pending) == 0, nil
	})
	if err != nil {
		c.noteEvent(runlog.Event{Label: "Not reconciled", Detail: strings.Join(pending, "; "),
			Tone: runlog.ToneBad})
		return "", failf(SourceTrenova, "reconcile", "Trenova has not reconciled the "+
			"partner's acknowledgments for %s: %s", c.Key, strings.Join(pending, "; "))
	}
	c.noteEvent(runlog.Event{Label: "Acknowledgments reconciled",
		Detail: fmt.Sprintf("%d documents", len(documents))})
	return fmt.Sprintf("%d documents reconciled", len(documents)), nil
}

func sameControl(a, b string) bool {
	return strings.TrimLeft(
		strings.TrimSpace(a),
		"0",
	) == strings.TrimLeft(
		strings.TrimSpace(b),
		"0",
	)
}

func (e *Env) expectState(
	ctx context.Context,
	c *Conversation,
	want *scenario.State,
	within time.Duration,
) (string, error) {
	if want.Exists != nil && !*want.Exists {
		if _, err := e.Instance(ctx, c, within); err != nil {
			return "", failure(SourceTrenova, "flow", err)
		}
		c.mu.Lock()
		shipmentID := c.shipmentID
		c.mu.Unlock()
		if shipmentID != "" {
			c.noteEvent(runlog.Event{Label: "A shipment was built", Tone: runlog.ToneBad})
			return "", failf(SourceExpect, "shipment", "%s has a shipment; the scenario "+
				"expects none", c.Key)
		}
		c.noteEvent(runlog.Event{Label: "No shipment built"})
		return "no shipment", nil
	}
	shipmentID, err := e.ShipmentOf(ctx, c, within)
	if err != nil {
		return "", failure(SourceExpect, "shipment", err)
	}
	var problems []string
	status := ""
	err = poll.Until(ctx, within, func(ctx context.Context) (bool, error) {
		shipment, err := e.Trenova.Shipment(ctx, shipmentID)
		if err != nil {
			return false, err
		}
		status = shipment.Status
		problems = problems[:0]
		if want.Status != "" && !strings.EqualFold(shipment.Status, want.Status) {
			problems = append(problems, fmt.Sprintf("status %s, wanted %s", shipment.Status,
				want.Status))
		}
		if stops := len(shipment.Stops()); want.Stops > 0 && stops != want.Stops {
			problems = append(problems, fmt.Sprintf("%d stops, wanted %d", stops, want.Stops))
		}
		if want.Total != "" {
			expected, expectErr := expectedCents(want.Total, c)
			if expectErr != nil {
				return false, poll.Stop(expectErr)
			}
			got, ok := shipment.TotalCents()
			if !ok || got != expected {
				problems = append(problems, fmt.Sprintf("total %s, wanted %s",
					firstNonEmpty(shipment.TotalChargeAmount, "none"), money(expected)))
			}
		}
		return len(problems) == 0, nil
	})
	if err != nil {
		c.noteEvent(runlog.Event{Label: "Shipment " + strings.ToLower(firstNonEmpty(status,
			"unreadable")), Detail: strings.Join(problems, "; "), Tone: runlog.ToneBad})
		if len(problems) == 0 {
			return "", failure(SourceTrenova, "shipment", err)
		}
		return "", failf(SourceExpect, "shipment", "%s's shipment: %s", c.Key,
			strings.Join(problems, "; "))
	}
	c.noteEvent(runlog.Event{Label: "Shipment " + strings.ToLower(status)})
	return "shipment " + status, nil
}
