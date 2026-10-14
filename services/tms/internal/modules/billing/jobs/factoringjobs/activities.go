package factoringjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/jobs/factoringjobs/factoringcontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxTargetInvoices = 5000

type SweepResult struct {
	OrganizationsProcessed int                           `json:"organizationsProcessed"`
	Totals                 factoringports.EvaluateResult `json:"totals"`
	NoasWarned             int                           `json:"noasWarned"`
	FailedOrganizations    []string                      `json:"failedOrganizations"`
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

	Readiness factoringports.ReadinessService
	States    factoringports.InvoiceStateRepository
	Logger    *zap.Logger
}

type Activities struct {
	readiness factoringports.ReadinessService
	states    factoringports.InvoiceStateRepository
	l         *zap.Logger
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		readiness: p.Readiness,
		states:    p.States,
		l:         p.Logger.Named("temporal.factoring-activities"),
	}
}

func (a *Activities) EvaluateFactoringTargetsActivity(
	ctx context.Context,
	payload *factoringcontract.EvaluatePayload,
) (*factoringports.EvaluateResult, error) {
	tenant := payload.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	heartbeat(ctx, tenant)

	if payload.Everything {
		swept, err := a.readiness.SweepTenant(ctx, tenant)
		if err != nil {
			return nil, err
		}
		return &swept.EvaluateResult, nil
	}

	ids, err := a.resolveInvoices(ctx, tenant, &payload.Pending)
	if err != nil {
		return nil, err
	}
	return a.readiness.Evaluate(ctx, &factoringports.EvaluateInvoicesRequest{
		TenantInfo: tenant,
		InvoiceIDs: ids,
	})
}

func (a *Activities) resolveInvoices(
	ctx context.Context,
	tenant pagination.TenantInfo,
	targets *factoringports.EvaluationTargets,
) ([]pulid.ID, error) {
	ids := append(make([]pulid.ID, 0, len(targets.InvoiceIDs)), targets.InvoiceIDs...)
	if len(targets.CustomerIDs) > 0 {
		found, err := a.states.ListCustomerInvoiceIDs(ctx, &factoringports.ListCustomerInvoiceIDsRequest{
			TenantInfo:  tenant,
			CustomerIDs: targets.CustomerIDs,
			Limit:       maxTargetInvoices,
		})
		if err != nil {
			return nil, err
		}
		ids = append(ids, found...)
	}
	for _, factorID := range targets.FactoringCompanyIDs {
		found, err := a.states.ListCustomerInvoiceIDs(ctx, &factoringports.ListCustomerInvoiceIDsRequest{
			TenantInfo:         tenant,
			FactoringCompanyID: factorID,
			Limit:              maxTargetInvoices,
		})
		if err != nil {
			return nil, err
		}
		ids = append(ids, found...)
	}
	return ids, nil
}

func (a *Activities) ListFactoringSweepTenantsActivity(ctx context.Context) ([]TenantRef, error) {
	tenants, err := a.readiness.ListSweepTenants(ctx)
	if err != nil {
		return nil, err
	}
	refs := make([]TenantRef, 0, len(tenants))
	for _, tenant := range tenants {
		refs = append(refs, TenantRef{OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID})
	}
	return refs, nil
}

func (a *Activities) SweepFactoringTenantActivity(
	ctx context.Context,
	ref *TenantRef,
) (*factoringports.SweepResult, error) {
	tenant := ref.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	heartbeat(ctx, tenant)

	swept, err := a.readiness.SweepTenant(ctx, tenant)
	if err != nil {
		a.l.Error("factoring readiness sweep failed",
			zap.String("organizationId", tenant.OrgID.String()),
			zap.Error(err))
		return nil, err
	}
	return swept, nil
}

func heartbeat(ctx context.Context, tenant pagination.TenantInfo) {
	if activity.IsActivity(ctx) {
		activity.RecordHeartbeat(ctx, tenant.OrgID.String())
	}
}
