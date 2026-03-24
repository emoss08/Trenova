package shipmentstate

import "github.com/emoss08/trenova/internal/core/domain/shipment"

var shipmentStatusTransitions = map[shipment.Status]map[shipment.Status]struct{}{
	shipment.StatusNew: {
		shipment.StatusPartiallyAssigned:  {},
		shipment.StatusAssigned:           {},
		shipment.StatusInTransit:          {},
		shipment.StatusPartiallyCompleted: {},
		shipment.StatusCompleted:          {},
		shipment.StatusCanceled:           {},
	},
	shipment.StatusPartiallyAssigned: {
		shipment.StatusAssigned:  {},
		shipment.StatusInTransit: {},
		shipment.StatusCanceled:  {},
	},
	shipment.StatusAssigned: {
		shipment.StatusInTransit: {},
		shipment.StatusCompleted: {},
		shipment.StatusCanceled:  {},
	},
	shipment.StatusInTransit: {
		shipment.StatusDelayed:            {},
		shipment.StatusPartiallyCompleted: {},
		shipment.StatusCompleted:          {},
		shipment.StatusCanceled:           {},
	},
	shipment.StatusDelayed: {
		shipment.StatusInTransit:          {},
		shipment.StatusPartiallyCompleted: {},
		shipment.StatusCompleted:          {},
		shipment.StatusCanceled:           {},
	},
	shipment.StatusPartiallyCompleted: {
		shipment.StatusCompleted: {},
		shipment.StatusCanceled:  {},
	},
	shipment.StatusReadyToInvoice: {
		shipment.StatusInvoiced: {},
		shipment.StatusCanceled: {},
	},
	shipment.StatusCompleted: {
		shipment.StatusReadyToInvoice: {},
	},
	shipment.StatusInvoiced: {},
	shipment.StatusCanceled: {},
}

var moveStatusTransitions = map[shipment.MoveStatus]map[shipment.MoveStatus]struct{}{
	shipment.MoveStatusNew: {
		shipment.MoveStatusAssigned:  {},
		shipment.MoveStatusInTransit: {},
		shipment.MoveStatusCompleted: {},
		shipment.MoveStatusCanceled:  {},
	},
	shipment.MoveStatusAssigned: {
		shipment.MoveStatusInTransit: {},
		shipment.MoveStatusCompleted: {},
		shipment.MoveStatusCanceled:  {},
	},
	shipment.MoveStatusInTransit: {
		shipment.MoveStatusCompleted: {},
		shipment.MoveStatusCanceled:  {},
	},
	shipment.MoveStatusCompleted: {},
	shipment.MoveStatusCanceled:  {},
}

var stopStatusTransitions = map[shipment.StopStatus]map[shipment.StopStatus]struct{}{
	shipment.StopStatusNew: {
		shipment.StopStatusInTransit: {},
		shipment.StopStatusCompleted: {},
		shipment.StopStatusCanceled:  {},
	},
	shipment.StopStatusInTransit: {
		shipment.StopStatusCompleted: {},
		shipment.StopStatusCanceled:  {},
	},
	shipment.StopStatusCompleted: {},
	shipment.StopStatusCanceled:  {},
}

func isAllowedShipmentStatusTransition(from, to shipment.Status) bool {
	if from == to || from == "" || to == "" {
		return true
	}

	next, ok := shipmentStatusTransitions[from]
	if !ok {
		return false
	}

	_, allowed := next[to]
	return allowed
}

func isAllowedMoveStatusTransition(from, to shipment.MoveStatus) bool {
	if from == to || from == "" || to == "" {
		return true
	}

	next, ok := moveStatusTransitions[from]
	if !ok {
		return false
	}

	_, allowed := next[to]
	return allowed
}

func CanTransitionMoveStatus(from, to shipment.MoveStatus) bool {
	return isAllowedMoveStatusTransition(from, to)
}

func isAllowedStopStatusTransition(from, to shipment.StopStatus) bool {
	if from == to || from == "" || to == "" {
		return true
	}

	next, ok := stopStatusTransitions[from]
	if !ok {
		return false
	}

	_, allowed := next[to]
	return allowed
}
