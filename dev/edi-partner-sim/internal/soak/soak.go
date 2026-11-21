package soak

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/play"
	"github.com/emoss08/trenova/edi-partner-sim/internal/poll"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/simclient"
	"github.com/sourcegraph/conc/pool"
	"golang.org/x/time/rate"
)

const (
	DefaultConversations = 20
	DefaultRate          = 1.0
	DefaultConcurrency   = 8
	DefaultWithin        = 10 * time.Minute
	DefaultSettle        = 10 * time.Minute
	maxPerPartner        = 999
	maxErrorsKept        = 20
	soakTenderCents      = 185000
	flowPage             = 100
)

var openStatuses = []string{"Open", "Held", "Escalated"}

const (
	MilestoneAck     = "997"
	MilestoneAnswer  = "990"
	MilestoneStatus  = "214"
	MilestoneInvoice = "210"
)

var milestoneOrder = []string{MilestoneAck, MilestoneAnswer, MilestoneStatus, MilestoneInvoice}

type Options struct {
	Play          play.Options
	Conversations int
	Rate          float64
	Concurrency   int
	Within        time.Duration
	Settle        time.Duration
}

func (o *Options) normalize() error {
	if o.Conversations < 1 {
		o.Conversations = DefaultConversations
	}
	if o.Rate <= 0 {
		o.Rate = DefaultRate
	}
	if o.Concurrency < 1 {
		o.Concurrency = DefaultConcurrency
	}
	if o.Within <= 0 {
		o.Within = DefaultWithin
	}
	if o.Settle < 0 {
		return errors.New("--settle cannot be negative")
	}
	return nil
}

type outcome struct {
	party        *play.Party
	conversation *play.Conversation
	took         map[string]time.Duration
	first        map[string]string
	lost         []string
	duplicates   map[string]int
	err          error
}

func Run(ctx context.Context, o *Options) (*Report, error) {
	if err := o.normalize(); err != nil {
		return nil, err
	}
	env, err := play.Open(ctx, &o.Play)
	if err != nil {
		return nil, err
	}
	defer env.Close()
	definitions, err := env.Definitions(o.Play.Partners)
	if err != nil {
		return nil, err
	}
	if per := (o.Conversations + len(definitions) - 1) / len(definitions); per > maxPerPartner {
		return nil, fmt.Errorf("%d conversations over %d partners is %d each; a run has "+
			"seeds for %d per partner", o.Conversations, len(definitions), per, maxPerPartner)
	}
	report := &Report{
		Run:           env.Run,
		Conversations: o.Conversations,
		Rate:          o.Rate,
		Concurrency:   o.Concurrency,
		StartedAt:     time.Now().UTC(),
	}
	var parties []*play.Party
	for _, def := range definitions {
		party := env.Provision(ctx, def)
		if party.Setup.Failure != "" {
			failed := newPartner(def.Name(), string(def.Partner.Role))
			failed.Failure = party.Setup.Failure
			report.Partners = append(report.Partners, failed)
			continue
		}
		parties = append(parties, party)
	}
	if len(parties) == 0 {
		report.finish(nil, time.Now())
		return report, errors.New("no partner could be provisioned")
	}
	started := time.Now()
	env.BeginRun(ctx, runlog.KindSoak, 0)
	outcomes, err := run(ctx, env, o, parties)
	if err != nil {
		env.EndRun(ctx, false)
		return nil, err
	}
	openFlows := settle(ctx, env, o.Settle, outcomes)
	report.finish(outcomes, started)
	for _, p := range report.Partners {
		p.OpenFlows = openFlows[p.Name]
		report.Total.OpenFlows += p.OpenFlows
	}
	env.EndRun(ctx, report.Passed())
	return report, nil
}

func run(
	ctx context.Context,
	env *play.Env,
	o *Options,
	parties []*play.Party,
) ([]*outcome, error) {
	limiter := rate.NewLimiter(rate.Limit(o.Rate), 1)
	work := pool.NewWithResults[*outcome]().WithContext(ctx).
		WithMaxGoroutines(o.Concurrency)
	for n := range o.Conversations {
		if err := limiter.Wait(ctx); err != nil {
			break
		}
		party := parties[n%len(parties)]
		offset := uint64(n/len(parties) + 1)
		c := &play.Conversation{
			Name:  "soak-" + strconv.Itoa(n+1),
			Party: party,
			Need:  party.Happy,
			Seed:  play.SeedOf(env.Run, offset),
			BOL:   fmt.Sprintf("SOAK%03d%04d", env.Run, n+1),
		}
		work.Go(func(ctx context.Context) (*outcome, error) {
			return converse(ctx, env, o.Within, c), nil
		})
	}
	outcomes, err := work.Wait()
	if err != nil {
		return nil, err
	}
	return outcomes, ctx.Err()
}

func converse(
	ctx context.Context,
	env *play.Env,
	within time.Duration,
	c *play.Conversation,
) *outcome {
	out := &outcome{
		party:        c.Party,
		conversation: c,
		took:         map[string]time.Duration{},
		first:        map[string]string{},
	}
	trace := new(play.Trace)
	c.Follow(trace)
	steps := phases(env, within, c)
	logged := planned(steps, c)
	env.PostScenario(ctx, logged)
	for index, step := range steps {
		err := step.run(ctx)
		matches, events, _ := trace.Take()
		entry := &logged.Steps[index]
		entry.Status, entry.Ended = runlog.StatusPassed, time.Now().UTC()
		entry.Matches, entry.Events, entry.Keys = matches, events, keysOf(c)
		if err != nil {
			out.err = fmt.Errorf("%s: %w", step.name, err)
			entry.Status, entry.Why = runlog.StatusFailed, err.Error()
		}
		env.PostStep(ctx, c.Party.Name(), logged.Slug, entry)
		if err != nil {
			break
		}
	}
	if c.Key == "" {
		out.lost = slices.Clone(milestoneOrder)
		finishLogged(ctx, env, logged, out, len(steps))
		return out
	}
	deadline := time.Now().Add(within)
	_ = env.Watch.Until(ctx, deadline, func(view simclient.View) bool {
		return len(observe(view, c, out)) == 0
	})
	env.Watch.Read(func(view simclient.View) {
		out.lost = observe(view, c, out)
		out.duplicates = duplicates(view, c)
	})
	finishLogged(ctx, env, logged, out, len(steps))
	return out
}

type phase struct {
	kind string
	name string
	text string
	run  func(context.Context) error
}

func phases(env *play.Env, within time.Duration, c *play.Conversation) []phase {
	if c.Party.Carrier() {
		return []phase{{kind: "trenova", name: "tender", text: "trenova tender " + c.Name,
			run: func(ctx context.Context) error {
				return env.TenderOut(ctx, c, soakTenderCents, within)
			}}}
	}
	original := partner.Variation{Kind: partner.KindOriginal, Need: c.Need}
	return []phase{
		{kind: "send", name: "send", text: "send 204 original as " + c.Name,
			run: func(ctx context.Context) error {
				_, err := env.SendTender(ctx, c.Party, []*play.Conversation{c}, original)
				return err
			}},
		{kind: "trenova", name: "accept", text: "trenova decide " + c.Name + " accept",
			run: func(ctx context.Context) error { return env.Decide(ctx, c, true, "", within) }},
		{kind: "trenova", name: "dispatch", text: "trenova dispatch " + c.Name,
			run: func(ctx context.Context) error { return env.Dispatch(ctx, c, within) }},
		{kind: "trenova", name: "deliver", text: "trenova deliver " + c.Name,
			run: func(ctx context.Context) error { return env.Deliver(ctx, c, within) }},
		{kind: "trenova", name: "invoice", text: "trenova invoice " + c.Name,
			run: func(ctx context.Context) error { return env.Invoice(ctx, c, within) }},
	}
}

func planned(steps []phase, c *play.Conversation) *runlog.Scenario {
	logged := &runlog.Scenario{
		Partner:   c.Party.Name(),
		Slug:      c.Name,
		Name:      "Soak conversation " + strings.TrimPrefix(c.Name, "soak-"),
		Seed:      c.Seed,
		Status:    runlog.StatusRunning,
		StartedAt: time.Now().UTC(),
		Keys:      []string{},
	}
	for index, step := range steps {
		logged.Steps = append(logged.Steps, runlog.Step{Index: index, Kind: step.kind,
			Text: step.text, Status: runlog.StatusPending})
	}
	for _, name := range milestoneOrder {
		logged.Steps = append(logged.Steps, runlog.Step{Index: len(logged.Steps),
			Kind: "expect", Text: milestoneText(name, c), Status: runlog.StatusPending})
	}
	return logged
}

func milestoneText(name string, c *play.Conversation) string {
	if name == MilestoneAck {
		return "expect Trenova's acknowledgment about " + c.Name
	}
	return "expect wire " + name + " about " + c.Name
}

func finishLogged(
	ctx context.Context,
	env *play.Env,
	logged *runlog.Scenario,
	out *outcome,
	from int,
) {
	logged.Keys = keysOf(out.conversation)
	for offset, name := range milestoneOrder {
		entry := &logged.Steps[from+offset]
		entry.Ended, entry.Keys = time.Now().UTC(), logged.Keys
		switch {
		case out.first[name] != "":
			entry.Status = runlog.StatusPassed
			entry.Matches = []runlog.Match{{MessageID: out.first[name],
				Key: out.conversation.Key}}
		case slices.Contains(out.lost, name):
			entry.Status, entry.Why = runlog.StatusFailed, "never reached the partner"
		default:
			entry.Status = runlog.StatusSkipped
		}
	}
	for index := range logged.Steps {
		if logged.Steps[index].Status == runlog.StatusPending {
			logged.Steps[index].Status = runlog.StatusSkipped
		}
	}
	logged.Status = runlog.StatusPassed
	if out.err != nil || len(out.lost) > 0 {
		logged.Status = runlog.StatusFailed
	}
	logged.Took = time.Since(logged.StartedAt).Seconds()
	env.PostScenario(ctx, logged)
}

func keysOf(c *play.Conversation) []string {
	if c.Key == "" {
		return []string{}
	}
	return []string{c.Key}
}

func documentsFrom(c *play.Conversation) journal.Direction {
	if c.Party.Carrier() {
		return journal.ToTrenova
	}
	return journal.FromTrenova
}

func observe(view simclient.View, c *play.Conversation, out *outcome) []string {
	documents := documentsFrom(c)
	for _, m := range view.About(c.Party.Name(), c.Key) {
		since := max(m.At.Sub(c.StartedAt), 0)
		for _, set := range m.Sets {
			name := set
			switch {
			case m.Purpose == journal.PurposeAck && m.Direction == journal.FromTrenova:
				name = MilestoneAck
			case m.Direction != documents:
				continue
			}
			if _, seen := out.took[name]; !seen && slices.Contains(milestoneOrder, name) {
				out.took[name] = since
				out.first[name] = m.ID
			}
		}
	}
	var missing []string
	for _, name := range milestoneOrder {
		if _, ok := out.took[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing
}

func duplicates(view simclient.View, c *play.Conversation) map[string]int {
	documents := documentsFrom(c)
	bodies := map[string]map[string]int{}
	for _, m := range view.About(c.Party.Name(), c.Key) {
		if m.Purpose != journal.PurposeDocument || m.Direction != documents {
			continue
		}
		for _, set := range m.Sets {
			if bodies[set] == nil {
				bodies[set] = map[string]int{}
			}
			bodies[set][play.BodyOf(&m, set, c.Key)]++
		}
	}
	out := map[string]int{}
	for set, counts := range bodies {
		for _, count := range counts {
			if count > 1 {
				out[set] += count - 1
			}
		}
	}
	return out
}

func settle(
	ctx context.Context,
	env *play.Env,
	within time.Duration,
	outcomes []*outcome,
) map[string]int {
	byPartner := map[string]*play.Party{}
	keys := map[string]map[string]bool{}
	for _, o := range outcomes {
		if o.conversation.Key == "" {
			continue
		}
		name := o.party.Name()
		byPartner[name] = o.party
		if keys[name] == nil {
			keys[name] = map[string]bool{}
		}
		keys[name][o.conversation.Key] = true
	}
	open := map[string]int{}
	var mu sync.Mutex
	work := pool.New().WithContext(ctx).WithMaxGoroutines(len(byPartner) + 1)
	for name, party := range byPartner {
		work.Go(func(ctx context.Context) error {
			count := 0
			_ = poll.Until(ctx, max(within, time.Second), func(ctx context.Context) (bool, error) {
				instances, err := env.Trenova.FlowsByStatus(ctx, party.Setup.PartnerID,
					openStatuses, flowPage)
				if err != nil {
					return false, err
				}
				count = 0
				for _, instance := range instances {
					if keys[name][instance.Key] {
						count++
					}
				}
				return count == 0, nil
			})
			mu.Lock()
			open[name] = count
			mu.Unlock()
			return nil
		})
	}
	_ = work.Wait()
	return open
}
