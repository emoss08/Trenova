package watchtower

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/shared/pulid"
)

func TestResolutionOf_AnOpenItemHasNone(t *testing.T) {
	t.Parallel()

	assert.Nil(t, ResolutionOf(openItem(), ResolvedViaPerson))
	assert.Nil(t, ResolutionOf(nil, ResolvedViaPerson))
}

func TestResolutionOf_APersonsResolutionCarriesTheWork(t *testing.T) {
	t.Parallel()

	item := openItem()
	person := pulid.MustNew("usr_")
	record := pulid.MustNew("shp_")
	item.AboutID = record
	item.AssigneeID = person
	item.Version = 7
	item.EscalationLevel = 2
	acknowledged := item.OccurredAt + 120
	item.AcknowledgedAt = &acknowledged
	due := item.OccurredAt + 600
	item.DueAt = &due
	handed := item.OccurredAt + 300
	item.AgentHandedAt = &handed
	require.NoError(t, item.Resolve(ResolutionCodeFixed, "", person, item.OccurredAt+900))

	resolution := ResolutionOf(item, ResolvedViaPerson)
	require.NotNil(t, resolution)
	assert.Equal(t, item.ID, resolution.ItemID)
	assert.Equal(t, int64(7), resolution.ItemVersion)
	assert.Equal(t, record, resolution.CaseID, "the case is the record the item is about")
	assert.Equal(t, person, resolution.AssigneeID)
	assert.Equal(t, person, resolution.ResolvedByID)
	assert.Equal(t, ResolutionCodeFixed, resolution.ResolutionCode)
	assert.Equal(t, int64(900), resolution.SecondsToResolve)
	require.NotNil(t, resolution.SecondsToAcknowledge)
	assert.Equal(t, int64(120), *resolution.SecondsToAcknowledge)
	assert.True(t, resolution.PastDue)
	assert.True(t, resolution.AgentHanded)
	assert.Equal(t, int16(2), resolution.EscalationLevel)
}

func TestResolutionOf_ASourceClearIsNobodysWork(t *testing.T) {
	t.Parallel()

	item := openItem()
	resolved := item.OccurredAt + 60
	item.ResolvedAt = &resolved
	item.Status = StatusResolved
	item.ResolutionCode = ResolutionCodeCleared
	item.ResolvedByID = pulid.MustNew("usr_")

	resolution := ResolutionOf(item, ResolvedViaSourceCleared)
	require.NotNil(t, resolution)
	assert.True(t, resolution.ResolvedByID.IsNil())
	assert.Equal(t, item.ID, resolution.CaseID, "an item about nothing else is its own case")
	assert.Nil(t, resolution.SecondsToAcknowledge)
	assert.False(t, resolution.PastDue, "an item with no due time is never late")
	assert.False(t, ResolvedViaSourceCleared.ByPerson())
	assert.True(t, ResolvedViaAgent.ByPerson())
}
