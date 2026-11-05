package scimjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/jobs/scimjobs/scimcontract"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Reconciler scimports.GrantReconciler
}

type Activities struct {
	reconciler scimports.GrantReconciler
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{reconciler: p.Reconciler}
}

func (a *Activities) ReconcilePendingActivity(
	ctx context.Context,
	payload *scimcontract.ReconcilePayload,
) (*scimports.ReconcileResult, error) {
	return a.reconciler.ReconcilePending(ctx, payload.TenantInfo(), payload.DirectoryID)
}

func (a *Activities) RequestPendingReconcilesActivity(ctx context.Context) (int, error) {
	return a.reconciler.RequestPendingReconciles(ctx)
}
