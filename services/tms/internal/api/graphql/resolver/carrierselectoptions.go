package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Resolver) resolveCarrierSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
				ID:         id,
				TenantInfo: req.tenantInfo,
			})
			if err != nil {
				return nil, err
			}
			items = append(items, carrierSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.carrierRepo.SelectOptions(
		ctx,
		&repositories.CarrierSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		carrierSelectOptionItem,
	)
}

func carrierSelectOptionItem(entity *carrier.Carrier) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		carrierSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func carrierSelectOption(entity *carrier.Carrier) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Code),
		Meta: map[string]any{
			"code":   entity.Code,
			"status": string(entity.Status),
			"scac":   entity.SCAC,
		},
	}
}
