package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attachmentRuntime() *Service {
	return newRuntime(
		&scriptedCompletion{},
		&stubQueryRegistry{Tools: []agentports.AgentQueryTool{
			queryTool(agentdefinition.CoreToolReadAttachment, nil, nil),
		}},
		&stubActionRegistry{},
		nil,
	)
}

func TestOpenTurn_PutsTheFilesOnTheMessageTheyCameWith(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_").String()
	turn := attachmentRuntime().OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Here it is",
		Context: agentdefinition.RuntimeContext{
			Attachments: []agentdefinition.RuntimeAttachment{{
				DocumentID: documentID,
				FileName:   "rate_confirmation_ch_robinson.pdf",
				Reading:    agentdefinition.AttachmentReadingDone,
				Text:       "SHIPPER#1:   BEX Logistics-OH Consolidation   Pick Up Date:   08/02/22",
			}},
		},
	})

	asked := turn.messages[len(turn.messages)-1].Content
	assert.Contains(t, asked, "- id: "+documentID)
	assert.Contains(t, asked, "BEX Logistics-OH Consolidation   Pick Up Date:   08/02/22")
	assert.True(t, strings.HasSuffix(asked, "</attachments>\n\nHere it is"))
	assert.NotContains(t, turn.system, documentID)
	assert.Equal(t, "Here it is", turn.result.Messages[0].Content,
		"the conversation keeps what the person typed")
	assert.True(t, turn.tools.offers(agentdefinition.CoreToolReadAttachment))
}

func TestOpenTurn_ReadsAttachmentsOnlyInAConversationWithFiles(t *testing.T) {
	t.Parallel()

	turn := attachmentRuntime().OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "What is due today?",
	})
	assert.False(t, turn.tools.offers(agentdefinition.CoreToolReadAttachment))

	turn = attachmentRuntime().OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Build the shipment from it",
		History: []conversation.Message{
			{
				Role:    conversation.RoleUser,
				Content: "Can you build a shipment based on this rate confirmation?",
				Attachments: []conversation.MessageAttachment{{
					DocumentID: pulid.MustNew("doc_"),
					FileName:   "405515639Rate Confirmation.pdf",
				}},
			},
			{Role: conversation.RoleAssistant, Content: "Which customer is it for?"},
		},
	})
	assert.True(t, turn.tools.offers(agentdefinition.CoreToolReadAttachment))
}

func TestReplayHistory_NamesTheFilesAnEarlierMessageCarried(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	history := []conversation.Message{
		{
			Role:    conversation.RoleUser,
			Content: "Can you build a shipment based on this rate confirmation?",
			Attachments: []conversation.MessageAttachment{{
				DocumentID:  documentID,
				FileName:    "405515639Rate Confirmation.pdf",
				ContentType: "application/pdf",
			}},
		},
		{Role: conversation.RoleAssistant, Content: "Which customer is it for?"},
		{Role: conversation.RoleUser, Content: "Acme"},
	}

	messages, _ := replayHistory(history, nil)

	require.Len(t, messages, 3)
	first := messages[0].Content
	assert.Contains(t, first, "- id: "+documentID.String())
	assert.Contains(t, first, "file: 405515639Rate Confirmation.pdf (application/pdf)")
	assert.Contains(t, first, "read a file with read_attachment")
	assert.True(t, strings.HasSuffix(first,
		"</attachments>\n\nCan you build a shipment based on this rate confirmation?"))
	assert.Equal(t, "Acme", messages[2].Content)
}

func TestOpenTurn_CountsEarlierFilesAsWhatTheTurnMayCite(t *testing.T) {
	t.Parallel()

	turn := attachmentRuntime().OpenTurn(t.Context(), &agentports.RunRequest{
		Definition:   testDefinition(),
		Actor:        testActor(),
		Input:        "Approved. Continue.",
		FileTexts: map[string]string{
			"doc_1": "Line Haul - FLAT RATE   1   $2,150.00   $2,150.00\nTotal:   $2,410.00",
		},
	})

	assert.Contains(t, turn.evidence, "2410")
	assert.Contains(t, turn.evidence, "2150")
	assert.NotContains(t, turn.messages[len(turn.messages)-1].Content, "2,410",
		"an earlier file is evidence, not part of the prompt")
}

func TestReplayHistory_ShowsARecentFileWithItsTextAndAnOldOneByName(t *testing.T) {
	t.Parallel()

	old, recent := pulid.MustNew("doc_"), pulid.MustNew("doc_")
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "the first load",
			Attachments: []conversation.MessageAttachment{{DocumentID: old, FileName: "old.pdf"}}},
		{Role: conversation.RoleAssistant, Content: "Booked."},
		{Role: conversation.RoleUser, Content: "two"},
		{Role: conversation.RoleAssistant, Content: "ok"},
		{Role: conversation.RoleUser, Content: "three"},
		{Role: conversation.RoleAssistant, Content: "ok"},
		{Role: conversation.RoleUser, Content: "build this one",
			Attachments: []conversation.MessageAttachment{{DocumentID: recent, FileName: "new.pdf"}}},
		{Role: conversation.RoleAssistant, Content: "Proposed the customer."},
	}
	files := map[string]string{
		old.String():    "OLD LOAD TOTAL $1,000.00",
		recent.String(): "SHIPPER#1:   Harbor Line Cold Storage   Total:   $2,410.00",
	}

	messages, _ := replayHistoryWithFiles(history, nil, files)

	assert.NotContains(t, messages[0].Content, "OLD LOAD TOTAL",
		"a file older than the recent turns is named, not read again")
	assert.Contains(t, messages[0].Content, "- id: "+old.String())
	assert.Contains(t, messages[6].Content, "<file_text>\nSHIPPER#1:   Harbor Line Cold Storage")
}
