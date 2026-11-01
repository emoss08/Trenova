package operationsselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/reference/domain/dothazmatreference"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Sources) DotHazmatReference(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.DotHazmatReferenceService.GetByIDs(ctx, req.IDs)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			dotHazmatReferenceID,
			dotHazmatReferenceSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.DotHazmatReferenceService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		dotHazmatReferenceSelectOptionItem,
	)
}

func dotHazmatReferenceID(entity *dothazmatreference.DotHazmatReference) pulid.ID {
	return entity.ID
}

func dotHazmatReferenceSelectOptionItem(
	entity *dothazmatreference.DotHazmatReference,
) selectoption.Item {
	return selectoption.ItemFor(
		dotHazmatReferenceSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func dotHazmatReferenceSelectOption(
	entity *dothazmatreference.DotHazmatReference,
) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.ProperShippingName,
		Description: stringutils.Ptr("UN" + entity.UnNumber),
		Meta: map[string]any{
			"unNumber":           entity.UnNumber,
			"properShippingName": entity.ProperShippingName,
			"hazardClass":        entity.HazardClass,
			"subsidiaryHazard":   entity.SubsidiaryHazard,
			"packingGroup":       entity.PackingGroup,
			"specialProvisions":  entity.SpecialProvisions,
			"ergGuide":           entity.ErgGuide,
			"symbols":            entity.Symbols,
		},
	}
}
