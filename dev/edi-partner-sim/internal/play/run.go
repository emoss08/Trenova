package play

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/scenario"
)

func Run(ctx context.Context, o *Options) (*Report, error) {
	env, err := Open(ctx, o)
	if err != nil {
		return nil, err
	}
	defer env.Close()
	return env.RunAll(ctx, o)
}

func (e *Env) RunAll(ctx context.Context, o *Options) (*Report, error) {
	definitions, err := e.Definitions(o.Partners)
	if err != nil {
		return nil, err
	}
	plans := make([]plan, 0, len(definitions))
	for _, def := range definitions {
		p, planErr := planFor(def, o.Scenarios)
		if planErr != nil {
			return nil, planErr
		}
		if len(p.scenarios) == 0 && len(o.Scenarios) > 0 {
			continue
		}
		p.keepGoing = o.KeepGoing
		plans = append(plans, p)
	}
	if len(plans) == 0 {
		return nil, fmt.Errorf("no partner has a scenario named %s",
			strings.Join(o.Scenarios, ", "))
	}
	if needsCLI(plans) {
		if err = e.CLI.Check(); err != nil {
			return nil, err
		}
	}
	started := time.Now()
	report := &Report{
		Scale:     e.scale,
		Run:       e.Run,
		API:       o.API,
		Sim:       o.Sim,
		StartedAt: started.UTC(),
		Partners:  []*PartnerReport{},
	}
	e.BeginRun(ctx, runlog.KindRun, 0)
	stopped := false
	for _, p := range plans {
		if stopped || ctx.Err() != nil {
			break
		}
		partnerReport := e.runPartner(ctx, &p)
		report.Partners = append(report.Partners, partnerReport)
		if !o.KeepGoing && !partnerPassed(partnerReport) {
			stopped = true
		}
	}
	report.Took = Duration(time.Since(started))
	report.tally()
	e.EndRun(ctx, report.Passed())
	if err = ctx.Err(); err != nil {
		return report, err
	}
	return report, nil
}

const happyPathSlug = "happy-path"

type plan struct {
	def       *partner.Definition
	scenarios []indexed
	keepGoing bool
}

type indexed struct {
	index    int
	scenario *scenario.Scenario
}

func planFor(def *partner.Definition, only []string) (plan, error) {
	all, err := scenario.Load(def.Dir)
	if err != nil {
		return plan{}, err
	}
	if len(all) > MaxScenarios {
		return plan{}, fmt.Errorf("%s has %d scenarios; a run gives each partner room for %d",
			def.Name(), len(all), MaxScenarios)
	}
	p := plan{def: def}
	for index, s := range all {
		if err = checkNeeds(def, s); err != nil {
			return plan{}, err
		}
		if len(only) > 0 && !slices.Contains(only, s.Slug) && !slices.Contains(only, s.Name) {
			continue
		}
		p.scenarios = append(p.scenarios, indexed{index: index, scenario: s})
	}
	return p, nil
}

func needsCLI(plans []plan) bool {
	for _, p := range plans {
		for _, s := range p.scenarios {
			if slices.ContainsFunc(s.scenario.Steps, func(step scenario.Step) bool {
				return step.Advance != nil
			}) {
				return true
			}
		}
	}
	return false
}

func (e *Env) runPartner(ctx context.Context, p *plan) *PartnerReport {
	party := e.Provision(ctx, p.def)
	report := &PartnerReport{
		Name:      p.def.Name(),
		Role:      string(p.def.Partner.Role),
		Provision: party.Setup,
		Scenarios: []*ScenarioReport{},
	}
	if party.Setup.Failure != "" {
		failure := Failure{
			Step:    -1,
			Source:  SourceSetup,
			Rule:    "provision",
			Message: party.Setup.Failure,
			Replay:  "edi-sim run --partner " + p.def.Name(),
		}
		report.Failures = append(report.Failures, failure)
		e.postProvisionFailure(ctx, p.def.Name(), &failure)
		return report
	}
	failed := false
	for _, s := range p.scenarios {
		if ctx.Err() != nil {
			break
		}
		if failed && !p.keepGoing {
			skipped := &ScenarioReport{
				Name:      s.scenario.Name,
				File:      s.scenario.File,
				Slug:      s.scenario.Slug,
				Seed:      SeedOf(e.Run, uint64(s.index)*scenarioSeedStride),
				Status:    StatusSkipped,
				StartedAt: time.Now().UTC(),
				Steps:     []*StepReport{},
			}
			report.Scenarios = append(report.Scenarios, skipped)
			e.PostScenario(ctx, skipped.LogScenario(p.def.Name()))
			e.finish(p.def.Name(), skipped)
			continue
		}
		result := e.RunScenario(ctx, party, s.scenario, s.index)
		report.Scenarios = append(report.Scenarios, result)
		e.finish(p.def.Name(), result)
		if result.Status == StatusFailed {
			failed = true
		}
	}
	return report
}

func (e *Env) postProvisionFailure(ctx context.Context, name string, f *Failure) {
	scenario := &runlog.Scenario{
		Partner:   name,
		Slug:      "provision",
		Name:      "Provision " + name + " in Trenova",
		Status:    runlog.StatusFailed,
		Replay:    f.Replay,
		StartedAt: time.Now().UTC(),
		Keys:      []string{},
		Steps: []runlog.Step{{
			Kind:   "provision",
			Text:   "provision the partner, its profile and its Lading project",
			Status: runlog.StatusFailed,
			Why:    f.Message,
			Ended:  time.Now().UTC(),
		}},
	}
	e.PostScenario(ctx, scenario)
}

func partnerPassed(p *PartnerReport) bool {
	if len(p.Failures) > 0 {
		return false
	}
	return !slices.ContainsFunc(p.Scenarios, func(s *ScenarioReport) bool {
		return s.Status == StatusFailed
	})
}

func checkNeeds(def *partner.Definition, s *scenario.Scenario) error {
	for index := range s.Steps {
		send := s.Steps[index].Send
		if send == nil || send.Load == nil {
			continue
		}
		if err := def.CheckNeed(send.Set, send.Load); err != nil {
			return fmt.Errorf("%s: steps[%d].send.load: %w", s.File, index, err)
		}
	}
	return nil
}

func happyNeed(def *partner.Definition) *partner.LoadNeed {
	all, err := scenario.Load(def.Dir)
	if err != nil {
		return nil
	}
	for _, s := range all {
		if s.Slug != happyPathSlug {
			continue
		}
		for index := range s.Steps {
			if send := s.Steps[index].Send; send != nil {
				return send.Load
			}
		}
	}
	return nil
}

func (e *Env) finish(partner string, report *ScenarioReport) {
	if e.finished != nil {
		e.finished(partner, report)
	}
}
