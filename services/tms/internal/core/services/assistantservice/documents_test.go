package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func documentRecorder(
	t *testing.T,
	repo *stubArtifactRepo,
) (*artifactRecorder, *conversation.Thread) {
	t.Helper()

	svc := &Service{logger: zap.NewNop(), artifacts: repo}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	return svc.newArtifactRecorder(t.Context(), thread, tenant, testActor(), nil), thread
}

func published(title, body string, revises pulid.ID) serviceports.ToolObservation {
	return serviceports.ToolObservation{
		Call: serviceports.ToolCall{ID: "call_publish", Name: "publish_artifact"},
		Data: serviceports.PublishedDocument{Title: title, Body: body, ArtifactID: revises},
	}
}

// A published document is kept as markdown under its title, and the runtime
// is told what the person now sees.
func TestArtifactRecorder_KeepsAPublishedDocument(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder, thread := documentRecorder(t, repo)

	shown, err := recorder.observe(published("SEED-SHP-007 brief", "# SEED-SHP-007", pulid.Nil))
	require.NoError(t, err)
	require.NotNil(t, shown)

	require.Len(t, repo.upserts, 1)
	kept := repo.upserts[0]
	assert.Equal(t, assistantartifact.KindDocument, kept.Kind)
	assert.Equal(t, thread.ID, kept.ThreadID)
	assert.Equal(t, "SEED-SHP-007 brief", kept.Title)
	assert.Equal(t, "# SEED-SHP-007", kept.Payload["body"])
	assert.Equal(t, "markdown", kept.Payload["format"])
	assert.Equal(t, kept.ID, shown.ID)
	assert.Equal(t, "document", shown.Kind)
}

// Revising a document replaces its text in place: the same artifact, with the
// message and pin it already had, so the pane holds one brief rather than a
// stack of drafts of it.
func TestArtifactRecorder_RevisesADocumentInPlace(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder, thread := documentRecorder(t, repo)
	existing := &assistantartifact.Artifact{
		ID:               pulid.MustNew("art_"),
		ThreadID:         thread.ID,
		MessageID:        pulid.MustNew("amsg_"),
		Kind:             assistantartifact.KindDocument,
		Title:            "Brief",
		SourceToolCallID: "call_first",
		Pinned:           true,
	}
	repo.stored = map[pulid.ID]*assistantartifact.Artifact{existing.ID: existing}

	shown, err := recorder.observe(published("Brief, revised", "# Revised", existing.ID))
	require.NoError(t, err)

	require.Len(t, repo.upserts, 1)
	kept := repo.upserts[0]
	assert.Equal(t, existing.ID, kept.ID)
	assert.Equal(t, existing.MessageID, kept.MessageID)
	assert.Equal(t, "call_first", kept.SourceToolCallID)
	assert.True(t, kept.Pinned)
	assert.Equal(t, "# Revised", kept.Payload["body"])
	assert.Equal(t, existing.ID, shown.ID)
}

// A revision may only reach this conversation's own documents. An id from
// another thread, or one that names a table rather than a document, is
// refused with a reason the model can act on, and nothing is written.
func TestArtifactRecorder_RefusesARevisionOfSomethingElse(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder, thread := documentRecorder(t, repo)
	elsewhere := &assistantartifact.Artifact{
		ID: pulid.MustNew(
			"art_",
		), ThreadID: pulid.MustNew("athr_"), Kind: assistantartifact.KindDocument,
	}
	table := &assistantartifact.Artifact{
		ID: pulid.MustNew("art_"), ThreadID: thread.ID, Kind: assistantartifact.KindTableView,
	}
	repo.stored = map[pulid.ID]*assistantartifact.Artifact{elsewhere.ID: elsewhere, table.ID: table}

	for _, target := range []pulid.ID{elsewhere.ID, table.ID, pulid.MustNew("art_")} {
		shown, err := recorder.observe(published("Brief", "# Brief", target))
		require.ErrorIs(t, err, errUnknownDocument)
		assert.Nil(t, shown)
	}
	assert.Empty(t, repo.upserts)
}
