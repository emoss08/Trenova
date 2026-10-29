package shipmentboardresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/pkg/authctx"
)

func (r *QueryResolver) requireCapacityRead(
	ctx context.Context,
	kind gqlmodel.CapacityUnitKind,
) (*authctx.AuthContext, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceShipment, permission.OpRead)
	if err != nil {
		return nil, err
	}

	unitResource := permission.ResourceWorker
	if kind == gqlmodel.CapacityUnitKindCarrier {
		unitResource = permission.ResourceCarrier
	}
	if _, err = r.RequirePermission(ctx, unitResource, permission.OpRead); err != nil {
		return nil, err
	}

	return authCtx, nil
}

func (r *QueryResolver) showsMargins(ctx context.Context, authCtx *authctx.AuthContext) bool {
	return r.VisibleFieldGroups(ctx, authCtx).Has(protectedfield.Margins)
}
