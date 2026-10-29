package worldwatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatchedRecords_AddsTheReadableRecordsInPlay(t *testing.T) {
	t.Parallel()

	req := &agentruntimeports.RunRequest{Context: agentturn.RuntimeContext{
		Anchors: []agentturn.RuntimeAnchor{
			{Kind: "shipment", ID: "shp_01M49Z0FTJPBFF0HBFFNGYRY9B"},
			{
				Kind: "invoice",
				ID:   "inv_01M49Z0FTJPBFF0HBFFNGYRY9B",
				Note: "you can no longer read this record",
			},
		},
	}}

	records := WatchedRecords(req)
	require.Len(t, records, 1)
	assert.Equal(t, "shp_01M49Z0FTJPBFF0HBFFNGYRY9B", records[0].ID)
}
