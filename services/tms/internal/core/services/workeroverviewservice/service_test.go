package workeroverviewservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/internal/core/services/workeroverviewservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeCredentials struct {
	calls   int
	summary *worker.WorkerCredentialSummary
}

func (f *fakeCredentials) Summary(
	_ context.Context, _ pagination.TenantInfo, _ pulid.ID,
) (*worker.WorkerCredentialSummary, error) {
	f.calls++
	return f.summary, nil
}

type fakeTraining struct {
	calls   int
	summary *worker.WorkerTrainingSummary
}

func (f *fakeTraining) Summary(
	_ context.Context, _ pagination.TenantInfo, _ pulid.ID,
) (*worker.WorkerTrainingSummary, error) {
	f.calls++
	return f.summary, nil
}

type fakeSafety struct {
	calls int
	card  *worker.SafetyScorecard
	err   error
}

func (f *fakeSafety) Scorecard(
	_ context.Context, _ pagination.TenantInfo, _ pulid.ID,
) (*worker.SafetyScorecard, error) {
	f.calls++
	return f.card, f.err
}

type fakeChecklists struct {
	calls      int
	checklists []*worker.WorkerChecklist
}

func (f *fakeChecklists) ListForWorker(
	_ context.Context, _ pagination.TenantInfo, _ pulid.ID, _ bool,
) ([]*worker.WorkerChecklist, error) {
	f.calls++
	return f.checklists, nil
}

type fakeBalances struct {
	calls    int
	balances []*ptoledgerservice.BalanceView
	summary  *repositories.PTOBalanceSummary
}

func (f *fakeBalances) GetBalances(
	_ context.Context, _ pagination.TenantInfo, _ pulid.ID,
) ([]*ptoledgerservice.BalanceView, error) {
	f.calls++
	return f.balances, nil
}

func (f *fakeBalances) Summary(
	_ context.Context, _ pagination.TenantInfo,
) (*repositories.PTOBalanceSummary, error) {
	if f.summary == nil {
		return &repositories.PTOBalanceSummary{}, nil
	}
	return f.summary, nil
}

type fakeReviews struct {
	calls   int
	reviews []*worker.PerformanceReview
}

func (f *fakeReviews) ListReviews(
	_ context.Context, _ pagination.TenantInfo, _ pulid.ID, _ []worker.ReviewStatus,
) ([]*worker.PerformanceReview, error) {
	f.calls++
	return f.reviews, nil
}

type fakeReviewCounter struct {
	count    int
	err      error
	statuses []worker.ReviewStatus
}

func (f *fakeReviewCounter) CountReviews(
	_ context.Context,
	req *repositories.CountPerformanceReviewsRequest,
) (int, error) {
	f.statuses = req.Statuses
	return f.count, f.err
}

type overviewHarness struct {
	svc          *workeroverviewservice.Service
	credentials  *fakeCredentials
	training     *fakeTraining
	safety       *fakeSafety
	checklists   *fakeChecklists
	pto          *fakeBalances
	reviews      *fakeReviews
	reviewCounts *fakeReviewCounter
	workerRepo   *mocks.MockWorkerRepository
	tenant       pagination.TenantInfo
	wrk          *worker.Worker
}

func newOverviewHarness(t *testing.T) *overviewHarness {
	t.Helper()
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	wrk := &worker.Worker{
		ID:             pulid.MustNew("wrk_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Status:         domaintypes.StatusActive,
		CanBeAssigned:  true,
	}

	workerRepo := mocks.NewMockWorkerRepository(t)
	workerRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(wrk, nil).Maybe()

	h := &overviewHarness{
		credentials: &fakeCredentials{
			summary: &worker.WorkerCredentialSummary{
				ComplianceStatus: worker.ComplianceStatusCompliant,
			},
		},
		training:   &fakeTraining{summary: &worker.WorkerTrainingSummary{Compliant: true}},
		safety:     &fakeSafety{card: &worker.SafetyScorecard{Rating: worker.SafetyRatingGood}},
		checklists: &fakeChecklists{},
		pto:        &fakeBalances{},
		reviews:    &fakeReviews{},
		tenant:     tenant,
		wrk:        wrk,
		workerRepo: workerRepo,
	}
	h.reviewCounts = &fakeReviewCounter{}

	h.svc = workeroverviewservice.NewWithDeps(workeroverviewservice.Deps{
		WorkerRepo:   workerRepo,
		ReviewCounts: h.reviewCounts,
		Credentials:  h.credentials,
		Training:     h.training,
		Safety:       h.safety,
		Checklists:   h.checklists,
		PTO:          h.pto,
		Reviews:      h.reviews,
	})
	return h
}

func (h *overviewHarness) get(
	t *testing.T,
	sections workeroverviewservice.Sections,
) *workeroverviewservice.Overview {
	t.Helper()
	overview, err := h.svc.Get(t.Context(), h.tenant, h.wrk.ID, sections)
	require.NoError(t, err)
	return overview
}

func TestGet_ComposesEverySectionWhenAllowed(t *testing.T) {
	h := newOverviewHarness(t)

	overview := h.get(t, workeroverviewservice.All())

	assert.Equal(t, worker.StandingGood, overview.Standing)
	assert.NotNil(t, overview.Credentials)
	assert.NotNil(t, overview.Training)
	assert.NotNil(t, overview.Safety)
	assert.Positive(t, overview.AsOf)
}

// A section the caller cannot read must not be queried at all. Reading it and
// discarding the answer would leak the work and, worse, could surface data
// through timing or an audit trail the user has no right to touch.
func TestGet_SkipsSectionsTheCallerCannotRead(t *testing.T) {
	h := newOverviewHarness(t)

	overview := h.get(t, workeroverviewservice.Sections{Credentials: true})

	assert.Equal(t, 1, h.credentials.calls)
	assert.Zero(t, h.training.calls)
	assert.Zero(t, h.safety.calls)
	assert.Zero(t, h.checklists.calls)
	assert.Zero(t, h.pto.calls)
	assert.Zero(t, h.reviews.calls)
	assert.Nil(t, overview.Safety)
	assert.Nil(t, overview.Training)
}

// A worker whose safety record is bad but whose viewer cannot see safety must
// not be reported as at risk on evidence the viewer is not allowed to read.
func TestGet_StandingIgnoresSectionsTheCallerCannotRead(t *testing.T) {
	h := newOverviewHarness(t)
	h.safety.card = &worker.SafetyScorecard{Rating: worker.SafetyRatingAtRisk, Score: 20}

	withSafety := h.get(t, workeroverviewservice.All())
	assert.Equal(t, worker.StandingAtRisk, withSafety.Standing)

	withoutSafety := h.get(t, workeroverviewservice.Sections{Credentials: true, Training: true})
	assert.Equal(t, worker.StandingGood, withoutSafety.Standing)
	assert.Empty(t, withoutSafety.Concerns)
}

// A missing section would read as "nothing wrong here", so a failure has to
// fail the whole query rather than degrade quietly.
func TestGet_SectionFailureFailsTheQuery(t *testing.T) {
	h := newOverviewHarness(t)
	h.safety.err = errors.New("safety repo is down")

	_, err := h.svc.Get(t.Context(), h.tenant, h.wrk.ID, workeroverviewservice.All())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "safety repo is down")
}

// A worker can have several checklists running at once. The live one is the
// most recently started; closed ones are history and must not be offered.
func TestGet_PicksTheNewestOpenChecklist(t *testing.T) {
	h := newOverviewHarness(t)
	h.checklists.checklists = []*worker.WorkerChecklist{
		{Name: "Older onboarding", Status: worker.ChecklistStatusOpen, StartedAt: 1_000},
		{Name: "Equipment handover", Status: worker.ChecklistStatusOpen, StartedAt: 9_000},
		{Name: "Closed one", Status: worker.ChecklistStatusCompleted, StartedAt: 99_000},
	}

	overview := h.get(t, workeroverviewservice.All())

	require.NotNil(t, overview.Checklist)
	assert.Equal(t, "Equipment handover", overview.Checklist.Name)
}

func TestGet_NoOpenChecklistLeavesItNil(t *testing.T) {
	h := newOverviewHarness(t)
	h.checklists.checklists = []*worker.WorkerChecklist{
		{Name: "Closed one", Status: worker.ChecklistStatusCompleted, StartedAt: 99_000},
	}

	assert.Nil(t, h.get(t, workeroverviewservice.All()).Checklist)
}

// The due date comes from the last review that closed. A review already under
// way means the next one is not also outstanding, so the date is withheld and
// the standing never calls an in-flight review overdue.
func TestGet_ReviewInFlightSuppressesTheDueDate(t *testing.T) {
	h := newOverviewHarness(t)
	due := int64(1_000)
	h.reviews.reviews = []*worker.PerformanceReview{
		{Status: worker.ReviewStatusClosed, PeriodEnd: 500, NextReviewAt: &due},
		{Status: worker.ReviewStatusDraft, PeriodEnd: 900},
	}

	overview := h.get(t, workeroverviewservice.All())

	require.NotNil(t, overview.OpenReview)
	require.NotNil(t, overview.LastReview)
	assert.Nil(t, overview.NextReviewAt)
	assert.Equal(t, worker.StandingGood, overview.Standing)
}

func TestGet_LastClosedReviewSuppliesTheDueDate(t *testing.T) {
	h := newOverviewHarness(t)
	older := int64(1_000)
	newer := int64(2_000)
	h.reviews.reviews = []*worker.PerformanceReview{
		{Status: worker.ReviewStatusClosed, PeriodEnd: 500, NextReviewAt: &older},
		{Status: worker.ReviewStatusClosed, PeriodEnd: 900, NextReviewAt: &newer},
	}

	overview := h.get(t, workeroverviewservice.All())

	require.NotNil(t, overview.NextReviewAt)
	assert.Equal(t, newer, *overview.NextReviewAt)
	assert.Nil(t, overview.OpenReview)
}

// The home tile reads three unrelated areas. Each has to be asked, and the
// numbers have to arrive on the one object the tile renders.
func TestRosterAttention_GathersEveryArea(t *testing.T) {
	h := newOverviewHarness(t)
	h.workerRepo.EXPECT().
		CountRosterAttention(mock.Anything, mock.Anything).
		Return(&repositories.RosterAttention{
			ActiveWorkers:   40,
			NonCompliant:    3,
			TrainingOverdue: 7,
			AtRisk:          1,
			ExpiringSoon:    5,
		}, nil).
		Once()
	h.reviewCounts.count = 4
	h.pto.summary = &repositories.PTOBalanceSummary{
		TotalBalanceDays: decimal.NewFromInt(312),
	}

	attention, err := h.svc.RosterAttention(t.Context(), h.tenant)
	require.NoError(t, err)

	assert.Equal(t, 40, attention.ActiveWorkers)
	assert.Equal(t, 3, attention.NonCompliant)
	assert.Equal(t, 7, attention.TrainingOverdue)
	assert.Equal(t, 1, attention.AtRisk)
	assert.Equal(t, 5, attention.ExpiringSoon)
	assert.Equal(t, 4, attention.ReviewsAwaitingSignOff)
	assert.True(t, decimal.NewFromInt(312).Equal(attention.PTOLiabilityDays))
}

// A draft review is still being written; it is its author's work in progress,
// not a queue anyone else can clear. Counting it would tell the office to
// chase people who have nothing to do.
func TestRosterAttention_CountsOnlySubmittedReviews(t *testing.T) {
	h := newOverviewHarness(t)
	h.workerRepo.EXPECT().
		CountRosterAttention(mock.Anything, mock.Anything).
		Return(&repositories.RosterAttention{}, nil).
		Once()

	_, err := h.svc.RosterAttention(t.Context(), h.tenant)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]worker.ReviewStatus{worker.ReviewStatusSubmitted},
		h.reviewCounts.statuses,
	)
}
