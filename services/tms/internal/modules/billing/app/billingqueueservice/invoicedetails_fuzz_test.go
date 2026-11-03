//go:build fuzz

package billingqueueservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/shared/pulid"
)

func FuzzPacketRotationsKeepOneQuarterTurnPerCarriedPage(f *testing.F) {
	f.Add([]byte{0, 1, 1, 1, 2, 2, 0, 3, 0xff}, uint8(2))
	f.Add([]byte{2, 0, 0}, uint8(0))
	f.Add([]byte{1, 5, 0x5a}, uint8(3))
	documents := []pulid.ID{
		pulid.ID("doc_01FUZZPACKETAAAAAAAAAAAAAA"),
		pulid.ID("doc_01FUZZPACKETBBBBBBBBBBBBBB"),
		pulid.ID("doc_01FUZZPACKETCCCCCCCCCCCCCC"),
	}
	f.Fuzz(func(t *testing.T, data []byte, carried uint8) {
		packet := documents[:int(carried)%(len(documents)+1)]
		turns := make([]billingqueue.PacketPageRotation, 0, len(data)/3)
		for i := 0; i+2 < len(data); i += 3 {
			turns = append(turns, billingqueue.PacketPageRotation{
				DocumentID: documents[int(data[i])%len(documents)],
				Page:       int(data[i+1]) * 10,
				Rotation:   int(int8(data[i+2])) * 45,
			})
		}

		got, err := packetRotations(turns, packet)
		if err != nil {
			return
		}
		order := make(map[pulid.ID]int, len(packet))
		for idx, id := range packet {
			order[id] = idx
		}
		seen := make(map[billingqueue.PacketPageRotation]bool, len(got))
		for i, turn := range got {
			if turn.Rotation != 90 && turn.Rotation != 180 && turn.Rotation != 270 {
				t.Fatalf("kept a turn of %d", turn.Rotation)
			}
			if _, ok := order[turn.DocumentID]; !ok {
				t.Fatalf("kept a turn of %s, which the packet does not carry", turn.DocumentID)
			}
			key := billingqueue.PacketPageRotation{DocumentID: turn.DocumentID, Page: turn.Page}
			if seen[key] {
				t.Fatalf("kept two turns of page %d of %s", turn.Page, turn.DocumentID)
			}
			seen[key] = true
			if i > 0 {
				before := got[i-1]
				if order[before.DocumentID] > order[turn.DocumentID] ||
					(before.DocumentID == turn.DocumentID && before.Page >= turn.Page) {
					t.Fatalf("turns are out of packet order at %d", i)
				}
			}
		}
	})
}
