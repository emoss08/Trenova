package locationresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

func (d *Deps) existingLocation(
	ctx context.Context,
	id string,
	authCtx *authctx.AuthContext,
) (*location.Location, error) {
	locationID, err := pulid.MustParse(id)
	if err != nil {
		return nil, err
	}

	return d.LocationService.Get(ctx, locationports.GetLocationByIDRequest{
		ID:         locationID,
		TenantInfo: resolvercore.TenantInfo(authCtx),
	})
}
