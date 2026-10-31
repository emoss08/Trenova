package billingqueue

import "github.com/emoss08/trenova/shared/pulid"

type RatedByKind string

const (
	RatedByAgreement = RatedByKind("Agreement")
	RatedByTemplate  = RatedByKind("Template")
)

// RatedBy names what priced a shipment, for the biller to open: the contract
// when one did, measured against even after a rate departed from it, or else
// the formula template. Code is the agreement's own.
type RatedBy struct {
	Kind RatedByKind `json:"kind"`
	ID   pulid.ID    `json:"id"`
	Code string      `json:"code,omitempty"`
	Name string      `json:"name"`
}
