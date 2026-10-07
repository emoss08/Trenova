package aituneup

import (
	"math"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentshadow"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	secondsPerDay  = 24 * 60 * 60
	secondsPerWeek = 7 * secondsPerDay
)

type ReadyTool struct {
	AgentDefinitionID pulid.ID
	ToolName          string
	From              agent.AutonomyTier
	To                agent.AutonomyTier
	Streak            int
	Approvals         int
	Rejections        int
	Since             int64
}

type ProviderTaskUsage struct {
	ProviderID pulid.ID
	Task       aiprovider.Task
	Calls      int
	Failed     int
	Rescued    int
}

type Inputs struct {
	Now            int64
	EarnedAutonomy bool
	Paused         bool
	Agents         []*agentdefinition.Definition
	Ready          []ReadyTool
	Shadow         map[pulid.ID]*agentshadow.Report
	Enabled        []*aiprovider.Provider
	Usage          []ProviderTaskUsage
}

func Suggest(in *Inputs) []*TuneUp {
	agents := make(map[pulid.ID]*agentdefinition.Definition, len(in.Agents))
	for _, definition := range in.Agents {
		agents[definition.ID] = definition
	}

	out := make([]*TuneUp, 0)
	out = append(out, raiseToolTiers(in, agents)...)
	out = append(out, reorderProviders(in)...)
	out = append(out, leaveShadow(in)...)
	out = append(out, assignTasks(in)...)
	out = append(out, turnOffIdle(in)...)
	for _, tuneUp := range out {
		tuneUp.ComputedAt = in.Now
		tuneUp.Status = StatusOpen
	}
	return out
}

func raiseToolTiers(in *Inputs, agents map[pulid.ID]*agentdefinition.Definition) []*TuneUp {
	if in.EarnedAutonomy {
		return nil
	}
	out := make([]*TuneUp, 0, len(in.Ready))
	for _, ready := range in.Ready {
		definition, ok := agents[ready.AgentDefinitionID]
		if !ok || !definition.Enabled {
			continue
		}
		out = append(out, &TuneUp{
			Kind:              KindRaiseToolTier,
			Fingerprint:       Fingerprint(KindRaiseToolTier, ready.AgentDefinitionID.String(), ready.ToolName),
			AgentDefinitionID: ready.AgentDefinitionID,
			ToolName:          ready.ToolName,
			Evidence: Evidence{
				FromTier:         ready.From,
				ToTier:           ready.To,
				Streak:           ready.Streak,
				Approvals:        ready.Approvals,
				Rejections:       ready.Rejections,
				ApprovalsPerWeek: perWeek(ready.Approvals, ready.Since, in.Now),
			},
		})
	}
	return out
}

func perWeek(count int, since, now int64) float64 {
	weeks := math.Max(float64(now-since)/secondsPerWeek, 1)
	return math.Round(float64(count)/weeks*10) / 10
}

type providerPair struct {
	failing pulid.ID
	rescuer pulid.ID
}

func reorderProviders(in *Inputs) []*TuneUp {
	usage := make(map[pulid.ID]map[aiprovider.Task]ProviderTaskUsage, len(in.Enabled))
	for _, row := range in.Usage {
		if usage[row.ProviderID] == nil {
			usage[row.ProviderID] = map[aiprovider.Task]ProviderTaskUsage{}
		}
		usage[row.ProviderID][row.Task] = row
	}

	evidence := map[providerPair]*Evidence{}
	order := make([]providerPair, 0)
	for _, task := range aiprovider.AllTasks() {
		chain := chainFor(in.Enabled, task)
		if len(chain) < 2 {
			continue
		}
		first := usage[chain[0].ID][task]
		if !failingBadly(first) {
			continue
		}
		rescuer := chain[1]
		caught := usage[rescuer.ID][task]
		if !rescuingWell(caught) {
			continue
		}
		pair := providerPair{failing: chain[0].ID, rescuer: rescuer.ID}
		found, ok := evidence[pair]
		if !ok {
			found = &Evidence{}
			evidence[pair] = found
			order = append(order, pair)
		}
		found.Failed += first.Failed
		found.Calls += caught.Calls
		found.Rescued += caught.Rescued
		found.Tasks = append(found.Tasks, task)
	}

	out := make([]*TuneUp, 0, len(order))
	for _, pair := range order {
		out = append(out, &TuneUp{
			Kind:            KindReorderProviders,
			Fingerprint:     Fingerprint(KindReorderProviders, pair.rescuer.String(), pair.failing.String()),
			ProviderID:      pair.rescuer,
			OtherProviderID: pair.failing,
			Evidence:        *evidence[pair],
		})
	}
	return out
}

func chainFor(enabled []*aiprovider.Provider, task aiprovider.Task) []*aiprovider.Provider {
	chain := make([]*aiprovider.Provider, 0, len(enabled))
	for _, provider := range enabled {
		if ok, _ := provider.CanServeTask(task); ok {
			chain = append(chain, provider)
		}
	}
	return chain
}

func failingBadly(row ProviderTaskUsage) bool {
	return row.Failed >= ReorderMinFailures && rate(row.Failed, row.Calls) >= ReorderMinFailureRate
}

func rescuingWell(row ProviderTaskUsage) bool {
	return row.Rescued >= ReorderMinRescues && rate(row.Failed, row.Calls) <= ReorderMaxRescuerFailureRate
}

func rate(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

func leaveShadow(in *Inputs) []*TuneUp {
	if in.Paused {
		return nil
	}
	out := make([]*TuneUp, 0)
	for _, definition := range in.Agents {
		if !definition.Enabled || !definition.ShadowMode {
			continue
		}
		report, ok := in.Shadow[definition.ID]
		if !ok || report.Recorded < ShadowMinRecorded {
			continue
		}
		matchRate := report.MatchRate()
		if matchRate == nil || *matchRate < ShadowMinMatchRate {
			continue
		}
		out = append(out, &TuneUp{
			Kind:              KindLeaveShadow,
			Fingerprint:       Fingerprint(KindLeaveShadow, definition.ID.String()),
			AgentDefinitionID: definition.ID,
			Evidence: Evidence{
				Recorded:  report.Recorded,
				MatchRate: math.Round(*matchRate*1000) / 1000,
				WouldFail: report.WouldFail,
			},
		})
	}
	return out
}

func assignTasks(in *Inputs) []*TuneUp {
	if len(in.Enabled) == 0 {
		return nil
	}
	out := make([]*TuneUp, 0)
	for _, task := range aiprovider.AllTasks() {
		if aiprovider.RouteFor(in.Enabled, task) != nil {
			continue
		}
		assignee := firstAbleTo(in.Enabled, task)
		if assignee == nil {
			continue
		}
		out = append(out, &TuneUp{
			Kind:        KindAssignTask,
			Fingerprint: Fingerprint(KindAssignTask, string(task), assignee.ID.String()),
			ProviderID:  assignee.ID,
			Task:        task,
			Evidence:    Evidence{Model: assignee.Model},
		})
	}
	return out
}

func firstAbleTo(enabled []*aiprovider.Provider, task aiprovider.Task) *aiprovider.Provider {
	for _, provider := range enabled {
		candidate := *provider
		candidate.Tasks = append(slices.Clone(provider.Tasks), task)
		if ok, _ := candidate.CanServeTask(task); ok {
			return provider
		}
	}
	return nil
}

func turnOffIdle(in *Inputs) []*TuneUp {
	cutoff := in.Now - IdleDays*secondsPerDay
	out := make([]*TuneUp, 0)
	for _, definition := range in.Agents {
		if !definition.Enabled || definition.IsSystem() ||
			definition.TriggerMode != agentdefinition.TriggerChat || definition.CreatedAt > cutoff {
			continue
		}
		idleSince := definition.CreatedAt
		if definition.LastRunAt != nil {
			idleSince = *definition.LastRunAt
		}
		if idleSince > cutoff {
			continue
		}
		out = append(out, &TuneUp{
			Kind:              KindTurnOffIdleAgent,
			Fingerprint:       Fingerprint(KindTurnOffIdleAgent, definition.ID.String()),
			AgentDefinitionID: definition.ID,
			Evidence: Evidence{
				LastRunAt: definition.LastRunAt,
				IdleSince: idleSince,
				Tools:     len(definition.ToolNames),
			},
		})
	}
	return out
}
