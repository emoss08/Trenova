package etajobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/etajobs/etacontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type RefreshResult struct {
	Refreshes int `json:"refreshes"`
	Shipments int `json:"shipments"`
	Rows      int `json:"rows"`
	Attention int `json:"attention"`
}

type SweepResult struct {
	OrganizationsProcessed int      `json:"organizationsProcessed"`
	Shipments              int      `json:"shipments"`
	Rows                   int      `json:"rows"`
	Attention              int      `json:"attention"`
	FailedOrganizations    []string `json:"failedOrganizations"`
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

	Refresher stopetaports.Refresher
	StopEtas  stopetaports.Repository
	Logger    *zap.Logger
}

type Activities struct {
	refresher stopetaports.Refresher
	stopEtas  stopetaports.Repository
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		refresher: p.Refresher,
		stopEtas:  p.StopEtas,
		l:         p.Logger.Named("temporal.shipment-eta-activities"),
	}
}

func (a *Activities) RefreshShipmentActivity(
	ctx context.Context,
	payload *etacontract.RefreshPayload,
) (*RefreshResult, error) {
	tenant := payload.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	refreshed, err := a.refresher.RefreshShipments(ctx, tenant, []pulid.ID{payload.ShipmentID})
	if err != nil {
		a.l.Warn("shipment eta refresh failed",
			zap.String("shipmentId", payload.ShipmentID.String()),
			zap.Error(err))
		return nil, err
	}

	return resultOf(refreshed), nil
}

func (a *Activities) ListSweepTenantsActivity(ctx context.Context) ([]TenantRef, error) {
	tenants, err := a.stopEtas.ListActiveTenants(ctx, stopeta.TrackedShipmentStatuses())
	if err != nil {
		return nil, err
	}

	refs := make([]TenantRef, 0, len(tenants))
	for _, tenant := range tenants {
		refs = append(refs, TenantRef{OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID})
	}
	return refs, nil
}

func (a *Activities) SweepTenantActivity(
	ctx context.Context,
	ref *TenantRef,
) (*RefreshResult, error) {
	tenant := ref.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	if activity.IsActivity(ctx) {
		activity.RecordHeartbeat(ctx, tenant.OrgID.String())
	}

	refreshed, err := a.refresher.RefreshTenant(ctx, tenant)
	if err != nil {
		a.l.Error("shipment eta sweep failed",
			zap.String("organizationId", tenant.OrgID.String()),
			zap.Error(err))
		return nil, err
	}

	a.l.Debug("shipment eta sweep completed for organization",
		zap.String("organizationId", tenant.OrgID.String()),
		zap.Int("shipments", refreshed.Shipments),
		zap.Int("attention", refreshed.Attention))

	return resultOf(refreshed), nil
}

func resultOf(refreshed *stopetaports.RefreshResult) *RefreshResult {
	if refreshed == nil {
		return &RefreshResult{}
	}
	return &RefreshResult{
		Refreshes: 1,
		Shipments: refreshed.Shipments,
		Rows:      refreshed.Rows,
		Attention: refreshed.Attention,
	}
}
