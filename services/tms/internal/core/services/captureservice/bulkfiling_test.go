package captureservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileItemsFilesWhatItCanAndSaysWhyNotForTheRest(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))
	batch := scan(t, s, principal, &OpenBatchInput{ClientKey: "bulk", Source: capture.SourceScan},
		[][]byte{pdfPage(t, 1), pdfPage(t, 2), pdfPage(t, 3)}, map[int]string{2: "T"})
	_, err := s.ProcessBatch(t.Context(), w.tenant, batch.ID, nil)
	require.NoError(t, err)

	items := w.itemsOf(batch.ID)
	require.Len(t, items, 2)
	shipmentID := w.addRecord(permission.ResourceShipment.String())
	missing := pulid.MustNew("shp_")

	_, err = s.FileItems(t.Context(), &FileItemsInput{
		TenantInfo: w.tenant,
		Items: []FileItemInput{
			{ItemID: items[0].ID, TargetType: "shipment", TargetID: shipmentID, Version: items[0].Version},
			{ItemID: items[0].ID, TargetType: "shipment", TargetID: shipmentID, Version: items[0].Version},
		},
	})
	assert.True(t, errortypes.IsError(err), "the same document twice is refused before anything files")
	assert.Equal(t, capture.ItemProposed, w.items[items[0].ID].Status)

	result, err := s.FileItems(t.Context(), &FileItemsInput{
		TenantInfo: w.tenant,
		Items: []FileItemInput{
			{ItemID: items[0].ID, TargetType: "shipment", TargetID: shipmentID, Version: items[0].Version},
			{ItemID: items[1].ID, TargetType: "shipment", TargetID: missing, Version: items[1].Version},
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Filed, 1)
	assert.Equal(t, items[0].ID, result.Filed[0].ID)
	require.Len(t, result.Failures, 1)
	assert.Equal(t, items[1].ID, result.Failures[0].ItemID)
	assert.NotEmpty(t, result.Failures[0].Message)
	assert.NotContains(t, result.Failures[0].Message, "validation failed",
		"a field error reads as the field's own message")
	assert.True(t, w.items[items[1].ID].Status.Open(), "the one that failed can still be filed")
}

func TestFileItemsNeedsFilingPermission(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	w.denied[permission.ResourceCaptureBatch.String()+":update"] = true

	_, err := s.FileItems(t.Context(), &FileItemsInput{
		TenantInfo: w.tenant,
		Items:      []FileItemInput{{ItemID: pulid.MustNew("citm_")}},
	})
	assert.True(t, errortypes.IsAuthorizationError(err))
}
