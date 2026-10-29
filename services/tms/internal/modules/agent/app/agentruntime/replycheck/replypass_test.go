package replycheck

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A reprint of a kept table is replaced by one sentence pointing to it, and
// the prose around it stays.
func TestPointToTables_ReplacesAReprintWithAPointer(t *testing.T) {
	t.Parallel()

	kept := runtimetest.KeptShipments(25)

	got, table := PointToTables(
		runtimetest.MarkdownReprint(8),
		[]agentruntimeports.ShownArtifact{kept},
	)

	require.NotNil(t, table)
	assert.Equal(t, kept.ID, table.ID)
	assert.Equal(t, "Here they are: "+ArtifactRef(&kept)+"\n\nCall the carriers first.", got)
	assert.Empty(t, StrayArtifactLinks(got))
}

// A few rows are an answer, a table of something else is not a reprint, and
// a table the reply already points to has been pointed to.
func TestPointToTables_LeavesWhatIsNotAReprint(t *testing.T) {
	t.Parallel()

	kept := runtimetest.KeptShipments(25)

	short := runtimetest.MarkdownReprint(minReprintRows - 1)
	got, table := PointToTables(short, []agentruntimeports.ShownArtifact{kept})
	assert.Equal(t, short, got)
	assert.Nil(t, table)

	other := "By customer:\n| Customer | Shipments |\n| --- | --- |\n" +
		strings.Repeat("| Acme | 3 |\n", 7)
	got, table = PointToTables(other, []agentruntimeports.ShownArtifact{kept})
	assert.Equal(t, other, got, "a per-customer summary is not the shipments table")
	assert.Nil(t, table)

	linked := "See " + ArtifactRef(&kept) + ".\n" + runtimetest.MarkdownReprint(8)
	got, table = PointToTables(linked, []agentruntimeports.ShownArtifact{kept})
	assert.Equal(t, linked, got)
	assert.Nil(t, table)
}

// The title is enough when the rows' values were not kept: the line above
// the reprint and its first header name what the table is.
func TestPointToTables_RecognisesAReprintByItsTitle(t *testing.T) {
	t.Parallel()

	kept := runtimetest.KeptShipments(0)
	reply := "The late shipments:\n| Shipment | Status |\n| --- | --- |\n" +
		strings.Repeat("| X | Late |\n", 7)

	_, table := PointToTables(reply, []agentruntimeports.ShownArtifact{kept})

	require.NotNil(t, table)
	assert.Equal(t, kept.ID, table.ID)
}

// A reply that reprinted two kept tables keeps neither copy: each is pointed
// to once, in its own place, and the prose between them stays.
func TestPointToTables_PointsEveryReprintToItsOwnTable(t *testing.T) {
	t.Parallel()

	late := runtimetest.KeptShipments(25)
	carriers := agentruntimeports.ShownArtifact{
		ID:     pulid.MustNew("art_"),
		Kind:   "table_view",
		Title:  "Carriers to call",
		Rows:   10,
		Labels: []string{"Acme", "Bolt", "Cargo", "Delta", "Eagle", "Falcon", "Giant", "Hawk"},
	}
	var carrierRows strings.Builder
	carrierRows.WriteString("| Carrier | Loads |\n| --- | --- |\n")
	for _, name := range carriers.Labels {
		fmt.Fprintf(&carrierRows, "| %s | 2 |\n", name)
	}
	reply := runtimetest.MarkdownReprint(8) + "\n\n" + carrierRows.String() + "\nThat is all."

	got, first := PointToTables(reply, []agentruntimeports.ShownArtifact{late, carriers})

	require.NotNil(t, first)
	assert.Equal(t, late.ID, first.ID, "the first table pointed to is reported")
	assert.Contains(t, got, "Here they are: "+ArtifactRef(&late))
	assert.Contains(t, got, "Call the carriers first ("+ArtifactRef(&carriers)+").")
	assert.Empty(t, StrayArtifactLinks(got))
	assert.NotContains(t, got, "| PRO-1")
	assert.NotContains(t, got, "| Acme |")
	assert.Contains(t, got, "That is all.")
}
