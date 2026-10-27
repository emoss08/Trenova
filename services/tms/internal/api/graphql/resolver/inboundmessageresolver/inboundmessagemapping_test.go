package inboundmessageresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboundMessageConnectionCursorsCarryTheArrivalSort(t *testing.T) {
	t.Parallel()

	first := &inboundmessage.InboundMessage{ID: pulid.MustNew("imsg_"), ReceivedAt: 200}
	second := &inboundmessage.InboundMessage{ID: pulid.MustNew("imsg_"), ReceivedAt: 100}
	sort := []pagination.CursorSortField{
		{Field: "receivedAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}
	edge := func(keyset string) string {
		cursor, err := pagination.EncodeKeysetCursor(sort, keyset)
		require.NoError(t, err)
		return cursor
	}
	result := &pagination.CursorListResult[*inboundmessage.InboundMessage]{
		Items:       []*inboundmessage.InboundMessage{first, second},
		HasNextPage: true,
		EdgeCursors: []string{edge("k1"), edge("k2")},
	}

	conn, err := inboundMessageConnection(result)
	require.NoError(t, err)

	require.Len(t, conn.Edges, 2)
	assert.True(t, conn.PageInfo.HasNextPage)
	require.NotNil(t, conn.PageInfo.EndCursor)
	assert.Equal(t, result.EdgeCursors[1], *conn.PageInfo.EndCursor)

	cursor, err := pagination.DecodeCursor(*conn.PageInfo.EndCursor)
	require.NoError(t, err)
	assert.True(t, cursor.MatchesSort(sort))
	assert.Equal(t, "k2", cursor.Keyset)
}
