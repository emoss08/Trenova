package watchtower

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type AgentStatus string

const (
	AgentStatusWorking  = AgentStatus("Working")
	AgentStatusNeedsYou = AgentStatus("NeedsYou")
	AgentStatusDone     = AgentStatus("Done")
)

func (s AgentStatus) String() string { return string(s) }

func (s AgentStatus) IsValid() bool {
	switch s {
	case AgentStatusWorking, AgentStatusNeedsYou, AgentStatusDone:
		return true
	default:
		return false
	}
}

type AgentStep string

const (
	AgentStepRead    = AgentStep("Read")
	AgentStepDecide  = AgentStep("Decide")
	AgentStepApprove = AgentStep("Approve")
)

func (s AgentStep) String() string { return string(s) }

func (s AgentStep) IsValid() bool {
	switch s {
	case AgentStepRead, AgentStepDecide, AgentStepApprove:
		return true
	default:
		return false
	}
}

func AgentSteps() []AgentStep {
	return []AgentStep{AgentStepRead, AgentStepDecide, AgentStepApprove}
}

type AgentWork struct {
	AgentID    pulid.ID
	RunID      pulid.ID
	HandedAt   int64
	HandedByID pulid.ID
	Status     AgentStatus
	Step       int
	Steps      []AgentStep
	Note       string
	Failed     bool
}

func (i *Item) HasAgent() bool {
	return i.AgentDefinitionID.IsNotNil() && i.AgentRunID.IsNotNil()
}

func (i *Item) AgentWorkFrom(run *agent.AgentRun) *AgentWork {
	if !i.HasAgent() {
		return nil
	}

	steps := AgentSteps()
	work := &AgentWork{
		AgentID:    i.AgentDefinitionID,
		RunID:      i.AgentRunID,
		HandedByID: i.AgentHandedByID,
		Steps:      steps,
	}
	if i.AgentHandedAt != nil {
		work.HandedAt = *i.AgentHandedAt
	}
	if run == nil || run.ID != i.AgentRunID {
		work.Status = AgentStatusDone
		work.Step = len(steps)
		return work
	}

	switch run.Status {
	case agent.RunStatusPending, agent.RunStatusGatheringContext:
		work.Status = AgentStatusWorking
		work.Step = 0
	case agent.RunStatusDiagnosing:
		work.Status = AgentStatusWorking
		work.Step = 1
	case agent.RunStatusAwaitingDecision:
		work.Status = AgentStatusNeedsYou
		work.Step = 2
		work.Note = strings.TrimSpace(run.Summary)
	case agent.RunStatusFailed:
		work.Status = AgentStatusNeedsYou
		work.Step = 1
		work.Failed = true
		work.Note = failureNote(run)
	case agent.RunStatusCompleted, agent.RunStatusShadowCompleted:
		work.Status = AgentStatusDone
		work.Step = len(steps)
		work.Note = strings.TrimSpace(run.Summary)
	default:
		work.Status = AgentStatusWorking
		work.Step = 0
	}

	return work
}

func (w *AgentWork) Active() bool {
	return w != nil && w.Status == AgentStatusWorking
}

func failureNote(run *agent.AgentRun) string {
	reason, _, _ := strings.Cut(strings.TrimSpace(run.ErrorMessage), "\n")
	return stringutils.FirstNonEmpty(strings.TrimSpace(reason), strings.TrimSpace(run.Summary))
}

func (i *Item) HandTo(definitionID, runID, handedBy pulid.ID, now int64) error {
	if err := i.requireOpen(); err != nil {
		return err
	}
	i.AgentDefinitionID = definitionID
	i.AgentRunID = runID
	i.AgentHandedByID = handedBy
	i.AgentHandedAt = &now
	if i.Status != StatusInProgress {
		i.Status = StatusInProgress
		i.SnoozedUntil = nil
		if i.AcknowledgedAt == nil {
			i.AcknowledgedAt = &now
		}
	}
	return nil
}
