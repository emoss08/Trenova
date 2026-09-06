package compliancejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

// clearinghouseHorizonDays is how far ahead the sweep looks. The annual query
// is a hard obligation, so the office wants it on the list before the day it
// lapses rather than after.
const clearinghouseHorizonDays = 30

// DrugAlcoholSweepResult reports what the sweep repaired.
type DrugAlcoholSweepResult struct {
	WorkersChecked int `json:"workersChecked"`
	Prohibited     int `json:"prohibited"`
	Failed         int `json:"failed"`
}

// DrugAlcoholSweepActivity repairs the roster's testing cache for drivers whose
// standing can move without anybody writing anything: the annual Clearinghouse
// query falls due on a date, and a driver nobody has queried at all has been
// overdue since their first day.
//
// It deliberately does not walk every driver. Only those inside the horizon, or
// with no query on file at all, can have changed — which keeps a nightly job
// over a few thousand drivers to a handful of recomputes.
func (a *Activities) DrugAlcoholSweepActivity(
	ctx context.Context,
) (*DrugAlcoholSweepResult, error) {
	result := new(DrugAlcoholSweepResult)
	until := timeutils.NowUnix() + clearinghouseHorizonDays*secondsPerDay

	refs, err := a.drugAlcoholRepo.ListWorkersWithClearinghouseDue(
		ctx,
		&repositories.ListWorkersWithClearinghouseDueRequest{
			Until: until,
			Limit: sweepPageSize,
		},
	)
	if err != nil {
		return nil, err
	}

	for _, ref := range refs {
		result.WorkersChecked++
		tenantInfo := pagination.TenantInfo{
			OrgID: ref.OrganizationID,
			BuID:  ref.BusinessUnitID,
		}
		standing, rErr := a.drugAlcohol.RefreshRollup(ctx, tenantInfo, ref.WorkerID)
		if rErr != nil {
			result.Failed++
			a.logger.Warn("failed to refresh drug and alcohol rollup after sweep",
				zap.String("workerId", ref.WorkerID.String()),
				zap.Error(rErr))
			continue
		}
		if standing.Status.Blocks() {
			result.Prohibited++
		}
		activity.RecordHeartbeat(ctx, result.WorkersChecked)
	}

	return result, nil
}
