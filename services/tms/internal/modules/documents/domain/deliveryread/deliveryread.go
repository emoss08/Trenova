// Package deliveryread turns what a model read from a delivery document into
// the document's delivery facts. The billing queue's review and Nova's
// paperwork reader both ask for this answer and record it the same way.
package deliveryread

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// Answer is one document's facts as a model read them.
type Answer struct {
	DocumentID string  `json:"documentId,omitempty"`
	SignedBy   string  `json:"signedBy"`
	Pieces     string  `json:"pieces"`
	OSD        string  `json:"osd"`
	OSDNote    string  `json:"osdNote"`
	Seal       string  `json:"seal"`
	Amount     *string `json:"amount"`
	Reference  string  `json:"reference"`
	ChargeID   string  `json:"chargeId"`
}

// Charge is an accessorial a receipt may bill as, by the id the model answers with.
type Charge struct {
	ID    pulid.ID
	Label string
}

// Properties is the answer's JSON schema properties, for a caller to place in
// its own schema.
func Properties() map[string]any {
	str := map[string]any{"type": "string"}
	return map[string]any{
		"signedBy": str,
		"pieces":   str,
		"osd": map[string]any{
			"type": "string",
			"enum": []string{"", "None", "Over", "Short", "Damaged"},
		},
		"osdNote":   str,
		"seal":      str,
		"amount":    map[string]any{"type": []string{"string", "null"}},
		"reference": str,
		"chargeId":  str,
	}
}

// Required names every property, as strict structured output demands.
func Required() []string {
	return []string{"signedBy", "pieces", "osd", "osdNote", "seal", "amount", "reference", "chargeId"}
}

// Facts is the answer as the document's delivery facts, read by Nova. An
// answer that read nothing returns nil.
func Facts(answer *Answer, charges map[string]Charge, now int64) *document.DeliveryFacts {
	facts := &document.DeliveryFacts{
		SignedBy:  answer.SignedBy,
		Pieces:    answer.Pieces,
		OSD:       document.OSDKind(answer.OSD),
		OSDNote:   answer.OSDNote,
		Seal:      answer.Seal,
		Reference: answer.Reference,
		Source:    document.FactsSourceAgent,
		UpdatedAt: now,
	}
	if !facts.OSD.IsValid() {
		facts.OSD = ""
	}
	if answer.Amount != nil {
		amount, err := decimal.NewFromString(strings.TrimSpace(*answer.Amount))
		if err == nil && amount.IsPositive() {
			facts.Amount = decimal.NewNullDecimal(amount.Round(2))
		}
	}
	if charge, ok := charges[answer.ChargeID]; ok && facts.Amount.Valid {
		facts.ChargeID = charge.ID
		facts.ChargeLabel = charge.Label
	}
	facts.Normalize()
	facts.ReadFields = readFields(facts)
	if len(facts.ReadFields) == 0 {
		return nil
	}

	return facts
}

// Same reports whether next says what current already does.
func Same(current, next *document.DeliveryFacts) bool {
	if current == nil || next == nil {
		return current == next
	}

	return Key(current) == Key(next)
}

// Kept reports whether a person recorded the facts, which Nova never overwrites.
func Kept(current *document.DeliveryFacts) bool {
	return current != nil && current.Source == document.FactsSourcePerson
}

func readFields(f *document.DeliveryFacts) []string {
	fields := make([]string, 0, 7)
	for _, field := range []struct {
		name string
		set  bool
	}{
		{"signedBy", f.SignedBy != ""},
		{"pieces", f.Pieces != ""},
		{"osd", f.OSD != ""},
		{"seal", f.Seal != ""},
		{"amount", f.Amount.Valid},
		{"reference", f.Reference != ""},
		{"chargeId", f.ChargeID.IsNotNil()},
	} {
		if field.set {
			fields = append(fields, field.name)
		}
	}

	return fields
}

// Key is the facts as one comparable string.
func Key(f *document.DeliveryFacts) string {
	amount := ""
	if f.Amount.Valid {
		amount = f.Amount.Decimal.StringFixed(2)
	}

	return strings.Join([]string{
		f.SignedBy, f.Pieces, string(f.OSD), f.OSDNote, f.Seal, amount, f.Reference, f.ChargeID.String(),
	}, "\x1f")
}
