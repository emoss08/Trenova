package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

func ResolveBatchBounds(
	control *tenant.SettlementControl,
	periodStart, periodEnd, now int64,
) (PeriodBounds, error) {
	return settlementshared.ResolveRequestedPeriod(settlementshared.PeriodSchedule{
		Frequency:    control.PayPeriodFrequency,
		EndDayOfWeek: control.PeriodEndDayOfWeek,
		PayDelayDays: control.PayDelayDays,
	}, periodStart, periodEnd, now)
}

func (s *Service) CurrentPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (PeriodBounds, error) {
	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return PeriodBounds{}, err
	}
	return ResolveCurrentPeriod(control, timeutils.NowUnix()), nil
}

func (s *Service) openBatchForPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	bounds PeriodBounds,
) (*driversettlement.SettlementBatch, error) {
	existing, err := s.batchRepo.GetForPeriod(
		ctx,
		tenantInfo,
		bounds.PeriodStart,
		bounds.PeriodEnd,
	)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status != driversettlement.BatchStatusOpen {
		return nil, errortypes.NewValidationError(
			"periodEnd",
			errortypes.ErrInvalidOperation,
			"The batch for this pay period is already completed; generate individual settlements for late accruals instead",
		)
	}
	return existing, nil
}

func (s *Service) periodAlreadySettled(
	ctx context.Context,
	req *GenerateForWorkerRequest,
) (bool, error) {
	if len(req.PayEventIDs) > 0 {
		return false, nil
	}
	return s.settlementRepo.ExistsForWorkerPeriod(
		ctx,
		req.TenantInfo,
		req.WorkerID,
		req.PeriodStart,
		req.PeriodEnd,
	)
}

type GenerationPlan struct {
	Settlement     *driversettlement.Settlement
	AlreadySettled bool
	AutoApprove    bool
	Refusal        error
}

func (s *Service) PlanGenerateForWorker(
	ctx context.Context,
	req *GenerateForWorkerRequest,
) (*GenerationPlan, error) {
	plan := &GenerationPlan{}
	if req.BatchID != nil && !req.BatchID.IsNil() {
		batch, err := s.batchRepo.GetByID(ctx, repositories.GetSettlementBatchByIDRequest{
			ID:         *req.BatchID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			if settlementshared.IsRefusal(err) {
				plan.Refusal = err
				return plan, nil
			}
			return nil, err
		}
		if batch.Status != driversettlement.BatchStatusOpen {
			plan.Refusal = errBatchNotOpen()
			return plan, nil
		}
	}

	exists, err := s.periodAlreadySettled(ctx, req)
	if err != nil {
		return nil, err
	}
	if exists {
		plan.AlreadySettled = true
		return plan, nil
	}

	control, err := s.settlementControl.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	settlement, _, err := s.buildSettlement(ctx, req, control)
	if err != nil {
		if settlementshared.IsRefusal(err) {
			plan.Refusal = err
			return plan, nil
		}
		return nil, err
	}
	plan.Settlement = settlement
	plan.AutoApprove = settlement != nil && control.AutoApproveClean && !settlement.HasExceptions
	return plan, nil
}

type BatchPlan struct {
	Bounds        PeriodBounds
	ExistingBatch *driversettlement.SettlementBatch
	Workers       []*repositories.UnsettledWorkerSummary
	AutoApprove   bool
	Refusal       error
}

func (p *BatchPlan) Refused() bool {
	return p.Refusal != nil
}

func (s *Service) PlanBatch(ctx context.Context, req *GenerateBatchRequest) (*BatchPlan, error) {
	control, err := s.settlementControl.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	plan := &BatchPlan{AutoApprove: control.AutoApproveClean}
	plan.Bounds, plan.Refusal = ResolveBatchBounds(
		control,
		req.PeriodStart,
		req.PeriodEnd,
		timeutils.NowUnix(),
	)
	if plan.Refused() {
		return plan, nil
	}

	plan.ExistingBatch, err = s.openBatchForPeriod(ctx, req.TenantInfo, plan.Bounds)
	if err != nil {
		if settlementshared.IsRefusal(err) {
			plan.Refusal = err
			return plan, nil
		}
		return nil, err
	}

	plan.Workers, err = s.payEventRepo.ListUnsettledWorkerSummaries(
		ctx,
		&repositories.ListUnsettledWorkerSummariesRequest{
			TenantInfo:  req.TenantInfo,
			PeriodStart: plan.Bounds.PeriodStart,
			PeriodEnd:   plan.Bounds.PeriodEnd,
		},
	)
	if err != nil {
		return nil, err
	}
	return plan, nil
}

func errBatchNotOpen() error {
	return errortypes.NewValidationError(
		"batchId",
		errortypes.ErrInvalidOperation,
		"Settlements can only be added to an open batch",
	)
}
