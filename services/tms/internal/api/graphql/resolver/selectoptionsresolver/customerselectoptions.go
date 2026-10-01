package selectoptionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Deps) resolveCustomerSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.CustomerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
				ID:         id,
				TenantInfo: req.TenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, customerSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.CustomerRepo.SelectOptions(
		ctx,
		&repositories.CustomerSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
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
		Description: stringutils.Ptr(entity.Code),
		Meta: map[string]any{
			"code":   entity.Code,
			"status": string(entity.Status),
		},
	}
}
