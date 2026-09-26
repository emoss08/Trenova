package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"

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

	// Completed, not "Delivered". These fixtures used to say Delivered, copied
	// from the tool's own description, which named a status the system does not
	// have — the documentation was wrong for long enough to mislead its tests.
	_, err := tool.Query(t.Context(), testParams(map[string]any{"status": "Completed"}))
	require.NoError(t, err)

	assert.Equal(t, "Completed", repo.captured.ShipmentOptions.Status)
}

func TestSearchShipments_EmptyResultNamesEveryFilterItApplied(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{items: nil}
	tool := newSearchShipmentsTool(repo)

	result, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"query": "PRO 12345", "status": "Completed"}),
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

// search_shipments returned the stored entity for the same reason
// search_worker did, and pays the same cost: a shipment carries its moves,
// stops, commodities and charges, none of which a list of matches needs.
func TestSearchShipments_ReturnsTheCuratedRowNotTheStoredEntity(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{items: []*shipment.Shipment{{
		ID:        pulid.MustNew("shp_"),
		ProNumber: "S-1001",
		Status:    shipment.StatusNew,
	}}}

	result, err := newSearchShipmentsTool(repo).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]shipmentRow)
	require.True(t, ok, "search must return the curated row, not the entity")
	require.Len(t, rows, 1)

	assert.Equal(t, "S-1001", rows[0].ProNumber)
}

// A shipment that has not arrived is in transit, not missing paperwork. The
// two absences read differently and have to stay that way through search.
func TestSearchShipments_DistinguishesInTransitFromUnrecorded(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{items: []*shipment.Shipment{{
		ID: pulid.MustNew("shp_"), ProNumber: "S-1001", Status: shipment.StatusNew,
	}}}

	result, err := newSearchShipmentsTool(repo).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, _ := result.(searchOutcome)
	encoded, err := sonic.Marshal(outcome.Items)
	require.NoError(t, err)

	assert.Contains(t, string(encoded), "not delivered yet")
	assert.NotContains(t, string(encoded), "none on file",
		"an undelivered load is not a record with a gap in it")
}

/*
The schema was telling the model to use a status that does not exist.

"Optional status filter, such as New, Assigned, InTransit, Delivered, or
Canceled" — and there is no Delivered. A delivered load is Completed. The
parameter was unvalidated, so the invented status went to the repository,
matched nothing, and came back as an empty page the model reports as "there are
no delivered shipments". The tool was instructing its caller to produce exactly
the failure the search outcome exists to prevent.
*/
func TestSearchShipments_RefusesAStatusThatDoesNotExist(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{}
	_, err := newSearchShipmentsTool(repo).Query(t.Context(), testParams(map[string]any{
		"status": "Delivered",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Completed", "the refusal has to name what to use instead")
	assert.Nil(t, repo.captured, "nothing may be queried on a refused argument")
}

func TestSearchShipments_AcceptsAStatusThatExists(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{}
	_, err := newSearchShipmentsTool(repo).Query(t.Context(), testParams(map[string]any{
		"status": "InTransit",
	}))

	require.NoError(t, err)
	assert.Equal(t, "InTransit", repo.captured.ShipmentOptions.Status)
}

// Every status the schema offers has to be one the tool accepts, or the model
// is being handed a value that fails on arrival.
func TestSearchShipmentsSchema_OffersOnlyStatusesTheToolAccepts(t *testing.T) {
	t.Parallel()

	properties, ok := newSearchShipmentsTool(&fakeShipmentRepo{}).
		ParamSchema()["properties"].(map[string]any)
	require.True(t, ok)
	offered, ok := properties["status"].(map[string]any)["enum"].([]string)
	require.True(t, ok)
	require.NotEmpty(t, offered)

	for _, value := range offered {
		_, err := shipmentStatusFilter(map[string]any{"status": value})
		assert.NoError(t, err, "schema offers %q", value)
	}
}

// "Not yet billed" is a shipment billing has never received: its transfer
// stage is empty. The filter says so, and isnull reaches the repository as
// the condition that finds them.
func TestListShipments_FindsShipmentsNeverTransferredToBilling(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{}
	tool := newListShipmentsTool(repo)

	assert.Contains(t, tool.Description(), "isnull means never transferred")

	_, err := tool.Query(t.Context(), testParams(map[string]any{
		"filters": []any{map[string]any{
			"field": "billingTransferStatus", "operator": "isnull",
		}},
	}))
	require.NoError(t, err)
	require.NotNil(t, repo.captured)
	require.Len(t, repo.captured.Filter.FieldFilters, 1)
	assert.Equal(t, "billingTransferStatus", repo.captured.Filter.FieldFilters[0].Field)
	assert.Equal(t, "isnull", string(repo.captured.Filter.FieldFilters[0].Operator))
}
