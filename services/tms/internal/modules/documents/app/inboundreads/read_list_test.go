package inboundreads

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareList_SortsTheFolderByDueOrNewest(t *testing.T) {
	t.Parallel()

	due := &inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{Query: "  tender "},
		Lane:   inboundmessageports.LaneWaiting,
		Sort:   inboundmessageports.SortDue,
	}
	require.NoError(t, PrepareList(due))
	assert.Equal(t, []domaintypes.SortField{
		{Field: "dueAt", Direction: dbtype.SortDirectionAsc},
		{Field: "receivedAt", Direction: dbtype.SortDirectionDesc},
	}, due.Filter.Sort)
	assert.Equal(t, "tender", due.Filter.Query)

	plain := &inboundmessageports.ListInboundMessagesRequest{Filter: &pagination.QueryOptions{}}
	require.NoError(t, PrepareList(plain))
	assert.Equal(t, inboundmessageports.SortNewest, plain.Sort)
	assert.Equal(t,
		[]domaintypes.SortField{{Field: "receivedAt", Direction: dbtype.SortDirectionDesc}},
		plain.Filter.Sort,
	)

	thread := &inboundmessageports.ListInboundMessagesRequest{
		Filter:    &pagination.QueryOptions{},
		Sort:      inboundmessageports.SortDue,
		ThreadKey: "abc@northwind.example",
	}
	require.NoError(t, PrepareList(thread))
	assert.Equal(t, inboundmessageports.SortNewest, thread.Sort, "a thread reads newest first")
}

func TestPrepareList_RefusesAFolderItCannotAnswer(t *testing.T) {
	t.Parallel()

	assert.Error(t, PrepareList(&inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{},
		Lane:   inboundmessageports.Lane("Archive"),
	}))
	assert.Error(t, PrepareList(&inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{},
		Sort:   inboundmessageports.Sort("Oldest"),
	}))
	assert.Error(t, PrepareList(&inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{},
		Lane:   inboundmessageports.LaneAssignedToMe,
	}), "assigned to me needs a viewer")
	assert.Error(t, PrepareList(&inboundmessageports.ListInboundMessagesRequest{
		Filter:    &pagination.QueryOptions{},
		ThreadKey: strings.Repeat("x", inboundmessage.MaxThreadKeyLength+1),
	}))
}

func TestFoldCounts_SplitsWaitingParkedAndMine(t *testing.T) {
	t.Parallel()

	intake := pulid.MustNew("imbx_")
	counts := FoldCounts([]inboundmessageports.InboundMessageCount{
		{Status: inboundmessage.StatusInReview, Classification: inboundmessage.ClassificationTender,
			MailboxID: intake, Mine: true, Overdue: true, Count: 2},
		{Status: inboundmessage.StatusInReview, Classification: inboundmessage.ClassificationTender,
			MailboxID: intake, Count: 3},
		{Status: inboundmessage.StatusInReview, Classification: inboundmessage.ClassificationTender,
			MailboxID: intake, Parked: inboundmessage.ParkingSnoozed, Mine: true, Overdue: true, Count: 4},
		{
			Status:         inboundmessage.StatusInReview,
			Classification: inboundmessage.ClassificationInvoice,
			MailboxID:      intake,
			Parked:         inboundmessage.ParkingAwaitingReply,
			Count:          1,
		},
		{Status: inboundmessage.StatusQuarantined, MailboxID: intake, Count: 1},
		{Status: inboundmessage.StatusProcessing, MailboxID: intake, Count: 5},
		{Status: inboundmessage.StatusClassified, MailboxID: intake, Count: 1},
		{Status: inboundmessage.StatusActioned, Classification: inboundmessage.ClassificationTender,
			MailboxID: intake, Count: 7},
		{Status: inboundmessage.StatusIgnored, MailboxID: intake, Count: 2},
	})

	assert.Equal(t, 6, counts.Waiting, "parked messages are not waiting")
	assert.Equal(t, 4, counts.Snoozed)
	assert.Equal(t, 1, counts.WaitingOnThem)
	assert.Equal(t, 2, counts.AssignedToMe, "a snoozed message of mine is not waiting on me")
	assert.Equal(t, 2, counts.Overdue, "a snoozed message is not overdue")
	assert.Equal(t, 6, counts.Processing)
	assert.Equal(t, 7, counts.Handled)
	assert.Equal(t, 2, counts.Ignored)
	assert.Equal(t, 1, counts.Quarantined)
	assert.Equal(t, 26, counts.Total)

	for _, row := range counts.ByClassification {
		if row.Classification == inboundmessage.ClassificationTender {
			assert.Equal(t, 16, row.Total)
			assert.Equal(t, 5, row.Waiting, "a kind's waiting leaves the parked ones out")
		}
	}
	require.Len(t, counts.ByMailbox, 1)
	assert.Equal(t, 6, counts.ByMailbox[0].Waiting)
}
