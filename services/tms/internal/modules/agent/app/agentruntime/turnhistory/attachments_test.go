package turnhistory

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	messages, _ := ReplayHistoryWithFiles(history, nil, files)

	assert.NotContains(t, messages[0].Content, "OLD LOAD TOTAL",
		"a file older than the recent turns is named, not read again")
	assert.Contains(t, messages[0].Content, "- id: "+old.String())
	assert.Contains(t, messages[6].Content, "<file_text>\nSHIPPER#1:   Harbor Line Cold Storage")
}
