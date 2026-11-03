package billingqueue

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestAnEmptyPickIsStillThePacket(t *testing.T) {
	item := &BillingQueueItem{}

	item.PickPacket(nil, true)

	packet, picked := item.PickedPacket()
	assert.True(t, picked)
	assert.Equal(t, []pulid.ID{}, packet)
}

func TestNamingADocumentPicksThePacket(t *testing.T) {
	pod := pulid.MustNew("doc_")
	item := &BillingQueueItem{}

	item.PickPacket([]pulid.ID{pod}, false)

	packet, picked := item.PickedPacket()
	assert.True(t, picked)
	assert.Equal(t, []pulid.ID{pod}, packet)
}

func TestAnItemNobodyPickedForFallsBackToTheRequiredDocuments(t *testing.T) {
	item := &BillingQueueItem{}

	item.PickPacket(nil, false)

	packet, picked := item.PickedPacket()
	assert.False(t, picked)
	assert.Nil(t, packet)
}
