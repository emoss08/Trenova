package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attachmentRuntime() *Service {
	return newRuntime(
		&scriptedCompletion{},
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
			queryTool(agentdefinition.CoreToolReadAttachment, nil, nil),
		}},
		&stubActionRegistry{},
		nil,
	)
}

func TestOpenTurn_PutsTheFilesOnTheMessageTheyCameWith(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_").String()
	turn := attachmentRuntime().OpenTurn(t.Context(), &agentruntimeports.RunRequest{
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

	turn := attachmentRuntime().OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "What is due today?",
	})
	assert.False(t, turn.tools.offers(agentdefinition.CoreToolReadAttachment))

	turn = attachmentRuntime().OpenTurn(t.Context(), &agentruntimeports.RunRequest{
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
	assert.Contains(t, first, "read a file with agent_read_attachment")
	assert.True(t, strings.HasSuffix(first,
		"</attachments>\n\nCan you build a shipment based on this rate confirmation?"))
	assert.Equal(t, "Acme", messages[2].Content)
}

func TestOpenTurn_CountsEarlierFilesAsWhatTheTurnMayCite(t *testing.T) {
	t.Parallel()

	turn := attachmentRuntime().OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Approved. Continue.",
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
			Attachments: []conversation.MessageAttachment{
				{DocumentID: recent, FileName: "new.pdf"},
			}},
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

/*
The files go on the message as references, each with where its entry stands in
the message once every line the turn adds in front of it is there, so a model
that reads files can swap the entry for the file. A clock line or an anchor
added ahead of the block must not leave the offsets pointing at the wrong text.
*/
func TestOpenTurn_CarriesEachFileAndWhereItsEntryStands(t *testing.T) {
	t.Parallel()

	pdf := pulid.MustNew("doc_").String()
	photo := pulid.MustNew("doc_").String()
	definition := testDefinition()
	definition.ContextProviders = append(definition.ContextProviders, agentdefinition.ContextClock)
	turn := attachmentRuntime().OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "Check these",
		Context: agentdefinition.RuntimeContext{
			Timezone: "America/Chicago",
			Attachments: []agentdefinition.RuntimeAttachment{
				{
					DocumentID:  pdf,
					FileName:    "ratecon.pdf",
					ContentType: "application/pdf",
					FileSize:    48_213,
					PageCount:   2,
					Reading:     agentdefinition.AttachmentReadingDone,
					Text:        "Linehaul $2,400.",
				},
				{
					DocumentID:  photo,
					FileName:    "seal.png",
					ContentType: "image/png",
					FileSize:    912_004,
					Reading:     agentdefinition.AttachmentReadingFailed,
				},
			},
		},
	})

	asked := turn.messages[len(turn.messages)-1]
	require.Len(t, asked.Files, 2)
	assert.Equal(t, llmports.MessageFile{
		DocumentID:  pdf,
		FileName:    "ratecon.pdf",
		ContentType: "application/pdf",
		Size:        48_213,
		PageCount:   2,
		EntryStart:  asked.Files[0].EntryStart,
		EntryEnd:    asked.Files[0].EntryEnd,
		NativeEntry: asked.Files[0].NativeEntry,
	}, asked.Files[0])
	for _, file := range asked.Files {
		entry := asked.Content[file.EntryStart:file.EntryEnd]
		assert.True(
			t,
			strings.HasPrefix(entry, agentdefinition.AttachmentEntryLead(file.DocumentID)),
		)
		assert.True(
			t,
			strings.HasPrefix(
				file.NativeEntry,
				agentdefinition.AttachmentEntryLead(file.DocumentID),
			),
		)
	}
	assert.Contains(
		t,
		asked.Content[asked.Files[0].EntryStart:asked.Files[0].EntryEnd],
		"Linehaul $2,400.",
	)
	assert.Positive(t, asked.Files[0].EntryStart, "the clock line stands ahead of the block")
}

// A task handed to another agent is not given the person's files.
func TestOpenTurn_GivesADelegatedTaskNoFiles(t *testing.T) {
	t.Parallel()

	turn := attachmentRuntime().OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Summarize the rate",
		Delegation: &agentruntimeports.Delegation{CallID: "call_1", StepScope: "scope"},
		Context: agentdefinition.RuntimeContext{
			Attachments: []agentdefinition.RuntimeAttachment{{
				DocumentID:  pulid.MustNew("doc_").String(),
				FileName:    "ratecon.pdf",
				ContentType: "application/pdf",
			}},
		},
	})

	assert.Empty(t, turn.messages[len(turn.messages)-1].Files)
}
