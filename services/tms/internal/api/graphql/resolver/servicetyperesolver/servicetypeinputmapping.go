package servicetyperesolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/pkg/authctx"
)

func serviceTypeFromInput(
	input *gqlmodel.ServiceTypeInput,
	authCtx *authctx.AuthContext,
) *servicetype.ServiceType {
	entity := &servicetype.ServiceType{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	applyServiceTypeInput(entity, input)

	return entity
}

func applyServiceTypeInput(entity *servicetype.ServiceType, input *gqlmodel.ServiceTypeInput) {
	entity.Status = input.Status
	entity.Code = input.Code
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.Color = gqlscalar.StringValue(input.Color)
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
}

func applyServiceTypePatch(
	entity *servicetype.ServiceType,
	input *gqlmodel.ServiceTypePatchInput,
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
