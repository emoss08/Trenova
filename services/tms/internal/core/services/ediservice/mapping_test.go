package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestMappingResolutionIndex(t *testing.T) {
	sourceID := pulid.MustNew("cus_")
	targetID := pulid.MustNew("cus_")
	unresolvedID := pulid.MustNew("st_")

	index := resolutionIndex([]edi.MappingResolution{
		{
			EntityType: edi.MappingEntityTypeCustomer,
			SourceID:   sourceID,
			TargetID:   targetID,
			Resolved:   true,
		},
		{
			EntityType: edi.MappingEntityTypeServiceType,
			SourceID:   unresolvedID,
			Resolved:   false,
		},
	})

	mapped, ok := mappedID(index, edi.MappingEntityTypeCustomer, sourceID)
	require.True(t, ok)
	require.Equal(t, targetID, mapped)

	_, ok = mappedID(index, edi.MappingEntityTypeServiceType, unresolvedID)
	require.False(t, ok)
}

func TestRequiredEntityTypesAreStable(t *testing.T) {
	required := map[edi.MappingEntityType][]pulid.ID{
		edi.MappingEntityTypeLocation:        {pulid.MustNew("loc_")},
		edi.MappingEntityTypeCustomer:        {pulid.MustNew("cus_")},
		edi.MappingEntityTypeFormulaTemplate: {pulid.MustNew("ft_")},
	}

	require.Equal(t, []edi.MappingEntityType{
		edi.MappingEntityTypeCustomer,
		edi.MappingEntityTypeFormulaTemplate,
		edi.MappingEntityTypeLocation,
	}, requiredEntityTypes(required))
}
