package benefitsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/workforce/app/benefitsservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// totalCompensationFor composes the statement from the areas that already own
// each half. The benefits service does not compute pay and the pay service does
// not know about cover; joining them here is what stops two answers to "what
// did this driver earn".
func (r *Deps) TotalCompensationFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	planYear *int,
	includePay bool,
) (*benefitsservice.TotalCompensation, error) {
	year, err := gqlscalar.NarrowIntValue[int16]("planYear", planYear)
	if err != nil {
		return nil, err
	}
	if year <= 0 {
		year = int16(
			timeutils.YearOfUnix(timeutils.NowUnix()),
		) //nolint:gosec // the current calendar year
	}

	req := &benefitsservice.TotalCompensationRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
		PlanYear:   year,
	}

	// Year-to-date pay is the settlement area's answer, not this one's. A
	// failure to read it is not a reason to refuse the whole statement — the
	// benefits half is still true — so it is left at zero.
	if includePay && r.DriverSettlementService != nil {
		summaries, payErr := r.DriverSettlementWorkspaceService.GetYTDPaySummaries(
			ctx,
			tenantInfo,
			int(year),
			"",
		)
		if payErr == nil {
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
