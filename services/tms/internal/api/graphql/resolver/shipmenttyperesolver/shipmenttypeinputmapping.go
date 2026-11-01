package shipmenttyperesolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/pkg/authctx"
)

func shipmentTypeFromInput(
	input *gqlmodel.ShipmentTypeInput,
	authCtx *authctx.AuthContext,
) *shipmenttype.ShipmentType {
	entity := &shipmenttype.ShipmentType{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	applyShipmentTypeInput(entity, input)

	return entity
}

func applyShipmentTypeInput(entity *shipmenttype.ShipmentType, input *gqlmodel.ShipmentTypeInput) {
	entity.Status = input.Status
	entity.Code = input.Code
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.Color = gqlscalar.StringValue(input.Color)
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
}

func applyShipmentTypePatch(
	entity *shipmenttype.ShipmentType,
	input *gqlmodel.ShipmentTypePatchInput,
) error {
	if err := gqlscalar.ApplyRequired("status", "Status", input.Status, &entity.Status); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("code", "Code", input.Code, &entity.Code); err != nil {
		return err
	}
	gqlscalar.ApplyClearable(input.Description, &entity.Description)
	gqlscalar.ApplyClearable(input.Color, &entity.Color)

	return gqlscalar.ApplyRequiredInt("version", "Version", input.Version, &entity.Version)
}
