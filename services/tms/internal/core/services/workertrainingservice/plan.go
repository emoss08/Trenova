package workertrainingservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type RecordChange = services.RecordChange[worker.WorkerTrainingRecord]

// PlanAssign is the record Assign would open, with its course, without
// opening it.
func (s *Service) PlanAssign(
	ctx context.Context,
	req *AssignRequest,
) (*worker.WorkerTrainingRecord, error) {
	course, err := s.repo.GetCourseByID(ctx, &repositories.GetTrainingCourseByIDRequest{
		ID:         req.CourseID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if course.Status != domaintypes.StatusActive {
		return nil, errortypes.NewValidationError(
			"courseId",
			errortypes.ErrInvalidOperation,
			"Inactive courses cannot be assigned",
		)
	}
	if _, err = s.loadWorker(ctx, req.TenantInfo, req.WorkerID); err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	entity := &worker.WorkerTrainingRecord{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		WorkerID:       req.WorkerID,
		CourseID:       req.CourseID,
		Status:         worker.TrainingStatusAssigned,
		AssignedAt:     now,
		DueAt:          req.DueAt,
		AssignedByID:   req.UserID,
		Notes:          strings.TrimSpace(req.Notes),
		Course:         course,
	}
	if entity.DueAt == nil {
		entity.DueAt = course.DueFor(now)
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if entity.DueAt != nil && *entity.DueAt < now-secondsPerDay {
		multiErr.Add("dueAt", errortypes.ErrInvalid, "Due date cannot be in the past")
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return entity, nil
}

// BulkAssignPlan is what BulkAssign would do with each worker and course:
// open a record, leave one already open, or refuse the pair.
type BulkAssignPlan struct {
	Records []*worker.WorkerTrainingRecord
	Skipped []BulkAssignOutcome
	Failed  []BulkAssignOutcome
}

func (s *Service) PlanBulkAssign(
	ctx context.Context,
	req *BulkAssignRequest,
) (*BulkAssignPlan, error) {
	if err := validateBulkAssign(req); err != nil {
		return nil, err
	}

	plan := &BulkAssignPlan{
		Records: make([]*worker.WorkerTrainingRecord, 0, len(req.WorkerIDs)*len(req.CourseIDs)),
	}
	for _, workerID := range req.WorkerIDs {
		open, err := s.openCourseIDs(ctx, req.TenantInfo, workerID)
		for _, courseID := range req.CourseIDs {
			outcome := BulkAssignOutcome{WorkerID: workerID, CourseID: courseID}
			if err != nil {
				outcome.Error = err.Error()
				plan.Failed = append(plan.Failed, outcome)
				continue
			}
			if recordID, already := open[courseID]; already {
				outcome.Skipped = true
				outcome.RecordID = recordID
				plan.Skipped = append(plan.Skipped, outcome)
				continue
			}
			record, assignErr := s.PlanAssign(ctx, &AssignRequest{
				TenantInfo: req.TenantInfo,
				WorkerID:   workerID,
				CourseID:   courseID,
				DueAt:      req.DueAt,
				Notes:      req.Notes,
				UserID:     req.UserID,
			})
			if assignErr != nil {
				outcome.Error = assignErr.Error()
				plan.Failed = append(plan.Failed, outcome)
				continue
			}
			plan.Records = append(plan.Records, record)
		}
	}
	return plan, nil
}

// PlanAssignRequired is the required courses AssignRequired would open for
// the worker: every one they are missing, have failed or have let lapse.
func (s *Service) PlanAssignRequired(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) ([]*worker.TrainingCourse, error) {
	wrk, err := s.loadWorker(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	summary, err := s.summarize(ctx, tenantInfo, wrk)
	if err != nil {
		return nil, err
	}
	return summary.RequiredGaps(), nil
}

// PlanComplete is the record Complete would leave, graded the same way. Before
// is nil when the completion would file a new record.
func (s *Service) PlanComplete(ctx context.Context, req *CompleteRequest) (*RecordChange, error) {
	record, original, err := s.openRecordFor(ctx, req)
	if err != nil {
		return nil, err
	}
	course := record.Course

	completedAt := req.CompletedAt
	if completedAt <= 0 {
		completedAt = timeutils.NowUnix()
	}
	if completedAt > timeutils.NowUnix()+secondsPerDay {
		return nil, errortypes.NewValidationError(
			"completedAt",
			errortypes.ErrInvalid,
			"Completion date cannot be in the future",
		)
	}
	passed, err := course.Grade(req.Score)
	if err != nil {
		return nil, err
	}
	if !req.DocumentID.IsNil() {
		if err = s.requireWorkerDocument(ctx, req.TenantInfo, record, req.DocumentID); err != nil {
			return nil, err
		}
		record.DocumentID = req.DocumentID
	}

	record.CompletedAt = &completedAt
	record.Score = req.Score
	record.Passed = &passed
	record.RecordedByID = req.UserID
	if notes := strings.TrimSpace(req.Notes); notes != "" {
		record.Notes = notes
	}
	if passed {
		record.Status = worker.TrainingStatusCompleted
		record.ExpiresAt = course.ExpiryFor(completedAt)
	} else {
		record.Status = worker.TrainingStatusFailed
		record.ExpiresAt = nil
	}

	multiErr := errortypes.NewMultiError()
	record.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return &RecordChange{Before: original, After: record}, nil
}

func (s *Service) loadOpenRecord(
	ctx context.Context,
	req *StatusRequest,
	refusal string,
) (*worker.WorkerTrainingRecord, error) {
	original, err := s.loadRecord(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if !original.IsOpen() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			refusal,
		)
	}
	return original, nil
}

func (s *Service) PlanWaive(ctx context.Context, req *StatusRequest) (*RecordChange, error) {
	original, err := s.loadOpenRecord(ctx, req,
		"Only an assigned or in-progress course can be waived")
	if err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Say why the course is waived",
		)
	}

	updated := *original
	updated.Status = worker.TrainingStatusWaived
	updated.WaivedReason = reason
	updated.RecordedByID = req.UserID
	return &RecordChange{Before: original, After: &updated}, nil
}

func (s *Service) PlanCancel(ctx context.Context, req *StatusRequest) (*RecordChange, error) {
	original, err := s.loadOpenRecord(ctx, req,
		"Only an assigned or in-progress course can be cancelled")
	if err != nil {
		return nil, err
	}

	updated := *original
	updated.Status = worker.TrainingStatusCancelled
	updated.RecordedByID = req.UserID
	if reason := strings.TrimSpace(req.Reason); reason != "" {
		updated.Notes = reason
	}
	return &RecordChange{Before: original, After: &updated}, nil
}

func (s *Service) PlanAttachDocument(
	ctx context.Context,
	req *AttachDocumentRequest,
) (*RecordChange, error) {
	original, err := s.loadRecord(ctx, req.TenantInfo, req.ID, 0)
	if err != nil {
		return nil, err
	}
	if original.Status == worker.TrainingStatusCancelled {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Cancelled assignments cannot take documents",
		)
	}
	if err = s.requireWorkerDocument(ctx, req.TenantInfo, original, req.DocumentID); err != nil {
		return nil, err
	}

	updated := *original
	updated.DocumentID = req.DocumentID
	return &RecordChange{Before: original, After: &updated}, nil
}
