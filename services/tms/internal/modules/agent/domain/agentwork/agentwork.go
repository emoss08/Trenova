package agentwork

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Status string

const (
	StatusWorking  = Status("Working")
	StatusNeedsYou = Status("NeedsYou")
	StatusDone     = Status("Done")
)

func (s Status) String() string { return string(s) }

func (s Status) IsValid() bool {
	switch s {
	case StatusWorking, StatusNeedsYou, StatusDone:
		return true
	default:
		return false
	}
}

type Step string

const (
	StepRead    = Step("Read")
	StepDecide  = Step("Decide")
	StepApprove = Step("Approve")
)

func (s Step) String() string { return string(s) }

func (s Step) IsValid() bool {
	switch s {
	case StepRead, StepDecide, StepApprove:
		return true
	default:
		return false
	}
}

func Steps() []Step {
	return []Step{StepRead, StepDecide, StepApprove}
}

// Work is where the agent an item was handed to stands on it, read from the
// item and the run it was handed to: working, waiting on a person, or done,
// and how far through its steps. It is a reading, not something an item
// stores; the item keeps only which agent and which run.
type Work struct {
	AgentID    pulid.ID
	RunID      pulid.ID
	HandedAt   int64
	HandedByID pulid.ID
	Status     Status
	Step       int
	Steps      []Step
	Note       string
	Failed     bool
}

// From reads the work on an item from the run it was handed to. Nil when
// the item has no agent. A run that is gone, or is not the one the item
// names, reads as done.
func From(item *watchtower.Item, run *agent.AgentRun) *Work {
	if !item.HasAgent() {
		return nil
	}

	steps := Steps()
	work := &Work{
		AgentID:    item.AgentDefinitionID,
		RunID:      item.AgentRunID,
		HandedByID: item.AgentHandedByID,
		Steps:      steps,
	}
	if item.AgentHandedAt != nil {
		work.HandedAt = *item.AgentHandedAt
	}
	if run == nil || run.ID != item.AgentRunID {
		work.Status = StatusDone
		work.Step = len(steps)
		return work
	}

	switch run.Status {
	case agent.RunStatusDiagnosing:
		work.Status = StatusWorking
		work.Step = 1
	case agent.RunStatusAwaitingDecision:
		work.Status = StatusNeedsYou
		work.Step = 2
		work.Note = strings.TrimSpace(run.Summary)
	case agent.RunStatusFailed:
		work.Status = StatusNeedsYou
		work.Step = 1
		work.Failed = true
		work.Note = failureNote(run)
	case agent.RunStatusCompleted, agent.RunStatusShadowCompleted:
		work.Status = StatusDone
		work.Step = len(steps)
		work.Note = strings.TrimSpace(run.Summary)
	default:
		if run.Status.IsWorking() {
			work.Status = StatusWorking
			work.Step = 0
			break
		}
		work.Status = StatusDone
		work.Step = len(steps)
	}

	return work
}

// AnsweredUnchanged reports an item whose agent finished its run and that
// says nothing new since: the agent read this description and gave its
// answer, so running it again would pay for the same answer. A failed run
// answered nothing and may be tried again.
func AnsweredUnchanged(item *watchtower.Item, run *agent.AgentRun) bool {
	if !item.HasAgent() || run == nil || run.ID != item.AgentRunID || item.AgentHandedAt == nil {
		return false
	}
	if run.Status != agent.RunStatusCompleted && run.Status != agent.RunStatusShadowCompleted {
		return false
	}

	return item.DescribedAt <= *item.AgentHandedAt
}

func (w *Work) Active() bool {
	return w != nil && w.Status == StatusWorking
}

func failureNote(run *agent.AgentRun) string {
	reason, _, _ := strings.Cut(strings.TrimSpace(run.ErrorMessage), "\n")
	return stringutils.FirstNonEmpty(strings.TrimSpace(reason), strings.TrimSpace(run.Summary))
}
