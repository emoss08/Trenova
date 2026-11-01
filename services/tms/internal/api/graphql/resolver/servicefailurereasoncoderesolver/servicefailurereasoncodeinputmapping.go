package servicefailurereasoncoderesolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/intutils"
)

func serviceFailureReasonCodeFromInput(
	input *gqlmodel.ServiceFailureReasonCodeInput,
	authCtx *authctx.AuthContext,
) *servicefailure.ReasonCode {
	entity := &servicefailure.ReasonCode{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		Active:         true,
	}
	applyServiceFailureReasonCodeInput(entity, input)

	return entity
}

func applyServiceFailureReasonCodeInput(
	entity *servicefailure.ReasonCode,
	input *gqlmodel.ServiceFailureReasonCodeInput,
) {
	entity.Code = input.Code
	entity.Label = input.Label
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.Category = input.Category
	entity.AppliesTo = input.AppliesTo
	entity.DefaultStatusCode = gqlscalar.StringValue(input.DefaultStatusCode)
	entity.DefaultReasonCode = gqlscalar.StringValue(input.DefaultReasonCode)
	entity.DefaultExceptionCode = gqlscalar.StringValue(input.DefaultExceptionCode)
	entity.DefaultNote = gqlscalar.StringValue(input.DefaultNote)
	if input.Active != nil {
		entity.Active = *input.Active
	}
	entity.SortOrder = intutils.SafeToInt32(input.SortOrder)
	entity.ExternalMap = input.ExternalMap
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
}
