package runlog

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	KindRun     = "run"
	KindSoak    = "soak"
	KindConform = "conform"
)

const (
	StatusPending = "pending"
	StatusRunning = "running"
	StatusPassed  = "passed"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
)

const (
	ToneTrenova = "trenova"
	ToneHold    = "hold"
	ToneBad     = "bad"
)

const (
	SourceLive   = "live"
	SourceReport = "report"
)

const (
	DefaultRuns  = 20
	MaxScenarios = 4000
	MaxSteps     = 400
	maxMatches   = 200
	maxEvents    = 50
	maxText      = 2000
)

var (
	ErrNoRun      = errors.New("no such run")
	ErrInvalid    = errors.New("invalid run progress")
	ErrNoScenario = errors.New("no such scenario")
)

type Run struct {
	ID         string      `json:"id"`
	Kind       string      `json:"kind"`
	Number     uint64      `json:"number,omitempty"`
	Seed       uint64      `json:"seed,omitempty"`
	API        string      `json:"api,omitempty"`
	Source     string      `json:"source"`
	File       string      `json:"file,omitempty"`
	Status     string      `json:"status"`
	StartedAt  time.Time   `json:"startedAt"`
	FinishedAt *time.Time  `json:"finishedAt,omitempty"`
	Scenarios  []*Scenario `json:"scenarios"`
}

type Scenario struct {
	Partner   string    `json:"partner"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Seed      uint64    `json:"seed,omitempty"`
	Status    string    `json:"status"`
	Replay    string    `json:"replay,omitempty"`
	StartedAt time.Time `json:"startedAt"`
	Took      float64   `json:"took,omitempty"`
	Keys      []string  `json:"keys"`
	Waiting   bool      `json:"waiting,omitempty"`
	Steps     []Step    `json:"steps"`
}

type Step struct {
	Index   int       `json:"index"`
	Kind    string    `json:"kind"`
	Text    string    `json:"text"`
	Status  string    `json:"status"`
	Note    string    `json:"note,omitempty"`
	Why     string    `json:"why,omitempty"`
	Ended   time.Time `json:"ended"`
	Keys    []string  `json:"keys,omitempty"`
	Matches []Match   `json:"matches,omitempty"`
	Events  []Event   `json:"events,omitempty"`
	Gap     string    `json:"gap,omitempty"`
}

type Match struct {
	MessageID    string `json:"messageId"`
	GroupControl string `json:"groupControl,omitempty"`
	Control      string `json:"control,omitempty"`
	Key          string `json:"key,omitempty"`
}

type Event struct {
	Key    string    `json:"key,omitempty"`
	At     time.Time `json:"at"`
	Label  string    `json:"label"`
	Detail string    `json:"detail,omitempty"`
	Tone   string    `json:"tone"`
	Who    string    `json:"who,omitempty"`
}

type Tally struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Waiting int `json:"waiting"`
	Running int `json:"running"`
	Pending int `json:"pending"`
}

type Summary struct {
	ID         string           `json:"id"`
	Kind       string           `json:"kind"`
	Number     uint64           `json:"number,omitempty"`
	Seed       uint64           `json:"seed,omitempty"`
	API        string           `json:"api,omitempty"`
	Source     string           `json:"source"`
	File       string           `json:"file,omitempty"`
	Status     string           `json:"status"`
	StartedAt  time.Time        `json:"startedAt"`
	FinishedAt *time.Time       `json:"finishedAt,omitempty"`
	Tally      Tally            `json:"tally"`
	Partners   map[string]Tally `json:"partners"`
}

type Start struct {
	Kind      string    `json:"kind"`
	Number    uint64    `json:"number,omitempty"`
	Seed      uint64    `json:"seed,omitempty"`
	API       string    `json:"api,omitempty"`
	StartedAt time.Time `json:"startedAt"`
}

type Finish struct {
	Status string `json:"status"`
}

type StepPost struct {
	Partner  string `json:"partner"`
	Scenario string `json:"scenario"`
	Step     Step   `json:"step"`
}

type Notice struct {
	Kind     string
	RunID    string
	Partner  string
	Scenario string
	Keys     []string
}

type Attached struct {
	Run      Summary  `json:"run"`
	Scenario Scenario `json:"scenario"`
}

type location struct {
	run      string
	scenario int
}

type Store struct {
	mu       sync.RWMutex
	capacity int
	seq      uint64
	runs     []*Run
	index    map[string]location
	now      func() time.Time
	notify   func(Notice)
}

func New(capacity int, now func() time.Time, notify func(Notice)) *Store {
	if capacity < 1 {
		capacity = DefaultRuns
	}
	if now == nil {
		now = time.Now
	}
	if notify == nil {
		notify = func(Notice) {}
	}
	return &Store{capacity: capacity, index: map[string]location{}, now: now, notify: notify}
}

func (s *Store) Start(start Start) (Summary, error) {
	return s.add(&Run{
		Kind:      start.Kind,
		Number:    start.Number,
		Seed:      start.Seed,
		API:       start.API,
		Source:    SourceLive,
		Status:    StatusRunning,
		StartedAt: start.StartedAt,
	})
}

func (s *Store) Load(run *Run) (Summary, error) {
	run.Source = SourceReport
	return s.add(run)
}

func (s *Store) add(run *Run) (Summary, error) {
	switch run.Kind {
	case KindRun, KindSoak, KindConform:
	default:
		return Summary{}, fmt.Errorf("%w: kind is run, soak or conform, not %q", ErrInvalid,
			run.Kind)
	}
	if run.StartedAt.IsZero() {
		run.StartedAt = s.now().UTC()
	}
	if run.Scenarios == nil {
		run.Scenarios = []*Scenario{}
	}
	s.mu.Lock()
	s.seq++
	run.ID = "r" + strconv.FormatUint(s.seq, 10)
	s.runs = append(s.runs, run)
	for len(s.runs) > s.capacity {
		s.forget(s.runs[0])
		s.runs = s.runs[1:]
	}
	for index, scenario := range run.Scenarios {
		normalize(scenario)
		s.indexKeys(run.ID, index, scenario.Partner, scenario.Keys)
	}
	summary := summarize(run)
	s.mu.Unlock()
	s.notify(Notice{Kind: "run.started", RunID: run.ID})
	return summary, nil
}

func (s *Store) Finish(id string, finish Finish) (Summary, error) {
	switch finish.Status {
	case StatusPassed, StatusFailed:
	default:
		return Summary{}, fmt.Errorf("%w: a run finishes passed or failed, not %q", ErrInvalid,
			finish.Status)
	}
	s.mu.Lock()
	run := s.find(id)
	if run == nil {
		s.mu.Unlock()
		return Summary{}, fmt.Errorf("%w: %s", ErrNoRun, id)
	}
	at := s.now().UTC()
	run.Status, run.FinishedAt = finish.Status, &at
	summary := summarize(run)
	s.mu.Unlock()
	s.notify(Notice{Kind: "run.finished", RunID: id})
	return summary, nil
}

func (s *Store) PutScenario(id string, in *Scenario) error {
	if strings.TrimSpace(in.Partner) == "" || strings.TrimSpace(in.Slug) == "" {
		return fmt.Errorf("%w: a scenario needs its partner and slug", ErrInvalid)
	}
	if err := checkStatus(in.Status); err != nil {
		return err
	}
	if len(in.Steps) > MaxSteps {
		return fmt.Errorf("%w: a scenario holds at most %d steps", ErrInvalid, MaxSteps)
	}
	s.mu.Lock()
	run := s.find(id)
	if run == nil {
		s.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrNoRun, id)
	}
	index := scenarioIndex(run, in.Partner, in.Slug)
	if index < 0 {
		if len(run.Scenarios) >= MaxScenarios {
			s.mu.Unlock()
			return fmt.Errorf("%w: a run holds at most %d scenarios", ErrInvalid, MaxScenarios)
		}
		fresh := *in
		fresh.Keys, fresh.Steps = nil, nil
		run.Scenarios = append(run.Scenarios, &fresh)
		index = len(run.Scenarios) - 1
	}
	stored := run.Scenarios[index]
	stored.Name, stored.Seed, stored.Replay = in.Name, in.Seed, in.Replay
	stored.Status, stored.Took = in.Status, in.Took
	if !in.StartedAt.IsZero() {
		stored.StartedAt = in.StartedAt
	}
	if in.Steps != nil {
		stored.Steps = make([]Step, 0, len(in.Steps))
		for _, step := range in.Steps {
			stored.Steps = append(stored.Steps, clip(step))
		}
	}
	stored.Keys = union(stored.Keys, in.Keys)
	normalize(stored)
	s.indexKeys(run.ID, index, stored.Partner, stored.Keys)
	notice := Notice{Kind: "run.scenario", RunID: id, Partner: stored.Partner,
		Scenario: stored.Slug, Keys: slices.Clone(stored.Keys)}
	s.mu.Unlock()
	s.notify(notice)
	return nil
}

func (s *Store) PutStep(id string, post *StepPost) error {
	if err := checkStatus(post.Step.Status); err != nil {
		return err
	}
	if post.Step.Index < 0 || post.Step.Index >= MaxSteps {
		return fmt.Errorf("%w: step index %d is out of range", ErrInvalid, post.Step.Index)
	}
	s.mu.Lock()
	run := s.find(id)
	if run == nil {
		s.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrNoRun, id)
	}
	index := scenarioIndex(run, post.Partner, post.Scenario)
	if index < 0 {
		s.mu.Unlock()
		return fmt.Errorf("%w: %s %s in run %s", ErrNoScenario, post.Partner, post.Scenario, id)
	}
	stored := run.Scenarios[index]
	for len(stored.Steps) <= post.Step.Index {
		stored.Steps = append(stored.Steps, Step{Index: len(stored.Steps),
			Status: StatusPending})
	}
	step := clip(post.Step)
	stored.Steps[step.Index] = step
	keys := step.Keys
	for _, match := range step.Matches {
		keys = union(keys, []string{match.Key})
	}
	for _, event := range step.Events {
		keys = union(keys, []string{event.Key})
	}
	stored.Keys = union(stored.Keys, keys)
	normalize(stored)
	s.indexKeys(run.ID, index, stored.Partner, stored.Keys)
	notice := Notice{Kind: "run.step", RunID: id, Partner: stored.Partner,
		Scenario: stored.Slug, Keys: slices.Clone(stored.Keys)}
	s.mu.Unlock()
	s.notify(notice)
	return nil
}

func (s *Store) Summaries() []Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Summary, 0, len(s.runs))
	for index := len(s.runs) - 1; index >= 0; index-- {
		out = append(out, summarize(s.runs[index]))
	}
	return out
}

func (s *Store) Current() (Summary, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.runs) == 0 {
		return Summary{}, false
	}
	return summarize(s.runs[len(s.runs)-1]), true
}

func (s *Store) Get(id string) (Run, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run := s.find(id)
	if run == nil {
		return Run{}, false
	}
	return clone(run), true
}

func (s *Store) For(partner, key string) (Attached, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	at, ok := s.index[indexKey(partner, key)]
	if !ok {
		return Attached{}, false
	}
	run := s.find(at.run)
	if run == nil || at.scenario >= len(run.Scenarios) {
		return Attached{}, false
	}
	return Attached{Run: summarize(run), Scenario: cloneScenario(run.Scenarios[at.scenario])},
		true
}

func (s *Store) Has(source, file string, startedAt time.Time) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.ContainsFunc(s.runs, func(r *Run) bool {
		return r.Source == source && r.File == file && r.StartedAt.Equal(startedAt)
	})
}

func (s *Store) Reset(partner string) {
	s.mu.Lock()
	if partner == "" {
		s.runs, s.index = nil, map[string]location{}
	} else {
		for _, run := range s.runs {
			s.forget(run)
			run.Scenarios = slices.DeleteFunc(run.Scenarios, func(sc *Scenario) bool {
				return sc.Partner == partner
			})
			for index, scenario := range run.Scenarios {
				s.indexKeys(run.ID, index, scenario.Partner, scenario.Keys)
			}
		}
	}
	s.mu.Unlock()
	s.notify(Notice{Kind: "run.reset", Partner: partner})
}

func (s *Store) find(id string) *Run {
	for _, run := range s.runs {
		if run.ID == id {
			return run
		}
	}
	return nil
}

func (s *Store) forget(run *Run) {
	for _, scenario := range run.Scenarios {
		for _, key := range scenario.Keys {
			name := indexKey(scenario.Partner, key)
			if at, ok := s.index[name]; ok && at.run == run.ID {
				delete(s.index, name)
			}
		}
	}
}

func (s *Store) indexKeys(runID string, scenario int, partner string, keys []string) {
	for _, key := range keys {
		s.index[indexKey(partner, key)] = location{run: runID, scenario: scenario}
	}
}

func indexKey(partner, key string) string {
	return partner + "\x00" + key
}

func scenarioIndex(run *Run, partner, slug string) int {
	return slices.IndexFunc(run.Scenarios, func(sc *Scenario) bool {
		return sc.Partner == partner && sc.Slug == slug
	})
}

func checkStatus(status string) error {
	switch status {
	case StatusPending, StatusRunning, StatusPassed, StatusFailed, StatusSkipped:
		return nil
	}
	return fmt.Errorf("%w: status %q is not pending, running, passed, failed or skipped",
		ErrInvalid, status)
}

func clip(step Step) Step {
	step.Text, step.Note, step.Why = cut(step.Text), cut(step.Note), cut(step.Why)
	if len(step.Matches) > maxMatches {
		step.Matches = step.Matches[:maxMatches]
	}
	if len(step.Events) > maxEvents {
		step.Events = step.Events[:maxEvents]
	}
	step.Keys = slices.Clone(step.Keys)
	step.Matches = slices.Clone(step.Matches)
	step.Events = slices.Clone(step.Events)
	for index := range step.Events {
		event := &step.Events[index]
		event.Label, event.Detail = cut(event.Label), cut(event.Detail)
		if event.Tone == "" {
			event.Tone = ToneTrenova
		}
	}
	return step
}

func cut(text string) string {
	if len(text) <= maxText {
		return text
	}
	return text[:maxText] + "…"
}

func union(base, more []string) []string {
	for _, value := range more {
		if value != "" && !slices.Contains(base, value) {
			base = append(base, value)
		}
	}
	return base
}

func normalize(scenario *Scenario) {
	if scenario.Keys == nil {
		scenario.Keys = []string{}
	}
	if scenario.Steps == nil {
		scenario.Steps = []Step{}
	}
	scenario.Waiting = waiting(scenario.Steps)
}

func waiting(steps []Step) bool {
	held := false
	for _, step := range steps {
		for _, event := range step.Events {
			switch event.Tone {
			case ToneHold:
				held = true
			case ToneTrenova:
				held = false
			}
		}
	}
	return held
}

func summarize(run *Run) Summary {
	summary := Summary{
		ID:         run.ID,
		Kind:       run.Kind,
		Number:     run.Number,
		Seed:       run.Seed,
		API:        run.API,
		Source:     run.Source,
		File:       run.File,
		Status:     run.Status,
		StartedAt:  run.StartedAt,
		FinishedAt: run.FinishedAt,
		Partners:   map[string]Tally{},
	}
	for _, scenario := range run.Scenarios {
		partner := summary.Partners[scenario.Partner]
		count(&summary.Tally, scenario)
		count(&partner, scenario)
		summary.Partners[scenario.Partner] = partner
	}
	return summary
}

func count(t *Tally, scenario *Scenario) {
	switch scenario.Status {
	case StatusPassed:
		t.Passed++
	case StatusFailed:
		t.Failed++
	case StatusRunning:
		if scenario.Waiting {
			t.Waiting++
		} else {
			t.Running++
		}
	case StatusPending:
		t.Pending++
	}
}

func clone(run *Run) Run {
	out := *run
	out.FinishedAt = clonePtr(run.FinishedAt)
	out.Scenarios = make([]*Scenario, 0, len(run.Scenarios))
	for _, scenario := range run.Scenarios {
		copied := cloneScenario(scenario)
		out.Scenarios = append(out.Scenarios, &copied)
	}
	return out
}

func cloneScenario(scenario *Scenario) Scenario {
	out := *scenario
	out.Keys = slices.Clone(scenario.Keys)
	out.Steps = make([]Step, 0, len(scenario.Steps))
	for _, step := range scenario.Steps {
		out.Steps = append(out.Steps, clip(step))
	}
	return out
}

func clonePtr[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

type Snapshot struct {
	Seq  uint64 `json:"seq"`
	Runs []Run  `json:"runs"`
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := Snapshot{Seq: s.seq, Runs: make([]Run, 0, len(s.runs))}
	for _, run := range s.runs {
		snapshot.Runs = append(snapshot.Runs, clone(run))
	}
	return snapshot
}

func (s *Store) Restore(snapshot Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq, s.runs, s.index = snapshot.Seq, nil, map[string]location{}
	runs := snapshot.Runs
	if len(runs) > s.capacity {
		runs = runs[len(runs)-s.capacity:]
	}
	for index := range runs {
		run := clone(&runs[index])
		s.runs = append(s.runs, &run)
		for at, scenario := range run.Scenarios {
			normalize(scenario)
			s.indexKeys(run.ID, at, scenario.Partner, scenario.Keys)
		}
	}
}
