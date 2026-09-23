package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// The helpers here drive a turn in process through the same steps the turn
// workflow takes, so the behaviour of each step is tested without a Temporal
// environment: prepare, run, finish.

func (s *Service) run(ctx context.Context, req *TurnRequest) (*TurnResult, error) {
	return s.runObserved(ctx, req, nil)
}

func (s *Service) runObserved(
	ctx context.Context,
	req *TurnRequest,
	emit serviceports.AssistantStreamEmitter,
) (*TurnResult, error) {
	if emit == nil {
		emit = func(serviceports.StreamEvent) {}
	}

	decision, runReq := s.admit(ctx, req)
	emit(openingEvent(req.Input, decision))
	if runReq == nil {
		return turnResultOf(req.Input, decision, nil, nil), nil
	}

	runReq.Emit = emit
	run, err := s.runtime.Run(ctx, runReq)

	return turnResultOf(req.Input, decision, run, err), err
}

func (s *Service) sendMessage(
	ctx context.Context,
	req *serviceports.SendMessageRequest,
	actor *serviceports.RequestActor,
	emit serviceports.AssistantStreamEmitter,
) (*serviceports.SendMessageResult, error) {
	if emit == nil {
		emit = func(serviceports.StreamEvent) {}
	}

	plan, runReq, err := s.prepareTurn(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	emit(plan.Opening())

	var run *serviceports.RunResult
	var runErr error
	var artifacts []*assistantartifact.Artifact
	if runReq != nil {
		runReq.Emit = emit
		runReq.ToolObserver = func(
			observation serviceports.ToolObservation,
		) (*serviceports.ShownArtifact, error) {
			shown, kept, err := s.ObserveTool(ctx, plan.ThreadID, actor, observation, emit)
			artifacts = append(artifacts, kept...)

			return shown, err
		}
		run, runErr = s.runtime.Run(ctx, runReq)
	}

	result, err := s.FinishTurn(context.WithoutCancel(ctx), &FinishTurnRequest{
		Plan:       plan,
		Actor:      actor,
		TenantInfo: req.TenantInfo,
		Run:        run,
		Failure:    runErr,
		Artifacts:  artifacts,
	}, emit)
	if runErr != nil {
		return nil, runErr
	}

	return result, err
}

func (s *Service) ask(
	ctx context.Context,
	req *serviceports.AskRequest,
	actor *serviceports.RequestActor,
	emit serviceports.AssistantStreamEmitter,
) (*serviceports.SendMessageResult, error) {
	if emit == nil {
		emit = func(serviceports.StreamEvent) {}
	}

	thread, err := s.StartAsk(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	emit(serviceports.StreamEvent{Event: serviceports.AssistantEventThread, Data: thread})

	return s.sendMessage(ctx, &serviceports.SendMessageRequest{
		ThreadID:   thread.ID,
		Content:    req.Content,
		Page:       req.Page,
		TenantInfo: req.TenantInfo,
		Mentions:   req.Mentions,
	}, actor, emit)
}
