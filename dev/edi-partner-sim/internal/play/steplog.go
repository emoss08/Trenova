package play

import (
	"context"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/scenario"
)

func (s *StepReport) LogStep() runlog.Step {
	return runlog.Step{
		Index:   s.Index,
		Kind:    s.Kind,
		Text:    s.Text,
		Status:  s.Status,
		Note:    s.Note,
		Why:     s.Why,
		Ended:   s.Ended,
		Keys:    s.Keys,
		Matches: s.Matches,
		Events:  s.Events,
		Gap:     s.Gap,
	}
}

func (s *ScenarioReport) LogScenario(partner string) *runlog.Scenario {
	out := &runlog.Scenario{
		Partner:   partner,
		Slug:      s.Slug,
		Name:      s.Name,
		Seed:      s.Seed,
		Status:    s.Status,
		Replay:    s.Replay,
		StartedAt: s.StartedAt,
		Took:      time.Duration(s.Took).Seconds(),
		Keys:      []string{},
		Steps:     make([]runlog.Step, 0, len(s.Steps)),
	}
	for _, c := range s.Conversations {
		if c.Key != "" {
			out.Keys = append(out.Keys, c.Key)
		}
	}
	for _, step := range s.Steps {
		out.Steps = append(out.Steps, step.LogStep())
	}
	return out
}

func (r *caseRun) post(ctx context.Context, status string) {
	scenario := r.report.LogScenario(r.party.Name())
	scenario.Status = status
	for _, c := range r.order {
		if c.Key != "" {
			scenario.Keys = append(scenario.Keys, c.Key)
		}
	}
	r.env.PostScenario(ctx, scenario)
}

func (r *caseRun) record(
	ctx context.Context,
	step *scenario.Step,
	report *StepReport,
	err error,
) {
	matches, events, gap := r.trace.Take()
	keys := r.keysOf(step)
	if err != nil {
		report.Why = err.Error()
		kind := step.Kind()
		if len(matches) == 0 && len(events) == 0 && kind != scenario.KindScript &&
			kind != scenario.KindAcks {
			key := ""
			if len(keys) == 1 {
				key = keys[0]
			}
			events = append(events, runlog.Event{
				Key:   key,
				At:    time.Now().UTC(),
				Label: report.Text,
				Tone:  runlog.ToneBad,
			})
		}
	}
	report.Ended = time.Now().UTC()
	report.Keys, report.Matches, report.Events, report.Gap = keys, matches, events, gap
	logged := report.LogStep()
	r.env.PostStep(ctx, r.party.Name(), r.scenario.Slug, &logged)
}

func (r *caseRun) keysOf(step *scenario.Step) []string {
	var keys []string
	for _, c := range r.group(aboutOf(step)) {
		if c.Key != "" {
			keys = append(keys, c.Key)
		}
	}
	return keys
}

func aboutOf(step *scenario.Step) string {
	switch step.Kind() {
	case scenario.KindSend:
		return firstNonEmpty(step.Send.As, step.Send.Of)
	case scenario.KindExpect:
		_, about := step.Expect.Subject()
		return about
	case scenario.KindTrenova:
		_, about := step.Trenova.Verb()
		return about
	case scenario.KindAdvance:
		return step.Advance.About
	case scenario.KindAcks:
		return step.Acks.About
	}
	return ""
}
