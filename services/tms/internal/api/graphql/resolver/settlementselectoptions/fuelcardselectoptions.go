package settlementselectoptions

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/shared/pulid"
)

func fuelCardID(entity *fuelpurchase.FuelCard) pulid.ID { return entity.ID }

func fuelCardSelectOptionItem(entity *fuelpurchase.FuelCard) selectoption.Item {
	return selectoption.ItemFor(
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
			"tractorId": optionalIDMetaFromPtr(entity.AssignedTractorID),
			"workerId":  optionalIDMetaFromPtr(entity.AssignedWorkerID),
		},
	}
}

// optionalIDMetaFromPtr renders an optional id into a GraphQL meta map, where
// an unset reference has to arrive as null rather than as "".
func optionalIDMetaFromPtr(id *pulid.ID) any {
	if id == nil || id.IsNil() {
		return nil
	}

	return id.String()
}
