package aicorrection

import (
	"time"
)

const (
	FieldReference      = "referenceNumber"
	FieldRate           = "rate"
	FieldWeight         = "weight"
	FieldPieces         = "pieceCount"
	FieldCommodity      = "commodity"
	FieldShipper        = "shipper"
	FieldConsignee      = "consignee"
	FieldPickupWindow   = "pickupWindow"
	FieldDeliveryWindow = "deliveryWindow"
)

func (s *Snapshot) FirstStop(role string) *StopSnapshot {
	for i := range s.Stops {
		if s.Stops[i].Role == role {
			return &s.Stops[i]
		}
	}

	return nil
}

func (s *Snapshot) LastStop(role string) *StopSnapshot {
	for i := len(s.Stops) - 1; i >= 0; i-- {
		if s.Stops[i].Role == role {
			return &s.Stops[i]
		}
	}

	return nil
}

func (s *StopSnapshot) Location() *time.Location {
	if s.Timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		return time.UTC
	}

	return loc
}
