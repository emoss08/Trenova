package boardanswer_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardanswer"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardtyped"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	janeDoe       = "wrk_01M7V48DVVQGDF0KG409BJH1FJ"
	robertBrown   = "wrk_01M7V48DWSAX7AFMQ3W2SHBHHR"
	sarahWilliams = "wrk_01M7V48DWFF0P2CGB7FEPECRM2"
)

var (
	lapsed = []any{"wcred_01M7V48E7D6K4KRNHMBA7X53YW", "wcred_01M7V48E7E1FF9PPHATT0922WK"}
	runOut = []any{"wcred_01M7V48E6DV5X1P0NVEWKRCAA0", "wcred_01M7V48E6F093SGBQZSGB1S0AJ",
		"wcred_01M7V48E6VS8YV6Y2D0Y9TZGTS", "wcred_01M7V48E6XQDHKSZNW85A0GP3W"}
	whoseOne = map[string]any{
		"wcred_01M7V48E7D6K4KRNHMBA7X53YW": janeDoe,
		"wcred_01M7V48E7E1FF9PPHATT0922WK": janeDoe,
		"wcred_01M7V48E6DV5X1P0NVEWKRCAA0": robertBrown,
		"wcred_01M7V48E6F093SGBQZSGB1S0AJ": robertBrown,
		"wcred_01M7V48E6VS8YV6Y2D0Y9TZGTS": sarahWilliams,
		"wcred_01M7V48E6XQDHKSZNW85A0GP3W": sarahWilliams,
	}
)

func credentialsAnswer(answer string) map[string]any {
	return map[string]any{
		"title":   "Credentials lapsed or running out",
		"answer":  answer,
		"measure": "expiry",
		"groups": []any{
			map[string]any{"title": "Already lapsed", "records": lapsed},
			map[string]any{"title": "Running out", "records": runOut},
		},
		boardtyped.PartOfField: whoseOne,
	}
}

func TestTranslate_CountsThePeopleTheCredentialsBelongTo(t *testing.T) {
	t.Parallel()

	answer := translated(t, credentialsAnswer(
		"{count} credentials across {count:workers} drivers, {count_1:workers} of them lapsed."))

	all := answer.Board.Reading("count_worker")
	require.NotNil(t, all, "final-a #2: {count} drivers needed workers' records to count")
	assert.Equal(t, board.AggregateCount, all.Aggregate)
	assert.Equal(t, recordkind.Kind("worker"), all.Records.Kind)
	assert.ElementsMatch(t, []pulid.ID{janeDoe, robertBrown, sarahWilliams}, all.Records.IDs)
	first := answer.Board.Reading("count_1_worker")
	require.NotNil(t, first)
	assert.Equal(t, []pulid.ID{janeDoe}, first.Records.IDs)
	assert.Equal(t, "Workers, Already lapsed", first.Label)
	assert.Len(t, answer.Board.Reading(boardanswer.NameCount).Records.IDs, 6,
		"{count} still counts the credentials")
}

func TestTranslate_PlacesATypedCountOfPeopleThatIsExact(t *testing.T) {
	t.Parallel()

	answer := translated(t, credentialsAnswer(
		"{count} credentials across three drivers have lapsed or run out by Dec 9."))

	assert.Contains(t, answer.Board.Verdict, "{count_worker} drivers have lapsed",
		"final-a #1 typed \"three drivers\"; three is how many people the credentials belong to")
	assert.NotNil(t, answer.Board.Reading("count_worker"))
	assert.Equal(t, "{count} credentials across {count_worker} drivers have lapsed or run out "+
		"by Dec 9.", answer.Board.Verdict)
}

func TestTranslate_RefusesATypedCountOfPeopleThatIsWrong(t *testing.T) {
	t.Parallel()

	_, problems := boardanswer.Translate(credentialsAnswer(
		"Two drivers have already lapsed, and {count_2} credentials run out soon."))

	require.Len(t, problems, 1)
	assert.Contains(t, problems[0], `the words type "Two drivers"`,
		"z-cred-1 #2 said two drivers had lapsed when one had")
	assert.Contains(t, problems[0], "they belong to 3 in all, {count:workers}")
	assert.Contains(t, problems[0], "1 in group 1, {count_1:workers}; 2 in group 2, "+
		"{count_2:workers}")
}

func TestTranslate_LeavesACountOfTheCredentialsThemselvesAlone(t *testing.T) {
	t.Parallel()

	answer := translated(t, credentialsAnswer(
		"{count:credentials} credentials run out, and 6 worker credentials are shown."))

	count := answer.Board.Reading(boardanswer.NameCount)
	require.NotNil(t, count)
	assert.Len(t, count.Records.IDs, 6)
	assert.Nil(t, answer.Board.Reading("count_worker"))
}

func TestTranslate_RefusesCountingPeopleWithoutKnowingWhoseEachIs(t *testing.T) {
	t.Parallel()

	args := credentialsAnswer("{count:workers} drivers have credentials running out.")
	delete(args, boardtyped.PartOfField)
	_, problems := boardanswer.Translate(args)

	require.NotEmpty(t, problems)
	assert.Contains(t, strings.Join(problems, "; "),
		"did not read whose each one is; write {count} for the records themselves")
}

func TestTranslate_RefusesADurationWordedAsADay(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"a metric figure after since (z2-r4 close #2, final-a close #1)": "{pendingEntries} " +
			"entries wait to post, the oldest waiting since {oldestPending:periodClose}.",
		"a metric figure after from (z2-r3 receipts #3)": "{pendingEntries} entries wait, " +
			"the oldest dating from {oldestPending:periodClose}.",
	}
	for name, words := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, problems := boardanswer.Translate(map[string]any{
				"title":  "What holds the close",
				"answer": strings.ReplaceAll(words, "{pendingEntries}", "{pendingEntries:periodClose}"),
				"groups": []any{map[string]any{
					"title":  "Close checks",
					"metric": "periodClose",
					"for":    "fp_01M7V48EJ3MZHT1XMECQED4H71",
				}},
			})

			joined := strings.Join(problems, "; ")
			assert.Contains(t, joined, "which is how long, yet")
			assert.Contains(t, joined, `"has waited {oldestPending:periodClose}"`)
		})
	}

	answer := translated(t, map[string]any{
		"title":  "What holds the close",
		"answer": "The oldest entry has waited {oldestPending:periodClose} days.",
		"groups": []any{map[string]any{
			"title": "Close checks", "metric": "periodClose", "for": "fp_01M7V48EJ3MZHT1XMECQED4H71",
		}},
	})
	assert.Equal(t, "The oldest entry has waited {oldestPending_1}.", answer.Board.Verdict,
		"a unit typed after a metric's time figure is dropped too")
}

func TestTranslate_SplitsTheSixFiguresBetweenTwoMetricGroups(t *testing.T) {
	t.Parallel()

	_, problems := boardanswer.Translate(map[string]any{
		"title": "Fuel cost per mile this month",
		"answer": "{currentCostPerMile:fuelCostPerMile} a mile at {pricePerGallon:fuelCostPerMile} " +
			"a gallon. {tractorsAboveRange:fuelCostByTractor} above, {tractorsBelowRange:" +
			"fuelCostByTractor} below, {highestTractor:fuelCostByTractor} at {highestCostPerMile:" +
			"fuelCostByTractor}, {lowestCostPerMile:fuelCostByTractor} the lowest.",
		"groups": []any{
			map[string]any{"title": "Cost per mile", "metric": "fuelCostPerMile"},
			map[string]any{"title": "Each tractor", "metric": "fuelCostByTractor"},
		},
	})

	require.Len(t, problems, 1)
	assert.Contains(t, problems[0], "place 7 figures; at most 6 fit")
	assert.Contains(t, problems[0], "fuelCostByTractor's answer leads with "+
		"{tractorsAboveRange:fuelCostByTractor}, {highestTractor:fuelCostByTractor}, "+
		"{highestCostPerMile:fuelCostByTractor}, and its 6 other figures",
		"beyond-i-2: four leads of one metric beside three of the other left no room")
}
