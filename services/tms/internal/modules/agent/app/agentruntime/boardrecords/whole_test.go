package boardrecords_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/boardrecords"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPointsAtShown_CountsOnlyWhatTheWordsPointAt(t *testing.T) {
	t.Parallel()

	for words, want := range map[string]bool{
		"Of the late loads, these {count} are the furthest behind.":         true,
		"The {count} shown here are the furthest behind.":                   true,
		"{count} loads are running late right now.":                         false,
		"{count} live loads are running late.":                              false,
		"{count} loads from this week are running late right now.":          false,
		"The dispatch board shows {count} late moves in the first 17 rows.": false,
		"Of those, {count} stays have no hold and can go on invoices now.":  false,
		"{count} of them have no driver.":                                   false,
		"{count} credentials have already lapsed.":                          false,
		"these {count} are late, and {count} loads have no driver.":         false,
	} {
		assert.Equal(t, want, boardrecords.PointsAtShown(words, "{count}"), words)
	}
}

const (
	pageOne = "shp_01M82X0YYRZZ87JRD7ZWH6WYP1"
	pageTwo = "shp_01M82VNP6842MNF8BD5CS5YY6V"
	kept    = "art_01M861HFM3CMMC4RFVB9KTAFMT"
)

func page(total, note string) string {
	return toolresult.Fence("shipment_list", `{"count":2,"hasMore":true,`+total+`"items":[`+
		`{"id":"`+pageOne+`","proNumber":"S26110986578623"},`+
		`{"id":"`+pageTwo+`","proNumber":"S26110933801752"}]}`) + note
}

func TestWholeOf_FindsTheTableALongerListWasKeptAs(t *testing.T) {
	t.Parallel()

	note := "\n\n[This result is kept as a table titled \"Shipments\" beside the " +
		"conversation. To point to it, write [Shipments](artifact:" + kept + ") as part of " +
		"the sentence.]"
	ids := []pulid.ID{pageOne, pageTwo}

	whole, ok := boardrecords.Of([]string{page(`"total":125,`, note)}).WholeOf(ids)
	require.True(t, ok)
	assert.Equal(t, boardrecords.WholeList{Table: kept, Total: 125}, whole)

	_, ok = boardrecords.Of([]string{page(`"total":125,`, "")}).WholeOf(ids)
	assert.False(t, ok, "a list kept as no table has nothing to read again")

	_, ok = boardrecords.Of([]string{page("", note)}).WholeOf(ids)
	assert.False(t, ok, "a list that reported no total has no number to read again")
}

const (
	workerOne  = "wrk_01M7V48DVVQGDF0KG409BJH1FJ"
	workerList = "art_01M861HFM3CMMC4RFVB9KTAFWK"
)

func workerPage() string {
	return toolresult.Fence("records_query", `{"count":1,"hasMore":true,"total":372,`+
		`"items":[{"id":"`+workerOne+`","name":"Ana Ruiz"}]}`) +
		"\n\n[This result is kept as a table titled \"Workers\" beside the conversation. " +
		"To point to it, write [Workers](artifact:" + workerList + ") as part of the sentence.]"
}

func TestAllNamed_ReadsTheListOfTheKindTheNounAfterAllCounts(t *testing.T) {
	t.Parallel()

	note := "\n\n[This result is kept as a table titled \"Shipments\" beside the " +
		"conversation. To point to it, write [Shipments](artifact:" + kept + ") as part of " +
		"the sentence.]"
	read := boardrecords.Of([]string{page(`"total":159,`, note), workerPage()})
	held := recordkind.Kind(permission.ResourceWorker)
	ids := []pulid.ID{workerOne}

	for words, want := range map[string]pulid.ID{
		"There are {all} new shipments waiting.":        kept,
		"{all} loads are new.":                          kept,
		"{all} drivers are active.":                     pulid.Nil,
		"{all} are waiting for a driver.":               pulid.Nil,
		"{all} of the shipments are new.":               kept,
		"All {all} of them fit; these drivers lead.":    pulid.Nil,
		"{all} new shipments, and {all} more shipments": kept,
	} {
		whole, elsewhere, refusal := read.AllNamed("show_answer", words, held, ids)
		require.Empty(t, refusal, words)
		assert.Equal(t, want.IsNotNil(), elsewhere, words)
		assert.Equal(t, want, whole.Table, words)
	}

	_, _, refusal := read.AllNamed("show_answer", "{all} invoices are open.", held, ids)
	assert.Contains(t, refusal, "which are workers (372 read), but the words count invoices")
}
