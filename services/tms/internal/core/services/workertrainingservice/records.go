package workertrainingservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (s *Service) ListForWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	includeClosed bool,
) ([]*worker.WorkerTrainingRecord, error) {
	return s.repo.ListForWorker(ctx, &repositories.ListWorkerTrainingRequest{
		TenantInfo:      tenantInfo,
		WorkerID:        workerID,
		IncludeClosed:   includeClosed,
		IncludeCourse:   true,
		IncludeDocument: true,
		IncludeActors:   true,
	})
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerTrainingRecord, error) {
	return s.repo.GetByID(ctx, &repositories.GetWorkerTrainingByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		IncludeCourse:   true,
		IncludeWorker:   true,
		IncludeDocument: true,
	})
}

// RefreshRollup recomputes the worker's training standing and caches it on the
// profile, which is what lets the roster filter and sort on training without
// replaying every record for every row. Best-effort by design: a cache that
// fails to update must not fail the write that triggered it, and the nightly
// sweep re-checks every worker anyway.
func (s *Service) RefreshRollup(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.WorkerTrainingSummary, error) {
	summary, err := s.Summary(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	if err = s.workerRepo.UpdateProfileTrainingRollup(
		ctx,
		&repositories.UpdateProfileTrainingRollupRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
			Health:     summary.WorstHealth(),
			NextDue:    summary.NextDue(),
		},
	); err != nil {
		return nil, err
	}
	return summary, nil
}

func (s *Service) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.WorkerTrainingSummary, error) {
	wrk, err := s.loadWorker(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	return s.summarize(ctx, tenantInfo, wrk)
}

func (s *Service) summarize(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	wrk *worker.Worker,
) (*worker.WorkerTrainingSummary, error) {
	courses, err := s.repo.ListActiveCourses(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	records, err := s.repo.ListForWorker(ctx, &repositories.ListWorkerTrainingRequest{
		TenantInfo:      tenantInfo,
		WorkerID:        wrk.ID,
		IncludeClosed:   true,
		IncludeCourse:   true,
		IncludeDocument: true,
	})
	if err != nil {
		return nil, err
	}
	return worker.BuildTrainingSummary(wrk, courses, records, timeutils.NowUnix()), nil
}

type AssignRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	CourseID   pulid.ID
	DueAt      *int64
	Notes      string
	UserID     pulid.ID
}

// Assign opens a course for a worker. The due date defaults from the course;
// a second open assignment of the same course is refused by the database.
func (s *Service) Assign(
	ctx context.Context,
	req *AssignRequest,
) (*worker.WorkerTrainingRecord, error) {
	log := s.l.With(
		zap.String("operation", "Assign"),
		zap.String("workerId", req.WorkerID.String()),
		zap.String("courseId", req.CourseID.String()),
	)

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

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to assign training", zap.Error(err))
		return nil, err
	}
	created.Course = course

	s.auditRecord(created, nil, permission.OpAssign, req.UserID, "Training assigned: "+course.Name, log)
	s.publish(ctx, req.TenantInfo, realtimeResource, permission.OpAssign, created.ID, req.UserID)
	s.refreshRollupQuietly(ctx, req.TenantInfo, created.WorkerID)

	return created, nil
}

const secondsPerDay = int64(86400)

// AssignRequired opens every required course the worker is missing, has
// failed, or has let lapse. It is what a hire, a rehire and a driver-type
// change call, so it never fails on a course that is already open.
func (s *Service) AssignRequired(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	userID pulid.ID,
) ([]*worker.WorkerTrainingRecord, error) {
	log := s.l.With(zap.String("operation", "AssignRequired"), zap.String("workerId", workerID.String()))

	wrk, err := s.loadWorker(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	summary, err := s.summarize(ctx, tenantInfo, wrk)
	if err != nil {
		return nil, err
	}

	gaps := summary.RequiredGaps()
	created := make([]*worker.WorkerTrainingRecord, 0, len(gaps))
	for _, course := range gaps {
		record, assignErr := s.Assign(ctx, &AssignRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
			CourseID:   course.ID,
			UserID:     userID,
		})
		if assignErr != nil {
			log.Warn("failed to assign required course",
				zap.String("courseId", course.ID.String()),
				zap.Error(assignErr))
			continue
		}
		created = append(created, record)
	}
	return created, nil
}

type CompleteRequest struct {
	TenantInfo pagination.TenantInfo
	// ID names an open record; when nil the completion is filed directly
	// against WorkerID + CourseID (a classroom session recorded after the fact).
	ID          pulid.ID
	WorkerID    pulid.ID
	CourseID    pulid.ID
	CompletedAt int64
	Score       decimal.NullDecimal
	DocumentID  pulid.ID
	Notes       string
	Version     int64
	UserID      pulid.ID
}

// Complete records the result of a course. A scored course needs a score and
// fails below the passing mark; a pass sets the expiry for recurring courses.
func (s *Service) Complete(
	ctx context.Context,
	req *CompleteRequest,
) (*worker.WorkerTrainingRecord, error) {
	log := s.l.With(zap.String("operation", "Complete"))

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

	var saved *worker.WorkerTrainingRecord
	if original == nil {
		saved, err = s.repo.Create(ctx, record)
	} else {
		saved, err = s.repo.Update(ctx, record)
	}
	if err != nil {
		log.Error("failed to record training completion", zap.Error(err))
		return nil, err
	}
	saved.Course = course

	comment := "Training completed: " + course.Name
	if !passed {
		comment = "Training failed: " + course.Name
	}
	s.auditRecord(saved, original, permission.OpUpdate, req.UserID, comment, log)
	s.publish(ctx, req.TenantInfo, realtimeResource, permission.OpUpdate, saved.ID, req.UserID)
	s.refreshRollupQuietly(ctx, req.TenantInfo, saved.WorkerID)

	return saved, nil
}

// openRecordFor resolves the record a completion applies to: the named open
// record, or a fresh record for the worker and course when none is open.
func (s *Service) openRecordFor(
	ctx context.Context,
	req *CompleteRequest,
) (record, original *worker.WorkerTrainingRecord, err error) {
	if !req.ID.IsNil() {
		original, err = s.loadRecord(ctx, req.TenantInfo, req.ID, req.Version)
		if err != nil {
			return nil, nil, err
		}
		if !original.IsOpen() {
			return nil, nil, errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"Only an assigned or in-progress course can be completed. Assign it again to record a renewal",
			)
		}
		copied := *original
		return &copied, original, nil
	}

	if req.WorkerID.IsNil() || req.CourseID.IsNil() {
		return nil, nil, errortypes.NewValidationError(
			"courseId",
			errortypes.ErrRequired,
			"Choose the course that was completed",
		)
	}
	course, err := s.repo.GetCourseByID(ctx, &repositories.GetTrainingCourseByIDRequest{
		ID:         req.CourseID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	if _, err = s.loadWorker(ctx, req.TenantInfo, req.WorkerID); err != nil {
		return nil, nil, err
	}
	open, err := s.repo.ListForWorker(ctx, &repositories.ListWorkerTrainingRequest{
		TenantInfo: req.TenantInfo,
		WorkerID:   req.WorkerID,
	})
	if err != nil {
		return nil, nil, err
	}
	for _, candidate := range open {
		if candidate.CourseID == req.CourseID {
			candidate.Course = course
			copied := *candidate
			return &copied, candidate, nil
		}
	}

	now := timeutils.NowUnix()
	return &worker.WorkerTrainingRecord{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		WorkerID:       req.WorkerID,
		CourseID:       req.CourseID,
		AssignedAt:     now,
		AssignedByID:   req.UserID,
		Course:         course,
	}, nil, nil
}

func (s *Service) loadRecord(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	version int64,
) (*worker.WorkerTrainingRecord, error) {
	original, err := s.repo.GetByID(ctx, &repositories.GetWorkerTrainingByIDRequest{
		ID:            id,
		TenantInfo:    tenantInfo,
		IncludeCourse: true,
	})
	if err != nil {
		return nil, err
	}
	if version > 0 && original.Version != version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Training record was changed by someone else. Reload and try again",
		)
	}
	if original.Course == nil {
		return nil, errortypes.NewNotFoundError("Course not found")
	}
	return original, nil
}

func (s *Service) requireWorkerDocument(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	record *worker.WorkerTrainingRecord,
	documentID pulid.ID,
) error {
	doc, err := s.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	ownedByRecord := doc.ResourceType == trainingResourceType && doc.ResourceID == record.ID.String()
	ownedByWorker := doc.ResourceType == workerResourceType && doc.ResourceID == record.WorkerID.String()
	if !ownedByRecord && !ownedByWorker {
		return errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document does not belong to this worker",
		)
	}
	return nil
}

type StatusRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Reason     string
	Version    int64
	UserID     pulid.ID
}

// Waive closes an open assignment as satisfied without a completion — prior
// experience, an equivalent certificate — and keeps the reason on the record.
func (s *Service) Waive(
	ctx context.Context,
	req *StatusRequest,
) (*worker.WorkerTrainingRecord, error) {
	log := s.l.With(zap.String("operation", "Waive"), zap.String("id", req.ID.String()))

	original, err := s.loadRecord(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if !original.IsOpen() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only an assigned or in-progress course can be waived",
		)
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
	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		log.Error("failed to waive training", zap.Error(err))
		return nil, err
	}
	saved.Course = original.Course

	s.auditRecord(saved, original, permission.OpCancel, req.UserID, "Training waived: "+reason, log)
	s.publish(ctx, req.TenantInfo, realtimeResource, permission.OpCancel, saved.ID, req.UserID)
	s.refreshRollupQuietly(ctx, req.TenantInfo, saved.WorkerID)

	return saved, nil
}

// Cancel withdraws an open assignment that should never have been made.
func (s *Service) Cancel(
	ctx context.Context,
	req *StatusRequest,
) (*worker.WorkerTrainingRecord, error) {
	log := s.l.With(zap.String("operation", "Cancel"), zap.String("id", req.ID.String()))

	original, err := s.loadRecord(ctx, req.TenantInfo, req.ID, req.Version)
	if err != nil {
		return nil, err
	}
	if !original.IsOpen() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only an assigned or in-progress course can be cancelled",
		)
	}

	updated := *original
	updated.Status = worker.TrainingStatusCancelled
	updated.RecordedByID = req.UserID
	if reason := strings.TrimSpace(req.Reason); reason != "" {
		updated.Notes = reason
	}
	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		log.Error("failed to cancel training", zap.Error(err))
		return nil, err
	}
	saved.Course = original.Course

	s.auditRecord(saved, original, permission.OpCancel, req.UserID, "Training assignment cancelled", log)
	s.publish(ctx, req.TenantInfo, realtimeResource, permission.OpCancel, saved.ID, req.UserID)
	s.refreshRollupQuietly(ctx, req.TenantInfo, saved.WorkerID)

	return saved, nil
}

type AttachDocumentRequest struct {
	ID         pulid.ID
	DocumentID pulid.ID
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

// AttachDocument files a certificate against a record.
func (s *Service) AttachDocument(
	ctx context.Context,
	req *AttachDocumentRequest,
) (*worker.WorkerTrainingRecord, error) {
	log := s.l.With(zap.String("operation", "AttachDocument"), zap.String("id", req.ID.String()))

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
	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		log.Error("failed to attach training document", zap.Error(err))
		return nil, err
	}
	saved.Course = original.Course

	s.auditRecord(saved, original, permission.OpUpdate, req.UserID, "Certificate attached", log)
	s.publish(ctx, req.TenantInfo, realtimeResource, permission.OpUpdate, saved.ID, req.UserID)
	s.refreshRollupQuietly(ctx, req.TenantInfo, saved.WorkerID)

	return saved, nil
}

// Start marks an assignment in progress the first time the driver opens it.
// It is idempotent so reopening a course never errors.
func (s *Service) Start(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	workerID pulid.ID,
) (*worker.WorkerTrainingRecord, error) {
	original, err := s.ownRecord(ctx, tenantInfo, id, workerID)
	if err != nil {
		return nil, err
	}
	if original.Status != worker.TrainingStatusAssigned {
		return original, nil
	}

	now := timeutils.NowUnix()
	updated := *original
	updated.Status = worker.TrainingStatusInProgress
	updated.StartedAt = &now
	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		return nil, err
	}
	saved.Course = original.Course
	s.publish(ctx, tenantInfo, realtimeResource, permission.OpUpdate, saved.ID, pulid.Nil)
	s.refreshRollupQuietly(ctx, tenantInfo, saved.WorkerID)
	return saved, nil
}

// Acknowledge is the driver saying they took the course. Self-serve courses
// without a passing score complete on the spot; anything scored or delivered
// in person stays in progress until the office records the result.
func (s *Service) Acknowledge(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	workerID pulid.ID,
) (*worker.WorkerTrainingRecord, error) {
	log := s.l.With(zap.String("operation", "Acknowledge"), zap.String("id", id.String()))

	original, err := s.ownRecord(ctx, tenantInfo, id, workerID)
	if err != nil {
		return nil, err
	}
	if !original.IsOpen() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"This course is no longer open",
		)
	}
	if original.IsAcknowledged() {
		return original, nil
	}

	now := timeutils.NowUnix()
	updated := *original
	updated.AcknowledgedAt = &now
	if updated.StartedAt == nil {
		updated.StartedAt = &now
	}
	course := original.Course
	if course.Delivery.SelfServe() && !course.PassingScore.Valid {
		passed := true
		updated.Status = worker.TrainingStatusCompleted
		updated.CompletedAt = &now
		updated.Passed = &passed
		updated.ExpiresAt = course.ExpiryFor(now)
	} else {
		updated.Status = worker.TrainingStatusInProgress
	}

	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		log.Error("failed to acknowledge training", zap.Error(err))
		return nil, err
	}
	saved.Course = course
	s.publish(ctx, tenantInfo, realtimeResource, permission.OpUpdate, saved.ID, pulid.Nil)
	s.refreshRollupQuietly(ctx, tenantInfo, saved.WorkerID)
	return saved, nil
}

func (s *Service) ownRecord(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	workerID pulid.ID,
) (*worker.WorkerTrainingRecord, error) {
	original, err := s.loadRecord(ctx, tenantInfo, id, 0)
	if err != nil {
		return nil, err
	}
	if original.WorkerID != workerID {
		return nil, errortypes.NewNotFoundError("Training record not found")
	}
	return original, nil
}

// MarkExpired flips a lapsed completion so the open-record rule lets a renewal
// be assigned. The sweep calls it; it is a no-op when nothing changed.
func (s *Service) MarkExpired(
	ctx context.Context,
	record *worker.WorkerTrainingRecord,
) (*worker.WorkerTrainingRecord, error) {
	if record == nil || record.Status != worker.TrainingStatusCompleted ||
		record.ExpiresAt == nil || *record.ExpiresAt > timeutils.NowUnix() {
		return record, nil
	}
	updated := *record
	updated.Status = worker.TrainingStatusExpired
	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		return nil, err
	}
	saved.Course = record.Course
	saved.Worker = record.Worker
	s.publish(ctx, recordTenant(saved), realtimeResource, permission.OpUpdate, saved.ID, pulid.Nil)
	s.refreshRollupQuietly(ctx, recordTenant(saved), saved.WorkerID)
	return saved, nil
}
