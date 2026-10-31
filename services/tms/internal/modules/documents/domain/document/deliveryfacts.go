package document

import (
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type OSDKind string

const (
	OSDNone    = OSDKind("None")
	OSDOver    = OSDKind("Over")
	OSDShort   = OSDKind("Short")
	OSDDamaged = OSDKind("Damaged")
)

func (k OSDKind) IsValid() bool {
	switch k {
	case OSDNone, OSDOver, OSDShort, OSDDamaged:
		return true
	default:
		return false
	}
}

type FactsSource string

const (
	FactsSourcePerson = FactsSource("Person")
	FactsSourceAgent  = FactsSource("Agent")
)

const maxDeliveryFactText = 200

type DeliveryFacts struct {
	SignedBy    string              `json:"signedBy,omitempty"`
	Pieces      string              `json:"pieces,omitempty"`
	OSD         OSDKind             `json:"osd,omitempty"`
	OSDNote     string              `json:"osdNote,omitempty"`
	Seal        string              `json:"seal,omitempty"`
	ArrivedAt   *int64              `json:"arrivedAt,omitempty"`
	DepartedAt  *int64              `json:"departedAt,omitempty"`
	Amount      decimal.NullDecimal `json:"amount"`
	Reference   string              `json:"reference,omitempty"`
	ChargeID    pulid.ID            `json:"chargeId,omitempty"`
	ChargeLabel string              `json:"chargeLabel,omitempty"`
	Source      FactsSource         `json:"source,omitempty"`
	ReadFields  []string            `json:"readFields,omitempty"`
	UpdatedByID pulid.ID            `json:"updatedById,omitempty"`
	UpdatedAt   int64               `json:"updatedAt,omitempty"`
}

func (f *DeliveryFacts) Normalize() {
	f.SignedBy = strings.TrimSpace(f.SignedBy)
	f.Pieces = strings.TrimSpace(f.Pieces)
	f.OSDNote = strings.TrimSpace(f.OSDNote)
	f.Seal = strings.TrimSpace(f.Seal)
	f.Reference = strings.TrimSpace(f.Reference)
	f.ChargeLabel = strings.TrimSpace(f.ChargeLabel)
	if f.ChargeID.IsNil() {
		f.ChargeLabel = ""
	}
	if f.OSD == OSDNone {
		f.OSDNote = ""
	}
}

func (f *DeliveryFacts) BillsCharge() bool {
	return f != nil && f.ChargeID.IsNotNil() && f.Amount.Valid && f.Amount.Decimal.IsPositive()
}

func (f *DeliveryFacts) Validate(multiErr *errortypes.MultiError) {
	if f.OSD != "" && !f.OSD.IsValid() {
		multiErr.Add(
			"deliveryFacts.osd",
			errortypes.ErrInvalid,
			"Choose none, over, short or damaged",
		)
	}
	for _, field := range []struct{ name, value string }{
		{"deliveryFacts.signedBy", f.SignedBy},
		{"deliveryFacts.pieces", f.Pieces},
		{"deliveryFacts.osdNote", f.OSDNote},
		{"deliveryFacts.seal", f.Seal},
		{"deliveryFacts.reference", f.Reference},
	} {
		if len(field.value) > maxDeliveryFactText {
			multiErr.Add(
				field.name,
				errortypes.ErrInvalid,
				"Keep this under {0} characters",
				maxDeliveryFactText,
			)
		}
	}
	if f.ArrivedAt != nil && f.DepartedAt != nil && *f.DepartedAt < *f.ArrivedAt {
		multiErr.Add(
			"deliveryFacts.departedAt",
			errortypes.ErrInvalid,
			"Departure can't be before arrival",
		)
	}
	if f.ChargeID.IsNotNil() && !f.Amount.Valid {
		multiErr.Add("deliveryFacts.amount", errortypes.ErrRequired, "Enter the amount to bill")
	}
	if f.Amount.Valid && f.Amount.Decimal.IsNegative() {
		multiErr.Add("deliveryFacts.amount", errortypes.ErrInvalid, "The amount can't be negative")
	}
}
