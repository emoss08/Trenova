package assistantjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdryrun"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/assistantjobs/assistantcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall/modelcallcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (

	// AssistantEventDryRunSteps is the last event of a dry run before its
	// end: each tool call with what it would have come to.
	AssistantEventDryRunSteps = "dry_run_steps"
	dryRunFailedMessage       = "The dry run could not finish. Try again in a moment."
	dryRunStoppedMessage      = "The dry run took too long and was stopped."
)

// DryRunStepsEvent is the dry run's account of its tool calls.
type DryRunStepsEvent struct {
	Steps []agentdryrun.Step `json:"steps"`
	Reply string             `json:"reply"`
}

// PrepareDryRunActivity opens the turn a draft agent would answer the message
// with.
func (a *Activities) PrepareDryRunActivity(
	ctx context.Context,
	payload *assistantcontract.AgentDryRunPayload,
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
	payload *assistantcontract.AgentDryRunPayload,
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
	payload *assistantcontract.AgentDryRunPayload,
) temporaltype.StreamItem {
	var a *Activities
	var plan assistantservice.DryRunPlan
	options := prepareOptions
	options.Priority = temporal.Priority{
		PriorityKey: agentflowcontract.PriorityInteractive,
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
			Event: agentports.AssistantEventDone,
			Data:  map[string]any{"reply": plan.Decision.Message},
			At:    workflow.Now(ctx).Unix(),
		}
	}

	runCtx, cancel := workflow.WithCancel(ctx)
	defer cancel()
	workflow.Go(ctx, func(timerCtx workflow.Context) {
		if workflow.Sleep(timerCtx, assistantcontract.DryRunTimeout) == nil {
			cancel()
		}
	})

	run := agentflow.NewRunContext(
		plan.RunRequest(&payload.Actor),
		agentflowcontract.PriorityInteractive,
	)
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
		Event: agentports.AssistantEventDone,
		Data:  map[string]any{"reply": reply},
		At:    workflow.Now(ctx).Unix(),
	}
}

func dryRunError(ctx workflow.Context, err error) temporaltype.StreamItem {
	message := dryRunFailedMessage
	if text := rejectionOf(err); text != "" {
		message = text
	} else if failure := modelcallcontract.FailureOf(err); failure != nil && failure.Stopped {
		message = dryRunStoppedMessage
	}

	return temporaltype.StreamItem{
		Event: agentports.AssistantEventError,
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
		if item.Event != agentports.AssistantEventToolFinished {
			continue
		}
		var finished agentports.AssistantToolFinishedEvent
		if err := jsonutils.Convert(
			item.Data,
			&finished,
		); err != nil ||
			finished.AgentID.IsNotNil() {
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
