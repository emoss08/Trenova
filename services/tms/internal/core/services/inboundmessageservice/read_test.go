package inboundmessageservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type readStubMessageRepo struct {
	repositories.InboundMessageRepository

	listed    *repositories.ListInboundMessagesRequest
	breakdown []repositories.InboundMessageCount
}

func (s *readStubMessageRepo) ListCursor(
	_ context.Context,
	req *repositories.ListInboundMessagesRequest,
) (*pagination.CursorListResult[*inboundmessage.InboundMessage], error) {
	s.listed = req

	return pagination.NewCursorListResultWithTotalCount(
		[]*inboundmessage.InboundMessage{}, 25, nil,
	), nil
}

func (s *readStubMessageRepo) CountBreakdown(
	_ context.Context,
	_ repositories.CountInboundMessagesRequest,
) ([]repositories.InboundMessageCount, error) {
	return s.breakdown, nil
}

/*
An inbox is read newest arrival first. The cursor's own default is the row's
creation time, which is when the pipeline wrote it — a message a provider
redelivered an hour late would otherwise sit above one that arrived after it.
*/
func TestListOrdersTheInboxByArrivalNewestFirst(t *testing.T) {
	t.Parallel()

	repo := &readStubMessageRepo{}
	svc := &Service{messageRepo: repo}

	_, err := svc.List(t.Context(), &repositories.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{},
	})
	require.NoError(t, err)

	require.NotNil(t, repo.listed)
	assert.Equal(t,
		[]domaintypes.SortField{{Field: "receivedAt", Direction: dbtype.SortDirectionDesc}},
		repo.listed.Filter.Sort,
	)
}

func TestListTrimsTheSearchAndDropsOneThatIsOnlySpace(t *testing.T) {
	t.Parallel()

	repo := &readStubMessageRepo{}
	svc := &Service{messageRepo: repo}

	_, err := svc.List(t.Context(), &repositories.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{Query: "  detention  "},
	})
	require.NoError(t, err)
	assert.Equal(t, "detention", repo.listed.Filter.Query)

	_, err = svc.List(t.Context(), &repositories.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{Query: "   "},
	})
	require.NoError(t, err)
	assert.Empty(t, repo.listed.Filter.Query)
}

func TestCountsFoldsTheBreakdownIntoLanesKindsAndMailboxes(t *testing.T) {
	t.Parallel()

	intake := pulid.MustNew("imbx_")
	billing := pulid.MustNew("imbx_")

	repo := &readStubMessageRepo{breakdown: []repositories.InboundMessageCount{
		{Status: inboundmessage.StatusInReview, Classification: inboundmessage.ClassificationTender, MailboxID: intake, Count: 3},
		{Status: inboundmessage.StatusActioned, Classification: inboundmessage.ClassificationTender, MailboxID: intake, Count: 5},
		{Status: inboundmessage.StatusQuarantined, Classification: "", MailboxID: intake, Count: 2},
		{Status: inboundmessage.StatusActioned, Classification: inboundmessage.ClassificationInvoice, MailboxID: billing, Count: 4},
		{Status: inboundmessage.StatusIgnored, Classification: inboundmessage.ClassificationOther, MailboxID: billing, Count: 1},
	}}
	svc := &Service{messageRepo: repo}

	counts, err := svc.Counts(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Equal(t, 5, counts.Waiting, "in review and held back are both waiting")
	assert.Equal(t, 9, counts.Handled)
	assert.Equal(t, 1, counts.Ignored)
	assert.Equal(t, 2, counts.Quarantined)
	assert.Equal(t, 15, counts.Total)

	byKind := make(map[inboundmessage.Classification]ClassificationCount, len(counts.ByClassification))
	for _, row := range counts.ByClassification {
		byKind[row.Classification] = row
	}
	assert.Len(t, counts.ByClassification, len(inboundmessage.AllClassifications()),
		"every kind is listed, so an empty one reads as empty rather than missing")
	assert.Equal(t, ClassificationCount{
		Classification: inboundmessage.ClassificationTender, Total: 8, Waiting: 3,
	}, byKind[inboundmessage.ClassificationTender])
	assert.Equal(t, ClassificationCount{
		Classification: inboundmessage.ClassificationInvoice, Total: 4, Waiting: 0,
	}, byKind[inboundmessage.ClassificationInvoice])
	assert.Zero(t, byKind[inboundmessage.ClassificationProofOfDelivery].Total)

	byMailbox := make(map[pulid.ID]MailboxCount, len(counts.ByMailbox))
	for _, row := range counts.ByMailbox {
		byMailbox[row.MailboxID] = row
	}
	assert.Equal(t, MailboxCount{MailboxID: intake, Total: 10, Waiting: 5}, byMailbox[intake])
	assert.Equal(t, MailboxCount{MailboxID: billing, Total: 5, Waiting: 0}, byMailbox[billing])
}
