package boardanswer_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardanswer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var credentials = []any{"wcred_01JCRED00000000000000001", "wcred_01JCRED00000000000000002"}

func unpaid(answer string) map[string]any {
	return map[string]any{
		"title":  "Peak Distributing's unpaid invoices",
		"answer": answer,
		"groups": []any{map[string]any{"title": "Open invoices", "records": invoices}},
	}
}

func running(answer string, records []any) map[string]any {
	return map[string]any{
		"title":  "Credentials running out",
		"answer": answer,
		"groups": []any{map[string]any{"title": "Running out", "records": records}},
	}
}

func TestTranslate_ReadsAKindsDefaultMeasureWhenNoneIsNamed(t *testing.T) {
	t.Parallel()

	answer := translated(t, unpaid("Peak Distributing owes {total} on {count} invoices."))

	assert.Equal(t, board.Measure("amountDue"), answer.Board.Blocks[0].Measure)
	assert.Equal(t, []string{"count", "total"}, keys(answer.Board.Readings))
	assert.Equal(t, "Total due", answer.Board.Readings[1].Label)
}

func TestTranslate_ReadsATotalOfOneRecordAsThatRecordsOwnFigure(t *testing.T) {
	t.Parallel()

	ten := dock[0].(string)
	args := comparison("SEED-SHP-010 brings in {total:" + ten + "}.")
	args["measure"] = "revenue"
	answer := translated(t, args)

	assert.Equal(t, "SEED-SHP-010 brings in {revenue_of_1}.", answer.Board.Verdict)
	require.Len(t, answer.Board.Readings, 1)
	assert.Equal(t, board.Measure("revenue"), answer.Board.Readings[0].Measure)
	assert.Equal(t, board.AggregateMax, answer.Board.Readings[0].Aggregate)
}

func TestTranslate_ReadsAShipmentsTotalAsItsRevenue(t *testing.T) {
	t.Parallel()

	args := oneGroup()
	args["measure"] = "total"
	args["answer"] = "They bring in {total}."
	answer := translated(t, args)

	assert.Equal(t, board.Measure("revenue"), answer.Board.Blocks[0].Measure)
}

func TestTranslate_CutsANoteOverItsLimitToWholeSentences(t *testing.T) {
	t.Parallel()

	first := strings.Repeat("Windows closed with no arrival recorded. ", 8)
	args := oneGroup()
	args["groups"].([]any)[0].(map[string]any)["note"] = first + "Drivers are off duty."
	answer := translated(t, args)

	caption := answer.Board.Blocks[0].Caption
	assert.LessOrEqual(t, len(caption), board.MaxCaptionRunes)
	assert.True(t, strings.HasSuffix(caption, "recorded."))
	assert.NotContains(t, caption, "off duty")
}

func TestTranslate_RefusesTypedDaysBesideACountdown(t *testing.T) {
	t.Parallel()

	_, problems := boardanswer.Translate(running(
		"Jane Doe's card lapsed 8 days ago and Robert Brown's runs out in 8 days.", credentials,
	))

	require.Len(t, problems, 1)
	assert.Equal(t, `the words type "8 days", a count of days that the rows' `+
		`expiry reads live and that is wrong by tomorrow. Write the day it falls on instead `+
		`("on Dec 1"), {largest} for the soonest, or one record's own as {expiry:<its id>}`,
		problems[0])
}

func TestTranslate_ReadsTypedDaysOfOneCountdownAsItsReading(t *testing.T) {
	t.Parallel()

	answer := translated(t, running(
		"Robert Brown's medical card runs out in 8 days.", credentials[:1],
	))

	assert.Equal(t, "Robert Brown's medical card runs out in {expiry_of_1}.",
		answer.Board.Verdict)
	require.Len(t, answer.Board.Readings, 1)
	assert.Equal(t, board.Measure("expiry"), answer.Board.Readings[0].Measure)
}

func TestTranslate_LeavesAWindowOfDaysAlone(t *testing.T) {
	t.Parallel()

	answer := translated(t, running(
		"{count} credentials run out within the next 30 days.", credentials,
	))

	assert.Equal(t, "{count} credentials run out within the next 30 days.",
		answer.Board.Verdict)
}

func TestTranslate_KeepsAnArtifactLinkInTheWords(t *testing.T) {
	t.Parallel()

	args := oneGroup()
	args["answer"] = "There are {count} late loads; I drafted [a note to the dock]" +
		"(artifact:art_01JDRAFT0000000000000001)."
	answer := translated(t, args)

	assert.Equal(t, []string{"art_01JDRAFT0000000000000001"}, answer.Board.ArtifactLinks())
}

func TestTranslate_ReadsABareTotalAsTheOneMeasureThatAddsUp(t *testing.T) {
	t.Parallel()

	args := oneGroup()
	delete(args, "measure")
	args["answer"] = "The {count} open loads bring in {total}; the most valuable is {largest}."
	answer := translated(t, args)

	assert.Equal(t, board.Measure("revenue"), answer.Board.Blocks[0].Measure)
	assert.Equal(t, []string{"count", "total", "largest"}, keys(answer.Board.Readings))

	args["answer"] = "Together they are {total} behind."
	_, problems := boardanswer.Translate(args)
	require.Len(t, problems, 1)
	assert.Contains(t, problems[0], "as {total:revenue}",
		"a total written as time is not read as an amount, and the example names one that adds up")
}

func TestTranslate_ReadsABareTotalBesideEachLoadsOwnLatenessAsRevenue(t *testing.T) {
	t.Parallel()

	ten, eight := dock[0].(string), held[0].(string)
	args := comparison("SEED-SHP-010 closed {lateness:" + ten + "} ago, against {lateness:" + eight +
		"}. Together they bring in {total}, with SEED-SHP-010 at {revenue:" + ten + "}.")
	delete(args, "measure")
	answer := translated(t, args)

	assert.Contains(t, keys(answer.Board.Readings), "total")
	assert.Equal(t, board.Measure("revenue"), answer.Board.Blocks[0].Measure)
}

func TestTranslate_TakesAGroupsTitleForAMissingTitleAndCutsALongOneAtAWord(t *testing.T) {
	t.Parallel()

	args := oneGroup()
	delete(args, "title")
	group := args["groups"].([]any)[0].(map[string]any)
	group["title"] = "Keystone Partners, Tri-State Lumber and Rodriguez & Evans loads due today"
	answer := translated(t, args)

	assert.Equal(t, "Keystone Partners, Tri-State Lumber and Rodriguez & Evans",
		answer.Board.Title)
	assert.Equal(t, "Keystone Partners, Tri-State Lumber and Rodriguez & Evans",
		answer.Board.Blocks[0].Title)
}
