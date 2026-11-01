package permitresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/permit"
	"github.com/emoss08/trenova/pkg/authctx"
)

func permitFromInput(
	input *gqlmodel.PermitInput,
	shipmentID string,
	authCtx *authctx.AuthContext,
) (*permit.Permit, error) {
	parsedShipmentID, err := gqlid.ParseIDField("shipmentId", shipmentID)
	if err != nil {
		return nil, err
	}
	stateID, err := gqlid.ParseIDField("stateId", input.StateID)
	if err != nil {
		return nil, err
	}
	cost, err := gqldecimal.ParseNullDecimalField("cost", input.Cost)
	if err != nil {
		return nil, err
	}

	return &permit.Permit{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		ShipmentID:     parsedShipmentID,
		StateID:        stateID,
		PermitNumber:   input.PermitNumber,
		Status:         input.Status,
		IssuedAt:       gqlscalar.Int64Ptr(input.IssuedAt),
		ExpiresAt:      gqlscalar.Int64Ptr(input.ExpiresAt),
		Cost:           cost,
		Notes:          gqlscalar.StringValue(input.Notes),
		Version:        gqlscalar.Int64Value(input.Version),
	}, nil
}
