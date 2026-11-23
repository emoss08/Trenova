package workeroverviewservice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paybenefit"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeroverviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/portalinvitation"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/safetyscorecard"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercompliance"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/portalaccessports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/ptoledgerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/selfserviceports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerclearanceports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workeremploymentports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerleaveports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerreviewports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerscheduleports"
	"github.com/emoss08/trenova/internal/testutil/mocks/workerportsmock"
	"github.com/emoss08/trenova/internal/testutil/orgzonetest"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testNow = int64(1_800_000_000)

type fakeCredentials struct {
	calls   int
	asOf    int64
	summary *workercredential.WorkerCredentialSummary
}

func (f *fakeCredentials) SummaryFor(
	_ context.Context, _ pagination.TenantInfo, _ *worker.Worker, asOf int64,
) (*workercredential.WorkerCredentialSummary, error) {
	f.calls++
	f.asOf = asOf
	return f.summary, nil
}

type fakeTraining struct {
	calls   int
	summary *workertraining.WorkerTrainingSummary
}

func (f *fakeTraining) SummaryFor(
	_ context.Context, _ pagination.TenantInfo, _ *worker.Worker, _ int64,
) (*workertraining.WorkerTrainingSummary, error) {
	f.calls++
	return f.summary, nil
}

type fakeSafety struct {
	calls      int
	discipline bool
	card       *safetyscorecard.SafetyScorecard
	err        error
}

func (f *fakeSafety) Evidence(
	_ context.Context, req *workersafetyservice.EvidenceRequest,
) (*workersafetyservice.Evidence, error) {
	f.calls++
	f.discipline = req.Discipline
	return &workersafetyservice.Evidence{Scorecard: f.card}, f.err
}

type fakeChecklists struct {
	closed     int
	calls      int
	checklists []*workerchecklist.WorkerChecklist
}

func (f *fakeChecklists) CountClosed(
	_ context.Context, _ pagination.TenantInfo, _ pulid.ID,
) (int, error) {
	return f.closed, nil
}

func (f *fakeChecklists) ListForWorker(
	_ context.Context, _ pagination.TenantInfo, _ pulid.ID, _ bool,
) ([]*workerchecklist.WorkerChecklist, error) {
	f.calls++
	return f.checklists, nil
}

type fakeBalances struct {
	calls        int
	summaryCalls int
	balances     []*ptoledgerservice.BalanceView
	summary      *ptoledgerports.PTOBalanceSummary
}

func (f *fakeBalances) GetBalances(
	_ context.Context, _ pagination.TenantInfo, _ pulid.ID,
) ([]*ptoledgerservice.BalanceView, error) {
	f.calls++
	return f.balances, nil
}

func (f *fakeBalances) Summary(
	_ context.Context, _ pagination.TenantInfo,
) (*ptoledgerports.PTOBalanceSummary, error) {
	f.summaryCalls++
	if f.summary == nil {
		return &ptoledgerports.PTOBalanceSummary{}, nil
	}
	return f.summary, nil
}

type fakeReviews struct {
	calls   int
	reviews []*workerreview.PerformanceReview
}

func (f *fakeReviews) ListReviews(
	_ context.Context, _ pagination.TenantInfo, _ pulid.ID, _ []workerreview.ReviewStatus,
) ([]*workerreview.PerformanceReview, error) {
	f.calls++
	return f.reviews, nil
}

type fakeReviewCounter struct {
	count    int
	err      error
	statuses []workerreview.ReviewStatus
}

func (f *fakeReviewCounter) CountReviews(
	_ context.Context,
	req *workerreviewports.CountPerformanceReviewsRequest,
) (int, error) {
	f.statuses = req.Statuses
	return f.count, f.err
}

type fakeLeave struct {
	count       int
	req         *workerleaveports.ListLeaveCasesRequest
	calls       int
	entitlement workerleave.LeaveEntitlement
}

func (f *fakeLeave) CountCases(
	_ context.Context,
	req *workerleaveports.ListLeaveCasesRequest,
) (int, error) {
	f.req = req
	return f.count, nil
}

func (f *fakeLeave) EntitlementFor(
	_ context.Context, _ pagination.TenantInfo, _ *worker.Worker, _ int64,
) (workerleave.LeaveEntitlement, error) {
	f.calls++
	return f.entitlement, nil
}

type fakeDrugAlcohol struct {
	calls    int
	standing workerdot.DrugAlcoholStanding
}

func (f *fakeDrugAlcohol) Standing(
	_ context.Context, _ pagination.TenantInfo, _ pulid.ID,
) (workerdot.DrugAlcoholStanding, error) {
	f.calls++
	return f.standing, nil
}

type fakeQualification struct {
	calls int
	req   *workerdqfservice.FileForRequest
	parts *workerdqfservice.FileParts
}

func (f *fakeQualification) FileFor(
	_ context.Context, req *workerdqfservice.FileForRequest,
) (*workerdqfservice.FileParts, error) {
	f.calls++
	f.req = req
	return f.parts, nil
}

type fakeEvents struct {
	calls   int
	summary *workeremploymentports.EmploymentEventSummary
}

func (f *fakeEvents) Summarize(
	_ context.Context, _ *workeremploymentports.SummarizeWorkerEmploymentEventsRequest,
) (*workeremploymentports.EmploymentEventSummary, error) {
	f.calls++
	if f.summary == nil {
		return &workeremploymentports.EmploymentEventSummary{}, nil
	}
	return f.summary, nil
}

type fakeShifts struct {
	calls       int
	assignments []*workerschedule.WorkerShiftAssignment
}

func (f *fakeShifts) ListAssignments(
	_ context.Context, _ *workerscheduleports.ListShiftAssignmentsRequest,
) ([]*workerschedule.WorkerShiftAssignment, error) {
	f.calls++
	return f.assignments, nil
}

type fakeHOS struct {
	calls int
	state *telematics.HOSLinkState
}

func (f *fakeHOS) WorkerHOSLink(
	_ context.Context, _ pagination.TenantInfo, _ pulid.ID,
) (*telematics.HOSLinkState, error) {
	f.calls++
	return f.state, nil
}

type fakePay struct {
	calls      int
	assignment *payassignment.WorkerPayAssignment
}

func (f *fakePay) GetEffectiveForWorker(
	_ context.Context, _ driverpayports.GetWorkerPayAssignmentRequest,
) (*payassignment.WorkerPayAssignment, error) {
	f.calls++
	if f.assignment == nil {
		return nil, errortypes.NewNotFoundError("no pay assignment")
	}
	return f.assignment, nil
}

type fakeBenefits struct {
	calls       int
	plans       []*paybenefit.BenefitPlan
	enrollments []*paybenefit.WorkerBenefitEnrollment
}

func (f *fakeBenefits) ListPlans(
	_ context.Context, _ *driverpayports.ListBenefitPlansRequest,
) ([]*paybenefit.BenefitPlan, error) {
	f.calls++
	return f.plans, nil
}

func (f *fakeBenefits) ListEnrollments(
	_ context.Context, _ *driverpayports.ListBenefitEnrollmentsRequest,
) ([]*paybenefit.WorkerBenefitEnrollment, error) {
	return f.enrollments, nil
}

type fakeInvitations struct {
	calls      int
	invitation *portalinvitation.PortalInvitation
}

func (f *fakeInvitations) GetPendingInvitationForWorker(
	_ context.Context, _ portalaccessports.GetPendingPortalInvitationRequest,
) (*portalinvitation.PortalInvitation, error) {
	f.calls++
	if f.invitation == nil {
		return nil, errortypes.NewNotFoundError("no invitation")
	}
	return f.invitation, nil
}

type fakeProfileChanges struct {
	calls    int
	requests []*profilechange.WorkerProfileChangeRequest
}

func (f *fakeProfileChanges) ListChangeRequests(
	_ context.Context, _ *selfserviceports.ListProfileChangeRequestsRequest,
) ([]*profilechange.WorkerProfileChangeRequest, error) {
	f.calls++
	return f.requests, nil
}

type fakeDocuments struct {
	calls int
	req   *documentports.CountDocumentsByResourceRequest
	count int
}

func (f *fakeDocuments) CountByResource(
	_ context.Context, req *documentports.CountDocumentsByResourceRequest,
) (int, error) {
	f.calls++
	f.req = req
	return f.count, nil
}

type fakeMemories struct {
	calls int
	req   *agentmemoryports.ListDeskMemoriesRequest
	all   int
}

func (f *fakeMemories) ListDesk(
	_ context.Context, req *agentmemoryports.ListDeskMemoriesRequest,
) (*agentmemoryports.DeskMemoryPage, error) {
	f.calls++
	f.req = req
	return &agentmemoryports.DeskMemoryPage{All: f.all}, nil
}

type fakeDispatch struct {
	calls    int
	req      *workerclearanceports.DispatchRequest
	verdicts *workercompliance.DispatchVerdicts
}

func (f *fakeDispatch) Verdicts(
	_ context.Context, req *workerclearanceports.DispatchRequest,
) (*workercompliance.DispatchVerdicts, error) {
	f.calls++
	f.req = req
	if f.verdicts == nil {
		return &workercompliance.DispatchVerdicts{}, nil
	}
	return f.verdicts, nil
}

type fakeTractors struct {
	calls int
	req   tractorports.GetTractorByIDRequest
	unit  *tractor.Tractor
}

func (f *fakeTractors) GetByID(
	_ context.Context, req tractorports.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	f.calls++
	f.req = req
	if f.unit == nil {
		return nil, errortypes.NewNotFoundError("no tractor")
	}
	return f.unit, nil
}

type fakeTimeOff struct {
	calls int
	req   *workerptoports.ListPTORequest
	items []*worker.WorkerPTO
}

func (f *fakeTimeOff) List(
	_ context.Context, req *workerptoports.ListPTORequest,
) (*pagination.CursorListResult[*worker.WorkerPTO], error) {
	f.calls++
	f.req = req
	return &pagination.CursorListResult[*worker.WorkerPTO]{Items: f.items}, nil
}

type overviewHarness struct {
	svc            *workeroverviewservice.Service
	credentials    *fakeCredentials
	training       *fakeTraining
	safety         *fakeSafety
	checklists     *fakeChecklists
	pto            *fakeBalances
	reviews        *fakeReviews
	reviewCounts   *fakeReviewCounter
	leave          *fakeLeave
	drugAlcohol    *fakeDrugAlcohol
	qualification  *fakeQualification
	events         *fakeEvents
	shifts         *fakeShifts
	hos            *fakeHOS
	pay            *fakePay
	benefits       *fakeBenefits
	invitations    *fakeInvitations
	profileChanges *fakeProfileChanges
	documents      *fakeDocuments
	memories       *fakeMemories
	dispatch       *fakeDispatch
	tractors       *fakeTractors
	timeOff        *fakeTimeOff
	workerRepo     *workerportsmock.MockWorkerRepository
	tenant         pagination.TenantInfo
	wrk            *worker.Worker
}

type harnessOptions struct {
	features platformcatalogports.FeatureGate
	zone     string
	now      int64
}

func newOverviewHarness(t *testing.T, opts ...func(*harnessOptions)) *overviewHarness {
	t.Helper()
	options := &harnessOptions{features: plantest.AllFeatures(), now: testNow}
	for _, opt := range opts {
		opt(options)
	}
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	wrk := &worker.Worker{
		ID:                   pulid.MustNew("wrk_"),
		OrganizationID:       tenantInfo.OrgID,
		BusinessUnitID:       tenantInfo.BuID,
		Status:               domaintypes.StatusActive,
		CanBeAssigned:        true,
		AvailableForDispatch: true,
		Profile:              &worker.WorkerProfile{},
	}

	workerRepo := workerportsmock.NewMockWorkerRepository(t)
	workerRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(wrk, nil).Maybe()

	h := &overviewHarness{
		credentials: &fakeCredentials{
			summary: &workercredential.WorkerCredentialSummary{
				ComplianceStatus: worker.ComplianceStatusCompliant,
			},
		},
		training: &fakeTraining{summary: &workertraining.WorkerTrainingSummary{Compliant: true}},
		safety: &fakeSafety{
			card: &safetyscorecard.SafetyScorecard{Rating: worker.SafetyRatingGood},
		},
		checklists:   &fakeChecklists{},
		pto:          &fakeBalances{},
		reviews:      &fakeReviews{},
		reviewCounts: &fakeReviewCounter{},
		leave:        &fakeLeave{},
		drugAlcohol: &fakeDrugAlcohol{
			standing: workerdot.DrugAlcoholStanding{Status: worker.DrugAlcoholClear},
		},
		qualification: &fakeQualification{
			parts: &workerdqfservice.FileParts{File: &workercompliance.DQFFile{Complete: true}},
		},
		events: &fakeEvents{},
		shifts: &fakeShifts{},
		hos: &fakeHOS{
			state: &telematics.HOSLinkState{Link: telematics.HOSLinkNotLinked},
		},
		pay:            &fakePay{},
		benefits:       &fakeBenefits{},
		invitations:    &fakeInvitations{},
		profileChanges: &fakeProfileChanges{},
		documents:      &fakeDocuments{count: 4},
		memories:       &fakeMemories{all: 2},
		dispatch:       &fakeDispatch{},
		tractors:       &fakeTractors{},
		timeOff:        &fakeTimeOff{},
		tenant:         tenantInfo,
		wrk:            wrk,
		workerRepo:     workerRepo,
	}

	h.svc = workeroverviewservice.New(workeroverviewservice.Params{
		Logger:         zap.NewNop(),
		Organizations:  orgzonetest.Organizations{Timezone: options.zone},
		Clock:          func() int64 { return options.now },
		Features:       options.features,
		WorkerRepo:     workerRepo,
		ReviewCounts:   h.reviewCounts,
		Credentials:    h.credentials,
		Training:       h.training,
		Safety:         h.safety,
		Checklists:     h.checklists,
		PTO:            h.pto,
		Reviews:        h.reviews,
		Leave:          h.leave,
		LeaveStanding:  h.leave,
		DrugAlcohol:    h.drugAlcohol,
		Qualification:  h.qualification,
		Employment:     h.events,
		Shifts:         h.shifts,
		HOS:            h.hos,
		Tractors:       h.tractors,
		TimeOff:        h.timeOff,
		Pay:            h.pay,
		Benefits:       h.benefits,
		Invitations:    h.invitations,
		ProfileChanges: h.profileChanges,
		Documents:      h.documents,
		Memories:       h.memories,
		Dispatch:       h.dispatch,
	})
	return h
}

func withFeatures(gate platformcatalogports.FeatureGate) func(*harnessOptions) {
	return func(o *harnessOptions) { o.features = gate }
}

func memoryActor() *agentmemoryports.DeskMemoryActor {
	return &agentmemoryports.DeskMemoryActor{
		Actor: &permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
	}
}

func (h *overviewHarness) get(
	t *testing.T,
	sections workeroverviewservice.Sections,
) *workeroverviewservice.Overview {
	t.Helper()
	overview, err := h.svc.Get(t.Context(), &workeroverviewservice.GetRequest{
		TenantInfo:  h.tenant,
		WorkerID:    h.wrk.ID,
		Sections:    sections,
		MemoryActor: memoryActor(),
	})
	require.NoError(t, err)
	return overview
}

func TestGet_ComposesEverySectionWhenAllowed(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)

	overview := h.get(t, workeroverviewservice.All())

	assert.Equal(t, workercompliance.StandingGood, overview.Standing)
	assert.Equal(t, testNow, overview.AsOf)
	assert.NotNil(t, overview.Credentials)
	assert.NotNil(t, overview.Clearance)
	assert.NotNil(t, overview.QualificationFile)
	assert.NotNil(t, overview.DrugAlcohol)
	assert.NotNil(t, overview.Training)
	assert.NotNil(t, overview.Safety)
	assert.NotNil(t, overview.Employment)
	assert.NotNil(t, overview.Leave)
	assert.NotNil(t, overview.Schedule)
	assert.NotNil(t, overview.HOS)
	assert.NotNil(t, overview.Benefits)
	assert.NotNil(t, overview.Portal)
	require.NotNil(t, overview.DocumentCount)
	assert.Equal(t, 4, *overview.DocumentCount)
	require.NotNil(t, overview.AgentMemoryCount)
	assert.Equal(t, 2, *overview.AgentMemoryCount)
}

func TestGet_ReadsTheWorkerRowOnce(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	workerRepo := workerportsmock.NewMockWorkerRepository(t)
	workerRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req workerports.GetWorkerByIDRequest) bool {
			return req.ID == h.wrk.ID && req.IncludeProfile && req.IncludePortalUser
		})).
		Return(h.wrk, nil).
		Once()
	svc := workeroverviewservice.New(workeroverviewservice.Params{
		Logger:         zap.NewNop(),
		Organizations:  orgzonetest.Organizations{},
		Clock:          func() int64 { return testNow },
		Features:       plantest.AllFeatures(),
		WorkerRepo:     workerRepo,
		ReviewCounts:   h.reviewCounts,
		Credentials:    h.credentials,
		Training:       h.training,
		Safety:         h.safety,
		Checklists:     h.checklists,
		PTO:            h.pto,
		Reviews:        h.reviews,
		Leave:          h.leave,
		LeaveStanding:  h.leave,
		DrugAlcohol:    h.drugAlcohol,
		Qualification:  h.qualification,
		Employment:     h.events,
		Shifts:         h.shifts,
		HOS:            h.hos,
		Tractors:       h.tractors,
		TimeOff:        h.timeOff,
		Pay:            h.pay,
		Benefits:       h.benefits,
		Invitations:    h.invitations,
		ProfileChanges: h.profileChanges,
		Documents:      h.documents,
		Memories:       h.memories,
		Dispatch:       h.dispatch,
	})

	_, err := svc.Get(t.Context(), &workeroverviewservice.GetRequest{
		TenantInfo: h.tenant,
		WorkerID:   h.wrk.ID,
		Sections:   workeroverviewservice.All(),
	})

	require.NoError(t, err)
	require.NotNil(t, h.qualification.req)
	assert.Same(t, h.wrk, h.qualification.req.Worker)
}

func TestGet_SkipsSectionsTheCallerCannotRead(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)

	overview := h.get(t, workeroverviewservice.Sections{Credentials: true})

	assert.Equal(t, 1, h.credentials.calls)
	for name, calls := range map[string]int{
		"training":         h.training.calls,
		"safety":           h.safety.calls,
		"checklists":       h.checklists.calls,
		"pto":              h.pto.calls,
		"reviews":          h.reviews.calls,
		"leave":            h.leave.calls,
		"drug and alcohol": h.drugAlcohol.calls,
		"qualification":    h.qualification.calls,
		"employment":       h.events.calls,
		"schedule":         h.shifts.calls,
		"hos":              h.hos.calls,
		"pay":              h.pay.calls,
		"benefits":         h.benefits.calls,
		"portal":           h.invitations.calls,
		"profile changes":  h.profileChanges.calls,
		"documents":        h.documents.calls,
		"memories":         h.memories.calls,
	} {
		assert.Zero(t, calls, "%s was read without its permission", name)
	}
	assert.Nil(t, overview.Training)
	assert.Nil(t, overview.Safety)
	assert.Nil(t, overview.QualificationFile)
	assert.Nil(t, overview.DrugAlcohol)
	assert.Nil(t, overview.Employment)
	assert.Nil(t, overview.Leave)
	assert.Nil(t, overview.Schedule)
	assert.Nil(t, overview.HOS)
	assert.Nil(t, overview.Pay)
	assert.Nil(t, overview.Benefits)
	assert.Nil(t, overview.Portal)
	assert.Nil(t, overview.DocumentCount)
	assert.Nil(t, overview.AgentMemoryCount)
}

func TestGet_ClearanceIsWithheldWithoutCredentialRead(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)

	overview := h.get(t, workeroverviewservice.Sections{Training: true, Reviews: true})

	assert.Nil(t, overview.Clearance)
	assert.Nil(t, overview.Credentials)
	assert.Zero(t, h.credentials.calls)
	assert.Zero(t, h.dispatch.calls)
}

func TestGet_ClearanceLeavesOutTrainingWithoutTrainingRead(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	due := testNow + 10*timeutils.SecondsPerDay
	h.training.summary = &workertraining.WorkerTrainingSummary{
		Items: []*workertraining.TrainingSummaryItem{{
			Course: &workertraining.TrainingCourse{ID: pulid.MustNew("trc_"), Name: "Hazmat"},
			Record: &workertraining.WorkerTrainingRecord{
				ID:     pulid.MustNew("wtr_"),
				Status: workertraining.TrainingStatusAssigned,
				DueAt:  &due,
			},
			Health: worker.TrainingHealthDueSoon,
		}},
	}

	with := h.get(t, workeroverviewservice.Sections{Credentials: true, Training: true})
	without := h.get(t, workeroverviewservice.Sections{Credentials: true})

	require.NotNil(t, with.Clearance)
	require.Len(t, with.Clearance.Items, 1)
	assert.Equal(t, workercompliance.ClearanceSourceTraining, with.Clearance.Items[0].Source)
	require.NotNil(t, without.Clearance)
	assert.Empty(t, without.Clearance.Items)
}

func TestGet_QualificationReadsCredentialsWithoutExposingThem(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)

	overview := h.get(t, workeroverviewservice.Sections{Qualification: true})

	assert.Equal(t, 1, h.credentials.calls)
	assert.Equal(t, 1, h.drugAlcohol.calls)
	assert.NotNil(t, overview.QualificationFile)
	assert.Nil(t, overview.Credentials)
	assert.Nil(t, overview.DrugAlcohol)
	assert.Nil(t, overview.Clearance)
	assert.Empty(t, overview.Concerns)
}

func TestGet_StandingIgnoresSectionsTheCallerCannotRead(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.safety.card = &safetyscorecard.SafetyScorecard{Rating: worker.SafetyRatingAtRisk, Score: 20}

	withSafety := h.get(t, workeroverviewservice.All())
	assert.Equal(t, workercompliance.StandingAtRisk, withSafety.Standing)

	withoutSafety := h.get(t, workeroverviewservice.Sections{Credentials: true, Training: true})
	assert.Equal(t, workercompliance.StandingGood, withoutSafety.Standing)
	assert.Empty(t, withoutSafety.Concerns)
}

func TestGet_SectionFailureFailsTheQuery(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.safety.err = errors.New("safety repo is down")

	_, err := h.svc.Get(t.Context(), &workeroverviewservice.GetRequest{
		TenantInfo: h.tenant,
		WorkerID:   h.wrk.ID,
		Sections:   workeroverviewservice.All(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "safety repo is down")
}

func TestGet_CapsConcernsAndCountsEveryOne(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	items := make([]*workercredential.CredentialSummaryItem, 0, 40)
	for range 40 {
		items = append(items, &workercredential.CredentialSummaryItem{
			CredentialType: &workercredential.WorkerCredentialType{
				ID:   pulid.MustNew("wct_"),
				Name: "Certificate",
			},
			Health:   workercredential.CredentialHealthMissing,
			Required: true,
		})
	}
	h.credentials.summary = &workercredential.WorkerCredentialSummary{
		ComplianceStatus: worker.ComplianceStatusNonCompliant,
		Items:            items,
	}

	overview := h.get(t, workeroverviewservice.Sections{Credentials: true})

	assert.Equal(t, 40, overview.ConcernCount)
	assert.Len(t, overview.Concerns, workeroverviewservice.ConcernCap)
	assert.Equal(t, workercompliance.StandingBlocked, overview.Standing)
	require.NotNil(t, overview.Clearance)
	assert.False(t, overview.Clearance.Cleared)
}

func TestGet_PicksTheNewestOpenChecklist(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.checklists.checklists = []*workerchecklist.WorkerChecklist{
		{Name: "Older onboarding", Status: workerchecklist.ChecklistStatusOpen, StartedAt: 1_000},
		{Name: "Equipment handover", Status: workerchecklist.ChecklistStatusOpen, StartedAt: 9_000},
		{Name: "Closed one", Status: workerchecklist.ChecklistStatusCompleted, StartedAt: 99_000},
	}

	overview := h.get(t, workeroverviewservice.All())

	require.NotNil(t, overview.Checklist)
	assert.Equal(t, "Equipment handover", overview.Checklist.Name)
	require.NotNil(t, overview.Employment.OpenChecklistCount)
	assert.Equal(t, 2, *overview.Employment.OpenChecklistCount)
}

func TestGet_EmploymentReportsTheStoredSummary(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.events.summary = &workeremploymentports.EmploymentEventSummary{
		EventCount:  3,
		LastHiredAt: new(int64(4_000)),
	}

	overview := h.get(t, workeroverviewservice.Sections{Employment: true})

	require.NotNil(t, overview.Employment)
	assert.Equal(t, 3, overview.Employment.EventCount)
	require.NotNil(t, overview.Employment.HiredAt)
	assert.Equal(t, int64(4_000), *overview.Employment.HiredAt)
	assert.Nil(t, overview.Employment.OpenChecklistCount)
}

func TestGet_ReviewInFlightSuppressesTheDueDate(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	due := int64(1_000)
	h.reviews.reviews = []*workerreview.PerformanceReview{
		{Status: workerreview.ReviewStatusClosed, PeriodEnd: 500, NextReviewAt: &due},
		{Status: workerreview.ReviewStatusDraft, PeriodEnd: 900},
	}

	overview := h.get(t, workeroverviewservice.All())

	require.NotNil(t, overview.OpenReview)
	require.NotNil(t, overview.LastReview)
	assert.Nil(t, overview.NextReviewAt)
	assert.Equal(t, workercompliance.StandingGood, overview.Standing)
}

func TestGet_LastClosedReviewSuppliesTheDueDate(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	older := int64(1_000)
	newer := int64(2_000)
	h.reviews.reviews = []*workerreview.PerformanceReview{
		{Status: workerreview.ReviewStatusClosed, PeriodEnd: 500, NextReviewAt: &older},
		{Status: workerreview.ReviewStatusClosed, PeriodEnd: 900, NextReviewAt: &newer},
	}

	overview := h.get(t, workeroverviewservice.All())

	require.NotNil(t, overview.NextReviewAt)
	assert.Equal(t, newer, *overview.NextReviewAt)
	assert.Nil(t, overview.OpenReview)
}

func TestGet_KeepsDisciplineOffTheCardOfAReaderWhoCannotSeeIt(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)

	h.get(t, workeroverviewservice.Sections{Safety: true})

	assert.Equal(t, 1, h.safety.calls)
	assert.False(t, h.safety.discipline)
}

func TestGet_ReadsDisciplineForAReaderWhoMaySeeIt(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)

	h.get(t, workeroverviewservice.Sections{Safety: true, Discipline: true})

	assert.Equal(t, 1, h.safety.calls)
	assert.True(t, h.safety.discipline)
}

func TestGet_HOSIsNullWhenThePlanLeavesOutTelematics(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(
		t,
		withFeatures(plantest.Without(platformcatalog.FeatureSamsaraIntegration)),
	)

	overview := h.get(t, workeroverviewservice.Sections{HOS: true})

	assert.Nil(t, overview.HOS)
	assert.Zero(t, h.hos.calls)
}

func TestGet_HOSCarriesTheStoredLink(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	since := testNow - 3_600
	h.hos.state = &telematics.HOSLinkState{
		Link:            telematics.HOSLinkReporting,
		DutyStatusSince: &since,
	}

	overview := h.get(t, workeroverviewservice.Sections{HOS: true})

	require.NotNil(t, overview.HOS)
	assert.Equal(t, telematics.HOSLinkReporting, overview.HOS.Link)
	assert.Equal(t, &since, overview.HOS.DutyStatusSince)
}

func TestGet_BenefitsAreNullWhenThePlanLeavesThemOut(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(
		t,
		withFeatures(plantest.Without(platformcatalog.FeatureWorkforceBenefits)),
	)

	overview := h.get(t, workeroverviewservice.Sections{Benefits: true})

	assert.Nil(t, overview.Benefits)
	assert.Zero(t, h.benefits.calls)
}

func TestGet_BenefitsCountCoverAndPlansNotStarted(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	medical := &paybenefit.BenefitPlan{ID: pulid.MustNew("bplan_")}
	dental := &paybenefit.BenefitPlan{ID: pulid.MustNew("bplan_")}
	vision := &paybenefit.BenefitPlan{ID: pulid.MustNew("bplan_")}
	h.benefits.plans = []*paybenefit.BenefitPlan{medical, dental, vision}
	h.benefits.enrollments = []*paybenefit.WorkerBenefitEnrollment{
		{BenefitPlanID: medical.ID, Status: paybenefit.EnrollmentActive, EmployeeCostMinor: 4_500},
		{BenefitPlanID: dental.ID, Status: paybenefit.EnrollmentWaived},
	}

	overview := h.get(t, workeroverviewservice.Sections{Benefits: true})

	require.NotNil(t, overview.Benefits)
	assert.Equal(t, 1, overview.Benefits.CoveredCount)
	assert.Equal(t, 1, overview.Benefits.NotStartedCount)
	assert.Equal(t, int64(4_500), overview.Benefits.EmployeeCostPerPeriodMinor)
}

func TestGet_PayIsNullWithoutAnAssignmentInForce(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)

	overview := h.get(t, workeroverviewservice.Sections{Pay: true})

	assert.Equal(t, 1, h.pay.calls)
	assert.Nil(t, overview.Pay)
}

func TestGet_PayQuotesTheOverriddenComponentFirst(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	mileage := &driverpay.PayProfileComponent{
		ID:       pulid.MustNew("dppc_"),
		Kind:     driverpay.ComponentKindLinehaul,
		Rate:     decimal.RequireFromString("0.55"),
		IsActive: true,
	}
	stop := &driverpay.PayProfileComponent{
		ID:       pulid.MustNew("dppc_"),
		Kind:     driverpay.ComponentKindStopPay,
		Rate:     decimal.RequireFromString("25"),
		IsActive: true,
	}
	h.pay.assignment = &payassignment.WorkerPayAssignment{
		EffectiveFrom: 1_000,
		RateOverrides: []payassignment.RateOverride{
			{ComponentID: stop.ID, Rate: decimal.RequireFromString("30")},
		},
		PayProfile: &driverpay.PayProfile{
			Name:       "Linehaul",
			Components: []*driverpay.PayProfileComponent{mileage, stop},
		},
	}

	overview := h.get(t, workeroverviewservice.Sections{Pay: true})

	require.NotNil(t, overview.Pay)
	assert.Equal(t, "Linehaul", overview.Pay.ProfileName)
	require.NotNil(t, overview.Pay.Headline)
	assert.True(t, overview.Pay.Headline.Overridden)
	assert.Equal(t, stop.Kind, overview.Pay.Headline.Kind)
	assert.True(t, decimal.RequireFromString("30").Equal(overview.Pay.Headline.Rate))
}

func TestGet_PortalLinkedCarriesTheLastSignIn(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	signedIn := testNow - 600
	h.wrk.UserID = pulid.MustNew("usr_")
	h.wrk.PortalUser = &tenant.User{LastLoginAt: &signedIn}

	overview := h.get(t, workeroverviewservice.Sections{Portal: true})

	require.NotNil(t, overview.Portal)
	assert.Equal(t, workeroverviewservice.PortalAccessLinked, overview.Portal.Access)
	assert.Equal(t, &signedIn, overview.Portal.LastSignInAt)
	assert.Zero(t, h.invitations.calls)
	assert.Nil(t, overview.Portal.PendingProfileChangeCount)
}

func TestGet_PortalInvitedWhileAnInvitationIsAcceptable(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.invitations.invitation = &portalinvitation.PortalInvitation{
		Status:    portalinvitation.PortalInvitationStatusPending,
		ExpiresAt: testNow + 3_600,
	}
	h.profileChanges.requests = []*profilechange.WorkerProfileChangeRequest{{}, {}}

	overview := h.get(t, workeroverviewservice.Sections{Portal: true, ProfileChanges: true})

	require.NotNil(t, overview.Portal)
	assert.Equal(t, workeroverviewservice.PortalAccessInvited, overview.Portal.Access)
	require.NotNil(t, overview.Portal.PendingProfileChangeCount)
	assert.Equal(t, 2, *overview.Portal.PendingProfileChangeCount)
}

func TestGet_PortalNotSetUpWithoutUserOrInvitation(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)

	overview := h.get(t, workeroverviewservice.Sections{Portal: true})

	require.NotNil(t, overview.Portal)
	assert.Equal(t, workeroverviewservice.PortalAccessNotSetUp, overview.Portal.Access)
}

func TestGet_DocumentsAreCountedOnTheWorker(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)

	h.get(t, workeroverviewservice.Sections{Documents: true})

	require.NotNil(t, h.documents.req)
	assert.Equal(t, "worker", h.documents.req.ResourceType)
	assert.Equal(t, h.wrk.ID.String(), h.documents.req.ResourceID)
}

func TestGet_MemoriesAreCountedForTheWorkerAsTheReaderSeesThem(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)

	overview := h.get(t, workeroverviewservice.Sections{AgentMemory: true})

	require.NotNil(t, overview.AgentMemoryCount)
	require.NotNil(t, h.memories.req)
	assert.Equal(t, agentmemory.MemorySubjectWorker, h.memories.req.SubjectType)
	assert.Equal(t, h.wrk.ID, h.memories.req.SubjectID)
}

func TestGet_MemoriesAreNullWithoutAPerson(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)

	overview, err := h.svc.Get(t.Context(), &workeroverviewservice.GetRequest{
		TenantInfo: h.tenant,
		WorkerID:   h.wrk.ID,
		Sections:   workeroverviewservice.Sections{AgentMemory: true},
	})

	require.NoError(t, err)
	assert.Nil(t, overview.AgentMemoryCount)
	assert.Zero(t, h.memories.calls)
}

func TestGet_DrugAlcoholCountsDaysToTheNextQuery(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	due := testNow + 17*timeutils.SecondsPerDay
	h.drugAlcohol.standing = workerdot.DrugAlcoholStanding{
		Status:                    worker.DrugAlcoholClear,
		NextClearinghouseQueryDue: &due,
	}

	overview := h.get(t, workeroverviewservice.Sections{DrugAlcohol: true})

	require.NotNil(t, overview.DrugAlcohol)
	require.NotNil(t, overview.DrugAlcohol.NextQueryDaysUntil)
	assert.Equal(t, int64(17), *overview.DrugAlcohol.NextQueryDaysUntil)
	require.Len(t, overview.Concerns, 1)
	assert.Equal(t, workercompliance.ConcernClearinghouseQueryDue, overview.Concerns[0].Code)
}

func TestGet_ScheduleNamesTheShiftInForce(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.shifts.assignments = []*workerschedule.WorkerShiftAssignment{
		{ShiftTemplate: &workerschedule.ShiftTemplate{Name: "Nights"}},
	}

	overview := h.get(t, workeroverviewservice.Sections{Schedule: true})

	require.NotNil(t, overview.Schedule)
	require.NotNil(t, overview.Schedule.CurrentShiftName)
	assert.Equal(t, "Nights", *overview.Schedule.CurrentShiftName)
}

func TestRosterAttention_GathersEveryArea(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.workerRepo.EXPECT().
		CountRosterAttention(mock.Anything, mock.Anything).
		Return(&workerports.RosterAttention{
			ActiveWorkers:   40,
			NonCompliant:    3,
			TrainingOverdue: 7,
			AtRisk:          1,
			ExpiringSoon:    5,
		}, nil).
		Once()
	h.reviewCounts.count = 4
	h.pto.summary = &ptoledgerports.PTOBalanceSummary{
		TotalBalanceDays: decimal.NewFromInt(312),
	}

	attention, err := h.svc.RosterAttention(t.Context(), h.tenant,
		workeroverviewservice.AttentionAccess{Reviews: true, PTO: true, Training: true})
	require.NoError(t, err)

	assert.Equal(t, 40, attention.ActiveWorkers)
	assert.Equal(t, 3, attention.NonCompliant)
	require.NotNil(t, attention.TrainingOverdue)
	assert.Equal(t, 7, *attention.TrainingOverdue)
	assert.Equal(t, 1, attention.AtRisk)
	assert.Equal(t, 5, attention.ExpiringSoon)
	require.NotNil(t, attention.ReviewsAwaitingSignOff)
	assert.Equal(t, 4, *attention.ReviewsAwaitingSignOff)
	require.NotNil(t, attention.PTOLiabilityDays)
	assert.True(t, decimal.NewFromInt(312).Equal(*attention.PTOLiabilityDays))
}

func TestRosterAttention_TheTwoWorkerFiguresAreNullWithoutTheirReadOrTheirFeature(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.workerRepo.EXPECT().
		CountRosterAttention(mock.Anything, mock.Anything).
		Return(&workerports.RosterAttention{ActiveWorkers: 40, TrainingOverdue: 7}, nil).
		Once()
	h.pto.summary = &ptoledgerports.PTOBalanceSummary{
		TotalBalanceDays: decimal.NewFromInt(312),
	}

	attention, err := h.svc.RosterAttention(t.Context(), h.tenant,
		workeroverviewservice.AttentionAccess{Reviews: true})
	require.NoError(t, err)

	assert.Equal(t, 40, attention.ActiveWorkers)
	assert.Nil(t, attention.TrainingOverdue)
	assert.Nil(t, attention.PTOLiabilityDays)
	assert.Zero(t, h.pto.summaryCalls, "the PTO ledger is not read for a reader without worker_pto")
}

func TestRosterAttention_CountsOnlySubmittedReviews(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.workerRepo.EXPECT().
		CountRosterAttention(mock.Anything, mock.Anything).
		Return(&workerports.RosterAttention{}, nil).
		Once()

	_, err := h.svc.RosterAttention(t.Context(), h.tenant,
		workeroverviewservice.AttentionAccess{Reviews: true})
	require.NoError(t, err)
	assert.Equal(
		t,
		[]workerreview.ReviewStatus{workerreview.ReviewStatusSubmitted},
		h.reviewCounts.statuses,
	)
}

func TestRosterAttention_CountsNothingTheReaderMayNotRead(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.workerRepo.EXPECT().
		CountRosterAttention(mock.Anything, mock.Anything).
		Return(&workerports.RosterAttention{ActiveWorkers: 12}, nil).
		Once()
	h.reviewCounts.count = 4

	attention, err := h.svc.RosterAttention(t.Context(), h.tenant,
		workeroverviewservice.AttentionAccess{})
	require.NoError(t, err)

	assert.Equal(t, 12, attention.ActiveWorkers)
	assert.Nil(t, attention.ReviewsAwaitingSignOff)
	assert.Nil(t, attention.LeaveCertificationsOutstanding)
	assert.Nil(t, h.reviewCounts.statuses, "reviews are not counted at all")
}

func TestRosterAttention_CountsLeaveAcrossTheReadersTeam(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.workerRepo.EXPECT().
		CountRosterAttention(mock.Anything, mock.Anything).
		Return(&workerports.RosterAttention{}, nil).
		Once()
	h.leave.count = 2
	manager := pulid.MustNew("usr_")

	attention, err := h.svc.RosterAttention(t.Context(), pagination.TenantInfo{},
		workeroverviewservice.AttentionAccess{Leave: true, LeaveManagerIDs: []pulid.ID{manager}})
	require.NoError(t, err)

	require.NotNil(t, attention.LeaveCertificationsOutstanding)
	assert.Equal(t, 2, *attention.LeaveCertificationsOutstanding)
	require.NotNil(t, h.leave.req)
	assert.Equal(t, []pulid.ID{manager}, h.leave.req.ManagerIDs)
	assert.True(t, h.leave.req.CertificationOutstandingOnly)
	assert.Nil(t, attention.ReviewsAwaitingSignOff)
}

func TestGet_HOSCarriesTheClocksAndTheTractor(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	drive, driveLimit := int64(3*3_600_000), int64(11*3_600_000)
	breakLeft, breakLimit := int64(1_800_000), int64(8*3_600_000)
	unit := pulid.MustNew("tr_")
	h.hos.state = &telematics.HOSLinkState{
		Link:             telematics.HOSLinkReporting,
		DriveRemainingMs: &drive,
		DriveLimitMs:     &driveLimit,
		BreakRemainingMs: &breakLeft,
		BreakLimitMs:     &breakLimit,
		CurrentTractorID: unit,
	}
	h.tractors.unit = &tractor.Tractor{ID: unit, Code: "T-104"}

	overview := h.get(t, workeroverviewservice.Sections{HOS: true, Tractor: true})

	require.NotNil(t, overview.HOS)
	assert.Equal(t, &driveLimit, overview.HOS.DriveLimitMs)
	assert.Equal(t, &breakLeft, overview.HOS.BreakRemainingMs)
	assert.Equal(t, &breakLimit, overview.HOS.BreakLimitMs)
	require.NotNil(t, overview.HOS.TractorCode)
	assert.Equal(t, "T-104", *overview.HOS.TractorCode)
	assert.Equal(t, unit, h.tractors.req.ID)
	assert.Equal(t, h.tenant, h.tractors.req.TenantInfo)
}

func TestGet_TractorIsWithheldWithoutTractorRead(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.hos.state = &telematics.HOSLinkState{
		Link:             telematics.HOSLinkReporting,
		CurrentTractorID: pulid.MustNew("tr_"),
	}
	h.tractors.unit = &tractor.Tractor{Code: "T-104"}

	overview := h.get(t, workeroverviewservice.Sections{HOS: true})

	require.NotNil(t, overview.HOS)
	assert.Nil(t, overview.HOS.TractorCode)
	assert.Zero(t, h.tractors.calls)
}

func TestGet_TractorCodeIsNullWhenTheUnitIsGone(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	h.hos.state = &telematics.HOSLinkState{
		Link:             telematics.HOSLinkReporting,
		CurrentTractorID: pulid.MustNew("tr_"),
	}

	overview := h.get(t, workeroverviewservice.Sections{HOS: true, Tractor: true})

	require.NotNil(t, overview.HOS)
	assert.Nil(t, overview.HOS.TractorCode)
	assert.Equal(t, 1, h.tractors.calls)
}

func TestGet_NextTimeOffIsTheEarliestRequestedOrApprovedNotYetEnded(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)
	next := &worker.WorkerPTO{ID: pulid.MustNew("pto_"), StartDate: testNow + 86_400}
	h.timeOff.items = []*worker.WorkerPTO{next}

	overview := h.get(t, workeroverviewservice.Sections{PTO: true})

	assert.Same(t, next, overview.NextTimeOff)
	require.NotNil(t, h.timeOff.req)
	req := h.timeOff.req
	assert.Equal(t, h.wrk.ID, req.WorkerID)
	assert.Equal(
		t,
		[]worker.PTOStatus{worker.PTOStatusRequested, worker.PTOStatusApproved},
		req.Statuses,
	)
	assert.True(t, req.Overlapping, "a time off already under way still counts")
	assert.Equal(t, testNow, req.StartDateFrom)
	assert.Equal(t, 1, req.Cursor.Limit)
	assert.False(t, req.Cursor.IncludeTotalCount)
	require.Len(t, req.Filter.Sort, 1)
	assert.Equal(t, "startDate", req.Filter.Sort[0].Field)
}

func TestGet_NextTimeOffIsNullWithNoneOrWithoutPTORead(t *testing.T) {
	t.Parallel()
	h := newOverviewHarness(t)

	none := h.get(t, workeroverviewservice.Sections{PTO: true})
	withheld := h.get(t, workeroverviewservice.Sections{Credentials: true})

	assert.Nil(t, none.NextTimeOff)
	assert.Nil(t, withheld.NextTimeOff)
	assert.Equal(t, 1, h.timeOff.calls)
}

func prohibitedHarness(t *testing.T) *overviewHarness {
	t.Helper()
	h := newOverviewHarness(t)
	due := testNow - 5*timeutils.SecondsPerDay
	h.drugAlcohol.standing = workerdot.DrugAlcoholStanding{
		Status:                    worker.DrugAlcoholProhibited,
		NextClearinghouseQueryDue: &due,
	}
	h.dispatch.verdicts = &workercompliance.DispatchVerdicts{
		RefusedToday: []workercompliance.DispatchRequirement{
			workercompliance.DispatchRequirementDrugAlcohol,
		},
	}
	return h
}

func TestGet_AProhibitionStaysOutOfSightWithoutDrugAndAlcoholRead(t *testing.T) {
	t.Parallel()
	h := prohibitedHarness(t)

	overview := h.get(t, workeroverviewservice.Sections{Credentials: true, Qualification: true})

	require.NotNil(t, overview.Clearance)
	assert.False(t, overview.Clearance.Cleared)
	assert.Equal(
		t,
		[]workercompliance.ClearanceBlock{workercompliance.ClearanceBlockDispatchHold},
		overview.Clearance.Blocks,
	)
	for _, item := range overview.Clearance.Items {
		assert.NotEqual(t, workercompliance.ClearanceSourceClearinghouseQuery, item.Source)
	}
	for _, concern := range overview.Concerns {
		assert.NotEqual(t, workercompliance.ConcernDrugAlcoholProhibited, concern.Code)
		assert.NotEqual(t, workercompliance.ConcernClearinghouseQueryOverdue, concern.Code)
	}
	assert.Nil(t, overview.DrugAlcohol)
}

func TestGet_AProhibitionIsNamedToADrugAndAlcoholReader(t *testing.T) {
	t.Parallel()
	h := prohibitedHarness(t)

	overview := h.get(t, workeroverviewservice.Sections{Credentials: true, DrugAlcohol: true})

	require.NotNil(t, overview.Clearance)
	assert.Equal(
		t,
		[]workercompliance.ClearanceBlock{workercompliance.ClearanceBlockDispatchCompliance},
		overview.Clearance.Blocks,
	)
	codes := make([]workercompliance.ConcernCode, 0, len(overview.Concerns))
	for _, concern := range overview.Concerns {
		codes = append(codes, concern.Code)
	}
	assert.Contains(t, codes, workercompliance.ConcernDrugAlcoholProhibited)
}

func TestGet_CountsDaysInTheOrganizationsCalendar(t *testing.T) {
	t.Parallel()
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	evening := time.Date(2026, time.November, 23, 21, 0, 0, 0, chicago).Unix()
	h := newOverviewHarness(t, func(o *harnessOptions) {
		o.zone = "America/Chicago"
		o.now = evening
	})

	overview := h.get(t, workeroverviewservice.Sections{Credentials: true})

	assert.Equal(t, evening, overview.AsOf, "asOf stays the instant the page was read")
	assert.Equal(
		t,
		time.Date(2026, time.November, 23, 0, 0, 0, 0, time.UTC).Unix(),
		h.credentials.asOf,
		"at 9pm in Chicago it is still the 23rd, though UTC is already on the 24th",
	)
}
