package benchresult

import (
	"slices"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/deskbench/benchstats"
)

type CaseMetrics struct {
	Usage            benchrecord.Usage `json:"usage"`
	ModelCalls       int               `json:"modelCalls"`
	ToolCalls        int               `json:"toolCalls"`
	RefusedCalls     int               `json:"refusedCalls"`
	RefusedByVerdict map[string]int    `json:"refusedByVerdict,omitempty"`
	ReasksByGuard    map[string]int    `json:"reasksByGuard,omitempty"`
	FirstTokenMs     *float64          `json:"firstTokenMs,omitempty"`
	Seconds          float64           `json:"seconds"`
	PeakPromptTokens int               `json:"peakPromptTokens"`
	LaterPrompt      int               `json:"laterPromptTokens,omitempty"`
	LaterCacheRead   int               `json:"laterCacheReadTokens,omitempty"`
	ToolTurns        int               `json:"toolTurns,omitempty"`
	FindFirstTurns   int               `json:"findFirstTurns,omitempty"`
	Selection        *Selection        `json:"selection,omitempty"`
	Retrieval        *Retrieval        `json:"retrieval,omitempty"`
}

type Selection struct {
	Precision      *float64 `json:"precision,omitempty"`
	Recall         float64  `json:"recall"`
	FirstCorrect   int      `json:"firstCorrect"`
	ForbiddenCalls int      `json:"forbiddenCalls"`
	ScoredCalls    int      `json:"scoredCalls"`
	Derived        bool     `json:"derived,omitempty"`
}

type Retrieval struct {
	PreselectionTurns  int      `json:"preselectionTurns"`
	PreselectionRecall *float64 `json:"preselectionRecall,omitempty"`
	FindAnswers        int      `json:"findAnswers"`
	FindHits           int      `json:"findHits"`
	FindRecall         *float64 `json:"findRecall,omitempty"`
	FindUnionRecall    *float64 `json:"findUnionRecall,omitempty"`
}

func taskTurns(result *CaseResult) []*benchrecord.TurnRecord {
	turns := make([]*benchrecord.TurnRecord, 0, len(result.Steps))
	for _, step := range result.Steps {
		turns = append(turns, step.Turns...)
	}

	return turns
}

func MeasureCase(result *CaseResult) CaseMetrics {
	turns := taskTurns(result)
	metrics := CaseMetrics{
		RefusedByVerdict: map[string]int{},
		ReasksByGuard:    map[string]int{},
		Seconds:          result.FinishedAt.Sub(result.StartedAt).Seconds(),
	}
	firstTokens := make([]float64, 0, len(turns))
	for _, turn := range turns {
		metrics.Usage.Add(&turn.Usage)
		metrics.ModelCalls += len(turn.Calls)
		metrics.ToolCalls += len(turn.Tools)
		for _, call := range turn.Tools {
			if call.Refused() {
				metrics.RefusedCalls++
				metrics.RefusedByVerdict[benchrecord.VerdictLabel(call)]++
			}
		}
		own := 0
		for idx := range turn.Calls {
			call := &turn.Calls[idx]
			if call.Reask != "" {
				metrics.ReasksByGuard[call.Reask]++
			}
			prompt := call.InputTokens + call.CacheRead + call.CacheWrite
			metrics.PeakPromptTokens = max(metrics.PeakPromptTokens, prompt)
			if call.Delegate != "" || call.Kind != benchrecord.ModelCallChat {
				continue
			}
			if own > 0 {
				metrics.LaterPrompt += prompt
				metrics.LaterCacheRead += call.CacheRead
			}
			own++
		}
		if first := firstOwnTool(turn); first != "" {
			metrics.ToolTurns++
			if first == findToolsName {
				metrics.FindFirstTurns++
			}
		}
		if turn.FirstTokenMs != nil {
			firstTokens = append(firstTokens, float64(*turn.FirstTokenMs))
		}
	}
	if len(firstTokens) > 0 {
		first := benchstats.MeanOf(firstTokens)
		metrics.FirstTokenMs = &first
	}
	metrics.Selection = scoreSelection(result.Scenario, turns)
	if result.Passed {
		metrics.Retrieval = scoreRetrieval(turns)
	}

	return metrics
}

func scoreSelection(scenario *benchscenario.Scenario, turns []*benchrecord.TurnRecord) *Selection {
	if scenario == nil || len(scenario.ExpectedTools) == 0 {
		return nil
	}

	tally := &selectionTally{
		scenario:  scenario,
		selection: &Selection{Derived: scenario.LabelsDerived},
		satisfied: make([]bool, len(scenario.ExpectedTools)),
		called:    make([]string, 0, 8),
	}
	for _, turn := range turns {
		for _, call := range turn.Tools {
			tally.observe(call.Name)
		}
	}

	return tally.score()
}

type selectionTally struct {
	scenario  *benchscenario.Scenario
	selection *Selection
	satisfied []bool
	called    []string
}

func (t *selectionTally) observe(name string) {
	expected := benchscenario.Labelled(name, t.scenario.ExpectedTools)
	forbidden := benchscenario.Labelled(name, t.scenario.ForbiddenTools)
	if forbidden {
		t.selection.ForbiddenCalls++
	}
	if benchscenario.Plumbing(name) && !expected && !forbidden {
		return
	}
	t.selection.ScoredCalls++
	if !slices.Contains(t.called, name) {
		t.called = append(t.called, name)
	}
	for idx, spec := range t.scenario.ExpectedTools {
		if slices.Contains(benchscenario.Alternatives(spec), name) {
			t.satisfied[idx] = true
		}
	}
	if expected && t.selection.FirstCorrect == 0 {
		t.selection.FirstCorrect = t.selection.ScoredCalls
	}
}

func (t *selectionTally) score() *Selection {
	selection, called, scenario := t.selection, t.called, t.scenario
	hits := 0
	for _, ok := range t.satisfied {
		if ok {
			hits++
		}
	}
	selection.Recall = float64(hits) / float64(len(t.satisfied))
	if len(called) > 0 {
		correct := 0
		for _, name := range called {
			if benchscenario.Labelled(name, scenario.ExpectedTools) {
				correct++
			}
		}
		precision := float64(correct) / float64(len(called))
		selection.Precision = &precision
	}

	return selection
}

func scoreRetrieval(turns []*benchrecord.TurnRecord) *Retrieval {
	retrieval := &Retrieval{}
	preselection := make([]float64, 0, len(turns))
	answers := make([]float64, 0, len(turns))
	unions := make([]float64, 0, len(turns))
	for _, turn := range turns {
		used := usedDomainTools(turn)
		if len(used) == 0 {
			continue
		}
		opening := openingTools(turn)
		missing := slices.DeleteFunc(slices.Clone(used), func(name string) bool {
			return slices.Contains(opening, name)
		})
		disclosed := false
		found := make([]string, 0, 8)
		for idx := range turn.Disclosures {
			disclosure := &turn.Disclosures[idx]
			if disclosure.DelegateCallID != "" {
				continue
			}
			switch disclosure.Kind {
			case benchrecord.DisclosurePreselected:
				disclosed = true
			case benchrecord.DisclosureFound:
				retrieval.FindAnswers++
				if slices.ContainsFunc(disclosure.Ranked, func(name string) bool {
					return slices.Contains(missing, name)
				}) {
					retrieval.FindHits++
				}
				if len(missing) > 0 {
					answers = append(answers, coverage(missing, disclosure.Ranked))
				}
				found = append(found, disclosure.Ranked...)
			}
		}
		if disclosed {
			retrieval.PreselectionTurns++
			preselection = append(preselection, coverage(used, opening))
		}
		if len(missing) > 0 && len(found) > 0 {
			unions = append(unions, coverage(missing, found))
		}
	}
	if len(preselection) > 0 {
		value := benchstats.MeanOf(preselection)
		retrieval.PreselectionRecall = &value
	}
	if len(answers) > 0 {
		value := benchstats.MeanOf(answers)
		retrieval.FindRecall = &value
	}
	if len(unions) > 0 {
		value := benchstats.MeanOf(unions)
		retrieval.FindUnionRecall = &value
	}
	if retrieval.PreselectionTurns == 0 && retrieval.FindAnswers == 0 {
		return nil
	}

	return retrieval
}

func usedDomainTools(turn *benchrecord.TurnRecord) []string {
	used := make([]string, 0, 8)
	for _, call := range turn.Tools {
		if call.DelegateCallID != "" || benchscenario.Plumbing(call.Name) {
			continue
		}
		if !slices.Contains(used, call.Name) {
			used = append(used, call.Name)
		}
	}

	return used
}

const findToolsName = "find_tools"

func firstOwnTool(turn *benchrecord.TurnRecord) string {
	for _, call := range turn.Tools {
		if call.DelegateCallID == "" {
			return call.Name
		}
	}

	return ""
}

func openingTools(turn *benchrecord.TurnRecord) []string {
	for _, call := range turn.ModelCalls {
		if call.Chat == nil || call.Attribution.DelegateCallID != "" {
			continue
		}
		names := make([]string, 0, len(call.Chat.Tools))
		for idx := range call.Chat.Tools {
			if !call.Chat.Tools[idx].Deferred {
				names = append(names, call.Chat.Tools[idx].Name)
			}
		}

		return names
	}

	return nil
}

func coverage(want, got []string) float64 {
	if len(want) == 0 {
		return 0
	}
	hits := 0
	for _, name := range want {
		if slices.Contains(got, name) {
			hits++
		}
	}

	return float64(hits) / float64(len(want))
}
