package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
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

func published(title, body string, revises pulid.ID) agentruntimeports.ToolObservation {
	return agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{ID: "call_publish", Name: "publish_artifact"},
		Data: agentruntimeports.PublishedDocument{Title: title, Body: body, ArtifactID: revises},
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

// Revising a document keeps the text it replaced: the revision is the next
// version of the same lineage, with the link and pin it already had, so the
// pane holds one brief whose earlier drafts can still be read and restored.
func TestArtifactRecorder_RevisesADocumentAsItsNextVersion(t *testing.T) {
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
		Slug:             "brief",
		LineageSeq:       1,
		Pinned:           true,
	}
	repo.stored = map[pulid.ID]*assistantartifact.Artifact{existing.ID: existing}

	shown, err := recorder.observe(published("Brief, revised", "# Revised", existing.ID))
	require.NoError(t, err)

	require.Len(t, repo.upserts, 1)
	kept := repo.upserts[0]
	assert.NotEqual(t, existing.ID, kept.ID, "the first draft stays as it was")
	assert.Equal(t, existing.ID, kept.LineageID)
	assert.Equal(t, 2, kept.LineageSeq)
	assert.Equal(t, "brief", kept.Slug)
	assert.Equal(t, "call_publish", kept.SourceToolCallID)
	assert.True(t, kept.Pinned)
	assert.Equal(t, "# Revised", kept.Payload["body"])
	assert.Equal(t, "Revised", kept.Payload["versionNote"])
	assert.Equal(t, kept.ID, shown.ID)
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

// Asked to publish an analysis, a model retyped its figures into the text. A
// document that names the analysis gets it added after its text from what the
// analysis kept, so the figures in it are the analysis's own.
func TestArtifactRecorder_PublishesTheAnalysesADocumentNames(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder, thread := documentRecorder(t, repo)
	kept := &assistantartifact.Artifact{
		ID:       pulid.MustNew("art_"),
		ThreadID: thread.ID,
		Kind:     assistantartifact.KindAnalysis,
		Title:    "Lane performance, 90 days",
		Payload: map[string]any{
			"figures": []any{map[string]any{
				"label": "Revenue", "value": 12517479.24, "kind": "money", "currency": "USD",
			}},
		},
	}
	repo.stored = map[pulid.ID]*assistantartifact.Artifact{kept.ID: kept}
	document := published("Lane brief", "Revenue is spread thin.", pulid.Nil)
	brief, ok := document.Data.(agentruntimeports.PublishedDocument)
	require.True(t, ok)
	brief.Analyses = []pulid.ID{kept.ID}
	document.Data = brief

	_, err := recorder.observe(document)
	require.NoError(t, err)

	require.Len(t, repo.upserts, 1)
	assert.Equal(t, "Revenue is spread thin.\n\n## Lane performance, 90 days\n\n"+
		"- **Revenue:** $12,517,479.24\n", repo.upserts[0].Payload["body"])
}

// Only an analysis of this conversation is added: another thread's, or an
// artifact that is not an analysis, is refused and nothing is kept.
func TestArtifactRecorder_RefusesAnAnalysisFromElsewhere(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder, thread := documentRecorder(t, repo)
	elsewhere := &assistantartifact.Artifact{
		ID: pulid.MustNew("art_"), ThreadID: pulid.MustNew("athr_"),
		Kind: assistantartifact.KindAnalysis,
	}
	table := &assistantartifact.Artifact{
		ID: pulid.MustNew("art_"), ThreadID: thread.ID, Kind: assistantartifact.KindTableView,
	}
	repo.stored = map[pulid.ID]*assistantartifact.Artifact{elsewhere.ID: elsewhere, table.ID: table}

	for _, target := range []pulid.ID{elsewhere.ID, table.ID, pulid.MustNew("art_")} {
		document := published("Brief", "# Brief", pulid.Nil)
		brief, ok := document.Data.(agentruntimeports.PublishedDocument)
		require.True(t, ok)
		brief.Analyses = []pulid.ID{target}
		document.Data = brief

		shown, err := recorder.observe(document)
		require.ErrorIs(t, err, errUnknownAnalysis)
		assert.Nil(t, shown)
	}
	assert.Empty(t, repo.upserts)
}
