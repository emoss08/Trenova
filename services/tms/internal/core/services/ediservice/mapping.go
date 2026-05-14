package ediservice

import (
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func mappingIndex(
	items []*edi.EDIMappingProfileItem,
) map[edi.MappingEntityType]map[pulid.ID]*edi.EDIMappingProfileItem {
	result := make(map[edi.MappingEntityType]map[pulid.ID]*edi.EDIMappingProfileItem)
	for _, item := range items {
		if _, ok := result[item.EntityType]; !ok {
			result[item.EntityType] = map[pulid.ID]*edi.EDIMappingProfileItem{}
		}
		result[item.EntityType][item.SourceID] = item
	}
	return result
}

func resolutionIndex(
	resolutions []edi.MappingResolution,
) map[edi.MappingEntityType]map[pulid.ID]pulid.ID {
	result := make(map[edi.MappingEntityType]map[pulid.ID]pulid.ID)
	for _, resolution := range resolutions {
		if !resolution.Resolved || resolution.TargetID.IsNil() {
			continue
		}
		if _, ok := result[resolution.EntityType]; !ok {
			result[resolution.EntityType] = map[pulid.ID]pulid.ID{}
		}
		result[resolution.EntityType][resolution.SourceID] = resolution.TargetID
	}
	return result
}

func mappedID(
	index map[edi.MappingEntityType]map[pulid.ID]pulid.ID,
	entityType edi.MappingEntityType,
	sourceID pulid.ID,
) (pulid.ID, bool) {
	if sourceID.IsNil() {
		return pulid.Nil, true
	}

	target, ok := index[entityType][sourceID]
	return target, ok && target.IsNotNil()
}

func optionalMappedID(
	index map[edi.MappingEntityType]map[pulid.ID]pulid.ID,
	entityType edi.MappingEntityType,
	sourceID pulid.ID,
) pulid.ID {
	target, ok := mappedID(index, entityType, sourceID)
	if !ok {
		return pulid.Nil
	}
	return target
}

func flattenRequiredIDs(required map[edi.MappingEntityType][]pulid.ID) []pulid.ID {
	total := 0
	for _, ids := range required {
		total += len(ids)
	}

	result := make([]pulid.ID, 0, total)
	for _, ids := range required {
		result = append(result, ids...)
	}
	return result
}

func requiredEntityTypes(
	required map[edi.MappingEntityType][]pulid.ID,
) []edi.MappingEntityType {
	ordered := []edi.MappingEntityType{
		edi.MappingEntityTypeCustomer,
		edi.MappingEntityTypeServiceType,
		edi.MappingEntityTypeShipmentType,
		edi.MappingEntityTypeFormulaTemplate,
		edi.MappingEntityTypeLocation,
		edi.MappingEntityTypeCommodity,
		edi.MappingEntityTypeAccessorialCharge,
	}

	result := make([]edi.MappingEntityType, 0, len(ordered))
	for _, entityType := range ordered {
		if len(required[entityType]) > 0 {
			result = append(result, entityType)
		}
	}
	return result
}

func unresolvedMappingsError(unresolved []edi.MappingResolution) error {
	multiErr := errortypes.NewMultiError()
	for _, item := range unresolved {
		multiErr.Add(
			string(item.EntityType),
			errortypes.ErrRequired,
			"Mapping is required for source ID "+item.SourceID.String(),
		)
	}
	return multiErr
}
