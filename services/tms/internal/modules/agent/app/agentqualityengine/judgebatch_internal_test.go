package agentqualityengine

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentevaluationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const judgeReply = `{"score":0.9,"rationale":"It answered from the records."}`

type judgedEvaluations struct {
	agentevaluationports.AgentEvaluationRepository
	stored  map[pulid.ID]*agentevaluation.Evaluation
	order   []pulid.ID
	updates int
}

func (f *judgedEvaluations) clone(id pulid.ID) *agentevaluation.Evaluation {
	stored := *f.stored[id]
	if stored.Checks != nil {
		checks := *stored.Checks
		stored.Checks = &checks
	}

	return &stored
}

func (f *judgedEvaluations) ListBySuiteRun(
	_ context.Context,
	req *agentevaluationports.ListSuiteEvaluationsRequest,
) ([]*agentevaluation.Evaluation, error) {
	out := make([]*agentevaluation.Evaluation, 0, len(f.order))
	for _, id := range f.order {
		if *f.stored[id].SuiteOrdinal > req.AfterOrdinal {
			out = append(out, f.clone(id))
		}
	}

	return out, nil
}

func (f *judgedEvaluations) GetByID(
	_ context.Context,
	req agentevaluationports.GetAgentEvaluationByIDRequest,
) (*agentevaluation.Evaluation, error) {
	if _, ok := f.stored[req.ID]; !ok {
		return nil, errortypes.NewNotFoundError("Evaluation not found")
	}

	return f.clone(req.ID), nil
}

func (f *judgedEvaluations) Update(
	_ context.Context,
	entity *agentevaluation.Evaluation,
) (*agentevaluation.Evaluation, error) {
	f.updates++
	saved := *entity
	f.stored[entity.ID] = &saved

	return entity, nil
}

type judgedCases struct {
	agentqualityports.AgentEvalCaseRepository
	cases []*agentquality.EvalCase
}

func (f *judgedCases) ListByAgent(
	context.Context,
	agentqualityports.ListAgentEvalCasesRequest,
) ([]*agentquality.EvalCase, error) {
	return f.cases, nil
}

func (f *judgedCases) GetByID(
	_ context.Context,
	req agentqualityports.GetAgentEvalCaseByIDRequest,
) (*agentquality.EvalCase, error) {
	for _, evalCase := range f.cases {
		if evalCase.ID == req.ID {
			return evalCase, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Case not found")
}

type askedJudge struct {
	llmports.CompletionService
	asked []*llmports.StructuredCompletionRequest
}

func (f *askedJudge) CompleteStructured(
	_ context.Context,
	req *llmports.StructuredCompletionRequest,
) (*llmports.StructuredCompletionResult, error) {
	f.asked = append(f.asked, req)

	return &llmports.StructuredCompletionResult{Text: judgeReply, ModelIdentifier: "judge"}, nil
}

type batchedJudge struct {
	submitted []*llmports.BatchCompletionRequest
	take      int
	err       error
	results   map[string]llmports.BatchItemResult
	reads     int
}

func (f *batchedJudge) SubmitBatch(
	_ context.Context,
	req *llmports.BatchCompletionRequest,
) (*llmports.BatchHandle, error) {
	f.submitted = append(f.submitted, req)
	if f.err != nil {
		return nil, f.err
	}

	take := len(req.Items)
	if f.take > 0 {
		take = min(take, f.take)
	}
	ids := make([]string, 0, take)
	for _, item := range req.Items[:take] {
		ids = append(ids, item.CustomID)
	}

	return &llmports.BatchHandle{
		ProviderID:       pulid.ID("aiprv_judge"),
		BatchID:          "msgbatch_1",
		CustomIDs:        ids,
		SubmittedAt:      1_800_000_100,
		EstimatedCostUSD: decimal.RequireFromString("0.02"),
	}, nil
}

func (f *batchedJudge) BatchStatus(
	context.Context,
	*llmports.BatchStatusRequest,
) (*llmports.BatchStatus, error) {
	return &llmports.BatchStatus{Ended: true}, nil
}

func (f *batchedJudge) BatchResults(
	context.Context,
	*llmports.BatchResultsRequest,
) (map[string]llmports.BatchItemResult, error) {
	f.reads++

	return f.results, nil
}

type judgeFixture struct {
	tenant      pagination.TenantInfo
	run         *agentqualitysuite.SuiteRun
	evaluations *judgedEvaluations
	ids         []pulid.ID
	completion  *askedJudge
	batches     *batchedJudge
	usage       *windowedUsage
	service     *Engine
}

func newJudgeFixture(t *testing.T, count int) *judgeFixture {
	t.Helper()

	tenant := pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")}
	agentID := pulid.MustNew("agd_")
	run := &agentqualitysuite.SuiteRun{
		ID:                agentqualitysuite.NewSuiteRunID(),
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		AgentDefinitionID: agentID,
	}

	evaluations := &judgedEvaluations{stored: map[pulid.ID]*agentevaluation.Evaluation{}}
	cases := &judgedCases{}
	ids := make([]pulid.ID, 0, count)
	for idx := range count {
		caseID := pulid.MustNew("aec_")
		cases.cases = append(cases.cases, &agentquality.EvalCase{
			ID:                caseID,
			AgentDefinitionID: agentID,
			Input:             "Which loads are late?",
			Rubric:            "Names every late load.",
		})
		ordinal := idx + 1
		evaluation := &agentevaluation.Evaluation{
			ID:                pulid.MustNew(agentevaluation.EvaluationIDPrefix),
			OrganizationID:    tenant.OrgID,
			BusinessUnitID:    tenant.BuID,
			AgentDefinitionID: agentID,
			EvalCaseID:        &caseID,
			Status:            agentevaluation.EvaluationStatusCompleted,
			Reply:             "Loads 41 and 42 are late.",
			Checks:            &agentevaluation.CaseChecks{Deterministic: 0.8},
			SuiteRunID:        &run.ID,
			SuiteOrdinal:      &ordinal,
		}
		evaluations.stored[evaluation.ID] = evaluation
		evaluations.order = append(evaluations.order, evaluation.ID)
		ids = append(ids, evaluation.ID)
	}

	completion := &askedJudge{}
	batches := &batchedJudge{}
	usage := &windowedUsage{spent: decimal.RequireFromString("1.00")}
	service := &Engine{
		l:           zap.NewNop(),
		suiteRuns:   &fakeSuiteRuns{runs: map[pulid.ID]*agentqualitysuite.SuiteRun{run.ID: run}},
		evaluations: evaluations,
		cases:       cases,
		usage:       usage,
		completion:  completion,
		batches:     batches,
		now:         func() int64 { return 1_800_000_500 },
	}

	return &judgeFixture{
		tenant:      tenant,
		run:         run,
		evaluations: evaluations,
		ids:         ids,
		completion:  completion,
		batches:     batches,
		usage:       usage,
		service:     service,
	}
}

func (f *judgeFixture) submit(t *testing.T, committed string) *JudgeBatch {
	t.Helper()

	batch, err := f.service.SubmitJudgeBatch(t.Context(), &SubmitJudgeBatchRequest{
		TenantInfo:    f.tenant,
		SuiteRunID:    f.run.ID,
		EvaluationIDs: f.ids,
		DayStart:      1_800_000_000,
		MonthStart:    1_799_000_000,
		Settings: agentqualitycontrol.SuiteSettings{
			NightlyBudgetUSD: decimal.RequireFromString("5.00"),
			MonthlyBudgetUSD: decimal.RequireFromString("50.00"),
			JudgeEnabled:     true,
		},
		CommittedUSD: decimal.RequireFromString(committed),
	})
	require.NoError(t, err)

	return batch
}

func TestJudgeRequest_IsTheSameAskedAloneOrInABatch(t *testing.T) {
	t.Parallel()

	f := newJudgeFixture(t, 1)
	_, err := f.service.JudgeCase(t.Context(), &JudgeCaseRequest{
		TenantInfo:   f.tenant,
		EvaluationID: f.ids[0],
	})
	require.NoError(t, err)
	require.Len(t, f.completion.asked, 1)

	f.evaluations.stored[f.ids[0]].Judge = nil
	batch := f.submit(t, "0")
	require.True(t, batch.Batched())
	require.Len(t, f.batches.submitted, 1)
	require.Len(t, f.batches.submitted[0].Items, 1)

	item := f.batches.submitted[0].Items[0]
	assert.Equal(t, f.ids[0].String(), item.CustomID)
	assert.Equal(t, *f.completion.asked[0], item.Request,
		"a batched judgement asks exactly what the judge is asked one case at a time")
}

func TestSubmitJudgeBatch_HoldsTheBatchToWhatIsLeftOfTheBudget(t *testing.T) {
	t.Parallel()

	f := newJudgeFixture(t, 2)
	batch := f.submit(t, "0.25")
	require.True(t, batch.Batched())
	assert.Equal(t, f.ids, batch.Submitted)
	assert.Empty(t, batch.Deferred)
	assert.True(t, batch.EstimateUSD.Equal(decimal.RequireFromString("0.02")))

	limit := f.batches.submitted[0].SpendLimitUSD
	require.NotNil(t, limit)
	assert.True(t, limit.Equal(decimal.RequireFromString("3.75")),
		"five dollars less one spent and a quarter held for other batches: %s", limit)
}

func TestSubmitJudgeBatch_StopsWhenTheBudgetIsSpent(t *testing.T) {
	t.Parallel()

	f := newJudgeFixture(t, 2)
	f.usage.spent = decimal.RequireFromString("5.00")

	batch := f.submit(t, "0")
	assert.False(t, batch.Batched())
	assert.NotEmpty(t, batch.StopReason)
	assert.Empty(t, f.batches.submitted, "a spent budget sends nothing")
}

func TestSubmitJudgeBatch_JudgesOneAtATimeWhenTheProviderCannotBatch(t *testing.T) {
	t.Parallel()

	f := newJudgeFixture(t, 2)
	f.batches.err = llmports.ErrBatchUnsupported

	batch := f.submit(t, "0")
	assert.False(t, batch.Batched())
	assert.Empty(t, batch.StopReason)
	assert.Equal(t, batchUnsupportedNote, batch.Note)
}

func TestSubmitJudgeBatch_DefersWhatTheBatchCouldNotTake(t *testing.T) {
	t.Parallel()

	f := newJudgeFixture(t, 3)
	f.batches.take = 2

	batch := f.submit(t, "0")
	require.True(t, batch.Batched())
	assert.Equal(t, f.ids[:2], batch.Submitted)
	assert.Equal(t, []pulid.ID{f.ids[2]}, batch.Deferred)
}

func TestSubmitJudgeBatch_LeavesOutWhatIsNotToBeJudged(t *testing.T) {
	t.Parallel()

	f := newJudgeFixture(t, 3)
	f.evaluations.stored[f.ids[0]].Checks.HardFailure = true
	f.evaluations.stored[f.ids[1]].Judge = &agentevaluation.JudgeVerdict{
		Score:     1,
		Rationale: "Done.",
	}

	batch := f.submit(t, "0")
	require.True(t, batch.Batched())
	assert.Equal(t, []pulid.ID{f.ids[2]}, batch.Submitted)
}

func TestCollectJudgeBatch_RecordsEachJudgementOnceAndRetriesTheRest(t *testing.T) {
	t.Parallel()

	f := newJudgeFixture(t, 3)
	f.batches.results = map[string]llmports.BatchItemResult{
		f.ids[0].String(): {
			CustomID: f.ids[0].String(),
			Result: &llmports.StructuredCompletionResult{
				Text:            judgeReply,
				ModelIdentifier: "claude-sonnet-5-5",
			},
		},
		f.ids[1].String(): {CustomID: f.ids[1].String(), FailureCode: "expired"},
	}
	batch := JudgeBatch{
		ProviderID:  pulid.ID("aiprv_judge"),
		BatchID:     "msgbatch_1",
		SubmittedAt: 1_800_000_100,
		Submitted:   f.ids,
	}
	collect := func() *CollectedJudgeBatch {
		collected, err := f.service.CollectJudgeBatch(t.Context(), &CollectJudgeBatchRequest{
			TenantInfo: f.tenant,
			SuiteRunID: f.run.ID,
			Batch:      batch,
		})
		require.NoError(t, err)

		return collected
	}

	first := collect()
	assert.Equal(t, 1, first.Judged)
	assert.Equal(t, []pulid.ID{f.ids[1], f.ids[2]}, first.Retry,
		"an expired item and one missing from the results are judged on their own")
	assert.Equal(t, 1, f.evaluations.updates)

	judged := f.evaluations.stored[f.ids[0]]
	require.NotNil(t, judged.Judge)
	assert.InDelta(t, 0.9, judged.Judge.Score, 0.0001)
	assert.Equal(t, "claude-sonnet-5-5", judged.Judge.Model)
	require.NotNil(t, judged.CaseScore)

	again := collect()
	assert.Equal(t, 1, again.Judged, "a retried collection counts the judgement it already kept")
	assert.Equal(t, 1, f.evaluations.updates, "and writes nothing twice")
	assert.Equal(t, 2, f.batches.reads)
}
