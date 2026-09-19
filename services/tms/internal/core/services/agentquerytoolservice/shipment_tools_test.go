package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeShipmentRepo struct {
	repositories.ShipmentRepository

	captured *repositories.ListShipmentsRequest
	items    []*shipment.Shipment
}

func (f *fakeShipmentRepo) List(
	_ context.Context,
	req *repositories.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	f.captured = req

	return &pagination.CursorListResult[*shipment.Shipment]{Items: f.items}, nil
}

// search_shipments carried the same required-query defect as search_worker, so
// "show me everything delivered yesterday" would have failed the same way. It
// already took an optional status, which is the shape the query should have had
// from the start.
func TestSearchShipments_ListsWithoutAQuery(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{items: []*shipment.Shipment{{ID: pulid.MustNew("shp_")}}}
	tool := newSearchShipmentsTool(repo)

	_, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)
	require.NotNil(t, repo.captured)

	assert.Empty(t, repo.captured.Filter.Query)
}

func TestSearchShipments_KeepsTheStatusFilter(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{}
	tool := newSearchShipmentsTool(repo)

	_, err := tool.Query(t.Context(), testParams(map[string]any{"status": "Delivered"}))
	require.NoError(t, err)

	assert.Equal(t, "Delivered", repo.captured.ShipmentOptions.Status)
}

func TestSearchShipments_EmptyResultNamesEveryFilterItApplied(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{items: nil}
	tool := newSearchShipmentsTool(repo)

	result, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"query": "PRO 12345", "status": "Delivered"}),
	)
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Len(t, outcome.SearchedFor, 2, "both the text and the status are reported back")
	assert.NotEmpty(t, outcome.Note)
}

// An unfiltered list that comes back empty means something different from a
// filtered one that does, and the note has to say which — otherwise the model
// reads a short page as the whole population, or an empty organization as a
// near miss.
func TestSearchShipments_UnfilteredEmptyResultSaysSo(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{items: nil}
	tool := newSearchShipmentsTool(repo)

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Contains(t, outcome.Note, "whole set")
}
