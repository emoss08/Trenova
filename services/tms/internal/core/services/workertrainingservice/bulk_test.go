package workertrainingservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workertrainingservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// bulkHarness differs from the shared one in a way this file needs: the worker
// repository answers per id, so a run can mix workers that exist with one that
// does not and the outcomes can be told apart.
type bulkHarness struct {
	svc     *workertrainingservice.Service
	repo    *fakeRepo
	tenant  pagination.TenantInfo
	userID  pulid.ID
	workers map[pulid.ID]*worker.Worker
}

func newBulkHarness(t *testing.T) *bulkHarness {
	t.Helper()
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	h := &bulkHarness{
		repo:    &fakeRepo{},
		tenant:  tenant,
		userID:  pulid.MustNew("usr_"),
		workers: map[pulid.ID]*worker.Worker{},
	}

	workerRepo := mocks.NewMockWorkerRepository(t)
	workerRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req repositories.GetWorkerByIDRequest,
		) (*worker.Worker, error) {
			wrk, ok := h.workers[req.ID]
			if !ok {
				return nil, errortypes.NewNotFoundError("Worker not found")
			}
			return wrk, nil
		}).
		Maybe()
	workerRepo.EXPECT().
		UpdateProfileTrainingRollup(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Maybe()

	h.svc = workertrainingservice.New(workertrainingservice.Params{
		Logger:       zap.NewNop(),
		Repo:         h.repo,
		WorkerRepo:   workerRepo,
		DocumentRepo: mocks.NewMockDocumentRepository(t),
		AuditService: audit,
	})
	return h
}

func (h *bulkHarness) worker() pulid.ID {
	id := pulid.MustNew("wrk_")
	h.workers[id] = &worker.Worker{
		ID:             id,
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		Status:         domaintypes.StatusActive,
		DriverType:     worker.DriverTypeOTR,
		Profile:        &worker.WorkerProfile{HireDate: 1_600_000_000},
	}
	return id
}

func (h *bulkHarness) course(code string) pulid.ID {
	c := &worker.TrainingCourse{
		ID:                      pulid.MustNew("trnc_"),
		OrganizationID:          h.tenant.OrgID,
		BusinessUnitID:          h.tenant.BuID,
		Code:                    code,
		Name:                    code,
		Category:                worker.TrainingCategorySafety,
		Status:                  domaintypes.StatusActive,
		Delivery:                worker.TrainingDeliveryClassroom,
		IsRequired:              true,
		RequiredForDriverTypes:  []worker.DriverType{},
		RenewalWindowDays:       30,
		DueDaysAfterAssignment:  30,
		RequiresAcknowledgement: true,
	}
	h.repo.courses = append(h.repo.courses, c)
	return c.ID
}

func (h *bulkHarness) assign(
	t *testing.T,
	workerIDs []pulid.ID,
	courseIDs []pulid.ID,
) *workertrainingservice.BulkAssignResult {
	t.Helper()
	result, err := h.svc.BulkAssign(t.Context(), &workertrainingservice.BulkAssignRequest{
		TenantInfo: h.tenant,
		WorkerIDs:  workerIDs,
		CourseIDs:  courseIDs,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	return result
}

func TestBulkAssign_OpensEveryCourseForEveryWorker(t *testing.T) {
	h := newBulkHarness(t)
	courseA, courseB := h.course("DEFDRIVE"), h.course("HAZMAT")
	workerA, workerB := h.worker(), h.worker()

	result := h.assign(t, []pulid.ID{workerA, workerB}, []pulid.ID{courseA, courseB})

	assert.Equal(t, 4, result.AssignedCount)
	assert.Zero(t, result.SkippedCount)
	assert.Zero(t, result.FailedCount)
	require.Len(t, result.Outcomes, 4)
	for _, outcome := range result.Outcomes {
		assert.False(t, outcome.RecordID.IsNil(), "every assignment names its record")
	}
}

// Clicking the same rollout twice is the common case, not an error. A worker
// who already has the course open keeps the record they have, and the run
// reports it as skipped so the toast does not cry failure.
func TestBulkAssign_AlreadyOpenIsSkippedNotFailed(t *testing.T) {
	h := newBulkHarness(t)
	courseID := h.course("DEFDRIVE")
	workerID := h.worker()

	first := h.assign(t, []pulid.ID{workerID}, []pulid.ID{courseID})
	require.Equal(t, 1, first.AssignedCount)

	second := h.assign(t, []pulid.ID{workerID}, []pulid.ID{courseID})

	assert.Zero(t, second.AssignedCount)
	assert.Equal(t, 1, second.SkippedCount)
	assert.Zero(t, second.FailedCount)
	require.Len(t, second.Outcomes, 1)
	assert.True(t, second.Outcomes[0].Skipped)
	assert.Equal(
		t,
		first.Outcomes[0].RecordID,
		second.Outcomes[0].RecordID,
		"the skip points at the record they already have",
	)
}

// A fleet rollout that trips over one driver must still enrol the rest. That
// is the whole reason this is not one transaction.
func TestBulkAssign_OneFailureDoesNotAbandonTheRest(t *testing.T) {
	h := newBulkHarness(t)
	courseID := h.course("DEFDRIVE")
	good := h.worker()
	missing := pulid.MustNew("wrk_")

	result := h.assign(t, []pulid.ID{missing, good}, []pulid.ID{courseID})

	assert.Equal(t, 1, result.AssignedCount)
	assert.Equal(t, 1, result.FailedCount)

	var failed, assigned workertrainingservice.BulkAssignOutcome
	for _, outcome := range result.Outcomes {
		if outcome.Error != "" {
			failed = outcome
		} else {
			assigned = outcome
		}
	}
	assert.Equal(t, missing, failed.WorkerID)
	assert.NotEmpty(t, failed.Error, "the failure says which worker and why")
	assert.Equal(t, good, assigned.WorkerID)
}

func TestBulkAssign_RejectsEmptySelections(t *testing.T) {
	h := newBulkHarness(t)
	courseID := h.course("DEFDRIVE")
	workerID := h.worker()

	tests := []struct {
		name  string
		req   *workertrainingservice.BulkAssignRequest
		field string
	}{
		{
			name: "no workers",
			req: &workertrainingservice.BulkAssignRequest{
				TenantInfo: h.tenant,
				CourseIDs:  []pulid.ID{courseID},
				UserID:     h.userID,
			},
			field: "workerIds",
		},
		{
			name: "no courses",
			req: &workertrainingservice.BulkAssignRequest{
				TenantInfo: h.tenant,
				WorkerIDs:  []pulid.ID{workerID},
				UserID:     h.userID,
			},
			field: "courseIds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.svc.BulkAssign(t.Context(), tt.req)
			require.Error(t, err)
			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
			assert.Equal(t, tt.field, multiErr.Errors[0].Field)
		})
	}
}

// Assigning to more workers than a request should carry is a data migration,
// not a rollout, and is refused rather than quietly truncated.
func TestBulkAssign_RefusesOversizedRuns(t *testing.T) {
	h := newBulkHarness(t)
	courseID := h.course("DEFDRIVE")

	workers := make([]pulid.ID, 501)
	for i := range workers {
		workers[i] = pulid.MustNew("wrk_")
	}

	_, err := h.svc.BulkAssign(t.Context(), &workertrainingservice.BulkAssignRequest{
		TenantInfo: h.tenant,
		WorkerIDs:  workers,
		CourseIDs:  []pulid.ID{courseID},
		UserID:     h.userID,
	})

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "workerIds", multiErr.Errors[0].Field)
}
