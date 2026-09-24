package agentqualityservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeSuiteRuns struct {
	repositories.AgentSuiteRunRepository
	runs    map[pulid.ID]*agentquality.SuiteRun
	history []*agentquality.SuiteRun
	updates int
}

func (f *fakeSuiteRuns) GetByID(
	_ context.Context,
	req repositories.GetAgentSuiteRunRequest,
) (*agentquality.SuiteRun, error) {
	run, ok := f.runs[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Suite run not found")
	}
	clone := *run

	return &clone, nil
}

func (f *fakeSuiteRuns) Update(
	_ context.Context,
	entity *agentquality.SuiteRun,
) (*agentquality.SuiteRun, error) {
	f.updates++
	clone := *entity
	f.runs[entity.ID] = &clone

	return entity, nil
}

func (f *fakeSuiteRuns) History(
	context.Context,
	repositories.AgentSuiteRunHistoryRequest,
) ([]*agentquality.SuiteRun, error) {
	return f.history, nil
}

type fakeEvaluations struct {
	repositories.AgentEvaluationRepository
	bySuite map[pulid.ID][]*agent.Evaluation
}

func (f *fakeEvaluations) ListBySuiteRun(
	_ context.Context,
	req repositories.ListSuiteEvaluationsRequest,
) ([]*agent.Evaluation, error) {
	out := make([]*agent.Evaluation, 0)
	for _, evaluation := range f.bySuite[req.SuiteRunID] {
		if *evaluation.SuiteOrdinal > req.AfterOrdinal {
			out = append(out, evaluation)
		}
	}

	return out, nil
}

func (f *fakeEvaluations) SkipPendingBySuiteRun(
	context.Context,
	repositories.SkipPendingSuiteEvaluationsRequest,
) (int, error) {
	return 0, nil
}

type fakeCaseSources struct {
	repositories.AgentEvalCaseRepository
}

func (f *fakeCaseSources) ListByAgent(
	context.Context,
	repositories.ListAgentEvalCasesRequest,
) ([]*agentquality.EvalCase, error) {
	return nil, nil
}

type fakeUsage struct {
	repositories.AIUsageRepository
}

func (f *fakeUsage) EvaluationCost(
	context.Context,
	repositories.AIUsageEvaluationCostRequest,
) (*repositories.AIUsageCost, error) {
	return &repositories.AIUsageCost{CostUSD: decimal.RequireFromString("0.42")}, nil
}

type fakeDefinitions struct {
	repositories.AgentDefinitionRepository
}

func (f *fakeDefinitions) GetByID(
	_ context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return &agentdefinition.Definition{ID: req.ID, Name: "Billing desk"}, nil
}

type fakeTower struct {
	upserts  []services.WatchtowerItemInput
	resolved []string
}

func (f *fakeTower) Upsert(_ context.Context, input services.WatchtowerItemInput) {
	f.upserts = append(f.upserts, input)
}

func (f *fakeTower) Resolve(
	_ context.Context,
	_ pagination.TenantInfo,
	_ watchtower.SourceKind,
	sourceID string,
) {
	f.resolved = append(f.resolved, sourceID)
}

type fakeNotifier struct {
	requests []notificationservice.NotifyPermittedRequest
	sent     map[string]struct{}
}

func (f *fakeNotifier) NotifyPermitted(
	_ context.Context,
	req notificationservice.NotifyPermittedRequest,
) (int, error) {
	f.requests = append(f.requests, req)
	correlation := *req.Notification.CorrelationID
	if _, dup := f.sent[correlation]; dup && req.DedupeSince > 0 {
		return 0, nil
	}
	f.sent[correlation] = struct{}{}

	return 1, nil
}

type regressionWorld struct {
	service  *Service
	runs     *fakeSuiteRuns
	tower    *fakeTower
	notifier *fakeNotifier
	run      *agentquality.SuiteRun
	baseline *agentquality.SuiteRun
}

func scoredEvaluation(caseID pulid.ID, ordinal int, score float64, hard bool) *agent.Evaluation {
	id := caseID
	suiteOrdinal := ordinal
	final := score
	if hard {
		final = 0
	}

	return &agent.Evaluation{
		ID:           pulid.MustNew(agent.EvaluationIDPrefix),
		EvalCaseID:   &id,
		Status:       agent.EvaluationStatusCompleted,
		SuiteOrdinal: &suiteOrdinal,
		CaseScore:    &final,
		Checks: &agent.CaseChecks{
			Deterministic: score,
			Final:         final,
			HardFailure:   hard,
			Passed:        !hard && final >= 0.8,
		},
	}
}

func newRegressionWorld(t *testing.T, currentHardFailure bool) *regressionWorld {
	t.Helper()

	now := int64(1_800_000_000)
	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	agentID := pulid.MustNew("agd_")
	caseIDs := make([]pulid.ID, 0, 12)
	for range 12 {
		caseIDs = append(caseIDs, pulid.MustNew("aec_"))
	}

	baselineScore := 0.9
	finished := now - 86400
	baseline := &agentquality.SuiteRun{
		ID:                agentquality.NewSuiteRunID(),
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		AgentDefinitionID: agentID,
		Status:            agentquality.SuiteRunStatusCompleted,
		QualityScore:      &baselineScore,
		CasesPassed:       12,
		StartedAt:         now - 90000,
		FinishedAt:        &finished,
	}
	baselineID := baseline.ID
	run := &agentquality.SuiteRun{
		ID:                agentquality.NewSuiteRunID(),
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		AgentDefinitionID: agentID,
		Status:            agentquality.SuiteRunStatusRunning,
		CasesTotal:        12,
		StartedAt:         now - 600,
		BaselineRunID:     &baselineID,
		FingerprintChanges: []agent.FingerprintChange{{
			Field: agent.FingerprintFieldModel, From: "model-a", To: "model-b",
		}},
	}

	baselineCases := make([]*agent.Evaluation, 0, len(caseIDs))
	currentCases := make([]*agent.Evaluation, 0, len(caseIDs))
	for idx, caseID := range caseIDs {
		baselineCases = append(baselineCases, scoredEvaluation(caseID, idx+1, 0.9, false))
		currentCases = append(currentCases,
			scoredEvaluation(caseID, idx+1, 0.9, currentHardFailure && idx == 0))
	}

	history := make([]*agentquality.SuiteRun, 0, 3)
	for idx, score := range []float64{0.9, 0.92, 0.88} {
		value := score
		at := now - int64((idx+2)*86400)
		history = append(history, &agentquality.SuiteRun{
			ID:           agentquality.NewSuiteRunID(),
			QualityScore: &value,
			CasesPassed:  12,
			StartedAt:    at,
			FinishedAt:   &at,
		})
	}

	runs := &fakeSuiteRuns{
		runs:    map[pulid.ID]*agentquality.SuiteRun{run.ID: run, baseline.ID: baseline},
		history: history,
	}
	tower := &fakeTower{}
	notifier := &fakeNotifier{sent: map[string]struct{}{}}

	return &regressionWorld{
		service: &Service{
			l:         zap.NewNop(),
			suiteRuns: runs,
			evaluations: &fakeEvaluations{bySuite: map[pulid.ID][]*agent.Evaluation{
				run.ID:      currentCases,
				baseline.ID: baselineCases,
			}},
			cases:       &fakeCaseSources{},
			usage:       &fakeUsage{},
			definitions: &fakeDefinitions{},
			watchtower:  tower,
			notifier:    notifier,
			now:         func() int64 { return now },
		},
		runs:     runs,
		tower:    tower,
		notifier: notifier,
		run:      run,
		baseline: baseline,
	}
}

func (w *regressionWorld) finalize(t *testing.T) *FinalizedSuite {
	t.Helper()

	finalized, err := w.service.FinalizeSuite(t.Context(), &FinalizeSuiteRequest{
		TenantInfo: pagination.TenantInfo{OrgID: w.run.OrganizationID, BuID: w.run.BusinessUnitID},
		SuiteRunID: w.run.ID,
		Settings: agentquality.SuiteSettings{
			RegressionThreshold: 0.1,
			MinCases:            10,
		},
	})
	require.NoError(t, err)

	return finalized
}

func TestFinalizeSuite_RaisesARegressionOnceAndSaysWhatChanged(t *testing.T) {
	t.Parallel()

	world := newRegressionWorld(t, true)

	finalized := world.finalize(t)
	assert.True(t, finalized.Regression)
	assert.Equal(t, agentquality.SuiteRunStatusCompleted, finalized.Status)
	assert.Equal(t, 1, finalized.HardFailures)

	require.Len(t, world.tower.upserts, 1)
	item := world.tower.upserts[0]
	assert.Equal(t, watchtower.SourceAgentQualityRegression, item.SourceKind)
	assert.Equal(t, world.run.ID.String(), item.SourceID, "the item is keyed on the suite run")
	assert.Equal(t, watchtower.SeverityCritical, item.Severity, "a hard failure is high")
	assert.Contains(t, item.Summary, "model changed from model-a to model-b")
	assert.Contains(t, item.Title, "Billing desk")
	assert.Contains(t, world.tower.resolved, world.baseline.ID.String(),
		"the previous run's item is closed")

	require.Len(t, world.notifier.requests, 1)
	notice := world.notifier.requests[0]
	assert.Equal(t, permission.ResourceAgentControl, notice.Resource)
	assert.Equal(t, permission.OpUpdate, notice.Operation)
	assert.Equal(t, world.run.ID.String(), *notice.Notification.CorrelationID)
	assert.Positive(t, notice.DedupeSince)
	assert.Equal(t, notification.PriorityHigh, notice.Notification.Priority)
	assert.Equal(t, EventQualityRegression, notice.Notification.EventType)
	assert.Equal(t, []string{"model"}, notice.Notification.Data["changes"])

	stored := world.runs.runs[world.run.ID]
	assert.True(t, stored.Regression)
	assert.Equal(t, agentquality.SuiteRunStatusCompleted, stored.Status)
	assert.Equal(t, "0.42", stored.CostUSD.StringFixed(2))
	require.NotNil(t, stored.BaselineScore)
	assert.InDelta(t, 0.9, *stored.BaselineScore, 1e-9)

	again := world.finalize(t)
	assert.True(t, again.Regression)
	assert.Len(t, world.tower.upserts, 1, "a finalized run is not raised again")
	assert.Len(t, world.notifier.requests, 1, "a finalized run notifies nobody again")
}

func TestFinalizeSuite_ASteadyRunRaisesNothing(t *testing.T) {
	t.Parallel()

	world := newRegressionWorld(t, false)

	finalized := world.finalize(t)
	assert.False(t, finalized.Regression)
	assert.Empty(t, world.tower.upserts)
	assert.Empty(t, world.notifier.requests)
	assert.Contains(t, world.tower.resolved, world.baseline.ID.String(),
		"a steady run closes whatever the previous run raised")
}

func TestNotifyRegression_IsKeyedOnTheRun(t *testing.T) {
	t.Parallel()

	world := newRegressionWorld(t, true)
	run := world.run
	run.Regression = true
	run.HardFailures = 1
	item := services.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{OrgID: run.OrganizationID, BuID: run.BusinessUnitID},
		Severity:   watchtower.SeverityWarning,
		Title:      "Billing desk answers its evaluation cases worse than before",
	}

	world.service.notifyRegression(t.Context(), run, item)
	world.service.notifyRegression(t.Context(), run, item)

	require.Len(t, world.notifier.requests, 2)
	assert.Equal(t,
		*world.notifier.requests[0].Notification.CorrelationID,
		*world.notifier.requests[1].Notification.CorrelationID,
	)
	assert.Len(t, world.notifier.sent, 1, "one notice goes out")
	assert.Equal(t, notification.PriorityMedium, world.notifier.requests[0].Notification.Priority)
}
