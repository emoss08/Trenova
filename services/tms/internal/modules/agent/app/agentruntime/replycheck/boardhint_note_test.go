package replycheck

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const partnerDayHint = "When the person needs to see what happened, in order rather than one " +
	"figure, do not answer in prose or a list: answer with show_answer {…partnerDay…}."

const twoMessages = `{"messages":[` +
	`{"id":"edimsg_01JMSG0000000000000000001","transactionSet":"990"},` +
	`{"id":"edimsg_01JMSG0000000000000000002","transactionSet":"990"}]}`

func hintedTurn() *boardTurn {
	return &boardTurn{messages: []llmports.Message{
		{Role: llmports.RoleUser, Content: "did all of today's tenders from acme go through?"},
		{
			Role: llmports.RoleTool, ToolName: "edi_list_transfers", ToolCallID: "call_1",
			Content: toolresult.Fence("edi_list_transfers",
				`{"count":5,"howToShow":"`+partnerDayHint+`"}`) + "\n\n[" + partnerDayHint + "]",
		},
	}}
}

func TestWithBoardNote_KeepsTheTurnsHintLastOverALaterRecordsNote(t *testing.T) {
	t.Parallel()

	content := toolresult.Fence("edi_get_ack_health", twoMessages)

	noted := WithBoardNote(hintedTurn(), "edi_get_ack_health", content, false, nil)

	assert.Equal(t, content+"\n\n["+partnerDayHint+"]", noted)
	body, note := splitBoardNote(noted)
	assert.Equal(t, content, body, "the repeated hint is still found as the trailing note")
	assert.Equal(t, "\n\n["+partnerDayHint+"]", note)

	earlier := hintedTurn()
	earlier.messages = append(earlier.messages,
		llmports.Message{Role: llmports.RoleUser, Content: "and yesterday's?"})
	assert.Contains(t, WithBoardNote(earlier, "edi_get_ack_health", content, false, nil),
		"These are 2 records", "a hint from an earlier turn is not this turn's")
}

func TestWithBoardNote_NamesTheWholeListWhenTheResultIsAPageOfIt(t *testing.T) {
	t.Parallel()

	page := strings.Replace(twoLateLoads, `"total":2`, `"count":2,"total":125`, 1)
	content := fencedResult("shipment_list", page)
	table := &agentruntimeports.ShownArtifact{
		Kind: string(assistantartifact.KindTableView), RowKind: "shipment", Rows: 125, Paged: true,
	}

	noted := WithBoardNote(readTurn("shipment_list", ""), "shipment_list", content, false, table)

	assert.Contains(t, noted, "They are a page of a list of 125: where the words mean the "+
		"whole list, write {all}, its own number, read again whenever the board is read; "+
		"{count} is only the records you show.]")
	body, note := splitBoardNote(noted)
	assert.Equal(t, content, body)
	assert.True(t, strings.HasPrefix(note, boardNoteLead))
}

func TestAskForBoard_RepeatsTheTurnsHintInItsCorrection(t *testing.T) {
	t.Parallel()

	turn := proseTurn()
	turn.messages = append(turn.messages, llmports.Message{
		Role: llmports.RoleTool, ToolName: "carrier_list_cover_ending", ToolCallID: "call_2",
		Content: toolresult.Fence("carrier_list_cover_ending",
			`{"howToShow":"`+partnerDayHint+`"}`),
	})

	back, ask := AskForBoard(turn, draftOf("SEED-SHP-010 and SEED-SHP-008 are both late."),
		&BoardAskState{})

	require.True(t, ask)
	assert.Contains(t, back.Messages[1].Content, "As this turn's result said: "+partnerDayHint)
}

const agingRead = `{"metric":"agingCompare","figures":[` +
	`{"name":"openNow","value":"$2,919,909.68"},{"name":"openThen","value":"$3,192,580.72"}]}`

func TestAskForBoard_TellsAMetricsCorrectionWhichRecordsItHasNotRead(t *testing.T) {
	t.Parallel()

	draft := draftOf("Open receivables are $2,919,909.68 today, down from $3,192,580.72 a month " +
		"ago, but the 31–60 bucket grew.")

	back, ask := AskForBoard(readTurn("metric_read", fencedResult("metric_read", agingRead)),
		draft, &BoardAskState{})

	require.True(t, ask)
	assert.Contains(t, back.Messages[1].Content, "Its board also lists the open invoices of "+
		"the bucket that grew")
	assert.Contains(t, back.Messages[1].Content, "read them with receivable_list_open_items first")
}
