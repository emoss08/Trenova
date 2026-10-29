package turnhistory

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func questionWithFile(documentID pulid.ID) *conversation.Message {
	return &conversation.Message{
		Role:    conversation.RoleUser,
		Content: "Create a shipment based on this rate confirmation",
		Attachments: []conversation.MessageAttachment{{
			DocumentID:  documentID,
			FileName:    "rate_confirmation_ch_robinson.pdf",
			ContentType: "application/pdf",
			FileSize:    81_920,
		}},
	}
}

/*
A model that reads files read the rate confirmation as itself while its text
was still being extracted, then saw on the next turn only its own figures, with
no file and no text behind them, and told the person it had made them up. A
recent message's file with no text yet goes again as itself; the entry it
replaces is the one in the replayed message.
*/
func TestReplayUserMessage_SendsAnUnreadFileAgainAsItself(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")

	replayed := replayUserMessage(questionWithFile(documentID), map[string]string{}, true)

	require.Len(t, replayed.Files, 1)
	file := replayed.Files[0]
	assert.Equal(t, documentID.String(), file.DocumentID)
	assert.Equal(t, "application/pdf", file.ContentType)
	assert.Equal(t, int64(81_920), file.Size)
	assert.Less(t, file.EntryStart, file.EntryEnd)
	assert.Contains(t, replayed.Content[file.EntryStart:file.EntryEnd], documentID.String())
	assert.Contains(t, file.NativeEntry, "the file itself is attached")
}

func TestReplayUserMessage_AFileWithItsTextGoesAsTheText(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")

	replayed := replayUserMessage(
		questionWithFile(documentID),
		map[string]string{documentID.String(): "SHIPPER#1: Anyco Trucking"},
		true,
	)

	assert.Empty(t, replayed.Files)
	assert.Contains(t, replayed.Content, "SHIPPER#1: Anyco Trucking")
}

func TestReplayUserMessage_AnOlderMessageNamesItsFileOnly(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")

	replayed := replayUserMessage(questionWithFile(documentID), map[string]string{}, false)

	assert.Empty(t, replayed.Files)
	assert.Contains(t, replayed.Content, "rate_confirmation_ch_robinson.pdf")
}

/*
An agent that could no longer see the lookup that found its ids looked them all
up again and told the person it could not confirm the records it had used. A
shortened result keeps the ids from the part the replay leaves out.
*/
func TestCompactToolResult_KeepsTheRecordIDsItLeavesOut(t *testing.T) {
	t.Parallel()

	payload := `{"customer":{"id":"cus_01M4PHSWHNYFKQ9GSBZ7CB2PJX","name":"C.H. Robinson"},` +
		`"note":"` + strings.Repeat("x", compactedToolResultChars) + `",` +
		`"locations":[{"id":"loc_01M4SAMFDCV8AGW5XZ28YHJYC6"},{"id":"loc_01M4SAMFS0M58Y5T8XVG9X7Q9P"}]}`

	compacted := compactToolResult(toolresult.Fence("shipment_match_records", payload))

	assert.Contains(t, compacted, "was shortened")
	assert.Contains(
		t,
		compacted,
		"The record ids it named still stand and can be used as they are: "+
			"loc_01M4SAMFDCV8AGW5XZ28YHJYC6, loc_01M4SAMFS0M58Y5T8XVG9X7Q9P.",
	)
	assert.Equal(t, 1, strings.Count(compacted, "cus_01M4PHSWHNYFKQ9GSBZ7CB2PJX"),
		"an id the kept head shows is not named twice")
}

func TestCompactToolResult_NamesNoIDsWhenTheCutHoldsNone(t *testing.T) {
	t.Parallel()

	compacted := compactToolResult(
		toolresult.Fence("agent_read_attachment", strings.Repeat("text ", 200)),
	)

	assert.Contains(t, compacted, "was shortened")
	assert.NotContains(t, compacted, "record ids")
}
