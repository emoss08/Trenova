package boardrecords_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/boardrecords"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
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
