package replycheck

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type boardTurn struct {
	req       agentruntimeports.RunRequest
	messages  []llmports.Message
	result    agentruntimeports.RunResult
	artifacts []agentruntimeports.ShownArtifact
	unoffered bool
}

func (b *boardTurn) Request() *agentruntimeports.RunRequest       { return &b.req }
func (b *boardTurn) Messages() []llmports.Message                 { return b.messages }
func (b *boardTurn) System() string                               { return "" }
func (b *boardTurn) Evidence() []string                           { return nil }
func (b *boardTurn) Result() *agentruntimeports.RunResult         { return &b.result }
func (b *boardTurn) Artifacts() []agentruntimeports.ShownArtifact { return b.artifacts }
func (b *boardTurn) Holds(string) bool                            { return false }

func (b *boardTurn) Offers(name string) bool {
	return name == showAnswerTool && !b.unoffered
}

const twoLateLoads = `{"items":[` +
	`{"id":"shp_01JLATE0000000000000000001","proNumber":"SEED-SHP-008",` +
	`"customer":{"id":"cus_01JSUNB000000000000000001","name":"Sunbelt Materials"},` +
	`"moves":[{"id":"sm_01JMOVE00000000000000000001","assignment":{"primaryWorker":` +
	`{"id":"wrk_01JLEON000000000000000001","firstName":"Leon","lastName":"Park"}}}]},` +
	`{"id":"shp_01JLATE0000000000000000002","proNumber":"SEED-SHP-010",` +
	`"customer":{"id":"cus_01JFRSH000000000000000001","name":"FreshHaul Foods"}}` +
	`],"total":2}`

func fencedResult(tool, payload string) string {
	return toolresult.Fence(tool, payload) +
		"\n\n[This view is kept beside the conversation as \"Late shipments\".]"
}

func readTurn(tool, content string) *boardTurn {
	return &boardTurn{messages: []llmports.Message{
		{Role: llmports.RoleUser, Content: "which loads are late?"},
		{Role: llmports.RoleTool, ToolName: tool, ToolCallID: "call_1", Content: content},
	}}
}

func TestWithBoardNote_NotesAResultOfTwoRecordsABoardTakes(t *testing.T) {
	t.Parallel()

	content := fencedResult("shipment_list", twoLateLoads)
	noted := WithBoardNote(readTurn("shipment_list", ""), "shipment_list", content, false, nil)

	note := "\n\n[These are 2 records. If your answer weighs them together, or gives a count, a " +
		"total or a comparison across them, do not point to a table and do not answer in " +
		"prose: answer with show_answer, naming the records it rests on and writing {count}, " +
		"{total} or {largest} where a figure goes, never typed. It is drawn in the conversation " +
		"as your reply and ends your turn, so make every other call first. Only a list the " +
		"person just asked to see is pointed to, and one record or one fact is a sentence.]"
	assert.Equal(t, content+note, noted, "the note is the last thing the model reads")

	tool, payload, fenced := toolresult.Unfence(noted)
	require.True(t, fenced)
	assert.Equal(t, "shipment_list", tool)
	assert.Equal(t, twoLateLoads, payload, "the fence parser reads the payload, not the note")
	assert.ElementsMatch(t, toolresult.CitedRecords(content), toolresult.CitedRecords(noted))
}

func TestReplaceShownNote_LooksPastATrailingBoardNote(t *testing.T) {
	t.Parallel()

	stale := "\n\n[This result is kept as a card titled \"SEED-SHP-008\".]"
	folded := "\n\n[This card went into the table \"Late shipments\".]"
	body := toolresult.Fence("shipment_list", twoLateLoads) + stale
	noted := WithBoardNote(proseTurn(), "shipment_list", body, false, nil)
	require.NotEqual(t, body, noted)

	replaced, ok := ReplaceShownNote(noted, stale, folded)
	require.True(t, ok)
	assert.Equal(t, toolresult.Fence("shipment_list", twoLateLoads)+folded+noted[len(body):],
		replaced, "the shown note is replaced and the board note stays last")

	plain, ok := ReplaceShownNote(body, stale, folded)
	require.True(t, ok)
	assert.Equal(t, toolresult.Fence("shipment_list", twoLateLoads)+folded, plain)

	_, ok = ReplaceShownNote(noted, "\n\n[Another note.]", folded)
	assert.False(t, ok)
}

func TestWithBoardNote_LeavesAResultThatDoesNotQualify(t *testing.T) {
	t.Parallel()

	oneLoad := `{"id":"shp_01JLATE0000000000000000001","proNumber":"SEED-SHP-008",` +
		`"customer":{"id":"cus_01JSUNB000000000000000001","name":"Sunbelt Materials"},` +
		`"tractor":{"id":"trac_01JTRAC00000000000000001","code":"TRC-004"}}`
	documents := `{"items":[` +
		`{"id":"doc_01JDOCS000000000000000001","originalName":"BOL-2026-0008.pdf"},` +
		`{"id":"doc_01JDOCS000000000000000002","originalName":"POD-2026-0010.pdf"}]}`
	shownBoard := readTurn("shipment_list", "")
	shownBoard.artifacts = []agentruntimeports.ShownArtifact{
		{ID: pulid.MustNew("art_"), Kind: "board", Title: "Late loads", Place: "beside"},
	}
	unoffered := readTurn("shipment_list", "")
	unoffered.unoffered = true
	delegated := readTurn("shipment_list", "")
	delegated.req.Delegation = &agentruntimeports.Delegation{}
	followUp := readTurn("shipment_list", "")
	followUp.req.TrenovaWritten = true
	unattended := readTurn("shipment_list", "")
	unattended.req.Unattended = true

	cases := map[string]struct {
		turn    *boardTurn
		tool    string
		content string
		failed  bool
	}{
		"one record with its relations": {
			turn: readTurn("shipment_get", ""), tool: "shipment_get",
			content: fencedResult("shipment_get", oneLoad),
		},
		"kinds a board does not take": {
			turn: readTurn("document_list", ""), tool: "document_list",
			content: fencedResult("document_list", documents),
		},
		"a failed result": {
			turn: readTurn("shipment_list", ""), tool: "shipment_list",
			content: toolresult.FenceFailure("shipment_list", twoLateLoads), failed: true,
		},
		"show_board itself": {
			turn: readTurn("shipment_list", ""), tool: "show_board",
			content: fencedResult("show_board", twoLateLoads),
		},
		"show_answer itself": {
			turn: readTurn("shipment_list", ""), tool: "show_answer",
			content: fencedResult("show_answer", twoLateLoads),
		},
		"publish_artifact": {
			turn: readTurn("shipment_list", ""), tool: "publish_artifact",
			content: fencedResult("publish_artifact", twoLateLoads),
		},
		"a turn not offered boards": {
			turn: unoffered, tool: "shipment_list",
			content: fencedResult("shipment_list", twoLateLoads),
		},
		"a turn working for another agent": {
			turn: delegated, tool: "shipment_list",
			content: fencedResult("shipment_list", twoLateLoads),
		},
		"a turn Trenova wrote, such as a decision follow-up": {
			turn: followUp, tool: "shipment_list",
			content: fencedResult("shipment_list", twoLateLoads),
		},
		"a background run": {
			turn: unattended, tool: "shipment_list",
			content: fencedResult("shipment_list", twoLateLoads),
		},
		"a board already shown": {
			turn: shownBoard, tool: "shipment_list",
			content: fencedResult("shipment_list", twoLateLoads),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.content, WithBoardNote(tc.turn, tc.tool, tc.content, tc.failed, nil))
		})
	}
}

func proseTurn() *boardTurn {
	return readTurn("shipment_list", fencedResult("shipment_list", twoLateLoads))
}

func draftOf(text string) *llmports.ChatCompletionResult {
	return &llmports.ChatCompletionResult{
		Text: text, ProviderMessageID: "msg_1", ProviderID: pulid.MustNew("aip_"),
	}
}

func TestAskForBoard_SendsAProseReplyAcrossRecordsBackOnce(t *testing.T) {
	t.Parallel()

	turn := proseTurn()
	draft := draftOf("SEED-SHP-010 is the furthest behind; SEED-SHP-008 is rolling but late.")
	state := &BoardAskState{}

	back, ask := AskForBoard(turn, draft, state)

	require.True(t, ask)
	assert.True(t, state.Asked)
	require.Len(t, back.Events, 2)
	assert.Equal(t, assistantstreamports.AssistantEventRetrying, back.Events[0].Event)
	retrying, ok := back.Events[0].Data.(assistantstreamports.AssistantRetryingEvent)
	require.True(t, ok)
	assert.Equal(t, llmports.RetryKindRestart, retrying.Kind)
	assert.Equal(t, BoardRetryReason, retrying.Reason)
	regrounded, ok := back.Events[1].Data.(assistantstreamports.AssistantReplyRegroundedEvent)
	require.True(t, ok)
	assert.Equal(t, assistantstreamports.RegroundAnswerWithBoard, regrounded.Action)

	require.Len(t, back.Messages, 2)
	assert.Equal(t, llmports.RoleAssistant, back.Messages[0].Role)
	assert.Equal(t, draft.Text, back.Messages[0].Content)
	assert.Equal(t, "msg_1", back.Messages[0].ReplyID)
	correction := back.Messages[1]
	assert.Equal(t, llmports.RoleSystem, correction.Role)
	assert.Contains(t, correction.Content, "it names SEED-SHP-008, SEED-SHP-010.")
	assert.Contains(t, correction.Content,
		"belongs on a board in the conversation, not in prose and not as a pointer to a table")
	assert.Contains(t, correction.Content, "call show_answer with the records it rests on")
	assert.Contains(t, correction.Content,
		"writing {count}, {total} or {largest} where a figure goes, never typed")
	assert.Contains(t, correction.Content, "If a board cannot show this answer, or the person "+
		"only asked to see a list or asked for one fact, reply again as you did.")
	assert.Contains(t, correction.Content, "reply again as you did")
	assert.NotContains(t, correction.Content, "shp_", "records are named by their labels")
	assert.True(t, isCorrection(&correction), "the draft before it reads as discarded")

	_, again := AskForBoard(turn, draftOf(draft.Text), state)
	assert.False(t, again, "the second reply is taken as it is")
}

func TestAskForBoard_LeavesAReplyTheConditionsLeaveOut(t *testing.T) {
	t.Parallel()

	both := "SEED-SHP-010 is the furthest behind; SEED-SHP-008 is rolling but late."
	unoffered := proseTurn()
	unoffered.unoffered = true
	shown := proseTurn()
	shown.artifacts = []agentruntimeports.ShownArtifact{
		{ID: pulid.MustNew("art_"), Kind: "board", Title: "Late loads", Place: "conversation"},
	}
	changed := proseTurn()
	changed.result.Actions = []agentruntimeports.PendingAction{{ToolName: "shipment_add_comment"}}
	delegated := proseTurn()
	delegated.req.Delegation = &agentruntimeports.Delegation{}
	failed := proseTurn()
	failed.messages[1].IsError = true
	followUp := proseTurn()
	followUp.req.TrenovaWritten = true
	kept := proseTurn()
	kept.artifacts = []agentruntimeports.ShownArtifact{{
		ID:     pulid.MustNew("art_"),
		Kind:   "table_view",
		Title:  "Late shipments",
		Rows:   2,
		Labels: []string{"SEED-SHP-008", "SEED-SHP-010"},
	}}

	cases := map[string]struct {
		turn  *boardTurn
		reply string
	}{
		"a turn not offered boards":                          {turn: unoffered, reply: both},
		"a turn that has shown a board":                      {turn: shown, reply: both},
		"a turn that proposed a change":                      {turn: changed, reply: both},
		"a turn working for another agent":                   {turn: delegated, reply: both},
		"a turn Trenova wrote, such as a decision follow-up": {turn: followUp, reply: both},
		"a reply that names two records and links the table": {
			turn: proseTurn(),
			reply: "SEED-SHP-010 is further behind than SEED-SHP-008, as " +
				"[Late shipments](artifact:art_01JTABLE00000000000000001) shows.",
		},
		"a reply that reprints a kept table": {
			turn: kept,
			reply: "Here they are:\n| Pro | Status |\n| --- | --- |\n| SEED-SHP-008 | Late |\n" +
				"| SEED-SHP-010 | Late |\n| SEED-SHP-008 | Late |\n| SEED-SHP-010 | Late |\n" +
				"| SEED-SHP-008 | Late |\n| SEED-SHP-010 | Late |",
		},
		"a reply that gives a count and links the list": {
			turn: proseTurn(),
			reply: "Two loads are late; they are in " +
				"[Late shipments](artifact:art_01JTABLE00000000000000001).",
		},
		"a reply that names one record": {
			turn: proseTurn(), reply: "SEED-SHP-010 is the furthest behind.",
		},
		"records only a failed result named": {turn: failed, reply: both},
		"labels inside longer words": {
			turn: proseTurn(), reply: "SEED-SHP-0100 and XSEED-SHP-008 are not in this list.",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			state := &BoardAskState{}
			_, ask := AskForBoard(tc.turn, draftOf(tc.reply), state)
			assert.False(t, ask)
			assert.False(t, state.Asked)
		})
	}
}

func TestNamedBoardRecords_MatchesWholeLabelsOfFourCharactersOrMore(t *testing.T) {
	t.Parallel()

	payload := `{"items":[` +
		`{"id":"cus_01JCUST000000000000000001","name":"Ace"},` +
		`{"id":"wrk_01JWORK000000000000000002","firstName":"Leon","lastName":"Park"},` +
		`{"id":"shp_01JSHIP000000000000000001","proNumber":"SHP-1"},` +
		`{"id":"shp_01JSHIP000000000000000002","proNumber":"SHP-10"}]}`
	turn := readTurn("shipment_list", fencedResult("shipment_list", payload))

	assert.Equal(t, []string{"Leon Park", "SHP-10"},
		namedBoardRecords(turn, "Ace hauls SHP-10, and Leon Park's other load is SHP-100."),
		"Ace is under four characters, and SHP-1 stands only inside SHP-10 and SHP-100")
	assert.Equal(t, []string{"Leon Park"},
		namedBoardRecords(turn, "Leon Park, then Leon Park again, then Leon Parkinson."),
		"a label counts once")
}

const peakOpenItems = `{"columns":["invoiceId","invoiceNumber","openAmount"],"count":3,"items":[` +
	`{"invoiceId":"inv_01JPEAK00000000000000005","invoiceNumber":"SEED-INV-0005","openAmount":"2081.25"},` +
	`{"invoiceId":"inv_01JPEAK00000000000000006","invoiceNumber":"SEED-INV-0006","openAmount":"2531.25"},` +
	`{"invoiceId":"inv_01JPEAK00000000000000007","invoiceNumber":"SEED-INV-0007","openAmount":"1562.50"}]}`

func invoiceTable() agentruntimeports.ShownArtifact {
	return agentruntimeports.ShownArtifact{
		ID:      pulid.MustNew("art_"),
		Kind:    "table_view",
		Title:   "Receivable open items",
		Rows:    3,
		RowKind: "invoice",
		Labels:  []string{"SEED-INV-0005", "SEED-INV-0006", "SEED-INV-0007"},
	}
}

func TestWithBoardNote_CountsTheRowsOfATableKeptAsRecords(t *testing.T) {
	t.Parallel()

	content := fencedResult("receivable_list_open_items", peakOpenItems)
	turn := readTurn("receivable_list_open_items", "")
	table := invoiceTable()

	assert.Equal(t, content, WithBoardNote(turn, "receivable_list_open_items", content, false, nil),
		"the rows carry invoiceId, not id, so the result alone names no records")
	assert.Contains(t,
		WithBoardNote(turn, "receivable_list_open_items", content, false, &table),
		"\n\n[These are 3 records.")

	table.RowKind = "billing_queue_item"
	assert.Equal(t, content,
		WithBoardNote(turn, "receivable_list_open_items", content, false, &table),
		"rows of a kind a board does not take are not counted")
}

func TestAskForBoard_NamesRecordsByTheLabelsOfAKeptTable(t *testing.T) {
	t.Parallel()

	turn := readTurn("receivable_list_open_items",
		fencedResult("receivable_list_open_items", peakOpenItems))
	turn.artifacts = []agentruntimeports.ShownArtifact{invoiceTable()}
	draft := draftOf(
		"Peak owes on SEED-INV-0005, SEED-INV-0006 and SEED-INV-0007, $6,175.00 in all.",
	)

	back, ask := AskForBoard(turn, draft, &BoardAskState{})

	require.True(t, ask)
	assert.Contains(t, back.Messages[1].Content,
		"it names SEED-INV-0005, SEED-INV-0006, SEED-INV-0007.")
}

func TestAskForBoard_SendsBackALinkedReplyThatWritesOutEveryRowOfAKeptTable(t *testing.T) {
	t.Parallel()

	turn := readTurn("receivable_list_open_items",
		fencedResult("receivable_list_open_items", peakOpenItems))
	turn.artifacts = []agentruntimeports.ShownArtifact{invoiceTable()}
	draft := draftOf("Peak has three unpaid invoices:\n- SEED-INV-0007, partly paid\n" +
		"- SEED-INV-0006, disputed\n- SEED-INV-0005, not yet due\n" +
		"The details are in [Receivable open items](artifact:art_01JTABLE00000000000000001).")

	back, ask := AskForBoard(turn, draft, &BoardAskState{})

	require.True(t, ask)
	assert.Contains(t, back.Messages[1].Content,
		"it names SEED-INV-0005, SEED-INV-0006, SEED-INV-0007.")
}

func TestAskForBoard_SendsBackALinkedReplyAboutRecordsTheTurnLookedUp(t *testing.T) {
	t.Parallel()

	turn := &boardTurn{messages: []llmports.Message{
		{Role: llmports.RoleUser, Content: "which loads are late, and why?"},
		{Role: llmports.RoleAssistant, ToolCalls: []llmports.ToolCall{{
			ID:   "call_1",
			Name: "shipment_get_tracking",
			Arguments: map[string]any{"shipmentIds": []any{
				"shp_01JLATE0000000000000000001", "shp_01JLATE0000000000000000002",
			}},
		}}},
		{
			Role: llmports.RoleTool, ToolName: "shipment_get_tracking", ToolCallID: "call_1",
			Content: fencedResult("shipment_get_tracking", twoLateLoads),
		},
	}}
	draft := draftOf("Two loads are late:\n- SEED-SHP-010: the delivery window closed with " +
		"no arrival.\n- SEED-SHP-008: no driver on the last leg.\nThe tracking is in " +
		"[Shipment trackings](artifact:art_01JTABLE00000000000000001).")

	_, ask := AskForBoard(turn, draft, &BoardAskState{})

	assert.True(t, ask)
}

func TestAskForBoard_LeavesALinkedReplyThatPicksAFewRowsOfAList(t *testing.T) {
	t.Parallel()

	labels := make([]string, 0, 21)
	for idx := range 21 {
		labels = append(labels, fmt.Sprintf("S261100%08d", idx))
	}
	turn := readTurn("shipment_list", fencedResult("shipment_list", twoLateLoads))
	turn.artifacts = []agentruntimeports.ShownArtifact{{
		ID: pulid.MustNew("art_"), Kind: "table_view", Title: "Shipments", Rows: 21,
		RowKind: "shipment", Labels: labels,
	}}
	draft := draftOf("There are 176 new loads; the first 21 are in " +
		"[Shipments](artifact:art_01JTABLE00000000000000001). The soonest pickups are " +
		"SEED-SHP-008 and SEED-SHP-010.")

	_, ask := AskForBoard(turn, draft, &BoardAskState{})

	assert.False(t, ask)
}

func TestAskForBoard_LeavesALinkedReplyThatWritesOutAPageOfALongerList(t *testing.T) {
	t.Parallel()

	turn := readTurn("receivable_list_open_items",
		fencedResult("receivable_list_open_items", peakOpenItems))
	table := invoiceTable()
	table.Paged = true
	turn.artifacts = []agentruntimeports.ShownArtifact{table}
	draft := draftOf("The first page: SEED-INV-0005, SEED-INV-0006 and SEED-INV-0007, all in " +
		"[Receivable open items](artifact:art_01JTABLE00000000000000001).")

	_, ask := AskForBoard(turn, draft, &BoardAskState{})

	assert.False(t, ask)
}
