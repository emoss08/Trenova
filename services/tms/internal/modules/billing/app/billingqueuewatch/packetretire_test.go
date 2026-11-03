package billingqueuewatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestARetiredDocumentLeavesThePacketWithItsTurns(t *testing.T) {
	pod, bol, receipt := pulid.MustNew("doc_"), pulid.MustNew("doc_"), pulid.MustNew("doc_")
	item := &billingqueue.BillingQueueItem{
		PacketDocumentIDs: []pulid.ID{pod, bol, receipt},
		PacketPageRotations: []billingqueue.PacketPageRotation{
			{DocumentID: bol, Page: 1, Rotation: 90},
			{DocumentID: receipt, Page: 2, Rotation: 180},
		},
	}

	packet, rotations, changed := withoutRetired(item, map[pulid.ID]pulid.ID{bol: ""})

	assert.True(t, changed)
	assert.Equal(t, []pulid.ID{pod, receipt}, packet)
	assert.Equal(t, []billingqueue.PacketPageRotation{
		{DocumentID: receipt, Page: 2, Rotation: 180},
	}, rotations)
}

func TestANewVersionTakesTheOldOnesPlaceInThePacket(t *testing.T) {
	pod, bol, newBOL := pulid.MustNew("doc_"), pulid.MustNew("doc_"), pulid.MustNew("doc_")
	item := &billingqueue.BillingQueueItem{PacketDocumentIDs: []pulid.ID{pod, bol}}

	packet, _, changed := withoutRetired(item, map[pulid.ID]pulid.ID{bol: newBOL})

	assert.True(t, changed)
	assert.Equal(t, []pulid.ID{pod, newBOL}, packet)
}

func TestANewVersionAlreadyInThePacketIsNotCarriedTwice(t *testing.T) {
	bol, newBOL := pulid.MustNew("doc_"), pulid.MustNew("doc_")
	item := &billingqueue.BillingQueueItem{PacketDocumentIDs: []pulid.ID{newBOL, bol}}

	packet, _, changed := withoutRetired(item, map[pulid.ID]pulid.ID{bol: newBOL})

	assert.True(t, changed)
	assert.Equal(t, []pulid.ID{newBOL}, packet)
}

func TestAPacketThatCarriesNoRetiredDocumentIsLeftAlone(t *testing.T) {
	pod := pulid.MustNew("doc_")
	item := &billingqueue.BillingQueueItem{PacketDocumentIDs: []pulid.ID{pod}}

	_, _, changed := withoutRetired(item, map[pulid.ID]pulid.ID{pulid.MustNew("doc_"): ""})

	assert.False(t, changed)
}
