package workerdrugalcoholservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type (
	TestChange  = kernelports.RecordChange[workerdot.WorkerDOTTest]
	DrawChange  = kernelports.RecordChange[workerdot.DOTRandomDraw]
	EntryChange = kernelports.RecordChange[workerdot.DOTRandomDrawEntry]
)

func (s *Service) PlanRecordTest(
	ctx context.Context,
	entity *workerdot.WorkerDOTTest,
	userID pulid.ID,
) (*workerdot.WorkerDOTTest, error) {
	planned := *entity
	if err := s.prepareRecordTest(ctx, &planned, userID); err != nil {
		return nil, err
	}
	return &planned, nil
}

func (s *Service) PlanCancelTest(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	reason string,
) (*TestChange, error) {
	original, err := s.repo.GetTestByID(ctx, &workerports.GetWorkerDOTTestByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !original.Status.CanTransitionTo(workerdot.DOTTestStatusCancelled) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A {0} test cannot be cancelled", strings.ToLower(string(original.Status)),
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

	entity := *original
	entity.Status = workerdot.DOTTestStatusCancelled
	entity.Result = workerdot.DOTResultCancelled
	entity.Notes = strings.TrimSpace(entity.Notes + "\nCancelled: " + reason)
	return &TestChange{Before: original, After: &entity}, nil
}

func (s *Service) PlanRunDraw(
	ctx context.Context,
	req *RunDrawRequest,
) (*workerdot.DOTRandomDraw, error) {
	plan, err := s.planDraw(ctx, req)
	if err != nil {
		return nil, err
	}
	draw := plan.draw
	draw.Pool = plan.pool
	draw.DrugSelected = min(draw.DrugTarget, draw.PoolSize)
	draw.AlcoholSelected = min(draw.AlcoholTarget, draw.PoolSize)
	return draw, nil
}

func (s *Service) PlanFinalizeDraw(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*DrawChange, error) {
	original, err := s.repo.GetDrawByID(ctx, &workerports.GetDOTRandomDrawByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.Status != workerdot.RandomDrawStatusDraft {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Only a draft round can be finalised",
		)
	}

	entity := *original
	finalizedAt := timeutils.NowUnix()
	entity.Status = workerdot.RandomDrawStatusFinal
	entity.FinalizedAt = &finalizedAt
	return &DrawChange{Before: original, After: &entity}, nil
}

func (s *Service) PlanCancelDraw(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	reason string,
) (*DrawChange, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Cancelling a round needs a reason on the record",
		)
	}

	original, err := s.repo.GetDrawByID(ctx, &workerports.GetDOTRandomDrawByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	entity := *original
	if original.Status == workerdot.RandomDrawStatusCancelled {
		return &DrawChange{Before: original, After: &entity}, nil
	}

	entity.Status = workerdot.RandomDrawStatusCancelled
	entity.FinalizedAt = nil
	entity.Notes = strings.TrimSpace(entity.Notes + "\nCancelled: " + reason)
	return &DrawChange{Before: original, After: &entity}, nil
}

func (s *Service) PlanUpdateDrawEntry(
	ctx context.Context,
	req *UpdateEntryRequest,
) (*EntryChange, error) {
	original, err := s.repo.GetDrawEntryByID(ctx, &workerports.GetDOTRandomDrawEntryByIDRequest{
		ID:         req.EntryID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if req.Status == workerdot.RandomEntryCompleted {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A selection completes when its test is recorded, not on its own",
		)
	}

	entity := *original
	entity.Status = req.Status
	entity.ExcuseReason = strings.TrimSpace(req.ExcuseReason)
	if req.Status == workerdot.RandomEntryNotified && entity.NotifiedAt == nil {
		notifiedAt := timeutils.NowUnix()
		entity.NotifiedAt = &notifiedAt
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return &EntryChange{Before: original, After: &entity}, nil
}
