package watchtower

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
No source can put an internal record id on the feed: an item's title,
summary, facts, impact and trail lose any id as the item is stored, and a
fact that held nothing but an id is dropped.
*/
func TestNormalize_TakesRecordIDsOutOfEverythingAPersonReads(t *testing.T) {
	t.Parallel()

	item := &Item{
		Title:   "Load monitor could not finish a run (ar_01M717A3AHMP2CBW5SSW8KZ5K1)",
		Summary: "The agent filed ap_01M71M4NJFEKJC94HTSATDTMHZ for a person.",
		Facts: []Fact{
			{Label: "Agent", Value: "Load monitor"},
			{Label: "Run", Value: "ar_01M717A3AHMP2CBW5SSW8KZ5K1"},
		},
		Impact: []string{"shp_01M717A20G2628JTGTZQE08QG9", "SEED-SHP-010"},
		Trail:  []TrailEvent{{At: 1, Text: "Started run ar_01M717A3AHMP2CBW5SSW8KZ5K1"}},
	}

	item.Normalize()

	assert.Equal(t, "Load monitor could not finish a run", item.Title)
	assert.NotContains(t, item.Summary, "ap_")
	require.Len(t, item.Facts, 1, "a fact that was only an id is dropped")
	assert.Equal(t, "Agent", item.Facts[0].Label)
	assert.Equal(t, []string{"SEED-SHP-010"}, item.Impact)
	require.Len(t, item.Trail, 1)
	assert.NotContains(t, item.Trail[0].Text, "ar_")
}
