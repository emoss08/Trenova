package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The recorder keeps what a tool produced as the tool finishes and tells the
// stream, so the pane can open the table while the reply is still arriving;
// once the turn is saved it ties each artifact to the message that made it.
func TestArtifactRecorder_KeepsAndAnnouncesWhatATurnProduced(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	var events []kernelports.StreamEvent
	svc := &Service{logger: zap.NewNop(), artifacts: repo}
	thread := &conversation.Thread{ID: pulid.MustNew("thr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	recorder := svc.newArtifactRecorder(
		t.Context(),
		thread,
		tenant,
		assistanttest.Actor(),
		func(event kernelports.StreamEvent) {
			events = append(events, event)
		},
	)

	recorder.Observe(
		assistanttest.Observation("customer_get", map[string]any{"id": "cus_1", "name": "Acme"}),
	)
	recorder.Observe(assistanttest.Observation("customer_list", map[string]any{"rows": []any{}}))

	require.Len(t, repo.Upserts, 1)
	assert.Equal(t, thread.ID, repo.Upserts[0].ThreadID)
	assert.Equal(t, tenant.OrgID, repo.Upserts[0].OrganizationID)
	require.Len(t, events, 1)
	assert.Equal(t, assistantstreamports.AssistantEventArtifact, events[0].Event)
	announced, ok := events[0].Data.(assistantstreamports.AssistantArtifactEvent)
	require.True(t, ok)
	assert.Equal(t, assistantartifact.KindEntityCard, announced.Kind)
	assert.Equal(t, "call_customer_get", announced.SourceToolCallID)

	messageID := pulid.MustNew("msg_")
	recorder.AttachMessages(map[string]pulid.ID{"call_customer_get": messageID})
	require.Len(t, repo.Upserts, 2)
	assert.Equal(t, messageID, repo.Upserts[1].MessageID)

	listed := recorder.Artifacts()
	require.Len(t, listed, 1)
	assert.Equal(t, messageID, listed[0].MessageID)
}

// The app follows a navigation the moment it is announced, before it has
// fetched the artifact, so the announcement carries where to go. Nothing
// else does: a card's link is for a person to choose.
func TestArtifactRecorder_AnnouncesWhereANavigationGoes(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	var events []kernelports.StreamEvent
	svc := &Service{logger: zap.NewNop(), artifacts: repo}
	thread := &conversation.Thread{ID: pulid.MustNew("thr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	recorder := svc.newArtifactRecorder(
		t.Context(),
		thread,
		tenant,
		assistanttest.Actor(),
		func(event kernelports.StreamEvent) {
			events = append(events, event)
		},
	)

	recorder.Observe(assistanttest.Observation("agent_open_page", map[string]any{
		"path": "/billing/invoices",
		"name": "Invoices",
	}))
	recorder.Observe(
		assistanttest.Observation(
			"shipment_get",
			map[string]any{"id": "shp_1", "proNumber": "PRO-1"},
		),
	)

	require.Len(t, events, 2)
	moved, ok := events[0].Data.(assistantstreamports.AssistantArtifactEvent)
	require.True(t, ok)
	assert.Equal(t, assistantartifact.KindNavigation, moved.Kind)
	assert.Equal(t, "/billing/invoices", moved.Path)

	card, ok := events[1].Data.(assistantstreamports.AssistantArtifactEvent)
	require.True(t, ok)
	assert.Empty(t, card.Path)
	// The move is announced and not kept: only the card was saved.
	require.Len(t, repo.Upserts, 1)
	assert.Equal(t, assistantartifact.KindEntityCard, repo.Upserts[0].Kind)
	require.Len(t, recorder.Artifacts(), 1)
}

// A nil recorder is a turn without a pane: every call is a no-op and the
// runtime gets no observer to call.
func TestArtifactRecorder_NilIsSafe(t *testing.T) {
	t.Parallel()

	var recorder *artifactrecorder.Recorder
	assert.Nil(t, recorder.Observer())
	recorder.FromProposals(nil, nil)
	recorder.AttachMessages(nil)
	assert.Nil(t, recorder.Artifacts())

	svc := &Service{logger: zap.NewNop()}
	assert.Nil(
		t,
		svc.newArtifactRecorder(
			t.Context(),
			&conversation.Thread{},
			pagination.TenantInfo{},
			assistanttest.Actor(),
			nil,
		),
	)
}

// The same list read again later in the conversation is the next version of
// the table read before, so the pane offers one table with its history.
func TestArtifactRecorder_ReadingTheSameSourceAgainMakesANewVersion(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	svc := &Service{logger: zap.NewNop(), artifacts: repo}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	recorder := svc.newArtifactRecorder(
		t.Context(),
		&conversation.Thread{ID: pulid.MustNew("athr_")},
		tenant,
		assistanttest.Actor(),
		nil,
	)
	rows := map[string]any{
		"items": []any{
			map[string]any{"id": "bqi_1", "number": "BQ-1", "status": "ReadyForReview"},
		},
		"columns": []any{"number", "status"},
		"count":   1,
	}
	// The second read finds the item moved on: the same source read again
	// over changed data is a new version. Over unchanged data it is the same
	// view, and the reply points to the one already kept.
	moved := map[string]any{
		"items":   []any{map[string]any{"id": "bqi_1", "number": "BQ-1", "status": "InReview"}},
		"columns": []any{"number", "status"},
		"count":   1,
	}
	read := func(callID string, status string, data map[string]any) agentruntimeports.ToolObservation {
		return agentruntimeports.ToolObservation{
			Call: llmports.ToolCall{
				ID:        callID,
				Name:      "billing_list_queue_items",
				Arguments: map[string]any{"status": status},
			},
			Data: data,
		}
	}

	_, err := recorder.Observe(read("call_1", "Ready", rows))
	require.NoError(t, err)
	_, err = recorder.Observe(read("call_2", "Ready", moved))
	require.NoError(t, err)
	_, err = recorder.Observe(read("call_3", "Posted", rows))
	require.NoError(t, err)

	require.Len(t, repo.Upserts, 3)
	first, second, third := repo.Upserts[0], repo.Upserts[1], repo.Upserts[2]
	assert.NotEmpty(t, first.LineageKey)
	assert.True(t, first.LineageID.IsNil())
	assert.Equal(t, first.ID, second.LineageID)
	assert.Equal(t, 2, second.LineageSeq)
	// Read again with another filter, it is still the billing queue: the
	// next version of the same table, not a second one.
	assert.Equal(t, first.ID, third.LineageID)
	assert.Equal(t, 3, third.LineageSeq)
}
