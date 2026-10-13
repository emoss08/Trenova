package customertrackingjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/jobs/customertrackingjobs/customertrackingcontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type EvaluateResult struct {
	Shipments int `json:"shipments"`
	Detected  int `json:"detected"`
	Recorded  int `json:"recorded"`
	Delivered int `json:"delivered"`
	Skipped   int `json:"skipped"`
	Failed    int `json:"failed"`
}

func (r *EvaluateResult) add(other *EvaluateResult) {
	if other == nil {
		return
	}
	r.Shipments += other.Shipments
	r.Detected += other.Detected
	r.Recorded += other.Recorded
	r.Delivered += other.Delivered
	r.Skipped += other.Skipped
	r.Failed += other.Failed
}

type SweepResult struct {
	OrganizationsProcessed int            `json:"organizationsProcessed"`
	Totals                 EvaluateResult `json:"totals"`
	FailedOrganizations    []string       `json:"failedOrganizations"`
}

type TenantRef struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

func (t *TenantRef) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: t.OrganizationID, BuID: t.BusinessUnitID}
}

type ActivitiesParams struct {
	fx.In

	Notifier customertrackingports.StatusNotifier
	Logger   *zap.Logger
}

type Activities struct {
	notifier customertrackingports.StatusNotifier
	l        *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		notifier: p.Notifier,
		l:        p.Logger.Named("temporal.customer-status-activities"),
	}
}

func (a *Activities) EvaluateCustomerStatusActivity(
	ctx context.Context,
	payload *customertrackingcontract.EvaluatePayload,
) (*EvaluateResult, error) {
	tenant := payload.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	evaluated, err := a.notifier.Evaluate(ctx, tenant, payload.ShipmentID)
	if err != nil {
		a.l.Warn("customer status evaluation failed",
			zap.String("shipmentId", payload.ShipmentID.String()),
			zap.Error(err))
		return nil, err
	}
	result := resultOf(evaluated)
	result.Shipments = 1
	return result, nil
}

func (a *Activities) ListCustomerStatusSweepTenantsActivity(
	ctx context.Context,
) ([]TenantRef, error) {
	tenants, err := a.notifier.ListSweepTenants(ctx)
	if err != nil {
		return nil, err
	}
	refs := make([]TenantRef, 0, len(tenants))
	for _, tenant := range tenants {
		refs = append(refs, TenantRef{OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID})
	}
	return refs, nil
}

func (a *Activities) SweepCustomerStatusTenantActivity(
	ctx context.Context,
	ref *TenantRef,
) (*EvaluateResult, error) {
	tenant := ref.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	if activity.IsActivity(ctx) {
		activity.RecordHeartbeat(ctx, tenant.OrgID.String())
	}

	swept, err := a.notifier.SweepTenant(ctx, tenant)
	if err != nil {
		a.l.Error("customer status sweep failed",
			zap.String("organizationId", tenant.OrgID.String()),
			zap.Error(err))
		return nil, err
	}
	result := resultOf(&swept.EvaluateResult)
	result.Shipments = swept.Shipments
	return result, nil
}

func (a *Activities) PurgeDeadCustomerTrackingLinksActivity(ctx context.Context) (int, error) {
	return a.notifier.PurgeDeadLinks(ctx)
}

func resultOf(evaluated *customertrackingports.EvaluateResult) *EvaluateResult {
	if evaluated == nil {
		return &EvaluateResult{}
	}
	return &EvaluateResult{
		Detected:  evaluated.Detected,
		Recorded:  evaluated.Recorded,
		Delivered: evaluated.Delivered,
		Skipped:   evaluated.Skipped,
		Failed:    evaluated.Failed,
	}
}
