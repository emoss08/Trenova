package workerdrugalcoholservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) ListViolations(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) ([]*worker.WorkerDOTViolation, error) {
	return s.repo.ListViolations(ctx, &repositories.ListWorkerDOTViolationsRequest{
		TenantInfo:   tenantInfo,
		WorkerID:     workerID,
		IncludeTests: true,
	})
}

func (s *Service) GetOpenViolation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.WorkerDOTViolation, error) {
	return s.repo.GetOpenViolation(ctx, tenantInfo, workerID)
}

// RecordViolation files a prohibition the office learned about some other way:
// actual knowledge of use, or a violation reported by a previous employer.
// Results that speak for themselves open their own violation through
// applyTestOutcome.
func (s *Service) RecordViolation(
	ctx context.Context,
	entity *worker.WorkerDOTViolation,
	userID pulid.ID,
) (*worker.WorkerDOTViolation, error) {
	tenantInfo := violationTenant(entity)

	if _, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         entity.WorkerID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetOpenViolation(ctx, tenantInfo, entity.WorkerID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrDuplicate,
			"This worker already has an unresolved violation; work through that one first",
		)
	}

	entity.RecordedByID = userID
	entity.Notes = strings.TrimSpace(entity.Notes)
	entity.SAPName = strings.TrimSpace(entity.SAPName)
	if entity.OccurredAt <= 0 {
		entity.OccurredAt = timeutils.NowUnix()
	}
	entity.ApplyDerivedStatus(timeutils.NowUnix())

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreateViolation(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceWorkerDOTTest,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    created,
		comment:    "Recorded a " + created.ViolationType.Label() + " violation",
	})
	s.publish(ctx, tenantInfo, realtimeViolation, permission.OpCreate, created.ID, userID)
	s.refreshRollupQuietly(ctx, tenantInfo, created.WorkerID)

	return created, nil
}

// UpdateViolationRequest carries a step of the return-to-duty process. Every
// field is optional: the office records each step as it happens, and the stage
// is derived from what has been filled in rather than set by hand.
type UpdateViolationRequest struct {
	TenantInfo                pagination.TenantInfo
	ViolationID               pulid.ID
	SAPName                   *string
	SAPReferredAt             *int64
	SAPEvaluationCompletedAt  *int64
	FollowUpTestCount         *int32
	FollowUpEndsAt            *int64
	ReportedToClearinghouseAt *int64
	DocumentID                pulid.ID
	Notes                     *string
	UserID                    pulid.ID
}

func (s *Service) UpdateViolation(
	ctx context.Context,
	req *UpdateViolationRequest,
) (*worker.WorkerDOTViolation, error) {
	entity, err := s.repo.GetViolationByID(ctx, &repositories.GetWorkerDOTViolationByIDRequest{
		ID:         req.ViolationID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	previous := *entity

	if req.SAPName != nil {
		entity.SAPName = strings.TrimSpace(*req.SAPName)
	}
	if req.SAPReferredAt != nil {
		entity.SAPReferredAt = req.SAPReferredAt
	}
	if req.SAPEvaluationCompletedAt != nil {
		entity.SAPEvaluationCompletedAt = req.SAPEvaluationCompletedAt
	}
	if req.FollowUpTestCount != nil {
		entity.FollowUpTestCount = *req.FollowUpTestCount
	}
	if req.FollowUpEndsAt != nil {
		entity.FollowUpEndsAt = req.FollowUpEndsAt
	}
	if req.ReportedToClearinghouseAt != nil {
		entity.ReportedToClearinghouseAt = req.ReportedToClearinghouseAt
	}
	if req.Notes != nil {
		entity.Notes = strings.TrimSpace(*req.Notes)
	}
	if !req.DocumentID.IsNil() {
		entity.DocumentID = req.DocumentID
	}

	entity.ApplyDerivedStatus(timeutils.NowUnix())

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.repo.UpdateViolation(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceWorkerDOTTest,
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Return-to-duty stage is now " + updated.Status.Label(),
	})
	s.publish(ctx, req.TenantInfo, realtimeViolation, permission.OpUpdate, updated.ID, req.UserID)
	s.refreshRollupQuietly(ctx, req.TenantInfo, updated.WorkerID)

	return updated, nil
}

// openViolationFor turns a violating result into the prohibition it is. A
// worker already in the return-to-duty process does not get a second row: the
// unique index would refuse it, and a second violation during the process is
// recorded against the one that is open.
func (s *Service) openViolationFor(
	ctx context.Context,
	test *worker.WorkerDOTTest,
	userID pulid.ID,
	log *zap.Logger,
) error {
	tenantInfo := testTenant(test)

	existing, err := s.repo.GetOpenViolation(ctx, tenantInfo, test.WorkerID)
	if err != nil {
		return err
	}
	if existing != nil {
		log.Info("violating result recorded against an open violation",
			zap.String("violationId", existing.ID.String()))
		return nil
	}

	violation := &worker.WorkerDOTViolation{
		OrganizationID: test.OrganizationID,
		BusinessUnitID: test.BusinessUnitID,
		WorkerID:       test.WorkerID,
		ViolationType:  test.ViolationTypeFor(),
		Status:         worker.DOTViolationStatusOpen,
		OccurredAt:     test.EffectiveAt(),
		SourceTestID:   test.ID,
		RecordedByID:   userID,
		Notes: "Opened automatically from a " + string(test.Result) + " " +
			test.TestType.Label() + " test",
	}

	created, err := s.repo.CreateViolation(ctx, violation)
	if err != nil {
		return err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceWorkerDOTTest,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    created,
		comment:    "Opened from a violating test result",
	})
	s.publish(ctx, tenantInfo, realtimeViolation, permission.OpCreate, created.ID, userID)

	return nil
}

// applyReturnToDuty records that the driver passed the test that lets them work
// again. The order is the regulation's, not ours — a passed test taken before
// the SAP evaluation releases nobody — but that is checked before the test is
// written, so by here it holds.
func (s *Service) applyReturnToDuty(
	ctx context.Context,
	violation *worker.WorkerDOTViolation,
	test *worker.WorkerDOTTest,
	userID pulid.ID,
) error {
	if violation.SAPEvaluationCompletedAt == nil || violation.RTDCompletedAt != nil {
		return nil
	}

	previous := *violation
	completedAt := test.EffectiveAt()
	violation.RTDCompletedAt = &completedAt
	violation.RTDTestID = test.ID
	if violation.FollowUpTestCount == 0 {
		violation.FollowUpTestCount = worker.MinimumFollowUpTests
	}
	violation.ApplyDerivedStatus(timeutils.NowUnix())

	updated, err := s.repo.UpdateViolation(ctx, violation)
	if err != nil {
		return err
	}

	tenantInfo := violationTenant(updated)
	s.audit(&auditParams{
		resource:   permission.ResourceWorkerDOTTest,
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Return-to-duty test passed",
	})
	s.publish(ctx, tenantInfo, realtimeViolation, permission.OpUpdate, updated.ID, userID)

	return nil
}

// applyFollowUp counts a passed follow-up test against the programme the SAP
// set. When the last one lands the violation resolves itself, so nobody has to
// remember to close it.
func (s *Service) applyFollowUp(
	ctx context.Context,
	violation *worker.WorkerDOTViolation,
	userID pulid.ID,
) error {
	if violation.RTDCompletedAt == nil ||
		violation.FollowUpTestsCompleted >= violation.FollowUpTestCount {
		return nil
	}

	previous := *violation
	violation.FollowUpTestsCompleted++
	violation.ApplyDerivedStatus(timeutils.NowUnix())

	updated, err := s.repo.UpdateViolation(ctx, violation)
	if err != nil {
		return err
	}

	tenantInfo := violationTenant(updated)
	s.audit(&auditParams{
		resource:   permission.ResourceWorkerDOTTest,
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Follow-up test recorded",
	})
	s.publish(ctx, tenantInfo, realtimeViolation, permission.OpUpdate, updated.ID, userID)

	return nil
}
