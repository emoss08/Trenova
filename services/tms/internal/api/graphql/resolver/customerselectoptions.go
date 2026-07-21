package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

func (r *Resolver) resolveCustomerSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
				ID:         id,
				TenantInfo: req.tenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, customerSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.customerRepo.SelectOptions(
		ctx,
		&repositories.CustomerSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		customerSelectOptionItem,
	)
}

func customerSelectOptionItem(entity *customer.Customer) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		customerSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func customerSelectOption(entity *customer.Customer) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(entity.Code),
		Meta: map[string]any{
			"code":   entity.Code,
			"status": string(entity.Status),
		},
	}
}
