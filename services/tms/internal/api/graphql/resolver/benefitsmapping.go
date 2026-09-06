package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/services/benefitsservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type benefitPlanFields struct {
	code              string
	name              string
	description       *string
	planType          driverpay.BenefitPlanType
	carrier           *string
	policyNumber      *string
	payCodeID         string
	planYear          int
	employeeCostMinor int
	employerCostMinor int
	waitingPeriodDays *int
	status            *domaintypes.Status
}

func benefitPlanFromInput(
	f *benefitPlanFields,
	tenantInfo pagination.TenantInfo,
) (*driverpay.BenefitPlan, error) {
	payCodeID, err := pulid.MustParse(f.payCodeID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"payCodeId",
			errortypes.ErrInvalid,
			"Pay code is invalid",
		)
	}

	entity := &driverpay.BenefitPlan{
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		Code:              f.code,
		Name:              f.name,
		Description:       stringValue(f.description),
		PlanType:          f.planType,
		Carrier:           stringValue(f.carrier),
		PolicyNumber:      stringValue(f.policyNumber),
		PayCodeID:         payCodeID,
		PlanYear:          int16(f.planYear),                    //nolint:gosec // a calendar year
		EmployeeCostMinor: int64(f.employeeCostMinor),           //nolint:gosec // a minor-unit amount
		EmployerCostMinor: int64(f.employerCostMinor),           //nolint:gosec // a minor-unit amount
		WaitingPeriodDays: int32(intValue(f.waitingPeriodDays)), //nolint:gosec // bounded 0..365
	}
	if f.status != nil {
		entity.Status = *f.status
	}

	return entity, nil
}

// totalCompensationFor composes the statement from the areas that already own
// each half. The benefits service does not compute pay and the pay service does
// not know about cover; joining them here is what stops two answers to "what
// did this driver earn".
func (r *Resolver) totalCompensationFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	planYear *int,
) (*benefitsservice.TotalCompensation, error) {
	year := intValue(planYear)
	if year <= 0 {
		year = timeutils.YearOfUnix(timeutils.NowUnix())
	}

	req := &benefitsservice.TotalCompensationRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
		PlanYear:   int16(year), //nolint:gosec // a calendar year
	}

	// Year-to-date pay is the settlement area's answer, not this one's. A
	// failure to read it is not a reason to refuse the whole statement — the
	// benefits half is still true — so it is left at zero.
	if r.driverSettlementService != nil {
		summaries, err := r.driverSettlementService.GetYTDPaySummaries(ctx, tenantInfo, year, "")
		if err == nil {
			for _, summary := range summaries {
				if summary != nil && summary.WorkerID == workerID {
					req.GrossPayMinor = summary.GrossEarningsMinor
					break
				}
			}
		}
	}

	return r.benefitsService.TotalCompensation(ctx, req)
}
