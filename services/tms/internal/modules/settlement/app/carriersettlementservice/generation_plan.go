package carriersettlementservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/carriersettlementcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

func ResolveBatchBounds(
	control *carriersettlementcontrol.CarrierSettlementControl,
	loc *time.Location,
	periodStart, periodEnd, now int64,
) (PeriodBounds, error) {
	return settlementshared.ResolveRequestedPeriod(settlementshared.PeriodSchedule{
		Frequency:    control.PayPeriodFrequency,
		EndDayOfWeek: control.PeriodEndDayOfWeek,
		PayDelayDays: control.PayDelayDays,
		Location:     loc,
	}, periodStart, periodEnd, now)
}

func (s *Service) Location(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*time.Location, error) {
	return tenantports.OrganizationLocation(ctx, s.organizations, tenantInfo)
}

func (s *Service) CurrentPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (PeriodBounds, error) {
	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return PeriodBounds{}, err
	}
	loc, err := s.Location(ctx, tenantInfo)
	if err != nil {
		return PeriodBounds{}, err
	}
	return ResolveCurrentPeriod(control, loc, timeutils.NowUnix()), nil
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

	loc, err := s.Location(ctx, batchReq.TenantInfo)
	if err != nil {
		return nil, err
	}

	plan := &BatchPlan{}
	plan.Bounds, plan.Refusal = ResolveBatchBounds(
		control,
		loc,
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
		carriersettlementports.ListCarriersWithPendingEventsRequest{
			TenantInfo:              batchReq.TenantInfo,
			PeriodEnd:               plan.Bounds.PeriodEnd,
			HoldUntilInvoiceMatched: control.HoldUntilInvoiceMatched,
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
