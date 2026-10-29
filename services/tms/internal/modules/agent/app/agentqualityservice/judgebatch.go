package agentqualityservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	batchUnavailableNote = "Judge batches are not available; each case is judged on its own."
	batchUnsupportedNote = "The judge's provider has no batch API; each case is judged on its own."
	batchOverBudgetNote  = "Not one judgement fits what is left of the budget at the batch rate."
	batchNothingNote     = "No sampled case has an answer left to judge."
)

type SubmitJudgeBatchRequest struct {
	TenantInfo    pagination.TenantInfo      `json:"tenantInfo"`
	SuiteRunID    pulid.ID                   `json:"suiteRunId"`
	EvaluationIDs []pulid.ID                 `json:"evaluationIds"`
	DayStart      int64                      `json:"dayStart"`
	MonthStart    int64                      `json:"monthStart"`
	Settings      agentquality.SuiteSettings `json:"settings"`
	CommittedUSD  decimal.Decimal            `json:"committedUsd"`
}

type JudgeBatch struct {
	ProviderID  pulid.ID        `json:"providerId,omitzero"`
	BatchID     string          `json:"batchId,omitempty"`
	SubmittedAt int64           `json:"submittedAt,omitempty"`
	Submitted   []pulid.ID      `json:"submitted,omitempty"`
	Deferred    []pulid.ID      `json:"deferred,omitempty"`
	EstimateUSD decimal.Decimal `json:"estimateUsd"`
	StopReason  string          `json:"stopReason,omitempty"`
	Note        string          `json:"note,omitempty"`
}

func (b *JudgeBatch) Batched() bool {
	return b != nil && b.BatchID != ""
}

func (s *Service) SubmitJudgeBatch(
	ctx context.Context,
	req *SubmitJudgeBatchRequest,
) (*JudgeBatch, error) {
	if s.batches == nil || s.completion == nil {
		return &JudgeBatch{Note: batchUnavailableNote}, nil
	}

	decision, err := s.CheckBudget(ctx, &CheckBudgetRequest{
		TenantInfo:   req.TenantInfo,
		DayStart:     req.DayStart,
		MonthStart:   req.MonthStart,
		Settings:     req.Settings,
		CommittedUSD: req.CommittedUSD,
	})
	if err != nil {
		return nil, err
	}
	if decision.Stop {
		return &JudgeBatch{StopReason: decision.Reason}, nil
	}

	items, deferred, err := s.judgeBatchItems(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return &JudgeBatch{Deferred: deferred, Note: batchNothingNote}, nil
	}

	room := decision.Room
	handle, err := s.batches.SubmitBatch(ctx, &llmports.BatchCompletionRequest{
		TenantInfo:    req.TenantInfo,
		Task:          aitask.TaskEvaluationJudge,
		Items:         items,
		SpendLimitUSD: &room,
	})
	switch {
	case errors.Is(err, llmports.ErrBatchUnsupported),
		errors.Is(err, llmports.ErrNoProviderConfigured):
		return &JudgeBatch{Note: batchUnsupportedNote}, nil
	case errors.Is(err, llmports.ErrBatchOverBudget):
		return &JudgeBatch{Note: batchOverBudgetNote}, nil
	case err != nil:
		return nil, err
	}

	submitted := make(map[string]struct{}, len(handle.CustomIDs))
	batch := &JudgeBatch{
		ProviderID:  handle.ProviderID,
		BatchID:     handle.BatchID,
		SubmittedAt: handle.SubmittedAt,
		Submitted:   make([]pulid.ID, 0, len(handle.CustomIDs)),
		EstimateUSD: handle.EstimatedCostUSD,
	}
	for _, custom := range handle.CustomIDs {
		submitted[custom] = struct{}{}
		batch.Submitted = append(batch.Submitted, pulid.ID(custom))
	}
	for idx := range items {
		if _, ok := submitted[items[idx].CustomID]; !ok {
			deferred = append(deferred, pulid.ID(items[idx].CustomID))
		}
	}
	batch.Deferred = deferred

	return batch, nil
}

func (s *Service) judgeBatchItems(
	ctx context.Context,
	req *SubmitJudgeBatchRequest,
) ([]llmports.BatchCompletionItem, []pulid.ID, error) {
	run, err := s.suiteRuns.GetByID(ctx, agentqualityports.GetAgentSuiteRunRequest{
		ID:         req.SuiteRunID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	evaluations, err := s.suiteEvaluationIndex(ctx, req.TenantInfo, run.ID)
	if err != nil {
		return nil, nil, err
	}
	cases, err := s.cases.ListByAgent(ctx, agentqualityports.ListAgentEvalCasesRequest{
		AgentDefinitionID: run.AgentDefinitionID,
		TenantInfo:        req.TenantInfo,
		Limit:             caseSourcePage,
	})
	if err != nil {
		return nil, nil, err
	}
	caseByID := make(map[pulid.ID]*agentquality.EvalCase, len(cases))
	for _, evalCase := range cases {
		caseByID[evalCase.ID] = evalCase
	}

	items := make([]llmports.BatchCompletionItem, 0, len(req.EvaluationIDs))
	deferred := make([]pulid.ID, 0)
	for _, id := range req.EvaluationIDs {
		evaluation, ok := evaluations[id]
		if !ok {
			deferred = append(deferred, id)

			continue
		}
		if settledJudgement(evaluation) != nil {
			continue
		}
		evalCase, ok := caseByID[*evaluation.EvalCaseID]
		if !ok {
			deferred = append(deferred, id)

			continue
		}
		items = append(items, llmports.BatchCompletionItem{
			CustomID: id.String(),
			Request:  *judgeRequest(req.TenantInfo, evaluation, evalCase),
		})
	}

	return items, deferred, nil
}

func (s *Service) suiteEvaluationIndex(
	ctx context.Context,
	tenant pagination.TenantInfo,
	suiteRunID pulid.ID,
) (map[pulid.ID]*agentevaluation.Evaluation, error) {
	evaluations, err := s.suiteEvaluationsOf(ctx, tenant, suiteRunID)
	if err != nil {
		return nil, err
	}

	index := make(map[pulid.ID]*agentevaluation.Evaluation, len(evaluations))
	for _, evaluation := range evaluations {
		index[evaluation.ID] = evaluation
	}

	return index, nil
}

type CheckJudgeBatchRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Batch      JudgeBatch            `json:"batch"`
}

type JudgeBatchStatus struct {
	Ended      bool `json:"ended"`
	Processing int  `json:"processing"`
	Succeeded  int  `json:"succeeded"`
	Errored    int  `json:"errored"`
	Expired    int  `json:"expired"`
	Canceled   int  `json:"canceled"`
}

func (s *Service) CheckJudgeBatch(
	ctx context.Context,
	req *CheckJudgeBatchRequest,
) (*JudgeBatchStatus, error) {
	if s.batches == nil {
		return nil, errortypes.NewBusinessError(batchUnavailableNote)
	}

	status, err := s.batches.BatchStatus(ctx, &llmports.BatchStatusRequest{
		TenantInfo: req.TenantInfo,
		ProviderID: req.Batch.ProviderID,
		BatchID:    req.Batch.BatchID,
	})
	if err != nil {
		return nil, err
	}

	return &JudgeBatchStatus{
		Ended:      status.Ended,
		Processing: status.Processing,
		Succeeded:  status.Succeeded,
		Errored:    status.Errored,
		Expired:    status.Expired,
		Canceled:   status.Canceled,
	}, nil
}

type CollectJudgeBatchRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	SuiteRunID pulid.ID              `json:"suiteRunId"`
	Batch      JudgeBatch            `json:"batch"`
}

type CollectedJudgeBatch struct {
	Judged int        `json:"judged"`
	Retry  []pulid.ID `json:"retry"`
}

func (s *Service) CollectJudgeBatch(
	ctx context.Context,
	req *CollectJudgeBatchRequest,
) (*CollectedJudgeBatch, error) {
	if s.batches == nil {
		return &CollectedJudgeBatch{Retry: req.Batch.Submitted}, nil
	}

	evaluations, err := s.suiteEvaluationIndex(ctx, req.TenantInfo, req.SuiteRunID)
	if err != nil {
		return nil, err
	}
	attributions := make(map[string]aiusageports.AIUsageAttribution, len(req.Batch.Submitted))
	for _, id := range req.Batch.Submitted {
		if evaluation, ok := evaluations[id]; ok {
			attributions[id.String()] = judgeAttribution(evaluation)
		}
	}

	results, err := s.batches.BatchResults(ctx, &llmports.BatchResultsRequest{
		TenantInfo:   req.TenantInfo,
		Task:         aitask.TaskEvaluationJudge,
		ProviderID:   req.Batch.ProviderID,
		BatchID:      req.Batch.BatchID,
		SubmittedAt:  req.Batch.SubmittedAt,
		Attributions: attributions,
	})
	if err != nil {
		return nil, err
	}

	collected := &CollectedJudgeBatch{Retry: make([]pulid.ID, 0)}
	for _, id := range req.Batch.Submitted {
		evaluation, ok := evaluations[id]
		if !ok {
			continue
		}
		item, ok := results[id.String()]
		if evaluation.Judge == nil && (!ok || !item.Succeeded()) {
			s.l.Debug("a batched judgement did not come back; judging the case on its own",
				zap.String("evaluation", id.String()),
				zap.String("failure", item.FailureCode),
			)
			collected.Retry = append(collected.Retry, id)

			continue
		}

		text, model := "", ""
		if item.Result != nil {
			text, model = item.Result.Text, item.Result.ModelIdentifier
		}
		judged, recordErr := s.recordJudgement(ctx, evaluation, text, model)
		if recordErr != nil {
			return nil, recordErr
		}
		if judged.Judged {
			collected.Judged++
		}
	}

	return collected, nil
}
