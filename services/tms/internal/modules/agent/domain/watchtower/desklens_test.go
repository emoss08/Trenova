package watchtower

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeskLens_FilesDecisionsMessagesAndTheRest(t *testing.T) {
	assert.Equal(t, DeskLensDecisions, SourceAgentProposal.DeskLens())
	assert.Equal(t, DeskLensDecisions, SourceAgentPlan.DeskLens())
	assert.Equal(t, DeskLensTriage, SourceInboundMessage.DeskLens())
	assert.Equal(t, DeskLensWatchtower, SourceEtaAtRisk.DeskLens())
}

func TestDeskLens_KindsCoverEveryKindExactlyOnce(t *testing.T) {
	assert.Nil(t, DeskLensEverything.Kinds())
	seen := map[SourceKind]int{}
	for _, lens := range []DeskLens{DeskLensWatchtower, DeskLensTriage, DeskLensDecisions} {
		for _, kind := range lens.Kinds() {
			seen[kind]++
		}
	}
	for _, kind := range AllSourceKinds() {
		assert.Equal(t, 1, seen[kind], "kind %s", kind)
	}
}

func TestItem_SetAboutTakesTheKindFromTheID(t *testing.T) {
	item := &Item{}
	item.SetAbout("shp_01JABCDEFGHJKMNPQRSTVWXYZ0")

	assert.Equal(t, "shipment", string(item.AboutKind))
	assert.Equal(t, "shp_01JABCDEFGHJKMNPQRSTVWXYZ0", item.AboutID.String())
	assert.Equal(t, item.AboutID, item.GroupKey())
}

func TestItem_SetAboutClearsWhatItCannotName(t *testing.T) {
	item := &Item{ID: "wt_01JABCDEFGHJKMNPQRSTVWXYZ0"}
	item.SetAbout("zzz_01JABCDEFGHJKMNPQRSTVWXYZ0")

	assert.Empty(t, item.AboutKind)
	assert.True(t, item.AboutID.IsNil())
	assert.Equal(t, item.ID, item.GroupKey(), "an item about nothing beyond itself is its own row")
}
