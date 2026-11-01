package ediinput

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/shared/pulid"
)

func MappingItems(
	items []*gqlmodel.EDIMappingProfileItemInput,
) ([]*edi.EDIMappingProfileItem, error) {
	entities := make([]*edi.EDIMappingProfileItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		id, err := gqlid.ParseOptionalIDField("id", item.ID)
		if err != nil {
			return nil, err
		}
		targetID, err := gqlid.ParseIDField("targetId", item.TargetID)
		if err != nil {
			return nil, err
		}
		entities = append(entities, &edi.EDIMappingProfileItem{
			ID:          id,
			EntityType:  item.EntityType,
			SourceID:    pulid.ID(item.SourceID),
			SourceLabel: gqlscalar.StringValue(item.SourceLabel),
			TargetID:    targetID,
			TargetLabel: gqlscalar.StringValue(item.TargetLabel),
		})
	}

	return entities, nil
}
