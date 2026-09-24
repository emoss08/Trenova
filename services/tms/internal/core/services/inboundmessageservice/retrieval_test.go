package inboundmessageservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeIndexer struct {
	err    error
	marked []pulid.ID
	types  []airetrieval.SourceType
}

func (f *fakeIndexer) MarkStale(
	_ context.Context,
	_ pagination.TenantInfo,
	sourceType airetrieval.SourceType,
	ids ...pulid.ID,
) error {
	f.marked = append(f.marked, ids...)
	f.types = append(f.types, sourceType)

	return f.err
}

func (f *fakeIndexer) DeleteSource(
	context.Context,
	pagination.TenantInfo,
	airetrieval.SourceType,
	pulid.ID,
) error {
	return nil
}

func (f *fakeIndexer) Reindex(
	context.Context,
	pagination.TenantInfo,
	airetrieval.SourceType,
) error {
	return nil
}

func TestProcessMessage_QueuesTheSettledMessageForSemanticIndexing(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{
		message: staged(inboundmessage.StatusReceived, inboundmessage.ReviewAutoHandle),
	}
	svc := settler(
		repo,
		`{"classification":"Tender","confidence":0.9,"reasoning":"Offers a load."}`,
	)
	indexer := &fakeIndexer{}
	svc.indexer = indexer

	_, err := svc.ProcessMessage(t.Context(), repo.message.ID, pagination.TenantInfo{})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{repo.message.ID}, indexer.marked)
	assert.Equal(t, []airetrieval.SourceType{airetrieval.SourceTypeInboundMessage}, indexer.types)
}

func TestProcessMessage_DoesNotQueueAMessageItLeftAlone(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{message: staged(inboundmessage.StatusActioned, inboundmessage.ReviewAlways)}
	svc := settler(repo, "")
	indexer := &fakeIndexer{}
	svc.indexer = indexer

	_, err := svc.ProcessMessage(t.Context(), repo.message.ID, pagination.TenantInfo{})
	require.NoError(t, err)
	assert.Empty(t, indexer.marked)
}

func TestMarkFailed_QueuesTheMessageAndSurvivesAnIndexingFailure(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{
		message: staged(inboundmessage.StatusReceived, inboundmessage.ReviewAutoHandle),
	}
	svc := settler(repo, "")
	indexer := &fakeIndexer{err: errors.New("outbox unavailable")}
	svc.indexer = indexer

	require.NoError(t, svc.MarkFailed(
		t.Context(), repo.message.ID, pagination.TenantInfo{},
		"SETTLE_FAILED", "the provider never answered",
	))
	assert.Equal(t, []pulid.ID{repo.message.ID}, indexer.marked)
	require.NotNil(t, repo.saved)
}
