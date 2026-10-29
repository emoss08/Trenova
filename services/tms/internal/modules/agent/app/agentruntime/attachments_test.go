package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmessage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
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
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Input:      "Here it is",
		Context: agentturn.RuntimeContext{
			Attachments: []agentturn.RuntimeAttachment{{
				DocumentID: documentID,
				FileName:   "rate_confirmation_ch_robinson.pdf",
				Reading:    agentturn.AttachmentReadingDone,
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
	assert.True(t, turn.tools.Offers(agentdefinition.CoreToolReadAttachment))
}

func TestOpenTurn_ReadsAttachmentsOnlyInAConversationWithFiles(t *testing.T) {
	t.Parallel()

	turn := attachmentRuntime().OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Input:      "What is due today?",
	})
	assert.False(t, turn.tools.Offers(agentdefinition.CoreToolReadAttachment))

	turn = attachmentRuntime().OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
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
	assert.True(t, turn.tools.Offers(agentdefinition.CoreToolReadAttachment))
}

func TestOpenTurn_CountsEarlierFilesAsWhatTheTurnMayCite(t *testing.T) {
	t.Parallel()

	turn := attachmentRuntime().OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
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
	definition := runtimetest.Definition()
	definition.ContextProviders = append(definition.ContextProviders, agentdefinition.ContextClock)
	turn := attachmentRuntime().OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "Check these",
		Context: agentturn.RuntimeContext{
			Timezone: "America/Chicago",
			Attachments: []agentturn.RuntimeAttachment{
				{
					DocumentID:  pdf,
					FileName:    "ratecon.pdf",
					ContentType: "application/pdf",
					FileSize:    48_213,
					PageCount:   2,
					Reading:     agentturn.AttachmentReadingDone,
					Text:        "Linehaul $2,400.",
				},
				{
					DocumentID:  photo,
					FileName:    "seal.png",
					ContentType: "image/png",
					FileSize:    912_004,
					Reading:     agentturn.AttachmentReadingFailed,
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
			strings.HasPrefix(entry, agentmessage.AttachmentEntryLead(file.DocumentID)),
		)
		assert.True(
			t,
			strings.HasPrefix(
				file.NativeEntry,
				agentmessage.AttachmentEntryLead(file.DocumentID),
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
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Input:      "Summarize the rate",
		Delegation: &agentruntimeports.Delegation{CallID: "call_1", StepScope: "scope"},
		Context: agentturn.RuntimeContext{
			Attachments: []agentturn.RuntimeAttachment{{
				DocumentID:  pulid.MustNew("doc_").String(),
				FileName:    "ratecon.pdf",
				ContentType: "application/pdf",
			}},
		},
	})

	assert.Empty(t, turn.messages[len(turn.messages)-1].Files)
}
