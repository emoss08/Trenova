package etajobs

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
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
	Queue     workqueueports.Queue
	Requester *etacontract.Requester
	Logger    *zap.Logger
}

type Activities struct {
	refresher stopetaports.Refresher
	stopEtas  stopetaports.Repository
	queue     workqueueports.Queue
	requester *etacontract.Requester
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		refresher: p.Refresher,
		stopEtas:  p.StopEtas,
		queue:     p.Queue,
		requester: p.Requester,
		l:         p.Logger.Named("temporal.shipment-eta-activities"),
	}
}

// DrainShipmentEtasActivity projects a page of the tenant's queued shipments
// in one batched refresh. A refresh that fails leaves the page due again
// shortly; the five-minute sweep also projects every active shipment.
func (a *Activities) DrainShipmentEtasActivity(
	ctx context.Context,
	payload *temporaljobs.DrainPayload,
) (*temporaljobs.DrainPage, error) {
	tenant := payload.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	return temporaljobs.DrainDue(ctx, &temporaljobs.DrainDueParams{
		Queue:      a.queue,
		Kind:       etacontract.DrainKind,
		Tenant:     tenant,
		PageSize:   drainPageSize,
		RetryAfter: drainRetryAfter,
		Handle: func(ctx context.Context, shipments []pulid.ID, _ map[pulid.ID]string) map[pulid.ID]error {
			if _, err := a.refresher.RefreshShipments(ctx, tenant, shipments); err != nil {
				a.l.Warn("shipment eta refresh failed; the page is tried again shortly",
					zap.String("organizationId", tenant.OrgID.String()),
					zap.Int("shipments", len(shipments)),
					zap.Error(err))

				return allFailed(shipments, err)
			}

			return nil
		},
	})
}

// WakeDueEtaDrainsActivity wakes every tenant with shipments still queued: the
// backstop for a wake that was lost.
func (a *Activities) WakeDueEtaDrainsActivity(ctx context.Context) (int, error) {
	return a.requester.WakeDue(ctx, drainWakeLimit)
}

func allFailed(ids []pulid.ID, err error) map[pulid.ID]error {
	failed := make(map[pulid.ID]error, len(ids))
	for _, id := range ids {
		failed[id] = err
	}

	return failed
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
