package servicetyperesolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/servicetypeports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

func (d *Deps) existingServiceType(
	ctx context.Context,
	id string,
	authCtx *authctx.AuthContext,
) (*servicetype.ServiceType, error) {
	entityID, err := pulid.MustParse(id)
	if err != nil {
		return nil, err
	}

	return d.ServiceTypeService.Get(ctx, servicetypeports.GetServiceTypeByIDRequest{
		ID:         entityID,
		TenantInfo: resolvercore.TenantInfo(authCtx),
	})
}
