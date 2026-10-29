package partiesselectoptions

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/shared/stringutils"
)

func customerSelectOptionItem(entity *customer.Customer) selectoption.Item {
	return selectoption.ItemFor(
		customerSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func customerSelectOption(entity *customer.Customer) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Code),
		Meta: map[string]any{
			"code":   entity.Code,
			"status": string(entity.Status),
		},
	}
}
