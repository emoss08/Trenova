package billingqueuemapping

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRequestFromFilterReadsEveryNarrowing(t *testing.T) {
	t.Parallel()

	include := true
	minAmount := "5000"
	maxAmount := "12500.50"
	before := 1_800_000_000
	req, err := ListRequestFromFilter(&gqlmodel.BillingQueueItemsFilterInput{
		Lanes:         []billingqueue.Lane{billingqueue.LaneReady, billingqueue.LaneNeedsYou},
		IncludePosted: &include,
		MinAmount:     &minAmount,
		MaxAmount:     &maxAmount,
		CreatedBefore: &before,
	})

	require.NoError(t, err)
	assert.Equal(t, []billingqueue.Lane{billingqueue.LaneReady, billingqueue.LaneNeedsYou}, req.Lanes)
	assert.True(t, req.IncludePosted)
	require.True(t, req.MinAmount.Valid)
	assert.Equal(t, "5000", req.MinAmount.Decimal.String())
	require.True(t, req.MaxAmount.Valid)
	assert.Equal(t, "12500.5", req.MaxAmount.Decimal.String())
	require.NotNil(t, req.CreatedBefore)
	assert.Equal(t, int64(before), *req.CreatedBefore)
}

func TestListRequestFromFilterWithoutAFilterNarrowsNothing(t *testing.T) {
	t.Parallel()

	req, err := ListRequestFromFilter(nil)

	require.NoError(t, err)
	assert.Empty(t, req.Lanes)
	assert.False(t, req.IncludePosted)
	assert.False(t, req.MinAmount.Valid)
	assert.False(t, req.MaxAmount.Valid)
	assert.Nil(t, req.CreatedBefore)
}

func TestListRequestFromFilterRejectsAnUnknownLane(t *testing.T) {
	t.Parallel()

	_, err := ListRequestFromFilter(&gqlmodel.BillingQueueItemsFilterInput{
		Lanes: []billingqueue.Lane{"Somewhere"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown billing queue lane")
}

func TestListRequestFromFilterRejectsANegativeOrMalformedAmount(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"-1", "lots"} {
		amount := value
		_, err := ListRequestFromFilter(&gqlmodel.BillingQueueItemsFilterInput{MinAmount: &amount})
		require.Error(t, err, value)
	}
}

func TestBillingQueueItemConnectionCarriesTheRowsHeadlineAndCursors(t *testing.T) {
	t.Parallel()

	verdict := billingqueue.AgentVerdictConcern
	item := &billingqueue.BillingQueueItem{
		ID:           pulid.MustNew("bqi_"),
		Lane:         billingqueue.LaneReady,
		AgentVerdict: &verdict,
		AgentSummary: "Fuel is 18% above this lane's usual",
		Headline: &billingqueue.Headline{
			Code:     billingqueue.IssuePODMissing,
			CheckKey: billingqueue.CheckPOD,
			Summary:  "Signed POD missing",
			Open:     2,
		},
	}
	total := 41

	conn, err := BillingQueueItemConnectionToModel(
		context.Background(),
		&pagination.CursorListResult[*billingqueue.BillingQueueItem]{
			Items:           []*billingqueue.BillingQueueItem{item},
			HasNextPage:     true,
			TotalCount:      &total,
			TotalCountExact: true,
			EdgeCursors:     []string{"cursor-1"},
		},
	)

	require.NoError(t, err)
	require.Len(t, conn.Edges, 1)
	node := conn.Edges[0].Node
	assert.Equal(t, "cursor-1", conn.Edges[0].Cursor)
	assert.Equal(t, billingqueue.LaneReady, node.Lane)
	require.NotNil(t, node.AgentSummary)
	assert.Equal(t, "Fuel is 18% above this lane's usual", *node.AgentSummary)
	require.NotNil(t, node.Headline)
	assert.Equal(t, "PODMissing", node.Headline.Code)
	assert.Equal(t, "pod", node.Headline.CheckKey)
	assert.Equal(t, 2, node.Headline.Open)
	assert.True(t, conn.PageInfo.HasNextPage)
	require.NotNil(t, conn.PageInfo.EndCursor)
	assert.Equal(t, "cursor-1", *conn.PageInfo.EndCursor)
	assert.Equal(t, &total, conn.TotalCount)
}
