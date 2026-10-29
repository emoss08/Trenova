package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func boardRecorder(
	t *testing.T,
	repo *assistanttest.StubArtifactRepo,
	boards assistantports.BoardFiller,
) (*artifactrecorder.Recorder, *conversation.Thread) {
	t.Helper()

	svc := &Service{logger: zap.NewNop(), artifacts: repo, boards: boards}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	return svc.newArtifactRecorder(t.Context(), thread, tenant, assistanttest.Actor(), nil), thread
}

func TestArtifactRecorder_KeepsABoardWithWhatItShowedWhenAnswered(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	boards := &assistanttest.StubBoards{}
	recorder, thread := boardRecorder(t, repo, boards)

	shown, err := recorder.Observe(assistanttest.ShownBoard(t, pulid.Nil))
	require.NoError(t, err)
	require.NotNil(t, shown)

	require.Len(t, boards.Checked, 1)
	require.Len(t, boards.Filled, 1)
	assert.Equal(t, thread.ID, boards.Checked[0].ThreadID)
	require.Len(t, repo.Upserts, 1)
	kept := repo.Upserts[0]
	assert.Equal(t, assistantartifact.KindBoard, kept.Kind)
	assert.Equal(t, "Six late shipments", kept.Title)
	assert.Equal(t, "call_board", kept.SourceToolCallID)

	payload, err := board.PayloadOf(kept.Payload)
	require.NoError(t, err)
	assert.Equal(t, "Six late shipments", payload.Board.Title)
	assert.Equal(t, "6", payload.Snapshot.Values[board.FigureKey(0, 0)].Number.String())
	assert.Equal(t, "board", shown.Kind)
	assert.Contains(t, shown.Brief, "- Late now: 6")
	assert.True(t, shown.MustBeLinked())
}

func TestArtifactRecorder_RevisesABoardAsItsNextVersion(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	recorder, thread := boardRecorder(t, repo, &assistanttest.StubBoards{})
	existing := &assistantartifact.Artifact{
		ID: pulid.MustNew("art_"), ThreadID: thread.ID, Kind: assistantartifact.KindBoard,
		Title: "Six late shipments", Slug: "six-late-shipments", LineageSeq: 1, Pinned: true,
	}
	repo.Stored = map[pulid.ID]*assistantartifact.Artifact{existing.ID: existing}

	_, err := recorder.Observe(assistanttest.ShownBoard(t, existing.ID))
	require.NoError(t, err)

	require.Len(t, repo.Upserts, 1)
	kept := repo.Upserts[0]
	assert.Equal(t, existing.ID, kept.LineageID)
	assert.Equal(t, 2, kept.LineageSeq)
	assert.Equal(t, "six-late-shipments", kept.Slug)
	assert.True(t, kept.Pinned)
}

func TestArtifactRecorder_RefusesARevisionOfSomethingThatIsNotABoard(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	recorder, thread := boardRecorder(t, repo, &assistanttest.StubBoards{})
	document := &assistantartifact.Artifact{
		ID: pulid.MustNew("art_"), ThreadID: thread.ID, Kind: assistantartifact.KindDocument,
	}
	repo.Stored = map[pulid.ID]*assistantartifact.Artifact{document.ID: document}

	for _, target := range []pulid.ID{document.ID, pulid.MustNew("art_")} {
		shown, err := recorder.Observe(assistanttest.ShownBoard(t, target))
		require.ErrorIs(t, err, artifactrecorder.ErrUnknownBoard)
		assert.Nil(t, shown)
	}
	assert.Empty(t, repo.Upserts)
}

func TestArtifactRecorder_SaysWhyABoardWasRefused(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	refusal := &agentruntimeports.PublishMistake{Reason: "the person may not read shipments"}
	recorder, _ := boardRecorder(t, repo, &assistanttest.StubBoards{Refuse: refusal})

	_, err := recorder.Observe(assistanttest.ShownBoard(t, pulid.Nil))

	require.ErrorIs(t, err, refusal)
	assert.Empty(t, repo.Upserts)
}

func TestArtifactRecorder_ShowsNoBoardWhereNoneCanBeFilled(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	recorder, _ := boardRecorder(t, repo, nil)

	_, err := recorder.Observe(assistanttest.ShownBoard(t, pulid.Nil))

	require.ErrorIs(t, err, artifactrecorder.ErrNoBoards)
	assert.Empty(t, repo.Upserts)
}

func TestArtifactRecorder_CreditsABoardToTheConversationsAgent(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	repo := &assistanttest.StubArtifactRepo{}
	svc := &Service{
		logger:    zap.NewNop(),
		artifacts: repo,
		boards:    &assistanttest.StubBoards{},
		definitions: &assistanttest.DelegateDefinitions{
			ByID: map[pulid.ID]*agentdefinition.Definition{
				agentID: {ID: agentID, Name: "Dispatch desk"},
			},
		},
	}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	svc.conversations = &assistanttest.StubConversationRepo{
		Thread: &conversation.Thread{ID: thread.ID, AgentDefinitionID: agentID},
	}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	recorder := svc.newArtifactRecorder(t.Context(), thread, tenant, assistanttest.Actor(), nil)

	_, err := recorder.Observe(assistanttest.ShownBoard(t, pulid.Nil))
	require.NoError(t, err)

	require.Len(t, repo.Upserts, 1)
	payload, err := board.PayloadOf(repo.Upserts[0].Payload)
	require.NoError(t, err)
	assert.Equal(t, "Dispatch desk", payload.Author)
}

func TestArtifactRecorder_CreditsTheAssistantWhenTheConversationCannotBeRead(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	svc := &Service{
		logger:      zap.NewNop(),
		artifacts:   repo,
		boards:      &assistanttest.StubBoards{},
		definitions: &assistanttest.DelegateDefinitions{},
		conversations: &assistanttest.StubConversationRepo{
			GetErr: errortypes.NewNotFoundError("gone"),
		},
	}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	recorder := svc.newArtifactRecorder(
		t.Context(),
		&conversation.Thread{ID: pulid.MustNew("athr_")},
		tenant,
		assistanttest.Actor(),
		nil,
	)

	_, err := recorder.Observe(assistanttest.ShownBoard(t, pulid.Nil))
	require.NoError(t, err)

	require.Len(t, repo.Upserts, 1)
	payload, err := board.PayloadOf(repo.Upserts[0].Payload)
	require.NoError(t, err)
	assert.Equal(t, "Assistant", payload.Author)
}
