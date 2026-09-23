package assistantjobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Error types the workflow must not retry. Each one names an outcome a second
// attempt would reach identically: a refusal, a missing record, a spent
// budget. Retrying those spends money to produce the same answer, and the
// issue this work comes from is explicit that validation and authorization
// outcomes are surfaced rather than retried.
const (
	errTypeTurnNotFound = "AssistantTurnNotFound"
	errTypeNoActor      = "AssistantTurnHasNoActor"
	errTypeRefused      = "AssistantTurnRefused"
)

type ActivitiesParams struct {
	fx.In

	Logger    *zap.Logger
	Assistant serviceports.AssistantService
	Turns     *assistantturnservice.Service
	TurnRepo  repositories.AssistantTurnRepository
	Steps     serviceports.RunStepLedger
}

type Activities struct {
	logger    *zap.Logger
	assistant serviceports.AssistantService
	turns     *assistantturnservice.Service
	turnRepo  repositories.AssistantTurnRepository
	steps     serviceports.RunStepLedger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		logger:    p.Logger.Named("job.assistant-turn"),
		assistant: p.Assistant,
		turns:     p.Turns,
		turnRepo:  p.TurnRepo,
		steps:     p.Steps,
	}
}

// turnBodyKey names the whole turn as one operation.
//
// The step ledger guards individual tool calls, which is what a background run
// needs, but a chat turn needs something coarser. A retry that re-ran the turn
// would ask the model again and append a second transcript to the same
// conversation, and the second one would not be a correction — it would be a
// duplicate reply to a question asked once. So the turn itself is claimed, and
// a retry that finds the claim taken does not answer again.
const turnBodyKey = "assistant-turn-body"

// AssistantTurnActivity answers one question.
//
// The name carries its package because Temporal's activity registry is keyed
// by method name across a whole task queue, and a collision is a panic at
// startup rather than a compile error.
func (a *Activities) AssistantTurnActivity(
	ctx context.Context,
	payload *AssistantTurnPayload,
) (*AssistantTurnResult, error) {
	if payload.Actor.UserID.IsNil() {
		// A turn runs as the person who asked. A worker that carried on
		// without one would be running as something nobody granted, which is
		// precisely what a background worker must never do.
		return nil, temporal.NewNonRetryableApplicationError(
			"this turn arrived without the person who asked for it", errTypeNoActor, nil,
		)
	}

	tenant := payload.tenantInfo()
	turn, err := a.turnRepo.GetByID(ctx, repositories.GetAssistantTurnRequest{
		ID:         payload.TurnID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"this turn's record could not be read", errTypeTurnNotFound, err,
		)
	}

	claimed, err := a.claimTurn(ctx, payload, tenant)
	if err != nil {
		return nil, err
	}
	if !claimed {
		a.logger.Info("this turn was already answered on an earlier attempt",
			zap.String("turn", payload.TurnID.String()),
		)

		return &AssistantTurnResult{Status: string(turn.Status), Replayed: true}, nil
	}

	activity.RecordHeartbeat(ctx, "running")

	return a.answer(ctx, payload, turn, tenant)
}

// claimTurn reserves the turn before any of it runs, and reports whether this
// attempt is the one that gets to answer.
func (a *Activities) claimTurn(
	ctx context.Context,
	payload *AssistantTurnPayload,
	tenant pagination.TenantInfo,
) (bool, error) {
	verdict, err := a.steps.Claim(ctx, tenant, serviceports.RunStep{
		OwnerKind: serviceports.RunStepOwnerAssistantTurn,
		OwnerID:   payload.TurnID,
		Attempt:   int(activity.GetInfo(ctx).Attempt),
		Kind:      serviceports.RunStepCompletion,
		Key:       turnBodyKey,
		ToolName:  turnBodyKey,
	})
	if err != nil {
		return false, fmt.Errorf("claim this turn: %w", err)
	}

	// Fresh is this attempt's to answer. Anything else means an earlier one
	// already did, or began and never finished — and a half-answered turn is
	// not one to answer again, because whatever it already said is in the
	// conversation.
	return verdict.State == serviceports.StepFresh, nil
}

func (a *Activities) answer(
	ctx context.Context,
	payload *AssistantTurnPayload,
	turn *conversation.AssistantTurn,
	tenant pagination.TenantInfo,
) (*AssistantTurnResult, error) {
	// Every event goes to the turn's stream, which is where the reader is
	// watching from. There is no second emitter: nobody is holding a
	// connection on this side.
	observed, closeStream := a.turns.Observe(ctx, turn, nil)

	beats := heartbeater(ctx, observed)

	result, runErr := a.assistant.SendMessageStream(ctx, &serviceports.SendMessageRequest{
		ThreadID:              payload.ThreadID,
		Content:               payload.Content,
		Page:                  payload.Request.Page,
		TenantInfo:            tenant,
		PreferredProviderID:   payload.Request.PreferredProviderID,
		ProviderChosen:        payload.Request.ProviderChosen,
		AttachmentDocumentIDs: payload.Request.AttachmentDocumentIDs,
		Mentions:              payload.Request.Mentions,
	}, &payload.Actor, beats)

	status := assistantturnservice.StatusFor(false, runErr)
	if runErr == nil {
		status = assistantturnservice.StatusFor(result.Refused, nil)
	}

	ending := endingFor(result, runErr)
	closeStream(ending)
	a.settleTurn(ctx, payload, tenant, status, runErr)
	a.turns.Complete(ctx, turn, status, runErr)

	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		return nil, runErr
	}

	return &AssistantTurnResult{
		Status:  string(status),
		Refused: runErr == nil && result.Refused,
	}, nil
}

// settleTurn closes the turn's claim so the ledger agrees with the record.
func (a *Activities) settleTurn(
	ctx context.Context,
	payload *AssistantTurnPayload,
	tenant pagination.TenantInfo,
	status conversation.AssistantTurnStatus,
	cause error,
) {
	outcome := serviceports.RunStepOutcome{Content: string(status), Failed: cause != nil}
	stepStatus := serviceports.RunStepCompleted
	if cause != nil {
		stepStatus = serviceports.RunStepFailed
	}

	// Outside the turn's context: a stopped turn cancels it, and a claim left
	// open would make the next attempt refuse a turn that finished cleanly.
	err := a.steps.Settle(context.WithoutCancel(ctx), tenant, serviceports.RunStep{
		OwnerKind: serviceports.RunStepOwnerAssistantTurn,
		OwnerID:   payload.TurnID,
		Kind:      serviceports.RunStepCompletion,
		Key:       turnBodyKey,
		Status:    stepStatus,
		Outcome:   outcome,
	})
	if err != nil {
		a.logger.Error("could not close a turn's claim",
			zap.String("turn", payload.TurnID.String()),
			zap.Error(err),
		)
	}
}

// heartbeater reports progress to Temporal on every event.
//
// It is what makes the turn cancellable: Temporal delivers a cancellation to
// an activity only through a heartbeat, so a turn that never beat could not be
// stopped by anybody pressing stop.
func heartbeater(
	ctx context.Context,
	emit serviceports.AssistantStreamEmitter,
) serviceports.AssistantStreamEmitter {
	return func(event serviceports.StreamEvent) {
		emit(event)
		activity.RecordHeartbeat(ctx, event.Event)
	}
}

func endingFor(
	result *serviceports.SendMessageResult,
	cause error,
) serviceports.StreamEvent {
	if cause != nil {
		message := "The assistant could not finish this reply. Try again in a moment."
		if errors.Is(cause, context.Canceled) {
			message = "Stopped. What was said so far has been kept in the conversation."
		}

		return serviceports.StreamEvent{
			Event: serviceports.AssistantEventError,
			Data:  map[string]any{"message": message},
		}
	}

	return serviceports.StreamEvent{Event: serviceports.AssistantEventDone, Data: result}
}
