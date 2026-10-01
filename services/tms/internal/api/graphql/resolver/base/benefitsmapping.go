package base

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/benefitsservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// totalCompensationFor composes the statement from the areas that already own
// each half. The benefits service does not compute pay and the pay service does
// not know about cover; joining them here is what stops two answers to "what
// did this driver earn".
func (r *Resolver) TotalCompensationFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	planYear *int,
) (*benefitsservice.TotalCompensation, error) {
	year := IntValue(planYear)
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
	if r.DriverSettlementService != nil {
		summaries, err := r.DriverSettlementService.GetYTDPaySummaries(ctx, tenantInfo, year, "")
		if err == nil {
			for _, summary := range summaries {
				if summary != nil && summary.WorkerID == workerID {
					req.GrossPayMinor = summary.GrossEarningsMinor
					break
				}
			}
		}
	}

	return r.BenefitsService.TotalCompensation(ctx, req)
}
