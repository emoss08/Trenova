package carriersettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/shared/timeutils"
)

func ResolveBatchBounds(
	control *tenant.CarrierSettlementControl,
	periodStart, periodEnd, now int64,
) (PeriodBounds, error) {
	return settlementshared.ResolveRequestedPeriod(settlementshared.PeriodSchedule{
		Frequency:    control.PayPeriodFrequency,
		EndDayOfWeek: control.PeriodEndDayOfWeek,
		PayDelayDays: control.PayDelayDays,
	}, periodStart, periodEnd, now)
}

type BatchPlanRequest struct {
	Batch *GenerateBatchRequest
	Limit int
}

type BatchPlan struct {
	Bounds        PeriodBounds
	ExistingBatch *carriersettlement.CarrierSettlementBatch
	Settlements   []*carriersettlement.CarrierSettlement
	CarrierCount  int
	SettledCount  int
	Refusal       error
}

func (p *BatchPlan) Refused() bool {
	return p.Refusal != nil
}

func (s *Service) PlanBatch(ctx context.Context, req *BatchPlanRequest) (*BatchPlan, error) {
	batchReq := req.Batch
	control, err := s.settlementControl.GetOrCreate(ctx, batchReq.TenantInfo)
	if err != nil {
		return nil, err
	}

	plan := &BatchPlan{}
	plan.Bounds, plan.Refusal = ResolveBatchBounds(
		control,
		batchReq.PeriodStart,
		batchReq.PeriodEnd,
		timeutils.NowUnix(),
	)
	if plan.Refused() {
		return plan, nil
	}

	plan.ExistingBatch, err = s.batchRepo.GetForPeriod(
		ctx,
		batchReq.TenantInfo,
		plan.Bounds.PeriodStart,
		plan.Bounds.PeriodEnd,
	)
	if err != nil {
		return nil, err
	}

	carrierIDs, err := s.costEventRepo.ListCarrierIDsWithPendingEvents(
		ctx,
		repositories.ListCarriersWithPendingEventsRequest{
			TenantInfo: batchReq.TenantInfo,
			PeriodEnd:  plan.Bounds.PeriodEnd,
		},
	)
	if err != nil {
		return nil, err
	}
	plan.CarrierCount = len(carrierIDs)

	plan.Settlements = make(
		[]*carriersettlement.CarrierSettlement,
		0,
		min(len(carrierIDs), req.Limit),
	)
	for _, carrierID := range carrierIDs {
		if len(plan.Settlements) >= req.Limit {
			break
		}
		exists, existsErr := s.settlementRepo.ExistsForCarrierPeriod(
			ctx,
			batchReq.TenantInfo,
			carrierID,
			plan.Bounds.PeriodStart,
			plan.Bounds.PeriodEnd,
		)
		if existsErr != nil {
			return nil, existsErr
		}
		if exists {
			plan.SettledCount++
			continue
		}
		settlement, _, buildErr := s.buildSettlement(ctx, &GenerateForCarrierRequest{
			TenantInfo:  batchReq.TenantInfo,
			CarrierID:   carrierID,
			PeriodStart: plan.Bounds.PeriodStart,
			PeriodEnd:   plan.Bounds.PeriodEnd,
			PayDate:     plan.Bounds.PayDate,
		})
		if buildErr != nil {
			return nil, buildErr
		}
		if settlement != nil {
			plan.Settlements = append(plan.Settlements, settlement)
		}
	}
	return plan, nil
}
