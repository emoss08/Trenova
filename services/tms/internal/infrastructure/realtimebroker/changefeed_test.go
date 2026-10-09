package realtimebroker

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const feedRecord = "shp_01J9FEEDFEEDFEEDFEEDFEED0"

func TestChangeSet_MergesEveryEventForOneRecord(t *testing.T) {
	t.Parallel()

	at := time.Unix(1_800_000_000, 0)
	set := newChangeSet()
	set.add(&changedRecord{RecordID: feedRecord, Action: "updated", Fields: []string{"status"},
		ActorType: "session_user", ActorUserID: "usr_a", OccurredAt: at})
	set.add(&changedRecord{EntityID: feedRecord, Action: "deleted", Fields: []string{"notes"},
		ActorType: "agent", OccurredAt: at.Add(time.Second)})
	set.add(&changedRecord{RecordID: feedRecord, Action: "updated", Fields: []string{"status"},
		ActorType: "system", OccurredAt: at.Add(-time.Second)})

	changes := set.changes()
	require.Len(t, changes, 1)
	assert.Equal(t, "deleted", changes[0].Action, "a deletion is not undone by a later update")
	assert.Equal(t, []string{"status", "notes"}, changes[0].Fields)
	assert.Equal(t, "agent", changes[0].ActorType, "the latest change names who made it")
	assert.Equal(t, at.Add(time.Second).Unix(), changes[0].At)
}

func TestRelevant_KeepsOnlyTenantWideInvalidationsNamingAWatchedRecord(t *testing.T) {
	t.Parallel()

	needles := [][]byte{[]byte(feedRecord)}
	payload := []byte(`{"resource":"audited:shipment","recordId":"` + feedRecord + `"}`)
	base := entry{tenant: "org:bu", event: services.RealtimeEventInvalidation, payload: payload}

	assert.True(t, relevant(&base, "org:bu", needles))

	other := base
	other.tenant = "org:other"
	assert.False(t, relevant(&other, "org:bu", needles), "another tenant's change")

	addressed := base
	addressed.audience = "usr_a"
	assert.False(t, relevant(&addressed, "org:bu", needles), "an event addressed to one person")

	presence := base
	presence.event = services.RealtimeEventPresence
	assert.False(t, relevant(&presence, "org:bu", needles))

	unrelated := base
	unrelated.payload = []byte(`{"resource":"audited:shipment","recordId":"shp_elsewhere"}`)
	assert.False(t, relevant(&unrelated, "org:bu", needles))
}
