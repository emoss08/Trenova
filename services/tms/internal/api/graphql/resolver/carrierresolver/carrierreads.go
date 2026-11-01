package carrierresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

func (d *Deps) existingCarrier(
	ctx context.Context,
	id string,
	authCtx *authctx.AuthContext,
) (*carrier.Carrier, error) {
	carrierID, err := pulid.MustParse(id)
	if err != nil {
		return nil, err
	}

	return d.CarrierService.Get(ctx, carrierports.GetCarrierByIDRequest{
		ID:         carrierID,
		TenantInfo: resolvercore.TenantInfo(authCtx),
		CarrierFilterOptions: carrierports.CarrierFilterOptions{
			IncludeContacts:          true,
			IncludeInsurancePolicies: true,
			IncludeEDIChannels:       true,
		},
	})
}
