//go:build fuzz

package invoicepacket

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
)

func rotationsFrom(data []byte) []billingvocab.PageRotation {
	out := make([]billingvocab.PageRotation, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		page := int(int8(data[i]))
		out = append(out, billingvocab.PageRotation{Page: page, Rotation: int(data[i+1]%4) * 90})
	}
	return out
}

func FuzzTurnedSpansCoverEveryPageOnceInOrder(f *testing.F) {
	f.Add([]byte{2, 1, 3, 1, 5, 2}, uint8(5))
	f.Add([]byte{1, 3}, uint8(1))
	f.Add([]byte{9, 1, 0, 2, 0xff, 1}, uint8(3))
	f.Add([]byte{}, uint8(0))
	f.Fuzz(func(t *testing.T, data []byte, count uint8) {
		pages := int(count)
		rotations := rotationsFrom(data)
		spans := turnedSpans(rotations, pages)

		want := make(map[int]int, len(rotations))
		for _, turn := range rotations {
			if turn.Page >= 1 && turn.Page <= pages && turn.Rotation != 0 {
				want[turn.Page] = turn.Rotation
			}
		}
		if len(want) == 0 {
			if spans != nil {
				t.Fatalf("no page turned, yet %d spans came back", len(spans))
			}
			return
		}
		next := 0
		for i, span := range spans {
			if span.First != next || span.Last < span.First {
				t.Fatalf("span %d is %d to %d after page %d", i, span.First, span.Last, next)
			}
			if i > 0 && spans[i-1].Rotation == span.Rotation {
				t.Fatalf("spans %d and %d turn alike and were not joined", i-1, i)
			}
			for page := span.First + 1; page <= span.Last+1; page++ {
				if want[page] != span.Rotation {
					t.Fatalf("page %d is turned %d, asked %d", page, span.Rotation, want[page])
				}
			}
			next = span.Last + 1
		}
		if next != pages {
			t.Fatalf("the spans end at page %d of %d", next, pages)
		}
	})
}

func FuzzPartsOfNumberThePacketsPagesWithoutGapsOrOverlaps(f *testing.F) {
	f.Add([]byte{1, 2, 1}, uint8(4))
	f.Add([]byte{3}, uint8(3))
	f.Add([]byte{1, 0, 2}, uint8(3))
	f.Add([]byte{}, uint8(0))
	f.Fuzz(func(t *testing.T, counts []byte, pages uint8) {
		sources := make([]packetSource, len(counts))
		perPart := make([]int, len(counts))
		total := 0
		for i, count := range counts {
			perPart[i] = int(count % 16)
			total += perPart[i]
			sources[i].meta = billingqueueports.PacketPart{
				Kind: billingqueueports.PacketPartDocument,
			}
		}
		parts := partsOf(sources, perPart, int(pages))

		if parts == nil {
			return
		}
		if total != int(pages) {
			t.Fatalf("parts of %d pages came back for a packet of %d", total, pages)
		}
		next := 1
		for i, part := range parts {
			if perPart[i] < 1 || part.FirstPage != next ||
				part.LastPage-part.FirstPage+1 != perPart[i] {
				t.Fatalf("part %d runs %d to %d for %d pages after page %d",
					i, part.FirstPage, part.LastPage, perPart[i], next-1)
			}
			next = part.LastPage + 1
		}
		if next-1 != int(pages) {
			t.Fatalf("the parts end at page %d of %d", next-1, pages)
		}
	})
}
