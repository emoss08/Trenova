package resolver

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
The inbox pages by arrival. An edge cursor that carried only the creation time
and no sort would be refused on the next page — "cursor sort does not match
request sort" — so "load more" would fail on every inbox longer than a page.
*/
func TestInboundMessageConnectionCursorsCarryTheArrivalSort(t *testing.T) {
	t.Parallel()

	first := &inboundmessage.InboundMessage{
		ID:         pulid.MustNew("imsg_"),
		ReceivedAt: 200,
		CreatedAt:  10,
	}
	second := &inboundmessage.InboundMessage{
		ID:         pulid.MustNew("imsg_"),
		ReceivedAt: 100,
		CreatedAt:  20,
	}
	sort := []pagination.CursorSortField{
		{Field: "receivedAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}
	result := pagination.NewCursorListResultWithTotalCount(
		[]*inboundmessage.InboundMessage{first, second, {ID: pulid.MustNew("imsg_")}}, 2, nil,
	).WithCursorSort(sort)
	require.NoError(t, result.WithCursorValues([][]any{
		{int64(200), first.ID.String()},
		{int64(100), second.ID.String()},
	}))

	conn, err := inboundMessageConnection(result)
	require.NoError(t, err)

	require.Len(t, conn.Edges, 2)
	assert.True(t, conn.PageInfo.HasNextPage)
	require.NotNil(t, conn.PageInfo.EndCursor)

	cursor, err := pagination.DecodeCursor(*conn.PageInfo.EndCursor)
	require.NoError(t, err)
	require.NoError(t, pagination.ValidateCursorSort(cursor, sort))
	assert.Equal(t, second.ID, cursor.ID)
}

func TestOptionalMatchTreatsARecordOutOfReachAsAbsent(t *testing.T) {
	t.Parallel()

	summary := &repositories.ShipmentSummary{ProNumber: "PRO-1"}

	got, err := optionalMatch(summary, nil)
	require.NoError(t, err)
	assert.Same(t, summary, got)

	got, err = optionalMatch[*repositories.ShipmentSummary](
		nil, errortypes.NewNotFoundError("Shipment not found within your organization"),
	)
	require.NoError(t, err, "a deleted or out-of-tenant shipment is no match, not a failed inbox")
	assert.Nil(t, got)

	boom := errors.New("database is down")
	_, err = optionalMatch[*repositories.ShipmentSummary](nil, boom)
	assert.ErrorIs(t, err, boom, "a real failure is still a failure")
}
