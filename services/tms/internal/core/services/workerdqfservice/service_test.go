package workerdqfservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workerdqfservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func ptrInt64(v int64) *int64 { return &v }

// fakeDQFRepo holds rows so the follow-up and status behaviour can be
// exercised for real rather than asserted against canned returns.
type fakeDQFRepo struct {
	verifications map[pulid.ID]*worker.WorkerEmploymentVerification
	candidates    []repositories.DQFRetentionCandidate
	lastRetention *repositories.ListDQFRetentionCandidatesRequest
}

func newFakeRepo() *fakeDQFRepo {
	return &fakeDQFRepo{verifications: map[pulid.ID]*worker.WorkerEmploymentVerification{}}
}

func (f *fakeDQFRepo) ListVerifications(
	_ context.Context,
	req *repositories.ListEmploymentVerificationsRequest,
) ([]*worker.WorkerEmploymentVerification, error) {
	out := make([]*worker.WorkerEmploymentVerification, 0, len(f.verifications))
	for _, entity := range f.verifications {
		if !req.WorkerID.IsNil() && entity.WorkerID != req.WorkerID {
			continue
		}
		if req.OutstandingOnly && !entity.IsOutstanding() {
			continue
		}
		out = append(out, entity)
	}
	return out, nil
}

func (f *fakeDQFRepo) ListVerificationsByWorkerIDs(
	_ context.Context,
	req *repositories.ListEmploymentVerificationsByWorkerIDsRequest,
) (map[pulid.ID][]*worker.WorkerEmploymentVerification, error) {
	wanted := make(map[pulid.ID]struct{}, len(req.WorkerIDs))
	for _, id := range req.WorkerIDs {
		wanted[id] = struct{}{}
	}
	matched := make([]*worker.WorkerEmploymentVerification, 0, len(f.verifications))
	for _, entity := range f.verifications {
		if _, ok := wanted[entity.WorkerID]; ok {
			matched = append(matched, entity)
		}
	}
	return sliceutils.GroupBy(matched, func(v *worker.WorkerEmploymentVerification) pulid.ID {
		return v.WorkerID
	}), nil
}

func (f *fakeDQFRepo) GetVerificationByID(
	_ context.Context,
	req *repositories.GetEmploymentVerificationByIDRequest,
) (*worker.WorkerEmploymentVerification, error) {
	entity, ok := f.verifications[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("verification not found")
	}
	return entity, nil
}

func (f *fakeDQFRepo) CreateVerification(
	_ context.Context,
	entity *worker.WorkerEmploymentVerification,
) (*worker.WorkerEmploymentVerification, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("wemv_")
	}
	f.verifications[entity.ID] = entity
	return entity, nil
}

func (f *fakeDQFRepo) UpdateVerification(
	_ context.Context,
	entity *worker.WorkerEmploymentVerification,
) (*worker.WorkerEmploymentVerification, error) {
	f.verifications[entity.ID] = entity
	return entity, nil
}

func (f *fakeDQFRepo) DeleteVerification(
	_ context.Context,
	req *repositories.GetEmploymentVerificationByIDRequest,
) error {
	delete(f.verifications, req.ID)
	return nil
}

func (f *fakeDQFRepo) ListRetentionCandidates(
	_ context.Context,
	req *repositories.ListDQFRetentionCandidatesRequest,
) ([]repositories.DQFRetentionCandidate, error) {
	f.lastRetention = req
	return f.candidates, nil
}

type fakeCredentials struct {
	summary *worker.WorkerCredentialSummary
}

func (f *fakeCredentials) Summary(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (*worker.WorkerCredentialSummary, error) {
	return f.summary, nil
}

type fakeDocuments struct {
	summary *documentpacketrule.PacketSummary
	calls   int
}

func (f *fakeDocuments) GetPacketSummary(
	_ context.Context,
	resourceType, _ string,
	_ pagination.TenantInfo,
) (*documentpacketrule.PacketSummary, error) {
	f.calls++
	if resourceType != "Worker" {
		return nil, errortypes.NewNotFoundError("unexpected resource type " + resourceType)
	}
	return f.summary, nil
}

type fakeStanding struct {
	standing worker.DrugAlcoholStanding
}

func (f *fakeStanding) Standing(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (worker.DrugAlcoholStanding, error) {
	return f.standing, nil
}

type fakeRetention struct {
	row *tenant.DataRetention
	err error
}

func (f *fakeRetention) List(
	_ context.Context,
) (*pagination.ListResult[*tenant.DataRetention], error) {
	return nil, nil
}

func (f *fakeRetention) Get(
	_ context.Context,
	_ repositories.GetDataRetentionRequest,
) (*tenant.DataRetention, error) {
	return f.row, f.err
}

func (f *fakeRetention) Update(
	_ context.Context,
	entity *tenant.DataRetention,
) (*tenant.DataRetention, error) {
	return entity, nil
}

func (f *fakeRetention) Upsert(
	_ context.Context,
	entity *tenant.DataRetention,
) (*tenant.DataRetention, error) {
	return entity, nil
}

type harness struct {
	svc       *workerdqfservice.Service
	repo      *fakeDQFRepo
	documents *fakeDocuments
	retention *fakeRetention
	tenant    pagination.TenantInfo
	workerID  pulid.ID
	userID    pulid.ID
}

func newHarness(t *testing.T, profile *worker.WorkerProfile) *harness {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	workerID := pulid.MustNew("wrk_")

	workerRepo := mocks.NewMockWorkerRepository(t)
	workerRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&worker.Worker{
			ID:             workerID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Profile:        profile,
		}, nil).
		Maybe()

	repo := newFakeRepo()
	documents := &fakeDocuments{}
	retention := &fakeRetention{}

	return &harness{
		svc: workerdqfservice.NewWithDeps(workerdqfservice.Deps{
			Repo:          repo,
			WorkerRepo:    workerRepo,
			RetentionRepo: retention,
			Credentials: &fakeCredentials{
				summary: &worker.WorkerCredentialSummary{},
			},
			Documents: documents,
			DrugAlcohol: &fakeStanding{
				standing: worker.DrugAlcoholStanding{
					Status:                worker.DrugAlcoholClear,
					HasPreEmploymentTest:  true,
					HasPreEmploymentQuery: true,
				},
			},
		}),
		repo:      repo,
		documents: documents,
		retention: retention,
		tenant:    pagination.TenantInfo{OrgID: orgID, BuID: buID},
		workerID:  workerID,
		userID:    pulid.MustNew("usr_"),
	}
}

func employedProfile() *worker.WorkerProfile {
	return &worker.WorkerProfile{HireDate: 1_700_000_000}
}

func (h *harness) settled(t *testing.T) *worker.WorkerEmploymentVerification {
	t.Helper()
	created, err := h.svc.RecordVerification(t.Context(), &worker.WorkerEmploymentVerification{
		OrganizationID:     h.tenant.OrgID,
		BusinessUnitID:     h.tenant.BuID,
		WorkerID:           h.workerID,
		EmployerName:       "Prior Carrier",
		WasDOTRegulated:    true,
		Status:             worker.VerificationReceived,
		RequestedAt:        ptrInt64(1_700_000_000),
		ResponseReceivedAt: ptrInt64(1_700_100_000),
	}, h.userID)
	require.NoError(t, err)
	created.DrugAlcoholResponseReceivedAt = ptrInt64(1_700_100_000)
	return created
}

// The file is composed from the areas that own each piece rather than stored,
// so every one of them has to be asked.
func TestFile_ComposesEveryArea(t *testing.T) {
	t.Parallel()
	h := newHarness(t, employedProfile())
	h.settled(t)

	file, err := h.svc.File(t.Context(), h.tenant, h.workerID)
	require.NoError(t, err)

	assert.Equal(t, h.workerID, file.WorkerID)
	assert.Equal(t, int64(1_700_000_000), file.HireDate)
	assert.Equal(t, 1, h.documents.calls, "the worker document packet is read once")
	assert.True(t, file.Complete)
}

// A tenant with no retention row configured falls back to the regulation's
// three years. Falling back to zero would report every terminated file as
// purgeable the day the driver left.
func TestFile_RetentionFallsBackToTheRegulation(t *testing.T) {
	t.Parallel()
	terminated := int64(1_700_000_000)
	h := newHarness(t, &worker.WorkerProfile{
		HireDate:        1_600_000_000,
		TerminationDate: &terminated,
	})
	h.settled(t)

	file, err := h.svc.File(t.Context(), h.tenant, h.workerID)
	require.NoError(t, err)

	require.NotNil(t, file.RetentionExpiresAt)
	assert.Equal(t, terminated+worker.DefaultDQFRetentionDays*86400, *file.RetentionExpiresAt)
}

func TestFile_RetentionUsesTheOrganisationSetting(t *testing.T) {
	t.Parallel()
	terminated := int64(1_700_000_000)
	h := newHarness(t, &worker.WorkerProfile{
		HireDate:        1_600_000_000,
		TerminationDate: &terminated,
	})
	h.retention.row = &tenant.DataRetention{DriverQualificationRetentionPeriod: 1800}
	h.settled(t)

	file, err := h.svc.File(t.Context(), h.tenant, h.workerID)
	require.NoError(t, err)

	require.NotNil(t, file.RetentionExpiresAt)
	assert.Equal(t, terminated+1800*86400, *file.RetentionExpiresAt)
}

func TestRecordVerification_Defaults(t *testing.T) {
	t.Parallel()
	h := newHarness(t, employedProfile())

	created, err := h.svc.RecordVerification(t.Context(), &worker.WorkerEmploymentVerification{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		EmployerName:   "  Prior Carrier  ",
	}, h.userID)
	require.NoError(t, err)

	assert.Equal(t, "Prior Carrier", created.EmployerName)
	assert.Equal(t, worker.VerificationPending, created.Status)
	assert.Equal(t, worker.VerificationByEmail, created.Method)
	assert.Equal(t, h.userID, created.RequestedByID)
}

// A non-regulated employer has no testing record to ask about, so anything
// recorded against the drug and alcohol half there is meaningless.
func TestRecordVerification_ClearsDrugAlcoholForNonRegulatedEmployment(t *testing.T) {
	t.Parallel()
	h := newHarness(t, employedProfile())

	created, err := h.svc.RecordVerification(t.Context(), &worker.WorkerEmploymentVerification{
		OrganizationID:                h.tenant.OrgID,
		BusinessUnitID:                h.tenant.BuID,
		WorkerID:                      h.workerID,
		EmployerName:                  "Local Warehouse",
		WasDOTRegulated:               false,
		HadDrugAlcoholViolations:      true,
		DrugAlcoholResponseReceivedAt: ptrInt64(1_700_100_000),
	}, h.userID)
	require.NoError(t, err)

	assert.Nil(t, created.DrugAlcoholResponseReceivedAt)
	assert.False(t, created.HadDrugAlcoholViolations)
	assert.False(t, created.NeedsDrugAlcoholAnswer())
}

func TestMarkRequested_StampsTheDate(t *testing.T) {
	t.Parallel()
	h := newHarness(t, employedProfile())

	created, err := h.svc.RecordVerification(t.Context(), &worker.WorkerEmploymentVerification{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		EmployerName:   "Prior Carrier",
	}, h.userID)
	require.NoError(t, err)

	updated, err := h.svc.MarkRequested(t.Context(), h.tenant, created.ID, h.userID)
	require.NoError(t, err)

	assert.Equal(t, worker.VerificationRequested, updated.Status)
	require.NotNil(t, updated.RequestedAt)
	assert.Positive(t, *updated.RequestedAt)
}

// The follow-up count is the evidence of good-faith effort 49 CFR 391.23 asks
// for, so each chase has to be counted and dated.
func TestRecordFollowUp_CountsEachChase(t *testing.T) {
	t.Parallel()
	h := newHarness(t, employedProfile())

	created, err := h.svc.RecordVerification(t.Context(), &worker.WorkerEmploymentVerification{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		EmployerName:   "Prior Carrier",
		Status:         worker.VerificationRequested,
		RequestedAt:    ptrInt64(1_700_000_000),
	}, h.userID)
	require.NoError(t, err)

	updated, err := h.svc.RecordFollowUp(t.Context(), h.tenant, created.ID, h.userID)
	require.NoError(t, err)
	assert.Equal(t, int32(1), updated.FollowUpCount)
	require.NotNil(t, updated.LastFollowUpAt)

	updated, err = h.svc.RecordFollowUp(t.Context(), h.tenant, created.ID, h.userID)
	require.NoError(t, err)
	assert.Equal(t, int32(2), updated.FollowUpCount)
}

// Chasing an employer who has already answered, or one nobody has written to
// yet, is not a follow-up — recording it would inflate the evidence.
func TestRecordFollowUp_OnlyWhileAwaitingAResponse(t *testing.T) {
	t.Parallel()
	h := newHarness(t, employedProfile())
	settled := h.settled(t)

	_, err := h.svc.RecordFollowUp(t.Context(), h.tenant, settled.ID, h.userID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "awaiting a response")
}

func TestRetentionCandidates_UsesTheConfiguredWindow(t *testing.T) {
	t.Parallel()
	h := newHarness(t, employedProfile())
	h.retention.row = &tenant.DataRetention{DriverQualificationRetentionPeriod: 1800}

	_, err := h.svc.RetentionCandidates(t.Context(), h.tenant, 0)
	require.NoError(t, err)

	require.NotNil(t, h.repo.lastRetention)
	assert.Equal(t, int64(1800*86400), h.repo.lastRetention.RetentionSeconds)
	assert.Positive(t, h.repo.lastRetention.AsOf)
}
