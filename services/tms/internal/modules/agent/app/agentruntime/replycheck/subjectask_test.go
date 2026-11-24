package replycheck

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardhint"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	q4Return    = "ifr_01M80ZFAMGBAG3MS1AQAV8EGSP"
	acmePartner = "edip_01M7V48HPFRT5N51S07BK4QZ5C"
	iftaAsked   = "are we ready to file the fourth quarter's IFTA?"
)

var q4List = toolresult.Fence("fuel_list_ifta_returns", `{"count":1,"items":[{"id":"`+
	q4Return+`","quarter":4,"year":2026,"status":"Draft","netDue":"5137.34","problems":[`+
	`{"code":"MissingRate","message":"No Diesel tax rate is on file for OK – Oklahoma in Q4 2026"},`+
	`{"code":"NoTractorMiles","message":"235 completed moves have no tractor assigned"}]}],`+
	`"total":1}`)

const iftaProse = "The Q4 2026 return is not ready to file. It is still a Draft, and two " +
	"problems block filing:\n\n1. **No Diesel tax rate for Oklahoma in Q4 2026.** Rates are " +
	"configuration.\n2. **235 completed moves have no tractor assigned.**\n\nThe net tax due is " +
	"currently $5,137.34, and the return is due January 31, 2027."

func askedTurn(asked, tool, content string) *boardTurn {
	turn := &boardTurn{messages: []llmports.Message{
		{Role: llmports.RoleUser, Content: asked},
		{Role: llmports.RoleTool, ToolName: tool, ToolCallID: "call_1", Content: content},
	}}
	turn.req.Input = asked

	return turn
}

func TestWithBoardNote_NamesTheMetricOfTheOneRecordAResultIsAbout(t *testing.T) {
	t.Parallel()

	noted := WithBoardNote(askedTurn(iftaAsked, "fuel_list_ifta_returns", q4List),
		"fuel_list_ifta_returns", q4List, false, nil)

	line, metric := boardhint.SubjectLine(pulid.ID(q4Return))
	require.Equal(t, "iftaReturnChecks", metric)
	assert.Equal(t, q4List+"\n\n["+line+"]", noted,
		"final-a #2 and #3 read the list alone and never learned the checks metric existed")
}

func TestWithBoardNote_FillsAListHintWithThePartnerEveryRowBelongsTo(t *testing.T) {
	t.Parallel()

	own := boardhint.Line(&boardhint.Show{Metric: "partnerDay"})
	row := func(id string) string {
		return `{"id":"` + id + `","bol":"88244","status":"Approved","counterparty":` +
			`{"code":"ACMEMFG","id":"` + acmePartner + `"}}`
	}
	content := toolresult.Fence("edi_list_transfers", `{"count":2,"howToShow":`+
		quoteJSON(own)+`,"items":[`+row("edilt_01M85JM3GBC6GG1DN85XW6W620")+","+
		row("edilt_01M85JM3FY0CV3268MED8VB0GD")+`]}`)

	noted := WithBoardNote(askedTurn("did all of today's tenders go through?",
		"edi_list_transfers", content), "edi_list_transfers", content, false, nil)

	assert.Contains(t, own, `"for": "the EDI partner's id"`)
	assert.True(t, strings.HasSuffix(noted, `"for": "`+acmePartner+`", "period": "…"}]}. `+
		noteTail(noted)), "the trailing note names Acme's id, not a placeholder")
}

func noteTail(noted string) string {
	at := strings.LastIndex(noted, `"period": "…"}]}. `)
	return noted[at+len(`"period": "…"}]}. `):]
}

func quoteJSON(text string) string {
	return `"` + strings.ReplaceAll(text, `"`, `\"`) + `"`
}

func TestAskForBoard_SendsBackChecksOfOneRecordSetOutInProse(t *testing.T) {
	t.Parallel()

	turn := askedTurn(iftaAsked, "fuel_list_ifta_returns", q4List)
	state := BoardAskState{}
	back, ask := AskForBoard(turn, &llmports.ChatCompletionResult{Text: iftaProse}, &state)

	require.True(t, ask, "final-a #1: a numbered prose answer about one return's checks")
	correction := back.Messages[len(back.Messages)-1].Content
	assert.True(t, strings.HasPrefix(correction, subjectCorrectionLead))
	assert.Contains(t, correction, `"metric": "iftaReturnChecks", "for": "`+q4Return+`"`)
	assert.True(t, isBoardCorrection(correction))
	assert.True(t, state.Asked)
}

func TestAskForBoard_LeavesOneFactAboutTheSubjectAsASentence(t *testing.T) {
	t.Parallel()

	turn := askedTurn("what did we owe on last quarter's IFTA return?", "fuel_list_ifta_returns",
		q4List)
	reply := "Last quarter's IFTA return (Q3 2026) was filed, and it came to **$8,836.25** net " +
		"due.\n\nThe return still carries one problem: 313 completed moves have no tractor."

	_, ask := AskForBoard(turn, &llmports.ChatCompletionResult{Text: reply}, &BoardAskState{})

	assert.False(t, ask, "the IFTA twin: one figure, no lines set out")
}

func TestAskForBoard_LeavesRecordsNamedInSeparateParagraphsAsProse(t *testing.T) {
	t.Parallel()

	turn := askedTurn("whats the BOL on the freshhaul reefer going to LA?", "shipment_list",
		fencedResult("shipment_list", twoLateLoads))
	reply := "The open FreshHaul load headed to Los Angeles is **SEED-SHP-010**, with BOL " +
		"**BOL-2026-0010**.\n\nThe other FreshHaul load going there, **SEED-SHP-008**, is " +
		"already invoiced, so it is not the one you are after."

	_, ask := AskForBoard(turn, &llmports.ChatCompletionResult{Text: reply}, &BoardAskState{})

	assert.False(t, ask, "final-a one-fact #1: a second load named only to set it aside")
}

func TestAskForBoard_SendsBackAQuestionAnsweredAsAListTheResultSaysToDraw(t *testing.T) {
	t.Parallel()

	table := pulid.MustNew("art_")
	hint := boardhint.Line(&boardhint.Show{Metric: "unappliedReceipts"})
	content := toolresult.Fence("receivable_list_bank_receipt_exceptions",
		`{"count":50,"hasMore":true,"total":56,"howToShow":`+quoteJSON(hint)+`,"items":[`+
			`{"id":"brcpt_01M86CR1C7R8R45SR0N4PA2SMV","referenceNumber":"SEED-INV-0006"},`+
			`{"id":"brcpt_01M86CR1CZVBBZSJMPSVRRNT8K","referenceNumber":"SEED-INV-0008"}]}`)
	reply := "There are **56** bank receipts not yet applied. The details are in " +
		"[Bank receipt exceptions](artifact:" + table.String() + ").\n\n" +
		"- **$2,531.25**, 11/23, reference SEED-INV-0006\n" +
		"- **$3,870.10**, 11/22, reference SEED-INV-0008"
	shown := agentruntimeports.ShownArtifact{
		ID: table, Kind: "table_view", Title: "Bank receipt exceptions", Rows: 50, Paged: true,
		RowKind: "bank_receipt", Labels: []string{"SEED-INV-0006", "SEED-INV-0008"},
	}

	asked := askedTurn("what came into the bank that i haven't applied yet?",
		"receivable_list_bank_receipt_exceptions", content)
	asked.artifacts = []agentruntimeports.ShownArtifact{shown}
	_, ask := AskForBoard(asked, &llmports.ChatCompletionResult{Text: reply}, &BoardAskState{})
	assert.True(t, ask, "final-a receipts #2: the question answered as a list beside its table")

	listed := askedTurn("show me the bank receipt exceptions, i'll go through them",
		"receivable_list_bank_receipt_exceptions", content)
	listed.artifacts = []agentruntimeports.ShownArtifact{shown}
	_, ask = AskForBoard(listed, &llmports.ChatCompletionResult{Text: reply}, &BoardAskState{})
	assert.False(t, ask, "the twin asked to see the list, and asked no question")
}

func TestWithBoardNote_SaysHowToCountThePeopleCredentialsBelongTo(t *testing.T) {
	t.Parallel()

	content := toolresult.Fence("worker_list_credentials", `{"count":2,"items":[`+
		`{"id":"wcred_01M7V48E7D6K4KRNHMBA7X53YW","worker":{"id":"wrk_01M7V48DVVQGDF0KG409BJH1FJ",`+
		`"firstName":"Jane","lastName":"Doe"}},`+
		`{"id":"wcred_01M7V48E6DV5X1P0NVEWKRCAA0","worker":{"id":"wrk_01M7V48DWSAX7AFMQ3W2SHBHHR",`+
		`"firstName":"Robert","lastName":"Brown"}}]}`)

	noted := WithBoardNote(askedTurn("whose credentials run out?", "worker_list_credentials",
		content), "worker_list_credentials", content, false, nil)

	assert.Contains(t, noted, "write {count:workers}, or {count_1:workers} for one group's, "+
		"never a typed number", "final-a #1 typed \"three drivers\" beside the credentials")
}
