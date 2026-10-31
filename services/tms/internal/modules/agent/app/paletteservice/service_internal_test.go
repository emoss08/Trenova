package paletteservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/palettejobs/palettecontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowID_NamesAQuestionByThePersonWhoAskedIt(t *testing.T) {
	t.Parallel()

	askID := pulid.MustNew("pask_")
	mine := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	theirs := mine
	theirs.UserID = pulid.MustNew("usr_")

	id := palettecontract.WorkflowID(paletteports.AskRef{AskID: askID, TenantInfo: mine})

	assert.True(t, strings.HasPrefix(id, "palette-answer/"+mine.OrgID.String()+"/"+mine.UserID.String()+"/"))
	assert.True(t, strings.HasSuffix(id, askID.String()))
	assert.NotEqual(t, id, palettecontract.WorkflowID(paletteports.AskRef{AskID: askID, TenantInfo: theirs}),
		"another person's session names a different question with the same id")
}

func TestValidQuestion_TrimsItAndRefusesAnEmptyOne(t *testing.T) {
	t.Parallel()

	got, err := validQuestion(&paletteports.Question{
		Content: "  who is on S-104?  ",
		Page:    &agentpagecontext.PageContext{Path: "/dispatch"},
	})
	require.NoError(t, err)
	assert.Equal(t, "who is on S-104?", got.Content)
	assert.Equal(t, "/dispatch", got.Page.Path)

	_, err = validQuestion(&paletteports.Question{Content: "   "})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "content", multiErr.Errors[0].Field)
}

func TestSeed_OpensTheDeskConversationWithTheQuestionAndTheAnswer(t *testing.T) {
	t.Parallel()

	providerID := pulid.MustNew("aip_")
	named := agentpagecontext.EntityRef{Type: "shipment", ID: "shp_1", Label: "S-104"}
	read := agentpagecontext.EntityRef{Type: "worker", ID: "wrk_1", Label: "Dana Cole"}
	messages := seed(&paletteports.Answer{
		Question: paletteports.Question{
			Content:  "who is on S-104?",
			Mentions: []agentpagecontext.EntityRef{named},
		},
		Answer: &llmports.NovaAnswer{
			Text:       "Dana is driving S-104.",
			Records:    []agentpagecontext.EntityRef{named, read},
			Model:      "haiku",
			ProviderID: providerID,
		},
	})

	require.Len(t, messages, 2, "the palette's reads are not replayed into the Desk")
	assert.Equal(t, conversation.RoleUser, messages[0].Role)
	assert.Equal(t, "who is on S-104?", messages[0].Content)
	assert.Equal(t, []agentpagecontext.EntityRef{named, read}, messages[0].Mentions,
		"the records Nova read ride on the question, each once")
	assert.Empty(t, messages[0].ToolCalls)

	assert.Equal(t, conversation.RoleAssistant, messages[1].Role)
	assert.Equal(t, "Dana is driving S-104.", messages[1].Content)
	assert.Empty(t, messages[1].ToolCalls)
	assert.Equal(t, "haiku", messages[1].Model)
	assert.Equal(t, providerID, messages[1].ProviderID)
}

func TestCarriedMentions_StopsAtWhatAMessageMayMention(t *testing.T) {
	t.Parallel()

	read := make([]agentpagecontext.EntityRef, 0, 15)
	for range 15 {
		read = append(read, agentpagecontext.EntityRef{Type: "shipment", ID: pulid.MustNew("shp_").String()})
	}

	assert.Len(t, carriedMentions(nil, read), agentpagecontext.MaxMentions)
}
