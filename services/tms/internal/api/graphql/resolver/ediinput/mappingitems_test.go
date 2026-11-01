package ediinput

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMappingItems_MapsEachEntry(t *testing.T) {
	t.Parallel()

	itemID := pulid.MustNew("edimi_")
	itemRaw := itemID.String()
	targetID := pulid.MustNew("cus_")
	label := "H-E-B San Antonio"

	items, err := MappingItems([]*gqlmodel.EDIMappingProfileItemInput{
		{
			ID:          &itemRaw,
			EntityType:  edi.MappingEntityTypeCustomer,
			SourceID:    "HEB-SAT",
			SourceLabel: &label,
			TargetID:    targetID.String(),
		},
		nil,
	})
	require.NoError(t, err)
	require.Len(t, items, 1)

	assert.Equal(t, itemID, items[0].ID)
	assert.Equal(t, edi.MappingEntityTypeCustomer, items[0].EntityType)
	assert.Equal(t, pulid.ID("HEB-SAT"), items[0].SourceID)
	assert.Equal(t, label, items[0].SourceLabel)
	assert.Equal(t, targetID, items[0].TargetID)
	assert.Empty(t, items[0].TargetLabel)
}

func TestMappingItems_RefusesATargetThatIsNotAnID(t *testing.T) {
	t.Parallel()

	_, err := MappingItems([]*gqlmodel.EDIMappingProfileItemInput{
		{EntityType: edi.MappingEntityTypeCustomer, SourceID: "HEB-SAT", TargetID: "nope"},
	})

	require.True(t, errortypes.IsError(err), "%v", err)
}
