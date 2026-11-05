package quotaservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

const managedPlanName = "Trenova Cloud"

type ManagedQuotaGuard struct {
	gate  managedusageports.Gate
	clock timeutils.Clock
}

var _ platformplanports.QuotaGuard = (*ManagedQuotaGuard)(nil)

func NewManaged(gate managedusageports.Gate, clock timeutils.Clock) *ManagedQuotaGuard {
	return &ManagedQuotaGuard{gate: gate, clock: clock}
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
	d, err := g.gate.Decide(ctx, req.Meter, req.Quantity)
	if err != nil {
		return nil, err
	}
	return &platformplanports.QuotaDecision{
		Meter:       req.Meter,
		Plan:        platformplan.PlanKeyManaged,
		Allowed:     d.Allowed,
		Unlimited:   !d.Managed,
		Limit:       d.Limit,
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
	state, err := g.gate.State(ctx)
	if err != nil {
		return nil, err
	}
	summary := &platformplanports.QuotaUsageSummary{
		OrganizationID:         tenantInfo.OrgID,
		BusinessUnitID:         tenantInfo.BuID,
		Plan:                   platformplan.PlanKeyManaged,
		PlanName:               managedPlanName,
		Origin:                 platformplan.OriginManaged,
		RestrictedCapabilities: []platformplan.Capability{},
		Meters:                 []platformplanports.QuotaMeterUsage{},
		CheckedAt:              g.clock(),
	}
	if state == nil {
		summary.Unlimited = true
		return summary, nil
	}
	for i := range state.Meters {
		m := &state.Meters[i]
		d := managedusage.Decide(state, m.Meter, 0)
		summary.Meters = append(summary.Meters, platformplanports.QuotaMeterUsage{
			Meter:       m.Meter,
			Window:      platformplan.WindowMonthly,
			Limit:       m.Allowance,
			Used:        m.Quantity,
			Remaining:   d.Remaining(),
			WindowStart: state.WindowStart,
			WindowEnd:   state.WindowEnd,
		})
	}
	return summary, nil
}
