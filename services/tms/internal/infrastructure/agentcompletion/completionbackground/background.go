package completionbackground

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionengine"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

type Background struct {
	*completionengine.Engine
}

// SubmitBackground defers a structured call when a candidate's protocol supports
// it, and runs it inline when none does.
//
// The inline path is not a fallback bolted on for completeness: a self-hosted
// runtime has no background mode at all, and an install that runs only a
// self-hosted server would otherwise lose document extraction entirely.
// Callers get an answer either way and branch on whether Handle came back
// empty.
func (s *Background) SubmitBackground(
	ctx context.Context,
	req *llmports.StructuredCompletionRequest,
) (*llmports.BackgroundSubmission, error) {
	if !s.AI.AIEnabled() {
		return nil, errortypes.NewBusinessError(completionengine.AIDisabledMessage)
	}

	if err := s.AssertWithinSpend(ctx, req.TenantInfo); err != nil {
		return nil, err
	}

	run := completionengine.StructuredRun(req)

	usable, err := s.CandidatesFor(ctx, run)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, provider := range usable {
		runner, call, buildErr := s.backgroundCall(provider, run)
		if buildErr != nil {
			if errors.Is(buildErr, errNoBackgroundSupport) {
				continue
			}

			lastErr = buildErr
			continue
		}

		handle, submitErr := runner.Submit(ctx, call)
		if submitErr != nil {
			lastErr = submitErr
			s.Logger.Warn("background submit failed, falling through",
				zap.String("provider", provider.Name),
				zap.String("task", string(run.Task)),
				zap.Error(submitErr),
			)

			continue
		}

		return &llmports.BackgroundSubmission{
			Handle:          handle.ID,
			ProviderID:      provider.ID,
			ProviderKind:    provider.Kind,
			ModelIdentifier: stringutils.FirstNonEmpty(handle.ModelIdentifier, provider.Model),
			RawStatus:       handle.Status,
		}, nil
	}

	// Every candidate either cannot defer or refused the submission. Running the
	// call inline still answers the question, and a submit failure that would
	// have been fatal is reported only if the inline attempt also fails.
	s.Logger.Debug("no provider accepted a background submission, running inline",
		zap.String("task", string(run.Task)),
		zap.Error(lastErr),
	)

	outcome, err := s.RunAmong(ctx, usable, run)
	if err != nil {
		if lastErr != nil {
			return nil, fmt.Errorf("%w (background submit failed: %w)", err, lastErr)
		}

		return nil, err
	}

	return &llmports.BackgroundSubmission{
		ProviderID:      outcome.ProviderID,
		ProviderKind:    outcome.ProviderKind,
		ModelIdentifier: outcome.Model,
		Result: &llmports.StructuredCompletionResult{
			Text:            outcome.Text,
			ModelIdentifier: outcome.Model,
			InputTokens:     outcome.InputTokens,
			OutputTokens:    outcome.OutputTokens,
			ProviderID:      outcome.ProviderID,
			ProviderKind:    outcome.ProviderKind,

			CacheReadTokens:  outcome.CacheReadTokens,
			CacheWriteTokens: outcome.CacheWriteTokens,
		},
	}, nil
}

// PollBackground asks the provider that issued a handle how the call is going.
func (s *Background) PollBackground(
	ctx context.Context,
	req *llmports.BackgroundPollRequest,
) (*llmports.BackgroundOutcome, error) {
	if !s.AI.AIEnabled() {
		return nil, errortypes.NewBusinessError(completionengine.AIDisabledMessage)
	}

	handle := strings.TrimSpace(req.Handle)
	if handle == "" {
		return nil, errortypes.NewBusinessError("A background handle is required")
	}

	provider, err := s.Repo.GetByID(ctx, aiproviderports.GetAIProviderByIDRequest{
		ID:         req.ProviderID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, errortypes.NewBusinessError(
			"The AI provider that started this call no longer exists",
		)
	}

	adapter, err := s.Adapters.Get(provider.Kind)
	if err != nil {
		return nil, err
	}

	runner, ok := adapter.(modeladapter.BackgroundRunner)
	if !ok {
		return nil, errortypes.NewBusinessError(
			"AI provider {0} cannot report on background calls", provider.Name,
		)
	}

	apiKey, err := s.ResolveAPIKey(provider)
	if err != nil {
		return nil, err
	}

	result, err := runner.Poll(ctx, &modeladapter.Call{
		Provider:                 provider,
		APIKey:                   apiKey,
		Client:                   s.ClientFor(provider),
		StreamClient:             s.StreamClientFor(provider),
		StreamIdle:               s.AI.GetStreamIdleTimeout(),
		ServerCredentialsAllowed: s.AI.ServerCloudCredentialsAllowed(),
	}, handle)
	if err != nil {
		return nil, err
	}

	outcome := &llmports.BackgroundOutcome{
		RawStatus:       result.RawStatus,
		ModelIdentifier: stringutils.FirstNonEmpty(result.ModelIdentifier, provider.Model),
		FailureCode:     result.FailureCode,
		FailureMessage:  result.FailureMessage,
	}

	switch result.State {
	case modeladapter.BackgroundPending:
		outcome.State = llmports.BackgroundPending
	case modeladapter.BackgroundFailed:
		outcome.State = llmports.BackgroundFailed
		s.recordBackground(ctx, provider, req, result)
	case modeladapter.BackgroundCompleted:
		outcome.State = llmports.BackgroundCompleted
		outcome.Result = &llmports.StructuredCompletionResult{
			Text:            result.Response.Text,
			ModelIdentifier: outcome.ModelIdentifier,
			InputTokens:     result.Response.InputTokens,
			OutputTokens:    result.Response.OutputTokens,
			ProviderID:      provider.ID,
			ProviderKind:    provider.Kind,

			CacheReadTokens:  result.Response.CacheReadTokens,
			CacheWriteTokens: result.Response.CacheWriteTokens,
		}
		s.recordBackground(ctx, provider, req, result)
	}

	return outcome, nil
}

func (s *Background) recordBackground(
	ctx context.Context,
	provider *aiprovider.Provider,
	req *llmports.BackgroundPollRequest,
	result *modeladapter.BackgroundOutcome,
) {
	task := req.Task
	if task == "" {
		task = aitask.TaskGeneral
	}

	attempt := completionengine.UsageAttempt{
		Provider:    provider,
		Task:        task,
		Surface:     completionengine.SurfaceFor(aiusage.SurfaceBackground, req.Attribution),
		Attribution: req.Attribution,
		Tenant:      req.TenantInfo,
		Latency:     completionengine.BackgroundLatency(req.SubmittedAt, time.Now()),
		Operation:   aitrace.OperationChat,
		Attempt:     1,
	}
	if result.Response != nil {
		attempt.Outcome = &completionengine.RunOutcome{
			Model: stringutils.FirstNonEmpty(
				result.Response.ModelIdentifier,
				result.ModelIdentifier,
			),
			InputTokens:      result.Response.InputTokens,
			OutputTokens:     result.Response.OutputTokens,
			ReasoningTokens:  result.Response.ReasoningTokens,
			CacheReadTokens:  result.Response.CacheReadTokens,
			CacheWriteTokens: result.Response.CacheWriteTokens,
			FinishReason:     completionengine.FinishReason(result.Response),
			Truncated:        result.Response.Truncated,
			ThinkingDropped:  result.Response.ThinkingDropped,
		}
	}
	if result.State == modeladapter.BackgroundFailed {
		attempt.Err = backgroundFailure(result)
	}

	s.Record(ctx, &attempt)
}

func backgroundFailure(result *modeladapter.BackgroundOutcome) error {
	return fmt.Errorf(
		"background call ended %s: %s",
		stringutils.FirstNonEmpty(result.FailureCode, result.RawStatus, "failed"),
		stringutils.FirstNonEmpty(result.FailureMessage, "no reason given"),
	)
}

var errNoBackgroundSupport = errors.New("provider protocol has no background mode")

func (s *Background) backgroundCall(
	provider *aiprovider.Provider,
	req *completionengine.RunRequest,
) (modeladapter.BackgroundRunner, *modeladapter.Call, error) {
	adapter, err := s.Adapters.Get(provider.Kind)
	if err != nil {
		return nil, nil, err
	}

	runner, ok := adapter.(modeladapter.BackgroundRunner)
	if !ok {
		return nil, nil, errNoBackgroundSupport
	}

	apiKey, err := s.ResolveAPIKey(provider)
	if err != nil {
		return nil, nil, err
	}

	return runner, s.CallFor(provider, apiKey, req), nil
}
