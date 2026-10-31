package reviewprompt

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
)

// ExcerptRunes is how much of a document's text the review is shown at once;
// the rest is read a part at a time.
const ExcerptRunes = 2500

// Evidence is everything a review is shown.
type Evidence struct {
	Item      *billingqueue.BillingQueueItem
	Documents []*document.Document
	Excerpts  map[pulid.ID]Excerpt
	Catalog   []*accessorialcharge.AccessorialCharge
	Recent    []*billingqueueports.RecentInvoice
	Comments  []*shipment.ShipmentComment
}

// Excerpt is the first part of a document's text and how many parts it has.
type Excerpt struct {
	Text  string
	Parts int
}

// Parts splits text into the parts the review reads it in.
func Parts(text string) []string {
	runes := []rune(text)
	parts := make([]string, 0, len(runes)/ExcerptRunes+1)
	for start := 0; start < len(runes); start += ExcerptRunes {
		parts = append(parts, string(runes[start:min(start+ExcerptRunes, len(runes))]))
	}

	return parts
}

// Hash names what the review was shown, so an item whose evidence has not
// moved is not reviewed again, and one whose evidence did is.
func Hash(e *Evidence) string {
	h := sha256.New()
	h.Write([]byte(e.Item.ID.String()))
	for _, section := range Sections(e) {
		h.Write([]byte{0x1e})
		h.Write([]byte(section.Title))
		h.Write([]byte{0x1f})
		h.Write([]byte(section.Content))
	}

	return hex.EncodeToString(h.Sum(nil))
}
