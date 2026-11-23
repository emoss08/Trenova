package runtimetools

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type answerTurn struct {
	req      agentruntimeports.RunRequest
	messages []llmports.Message
	result   agentruntimeports.RunResult
}

func (a *answerTurn) Request() *agentruntimeports.RunRequest { return &a.req }
func (a *answerTurn) Messages() []llmports.Message           { return a.messages }
func (a *answerTurn) Result() *agentruntimeports.RunResult   { return &a.result }
func (a *answerTurn) Decisions() map[pulid.ID]struct{}       { return nil }
func (a *answerTurn) System() string                         { return "" }
func (a *answerTurn) Evidence() []string                     { return nil }
func (a *answerTurn) Holds(string) bool                      { return false }
func (a *answerTurn) Offers(string) bool                     { return true }

func (a *answerTurn) Artifacts() []agentruntimeports.ShownArtifact { return nil }

const (
	peakFirst  = "inv_01JPEAK00000000000000001"
	peakSecond = "inv_01JPEAK00000000000000002"
)

func peakTurn() *answerTurn {
	return &answerTurn{
		req: agentruntimeports.RunRequest{Input: "what does peak owe us?"},
		messages: []llmports.Message{{
			Role:     llmports.RoleTool,
			ToolName: "receivable_list_open_items",
			Content: toolresult.Fence("receivable_list_open_items", `{"items":[`+
				`{"id":"`+peakFirst+`","number":"SEED-INV-0005","amountDue":"2081.25"},`+
				`{"id":"`+peakSecond+`","number":"SEED-INV-0006","amountDue":"2531.25"}],`+
				`"totalDue":4612.5}`),
		}},
	}
}

func peakAnswer(answer string) map[string]any {
	return map[string]any{
		"title":   "Peak Distributing's unpaid invoices",
		"answer":  answer,
		"measure": "amountDue",
		"groups":  []any{map[string]any{"records": []any{peakFirst, peakSecond}}},
	}
}

func TestShowAnswer_HandsTheBoardPathABoardInTheConversation(t *testing.T) {
	t.Parallel()

	outcome := showAnswer(peakTurn(), peakAnswer("Peak owes {total} on {count} invoices."))

	require.False(t, outcome.Failed, outcome.Content)
	assert.True(t, outcome.Publishes)
	request, ok := outcome.Data.(agentruntimeports.BoardRequest)
	require.True(t, ok)
	assert.Equal(t, board.PlaceConversation, request.Asked)
	assert.Equal(t, board.PlaceConversation, request.Board.Place)
	assert.Equal(t, "Peak owes {total} on {count} invoices.", request.Board.Verdict)
	assert.Len(t, request.Board.Readings, 2)

	again := AnswerStepOutcome(peakAnswer("Peak owes {total} on {count} invoices."))
	assert.Equal(t, request, again.Data, "the durable publish step builds the same board")
}

func TestShowAnswer_LetsATypedFigureTheConversationReadStand(t *testing.T) {
	t.Parallel()

	outcome := showAnswer(peakTurn(), peakAnswer("Peak owes $4,612.50 on {count} invoices."))

	require.False(t, outcome.Failed, outcome.Content)
}

func TestShowAnswer_RefusesATypedFigureNothingRead(t *testing.T) {
	t.Parallel()

	outcome := showAnswer(peakTurn(), peakAnswer("Peak owes $5,100 on {count} invoices."))

	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, `it states "5,100", which no tool result in this `+
		"conversation holds")
	assert.Contains(t, outcome.Content, "Write {count}, {total} or {largest}")
}

func TestShowAnswer_RefusesARecordNoToolReturned(t *testing.T) {
	t.Parallel()

	args := peakAnswer("Peak owes {total}.")
	args["groups"] = []any{map[string]any{
		"records": []any{peakFirst, "inv_01JPEAK00000000000000009"},
	}}
	outcome := showAnswer(peakTurn(), args)

	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "the answer names inv_01JPEAK00000000000000009, which no "+
		"tool in this conversation returned")
}

func TestShowAnswer_SaysWhatItCouldNotTranslate(t *testing.T) {
	t.Parallel()

	outcome := AnswerStepOutcome(peakAnswer("Peak owes {owed}."))

	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, `Tool "show_answer" was not run: answer places {owed}`)
}

func TestAnswer_OffersShowAnswerOnlyWhereBoardsAre(t *testing.T) {
	t.Parallel()

	call := &llmports.ToolCall{Name: ShowAnswerTool, Arguments: peakAnswer("Peak owes {total}.")}
	refused, handled := Answer(peakTurn(), func(string) bool { return false }, call)
	require.True(t, handled)
	assert.True(t, refused.Failed)
	assert.Contains(t, refused.Content, UnboardableRefusal)

	shown, handled := Answer(peakTurn(), func(name string) bool { return name == ShowAnswerTool },
		call)
	require.True(t, handled)
	assert.False(t, shown.Failed, shown.Content)
	assert.Equal(t, ShowAnswerTool, ShowAnswerSpec().Name)
}

func TestShowAnswer_TakesWhatAReplyMaySay(t *testing.T) {
	t.Parallel()

	for _, answer := range []string{
		"Peak owes {total}; the oldest is 4 days past due and the newest 22 hours.",
		"Of the 3 invoices, two came in after 2d.",
		"SEED-INV-0005 and SEED-INV-0006 come to $4,612.50 between them.",
		"$2,081.25 and $2,531.25 come to $4,612.50.",
	} {
		outcome := showAnswer(peakTurn(), peakAnswer(answer))
		assert.False(t, outcome.Failed, "%s: %s", answer, outcome.Content)
	}
}

func TestShowAnswer_SaysWhatAnIDWithTheWrongPrefixReallyIs(t *testing.T) {
	t.Parallel()

	body := "01M7WG94MSAZY6J8J65G8H8KMN"
	shipment := "shp_01M7WG94KWR38VFS2VQNHB2QWS"
	turn := &answerTurn{messages: []llmports.Message{{
		Role:     llmports.RoleTool,
		ToolName: "dispatch_get_board",
		Content: toolresult.Fence("dispatch_get_board", `{"moves":[{"moveId":"sm_`+body+
			`","shipmentId":"`+shipment+`","lateBy":"2d 2h"}]}`),
	}}}
	args := map[string]any{
		"title":  "Late loads",
		"answer": "There are {count} late loads.",
		"groups": []any{map[string]any{"records": []any{"shp_" + body}}},
	}

	outcome := showAnswer(turn, args)

	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "shp_"+body+" is a shipment move's id written as a "+
		"shipment's: that shipment move's shipment is "+shipment+". Send the shipmentId.")
}

func TestMisprefixed_NamesTheKindWhenNoSiblingSaysWhatToSend(t *testing.T) {
	t.Parallel()

	body := "01M7WG94MSAZY6J8J65G8H8KMN"
	read := []string{`{"moves":[{"moveId":"sm_` + body + `"}]}`}

	assert.Equal(t, " shp_"+body+" is a shipment move's id written as a shipment's.",
		misprefixed(read, []string{"shp_" + body}))
	assert.Empty(t, misprefixed(read, []string{"shp_01M7WG94ZZZZZZZZZZZZZZZZZZ"}))
}

const (
	loadEight = "shp_01M7V48EQGE9TBM60WEQJJGQTR"
	loadTen   = "shp_01M7V48EQGE9TBM60WF4JKM3VQ"
	loadOne   = "shp_01M7V48EQGE9TBM60WDH9D4TRN"
)

func comparedTurn() *answerTurn {
	return &answerTurn{
		req: agentruntimeports.RunRequest{
			Input: "Compare SEED-SHP-008 and SEED-SHP-010: which is further behind?",
		},
		messages: []llmports.Message{
			{
				Role: llmports.RoleTool, ToolName: "shipment_get",
				Content: toolresult.Fence("shipment_get", `{"items":[`+
					`{"id":"`+loadEight+`","proNumber":"SEED-SHP-008","totalCharge":"2400"}]}`),
			},
			{
				Role: llmports.RoleTool, ToolName: "shipment_get",
				Content: toolresult.Fence("shipment_get",
					`{"id":"`+loadOne+`","proNumber":"SEED-SHP-001","totalCharge":"3200"}`),
			},
			{
				Role: llmports.RoleTool, ToolName: "shipment_get",
				Content: toolresult.Fence("shipment_get",
					`{"id":"`+loadTen+`","proNumber":"SEED-SHP-010","totalCharge":"4925"}`),
			},
		},
	}
}

func comparedAnswer(title string, second string) map[string]any {
	return map[string]any{
		"title":   title,
		"answer":  "SEED-SHP-010 is further behind: {revenue:" + loadTen + "} against {revenue:" + second + "}.",
		"measure": "revenue",
		"groups": []any{
			map[string]any{"title": "Further behind", "records": []any{loadTen}},
			map[string]any{"title": "Less behind", "records": []any{second}},
		},
	}
}

func shownRecords(t *testing.T, outcome *agentruntimeports.BoardRequest) []pulid.ID {
	t.Helper()

	ids := make([]pulid.ID, 0, 2)
	for _, block := range outcome.Board.Blocks {
		if block.Records != nil {
			ids = append(ids, block.Records.IDs...)
		}
	}

	return ids
}

func TestShowAnswer_ShowsTheRecordItsWordsNameInPlaceOfOneTheyNeverName(t *testing.T) {
	t.Parallel()

	asked := comparedAnswer("SEED-SHP-010 and SEED-SHP-008 compared", loadOne)
	outcome := showAnswer(comparedTurn(), asked)

	require.False(t, outcome.Failed, outcome.Content)
	request, ok := outcome.Data.(agentruntimeports.BoardRequest)
	require.True(t, ok)
	assert.Equal(t, []pulid.ID{loadTen, loadEight}, shownRecords(t, &request))
	require.NotNil(t, outcome.RanWith, "the durable publish step is handed the repaired call")
	again := AnswerStepOutcome(outcome.RanWith)
	assert.Equal(t, request, again.Data)
	assert.Equal(t, []any{loadOne},
		asked["groups"].([]any)[1].(map[string]any)["records"],
		"the model's own call is left as it sent it")
}

func TestShowAnswer_RefusesWordsNamingARecordItCannotPlace(t *testing.T) {
	t.Parallel()

	outcome := showAnswer(comparedTurn(), comparedAnswer(
		"SEED-SHP-010, SEED-SHP-001 and SEED-SHP-008 compared", loadOne,
	))

	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "its words name SEED-SHP-008 (the shipment "+
		loadEight+"), but no group holds it")
}

func TestShowAnswer_ReadsARecordNumberAsTheRecordAResultGaveIt(t *testing.T) {
	t.Parallel()

	asked := comparedAnswer("SEED-SHP-010 and SEED-SHP-008 compared", "SEED-SHP-008")
	asked["answer"] = "SEED-SHP-010 is {lateness:SEED-SHP-010} behind and brings in " +
		"{revenue:SEED-SHP-010}, against {SEED-SHP-008:lateness} and {revenue:SEED-SHP-008}."
	outcome := showAnswer(comparedTurn(), asked)

	require.False(t, outcome.Failed, outcome.Content)
	request, ok := outcome.Data.(agentruntimeports.BoardRequest)
	require.True(t, ok)
	assert.Equal(t, []pulid.ID{loadTen, loadEight}, shownRecords(t, &request))
	assert.Equal(t, request, AnswerStepOutcome(outcome.RanWith).Data)
}

func TestShowAnswer_LeavesACallThatNeedsNoRepairAsItWasSent(t *testing.T) {
	t.Parallel()

	outcome := showAnswer(comparedTurn(), comparedAnswer(
		"SEED-SHP-010 and SEED-SHP-008 compared", loadEight,
	))

	require.False(t, outcome.Failed, outcome.Content)
	assert.Nil(t, outcome.RanWith)
}

func pagedTurn(page string) *answerTurn {
	return &answerTurn{
		req: agentruntimeports.RunRequest{Input: "which loads are running late right now?"},
		messages: []llmports.Message{{
			Role: llmports.RoleTool, ToolName: "shipment_list",
			Content: toolresult.Fence("shipment_list", `{"count":2,`+page+`"items":[`+
				`{"id":"`+loadEight+`","proNumber":"SEED-SHP-008"},`+
				`{"id":"`+loadTen+`","proNumber":"SEED-SHP-010"}]}`),
		}},
	}
}

func lateAnswer(answer string) map[string]any {
	return map[string]any{
		"title":   "Late loads",
		"answer":  answer,
		"measure": "lateness",
		"groups":  []any{map[string]any{"title": "Late", "records": []any{loadEight, loadTen}}},
	}
}

func TestShowAnswer_RefusesACountThatIsNotTheWholeList(t *testing.T) {
	t.Parallel()

	outcome := showAnswer(pagedTurn(`"hasMore":true,`),
		lateAnswer("There are {count} late loads."))
	require.True(t, outcome.Failed)
	assert.Equal(t, `Tool "show_answer" was not run: {count} would read 2, the records shown, `+
		`but they are part of a longer list (it has more pages than were read), so it would `+
		`state a number that is not true. If the words mean all of them, write the list's own `+
		`number as the result gave it, or read the whole list first; if they mean only the `+
		`records shown, count a group, as in "{count_1} of them".`, outcome.Content)

	outcome = showAnswer(pagedTurn(`"total":40,`), lateAnswer("There are {count} late loads."))
	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "(it holds 40 in all)")

	outcome = showAnswer(pagedTurn(`"hasMore":false,"total":2,`),
		lateAnswer("There are {count} late loads."))
	assert.False(t, outcome.Failed, outcome.Content)

	outcome = showAnswer(pagedTurn(`"hasMore":true,`),
		lateAnswer("The furthest is {largest} behind; these are 2 of them."))
	assert.False(t, outcome.Failed, outcome.Content)
}

func TestShowAnswer_ReadsEachLoadsFiguresFromTheLoadTheSentenceNames(t *testing.T) {
	t.Parallel()

	asked := map[string]any{
		"title":   "SEED-SHP-008 and SEED-SHP-010 side by side",
		"measure": "lateness",
		"answer": "SEED-SHP-010 is {total_2} behind and brings in {total_2:revenue}, while " +
			"SEED-SHP-008 is {total_1} behind and brings in {total_1:revenue}.",
		"groups": []any{
			map[string]any{"title": "Less behind", "records": []any{loadEight}},
			map[string]any{"title": "Further behind", "records": []any{loadTen}},
		},
	}

	outcome := showAnswer(comparedTurn(), asked)

	require.False(t, outcome.Failed, outcome.Content)
	request, ok := outcome.Data.(agentruntimeports.BoardRequest)
	require.True(t, ok)
	assert.Equal(t, "SEED-SHP-010 is {lateness_of_2} behind and brings in {revenue_of_2}, while "+
		"SEED-SHP-008 is {lateness_of_1} behind and brings in {revenue_of_1}.",
		request.Board.Verdict)
	for _, reading := range request.Board.Readings {
		want := pulid.ID(loadEight)
		if strings.HasSuffix(reading.Key, "_of_2") {
			want = loadTen
		}
		assert.Equal(t, []pulid.ID{want}, reading.Records.IDs, reading.Key)
	}
	assert.Equal(t, request, AnswerStepOutcome(outcome.RanWith).Data)
}
