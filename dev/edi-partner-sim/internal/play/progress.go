package play

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
)

const progressTimeout = 5 * time.Second

type Trace struct {
	mu      sync.Mutex
	matches []runlog.Match
	events  []runlog.Event
	gap     string
}

func (t *Trace) match(m runlog.Match) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if !slices.Contains(t.matches, m) {
		t.matches = append(t.matches, m)
	}
}

func (t *Trace) event(e runlog.Event) {
	if t == nil {
		return
	}
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	if e.Tone == "" {
		e.Tone = runlog.ToneTrenova
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, e)
}

func (t *Trace) addGap(gap string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.gap != "" {
		t.gap += ", "
	}
	t.gap += gap
}

func (t *Trace) Take() ([]runlog.Match, []runlog.Event, string) {
	if t == nil {
		return nil, nil, ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	matches, events, gap := t.matches, t.events, t.gap
	t.matches, t.events, t.gap = nil, nil, ""
	return matches, events, gap
}

func (c *Conversation) Follow(t *Trace) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trace = t
}

func (c *Conversation) tracer() *Trace {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.trace
}

func (c *Conversation) noteMatch(m runlog.Match) {
	if m.Key == "" {
		m.Key = c.Key
	}
	c.tracer().match(m)
}

func (c *Conversation) noteEvent(e runlog.Event) {
	if e.Key == "" {
		e.Key = c.Key
	}
	c.tracer().event(e)
}

type progress struct {
	env    *Env
	runID  string
	failed atomic.Bool
}

func (e *Env) BeginRun(ctx context.Context, kind string, seed uint64) {
	ctx, cancel := context.WithTimeout(ctx, progressTimeout)
	defer cancel()
	summary, err := e.Sim.StartRun(ctx, &runlog.Start{
		Kind:      kind,
		Number:    e.Run,
		Seed:      seed,
		API:       e.api,
		StartedAt: time.Now().UTC(),
	})
	e.progress = &progress{env: e, runID: summary.ID}
	if err != nil {
		e.progress.warn(err)
	}
}

func (e *Env) EndRun(ctx context.Context, passed bool) {
	status := runlog.StatusPassed
	if !passed {
		status = runlog.StatusFailed
	}
	e.progress.post(ctx, func(ctx context.Context, id string) error {
		return e.Sim.FinishRun(ctx, id, &runlog.Finish{Status: status})
	})
}

func (e *Env) PostScenario(ctx context.Context, scenario *runlog.Scenario) {
	e.progress.post(ctx, func(ctx context.Context, id string) error {
		return e.Sim.PutScenario(ctx, id, scenario)
	})
}

func (e *Env) PostStep(ctx context.Context, partner, slug string, step *runlog.Step) {
	e.progress.post(ctx, func(ctx context.Context, id string) error {
		return e.Sim.PutStep(ctx, id, &runlog.StepPost{
			Partner:  partner,
			Scenario: slug,
			Step:     *step,
		})
	})
}

func (p *progress) post(ctx context.Context, send func(context.Context, string) error) {
	if p == nil || p.runID == "" || p.failed.Load() {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), progressTimeout)
	defer cancel()
	if err := send(ctx, p.runID); err != nil {
		p.warn(err)
	}
}

func (p *progress) warn(err error) {
	if p.failed.Swap(true) {
		return
	}
	p.env.logf("the viewer will not show this run: the simulator did not take its progress: %v",
		err)
}
