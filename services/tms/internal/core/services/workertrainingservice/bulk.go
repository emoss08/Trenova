package workertrainingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// maxBulkAssignWorkers caps one bulk assignment. A fleet-wide rollout of a new
// course is the intended use; anything larger is a data migration and should
// not run inside a request.
const maxBulkAssignWorkers = 500

// BulkAssignRequest opens one or more courses for a set of workers. The caller
// resolves who the workers are — the matrix page already knows, having listed
// them — so this does not re-implement roster filtering.
type BulkAssignRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerIDs  []pulid.ID
	CourseIDs  []pulid.ID
	DueAt      *int64
	Notes      string
	UserID     pulid.ID
}

// BulkAssignOutcome is what happened for one worker and course pair.
type BulkAssignOutcome struct {
	WorkerID pulid.ID
	CourseID pulid.ID
	RecordID pulid.ID
	// Skipped is set when the worker already had that course open. It is not a
	// failure: assigning a course twice is what somebody clicking the same
	// button twice looks like, and the record they already have is the answer.
	Skipped bool
	Error   string
}

// BulkAssignResult reports the whole run. Counts are separated so the toast can
// say "24 assigned, 3 already open" rather than implying three failures.
type BulkAssignResult struct {
	Outcomes      []BulkAssignOutcome
	AssignedCount int
	SkippedCount  int
	FailedCount   int
}

// BulkAssign opens every course for every worker given. It is deliberately
// per-pair rather than one transaction: a fleet rollout that fails on one
// driver should still enrol the other forty-nine, and each assignment is
// independently meaningful.
func (s *Service) BulkAssign(
	ctx context.Context,
	req *BulkAssignRequest,
) (*BulkAssignResult, error) {
	if err := validateBulkAssign(req); err != nil {
		return nil, err
	}

	log := s.l.With(
		zap.String("operation", "BulkAssign"),
		zap.Int("workers", len(req.WorkerIDs)),
		zap.Int("courses", len(req.CourseIDs)),
	)

	result := &BulkAssignResult{
		Outcomes: make([]BulkAssignOutcome, 0, len(req.WorkerIDs)*len(req.CourseIDs)),
	}
	touched := make(map[pulid.ID]struct{}, len(req.WorkerIDs))

	for _, workerID := range req.WorkerIDs {
		// One read of the worker's records covers every course in this run;
		// asking per course would multiply the queries for no new information.
		open, err := s.openCourseIDs(ctx, req.TenantInfo, workerID)
		for _, courseID := range req.CourseIDs {
			outcome := BulkAssignOutcome{WorkerID: workerID, CourseID: courseID}
			switch {
			case err != nil:
				outcome.Error = err.Error()
			default:
				if recordID, already := open[courseID]; already {
					outcome.Skipped = true
					outcome.RecordID = recordID
				} else {
					outcome = s.assignOne(ctx, req, workerID, courseID)
				}
			}
			switch {
			case outcome.Error != "":
				result.FailedCount++
			case outcome.Skipped:
				result.SkippedCount++
			default:
				result.AssignedCount++
				touched[workerID] = struct{}{}
			}
			result.Outcomes = append(result.Outcomes, outcome)
		}
	}

	// One refresh per worker rather than one per assignment: the roll-up is a
	// function of every record, so recomputing it after each course would do
	// the same work several times over.
	for workerID := range touched {
		s.refreshRollupQuietly(ctx, req.TenantInfo, workerID)
	}

	log.Info("bulk training assignment finished",
		zap.Int("assigned", result.AssignedCount),
		zap.Int("skipped", result.SkippedCount),
		zap.Int("failed", result.FailedCount))
	return result, nil
}

// openCourseIDs maps the courses a worker already has open to those records,
// so a rollout can tell "already enrolled" from "newly assigned" instead of
// reporting a duplicate as a failure.
func (s *Service) openCourseIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (map[pulid.ID]pulid.ID, error) {
	records, err := s.repo.ListForWorker(ctx, &repositories.ListWorkerTrainingRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return nil, err
	}
	open := make(map[pulid.ID]pulid.ID, len(records))
	for _, record := range records {
		if record != nil && record.IsOpen() {
			open[record.CourseID] = record.ID
		}
	}
	return open, nil
}

func (s *Service) assignOne(
	ctx context.Context,
	req *BulkAssignRequest,
	workerID pulid.ID,
	courseID pulid.ID,
) BulkAssignOutcome {
	outcome := BulkAssignOutcome{WorkerID: workerID, CourseID: courseID}

	record, err := s.Assign(ctx, &AssignRequest{
		TenantInfo: req.TenantInfo,
		WorkerID:   workerID,
		CourseID:   courseID,
		DueAt:      req.DueAt,
		Notes:      req.Notes,
		UserID:     req.UserID,
	})
	if err != nil {
		outcome.Error = err.Error()
		return outcome
	}
	outcome.RecordID = record.ID
	return outcome
}

func validateBulkAssign(req *BulkAssignRequest) error {
	multiErr := errortypes.NewMultiError()
	if len(req.WorkerIDs) == 0 {
		multiErr.Add("workerIds", errortypes.ErrRequired, "Choose at least one worker")
	}
	if len(req.CourseIDs) == 0 {
		multiErr.Add("courseIds", errortypes.ErrRequired, "Choose at least one course")
	}
	if len(req.WorkerIDs) > maxBulkAssignWorkers {
		multiErr.Add(
			"workerIds",
			errortypes.ErrInvalid,
			"Assign to at most 500 workers at a time",
		)
	}
	if req.DueAt != nil && *req.DueAt <= 0 {
		multiErr.Add("dueAt", errortypes.ErrInvalid, "Due date must be a valid date")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
