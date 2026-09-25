package completionrouter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

// SubmitBackground defers a structured call when a candidate's protocol supports
// it, and runs it inline when none does.
//
// The inline path is not a fallback bolted on for completeness: a self-hosted
// runtime has no background mode at all, and an install that runs only Ollama
// would otherwise lose document extraction entirely. Callers get an answer
// either way and branch on whether Handle came back empty.
func (s *Service) SubmitBackground(
	ctx context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.BackgroundSubmission, error) {
	if !s.ai.AIEnabled() {
		return nil, errortypes.NewBusinessError(aiDisabledMessage)
	}

	task := req.Task
	if task == "" {
		task = aiprovider.TaskGeneral
	}

	run := &runRequest{
		TenantInfo:          req.TenantInfo,
		Task:                task,
		System:              req.System,
		UserContent:         modeladapter.BuildContextText(req.Context),
		Schema:              req.OutputSchema,
		SchemaName:          req.SchemaName,
		MaxTokens:           req.MaxTokens,
		PreferredProviderID: req.PreferredProviderID,
		Attribution:         req.Attribution,
	}

	usable, err := s.candidatesFor(ctx, run)
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
			s.logger.Warn("background submit failed, falling through",
				zap.String("provider", provider.Name),
				zap.String("task", string(task)),
				zap.Error(submitErr),
			)

			continue
		}

		return &serviceports.BackgroundSubmission{
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
	s.logger.Debug("no provider accepted a background submission, running inline",
		zap.String("task", string(task)),
		zap.Error(lastErr),
	)

	outcome, err := s.runAmong(ctx, usable, run)
	if err != nil {
		if lastErr != nil {
			return nil, fmt.Errorf("%w (background submit failed: %w)", err, lastErr)
		}

		return nil, err
	}

	return &serviceports.BackgroundSubmission{
		ProviderID:      outcome.ProviderID,
		ProviderKind:    outcome.ProviderKind,
		ModelIdentifier: outcome.Model,
		Result: &serviceports.StructuredCompletionResult{
			Text:            outcome.Text,
			ModelIdentifier: outcome.Model,
			InputTokens:     outcome.InputTokens,
			OutputTokens:    outcome.OutputTokens,
			ProviderID:      outcome.ProviderID,
			ProviderKind:    outcome.ProviderKind,
		},
	}, nil
}

// PollBackground asks the provider that issued a handle how the call is going.
func (s *Service) PollBackground(
	ctx context.Context,
	req *serviceports.BackgroundPollRequest,
) (*serviceports.BackgroundOutcome, error) {
	if !s.ai.AIEnabled() {
		return nil, errortypes.NewBusinessError(aiDisabledMessage)
	}

	handle := strings.TrimSpace(req.Handle)
	if handle == "" {
		return nil, errortypes.NewBusinessError("A background handle is required")
	}

	provider, err := s.repo.GetByID(ctx, repositories.GetAIProviderByIDRequest{
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

	adapter, err := s.adapters.Get(provider.Kind)
	if err != nil {
		return nil, err
	}

	runner, ok := adapter.(modeladapter.BackgroundRunner)
	if !ok {
		return nil, errortypes.NewBusinessError(
			"AI provider {0} cannot report on background calls", provider.Name,
		)
	}

	apiKey, err := s.resolveAPIKey(provider)
	if err != nil {
		return nil, err
	}

	result, err := runner.Poll(ctx, &modeladapter.Call{
		Provider:     provider,
		APIKey:       apiKey,
		Client:       s.clientFor(provider),
		StreamClient: s.streamClientFor(provider),
		StreamIdle:   s.ai.GetStreamIdleTimeout(),
	}, handle)
	if err != nil {
		return nil, err
	}

	outcome := &serviceports.BackgroundOutcome{
		RawStatus:       result.RawStatus,
		ModelIdentifier: stringutils.FirstNonEmpty(result.ModelIdentifier, provider.Model),
		FailureCode:     result.FailureCode,
		FailureMessage:  result.FailureMessage,
	}

	switch result.State {
	case modeladapter.BackgroundPending:
		outcome.State = serviceports.BackgroundPending
	case modeladapter.BackgroundFailed:
		outcome.State = serviceports.BackgroundFailed
		s.recordBackground(ctx, provider, req, result)
	case modeladapter.BackgroundCompleted:
		outcome.State = serviceports.BackgroundCompleted
		outcome.Result = &serviceports.StructuredCompletionResult{
			Text:            result.Response.Text,
			ModelIdentifier: outcome.ModelIdentifier,
			InputTokens:     result.Response.InputTokens,
			OutputTokens:    result.Response.OutputTokens,
			ProviderID:      provider.ID,
			ProviderKind:    provider.Kind,
		}
		s.recordBackground(ctx, provider, req, result)
	}

	return outcome, nil
}

func (s *Service) recordBackground(
	ctx context.Context,
	provider *aiprovider.Provider,
	req *serviceports.BackgroundPollRequest,
	result *modeladapter.BackgroundOutcome,
) {
	task := req.Task
	if task == "" {
		task = aiprovider.TaskGeneral
	}

	attempt := usageAttempt{
		provider:    provider,
		task:        task,
		surface:     surfaceFor(aiusage.SurfaceBackground, req.Attribution),
		attribution: req.Attribution,
		tenant:      req.TenantInfo,
		latency:     backgroundLatency(req.SubmittedAt, time.Now()),
		operation:   aitrace.OperationChat,
		attempt:     1,
	}
	if result.Response != nil {
		attempt.outcome = &runOutcome{
			Model: stringutils.FirstNonEmpty(
				result.Response.ModelIdentifier,
				result.ModelIdentifier,
			),
			InputTokens:      result.Response.InputTokens,
			OutputTokens:     result.Response.OutputTokens,
			ReasoningTokens:  result.Response.ReasoningTokens,
			CacheReadTokens:  result.Response.CacheReadTokens,
			CacheWriteTokens: result.Response.CacheWriteTokens,
			FinishReason:     finishReason(result.Response),
			Truncated:        result.Response.Truncated,
		}
	}
	if result.State == modeladapter.BackgroundFailed {
		attempt.err = backgroundFailure(result)
	}

	s.record(ctx, &attempt)
}

func backgroundLatency(submittedAt int64, now time.Time) time.Duration {
	if submittedAt <= 0 {
		return 0
	}

	return max(now.Sub(time.Unix(submittedAt, 0)), 0)
}

func backgroundFailure(result *modeladapter.BackgroundOutcome) error {
	return fmt.Errorf(
		"background call ended %s: %s",
		stringutils.FirstNonEmpty(result.FailureCode, result.RawStatus, "failed"),
		stringutils.FirstNonEmpty(result.FailureMessage, "no reason given"),
	)
}

var errNoBackgroundSupport = errors.New("provider protocol has no background mode")

func (s *Service) backgroundCall(
	provider *aiprovider.Provider,
	req *runRequest,
) (modeladapter.BackgroundRunner, *modeladapter.Call, error) {
	adapter, err := s.adapters.Get(provider.Kind)
	if err != nil {
		return nil, nil, err
	}

	runner, ok := adapter.(modeladapter.BackgroundRunner)
	if !ok {
		return nil, nil, errNoBackgroundSupport
	}

	apiKey, err := s.resolveAPIKey(provider)
	if err != nil {
		return nil, nil, err
	}

	return runner, s.callFor(provider, apiKey, req), nil
}
