package partiesselectoptions

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
)

func factoringCompanySelectOptionItem(
	entity *factoringcompany.FactoringCompany,
) selectoption.Item {
	description := entity.Code
	if entity.City != "" {
		description += " · " + entity.City
	}
	return selectoption.ItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: &description,
			Meta: map[string]any{
				selectoption.MetaKeyCode:   entity.Code,
				selectoption.MetaKeyStatus: string(entity.Status),
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}
