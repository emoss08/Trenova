package ptojobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Ledger *ptoledgerservice.Service
	Logger *zap.Logger
}

type Activities struct {
	ledger *ptoledgerservice.Service
	logger *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		ledger: p.Ledger,
		logger: p.Logger.Named("pto-activities"),
	}
}

func (a *Activities) ListAccrualTenantsActivity(
	ctx context.Context,
	input PTOAccrualWorkflowInput,
) ([]temporaljobs.TenantWorkItem, error) {
	if !input.OrganizationID.IsNil() && !input.BusinessUnitID.IsNil() {
		return []temporaljobs.TenantWorkItem{
			temporaljobs.NewTenantWorkItem(pagination.TenantInfo{
				OrgID: input.OrganizationID,
				BuID:  input.BusinessUnitID,
			}, accrualPageSize),
		}, nil
	}

	tenants, err := a.ledger.ListAccrualTenants(ctx)
	if err != nil {
		return nil, err
	}
	return temporaljobs.BuildTenantWorkItems(tenants, accrualPageSize), nil
}

func (a *Activities) AccrueTenantPageActivity(
	ctx context.Context,
	input AccrualTenantPageInput,
) (*AccrualTenantPageResult, error) {
	asOf := input.AsOf
	if asOf <= 0 {
		asOf = timeutils.NowUnix()
	}
	tenantInfo := input.TenantInfo()
	actor := ptoledgerservice.SystemActor()
	result := &AccrualTenantPageResult{}

	if !input.WorkerID.IsNil() {
		run, err := a.ledger.RunAccrualForWorker(ctx, tenantInfo, input.WorkerID, asOf, actor)
		if err != nil {
			return nil, err
		}
		copyRun(result, run)
		return result, nil
	}

	page, err := a.ledger.RunAccrualForTenant(
		ctx,
		tenantInfo,
		asOf,
		input.AfterID,
		temporaljobs.NormalizeLimit(input.Limit, accrualPageSize),
		actor,
	)
	if err != nil {
		return nil, err
	}
	activity.RecordHeartbeat(ctx, page.NextAfter)
	copyRun(result, page.Result)
	result.NextAfterID = page.NextAfter

	return result, nil
}

func copyRun(dst *AccrualTenantPageResult, src *ptoledgerservice.AccrualRunResult) {
	if src == nil {
		return
	}
	dst.WorkersProcessed = src.WorkersProcessed
	dst.WorkersSkipped = src.WorkersSkipped
	dst.EntriesPosted = src.EntriesPosted
	dst.EntriesCapped = src.EntriesCapped
	dst.EntriesSkipped = src.EntriesSkipped
}
