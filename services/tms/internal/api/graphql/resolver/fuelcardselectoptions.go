package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *Resolver) resolveFuelCardSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.fuelPurchaseService.GetCardsByIDs(
			ctx,
			&repositories.GetFuelCardsByIDsRequest{
				TenantInfo: req.tenantInfo,
				IDs:        req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(req.ids, entities, fuelCardID, fuelCardSelectOptionItem)

		return selectOptionConnection(items, len(items), 0)
	}

	entities, err := r.fuelPurchaseService.ListActiveCards(
		ctx,
		&repositories.ListActiveFuelCardsRequest{
			TenantInfo: req.selectQuery.TenantInfo,
			Query:      req.selectQuery.Query,
			Limit:      req.selectQuery.Pagination.Limit,
		},
	)
	if err != nil {
		return nil, err
	}

	items := make([]selectOptionConnectionItem, 0, len(entities))
	for _, entity := range entities {
		items = append(items, fuelCardSelectOptionItem(entity))
	}

	return selectOptionConnection(items, len(items), req.selectQuery.Pagination.SafeOffset())
}

func fuelCardID(entity *fuelpurchase.FuelCard) pulid.ID { return entity.ID }

func fuelCardSelectOptionItem(entity *fuelpurchase.FuelCard) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		fuelCardSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func fuelCardSelectOption(entity *fuelpurchase.FuelCard) *gqlmodel.SelectOption {
	description := entity.Provider.Label()

	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Label + " •••• " + entity.LastFour,
		Description: &description,
		Meta: map[string]any{
			"provider":  string(entity.Provider),
			"lastFour":  entity.LastFour,
			"tractorId": optionalIDPtrString(entity.AssignedTractorID),
			"workerId":  optionalIDPtrString(entity.AssignedWorkerID),
		},
	}
}

func optionalIDPtrString(id *pulid.ID) any {
	if id == nil {
		return nil
	}

	return optionalIDString(*id)
}
