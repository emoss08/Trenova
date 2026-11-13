package watchtower

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
