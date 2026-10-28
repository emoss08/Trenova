package assistantservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubBoards struct {
	checked []*agentports.BoardFillRequest
	filled  []*agentports.BoardFillRequest
	refuse  error
}

func (s *stubBoards) Check(_ context.Context, req *agentports.BoardFillRequest) error {
	s.checked = append(s.checked, req)

	return s.refuse
}

func (s *stubBoards) Fill(
	_ context.Context,
	req *agentports.BoardFillRequest,
) (*board.Filled, error) {
	s.filled = append(s.filled, req)

	return &board.Filled{
		Title:  req.Board.Title,
		ReadAt: 1_793_000_000,
		Blocks: []board.FilledBlock{{
			Type: board.BlockFigures, Numeral: 1, State: board.StateReady,
			Figures: []board.FilledFigure{{
				Label: "Late now",
				Value: &board.Value{Kind: board.ValueCount, Number: decimal.NewFromInt(6)},
			}},
		}},
	}, nil
}

func (s *stubBoards) Card(
	context.Context,
	*agentports.BoardCardRequest,
) (*agentports.BoardCard, error) {
	return nil, nil
}

func (s *stubBoards) Location(context.Context, pagination.TenantInfo) *time.Location {
	return time.UTC
}

func boardRecorder(
	t *testing.T,
	repo *stubArtifactRepo,
	boards agentports.BoardFiller,
) (*artifactRecorder, *conversation.Thread) {
	t.Helper()

	svc := &Service{logger: zap.NewNop(), artifacts: repo, boards: boards}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	return svc.newArtifactRecorder(t.Context(), thread, tenant, testActor(), nil), thread
}

func shownBoard(t *testing.T, revises pulid.ID) agentports.ToolObservation {
	t.Helper()
	parsed, problems := board.Parse(map[string]any{
		"title":   "Six late shipments",
		"verdict": "Four are [held][1].",
		"blocks": []any{map[string]any{"type": "figures", "figures": []any{map[string]any{
			"label":   "Late now",
			"records": []any{pulid.MustNew("shp_").String()},
			"measure": "lateness",
		}}}},
	})
	require.Empty(t, problems)

	return agentports.ToolObservation{
		Call: agentports.ToolCall{ID: "call_board", Name: "show_board"},
		Data: agentports.BoardRequest{Board: *parsed, ArtifactID: revises},
	}
}

func TestArtifactRecorder_KeepsABoardWithWhatItShowedWhenAnswered(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	boards := &stubBoards{}
	recorder, thread := boardRecorder(t, repo, boards)

	shown, err := recorder.observe(shownBoard(t, pulid.Nil))
	require.NoError(t, err)
	require.NotNil(t, shown)

	require.Len(t, boards.checked, 1)
	require.Len(t, boards.filled, 1)
	assert.Equal(t, thread.ID, boards.checked[0].ThreadID)
	require.Len(t, repo.upserts, 1)
	kept := repo.upserts[0]
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

	repo := &stubArtifactRepo{}
	recorder, thread := boardRecorder(t, repo, &stubBoards{})
	existing := &assistantartifact.Artifact{
		ID: pulid.MustNew("art_"), ThreadID: thread.ID, Kind: assistantartifact.KindBoard,
		Title: "Six late shipments", Slug: "six-late-shipments", LineageSeq: 1, Pinned: true,
	}
	repo.stored = map[pulid.ID]*assistantartifact.Artifact{existing.ID: existing}

	_, err := recorder.observe(shownBoard(t, existing.ID))
	require.NoError(t, err)

	require.Len(t, repo.upserts, 1)
	kept := repo.upserts[0]
	assert.Equal(t, existing.ID, kept.LineageID)
	assert.Equal(t, 2, kept.LineageSeq)
	assert.Equal(t, "six-late-shipments", kept.Slug)
	assert.True(t, kept.Pinned)
}

func TestArtifactRecorder_RefusesARevisionOfSomethingThatIsNotABoard(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder, thread := boardRecorder(t, repo, &stubBoards{})
	document := &assistantartifact.Artifact{
		ID: pulid.MustNew("art_"), ThreadID: thread.ID, Kind: assistantartifact.KindDocument,
	}
	repo.stored = map[pulid.ID]*assistantartifact.Artifact{document.ID: document}

	for _, target := range []pulid.ID{document.ID, pulid.MustNew("art_")} {
		shown, err := recorder.observe(shownBoard(t, target))
		require.ErrorIs(t, err, errUnknownBoard)
		assert.Nil(t, shown)
	}
	assert.Empty(t, repo.upserts)
}

func TestArtifactRecorder_SaysWhyABoardWasRefused(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	refusal := &agentports.PublishMistake{Reason: "the person may not read shipments"}
	recorder, _ := boardRecorder(t, repo, &stubBoards{refuse: refusal})

	_, err := recorder.observe(shownBoard(t, pulid.Nil))

	require.ErrorIs(t, err, refusal)
	assert.Empty(t, repo.upserts)
}

func TestArtifactRecorder_ShowsNoBoardWhereNoneCanBeFilled(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	recorder, _ := boardRecorder(t, repo, nil)

	_, err := recorder.observe(shownBoard(t, pulid.Nil))

	require.ErrorIs(t, err, errNoBoards)
	assert.Empty(t, repo.upserts)
}

type cardBoards struct {
	stubBoards
	asked []*agentports.BoardCardRequest
}

func (c *cardBoards) Card(
	_ context.Context,
	req *agentports.BoardCardRequest,
) (*agentports.BoardCard, error) {
	c.asked = append(c.asked, req)

	return &agentports.BoardCard{
		Title:      "Six late shipments",
		Author:     "Dispatch Desk",
		AnsweredAt: 1_793_000_000,
		Markdown:   "# Six late shipments\n\nFour are held.\n",
	}, nil
}

func TestExportDocument_PrintsABoardAsItReadsNow(t *testing.T) {
	t.Parallel()

	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	artifact := &assistantartifact.Artifact{
		ID: pulid.MustNew("art_"), ThreadID: thread.ID, Kind: assistantartifact.KindBoard,
		Title: "Six late shipments",
	}
	boards := &cardBoards{}
	svc := &Service{
		logger:        zap.NewNop(),
		conversations: &stubConversationRepo{thread: thread},
		artifacts: &stubArtifactRepo{
			stored: map[pulid.ID]*assistantartifact.Artifact{artifact.ID: artifact},
		},
		boards: boards,
	}
	req := agentports.GetThreadRequest{
		ID:         thread.ID,
		UserID:     pulid.MustNew("usr_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}

	file, err := svc.ExportDocument(t.Context(), req, artifact.ID, DocumentExportDOCX)

	require.NoError(t, err)
	assert.Equal(t, "six-late-shipments.docx", file.FileName)
	assert.NotEmpty(t, file.Body)
	require.Len(t, boards.asked, 1)
	assert.Equal(t, req.UserID, boards.asked[0].Actor.UserID)
	assert.Equal(t, req.TenantInfo.OrgID, boards.asked[0].Actor.OrganizationID)
}
