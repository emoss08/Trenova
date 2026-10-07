package assistantjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentdryrun"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	AgentDryRunWorkflowName = "AgentDryRunWorkflow"
	// AssistantEventDryRunSteps is the last event of a dry run before its
	// end: each tool call with what it would have come to.
	AssistantEventDryRunSteps = "dry_run_steps"
	dryRunFailedMessage  = "The dry run could not finish. Try again in a moment."
	dryRunStoppedMessage = "The dry run took too long and was stopped."
	// dryRunTimeout bounds a dry run; a person is watching it.
	dryRunTimeout = 5 * time.Minute
)

// DryRunWorkflowIDFor names the execution carrying a dry run.
func DryRunWorkflowIDFor(runID pulid.ID) string {
	return "agent-dry-run:" + runID.String()
}

// AgentDryRunPayload is a draft agent and the message it is tried with. The
// draft travels whole: it is not saved, so there is nothing to read it back
// from.
type AgentDryRunPayload struct {
	temporaltype.BasePayload

	RunID      pulid.ID                    `json:"runId"`
	Actor      serviceports.RequestActor   `json:"actor"`
	Definition *agentdefinition.Definition `json:"definition"`
	Prompt     string                      `json:"prompt"`
}

// DryRunStepsEvent is the dry run's account of its tool calls.
type DryRunStepsEvent struct {
	Steps []agentdryrun.Step `json:"steps"`
	Reply string             `json:"reply"`
}

// StartDryRunWorkflow starts a dry run on the chat queue, where somebody is
// watching it arrive.
func StartDryRunWorkflow(
	ctx context.Context,
	workflows serviceports.WorkflowStarter,
	payload *AgentDryRunPayload,
) (client.WorkflowRun, error) {
	return workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                                       DryRunWorkflowIDFor(payload.RunID),
		TaskQueue:                                temporaltype.TaskQueueAgentChat.String(),
		WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
		WorkflowExecutionTimeout:                 dryRunTimeout + agentflow.DrainWindow,
		StaticSummary:                            "Agent dry run",
		Priority: temporal.Priority{
			PriorityKey: agentflow.PriorityInteractive,
			FairnessKey: payload.Actor.OrganizationID.String(),
		},
	}, AgentDryRunWorkflowName, payload)
}

// PrepareDryRunActivity opens the turn a draft agent would answer the message
// with.
func (a *Activities) PrepareDryRunActivity(
	ctx context.Context,
	payload *AgentDryRunPayload,
) (*assistantservice.DryRunPlan, error) {
	if payload.Actor.UserID.IsNil() {
		return nil, temporal.NewNonRetryableApplicationError(
			"this dry run arrived without the person who asked for it", errTypeNoActor, nil,
		)
	}

	plan, err := a.assistant.PrepareDryRun(ctx, &assistantservice.DryRunRequest{
		RunID:      payload.RunID,
		Definition: payload.Definition,
		Prompt:     payload.Prompt,
	}, &payload.Actor)
	if err != nil {
		if rejected(err) {
			return nil, temporal.NewNonRetryableApplicationError(err.Error(), errTypeRejected, err)
		}
		return nil, fmt.Errorf("prepare this dry run: %w", err)
	}

	return plan, nil
}

// AgentDryRunWorkflow runs a draft agent once against live data with every
// write simulated, and streams what it does. It writes nothing: no thread, no
// run, no proposal. Its last word before the end is each tool call with what
// it would have come to had the draft been saved.
func (w *Workflows) AgentDryRunWorkflow(
	ctx workflow.Context,
	payload *AgentDryRunPayload,
) error {
	stream, err := agentflow.HostStream(ctx)
	if err != nil {
		return err
	}

	ending := w.dryRun(ctx, stream, payload)
	stream.Publish(ctx, ending)
	stream.Close(ctx)

	return nil
}

func (w *Workflows) dryRun(
	ctx workflow.Context,
	stream *agentflow.Stream,
	payload *AgentDryRunPayload,
) temporaltype.StreamItem {
	var a *Activities
	var plan assistantservice.DryRunPlan
	options := prepareOptions
	options.Priority = temporal.Priority{
		PriorityKey: agentflow.PriorityInteractive,
		FairnessKey: payload.Actor.OrganizationID.String(),
	}
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, options), a.PrepareDryRunActivity, payload,
	).Get(ctx, &plan); err != nil {
		return dryRunError(ctx, err)
	}

	opening := plan.Opening()
	stream.Publish(ctx, temporaltype.StreamItem{
		Event: opening.Event,
		Data:  opening.Data,
		At:    workflow.Now(ctx).Unix(),
	})
	if plan.Refused() {
		return temporaltype.StreamItem{
			Event: serviceports.AssistantEventDone,
			Data:  map[string]any{"reply": plan.Decision.Message},
			At:    workflow.Now(ctx).Unix(),
		}
	}

	runCtx, cancel := workflow.WithCancel(ctx)
	defer cancel()
	workflow.Go(ctx, func(timerCtx workflow.Context) {
		if workflow.Sleep(timerCtx, dryRunTimeout) == nil {
			cancel()
		}
	})

	run := agentflow.NewRunContext(plan.RunRequest(&payload.Actor), agentflow.PriorityInteractive)
	outcome, err := agentflow.Run(runCtx, w.runtime, stream.Events(), run, plan.Turn)
	if err != nil {
		return dryRunError(ctx, err)
	}

	reply := ""
	if outcome != nil && outcome.Result != nil {
		reply = outcome.Result.Reply
	}
	stream.Publish(ctx, temporaltype.StreamItem{
		Event: AssistantEventDryRunSteps,
		Data: DryRunStepsEvent{
			Steps: dryRunSteps(&plan, outcome),
			Reply: reply,
		},
		At: workflow.Now(ctx).Unix(),
	})

	return temporaltype.StreamItem{
		Event: serviceports.AssistantEventDone,
		Data:  map[string]any{"reply": reply},
		At:    workflow.Now(ctx).Unix(),
	}
}

func dryRunError(ctx workflow.Context, err error) temporaltype.StreamItem {
	message := dryRunFailedMessage
	if text := rejectionOf(err); text != "" {
		message = text
	} else if failure := modelcall.FailureOf(err); failure != nil && failure.Stopped {
		message = dryRunStoppedMessage
	}

	return temporaltype.StreamItem{
		Event: serviceports.AssistantEventError,
		Data:  map[string]any{"message": message},
		At:    workflow.Now(ctx).Unix(),
	}
}

// dryRunSteps sets each tool call the run finished beside the write it
// became, when it became one.
func dryRunSteps(plan *assistantservice.DryRunPlan, outcome *agentflow.Outcome) []agentdryrun.Step {
	if outcome == nil {
		return []agentdryrun.Step{}
	}
	calls := make([]agentdryrun.Call, 0, len(outcome.Events))
	for idx := range outcome.Events {
		item := &outcome.Events[idx]
		if item.Event != serviceports.AssistantEventToolFinished {
			continue
		}
		var finished serviceports.AssistantToolFinishedEvent
		if err := jsonutils.Convert(item.Data, &finished); err != nil || finished.AgentID.IsNotNil() {
			continue
		}
		calls = append(calls, agentdryrun.Call{
			CallID:  finished.CallID,
			Tool:    finished.Name,
			Verdict: finished.Verdict,
			Failed:  finished.Failed,
			Summary: finished.Summary,
		})
	}

	actions := make(map[string]agentdryrun.Action)
	if outcome.Result != nil {
		for idx := range outcome.Result.Actions {
			action := &outcome.Result.Actions[idx]
			actions[action.ToolCallID] = agentdryrun.Action{
				Tier:      action.Tier,
				Simulated: action.Simulated,
			}
		}
	}

	return agentdryrun.Classify(agentdryrun.Draft{
		Held:   plan.Turn.Held,
		Shadow: plan.Shadow,
	}, calls, actions)
}

