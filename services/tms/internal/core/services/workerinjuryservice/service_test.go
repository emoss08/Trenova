package workerinjuryservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workerinjuryservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// fakeRepo holds rows so case numbering and the certification guards can be
// exercised for real rather than asserted against canned returns.
type fakeRepo struct {
	injuries  map[pulid.ID]*worker.WorkerInjury
	summaries map[int16]*worker.OSHAAnnualSummary
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		injuries:  map[pulid.ID]*worker.WorkerInjury{},
		summaries: map[int16]*worker.OSHAAnnualSummary{},
	}
}

func (f *fakeRepo) ListInjuries(
	_ context.Context,
	req *repositories.ListWorkerInjuriesRequest,
) ([]*worker.WorkerInjury, error) {
	out := make([]*worker.WorkerInjury, 0, len(f.injuries))
	for _, injury := range f.injuries {
		if !req.WorkerID.IsNil() && injury.WorkerID != req.WorkerID {
			continue
		}
		if req.CaseYear > 0 && injury.CaseYear != req.CaseYear {
			continue
		}
		if req.RecordableOnly && !injury.IsRecordable() {
			continue
		}
		if req.OpenOnly && !injury.IsOpen() {
			continue
		}
		out = append(out, injury)
	}
	return out, nil
}

func (f *fakeRepo) GetInjuryByID(
	_ context.Context,
	req *repositories.GetWorkerInjuryByIDRequest,
) (*worker.WorkerInjury, error) {
	injury, ok := f.injuries[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("case not found")
	}
	return injury, nil
}

func (f *fakeRepo) CreateInjury(
	_ context.Context,
	entity *worker.WorkerInjury,
) (*worker.WorkerInjury, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("winj_")
	}
	f.injuries[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateInjury(
	_ context.Context,
	entity *worker.WorkerInjury,
) (*worker.WorkerInjury, error) {
	f.injuries[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) DeleteInjury(
	_ context.Context,
	req *repositories.GetWorkerInjuryByIDRequest,
) error {
	delete(f.injuries, req.ID)
	return nil
}

func (f *fakeRepo) NextCaseNumber(
	_ context.Context,
	req *repositories.NextCaseNumberRequest,
) (int32, error) {
	var highest int32
	for _, injury := range f.injuries {
		if injury.CaseYear == req.CaseYear && injury.CaseNumber > highest {
			highest = injury.CaseNumber
		}
	}
	return highest + 1, nil
}

func (f *fakeRepo) GetSummary(
	_ context.Context,
	req *repositories.GetOSHASummaryRequest,
) (*worker.OSHAAnnualSummary, error) {
	return f.summaries[req.Year], nil
}

func (f *fakeRepo) ListSummaries(
	_ context.Context,
	_ *repositories.ListOSHASummariesRequest,
) ([]*worker.OSHAAnnualSummary, error) {
	out := make([]*worker.OSHAAnnualSummary, 0, len(f.summaries))
	for _, summary := range f.summaries {
		out = append(out, summary)
	}
	return out, nil
}

func (f *fakeRepo) CreateSummary(
	_ context.Context,
	entity *worker.OSHAAnnualSummary,
) (*worker.OSHAAnnualSummary, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("osum_")
	}
	f.summaries[entity.Year] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateSummary(
	_ context.Context,
	entity *worker.OSHAAnnualSummary,
) (*worker.OSHAAnnualSummary, error) {
	f.summaries[entity.Year] = entity
	return entity, nil
}

type harness struct {
	svc      *workerinjuryservice.Service
	repo     *fakeRepo
	tenant   pagination.TenantInfo
	workerID pulid.ID
	userID   pulid.ID
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	workerID := pulid.MustNew("wrk_")

	workerRepo := mocks.NewMockWorkerRepository(t)
	workerRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&worker.Worker{ID: workerID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Maybe()

	repo := newFakeRepo()

	return &harness{
		svc: workerinjuryservice.NewWithDeps(workerinjuryservice.Deps{
			Repo:       repo,
			WorkerRepo: workerRepo,
		}),
		repo:     repo,
		tenant:   pagination.TenantInfo{OrgID: orgID, BuID: buID},
		workerID: workerID,
		userID:   pulid.MustNew("usr_"),
	}
}

// 2026-01-15
const occurred2026 = int64(1_768_435_200)

func (h *harness) newCase() *worker.WorkerInjury {
	return &worker.WorkerInjury{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		OccurredAt:     occurred2026,
		Description:    "Slipped on the dock",
	}
}

// Case numbers restart each calendar year and run in order; that is how the
// 300 log reads.
func TestRecordInjury_NumbersTheCase(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	first, err := h.svc.RecordInjury(t.Context(), h.newCase(), h.userID)
	require.NoError(t, err)
	assert.Equal(t, int32(1), first.CaseNumber)
	assert.Equal(t, int16(2026), first.CaseYear)

	second, err := h.svc.RecordInjury(t.Context(), h.newCase(), h.userID)
	require.NoError(t, err)
	assert.Equal(t, int32(2), second.CaseNumber)
}

// Recordability is the employer's judgement, but it should not have to be made
// from a blank field.
func TestRecordInjury_SuggestsTheClassification(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	entity := h.newCase()
	entity.Treatment = worker.TreatmentMedical

	created, err := h.svc.RecordInjury(t.Context(), entity, h.userID)
	require.NoError(t, err)

	assert.Equal(t, worker.CaseOtherRecordable, created.Classification)
	assert.True(t, created.IsRecordable())
}

func TestRecordInjury_FirstAidIsNotRecordable(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	entity := h.newCase()
	entity.Treatment = worker.TreatmentFirstAid

	created, err := h.svc.RecordInjury(t.Context(), entity, h.userID)
	require.NoError(t, err)

	assert.Equal(t, worker.CaseFirstAidOnly, created.Classification)
	assert.False(t, created.IsRecordable())
}

// A claim nobody filed has no dates, however the form was filled in before
// somebody changed their mind.
func TestRecordInjury_ClearsDatesOnAnUnfiledClaim(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	entity := h.newCase()
	filed := occurred2026 + 86400
	entity.ClaimStatus = worker.ClaimNotFiled
	entity.ClaimFiledAt = &filed

	created, err := h.svc.RecordInjury(t.Context(), entity, h.userID)
	require.NoError(t, err)

	assert.Nil(t, created.ClaimFiledAt)
}

func TestLog_CountsOnlyRecordableCases(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	medical := h.newCase()
	medical.Treatment = worker.TreatmentMedical
	_, err := h.svc.RecordInjury(t.Context(), medical, h.userID)
	require.NoError(t, err)

	firstAid := h.newCase()
	firstAid.Treatment = worker.TreatmentFirstAid
	_, err = h.svc.RecordInjury(t.Context(), firstAid, h.userID)
	require.NoError(t, err)

	log, err := h.svc.Log(t.Context(), h.tenant, 2026)
	require.NoError(t, err)

	// Both cases are on the log so the office can see the decisions that were
	// made; only the recordable one counts.
	assert.Len(t, log.Cases, 2)
	assert.Equal(t, 1, log.Totals.TotalRecordableCases)
	assert.Positive(t, log.PostThrough)
	assert.Greater(t, log.PostThrough, log.PostFrom)
	assert.Nil(t, log.TotalRecordableIncidentRate, "no summary yet, so no hours and no rate")
}

func TestLog_RatesOnceHoursAreRecorded(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	medical := h.newCase()
	medical.Treatment = worker.TreatmentMedical
	_, err := h.svc.RecordInjury(t.Context(), medical, h.userID)
	require.NoError(t, err)

	hours := int64(200_000)
	employees := int32(100)
	_, err = h.svc.SaveSummary(t.Context(), &workerinjuryservice.SaveSummaryRequest{
		TenantInfo:       h.tenant,
		Year:             2026,
		TotalHoursWorked: &hours,
		AverageEmployees: &employees,
		UserID:           h.userID,
	})
	require.NoError(t, err)

	log, err := h.svc.Log(t.Context(), h.tenant, 2026)
	require.NoError(t, err)

	require.NotNil(t, log.TotalRecordableIncidentRate)
	assert.InDelta(t, 1.0, *log.TotalRecordableIncidentRate, 0.0001)
}

func (h *harness) draftSummary(t *testing.T) *worker.OSHAAnnualSummary {
	t.Helper()
	hours := int64(83_200)
	employees := int32(40)
	name := "Alex Chen"
	saved, err := h.svc.SaveSummary(t.Context(), &workerinjuryservice.SaveSummaryRequest{
		TenantInfo:       h.tenant,
		Year:             2026,
		TotalHoursWorked: &hours,
		AverageEmployees: &employees,
		ExecutiveName:    &name,
		UserID:           h.userID,
	})
	require.NoError(t, err)
	return saved
}

// The posting window is filled in when the summary is started, so nobody has to
// look up February 1 – April 30 themselves.
func TestSaveSummary_SetsThePostingWindow(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	saved := h.draftSummary(t)

	from, through := worker.PostingWindow(2026)
	require.NotNil(t, saved.PostedFrom)
	require.NotNil(t, saved.PostedThrough)
	assert.Equal(t, from, *saved.PostedFrom)
	assert.Equal(t, through, *saved.PostedThrough)
}

func TestCertifySummary_RecordsWhoSigned(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.draftSummary(t)

	certified, err := h.svc.CertifySummary(t.Context(), h.tenant, 2026, h.userID)
	require.NoError(t, err)

	assert.Equal(t, worker.SummaryCertified, certified.Status)
	require.NotNil(t, certified.CertifiedAt)
	assert.Equal(t, h.userID, certified.CertifiedByID)
}

// Certifying a log that has not finished moving is certifying a number that is
// about to change.
func TestCertifySummary_RefusedWhileCasesAreStillOpen(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.draftSummary(t)

	open := h.newCase()
	open.Treatment = worker.TreatmentMedical
	open.Status = worker.InjuryCaseOpen
	_, err := h.svc.RecordInjury(t.Context(), open, h.userID)
	require.NoError(t, err)

	_, err = h.svc.CertifySummary(t.Context(), h.tenant, 2026, h.userID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "still accruing days")
}

func TestCertifySummary_NeedsASummaryFirst(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, err := h.svc.CertifySummary(t.Context(), h.tenant, 2026, h.userID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "employment figures")
}

// Nobody quietly changes a figure an executive has signed for.
func TestSaveSummary_RefusedOnceCertified(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.draftSummary(t)
	_, err := h.svc.CertifySummary(t.Context(), h.tenant, 2026, h.userID)
	require.NoError(t, err)

	hours := int64(90_000)
	_, err = h.svc.SaveSummary(t.Context(), &workerinjuryservice.SaveSummaryRequest{
		TenantInfo:       h.tenant,
		Year:             2026,
		TotalHoursWorked: &hours,
		UserID:           h.userID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "uncertify")
}

// A signature on figures that have since changed is worse than no signature, so
// reopening clears it.
func TestUncertifySummary_ClearsTheSignature(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.draftSummary(t)
	_, err := h.svc.CertifySummary(t.Context(), h.tenant, 2026, h.userID)
	require.NoError(t, err)

	reopened, err := h.svc.UncertifySummary(t.Context(), h.tenant, 2026, h.userID)
	require.NoError(t, err)

	assert.Equal(t, worker.SummaryDraft, reopened.Status)
	assert.Nil(t, reopened.CertifiedAt)
	assert.True(t, reopened.CertifiedByID.IsNil())

	hours := int64(90_000)
	_, err = h.svc.SaveSummary(t.Context(), &workerinjuryservice.SaveSummaryRequest{
		TenantInfo:       h.tenant,
		Year:             2026,
		TotalHoursWorked: &hours,
		UserID:           h.userID,
	})
	assert.NoError(t, err, "reopening is what makes a correction possible")
}

func TestUpdateInjury_ClosingACaseKeepsItOnTheLog(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	entity := h.newCase()
	entity.Treatment = worker.TreatmentMedical
	created, err := h.svc.RecordInjury(t.Context(), entity, h.userID)
	require.NoError(t, err)

	closed := worker.InjuryCaseClosed
	days := int32(4)
	classification := worker.CaseDaysAway
	updated, err := h.svc.UpdateInjury(t.Context(), &workerinjuryservice.UpdateInjuryRequest{
		TenantInfo:     h.tenant,
		InjuryID:       created.ID,
		Status:         &closed,
		DaysAway:       &days,
		Classification: &classification,
		UserID:         h.userID,
	})
	require.NoError(t, err)

	assert.Equal(t, worker.InjuryCaseClosed, updated.Status)
	assert.Equal(t, int32(4), updated.DaysAway)

	log, err := h.svc.Log(t.Context(), h.tenant, 2026)
	require.NoError(t, err)
	assert.Equal(t, 1, log.Totals.DaysAwayCases)
	assert.Equal(t, 4, log.Totals.TotalDaysAway)
	assert.Equal(t, 0, log.Totals.OpenCases)
}
