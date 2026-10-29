package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	emit kernelports.AssistantStreamEmitter,
) (*TurnResult, error) {
	if emit == nil {
		emit = func(kernelports.StreamEvent) {}
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
	req *assistantports.SendMessageRequest,
	actor *permissionports.RequestActor,
	emit kernelports.AssistantStreamEmitter,
) (*assistantports.SendMessageResult, error) {
	if emit == nil {
		emit = func(kernelports.StreamEvent) {}
	}

	plan, runReq, err := s.prepareTurn(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	emit(plan.Opening())

	var run *agentruntimeports.RunResult
	var runErr error
	var artifacts []*assistantartifact.Artifact
	if runReq != nil {
		runReq.Emit = emit
		runReq.ToolObserver = func(
			observation agentruntimeports.ToolObservation,
		) (*agentruntimeports.ShownArtifact, error) {
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
	req *assistantports.AskRequest,
	actor *permissionports.RequestActor,
	emit kernelports.AssistantStreamEmitter,
) (*assistantports.SendMessageResult, error) {
	if emit == nil {
		emit = func(kernelports.StreamEvent) {}
	}

	thread, err := s.StartAsk(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	emit(kernelports.StreamEvent{Event: assistantstreamports.AssistantEventThread, Data: thread})

	return s.sendMessage(ctx, &assistantports.SendMessageRequest{
		ThreadID:   thread.ID,
		Content:    req.Content,
		Page:       req.Page,
		TenantInfo: req.TenantInfo,
		Mentions:   req.Mentions,
	}, actor, emit)
}
