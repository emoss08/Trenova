package order

import "github.com/emoss08/trenova/internal/core/domain/shipment"

type Status string

const (
	StatusDraft      = Status("Draft")
	StatusConfirmed  = Status("Confirmed")
	StatusInProgress = Status("InProgress")
	StatusCompleted  = Status("Completed")
	StatusBilled     = Status("Billed")
	StatusClosed     = Status("Closed")
	StatusCanceled   = Status("Canceled")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusDraft,
		StatusConfirmed,
		StatusInProgress,
		StatusCompleted,
		StatusBilled,
		StatusClosed,
		StatusCanceled:
		return true
	}
	return false
}

// AllowsMembershipChange reports whether the order's legs and charges may still be
// modified. Billed orders must be adjusted through the invoice-adjustment flow,
// Closed is terminal, and Canceled orders are read-only.
func (s Status) AllowsMembershipChange() bool {
	switch s {
	case StatusBilled, StatusClosed, StatusCanceled:
		return false
	case StatusDraft, StatusConfirmed, StatusInProgress, StatusCompleted:
		return true
	default:
		return false
	}
}

// Derive computes an order's status from the statuses of its legs (shipments).
//
// It is a pure function so it can be unit-tested and re-run idempotently by the
// derivation observer. Canceled legs are excluded from the progress calculation;
// an order is only Canceled when every leg is Canceled. StatusClosed is terminal
// and never produced here — it is set exclusively by the manual close / AR
// settlement flow, so the caller must guard against overwriting it.
func Derive(legs []shipment.Status) Status {
	if len(legs) == 0 {
		return StatusDraft
	}

	active := make([]shipment.Status, 0, len(legs))
	for _, s := range legs {
		if s != shipment.StatusCanceled {
			active = append(active, s)
		}
	}
	if len(active) == 0 {
		return StatusCanceled
	}

	allInvoiced := true
	allDelivered := true
	allNew := true
	for _, s := range active {
		if s != shipment.StatusInvoiced {
			allInvoiced = false
		}
		if !isDelivered(s) {
			allDelivered = false
		}
		if s != shipment.StatusNew {
			allNew = false
		}
	}

	switch {
	case allInvoiced:
		return StatusBilled
	case allDelivered:
		return StatusCompleted
	case allNew:
		return StatusConfirmed
	default:
		return StatusInProgress
	}
}

func isDelivered(s shipment.Status) bool {
	//nolint:exhaustive // only delivered/billed leg statuses matter here
	switch s {
	case shipment.StatusReadyToInvoice, shipment.StatusCompleted, shipment.StatusInvoiced:
		return true
	default:
		return false
	}
}
