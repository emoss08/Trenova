package boardtyped_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/boardtyped"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var credential = recordkind.Kind(permission.ResourceWorkerCredential)

func TestDaysAreTheCountsOfDaysACountdownReadsLive(t *testing.T) {
	t.Parallel()

	typed := boardtyped.Days("Robert's expire in 7 days, Jane's lapsed 9 days ago, both 7 days left.")
	assert.Equal(t, []string{"7 days", "9 days"}, typed)
	assert.Equal(t, "Jane's lapsed {expiry:x} ago.",
		boardtyped.MarkDays("Jane's lapsed 9 days ago.", "{expiry:x}"))
}

func TestDayWordIsTheWordThatMakesAFigureADay(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "since", boardtyped.DayWord("the oldest has waited since "))
	assert.Equal(t, "from", boardtyped.DayWord("the oldest dates from "))
	assert.Empty(t, boardtyped.DayWord("the oldest has waited "))
	assert.Empty(t, boardtyped.DayWord("a fronted sum "))
}

func TestAWorkerCredentialIsPartOfAWorkerAndPeopleCallWorkersDrivers(t *testing.T) {
	t.Parallel()

	whole, part := boardtyped.WholeOf(credential)
	require.True(t, part)
	assert.Equal(t, recordkind.Kind(permission.ResourceWorker), whole.Kind)
	assert.True(t, whole.Names("drivers"))
	assert.True(t, whole.Names("People"))
	assert.False(t, whole.Names("credentials"))

	_, part = boardtyped.WholeOf(recordkind.Kind(permission.ResourceShipment))
	assert.False(t, part, "a shipment is part of nothing")
}

func TestPlacePutsTheReadingWhereAnExactCountOfPeopleIsTyped(t *testing.T) {
	t.Parallel()

	whole, _ := boardtyped.WholeOf(credential)
	cases := map[string]struct {
		text   string
		placed string
		wrong  []string
	}{
		"the true number of people (final-a credentials #1)": {
			text:   "6 credentials across three drivers have lapsed or run out.",
			placed: "6 credentials across {count:workers} drivers have lapsed or run out.",
		},
		"a number that is not how many people (z-cred-1 #2)": {
			text:   "Two drivers have already lapsed.",
			placed: "Two drivers have already lapsed.",
			wrong:  []string{"Two drivers"},
		},
		"a count of the credentials themselves": {
			text:   "3 worker credentials and 3 people.",
			placed: "3 worker credentials and {count:workers} people.",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			placed, wrong := whole.Place(tc.text, credential, 3)
			assert.Equal(t, tc.placed, placed)
			assert.Equal(t, tc.wrong, nilIfEmpty(wrong))
		})
	}
}

func nilIfEmpty(texts []string) []string {
	if len(texts) == 0 {
		return nil
	}

	return texts
}

func TestDaysIncludeCountsWrittenInWordsOrInParentheses(t *testing.T) {
	t.Parallel()

	typed := boardtyped.Days("Jane's lapsed on Nov 15, nine days ago; Robert's run out on Dec 1 " +
		"(7 days). Both took 30 days to verify.")
	assert.Equal(t, []string{"nine days", "7 days"}, typed,
		"z2-final credentials #1 typed both into a board's verdict")
}
