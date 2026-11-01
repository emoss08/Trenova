package shipmenttyperesolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/shipmenttypeports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

func (d *Deps) existingShipmentType(
	ctx context.Context,
	id string,
	authCtx *authctx.AuthContext,
) (*shipmenttype.ShipmentType, error) {
	entityID, err := pulid.MustParse(id)
	if err != nil {
		return nil, err
	}

	return d.ShipmentTypeService.Get(ctx, shipmenttypeports.GetShipmentTypeByIDRequest{
		ID:         entityID,
		TenantInfo: resolvercore.TenantInfo(authCtx),
	})
}
