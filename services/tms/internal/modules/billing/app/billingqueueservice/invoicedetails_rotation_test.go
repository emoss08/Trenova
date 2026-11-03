package billingqueueservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func turn(documentID pulid.ID, page, rotation int) billingqueue.PacketPageRotation {
	return billingqueue.PacketPageRotation{DocumentID: documentID, Page: page, Rotation: rotation}
}

func TestPacketRotationsKeepTheLastTurnOfEachPageInPacketOrder(t *testing.T) {
	pod, bol, gone := pulid.MustNew("doc_"), pulid.MustNew("doc_"), pulid.MustNew("doc_")

	got, err := packetRotations([]billingqueue.PacketPageRotation{
		turn(bol, 1, 90),
		turn(pod, 3, 90),
		turn(pod, 3, -90),
		turn(pod, 1, 450),
		turn(gone, 1, 90),
		turn(bol, 2, 360),
	}, []pulid.ID{pod, bol})

	require.NoError(t, err)
	assert.Equal(t, []billingqueue.PacketPageRotation{
		turn(pod, 1, 90),
		turn(pod, 3, 270),
		turn(bol, 1, 90),
	}, got)
}

func TestAPacketThatPicksNoDocumentsKeepsNoTurns(t *testing.T) {
	got, err := packetRotations(
		[]billingqueue.PacketPageRotation{turn(pulid.MustNew("doc_"), 1, 90)},
		nil,
	)

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestPacketRotationsRefuseWhatIsNotAQuarterTurnOfARealPage(t *testing.T) {
	doc := pulid.MustNew("doc_")
	for name, turns := range map[string][]billingqueue.PacketPageRotation{
		"not a quarter turn": {turn(doc, 1, 45)},
		"page zero":          {turn(doc, 0, 90)},
		"past the last page": {turn(doc, 2001, 90)},
		"too many":           make([]billingqueue.PacketPageRotation, maxPacketRotations+1),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := packetRotations(turns, []pulid.ID{doc})

			var invalid *errortypes.Error
			require.True(t, errors.As(err, &invalid), "got %v", err)
			assert.Equal(t, "packetPageRotations", invalid.Field)
		})
	}
}
