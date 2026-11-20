package quotaservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

const managedPlanName = "Trenova Cloud"

type ManagedQuotaGuard struct {
	gate   managedusageports.Gate
	scopes *managedscope.Resolver
	clock  timeutils.Clock
}

var _ platformplanports.QuotaGuard = (*ManagedQuotaGuard)(nil)

func NewManaged(
	gate managedusageports.Gate,
	scopes *managedscope.Resolver,
	clock timeutils.Clock,
) *ManagedQuotaGuard {
	return &ManagedQuotaGuard{gate: gate, scopes: scopes, clock: clock}
}

func (g *ManagedQuotaGuard) scope(tenantInfo pagination.TenantInfo) (managedscope.Scope, error) {
	return g.scopes.ForTenant(tenantInfo.DBTenant())
}

func (g *ManagedQuotaGuard) Enforce(
	ctx context.Context,
	req *platformplanports.QuotaRequest,
) error {
	decision, err := g.Check(ctx, req)
	if err != nil {
		return err
	}
	return DecisionError(decision)
}

func (g *ManagedQuotaGuard) Check(
	ctx context.Context,
	req *platformplanports.QuotaRequest,
) (*platformplanports.QuotaDecision, error) {
	if req == nil {
		return nil, ErrRequestRequired
	}
	if req.Quantity < 0 {
		return nil, ErrInvalidQuantity
	}
	scope, err := g.scope(req.TenantInfo)
	if err != nil {
		return nil, err
	}
	d, err := g.gate.Decide(ctx, scope, req.Meter, req.Quantity)
	if err != nil {
		return nil, err
	}
	return &platformplanports.QuotaDecision{
		Meter:       req.Meter,
		Plan:        platformplan.PlanKeyManaged,
		Allowed:     d.Allowed,
		Unlimited:   !d.Managed,
		Limit:       d.Limit,
		Ceiling:     d.Ceiling,
		CeilingHit:  d.CeilingReached(),
		Used:        d.Used,
		Remaining:   d.Remaining(),
		Requested:   req.Quantity,
		WindowStart: d.WindowStart,
		WindowEnd:   d.WindowEnd,
	}, nil
}

func (g *ManagedQuotaGuard) Usage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*platformplanports.QuotaUsageSummary, error) {
	scope, err := g.scope(tenantInfo)
	if err != nil {
		return nil, err
	}
	state, err := g.gate.State(ctx, scope)
	if err != nil {
		return nil, err
	}
	summary := &platformplanports.QuotaUsageSummary{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Plan:           platformplan.PlanKeyManaged,
		PlanName:       managedPlanName,
		Origin:         platformplan.OriginManaged,
		Meters:         []platformplanports.QuotaMeterUsage{},
		CheckedAt:      g.clock(),
	}
	if state == nil {
		summary.Unlimited = true
		return summary, nil
	}
	summary.Overage = state.Overage
	for i := range state.Meters {
		m := &state.Meters[i]
		d, decideErr := g.gate.Decide(ctx, scope, m.Meter, 0)
		if decideErr != nil {
			return nil, decideErr
		}
		summary.Meters = append(summary.Meters, platformplanports.QuotaMeterUsage{
			Meter:       m.Meter,
			Window:      platformplan.WindowMonthly,
			Limit:       m.Allowance,
			Used:        d.Used,
			Remaining:   d.Remaining(),
			WindowStart: state.WindowStart,
			WindowEnd:   state.WindowEnd,
			HardCap:     m.HardCap,
			Stopped:     !d.Allowed,
		})
	}
	return summary, nil
}
