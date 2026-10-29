package assistantartifacts

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type cardBoards struct {
	assistanttest.StubBoards
	asked []*assistantports.BoardCardRequest
}

func (c *cardBoards) Card(
	_ context.Context,
	req *assistantports.BoardCardRequest,
) (*assistantports.BoardCard, error) {
	c.asked = append(c.asked, req)

	return &assistantports.BoardCard{
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
		conversations: &assistanttest.StubConversationRepo{Thread: thread},
		artifacts: &assistanttest.StubArtifactRepo{
			Stored: map[pulid.ID]*assistantartifact.Artifact{artifact.ID: artifact},
		},
		boards: boards,
	}
	req := conversationports.GetThreadRequest{
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
