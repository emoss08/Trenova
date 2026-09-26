package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureFilingParsesIdsAndTreatsABlankDocumentTypeAsNone(t *testing.T) {
	t.Parallel()

	itemID, shipmentID := pulid.MustNew("citm_"), pulid.MustNew("shp_")
	blank := ""

	in, err := captureFiling(itemID.String(), "shipment", shipmentID.String(), &blank, 4)
	require.NoError(t, err)
	assert.Equal(t, itemID, in.ItemID)
	assert.Equal(t, shipmentID, in.TargetID)
	assert.Nil(t, in.DocumentTypeID)
	assert.Equal(t, int64(4), in.Version)

	_, err = captureFiling(itemID.String(), "shipment", "not an id", nil, 1)
	require.Error(t, err)
}

func TestCaptureLayoutsKeepPageOrderAndRotationsKeyByPage(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("cpg_"), pulid.MustNew("cpg_")

	layouts, err := captureItemLayouts(&gqlmodel.EditCaptureItemsInput{
		Items: []*gqlmodel.CaptureItemLayoutInput{
			{PageIds: []string{second.String(), first.String()}},
		},
	})
	require.NoError(t, err)
	require.Len(t, layouts, 1)
	assert.Equal(t, []pulid.ID{second, first}, layouts[0].PageIDs)

	_, err = captureItemLayouts(&gqlmodel.EditCaptureItemsInput{
		Items: []*gqlmodel.CaptureItemLayoutInput{nil},
	})
	require.Error(t, err)

	rotations, err := capturePageRotations([]*gqlmodel.CapturePageRotationInput{
		{PageID: first.String(), Rotation: 90},
	})
	require.NoError(t, err)
	assert.Equal(t, map[pulid.ID]int{first: 90}, rotations)

	none, err := capturePageRotations(nil)
	require.NoError(t, err)
	assert.Nil(t, none, "no rotations leaves every page as it is")
}

func TestCaptureCoverSheetSpecsAllowAPlainSeparator(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	kind := "shipment"
	target := shipmentID.String()

	specs, err := captureCoverSheetSpecs([]*gqlmodel.CaptureCoverSheetInput{
		{},
		{TargetType: &kind, TargetID: &target},
	})
	require.NoError(t, err)
	require.Len(t, specs, 2)
	assert.Empty(t, specs[0].TargetType)
	assert.Nil(t, specs[0].TargetID)
	require.NotNil(t, specs[1].TargetID)
	assert.Equal(t, shipmentID, *specs[1].TargetID)
}
