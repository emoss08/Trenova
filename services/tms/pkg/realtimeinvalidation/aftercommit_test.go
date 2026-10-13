package realtimeinvalidation

import (
	"context"

	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type fakeRecord struct {
	id, org, bu pulid.ID
}

func (r fakeRecord) GetID() pulid.ID             { return r.id }
func (r fakeRecord) GetOrganizationID() pulid.ID { return r.org }
func (r fakeRecord) GetBusinessUnitID() pulid.ID { return r.bu }

type capturedRealtime struct {
	agentports.RealtimeService

	published []*agentports.PublishResourceInvalidationRequest
}

func (c *capturedRealtime) PublishResourceInvalidation(
	_ context.Context,
	req *agentports.PublishResourceInvalidationRequest,
) error {
	c.published = append(c.published, req)
	return nil
}

func TestPublishAfterCommitSendsTheChangeOutsideATransaction(t *testing.T) {
	t.Parallel()

	record := fakeRecord{id: pulid.MustNew("inv_"), org: pulid.MustNew("org_"), bu: pulid.MustNew("bu_")}
	actor := kernelports.AuditActor{UserID: pulid.MustNew("usr_")}
	realtime := &capturedRealtime{}

	PublishAfterCommit(t.Context(), realtime, nil, ChangeOf("invoice", record, "send.updated", actor, map[string]string{"number": "INV-1"}))

	require.Len(t, realtime.published, 1)
	got := realtime.published[0]
	require.Equal(t, "invoice", got.Resource)
	require.Equal(t, "send.updated", got.Action)
	require.Equal(t, record.id, got.RecordID)
	require.Equal(t, record.org, got.OrganizationID)
	require.Equal(t, actor.UserID, got.ActorUserID)
}

func TestPublishAfterCommitIgnoresMissingPieces(t *testing.T) {
	t.Parallel()

	realtime := &capturedRealtime{}
	PublishAfterCommit(t.Context(), nil, nil, &RecordChange{RecordID: pulid.MustNew("inv_")})
	PublishAfterCommit(t.Context(), realtime, nil, nil)
	PublishAfterCommit(t.Context(), realtime, nil, &RecordChange{})
	require.Empty(t, realtime.published)
}
