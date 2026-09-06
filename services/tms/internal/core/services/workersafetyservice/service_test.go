package workersafetyservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workeremploymentservice"
	"github.com/emoss08/trenova/internal/core/services/workersafetyservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const day = int64(86400)

type fakeRepo struct {
	repositories.WorkerSafetyRepository
	events       []*worker.WorkerSafetyEvent
	actions      []*worker.WorkerDisciplinaryAction
	recognitions []*worker.WorkerRecognition
}

func (f *fakeRepo) ListEvents(_ context.Context, req *repositories.ListWorkerSafetyEventsRequest) ([]*worker.WorkerSafetyEvent, error) {
	out := make([]*worker.WorkerSafetyEvent, 0, len(f.events))
	for _, e := range f.events {
		if e.WorkerID == req.WorkerID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetEventByID(_ context.Context, req *repositories.GetWorkerSafetyEventByIDRequest) (*worker.WorkerSafetyEvent, error) {
	for _, e := range f.events {
		if e.ID == req.ID {
			copied := *e
			return &copied, nil
		}
	}
	return nil, errortypes.NewNotFoundError("WorkerSafetyEvent not found")
}

func (f *fakeRepo) CreateEvent(_ context.Context, e *worker.WorkerSafetyEvent) (*worker.WorkerSafetyEvent, error) {
	e.ID = pulid.MustNew("wsev_")
	f.events = append(f.events, e)
	return e, nil
}

func (f *fakeRepo) UpdateEvent(_ context.Context, e *worker.WorkerSafetyEvent) (*worker.WorkerSafetyEvent, error) {
	for i, existing := range f.events {
		if existing.ID == e.ID {
			e.Version = existing.Version + 1
			f.events[i] = e
			return e, nil
		}
	}
	return nil, errortypes.NewNotFoundError("WorkerSafetyEvent not found")
}

func (f *fakeRepo) DeleteEvent(_ context.Context, req *repositories.GetWorkerSafetyEventByIDRequest) error {
	for i, e := range f.events {
		if e.ID == req.ID {
			f.events = append(f.events[:i], f.events[i+1:]...)
			return nil
		}
	}
	return errortypes.NewNotFoundError("WorkerSafetyEvent not found")
}

func (f *fakeRepo) ListActions(_ context.Context, req *repositories.ListWorkerDisciplinaryActionsRequest) ([]*worker.WorkerDisciplinaryAction, error) {
	out := make([]*worker.WorkerDisciplinaryAction, 0, len(f.actions))
	for _, a := range f.actions {
		if a.WorkerID == req.WorkerID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetActionByID(_ context.Context, req *repositories.GetWorkerDisciplinaryActionByIDRequest) (*worker.WorkerDisciplinaryAction, error) {
	for _, a := range f.actions {
		if a.ID == req.ID {
			copied := *a
			return &copied, nil
		}
	}
	return nil, errortypes.NewNotFoundError("WorkerDisciplinaryAction not found")
}

func (f *fakeRepo) CreateAction(_ context.Context, a *worker.WorkerDisciplinaryAction) (*worker.WorkerDisciplinaryAction, error) {
	a.ID = pulid.MustNew("wdac_")
	f.actions = append(f.actions, a)
	return a, nil
}

func (f *fakeRepo) UpdateAction(_ context.Context, a *worker.WorkerDisciplinaryAction) (*worker.WorkerDisciplinaryAction, error) {
	for i, existing := range f.actions {
		if existing.ID == a.ID {
			a.Version = existing.Version + 1
			f.actions[i] = a
			return a, nil
		}
	}
	return nil, errortypes.NewNotFoundError("WorkerDisciplinaryAction not found")
}

func (f *fakeRepo) ListRecognitions(_ context.Context, req *repositories.ListWorkerRecognitionsRequest) ([]*worker.WorkerRecognition, error) {
	out := make([]*worker.WorkerRecognition, 0, len(f.recognitions))
	for _, r := range f.recognitions {
		if r.WorkerID == req.WorkerID && (!req.VisibleOnly || r.VisibleToWorker) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRepo) CreateRecognition(_ context.Context, r *worker.WorkerRecognition) (*worker.WorkerRecognition, error) {
	r.ID = pulid.MustNew("wrec_")
	f.recognitions = append(f.recognitions, r)
	return r, nil
}

type fakeEmployment struct {
	recorded []*workeremploymentservice.RecordRequest
}

func (f *fakeEmployment) Record(_ context.Context, req *workeremploymentservice.RecordRequest) (*workeremploymentservice.RecordResult, error) {
	f.recorded = append(f.recorded, req)
	return &workeremploymentservice.RecordResult{
		Event: &worker.WorkerEmploymentEvent{ID: pulid.MustNew("wee_"), Kind: req.Kind},
	}, nil
}

type harness struct {
	svc        *workersafetyservice.Service
	repo       *fakeRepo
	employment *fakeEmployment
	tenant     pagination.TenantInfo
	wrk        *worker.Worker
	userID     pulid.ID
	now        int64
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	wrk := &worker.Worker{
		ID:             pulid.MustNew("wrk_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Status:         domaintypes.StatusActive,
		DriverType:     worker.DriverTypeOTR,
		UserID:         pulid.MustNew("usr_"),
		Profile:        &worker.WorkerProfile{HireDate: 1_600_000_000},
	}
	workerRepo := mocks.NewMockWorkerRepository(t)
	workerRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(wrk, nil).Maybe()
	// Every write refreshes the roster cache; the assertions here are about
	// the change itself, not the cache.
	workerRepo.EXPECT().UpdateProfileSafetyRollup(mock.Anything, mock.Anything).Return(nil).Maybe()
	docs := mocks.NewMockDocumentRepository(t)
	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Maybe()

	repo := &fakeRepo{}
	employment := &fakeEmployment{}
	svc := workersafetyservice.NewWithDeps(workersafetyservice.Deps{
		Repo:         repo,
		WorkerRepo:   workerRepo,
		DocumentRepo: docs,
		Employment:   employment,
		AuditService: audit,
	})
	return &harness{
		svc:        svc,
		repo:       repo,
		employment: employment,
		tenant:     tenant,
		wrk:        wrk,
		userID:     pulid.MustNew("usr_"),
		now:        timeutils.NowUnix(),
	}
}

func (h *harness) event(kind worker.SafetyEventKind, mutate func(*worker.WorkerSafetyEvent)) *worker.WorkerSafetyEvent {
	e := &worker.WorkerSafetyEvent{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.wrk.ID,
		Kind:           kind,
		Severity:       worker.SafetySeverityModerate,
		OccurredAt:     h.now - 3*day,
		Description:    "Backed into a dock door at the consignee",
	}
	if mutate != nil {
		mutate(e)
	}
	return e
}

func TestCreateEvent_DefaultsExpiryAndScrubsKindSpecificFields(t *testing.T) {
	h := newHarness(t)

	created, err := h.svc.CreateEvent(context.Background(), h.event(worker.SafetyEventAccident, func(e *worker.WorkerSafetyEvent) {
		e.Points = 4
		e.Preventable = true
		e.InspectionResult = worker.InspectionResultPass
	}), h.userID)
	require.NoError(t, err)
	assert.Equal(t, worker.SafetyEventStatusOpen, created.Status)
	assert.Equal(t, h.userID, created.RecordedByID)
	assert.Empty(t, created.InspectionResult, "only inspections carry a result")
	require.NotNil(t, created.PointsExpireAt)
	assert.Equal(t, timeutils.AddMonthsUTC(created.OccurredAt, 24), *created.PointsExpireAt)

	_, err = h.svc.CreateEvent(context.Background(), h.event(worker.SafetyEventInspection, nil), h.userID)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr, "an inspection needs a result")
	assert.Equal(t, "inspectionResult", multiErr.Errors[0].Field)

	oos, err := h.svc.CreateEvent(context.Background(), h.event(worker.SafetyEventInspection, func(e *worker.WorkerSafetyEvent) {
		e.InspectionResult = worker.InspectionResultOutOfService
		e.Points = 5
	}), h.userID)
	require.NoError(t, err)
	assert.True(t, oos.OutOfService, "an out-of-service result implies the flag")

	card, err := h.svc.Scorecard(context.Background(), h.tenant, h.wrk.ID)
	require.NoError(t, err)
	assert.Equal(t, int32(9), card.ActivePoints)
	assert.Equal(t, int32(1), card.PreventableAccidents)
	assert.Equal(t, int32(1), card.OutOfServiceOrders)
	assert.Equal(t, worker.SafetyRatingAtRisk, card.Rating)
}

func TestCloseAndDeleteEvent(t *testing.T) {
	h := newHarness(t)
	created, err := h.svc.CreateEvent(context.Background(), h.event(worker.SafetyEventIncident, nil), h.userID)
	require.NoError(t, err)

	_, err = h.svc.CloseEvent(context.Background(), &workersafetyservice.EventStatusRequest{
		ID: created.ID, TenantInfo: h.tenant, UserID: h.userID,
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr, "closing needs a resolution")
	assert.Equal(t, "resolution", multiErr.Errors[0].Field)

	closed, err := h.svc.CloseEvent(context.Background(), &workersafetyservice.EventStatusRequest{
		ID: created.ID, TenantInfo: h.tenant, Resolution: "Coached; dock procedure reviewed", UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.SafetyEventStatusClosed, closed.Status)
	require.NotNil(t, closed.ClosedAt)
	assert.Equal(t, h.userID, closed.ClosedByID)

	err = h.svc.DeleteEvent(context.Background(), h.tenant, created.ID, h.userID)
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr, "closed events are history")

	reopened, err := h.svc.ReopenEvent(context.Background(), &workersafetyservice.EventStatusRequest{
		ID: created.ID, TenantInfo: h.tenant, UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.SafetyEventStatusOpen, reopened.Status)
	assert.Nil(t, reopened.ClosedAt)
	require.NoError(t, h.svc.DeleteEvent(context.Background(), h.tenant, created.ID, h.userID))
	assert.Empty(t, h.repo.events)
}

func TestIssueAction_ClimbsTheLadderAndRecordsEmployment(t *testing.T) {
	h := newHarness(t)

	ladder, err := h.svc.Ladder(context.Background(), h.tenant, h.wrk.ID)
	require.NoError(t, err)
	assert.Equal(t, worker.DisciplinaryLevelCoaching, ladder.SuggestedLevel)

	first, err := h.svc.IssueAction(context.Background(), &workersafetyservice.IssueActionRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, Level: worker.DisciplinaryLevelWrittenWarning,
		Reason: "Two missed pre-trip inspections", UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.DisciplinaryStatusActive, first.Action.Status)
	require.NotNil(t, first.Action.ExpiresAt)
	assert.Equal(t, timeutils.AddMonthsUTC(first.Action.IssuedAt, 12), *first.Action.ExpiresAt)
	assert.Nil(t, first.EmploymentEvent)
	assert.Empty(t, h.employment.recorded, "a warning does not touch the timeline")

	ladder, err = h.svc.Ladder(context.Background(), h.tenant, h.wrk.ID)
	require.NoError(t, err)
	assert.Equal(t, worker.DisciplinaryLevelFinalWarning, ladder.SuggestedLevel)

	_, err = h.svc.IssueAction(context.Background(), &workersafetyservice.IssueActionRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, Level: worker.DisciplinaryLevelSuspension,
		Reason: "Third missed inspection", UserID: h.userID,
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr, "a suspension needs its length")
	assert.Equal(t, "suspensionDays", multiErr.Errors[0].Field)

	days := int32(3)
	suspension, err := h.svc.IssueAction(context.Background(), &workersafetyservice.IssueActionRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, Level: worker.DisciplinaryLevelSuspension,
		Reason: "Third missed inspection", SuspensionDays: &days, RecordEmploymentEvent: true, UserID: h.userID,
	})
	require.NoError(t, err)
	require.NotNil(t, suspension.EmploymentEvent)
	require.Len(t, h.employment.recorded, 1)
	assert.Equal(t, worker.EmploymentEventSuspended, h.employment.recorded[0].Kind)
	assert.Equal(t, "Third missed inspection", h.employment.recorded[0].Reason)

	rescinded, err := h.svc.RescindAction(context.Background(), &workersafetyservice.ActionStatusRequest{
		ID: first.Action.ID, TenantInfo: h.tenant, Reason: "Inspections were logged under the wrong unit", UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.DisciplinaryStatusRescinded, rescinded.Status)

	acknowledged, err := h.svc.AcknowledgeAction(context.Background(), h.tenant, suspension.Action.ID, h.wrk.ID, "Understood")
	require.NoError(t, err)
	assert.True(t, acknowledged.IsAcknowledged())
	assert.Equal(t, "Understood", acknowledged.WorkerComment)

	_, err = h.svc.AcknowledgeAction(context.Background(), h.tenant, suspension.Action.ID, pulid.MustNew("wrk_"), "")
	require.Error(t, err, "another driver cannot acknowledge it")
}

func TestGiveRecognition(t *testing.T) {
	h := newHarness(t)
	given, err := h.svc.GiveRecognition(context.Background(), &worker.WorkerRecognition{
		OrganizationID: h.tenant.OrgID, BusinessUnitID: h.tenant.BuID, WorkerID: h.wrk.ID,
		Kind: worker.RecognitionKindSafetyMilestone, Title: "  One year accident-free  ", VisibleToWorker: true,
	}, h.userID)
	require.NoError(t, err)
	assert.Equal(t, "One year accident-free", given.Title)
	assert.Equal(t, h.userID, given.AwardedByID)
	assert.NotZero(t, given.OccurredAt)

	hidden, err := h.svc.GiveRecognition(context.Background(), &worker.WorkerRecognition{
		OrganizationID: h.tenant.OrgID, BusinessUnitID: h.tenant.BuID, WorkerID: h.wrk.ID,
		Kind: worker.RecognitionKindOther, Title: "Internal note",
	}, h.userID)
	require.NoError(t, err)
	assert.False(t, hidden.VisibleToWorker)

	visible, err := h.svc.ListRecognitions(context.Background(), h.tenant, h.wrk.ID, true)
	require.NoError(t, err)
	assert.Len(t, visible, 1)
}
