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
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (s *Service) ListTests(
	ctx context.Context,
	req *repositories.ListWorkerDOTTestsRequest,
) ([]*worker.WorkerDOTTest, error) {
	return s.repo.ListTests(ctx, req)
}

func (s *Service) GetTest(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerDOTTest, error) {
	return s.repo.GetTestByID(ctx, &repositories.GetWorkerDOTTestByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		IncludeWorker:   true,
		IncludeDocument: true,
		IncludeActors:   true,
	})
}

func (s *Service) prepareTest(entity *worker.WorkerDOTTest) error {
	// The columns default to Scheduled and Pending, but validation runs before
	// the insert does, so the defaults have to be applied here too.
	if entity.Status == "" {
		entity.Status = worker.DOTTestStatusScheduled
	}
	if entity.Result == "" {
		entity.Result = worker.DOTResultPending
	}
	entity.Reason = strings.TrimSpace(entity.Reason)
	entity.CollectionSite = strings.TrimSpace(entity.CollectionSite)
	entity.CollectorName = strings.TrimSpace(entity.CollectorName)
	entity.SpecimenID = strings.TrimSpace(entity.SpecimenID)
	entity.LabName = strings.TrimSpace(entity.LabName)
	entity.MROName = strings.TrimSpace(entity.MROName)
	entity.Notes = strings.TrimSpace(entity.Notes)

	// The two halves of a collection cannot borrow each other's fields, so
	// clear whatever the caller sent that this substance cannot produce rather
	// than storing a number the analysis never generated.
	if entity.Substance == worker.DOTSubstanceAlcohol {
		entity.MROName = ""
		entity.MROVerifiedAt = nil
	} else {
		entity.AlcoholConcentration = nil
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// RecordTest files a collection. A test can be recorded at any stage — the
// office often enters one that is already finished — so a result that arrives
// with it runs the same follow-ons an update would.
func (s *Service) RecordTest(
	ctx context.Context,
	entity *worker.WorkerDOTTest,
	userID pulid.ID,
) (*worker.WorkerDOTTest, error) {
	tenantInfo := testTenant(entity)
	log := s.l.With(
		zap.String("operation", "RecordTest"),
		zap.String("workerId", entity.WorkerID.String()),
	)

	if _, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         entity.WorkerID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return nil, err
	}

	entity.RecordedByID = userID
	if entity.OrderedByID.IsNil() {
		entity.OrderedByID = userID
	}
	if err := s.prepareTest(entity); err != nil {
		return nil, err
	}
	if entity.Status == worker.DOTTestStatusCompleted {
		if err := s.checkOutcomePreconditions(
			ctx, tenantInfo, entity.WorkerID, entity.TestType, entity.Result,
		); err != nil {
			return nil, err
		}
	}

	created, err := s.repo.CreateTest(ctx, entity)
	if err != nil {
		return nil, err
	}

	if err = s.applyTestOutcome(ctx, created, userID, log); err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceWorkerDOTTest,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    created,
		comment:    "Recorded a " + created.TestType.Label() + " test",
	})
	s.publish(ctx, tenantInfo, realtimeTest, permission.OpCreate, created.ID, userID)
	s.refreshRollupQuietly(ctx, tenantInfo, created.WorkerID)

	return created, nil
}

// RecordResultRequest carries what the laboratory, the medical review officer
// or the evidential breath testing device reported back.
type RecordResultRequest struct {
	TenantInfo           pagination.TenantInfo
	TestID               pulid.ID
	Result               worker.DOTTestResult
	ResultAt             int64
	LabName              string
	MROName              string
	MROVerifiedAt        *int64
	AlcoholConcentration *decimal.Decimal
	Notes                string
	DocumentID           pulid.ID
	UserID               pulid.ID
}

// RecordResult closes out a collection. An alcohol reading grades itself: the
// concentration decides the result, so a clerk cannot file a 0.11 as negative.
func (s *Service) RecordResult(
	ctx context.Context,
	req *RecordResultRequest,
) (*worker.WorkerDOTTest, error) {
	log := s.l.With(
		zap.String("operation", "RecordResult"),
		zap.String("testId", req.TestID.String()),
	)

	entity, err := s.repo.GetTestByID(ctx, &repositories.GetWorkerDOTTestByIDRequest{
		ID:         req.TestID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !entity.Status.CanTransitionTo(worker.DOTTestStatusCompleted) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"This test is already {0}; record a correction as a new test", strings.ToLower(string(entity.Status)),
		)
	}

	// The result an alcohol test lands on is decided by the concentration, not
	// by the caller, so work it out before asking whether it may be recorded.
	intended := req.Result
	if entity.Substance == worker.DOTSubstanceAlcohol {
		if req.AlcoholConcentration == nil {
			return nil, errortypes.NewValidationError(
				"alcoholConcentration",
				errortypes.ErrRequired,
				"An alcohol test result needs the concentration that was measured",
			)
		}
		intended = worker.ResultForConcentration(*req.AlcoholConcentration)
	}

	// Asked before a single field is touched: the entity here is the one the
	// repository holds, and a refused result must leave it exactly as it was.
	if err = s.checkOutcomePreconditions(
		ctx, req.TenantInfo, entity.WorkerID, entity.TestType, intended,
	); err != nil {
		return nil, err
	}

	previous := *entity

	entity.Status = worker.DOTTestStatusCompleted
	entity.Result = intended
	resultAt := req.ResultAt
	if resultAt <= 0 {
		resultAt = timeutils.NowUnix()
	}
	entity.ResultAt = &resultAt
	if entity.CollectedAt == nil || *entity.CollectedAt <= 0 {
		entity.CollectedAt = &resultAt
	}
	if req.LabName != "" {
		entity.LabName = req.LabName
	}
	if req.Notes != "" {
		entity.Notes = req.Notes
	}
	if !req.DocumentID.IsNil() {
		entity.DocumentID = req.DocumentID
	}

	if entity.Substance == worker.DOTSubstanceAlcohol {
		entity.AlcoholConcentration = req.AlcoholConcentration
	} else {
		entity.MROName = req.MROName
		entity.MROVerifiedAt = req.MROVerifiedAt
	}

	if err = s.prepareTest(entity); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateTest(ctx, entity)
	if err != nil {
		return nil, err
	}

	if err = s.applyTestOutcome(ctx, updated, req.UserID, log); err != nil {
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
		comment:    "Recorded a " + string(updated.Result) + " result",
	})
	s.publish(ctx, req.TenantInfo, realtimeTest, permission.OpUpdate, updated.ID, req.UserID)
	s.refreshRollupQuietly(ctx, req.TenantInfo, updated.WorkerID)

	return updated, nil
}

// CancelTest voids a collection that did not happen. The row stays on file:
// what was scheduled and why it was voided is part of the record an auditor
// reads.
func (s *Service) CancelTest(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	reason string,
	userID pulid.ID,
) (*worker.WorkerDOTTest, error) {
	entity, err := s.repo.GetTestByID(ctx, &repositories.GetWorkerDOTTestByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !entity.Status.CanTransitionTo(worker.DOTTestStatusCancelled) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A {0} test cannot be cancelled", strings.ToLower(string(entity.Status)),
		)
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Cancelling a collection needs a reason on the record",
		)
	}

	previous := *entity
	entity.Status = worker.DOTTestStatusCancelled
	entity.Result = worker.DOTResultCancelled
	entity.Notes = strings.TrimSpace(entity.Notes + "\nCancelled: " + reason)

	updated, err := s.repo.UpdateTest(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceWorkerDOTTest,
		resourceID: updated.ID.String(),
		operation:  permission.OpCancel,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    reason,
	})
	s.publish(ctx, tenantInfo, realtimeTest, permission.OpCancel, updated.ID, userID)
	s.refreshRollupQuietly(ctx, tenantInfo, updated.WorkerID)

	return updated, nil
}

// checkOutcomePreconditions refuses a result whose follow-on cannot legally be
// applied, before anything is written.
//
// It exists because the follow-ons run after the insert: without this, a
// return-to-duty test taken too early would be saved and *then* rejected,
// leaving the office with a completed test on file and an error on screen,
// and a retry that fails because the test is already completed.
func (s *Service) checkOutcomePreconditions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	testType worker.DOTTestType,
	result worker.DOTTestResult,
) error {
	if testType != worker.DOTTestReturnToDuty || !result.IsNegative() {
		return nil
	}

	violation, err := s.repo.GetOpenViolation(ctx, tenantInfo, workerID)
	if err != nil {
		return err
	}
	if violation == nil || violation.RTDCompletedAt != nil {
		return nil
	}
	if violation.SAPEvaluationCompletedAt == nil {
		return errortypes.NewValidationError(
			"testType",
			errortypes.ErrInvalid,
			"Record the SAP evaluation before a return-to-duty test (49 CFR 40.305)",
		)
	}

	return nil
}

// applyTestOutcome runs everything a finished test sets in motion: a violation
// opens, a random selection is closed out, a return-to-duty test releases the
// driver, and a follow-up test counts against the programme.
//
// Each of these is a consequence of the result, not a separate decision the
// office has to remember to make — which is the whole point of holding the
// testing programme in one place.
func (s *Service) applyTestOutcome(
	ctx context.Context,
	entity *worker.WorkerDOTTest,
	userID pulid.ID,
	log *zap.Logger,
) error {
	if entity.Status != worker.DOTTestStatusCompleted {
		return nil
	}

	tenantInfo := testTenant(entity)

	if err := s.closeDrawEntry(ctx, entity, log); err != nil {
		return err
	}

	if entity.IsViolation() {
		return s.openViolationFor(ctx, entity, userID, log)
	}

	if !entity.Result.IsNegative() {
		return nil
	}

	violation, err := s.repo.GetOpenViolation(ctx, tenantInfo, entity.WorkerID)
	if err != nil {
		return err
	}
	if violation == nil {
		return nil
	}

	switch entity.TestType {
	case worker.DOTTestReturnToDuty:
		return s.applyReturnToDuty(ctx, violation, entity, userID)
	case worker.DOTTestFollowUp:
		return s.applyFollowUp(ctx, violation, userID)
	default:
		return nil
	}
}

// closeDrawEntry marks the random selection this test answers as completed, so
// the round's outstanding list shrinks as collections come in.
func (s *Service) closeDrawEntry(
	ctx context.Context,
	entity *worker.WorkerDOTTest,
	log *zap.Logger,
) error {
	if entity.DrawEntryID.IsNil() {
		return nil
	}

	tenantInfo := testTenant(entity)
	dbEntry, err := s.repo.GetDrawEntryByID(ctx, &repositories.GetDOTRandomDrawEntryByIDRequest{
		ID:         entity.DrawEntryID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		log.Warn("failed to read draw entry for test", zap.Error(err))
		return nil
	}
	if dbEntry.Status == worker.RandomEntryCompleted {
		return nil
	}

	completedAt := entity.EffectiveAt()
	dbEntry.Status = worker.RandomEntryCompleted
	dbEntry.CompletedAt = &completedAt
	dbEntry.TestID = entity.ID

	if _, err = s.repo.UpdateDrawEntry(ctx, dbEntry); err != nil {
		log.Warn("failed to close draw entry", zap.Error(err))
	}

	return nil
}
