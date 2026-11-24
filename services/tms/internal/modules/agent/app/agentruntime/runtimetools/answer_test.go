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
	req       agentruntimeports.RunRequest
	messages  []llmports.Message
	result    agentruntimeports.RunResult
	artifacts []agentruntimeports.ShownArtifact
}

func (a *answerTurn) Request() *agentruntimeports.RunRequest { return &a.req }
func (a *answerTurn) Messages() []llmports.Message           { return a.messages }
func (a *answerTurn) Result() *agentruntimeports.RunResult   { return &a.result }
func (a *answerTurn) Decisions() map[pulid.ID]struct{}       { return nil }
func (a *answerTurn) System() string                         { return "" }
func (a *answerTurn) Evidence() []string                     { return nil }
func (a *answerTurn) Holds(string) bool                      { return false }
func (a *answerTurn) Offers(string) bool                     { return true }

func (a *answerTurn) Artifacts() []agentruntimeports.ShownArtifact { return a.artifacts }

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

func TestShowAnswer_SendsTheShipmentAMovesIDWrittenAsOneStandsFor(t *testing.T) {
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

	require.False(t, outcome.Failed, outcome.Content)
	request, _ := outcome.Data.(agentruntimeports.BoardRequest)
	assert.Equal(t, []pulid.ID{pulid.ID(shipment)},
		request.Board.Records()["shipment"])
	assert.Equal(t, []any{shipment},
		outcome.RanWith["groups"].([]any)[0].(map[string]any)["records"])
}

func TestShowAnswer_SaysWhatAnIDWithTheWrongPrefixReallyIsWhenNothingCarriesItsRecord(
	t *testing.T,
) {
	t.Parallel()

	body := "01M7WG94MSAZY6J8J65G8H8KMN"
	turn := &answerTurn{messages: []llmports.Message{{
		Role:     llmports.RoleTool,
		ToolName: "dispatch_get_board",
		Content: toolresult.Fence("dispatch_get_board", `{"moves":[{"moveId":"sm_`+body+
			`","lateBy":"2d 2h"}],"shipments":[{"id":"shp_01M7WG94KWR38VFS2VQNHB2QWS"}]}`),
	}}}
	args := map[string]any{
		"title":  "Late loads",
		"answer": "There are {count} late loads.",
		"groups": []any{map[string]any{"records": []any{"shp_" + body}}},
	}

	outcome := showAnswer(turn, args)

	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "shp_"+body+" is a shipment move's id written as a "+
		"shipment's.")
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
				Content: toolresult.Fence("shipment_get",
					`{"id":"`+loadOne+`","proNumber":"SEED-SHP-001","totalCharge":"3200"}`),
			},
			{
				Role:    llmports.RoleUser,
				Content: "Compare SEED-SHP-008 and SEED-SHP-010: which is further behind?",
			},
			{
				Role: llmports.RoleTool, ToolName: "shipment_get",
				Content: toolresult.Fence("shipment_get", `{"items":[`+
					`{"id":"`+loadEight+`","proNumber":"SEED-SHP-008","totalCharge":"2400"}]}`),
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

const lateList = "art_01M861HFM3CMMC4RFVB9KTAFMT"

func pagedTurn(page string) *answerTurn {
	return pagedTurnKept(page, "")
}

func pagedTurnKept(page, kept string) *answerTurn {
	return &answerTurn{
		req: agentruntimeports.RunRequest{
			Input: "which loads are running late right now, and why is each one late?",
		},
		messages: []llmports.Message{{
			Role: llmports.RoleTool, ToolName: "shipment_list",
			Content: toolresult.Fence("shipment_list", `{"count":2,`+page+`"items":[`+
				`{"id":"`+loadEight+`","proNumber":"SEED-SHP-008"},`+
				`{"id":"`+loadTen+`","proNumber":"SEED-SHP-010"}]}`) + kept,
		}},
	}
}

func keptNote() string {
	return "\n\n[This result is kept as a table titled \"Shipments\" beside the conversation. " +
		"To point to it, write [Shipments](artifact:" + lateList + ") as part of the sentence.]"
}

func lateAnswer(answer string) map[string]any {
	return map[string]any{
		"title":   "Late loads",
		"answer":  answer,
		"measure": "lateness",
		"groups":  []any{map[string]any{"title": "Late", "records": []any{loadEight, loadTen}}},
	}
}

func TestShowAnswer_RefusesACountWhoseWordsDoNotSayWhichList(t *testing.T) {
	t.Parallel()

	outcome := showAnswer(pagedTurnKept(`"total":125,`, keptNote()),
		lateAnswer("{count} live loads are running late."))
	require.True(t, outcome.Failed)
	assert.Equal(t, `Tool "show_answer" was not run: {count} would read 2, the records shown, `+
		`but they are part of a longer list (it holds 125 in all), so it would state a number `+
		`that is not true. Where the words mean the whole list, write {all}: it reads the `+
		`list's own number, and again whenever the board is read ("{all} shipments"). Where `+
		`they mean only the records shown, say so: "these {count}", "the {count} shown here"; `+
		`a count of part of the whole list is a figure Trenova keeps (a metric's) or none.`,
		outcome.Content)

	outcome = showAnswer(pagedTurnKept(`"total":125,`, keptNote()),
		lateAnswer("The board shows {count} late moves in the first 17 rows."))
	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "(it holds 125 in all)")
}

func TestShowAnswer_ReadsTheWholeListFromTheTableItWasKeptAs(t *testing.T) {
	t.Parallel()

	outcome := showAnswer(pagedTurnKept(`"total":125,`, keptNote()),
		lateAnswer("{count} loads are running late right now."))
	require.True(t, outcome.Failed, "a count of a page is never read as the whole list's")
	assert.Contains(t, outcome.Content, "write {all}")

	outcome = showAnswer(pagedTurnKept(`"total":125,`, keptNote()),
		lateAnswer("{all} loads are late, and these two are the furthest behind."))

	require.False(t, outcome.Failed, outcome.Content)
	request, ok := outcome.Data.(agentruntimeports.BoardRequest)
	require.True(t, ok)
	all := request.Board.Reading("all")
	require.NotNil(t, all)
	assert.Equal(t, pulid.ID(lateList), all.List)
	assert.Nil(t, all.Records)
	assert.Equal(t, board.AggregateCount, all.Aggregate)
	assert.Contains(t, request.Board.Verdict, "{all}")

	ranWith := outcome.RanWith
	require.NotNil(t, ranWith, "the repaired call travels to the durable publish")
	assert.Contains(t, ranWith["answer"], "{all:"+lateList+"}")
	durable := AnswerStepOutcome(ranWith)
	require.False(t, durable.Failed, durable.Content)
	replayed, ok := durable.Data.(agentruntimeports.BoardRequest)
	require.True(t, ok)
	assert.Equal(t, request.Board.Readings, replayed.Board.Readings,
		"the durable publish reads the same list, not the records shown")
}

func TestShowAnswer_CountsTheRecordsShownWhenTheWordsSaySo(t *testing.T) {
	t.Parallel()

	for _, words := range []string{
		"Of the late loads, these {count} are the furthest behind.",
		"{all} loads are late; the {count} shown here are the furthest behind.",
	} {
		outcome := showAnswer(pagedTurnKept(`"total":125,`, keptNote()), lateAnswer(words))

		require.False(t, outcome.Failed, outcome.Content)
		request, ok := outcome.Data.(agentruntimeports.BoardRequest)
		require.True(t, ok)
		counted := request.Board.Reading("count")
		require.NotNil(t, counted, words)
		assert.Len(t, counted.Records.IDs, 2)
	}
}

func TestShowAnswer_RefusesASumOfThePageTiedToTheWholeList(t *testing.T) {
	t.Parallel()

	args := lateAnswer("There are {count} late loads on the board, with {total} of late " +
		"moves in the list of 125 late shipments.")
	args["measure"] = "revenue"

	outcome := showAnswer(pagedTurnKept(`"total":125,`, keptNote()), args)

	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "{total} reads only the 2 records shown, but its words "+
		"tie it to the whole list (125 in all)")

	args["answer"] = "{all} loads are late; these {count} bring in {total}."
	outcome = showAnswer(pagedTurnKept(`"total":125,`, keptNote()), args)
	assert.False(t, outcome.Failed, outcome.Content)
}

func TestShowAnswer_RefusesAGroupCountOfAPageSaidOfTheWholeList(t *testing.T) {
	t.Parallel()

	args := lateAnswer("{all} loads are late. Of those, {count_1} have no driver.")
	args["groups"] = []any{
		map[string]any{"title": "No driver", "records": []any{loadEight}},
		map[string]any{"title": "Off duty", "records": []any{loadTen}},
	}

	outcome := showAnswer(pagedTurnKept(`"total":125,`, keptNote()), args)

	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "{count_1} would read 1, the records shown")

	args["answer"] = "{all} loads are late; these {count_1} have no driver."
	outcome = showAnswer(pagedTurnKept(`"total":125,`, keptNote()), args)
	assert.False(t, outcome.Failed, outcome.Content)
}

func TestShowAnswer_RefusesTheWholeListWhenItKeptNoTotal(t *testing.T) {
	t.Parallel()

	outcome := showAnswer(pagedTurn(`"hasMore":true,`), lateAnswer("{all} loads are late."))
	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "reported no total it can be read again by")

	outcome = showAnswer(pagedTurn(`"hasMore":false,"total":2,`),
		lateAnswer("There are {count} late loads."))
	assert.False(t, outcome.Failed, outcome.Content)

	outcome = showAnswer(pagedTurn(`"hasMore":false,"total":2,`),
		lateAnswer("{all} loads are late."))
	require.False(t, outcome.Failed, outcome.Content)
	request, ok := outcome.Data.(agentruntimeports.BoardRequest)
	require.True(t, ok)
	assert.Len(t, request.Board.Reading("all").Records.IDs, 2,
		"a list read whole is counted from its records")

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

func TestShowAnswer_TakesACountOfTheShownWhenTheWordsTypeTheListsOwnTotal(t *testing.T) {
	t.Parallel()

	outcome := showAnswer(pagedTurn(`"total":40,`),
		lateAnswer("There are 40 late loads; {count} of them are on the road."))

	assert.False(t, outcome.Failed, outcome.Content)
}

const drafted = "art_01JDRAFT0000000000000001"

func TestShowAnswer_LinksAnArtifactThisTurnMade(t *testing.T) {
	t.Parallel()

	answer := peakAnswer("Peak owes {total}; I drafted [a reminder](artifact:" + drafted + ").")
	turn := peakTurn()

	refused := showAnswer(turn, answer)
	require.True(t, refused.Failed)
	assert.Equal(t, `Tool "show_answer" was not run: its words link `+drafted+`, which no `+
		`call in this turn made. Words may link only an artifact this turn made, as `+
		`[words](artifact:<id>) with the id its result gave; take the link out, or make the `+
		`artifact first.`, refused.Content)

	turn.artifacts = []agentruntimeports.ShownArtifact{{
		ID: pulid.ID(drafted), Kind: "email_draft", Title: "Reminder to Peak Distributing",
	}}
	outcome := showAnswer(turn, answer)
	require.False(t, outcome.Failed, outcome.Content)
	request, _ := outcome.Data.(agentruntimeports.BoardRequest)
	assert.Equal(t, []string{drafted}, request.Board.ArtifactLinks())
}

func TestShowAnswer_SendsABoardOfWorkersAboutTheirCredentialsToTheCredentials(t *testing.T) {
	t.Parallel()

	worker := "wrk_01M7V48DVVQGDF0KG409BJH1FJ"
	card := "wcred_01M7V48E7D6K4KRNHMBA7X53YW"
	turn := &answerTurn{messages: []llmports.Message{{
		Role:     llmports.RoleTool,
		ToolName: "worker_list_credentials",
		Content: toolresult.Fence("worker_list_credentials", `{"count":1,"items":[`+
			`{"id":"`+card+`","worker":{"id":"`+worker+`","wholeName":"Jane Doe"}}]}`),
	}}}

	outcome := showAnswer(turn, map[string]any{
		"title":  "Credentials lapsed",
		"answer": "{count} worker holds credentials that have lapsed.",
		"groups": []any{map[string]any{"title": "Lapsed", "records": []any{worker}}},
	})

	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "its words are about worker credentials, but its "+
		"records are workers")
	assert.Contains(t, outcome.Content, card)
}

func TestShowAnswer_LeavesAGroupOfARecordThisTurnRead(t *testing.T) {
	t.Parallel()

	turn := comparedTurn()
	turn.messages = append(turn.messages, llmports.Message{
		Role: llmports.RoleTool, ToolName: "shipment_get",
		Content: toolresult.Fence("shipment_get",
			`{"id":"`+loadOne+`","proNumber":"SEED-SHP-001","totalCharge":"3200"}`),
	})
	outcome := showAnswer(turn, comparedAnswer("SEED-SHP-010 and SEED-SHP-008 compared", loadOne))

	require.True(t, outcome.Failed, "SEED-SHP-001 was read in this turn and has its own group")
	assert.Contains(t, outcome.Content, "its words name SEED-SHP-008")
	assert.Contains(t, outcome.Content, "SEED-SHP-001 ("+loadOne+"), which this turn read")
}

func TestShowAnswer_SendsAMetricBoardBackOnceForTheRecordsItListsBesideIt(t *testing.T) {
	t.Parallel()

	turn := &answerTurn{req: agentruntimeports.RunRequest{
		Input: "how does our receivable aging compare with last month?",
	}}
	args := map[string]any{
		"title":  "Receivable aging against last month",
		"answer": "Open receivables are {openNow:agingCompare} today.",
		"groups": []any{map[string]any{"title": "Aging", "metric": "agingCompare"}},
	}

	outcome := showAnswer(turn, args)
	require.True(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "the agingCompare board lists beside it the open "+
		"invoices of the bucket that grew")
	assert.Contains(t, outcome.Content, "receivable_list_open_items")

	turn.messages = append(turn.messages, llmports.Message{
		Role: llmports.RoleTool, ToolName: ShowAnswerTool, IsError: true, Content: outcome.Content,
	})
	again := showAnswer(turn, args)
	assert.False(t, again.Failed, "it is asked for once: %s", again.Content)
}
