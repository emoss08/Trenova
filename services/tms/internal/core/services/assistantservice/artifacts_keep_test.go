package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type deletingArtifactRepo struct {
	*stubArtifactRepo

	deleted []pulid.ID
}

func (r *deletingArtifactRepo) Delete(_ context.Context, req *repositories.DeleteArtifactsRequest) error {
	r.deleted = append(r.deleted, req.IDs...)

	return nil
}

func keepRecorder(t *testing.T, repo repositories.AssistantArtifactRepository) *artifactRecorder {
	t.Helper()
	svc := &Service{logger: zap.NewNop(), artifacts: repo}

	return svc.newArtifactRecorder(
		t.Context(),
		&conversation.Thread{ID: pulid.MustNew("thr_")},
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		testActor(),
		nil,
	)
}

// A lookup's card stays only when the reply points to it; what the agent
// wrote on purpose stays whether or not it is named.
func TestArtifactRecorder_KeepsOnlyTheLookupsTheReplyPointsTo(t *testing.T) {
	t.Parallel()

	repo := &deletingArtifactRepo{stubArtifactRepo: &stubArtifactRepo{}}
	recorder := keepRecorder(t, repo)
	used, _ := recorder.observe(observation("get_customer", map[string]any{"id": "cus_1", "name": "Acme"}))
	unused, _ := recorder.observe(observation("get_worker", map[string]any{"id": "wrk_1", "name": "Dana"}))
	document, _ := recorder.observe(observation("publish_artifact", serviceports.PublishedDocument{
		Title: "Brief", Body: "# Brief",
	}))
	require.NotNil(t, used)
	require.NotNil(t, unused)
	require.NotNil(t, document)

	reply := []conversation.Message{
		{Role: conversation.RoleUser, Content: "[Not a reply](artifact:" + unused.ID.String() + ")"},
		{Role: conversation.RoleAssistant, Content: "Acme is on [Customer Acme](artifact:" + used.ID.String() + ")."},
	}
	recorder.keepLinked(linkedArtifacts(reply))

	assert.Equal(t, []pulid.ID{unused.ID}, repo.deleted)
	kept := map[pulid.ID]bool{}
	for _, artifact := range recorder.artifacts() {
		kept[artifact.ID] = true
	}
	assert.True(t, kept[used.ID])
	assert.True(t, kept[document.ID])
	assert.False(t, kept[unused.ID])
}

// Running the same lookup over data that has not changed points to the last
// version rather than stacking an identical one on it.
func TestArtifactRecorder_AnUnchangedLookupReusesItsLastVersion(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder := keepRecorder(t, repo)
	first, _ := recorder.observe(serviceports.ToolObservation{
		Call: serviceports.ToolCall{ID: "call_1", Name: "get_customer", Arguments: map[string]any{"id": "cus_1"}},
		Data: map[string]any{"id": "cus_1", "name": "Acme"},
	})
	again, _ := recorder.observe(serviceports.ToolObservation{
		Call: serviceports.ToolCall{ID: "call_2", Name: "get_customer", Arguments: map[string]any{"id": "cus_1"}},
		Data: map[string]any{"id": "cus_1", "name": "Acme"},
	})
	changed, _ := recorder.observe(serviceports.ToolObservation{
		Call: serviceports.ToolCall{ID: "call_3", Name: "get_customer", Arguments: map[string]any{"id": "cus_1"}},
		Data: map[string]any{"id": "cus_1", "name": "Acme Manufacturing"},
	})

	require.NotNil(t, first)
	require.NotNil(t, again)
	require.NotNil(t, changed)
	assert.Equal(t, first.ID, again.ID)
	assert.NotEqual(t, first.ID, changed.ID)
	assert.Len(t, repo.upserts, 2)
	assert.Equal(t, assistantartifact.KindEntityCard, repo.upserts[1].Kind)
	assert.Equal(t, 2, repo.upserts[1].LineageSeq)
}
