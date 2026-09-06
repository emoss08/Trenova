package workertrainingservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workertrainingservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
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

const day = int64(86400)

type fakeRepo struct {
	repositories.WorkerTrainingRepository
	courses []*worker.TrainingCourse
	records []*worker.WorkerTrainingRecord
}

func (f *fakeRepo) ListActiveCourses(
	context.Context,
	pagination.TenantInfo,
) ([]*worker.TrainingCourse, error) {
	out := make([]*worker.TrainingCourse, 0, len(f.courses))
	for _, c := range f.courses {
		if c.Status == domaintypes.StatusActive {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetCourseByID(
	_ context.Context,
	req *repositories.GetTrainingCourseByIDRequest,
) (*worker.TrainingCourse, error) {
	for _, c := range f.courses {
		if c.ID == req.ID {
			return c, nil
		}
	}
	return nil, errortypes.NewNotFoundError("TrainingCourse not found")
}

func (f *fakeRepo) ListForWorker(
	_ context.Context,
	req *repositories.ListWorkerTrainingRequest,
) ([]*worker.WorkerTrainingRecord, error) {
	out := make([]*worker.WorkerTrainingRecord, 0, len(f.records))
	for _, r := range f.records {
		if r.WorkerID != req.WorkerID {
			continue
		}
		if !req.IncludeClosed && !r.IsOpen() {
			continue
		}
		if req.IncludeCourse {
			for _, c := range f.courses {
				if c.ID == r.CourseID {
					r.Course = c
				}
			}
		}
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeRepo) GetByID(
	_ context.Context,
	req *repositories.GetWorkerTrainingByIDRequest,
) (*worker.WorkerTrainingRecord, error) {
	for _, r := range f.records {
		if r.ID == req.ID {
			copied := *r
			for _, c := range f.courses {
				if c.ID == r.CourseID {
					copied.Course = c
				}
			}
			return &copied, nil
		}
	}
	return nil, errortypes.NewNotFoundError("WorkerTrainingRecord not found")
}

func (f *fakeRepo) Create(
	_ context.Context,
	entity *worker.WorkerTrainingRecord,
) (*worker.WorkerTrainingRecord, error) {
	for _, r := range f.records {
		if r.WorkerID == entity.WorkerID && r.CourseID == entity.CourseID && r.IsOpen() && entity.IsOpen() {
			return nil, errortypes.NewValidationError("courseId", errortypes.ErrDuplicate, "open")
		}
	}
	entity.ID = pulid.MustNew("wtrn_")
	f.records = append(f.records, entity)
	return entity, nil
}

func (f *fakeRepo) Update(
	_ context.Context,
	entity *worker.WorkerTrainingRecord,
) (*worker.WorkerTrainingRecord, error) {
	for i, r := range f.records {
		if r.ID == entity.ID {
			entity.Version = r.Version + 1
			f.records[i] = entity
			return entity, nil
		}
	}
	return nil, errortypes.NewNotFoundError("WorkerTrainingRecord not found")
}

type harness struct {
	svc    *workertrainingservice.Service
	repo   *fakeRepo
	docs   *mocks.MockDocumentRepository
	tenant pagination.TenantInfo
	wrk    *worker.Worker
	userID pulid.ID
	now    int64
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
		Profile:        &worker.WorkerProfile{HireDate: 1_600_000_000},
	}
	workerRepo := mocks.NewMockWorkerRepository(t)
	workerRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(wrk, nil).Maybe()
	// Every write refreshes the roster cache; the assertions here are about
	// the change itself, not the cache.
	workerRepo.EXPECT().UpdateProfileTrainingRollup(mock.Anything, mock.Anything).Return(nil).Maybe()
	docs := mocks.NewMockDocumentRepository(t)
	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Maybe()

	repo := &fakeRepo{}
	svc := workertrainingservice.New(workertrainingservice.Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		WorkerRepo:   workerRepo,
		DocumentRepo: docs,
		AuditService: audit,
	})
	return &harness{
		svc:    svc,
		repo:   repo,
		docs:   docs,
		tenant: tenant,
		wrk:    wrk,
		userID: pulid.MustNew("usr_"),
		now:    timeutils.NowUnix(),
	}
}

func (h *harness) course(code string, required bool, mutate func(*worker.TrainingCourse)) *worker.TrainingCourse {
	c := &worker.TrainingCourse{
		ID:                      pulid.MustNew("trnc_"),
		OrganizationID:          h.tenant.OrgID,
		BusinessUnitID:          h.tenant.BuID,
		Code:                    code,
		Name:                    code,
		Category:                worker.TrainingCategorySafety,
		Status:                  domaintypes.StatusActive,
		Delivery:                worker.TrainingDeliveryClassroom,
		IsRequired:              required,
		RequiredForDriverTypes:  []worker.DriverType{},
		RenewalWindowDays:       30,
		DueDaysAfterAssignment:  30,
		RequiresAcknowledgement: true,
	}
	if mutate != nil {
		mutate(c)
	}
	h.repo.courses = append(h.repo.courses, c)
	return c
}

func TestAssignRequired_OpensOnlyTheGaps(t *testing.T) {
	h := newHarness(t)
	orientation := h.course("ORIENT", true, nil)
	hazmat := h.course("HAZMAT", true, func(c *worker.TrainingCourse) {
		c.RequiredForDriverTypes = []worker.DriverType{worker.DriverTypeOTR}
	})
	h.course("LOCAL", true, func(c *worker.TrainingCourse) {
		c.RequiredForDriverTypes = []worker.DriverType{worker.DriverTypeLocal}
	})
	h.course("OPTIONAL", false, nil)
	completed := h.now - 100*day
	h.repo.records = append(h.repo.records, &worker.WorkerTrainingRecord{
		ID: pulid.MustNew("wtrn_"), WorkerID: h.wrk.ID, CourseID: orientation.ID,
		Status: worker.TrainingStatusCompleted, CompletedAt: &completed,
	})

	created, err := h.svc.AssignRequired(context.Background(), h.tenant, h.wrk.ID, h.userID)
	require.NoError(t, err)
	require.Len(t, created, 1, "only the missing OTR course is opened")
	assert.Equal(t, hazmat.ID, created[0].CourseID)
	assert.Equal(t, worker.TrainingStatusAssigned, created[0].Status)
	require.NotNil(t, created[0].DueAt)
	assert.Equal(t, created[0].AssignedAt+30*day, *created[0].DueAt, "due date comes from the course")

	again, err := h.svc.AssignRequired(context.Background(), h.tenant, h.wrk.ID, h.userID)
	require.NoError(t, err)
	assert.Empty(t, again, "an open assignment is not duplicated")
}

func TestAssign_RefusesInactiveAndPastDue(t *testing.T) {
	h := newHarness(t)
	retired := h.course("OLD", false, func(c *worker.TrainingCourse) { c.Status = domaintypes.StatusInactive })
	live := h.course("LIVE", false, nil)

	_, err := h.svc.Assign(context.Background(), &workertrainingservice.AssignRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, CourseID: retired.ID, UserID: h.userID,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "courseId", verr.Field)

	past := h.now - 5*day
	_, err = h.svc.Assign(context.Background(), &workertrainingservice.AssignRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, CourseID: live.ID, DueAt: &past, UserID: h.userID,
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "dueAt", multiErr.Errors[0].Field)
}

func TestComplete_ScoresAndSetsExpiry(t *testing.T) {
	h := newHarness(t)
	scored := h.course("SCORED", true, func(c *worker.TrainingCourse) {
		c.PassingScore = decimal.NewNullDecimal(decimal.NewFromInt(80))
		months := int32(12)
		c.ValidityMonths = &months
	})
	record, err := h.svc.Assign(context.Background(), &workertrainingservice.AssignRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, CourseID: scored.ID, UserID: h.userID,
	})
	require.NoError(t, err)

	_, err = h.svc.Complete(context.Background(), &workertrainingservice.CompleteRequest{
		TenantInfo: h.tenant, ID: record.ID, UserID: h.userID,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr, "a scored course needs a score")
	assert.Equal(t, "score", verr.Field)

	failed, err := h.svc.Complete(context.Background(), &workertrainingservice.CompleteRequest{
		TenantInfo: h.tenant, ID: record.ID, Score: decimal.NewNullDecimal(decimal.NewFromInt(60)), UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.TrainingStatusFailed, failed.Status)
	assert.Nil(t, failed.ExpiresAt)
	require.NotNil(t, failed.Passed)
	assert.False(t, *failed.Passed)

	_, err = h.svc.Complete(context.Background(), &workertrainingservice.CompleteRequest{
		TenantInfo: h.tenant, ID: record.ID, Score: decimal.NewNullDecimal(decimal.NewFromInt(90)), UserID: h.userID,
	})
	require.ErrorAs(t, err, &verr, "a closed record cannot be completed again")

	completedAt := time.Date(2026, time.January, 31, 12, 0, 0, 0, time.UTC).Unix()
	direct, err := h.svc.Complete(context.Background(), &workertrainingservice.CompleteRequest{
		TenantInfo:  h.tenant,
		WorkerID:    h.wrk.ID,
		CourseID:    scored.ID,
		CompletedAt: completedAt,
		Score:       decimal.NewNullDecimal(decimal.NewFromInt(95)),
		UserID:      h.userID,
	})
	require.NoError(t, err, "a completion can be filed without an open record")
	assert.Equal(t, worker.TrainingStatusCompleted, direct.Status)
	require.NotNil(t, direct.ExpiresAt)
	assert.Equal(t, time.Date(2027, time.January, 31, 12, 0, 0, 0, time.UTC).Unix(), *direct.ExpiresAt)
	assert.Equal(t, h.userID, direct.RecordedByID)

	summary, err := h.svc.Summary(context.Background(), h.tenant, h.wrk.ID)
	require.NoError(t, err)
	assert.True(t, summary.Compliant)
	assert.Equal(t, worker.TrainingHealthCurrent, summary.Items[0].Health)
}

func TestComplete_DocumentMustBelongToWorker(t *testing.T) {
	h := newHarness(t)
	c := h.course("DOCS", false, nil)
	record, err := h.svc.Assign(context.Background(), &workertrainingservice.AssignRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, CourseID: c.ID, UserID: h.userID,
	})
	require.NoError(t, err)

	strangerDoc := pulid.MustNew("doc_")
	h.docs.EXPECT().GetByID(mock.Anything, mock.MatchedBy(func(req repositories.GetDocumentByIDRequest) bool {
		return req.ID == strangerDoc
	})).Return(&document.Document{ID: strangerDoc, ResourceType: "worker", ResourceID: pulid.MustNew("wrk_").String()}, nil)

	_, err = h.svc.Complete(context.Background(), &workertrainingservice.CompleteRequest{
		TenantInfo: h.tenant, ID: record.ID, DocumentID: strangerDoc, UserID: h.userID,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "documentId", verr.Field)

	ownDoc := pulid.MustNew("doc_")
	h.docs.EXPECT().GetByID(mock.Anything, mock.MatchedBy(func(req repositories.GetDocumentByIDRequest) bool {
		return req.ID == ownDoc
	})).Return(&document.Document{ID: ownDoc, ResourceType: "worker", ResourceID: h.wrk.ID.String()}, nil)
	saved, err := h.svc.Complete(context.Background(), &workertrainingservice.CompleteRequest{
		TenantInfo: h.tenant, ID: record.ID, DocumentID: ownDoc, UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, ownDoc, saved.DocumentID)
	assert.Equal(t, worker.TrainingStatusCompleted, saved.Status)
}

func TestWaiveAndCancel(t *testing.T) {
	h := newHarness(t)
	c := h.course("WAIVE", true, nil)
	record, err := h.svc.Assign(context.Background(), &workertrainingservice.AssignRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, CourseID: c.ID, UserID: h.userID,
	})
	require.NoError(t, err)

	_, err = h.svc.Waive(context.Background(), &workertrainingservice.StatusRequest{
		TenantInfo: h.tenant, ID: record.ID, UserID: h.userID,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr, "a waiver needs a reason")
	assert.Equal(t, "reason", verr.Field)

	waived, err := h.svc.Waive(context.Background(), &workertrainingservice.StatusRequest{
		TenantInfo: h.tenant, ID: record.ID, Reason: "Equivalent certificate on file", UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.TrainingStatusWaived, waived.Status)

	summary, err := h.svc.Summary(context.Background(), h.tenant, h.wrk.ID)
	require.NoError(t, err)
	assert.True(t, summary.Compliant, "a waiver satisfies the requirement")

	_, err = h.svc.Cancel(context.Background(), &workertrainingservice.StatusRequest{
		TenantInfo: h.tenant, ID: record.ID, UserID: h.userID,
	})
	require.ErrorAs(t, err, &verr, "closed records cannot be cancelled")
}

func TestPortalStartAndAcknowledge(t *testing.T) {
	h := newHarness(t)
	selfServe := h.course("ONLINE", true, func(c *worker.TrainingCourse) {
		c.Delivery = worker.TrainingDeliveryOnline
		c.ContentURL = "https://lms.example.com/1"
	})
	scored := h.course("SCORED", true, func(c *worker.TrainingCourse) {
		c.Delivery = worker.TrainingDeliveryOnline
		c.ContentURL = "https://lms.example.com/2"
		c.PassingScore = decimal.NewNullDecimal(decimal.NewFromInt(80))
	})
	open1, err := h.svc.Assign(context.Background(), &workertrainingservice.AssignRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, CourseID: selfServe.ID, UserID: h.userID,
	})
	require.NoError(t, err)
	open2, err := h.svc.Assign(context.Background(), &workertrainingservice.AssignRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, CourseID: scored.ID, UserID: h.userID,
	})
	require.NoError(t, err)

	_, err = h.svc.Start(context.Background(), h.tenant, open1.ID, pulid.MustNew("wrk_"))
	require.Error(t, err, "another driver's record is not found")

	started, err := h.svc.Start(context.Background(), h.tenant, open1.ID, h.wrk.ID)
	require.NoError(t, err)
	assert.Equal(t, worker.TrainingStatusInProgress, started.Status)
	require.NotNil(t, started.StartedAt)

	done, err := h.svc.Acknowledge(context.Background(), h.tenant, open1.ID, h.wrk.ID)
	require.NoError(t, err)
	assert.Equal(t, worker.TrainingStatusCompleted, done.Status, "unscored self-serve courses finish on acknowledgement")
	assert.True(t, done.IsAcknowledged())

	pending, err := h.svc.Acknowledge(context.Background(), h.tenant, open2.ID, h.wrk.ID)
	require.NoError(t, err)
	assert.Equal(t, worker.TrainingStatusInProgress, pending.Status, "scored courses wait for the office")
	assert.True(t, pending.IsAcknowledged())
}
