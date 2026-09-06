package workerleaveservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workerleaveservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func at(year int, month time.Month, day int) int64 {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix()
}

// fakeRepo holds rows so the designation-copying and certification behaviour
// can be exercised for real rather than asserted against canned returns.
type fakeRepo struct {
	control *worker.LeaveControl
	cases   map[pulid.ID]*worker.WorkerLeaveCase
	entries map[pulid.ID]*worker.WorkerLeaveEntry
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		control: worker.DefaultLeaveControl(),
		cases:   map[pulid.ID]*worker.WorkerLeaveCase{},
		entries: map[pulid.ID]*worker.WorkerLeaveEntry{},
	}
}

func (f *fakeRepo) GetControl(
	_ context.Context,
	_ pagination.TenantInfo,
) (*worker.LeaveControl, error) {
	return f.control, nil
}

func (f *fakeRepo) UpdateControl(
	_ context.Context,
	entity *worker.LeaveControl,
) (*worker.LeaveControl, error) {
	f.control = entity
	return entity, nil
}

func (f *fakeRepo) ListCases(
	_ context.Context,
	req *repositories.ListLeaveCasesRequest,
) ([]*worker.WorkerLeaveCase, error) {
	out := make([]*worker.WorkerLeaveCase, 0, len(f.cases))
	for _, entity := range f.cases {
		if !req.WorkerID.IsNil() && entity.WorkerID != req.WorkerID {
			continue
		}
		if req.OpenOnly && !entity.IsOpen() {
			continue
		}
		if req.CertificationOutstandingOnly && !entity.CertificationStatus.IsOutstanding() {
			continue
		}
		out = append(out, entity)
	}
	return out, nil
}

func (f *fakeRepo) CountCases(
	ctx context.Context,
	req *repositories.ListLeaveCasesRequest,
) (int, error) {
	cases, err := f.ListCases(ctx, req)
	if err != nil {
		return 0, err
	}
	return len(cases), nil
}

func (f *fakeRepo) GetCaseByID(
	_ context.Context,
	req *repositories.GetLeaveCaseByIDRequest,
) (*worker.WorkerLeaveCase, error) {
	entity, ok := f.cases[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("case not found")
	}
	return entity, nil
}

func (f *fakeRepo) CreateCase(
	_ context.Context,
	entity *worker.WorkerLeaveCase,
) (*worker.WorkerLeaveCase, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("wlc_")
	}
	f.cases[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateCase(
	_ context.Context,
	entity *worker.WorkerLeaveCase,
) (*worker.WorkerLeaveCase, error) {
	f.cases[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) ListEntries(
	_ context.Context,
	req *repositories.ListLeaveEntriesRequest,
) ([]*worker.WorkerLeaveEntry, error) {
	out := make([]*worker.WorkerLeaveEntry, 0, len(f.entries))
	for _, entity := range f.entries {
		if !req.WorkerID.IsNil() && entity.WorkerID != req.WorkerID {
			continue
		}
		if !req.LeaveCaseID.IsNil() && entity.LeaveCaseID != req.LeaveCaseID {
			continue
		}
		if req.Since > 0 && entity.UsedOn < req.Since {
			continue
		}
		out = append(out, entity)
	}
	return out, nil
}

func (f *fakeRepo) ListEntriesByCaseIDs(
	_ context.Context,
	req *repositories.ListLeaveEntriesByCaseIDsRequest,
) (map[pulid.ID][]*worker.WorkerLeaveEntry, error) {
	wanted := make(map[pulid.ID]struct{}, len(req.LeaveCaseIDs))
	for _, id := range req.LeaveCaseIDs {
		wanted[id] = struct{}{}
	}
	matched := make([]*worker.WorkerLeaveEntry, 0, len(f.entries))
	for _, entity := range f.entries {
		if _, ok := wanted[entity.LeaveCaseID]; ok {
			matched = append(matched, entity)
		}
	}
	return sliceutils.GroupBy(matched, func(e *worker.WorkerLeaveEntry) pulid.ID {
		return e.LeaveCaseID
	}), nil
}

func (f *fakeRepo) GetEntryByID(
	_ context.Context,
	req *repositories.GetLeaveEntryByIDRequest,
) (*worker.WorkerLeaveEntry, error) {
	entity, ok := f.entries[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("entry not found")
	}
	return entity, nil
}

func (f *fakeRepo) CreateEntry(
	_ context.Context,
	entity *worker.WorkerLeaveEntry,
) (*worker.WorkerLeaveEntry, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("wle_")
	}
	f.entries[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateEntry(
	_ context.Context,
	entity *worker.WorkerLeaveEntry,
) (*worker.WorkerLeaveEntry, error) {
	f.entries[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) DeleteEntry(
	_ context.Context,
	req *repositories.GetLeaveEntryByIDRequest,
) error {
	delete(f.entries, req.ID)
	return nil
}

type harness struct {
	svc      *workerleaveservice.Service
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
		Return(&worker.Worker{
			ID:             workerID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Profile:        &worker.WorkerProfile{HireDate: at(2020, time.January, 6)},
		}, nil).
		Maybe()

	repo := newFakeRepo()

	return &harness{
		svc: workerleaveservice.NewWithDeps(workerleaveservice.Deps{
			Repo:       repo,
			WorkerRepo: workerRepo,
		}),
		repo:     repo,
		tenant:   pagination.TenantInfo{OrgID: orgID, BuID: buID},
		workerID: workerID,
		userID:   pulid.MustNew("usr_"),
	}
}

func (h *harness) openCase(t *testing.T) *worker.WorkerLeaveCase {
	t.Helper()
	created, err := h.svc.OpenCase(t.Context(), &worker.WorkerLeaveCase{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		LeaveType:      worker.LeaveTypeFMLA,
		Frequency:      worker.LeaveIntermittent,
		StartsAt:       at(2026, time.March, 2),
		RequestedAt:    at(2026, time.February, 20),
	}, h.userID)
	require.NoError(t, err)
	return created
}

func (h *harness) approved(t *testing.T, designate bool) *worker.WorkerLeaveCase {
	t.Helper()
	created := h.openCase(t)
	decided, err := h.svc.DecideCase(t.Context(), &workerleaveservice.DecideCaseRequest{
		TenantInfo: h.tenant,
		CaseID:     created.ID,
		Approve:    true,
		Designate:  designate,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	return decided
}

func TestOpenCase_Defaults(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	created := h.openCase(t)

	assert.Equal(t, worker.LeaveCasePending, created.Status)
	assert.False(t, created.FMLADesignated)
	assert.Equal(t, worker.CertificationNotRequired, created.CertificationStatus)
	assert.Equal(t, h.userID, created.RecordedByID)
}

// Approving and designating are different decisions: leave can be granted
// without counting against the entitlement (29 CFR 825.301).
func TestDecideCase_ApproveWithoutDesignating(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	decided := h.approved(t, false)

	assert.Equal(t, worker.LeaveCaseApproved, decided.Status)
	assert.False(t, decided.FMLADesignated)
	assert.False(t, decided.CountsAgainstEntitlement())
	require.NotNil(t, decided.DecidedAt)
	assert.Equal(t, h.userID, decided.DecidedByID)
}

// Denied leave cannot draw down an entitlement, whatever the caller asked for.
func TestDecideCase_DenyingClearsTheDesignation(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	created := h.openCase(t)

	denied, err := h.svc.DecideCase(t.Context(), &workerleaveservice.DecideCaseRequest{
		TenantInfo: h.tenant,
		CaseID:     created.ID,
		Approve:    false,
		Designate:  true,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	assert.Equal(t, worker.LeaveCaseDenied, denied.Status)
	assert.False(t, denied.FMLADesignated)
}

func TestRecordDay_RefusedOnAClosedCase(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	decided := h.approved(t, true)
	_, err := h.svc.CloseCase(t.Context(), h.tenant, decided.ID, h.userID)
	require.NoError(t, err)

	_, err = h.svc.RecordDay(t.Context(), &workerleaveservice.RecordDayRequest{
		TenantInfo: h.tenant,
		CaseID:     decided.ID,
		UsedOn:     at(2026, time.March, 3),
		Hours:      decimal.NewFromInt(8),
		UserID:     h.userID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

// Whether a day counts is copied from the case as it stood when the day was
// recorded. Undesignating the case months later must not rewrite what was
// already counted against a period that has since closed.
func TestRecordDay_CopiesTheDesignationAtTheTime(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	decided := h.approved(t, true)

	entry, err := h.svc.RecordDay(t.Context(), &workerleaveservice.RecordDayRequest{
		TenantInfo: h.tenant,
		CaseID:     decided.ID,
		UsedOn:     at(2026, time.March, 3),
		Hours:      decimal.NewFromInt(8),
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.True(t, entry.CountsAgainstEntitlement)

	// The employer changes its mind about the case.
	stored := h.repo.cases[decided.ID]
	stored.FMLADesignated = false

	assert.True(t, h.repo.entries[entry.ID].CountsAgainstEntitlement,
		"the day already counted stays counted")
}

func TestRecordDay_UndesignatedCaseDoesNotDrawDown(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	decided := h.approved(t, false)

	entry, err := h.svc.RecordDay(t.Context(), &workerleaveservice.RecordDayRequest{
		TenantInfo: h.tenant,
		CaseID:     decided.ID,
		UsedOn:     at(2026, time.March, 3),
		Hours:      decimal.NewFromInt(8),
		UserID:     h.userID,
	})
	require.NoError(t, err)

	assert.False(t, entry.CountsAgainstEntitlement)
}

func TestEntitlement_DrawsDownDesignatedDays(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	decided := h.approved(t, true)

	for _, day := range []int{3, 4, 5} {
		_, err := h.svc.RecordDay(t.Context(), &workerleaveservice.RecordDayRequest{
			TenantInfo: h.tenant,
			CaseID:     decided.ID,
			UsedOn:     at(2026, time.March, day),
			Hours:      decimal.NewFromInt(8),
			UserID:     h.userID,
		})
		require.NoError(t, err)
	}

	entitlement, err := h.svc.Entitlement(
		t.Context(),
		h.tenant,
		h.workerID,
		at(2026, time.April, 1),
	)
	require.NoError(t, err)

	assert.True(t, decimal.NewFromInt(24).Equal(entitlement.UsedHours),
		"used: %s", entitlement.UsedHours)
	assert.True(t, decimal.NewFromInt(456).Equal(entitlement.RemainingHours))
	assert.True(t, entitlement.EligibleOnTenure)
	assert.Equal(t, 1, entitlement.OpenCaseCount)
}

// The deadline comes from the organisation's setting, which defaults to the
// fifteen calendar days 29 CFR 825.305(b) allows.
func TestRequestCertification_SetsTheDeadline(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	decided := h.approved(t, true)

	updated, err := h.svc.RequestCertification(
		t.Context(),
		&workerleaveservice.RequestCertificationRequest{
			TenantInfo: h.tenant,
			CaseID:     decided.ID,
			UserID:     h.userID,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, worker.CertificationRequested, updated.CertificationStatus)
	require.NotNil(t, updated.CertificationRequestedAt)
	require.NotNil(t, updated.CertificationDueAt)
	assert.Equal(t,
		*updated.CertificationRequestedAt+15*86400,
		*updated.CertificationDueAt)
}

func TestRecordCertification_StampsTheArrival(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	decided := h.approved(t, true)
	_, err := h.svc.RequestCertification(
		t.Context(),
		&workerleaveservice.RequestCertificationRequest{
			TenantInfo: h.tenant,
			CaseID:     decided.ID,
			UserID:     h.userID,
		},
	)
	require.NoError(t, err)

	updated, err := h.svc.RecordCertification(
		t.Context(),
		&workerleaveservice.RecordCertificationRequest{
			TenantInfo: h.tenant,
			CaseID:     decided.ID,
			Status:     worker.CertificationReceived,
			UserID:     h.userID,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, worker.CertificationReceived, updated.CertificationStatus)
	require.NotNil(t, updated.CertificationReceivedAt)
}

func TestCloseCase_NeedsADecisionFirst(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	created := h.openCase(t)

	_, err := h.svc.CloseCase(t.Context(), h.tenant, created.ID, h.userID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Decide the case")
}

// Closing a case does not give the entitlement back: the days were taken.
func TestCloseCase_KeepsTheDaysDrawnDown(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	decided := h.approved(t, true)
	_, err := h.svc.RecordDay(t.Context(), &workerleaveservice.RecordDayRequest{
		TenantInfo: h.tenant,
		CaseID:     decided.ID,
		UsedOn:     at(2026, time.March, 3),
		Hours:      decimal.NewFromInt(8),
		UserID:     h.userID,
	})
	require.NoError(t, err)

	_, err = h.svc.CloseCase(t.Context(), h.tenant, decided.ID, h.userID)
	require.NoError(t, err)

	entitlement, err := h.svc.Entitlement(
		t.Context(),
		h.tenant,
		h.workerID,
		at(2026, time.April, 1),
	)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(8).Equal(entitlement.UsedHours))
	assert.Equal(t, 0, entitlement.OpenCaseCount)
}

// Removing a day recorded in error gives the hours back, which is the whole
// point of being able to remove one.
func TestDeleteDay_GivesTheHoursBack(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	decided := h.approved(t, true)
	entry, err := h.svc.RecordDay(t.Context(), &workerleaveservice.RecordDayRequest{
		TenantInfo: h.tenant,
		CaseID:     decided.ID,
		UsedOn:     at(2026, time.March, 3),
		Hours:      decimal.NewFromInt(8),
		UserID:     h.userID,
	})
	require.NoError(t, err)

	require.NoError(t, h.svc.DeleteDay(t.Context(), h.tenant, entry.ID, h.userID))

	entitlement, err := h.svc.Entitlement(
		t.Context(),
		h.tenant,
		h.workerID,
		at(2026, time.April, 1),
	)
	require.NoError(t, err)
	assert.True(t, entitlement.UsedHours.IsZero())
}

func TestUpdateControl_RejectsLessThanTheStatute(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	weeks := decimal.NewFromInt(6)
	_, err := h.svc.UpdateControl(t.Context(), &workerleaveservice.UpdateControlRequest{
		TenantInfo:       h.tenant,
		EntitlementWeeks: &weeks,
		UserID:           h.userID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least twelve weeks")
}

func TestUpdateControl_ChangesTheMeasurementMethod(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	method := worker.MeasureCalendarYear
	updated, err := h.svc.UpdateControl(t.Context(), &workerleaveservice.UpdateControlRequest{
		TenantInfo:        h.tenant,
		MeasurementMethod: &method,
		UserID:            h.userID,
	})
	require.NoError(t, err)

	assert.Equal(t, worker.MeasureCalendarYear, updated.MeasurementMethod)

	entitlement, err := h.svc.Entitlement(
		t.Context(),
		h.tenant,
		h.workerID,
		at(2026, time.June, 15),
	)
	require.NoError(t, err)
	assert.Equal(t, worker.MeasureCalendarYear, entitlement.Method)
	assert.Equal(t, at(2026, time.January, 1), entitlement.Window.From)
}
