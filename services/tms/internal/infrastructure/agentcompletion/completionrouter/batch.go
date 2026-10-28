package completionrouter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/llmtokens"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	maxBatchItems     = 10_000
	batchUsagePrefix  = "aiu_"
	batchReadTimeout  = 10 * time.Minute
	batchFailedPrefix = "batch item ended "
)

var _ llmports.BatchCompletionService = (*Service)(nil)

func (s *Service) SubmitBatch(
	ctx context.Context,
	req *llmports.BatchCompletionRequest,
) (*llmports.BatchHandle, error) {
	if !s.ai.AIEnabled() {
		return nil, errortypes.NewBusinessError(aiDisabledMessage)
	}
	runs, err := batchRuns(req)
	if err != nil {
		return nil, err
	}
	if err = s.assertWithinSpend(ctx, req.TenantInfo); err != nil {
		return nil, err
	}

	usable, err := s.candidatesFor(ctx, runs[0])
	if err != nil {
		return nil, err
	}
	provider := usable[0]
	runner, ok := s.batchRunner(provider)
	if !ok {
		return nil, fmt.Errorf("%s: %w", provider.Name, llmports.ErrBatchUnsupported)
	}

	included, estimate := affordable(provider, req, runs)
	if len(included) == 0 {
		return nil, fmt.Errorf("%s: %w", provider.Name, llmports.ErrBatchOverBudget)
	}

	var submitted *modeladapter.BatchSubmission
	limited := s.withDeadline(ctx, provider.ResolvedTimeout())
	err = limited.finish(s.withKeyFallback(provider, func(apiKey string) error {
		requests := make([]modeladapter.BatchRequest, 0, len(included))
		for _, idx := range included {
			requests = append(requests, modeladapter.BatchRequest{
				CustomID: req.Items[idx].CustomID,
				Request:  s.callFor(provider, apiKey, runs[idx]).Request,
			})
		}

		var callErr error
		submitted, callErr = s.submitBatch(
			limited.ctx,
			runner,
			s.batchCall(provider, apiKey),
			requests,
		)

		return callErr
	}))
	s.observe(ctx, provider, err)
	if err != nil {
		return nil, fmt.Errorf("submit a batch to %s: %w", provider.Name, err)
	}
	s.touchKey(ctx, provider, req.TenantInfo)

	customIDs := make([]string, 0, len(included))
	for _, idx := range included {
		customIDs = append(customIDs, req.Items[idx].CustomID)
	}

	return &llmports.BatchHandle{
		ProviderID:       provider.ID,
		BatchID:          submitted.ID,
		Model:            provider.Model,
		CustomIDs:        customIDs,
		SubmittedAt:      time.Now().Unix(),
		EstimatedCostUSD: estimate,
	}, nil
}

func batchRuns(req *llmports.BatchCompletionRequest) ([]*runRequest, error) {
	if len(req.Items) == 0 {
		return nil, errortypes.NewBusinessError("A batch needs at least one request")
	}
	if len(req.Items) > maxBatchItems {
		return nil, errortypes.NewBusinessError(
			"A batch takes at most {0} requests", maxBatchItems,
		)
	}

	task := req.Task
	if task == "" {
		task = aiprovider.TaskGeneral
	}
	seen := make(map[string]struct{}, len(req.Items))
	runs := make([]*runRequest, 0, len(req.Items))
	for idx := range req.Items {
		item := &req.Items[idx]
		custom := strings.TrimSpace(item.CustomID)
		if custom == "" {
			return nil, errortypes.NewBusinessError("Every request in a batch needs an id")
		}
		if _, dup := seen[custom]; dup {
			return nil, errortypes.NewBusinessError(
				"Request id {0} appears twice in one batch", custom,
			)
		}
		seen[custom] = struct{}{}

		run := structuredRun(&item.Request)
		run.TenantInfo = req.TenantInfo
		run.Task = task
		runs = append(runs, run)
	}

	return runs, nil
}

func affordable(
	provider *aiprovider.Provider,
	req *llmports.BatchCompletionRequest,
	runs []*runRequest,
) ([]int, decimal.Decimal) {
	included := make([]int, 0, len(runs))
	total := decimal.Zero
	for idx, run := range runs {
		cost := batchEstimate(provider, req.Task, run)
		if cost != nil {
			next := total.Add(*cost)
			if req.SpendLimitUSD != nil && next.GreaterThan(*req.SpendLimitUSD) {
				break
			}
			total = next
		}
		included = append(included, idx)
	}

	return included, total
}

func batchEstimate(
	provider *aiprovider.Provider,
	task aiprovider.Task,
	run *runRequest,
) *decimal.Decimal {
	output := run.MaxTokens
	if output <= 0 {
		output = provider.ResolvedMaxTokens()
	}

	return provider.BatchCostForTask(task, aiprovider.TokenUsage{
		Input: llmtokens.EstimateAll([]string{
			structuredSystem(run, provider.StructuredOutputMode),
			run.UserContent,
		}),
		Output: output,
	})
}

func (s *Service) submitBatch(
	ctx context.Context,
	runner modeladapter.BatchRunner,
	call *modeladapter.Call,
	requests []modeladapter.BatchRequest,
) (*modeladapter.BatchSubmission, error) {
	var waited time.Duration
	for attempt := 0; ; attempt++ {
		submitted, err := runner.SubmitBatch(ctx, call, requests)
		if err == nil {
			return submitted, nil
		}
		wait, again := s.retryWait(err, attempt, waited)
		if !again {
			return nil, err
		}
		if err = s.wait(ctx, wait); err != nil {
			return nil, err
		}
		waited += wait
	}
}

func (s *Service) batchRunner(provider *aiprovider.Provider) (modeladapter.BatchRunner, bool) {
	adapter, err := s.adapters.Get(provider.Kind)
	if err != nil {
		return nil, false
	}
	runner, ok := adapter.(modeladapter.BatchRunner)
	if !ok || !runner.Batches(provider) {
		return nil, false
	}

	return runner, true
}

func (s *Service) batchCall(provider *aiprovider.Provider, apiKey string) *modeladapter.Call {
	return &modeladapter.Call{
		Provider:                 provider,
		APIKey:                   apiKey,
		Client:                   s.clientFor(provider),
		ServerCredentialsAllowed: s.ai.ServerCloudCredentialsAllowed(),
	}
}

func (s *Service) batchProvider(
	ctx context.Context,
	tenant pagination.TenantInfo,
	providerID pulid.ID,
	batchID string,
) (*aiprovider.Provider, modeladapter.BatchRunner, error) {
	if !s.ai.AIEnabled() {
		return nil, nil, errortypes.NewBusinessError(aiDisabledMessage)
	}
	if strings.TrimSpace(batchID) == "" {
		return nil, nil, errortypes.NewBusinessError("A batch id is required")
	}

	provider, err := s.repo.GetByID(ctx, aiproviderports.GetAIProviderByIDRequest{
		ID:         providerID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, nil, err
	}
	if provider == nil {
		return nil, nil, errortypes.NewBusinessError(
			"The AI provider that took this batch no longer exists",
		)
	}

	adapter, err := s.adapters.Get(provider.Kind)
	if err != nil {
		return nil, nil, err
	}
	runner, ok := adapter.(modeladapter.BatchRunner)
	if !ok {
		return nil, nil, fmt.Errorf("%s: %w", provider.Name, llmports.ErrBatchUnsupported)
	}

	return provider, runner, nil
}

func (s *Service) BatchStatus(
	ctx context.Context,
	req *llmports.BatchStatusRequest,
) (*llmports.BatchStatus, error) {
	provider, runner, err := s.batchProvider(ctx, req.TenantInfo, req.ProviderID, req.BatchID)
	if err != nil {
		return nil, err
	}

	var progress *modeladapter.BatchProgress
	limited := s.withDeadline(ctx, provider.ResolvedTimeout())
	err = limited.finish(s.withKeyFallback(provider, func(apiKey string) error {
		var callErr error
		progress, callErr = runner.PollBatch(
			limited.ctx,
			s.batchCall(provider, apiKey),
			req.BatchID,
		)

		return callErr
	}))
	if err != nil {
		return nil, fmt.Errorf("read batch %s from %s: %w", req.BatchID, provider.Name, err)
	}

	return &llmports.BatchStatus{
		Ended:      progress.Ended,
		RawStatus:  progress.RawStatus,
		Processing: progress.Processing,
		Succeeded:  progress.Succeeded,
		Errored:    progress.Errored,
		Canceled:   progress.Canceled,
		Expired:    progress.Expired,
	}, nil
}

func (s *Service) BatchResults(
	ctx context.Context,
	req *llmports.BatchResultsRequest,
) (map[string]llmports.BatchItemResult, error) {
	provider, runner, err := s.batchProvider(ctx, req.TenantInfo, req.ProviderID, req.BatchID)
	if err != nil {
		return nil, err
	}

	task := req.Task
	if task == "" {
		task = aiprovider.TaskGeneral
	}
	reader := &batchReader{service: s, provider: provider, task: task, req: req}
	limited := s.withDeadline(ctx, batchReadTimeout)
	err = limited.finish(s.withKeyFallback(provider, func(apiKey string) error {
		reader.reset()

		return runner.ReadBatch(
			limited.ctx,
			s.batchCall(provider, apiKey),
			req.BatchID,
			func(item *modeladapter.BatchItem) error {
				reader.read(limited.ctx, item)

				return nil
			},
		)
	}))
	if err != nil {
		return nil, fmt.Errorf("read the results of batch %s: %w", req.BatchID, err)
	}

	if err = s.recordBatch(ctx, req.TenantInfo, reader.rows); err != nil {
		return nil, err
	}

	return reader.results, nil
}

type batchReader struct {
	service  *Service
	provider *aiprovider.Provider
	task     aiprovider.Task
	req      *llmports.BatchResultsRequest
	results  map[string]llmports.BatchItemResult
	rows     []*aiusage.AIUsageRecord
}

func (r *batchReader) reset() {
	r.results = make(map[string]llmports.BatchItemResult)
	r.rows = r.rows[:0]
}

func (r *batchReader) read(ctx context.Context, item *modeladapter.BatchItem) {
	result := llmports.BatchItemResult{
		CustomID:       item.CustomID,
		FailureCode:    item.FailureCode,
		FailureMessage: item.FailureMessage,
	}

	var outcome *runOutcome
	var failure error
	switch {
	case item.Response != nil:
		outcome = batchOutcome(r.provider, item.Response)
		failure = batchResponseFailure(item.Response)
	case item.FailureCode != "":
		failure = errors.New(batchFailedPrefix + item.FailureCode + ": " +
			stringutils.FirstNonEmpty(item.FailureMessage, "no reason given"))
	}
	if outcome != nil && failure == nil {
		result.Result = &llmports.StructuredCompletionResult{
			Text:             outcome.Text,
			ModelIdentifier:  outcome.Model,
			InputTokens:      outcome.InputTokens,
			OutputTokens:     outcome.OutputTokens,
			CacheReadTokens:  outcome.CacheReadTokens,
			CacheWriteTokens: outcome.CacheWriteTokens,
			CostUSD:          outcome.cost(r.provider, r.task),
			ProviderID:       r.provider.ID,
			ProviderKind:     r.provider.Kind,
		}
	}
	if failure != nil && result.FailureCode == "" {
		result.FailureCode = "unusable"
		result.FailureMessage = failure.Error()
	}
	r.results[item.CustomID] = result

	if r.service.usage == nil || item.NotRun {
		return
	}
	row := usageRow(ctx, &usageAttempt{
		provider:    r.provider,
		task:        r.task,
		surface:     surfaceFor(aiusage.SurfaceBackground, r.req.Attributions[item.CustomID]),
		attribution: r.req.Attributions[item.CustomID],
		tenant:      r.req.TenantInfo,
		latency:     backgroundLatency(r.req.SubmittedAt, time.Now()),
		operation:   aitrace.OperationChat,
		attempt:     1,
		outcome:     outcome,
		err:         failure,
	})
	row.ID = pulid.Derived(
		batchUsagePrefix,
		time.Unix(r.req.SubmittedAt, 0),
		r.req.BatchID+"/"+item.CustomID,
	)
	row.Batched = true
	r.rows = append(r.rows, row)
}

func batchOutcome(provider *aiprovider.Provider, resp *modeladapter.Response) *runOutcome {
	return &runOutcome{
		Text:             resp.Text,
		Model:            stringutils.FirstNonEmpty(resp.ModelIdentifier, provider.Model),
		InputTokens:      resp.InputTokens,
		OutputTokens:     resp.OutputTokens,
		ReasoningTokens:  resp.ReasoningTokens,
		CacheReadTokens:  resp.CacheReadTokens,
		CacheWriteTokens: resp.CacheWriteTokens,
		FinishReason:     finishReason(resp),
		Truncated:        resp.Truncated,
		ProviderID:       provider.ID,
		ProviderKind:     provider.Kind,
		Batched:          true,
	}
}

func batchResponseFailure(resp *modeladapter.Response) error {
	if resp.Refused {
		return errRefused
	}
	if strings.TrimSpace(resp.Text) == "" {
		return errors.New("provider returned no content")
	}

	return objectReply(resp.Text)
}

func (s *Service) recordBatch(
	ctx context.Context,
	tenant pagination.TenantInfo,
	rows []*aiusage.AIUsageRecord,
) error {
	if s.usage == nil || len(rows) == 0 {
		return nil
	}

	scoped := dbscope.WithValidTenant(ctx, tenant.DBTenant())
	if _, err := s.usage.CreateOrSkip(scoped, rows); err != nil {
		return fmt.Errorf("record the usage of a batch: %w", err)
	}

	return nil
}
