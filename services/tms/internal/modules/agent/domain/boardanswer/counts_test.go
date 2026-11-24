package boardanswer_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/settlementboard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	pastLine = []any{"wrk_01JOVER00000000000000001", "wrk_01JOVER00000000000000002"}
	withRoom = []any{"wrk_01JROOM00000000000000001", "wrk_01JROOM00000000000000002",
		"wrk_01JROOM00000000000000003"}
)

func rota(answer string) map[string]any {
	return map[string]any{
		"title":   "Overtime risk this week",
		"answer":  answer,
		"measure": "hoursThisWeek",
		"groups": []any{
			map[string]any{"title": "Past the line", "records": pastLine},
			map[string]any{"title": "Room under the line", "records": withRoom},
		},
	}
}

func TestTranslate_RefusesAShareOfAGroupItIsNotPartOf(t *testing.T) {
	t.Parallel()

	problems := refused(t, rota("{count_1} of {count_2} workers on this week's rota are "+
		"projected past the 40-hour line."))

	assert.Contains(t, problems, "{count_1} of {count_2} would read 2 of 3, but group 1's "+
		"records are not among group 2's")
	assert.Contains(t, problems, "write {count_1} of {count}")

	answer := translated(t, rota("{count_1} of the {count} workers on the rota are projected "+
		"past the 40-hour line."))
	assert.Contains(t, answer.Board.Verdict, "{count_1} of the {count}")
}

func TestTranslate_ReadsWhatIsOwedWhenTheWordsSayOwed(t *testing.T) {
	t.Parallel()

	args := unpaid("Peak Distributing has {count} unpaid invoices, adding up to {total} " +
		"still owed.")
	args["measure"] = "total"

	answer := translated(t, args)

	total := answer.Board.Reading("total_amountDue")
	require.NotNil(t, total, answer.Board.Verdict)
	assert.Equal(t, board.Measure("amountDue"), total.Measure)
	assert.Equal(t, board.AggregateSum, total.Aggregate)
	assert.Equal(t, board.Measure("total"), answer.Board.Blocks[0].Measure,
		"the rows keep the measure the call named")
}

func TestTranslate_LeavesATotalWhoseWordsNameBothMeasures(t *testing.T) {
	t.Parallel()

	args := unpaid("They were billed {total}, and two are past due.")
	args["measure"] = "total"

	answer := translated(t, args)

	assert.NotNil(t, answer.Board.Reading("total"))
}

func TestTranslate_NamesTheFiguresAMetricLeadsWithWhenTooManyArePlaced(t *testing.T) {
	t.Parallel()

	problems := refused(t, metricAnswer(
		"It nets {batchNet:settlementBatch} across {batchSettlements:settlementBatch}, "+
			"paying {batchPayDate:settlementBatch}; {readyNet:settlementBatch} is ready on "+
			"{readySettlements:settlementBatch}, and {blockedSettlements:settlementBatch}, "+
			"worth {blockedNet:settlementBatch}, cannot be released: "+
			"{criticalExceptionSettlements:settlementBatch} have a critical exception.",
		map[string]any{"title": "Batch totals", "metric": settlementboard.MetricSettlementBatch},
	))

	assert.Contains(t, problems, "the answer and notes place 8 figures; at most 6 fit")
	assert.Contains(t, problems, "settlementBatch's answer leads with "+
		"{batchNet:settlementBatch}, {blockedSettlements:settlementBatch}, "+
		"{criticalExceptionSettlements:settlementBatch}")
}

func TestTranslate_TellsWhereAMetricsSubjectGoesWhenItIsListedAsARecord(t *testing.T) {
	t.Parallel()

	problems := refused(t, metricAnswer(
		"The return is not ready to file: {filingBlockers:iftaReturnChecks} check blocks it.",
		map[string]any{
			"title":  "What blocks filing",
			"metric": "iftaReturnChecks",
			"for":    "ifr_01M80ZFAMGBAG3MS1AQAV8EGSP",
		},
		map[string]any{"title": "The return", "records": []any{"ifr_01M80ZFAMGBAG3MS1AQAV8EGSP"}},
	))

	assert.Contains(t, problems, "it is the IFTA return the iftaReturnChecks metric reads, "+
		`so give it as that metric group's for: {"metric": "iftaReturnChecks", "for": `+
		`"ifr_01M80ZFAMGBAG3MS1AQAV8EGSP"}`)
	assert.Contains(t, problems, "fuel_get_ifta_return")
}

func TestTranslate_RefusesAOneGroupFigureBesideItsBareTwin(t *testing.T) {
	t.Parallel()

	problems := refused(t, unpaid("Peak Distributing has {count} unpaid invoices, totaling "+
		"{total}. Two of them, worth {total_1}, are past due."))

	assert.Contains(t, problems, "{total_1} is the one group's figure, the same as {total}, so "+
		"beside it it would read all 2 records again")

	answer := translated(t, unpaid("Peak Distributing owes {total_1} on {count_1} invoices."))
	assert.NotEmpty(t, answer.Board.Readings, "a numbered figure alone is still read")
}

func TestTranslate_ReadsTheWholeListATableWasKeptFrom(t *testing.T) {
	t.Parallel()

	answer := translated(t, unpaid("{all:art_01M861HFM3CMMC4RFVB9KTAFMT} invoices are open; "+
		"these {count} owe {total}."))

	all := answer.Board.Reading("all")
	require.NotNil(t, all)
	assert.Equal(t, "art_01M861HFM3CMMC4RFVB9KTAFMT", all.List.String())
	assert.Nil(t, all.Records)
	assert.Equal(t, "{all} invoices are open; these {count} owe {total}.", answer.Board.Verdict)
}
