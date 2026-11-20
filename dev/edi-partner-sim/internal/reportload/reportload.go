package reportload

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/conform"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/play"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/sim"
)

var (
	ErrUnknownReport = errors.New("the file is not a run or conform report")
	ErrSoakReport    = errors.New(
		"a soak report holds timings, not the wire; watch a soak live instead",
	)
)

type Result struct {
	Kind      string
	Scenarios int
	Messages  int
	Skipped   []string
	Already   bool
}

type probe struct {
	Count    *uint64 `json:"count"`
	Run      *uint64 `json:"run"`
	Partners []struct {
		Milestones map[string]any `json:"milestones"`
	} `json:"partners"`
}

func Load(s *sim.Server, path string) (Result, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read the report: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return Result{}, fmt.Errorf("read the report: %w", err)
	}
	var shape probe
	if err = sonic.Unmarshal(raw, &shape); err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrUnknownReport, err)
	}
	l := &loader{server: s, file: filepath.Base(path), base: info.ModTime().UTC()}
	for _, p := range shape.Partners {
		if p.Milestones != nil {
			return Result{}, ErrSoakReport
		}
	}
	switch {
	case shape.Count != nil:
		report := new(conform.Report)
		if err = sonic.Unmarshal(raw, report); err != nil {
			return Result{}, fmt.Errorf("read the conform report: %w", err)
		}
		return l.conform(report)
	case shape.Run != nil:
		report := new(play.Report)
		if err = sonic.Unmarshal(raw, report); err != nil {
			return Result{}, fmt.Errorf("read the run report: %w", err)
		}
		return l.run(report)
	}
	return Result{}, ErrUnknownReport
}

type loader struct {
	server *sim.Server
	file   string
	base   time.Time
	tick   int
	result Result
}

func (l *loader) served(name string) bool {
	for _, info := range l.server.Partners() {
		if info.Name == name {
			return true
		}
	}
	l.result.Skipped = append(l.result.Skipped, name)
	return false
}

func (l *loader) run(report *play.Report) (Result, error) {
	l.result.Kind = runlog.KindRun
	if l.server.Runs().Has(runlog.SourceReport, l.file, report.StartedAt) {
		l.result.Already = true
		return l.result, nil
	}
	finished := report.StartedAt.Add(time.Duration(report.Took))
	run := &runlog.Run{
		Kind:       runlog.KindRun,
		Number:     report.Run,
		API:        report.API,
		File:       l.file,
		Status:     statusOf(report.Passed()),
		StartedAt:  report.StartedAt,
		FinishedAt: &finished,
	}
	for _, p := range report.Partners {
		if !l.served(p.Name) {
			continue
		}
		ids, err := l.importWire(p)
		if err != nil {
			return l.result, err
		}
		for _, scenario := range p.Scenarios {
			logged := scenario.LogScenario(p.Name)
			for index := range logged.Steps {
				logged.Steps[index].Matches = remap(logged.Steps[index].Matches, ids)
			}
			run.Scenarios = append(run.Scenarios, logged)
		}
		for _, f := range p.Failures {
			run.Scenarios = append(run.Scenarios, failedSetup(p.Name, f.Message, f.Replay,
				report.StartedAt))
		}
	}
	return l.finish(run)
}

func (l *loader) importWire(p *play.PartnerReport) (map[string]string, error) {
	entries := map[string]play.WireEntry{}
	for _, scenario := range p.Scenarios {
		for _, c := range scenario.Conversations {
			for _, entry := range c.Wire {
				entries[entry.MessageID] = entry
			}
		}
	}
	ordered := make([]play.WireEntry, 0, len(entries))
	for _, entry := range entries {
		ordered = append(ordered, entry)
	}
	slices.SortFunc(ordered, func(a, b play.WireEntry) int {
		return cmp.Or(a.At.Compare(b.At), cmp.Compare(a.MessageID, b.MessageID))
	})
	ids := map[string]string{}
	for index := range ordered {
		entry := &ordered[index]
		direction := journal.ToTrenova
		if entry.From == "trenova" {
			direction = journal.FromTrenova
		}
		m, err := l.server.Import(p.Name, &sim.Imported{
			Direction: direction,
			Purpose:   journal.Purpose(entry.Purpose),
			State:     journal.State(entry.State),
			At:        entry.At,
			Raw:       entry.X12,
			Variation: entry.Variation,
			Error:     entry.Error,
		})
		if err != nil {
			return nil, err
		}
		ids[entry.MessageID] = m.ID
		l.result.Messages++
	}
	return ids, nil
}

func remap(matches []runlog.Match, ids map[string]string) []runlog.Match {
	out := make([]runlog.Match, 0, len(matches))
	for _, match := range matches {
		if id, ok := ids[match.MessageID]; ok {
			match.MessageID = id
			out = append(out, match)
		}
	}
	return out
}

func (l *loader) conform(report *conform.Report) (Result, error) {
	l.result.Kind = runlog.KindConform
	if l.server.Runs().Has(runlog.SourceReport, l.file, l.base) {
		l.result.Already = true
		return l.result, nil
	}
	finished := l.base
	run := &runlog.Run{
		Kind:       runlog.KindConform,
		Seed:       report.Seed,
		File:       l.file,
		Status:     statusOf(report.Passed()),
		StartedAt:  l.base,
		FinishedAt: &finished,
	}
	for _, p := range report.Partners {
		if !l.served(p.Name) {
			continue
		}
		for _, f := range p.Failures {
			run.Scenarios = append(run.Scenarios, failedSetup(p.Name, f.Message,
				"edi-sim conform --partner "+p.Name, l.base))
		}
		for _, c := range p.Cases {
			scenario, err := l.conformCase(p.Name, c)
			if err != nil {
				return l.result, err
			}
			run.Scenarios = append(run.Scenarios, scenario)
		}
	}
	return l.finish(run)
}

func (l *loader) conformCase(name string, c *conform.Case) (*runlog.Scenario, error) {
	scenario := &runlog.Scenario{
		Partner: name,
		Slug:    fmt.Sprintf("seed-%d", c.Seed),
		Name: fmt.Sprintf(
			"%s · seed %d",
			firstNonEmpty(c.Title, c.Variation, "Generated case"),
			c.Seed,
		),
		Seed:      c.Seed,
		Status:    statusOf(len(c.Failures) == 0),
		Replay:    fmt.Sprintf("edi-sim conform --replay %s:%d", name, c.Seed),
		StartedAt: l.base,
		Keys:      []string{},
	}
	for index, step := range c.Steps {
		logged := runlog.Step{
			Index:  index,
			Kind:   "conform",
			Text:   firstNonEmpty(step.Text, strings.TrimSpace(step.Name+" "+step.Set)),
			Status: conformStatus(step.Status),
			Note:   firstNonEmpty(step.Note, step.Status),
			Why:    whyOf(c, step),
			Ended:  l.at(),
		}
		if step.X12 != "" {
			direction, state := journal.FromTrenova, journal.StateReceived
			if step.Direction == "inbound" {
				direction, state = journal.ToTrenova, journal.StateDelivered
			}
			m, err := l.server.Import(name, &sim.Imported{
				Direction: direction,
				Purpose:   journal.PurposeDocument,
				State:     state,
				At:        logged.Ended,
				Raw:       step.X12,
				Variation: c.Variation,
			})
			if err != nil {
				return nil, err
			}
			l.result.Messages++
			logged.Matches = []runlog.Match{{MessageID: m.ID}}
			for _, key := range m.Keys {
				if !slices.Contains(scenario.Keys, key) {
					scenario.Keys = append(scenario.Keys, key)
				}
			}
		}
		if event, ok := conformEvent(step, logged.Ended); ok {
			logged.Events = []runlog.Event{event}
		}
		scenario.Steps = append(scenario.Steps, logged)
	}
	return scenario, nil
}

func (l *loader) at() time.Time {
	l.tick++
	return l.base.Add(time.Duration(l.tick) * time.Millisecond)
}

func conformEvent(step *conform.Step, at time.Time) (runlog.Event, bool) {
	switch step.Status {
	case conform.StatusHeld:
		return runlog.Event{At: at, Label: "Held by a rule", Detail: step.Set,
			Tone: runlog.ToneHold}, true
	case conform.StatusDeclined:
		return runlog.Event{At: at, Label: "Declined by a rule", Detail: step.Set,
			Tone: runlog.ToneTrenova}, true
	case conform.StatusNotSent:
		return runlog.Event{At: at, Label: "Not sent", Detail: step.Set,
			Tone: runlog.ToneTrenova}, true
	}
	return runlog.Event{}, false
}

func conformStatus(status string) string {
	switch status {
	case conform.StatusFailed:
		return runlog.StatusFailed
	case conform.StatusSkipped, conform.StatusNotSent:
		return runlog.StatusSkipped
	}
	return runlog.StatusPassed
}

func whyOf(c *conform.Case, step *conform.Step) string {
	var parts []string
	for _, f := range c.Failures {
		if f.Step != step.Name || f.Set != step.Set {
			continue
		}
		text := f.Message
		if f.Rule != "" {
			text = f.Rule + ": " + text
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "; ")
}

func failedSetup(name, why, replay string, at time.Time) *runlog.Scenario {
	return &runlog.Scenario{
		Partner:   name,
		Slug:      "setup",
		Name:      "Set " + name + " up",
		Status:    runlog.StatusFailed,
		Replay:    replay,
		StartedAt: at,
		Keys:      []string{},
		Steps: []runlog.Step{{
			Kind:   "setup",
			Text:   "check and certify the partner's Lading project",
			Status: runlog.StatusFailed,
			Why:    why,
			Ended:  at,
		}},
	}
}

func (l *loader) finish(run *runlog.Run) (Result, error) {
	if len(run.Scenarios) > runlog.MaxScenarios {
		run.Scenarios = run.Scenarios[len(run.Scenarios)-runlog.MaxScenarios:]
	}
	l.result.Scenarios = len(run.Scenarios)
	if _, err := l.server.Runs().Load(run); err != nil {
		return l.result, err
	}
	return l.result, nil
}

func statusOf(passed bool) string {
	if passed {
		return runlog.StatusPassed
	}
	return runlog.StatusFailed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
