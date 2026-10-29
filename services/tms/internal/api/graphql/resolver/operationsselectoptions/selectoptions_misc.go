package operationsselectoptions

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/shared/stringutils"
)

func serviceFailureReasonCodeSelectOptionItem(
	entity *servicefailure.ReasonCode,
) selectoption.Item {
	return selectoption.ItemFor(
		serviceFailureReasonCodeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func serviceFailureReasonCodeSelectOption(
	entity *servicefailure.ReasonCode,
) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringutils.Ptr(entity.Label),
		Meta: map[string]any{
			"code":  entity.Code,
			"label": entity.Label,
		},
	}
}
