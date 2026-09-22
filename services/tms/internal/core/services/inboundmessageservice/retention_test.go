package inboundmessageservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type retentionRepo struct {
	repositories.InboundMessageRepository

	settled []*inboundmessage.InboundMessage
	listed  *repositories.ListSettledInboundMessagesRequest
	deleted *repositories.DeleteInboundMessagesRequest
}

func (r *retentionRepo) ListSettledBefore(
	_ context.Context,
	req repositories.ListSettledInboundMessagesRequest,
) ([]*inboundmessage.InboundMessage, error) {
	r.listed = &req
	return r.settled, nil
}

func (r *retentionRepo) DeleteByIDs(
	_ context.Context,
	req repositories.DeleteInboundMessagesRequest,
) (int, error) {
	r.deleted = &req
	return len(req.IDs), nil
}

type deletingStorage struct {
	storage.Client

	refuse  map[string]bool
	removed []string
}

func (s *deletingStorage) Delete(_ context.Context, key string) error {
	if s.refuse[key] {
		return errors.New("storage unavailable")
	}
	s.removed = append(s.removed, key)

	return nil
}

func TestPurgeSettled_RemovesTheStoredCopyBeforeTheRow(t *testing.T) {
	t.Parallel()

	kept := &inboundmessage.InboundMessage{ID: pulid.MustNew("imsg_"), HTMLKey: "inbound/kept.eml"}
	gone := &inboundmessage.InboundMessage{ID: pulid.MustNew("imsg_"), HTMLKey: "inbound/gone.eml"}
	bare := &inboundmessage.InboundMessage{ID: pulid.MustNew("imsg_")}
	repo := &retentionRepo{settled: []*inboundmessage.InboundMessage{kept, gone, bare}}
	store := &deletingStorage{refuse: map[string]bool{"inbound/kept.eml": true}}
	svc := &Service{l: zap.NewNop(), messageRepo: repo, storage: store}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	deleted, err := svc.PurgeSettled(t.Context(), PurgeSettledRequest{
		TenantInfo: tenant,
		Before:     1_700_000_000,
		Limit:      100,
	})
	require.NoError(t, err)

	assert.Equal(t, 2, deleted)
	assert.Equal(t, []string{"inbound/gone.eml"}, store.removed)
	require.NotNil(t, repo.deleted)
	assert.Equal(t, tenant, repo.deleted.TenantInfo)
	assert.ElementsMatch(t, []pulid.ID{gone.ID, bare.ID}, repo.deleted.IDs,
		"a message whose copy could not be removed keeps its row for the next pass")
	assert.Equal(t, int64(1_700_000_000), repo.listed.Before)
	assert.Equal(t, 100, repo.listed.Limit)
}

func TestPurgeSettled_DoesNothingWhenNothingIsOldEnough(t *testing.T) {
	t.Parallel()

	repo := &retentionRepo{}
	svc := &Service{l: zap.NewNop(), messageRepo: repo, storage: &deletingStorage{}}

	deleted, err := svc.PurgeSettled(t.Context(), PurgeSettledRequest{Limit: 100})
	require.NoError(t, err)

	assert.Zero(t, deleted)
	assert.Nil(t, repo.deleted)
}
