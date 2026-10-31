package insightservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice/detector"
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice/narrator"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const narrationReply = `{"insights":[{"dedupeKey":"ontime:cus_1",` +
	`"headline":"Acme Foods is arriving late more often",` +
	`"narrative":"On-time delivery sits at 82.4%.",` +
	`"recommendation":"Review the appointment windows on its lanes."}]}`

// storingRepo stores findings the way the real repository does: each is
// given its id as it is inserted.
type storingRepo struct {
	stubRepo
}

func (r *storingRepo) ReplaceDetectorFindings(
	ctx context.Context,
	req insightports.ReplaceDetectorFindingsRequest,
) (insightports.ReplaceDetectorFindingsResult, error) {
	for _, entity := range req.Insights {
		entity.ID = pulid.MustNew("inst_")
	}

	return r.stubRepo.ReplaceDetectorFindings(ctx, req)
}

type answeringNova struct {
	text  string
	calls int
}

func (n *answeringNova) Available(context.Context, aitask.Task, pagination.TenantInfo) (bool, error) {
	return true, nil
}

func (n *answeringNova) Run(
	_ context.Context,
	task *llmports.NovaTask,
	out any,
) (*llmports.StructuredCompletionResult, error) {
	n.calls++
	result := &llmports.StructuredCompletionResult{Text: n.text, ModelIdentifier: "haiku"}

	return result, llmports.DecodeAnswer(task.Name, n.text, out)
}

type stubBatches struct {
	submitted *llmports.NovaBatch
	ended     bool
	results   map[string]llmports.BatchItemResult
}

func (b *stubBatches) SubmitTasks(
	_ context.Context,
	batch *llmports.NovaBatch,
) (*llmports.BatchHandle, error) {
	b.submitted = batch

	return &llmports.BatchHandle{BatchID: "msgbatch_1", ProviderID: pulid.MustNew("aip_")}, nil
}

func (b *stubBatches) BatchStatus(
	context.Context,
	*llmports.BatchStatusRequest,
) (*llmports.BatchStatus, error) {
	return &llmports.BatchStatus{Ended: b.ended}, nil
}

func (*stubBatches) CancelBatch(context.Context, *llmports.BatchCancelRequest) error {
	return nil
}

func (b *stubBatches) BatchResults(
	context.Context,
	*llmports.BatchResultsRequest,
) (map[string]llmports.BatchItemResult, error) {
	return b.results, nil
}

func narratingService(
	repo insightports.InsightRepository,
	nova llmports.TaskRunner,
	batches llmports.TaskBatcher,
	ds ...detector.Detector,
) *Service {
	return &Service{
		l:         zap.NewNop(),
		repo:      repo,
		detectors: detector.NewRegistry(ds...),
		narrator: narrator.New(narrator.Params{
			Logger:  zap.NewNop(),
			Nova:    nova,
			Batches: batches,
		}),
		permissions: &stubPermissions{},
	}
}

func ontime(findings ...detector.Finding) *stubDetector {
	return &stubDetector{key: "ontime", category: insight.CategoryServiceQuality, findings: findings}
}

func TestRefresh_KeepsTheWordingOfAFindingWhoseFiguresHaveNotMoved(t *testing.T) {
	t.Parallel()

	same := testFinding("ontime:cus_1")
	moved := testFinding("ontime:cus_2")
	repo := &stubRepo{narrated: []*insight.Insight{
		{
			DedupeKey: same.DedupeKey, Subject: same.Subject, Severity: same.Severity,
			Metrics: same.Metrics, Headline: "Acme Foods is arriving late more often",
			Narrated: true, ModelIdentifier: "haiku",
		},
		{
			DedupeKey: moved.DedupeKey, Subject: moved.Subject, Severity: moved.Severity,
			Metrics: []insight.Metric{detector.Percent(
				"onTimePercent", "On-time delivery", decimal.NewFromFloat(91),
				insight.DirectionLowerIsWorse,
			)},
			Headline: "Old wording about 91%", Narrated: true,
		},
	}}
	nova := &answeringNova{text: `{"insights":[]}`}
	svc := narratingService(repo, nova, nil, ontime(same, moved))

	result, err := svc.Refresh(t.Context(), RefreshRequest{TenantInfo: tenant(), Now: 1_800_000_000})
	require.NoError(t, err)

	stored := repo.replaced[0].Insights
	require.Len(t, stored, 2)
	assert.Equal(t, "Acme Foods is arriving late more often", stored[0].Headline)
	assert.True(t, stored[0].Narrated)
	assert.Equal(t, "haiku", stored[0].ModelIdentifier)
	assert.Equal(t, moved.Headline, stored[1].Headline, "wording for old figures is not kept")
	assert.False(t, stored[1].Narrated)
	assert.Equal(t, 1, nova.calls, "only the finding whose figures moved is narrated")
	assert.Equal(t, 1, result.Narrated)
}

func TestRefresh_DeferredSendsNewFindingsToTheBatchAndStoresTheirOwnWording(t *testing.T) {
	t.Parallel()

	finding := testFinding("ontime:cus_1")
	repo := &storingRepo{}
	batches := &stubBatches{}
	nova := &answeringNova{text: narrationReply}
	svc := narratingService(repo, nova, batches, ontime(finding))

	result, err := svc.Refresh(t.Context(), RefreshRequest{
		TenantInfo: tenant(), Now: 1_800_000_000, Deferred: true,
	})
	require.NoError(t, err)

	stored := repo.replaced[0].Insights
	assert.Equal(t, finding.Headline, stored[0].Headline)
	assert.False(t, stored[0].Narrated)
	assert.Zero(t, nova.calls, "nothing is narrated one at a time")
	require.NotNil(t, batches.submitted)
	assert.Equal(t, aitask.TaskOperationalInsights, batches.submitted.Task)
	require.Len(t, batches.submitted.Items, 1)
	require.NotNil(t, result.Pending)
	assert.Equal(t, "msgbatch_1", result.Pending.Handle.BatchID)
	assert.Equal(t, []pulid.ID{stored[0].ID}, result.Pending.InsightIDs)
}

func TestRefresh_DeferredNarratesNowWhereTheProviderCannotBatch(t *testing.T) {
	t.Parallel()

	repo := &storingRepo{}
	nova := &answeringNova{text: narrationReply}
	svc := narratingService(repo, nova, nil, ontime(testFinding("ontime:cus_1")))

	result, err := svc.Refresh(t.Context(), RefreshRequest{
		TenantInfo: tenant(), Now: 1_800_000_000, Deferred: true,
	})
	require.NoError(t, err)

	assert.Nil(t, result.Pending)
	assert.Equal(t, 1, nova.calls)
	require.Len(t, repo.applied, 1)
	assert.Equal(t, "Acme Foods is arriving late more often", repo.applied[0].Headline)
	assert.Equal(t, "haiku", repo.applied[0].ModelIdentifier)
	assert.Equal(t, 1, result.Narrated)
}

func TestApplyNarration_WordsOnlyFindingsStillActiveAndUnworded(t *testing.T) {
	t.Parallel()

	waiting := testFinding("ontime:cus_1")
	repo := &storingRepo{}
	batches := &stubBatches{ended: true, results: map[string]llmports.BatchItemResult{
		"narration-0": {CustomID: "narration-0", Result: &llmports.StructuredCompletionResult{
			Text: narrationReply, ModelIdentifier: "haiku",
		}},
	}}
	svc := narratingService(repo, &answeringNova{}, batches, ontime(waiting))

	result, err := svc.Refresh(t.Context(), RefreshRequest{
		TenantInfo: tenant(), Now: 1_800_000_000, Deferred: true,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Pending)
	stored := repo.replaced[0].Insights[0]
	stored.Status = insight.StatusActive

	ended, err := svc.NarrationEnded(t.Context(), tenant(), result.Pending)
	require.NoError(t, err)
	assert.True(t, ended)

	worded, err := svc.ApplyNarration(t.Context(), tenant(), result.Pending)
	require.NoError(t, err)

	assert.Equal(t, 1, worded)
	require.Len(t, repo.applied, 1)
	assert.Equal(t, stored.ID, repo.applied[0].ID)
	assert.Equal(t, "On-time delivery sits at 82.4%.", repo.applied[0].Narrative)

	stored.Status = insight.StatusSuperseded
	stored.Narrated = false
	worded, err = svc.ApplyNarration(t.Context(), tenant(), result.Pending)
	require.NoError(t, err)
	assert.Zero(t, worded, "a finding a later refresh superseded keeps what it had")
}
