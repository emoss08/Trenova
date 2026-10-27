package selectoptionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
)

const (
	metaKeyCode   = "code"
	metaKeyStatus = "status"
)

func (r *Deps) resolveFactoringCompanySelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.FactoringCompanyRepo.GetByID(
				ctx,
				factoringcompanyports.GetFactoringCompanyByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, factoringCompanySelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.FactoringCompanyRepo.SelectOptions(
		ctx,
		&factoringcompanyports.FactoringCompanySelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination,
		factoringCompanySelectOptionItem,
	)
}

func factoringCompanySelectOptionItem(
	entity *factoringcompany.FactoringCompany,
) selectOptionConnectionItem {
	description := entity.Code
	if entity.City != "" {
		description += " · " + entity.City
	}
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: &description,
			Meta: map[string]any{
				metaKeyCode:   entity.Code,
				metaKeyStatus: string(entity.Status),
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}
