package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCaptureBatches struct {
	listed []*capture.CaptureBatch
	full   map[pulid.ID]*capture.CaptureBatch
	input  *captureservice.ListBatchesInput
}

func (f *fakeCaptureBatches) ListBatches(
	_ context.Context,
	in *captureservice.ListBatchesInput,
) (*pagination.CursorListResult[*capture.CaptureBatch], error) {
	f.input = in

	return &pagination.CursorListResult[*capture.CaptureBatch]{Items: f.listed}, nil
}

func (f *fakeCaptureBatches) GetBatch(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*capture.CaptureBatch, error) {
	return f.full[id], nil
}

func TestListCaptureBatches_ReadsEachOpenStackWithItsDocuments(t *testing.T) {
	t.Parallel()

	suggested := pulid.MustNew("shp_")
	confidence := 0.92
	batch := &capture.CaptureBatch{
		ID:        pulid.MustNew("cbat_"),
		Status:    capture.BatchReady,
		Source:    capture.SourceScan,
		JobName:   "Morning PODs",
		ItemCount: 1,
		Version:   4,
		Items: []*capture.CaptureItem{{
			ID:                   pulid.MustNew("citm_"),
			Status:               capture.ItemProposed,
			PageIDs:              []pulid.ID{pulid.MustNew("cpg_")},
			SuggestedType:        "shipment",
			SuggestedID:          &suggested,
			SuggestionConfidence: &confidence,
			Version:              2,
		}},
	}
	batches := &fakeCaptureBatches{
		listed: []*capture.CaptureBatch{{ID: batch.ID}},
		full:   map[pulid.ID]*capture.CaptureBatch{batch.ID: batch},
	}
	tool := newListCaptureBatchesTool(batches)

	out, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)
	rows, ok := out.(map[string]any)["batches"].([]captureBatchRow)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(4), rows[0].Version)
	require.Len(t, rows[0].Items, 1)
	assert.Equal(t, suggested.String(), rows[0].Items[0].SuggestedID)
	assert.Equal(t, int64(2), rows[0].Items[0].Version)
	assert.ElementsMatch(t, capture.OpenBatchStatuses(), batches.input.Statuses)
	assert.Equal(t, permission.ResourceCaptureBatch, tool.Policy().Resource)

	one, err := tool.Query(t.Context(), testParams(map[string]any{
		paramCaptureBatchID: batch.ID.String(),
	}))
	require.NoError(t, err)
	assert.Len(t, one.(map[string]any)["batches"].([]captureBatchRow), 1)
}
