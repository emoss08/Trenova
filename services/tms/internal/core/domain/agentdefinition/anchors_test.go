package agentdefinition_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/stretchr/testify/assert"
)

func TestDescribeAnchors_DatesWhatASourceReported(t *testing.T) {
	t.Parallel()

	now := int64(1791600000)
	text := agentdefinition.DescribeAnchors([]agentdefinition.RuntimeAnchor{
		{
			Kind:  "shipment",
			ID:    "shp_01M49Z0FTJPBFF0HBFFNGYRY9B",
			Label: "SEED-DET-001",
			Facts: []agentdefinition.AnchorFact{
				{Name: "status", Value: "InTransit for Acme Manufacturing"},
				{Name: "last position", Value: "near Amarillo, TX", SeenAt: now - 40*60},
				{Name: "driver hours", Value: "Driving", SeenAt: now - 3*86400, Stale: true},
			},
		},
		{Kind: "invoice", ID: "inv_01M49Z0FTJPBFF0HBFFNGYRY9B", Note: "you can no longer read this record"},
	}, now, "America/New_York")

	assert.Contains(t, text, "<records_in_play>")
	assert.Contains(t, text, "</records_in_play>")
	assert.Contains(t, text, "- Shipment SEED-DET-001 (shp_01M49Z0FTJPBFF0HBFFNGYRY9B)")
	assert.Contains(t, text, "  - status: InTransit for Acme Manufacturing\n")
	assert.Contains(t, text, "near Amarillo, TX (reported 40m ago)")
	assert.Contains(t, text, "Driving (reported 3d 0h ago, too old to plan on)")
	assert.Contains(t, text, "- Invoice (inv_01M49Z0FTJPBFF0HBFFNGYRY9B): you can no longer read this record")
	assert.Contains(t, text, "EDT")
}

func TestDescribeAnchors_IsEmptyWithNothingInPlay(t *testing.T) {
	t.Parallel()

	assert.Empty(t, agentdefinition.DescribeAnchors(nil, 1, "UTC"))
}
