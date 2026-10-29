package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/toolartifacts"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type deletingArtifactRepo struct {
	*assistanttest.StubArtifactRepo

	deleted []pulid.ID
}

func (r *deletingArtifactRepo) Delete(
	_ context.Context,
	req *assistantartifactports.DeleteArtifactsRequest,
) error {
	r.deleted = append(r.deleted, req.IDs...)

	return nil
}

func keepRecorder(
	t *testing.T,
	repo assistantartifactports.AssistantArtifactRepository,
) *artifactrecorder.Recorder {
	t.Helper()
	svc := &Service{logger: zap.NewNop(), artifacts: repo}

	return svc.newArtifactRecorder(
		t.Context(),
		&conversation.Thread{ID: pulid.MustNew("thr_")},
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		assistanttest.Actor(),
		nil,
	)
}

// A lookup's card stays only when the reply points to it; what the agent
// wrote on purpose stays whether or not it is named.
func TestArtifactRecorder_KeepsOnlyTheLookupsTheReplyPointsTo(t *testing.T) {
	t.Parallel()

	repo := &deletingArtifactRepo{StubArtifactRepo: &assistanttest.StubArtifactRepo{}}
	recorder := keepRecorder(t, repo)
	used, _ := recorder.Observe(
		assistanttest.Observation("customer_get", map[string]any{"id": "cus_1", "name": "Acme"}),
	)
	unused, _ := recorder.Observe(
		assistanttest.Observation("worker_get", map[string]any{"id": "wrk_1", "name": "Dana"}),
	)
	document, _ := recorder.Observe(
		assistanttest.Observation("publish_artifact", agentruntimeports.PublishedDocument{
			Title: "Brief", Body: "# Brief",
		}),
	)
	require.NotNil(t, used)
	require.NotNil(t, unused)
	require.NotNil(t, document)

	reply := []conversation.Message{
		{
			Role:    conversation.RoleUser,
			Content: "[Not a reply](artifact:" + unused.ID.String() + ")",
		},
		{
			Role:    conversation.RoleAssistant,
			Content: "Acme is on [Customer Acme](artifact:" + used.ID.String() + ").",
		},
	}
	recorder.KeepLinked(toolartifacts.LinkedArtifacts(reply))

	assert.Equal(t, []pulid.ID{unused.ID}, repo.deleted)
	kept := map[pulid.ID]bool{}
	for _, artifact := range recorder.Artifacts() {
		kept[artifact.ID] = true
	}
	assert.True(t, kept[used.ID])
	assert.True(t, kept[document.ID])
	assert.False(t, kept[unused.ID])
}

// An agent the conversation handed a task to cites records in its report to
// that agent. They are its working: a card stays only if the conversation's
// own reply points to it too.
func TestArtifactRecorder_ADelegatesCitationsDoNotKeepItsLookups(t *testing.T) {
	t.Parallel()

	repo := &deletingArtifactRepo{StubArtifactRepo: &assistanttest.StubArtifactRepo{}}
	recorder := keepRecorder(t, repo)
	cited, _ := recorder.Observe(
		assistanttest.Observation(
			"shipment_get",
			map[string]any{"id": "shp_1", "proNumber": "P-1"},
		),
	)
	both, _ := recorder.Observe(
		assistanttest.Observation("customer_get", map[string]any{"id": "cus_1", "name": "Acme"}),
	)
	require.NotNil(t, cited)
	require.NotNil(t, both)

	reply := []conversation.Message{
		{
			Role:           conversation.RoleAssistant,
			DelegateCallID: "call_delegate",
			Content: "See [shipment P-1](artifact:" + cited.ID.String() + ") and " +
				"[Acme](artifact:" + both.ID.String() + ").",
		},
		{
			Role:    conversation.RoleAssistant,
			Content: "Acme is on [Customer Acme](artifact:" + both.ID.String() + ").",
		},
	}
	recorder.KeepLinked(toolartifacts.LinkedArtifacts(reply))

	assert.Equal(t, []pulid.ID{cited.ID}, repo.deleted)
}

// Running the same lookup over data that has not changed points to the last
// version rather than stacking an identical one on it.
func TestArtifactRecorder_AnUnchangedLookupReusesItsLastVersion(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	recorder := keepRecorder(t, repo)
	first, _ := recorder.Observe(agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{
			ID:        "call_1",
			Name:      "customer_get",
			Arguments: map[string]any{"id": "cus_1"},
		},
		Data: map[string]any{"id": "cus_1", "name": "Acme"},
	})
	again, _ := recorder.Observe(agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{
			ID:        "call_2",
			Name:      "customer_get",
			Arguments: map[string]any{"id": "cus_1"},
		},
		Data: map[string]any{"id": "cus_1", "name": "Acme"},
	})
	changed, _ := recorder.Observe(agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{
			ID:        "call_3",
			Name:      "customer_get",
			Arguments: map[string]any{"id": "cus_1"},
		},
		Data: map[string]any{"id": "cus_1", "name": "Acme Manufacturing"},
	})

	require.NotNil(t, first)
	require.NotNil(t, again)
	require.NotNil(t, changed)
	assert.Equal(t, first.ID, again.ID)
	assert.NotEqual(t, first.ID, changed.ID)
	assert.Len(t, repo.Upserts, 2)
	assert.Equal(t, assistantartifact.KindEntityCard, repo.Upserts[1].Kind)
	assert.Equal(t, 2, repo.Upserts[1].LineageSeq)
}
