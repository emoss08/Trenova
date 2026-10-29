package shipmentstate

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
)

var shipmentStatusTransitions = map[shipmentvocab.Status]map[shipmentvocab.Status]struct{}{
	shipmentvocab.StatusNew: {
		shipmentvocab.StatusPartiallyAssigned:  {},
		shipmentvocab.StatusAssigned:           {},
		shipmentvocab.StatusInTransit:          {},
		shipmentvocab.StatusPartiallyCompleted: {},
		shipmentvocab.StatusCompleted:          {},
		shipmentvocab.StatusCanceled:           {},
	},
	shipmentvocab.StatusPartiallyAssigned: {
		shipmentvocab.StatusAssigned:  {},
		shipmentvocab.StatusInTransit: {},
		shipmentvocab.StatusCanceled:  {},
	},
	shipmentvocab.StatusAssigned: {
		shipmentvocab.StatusInTransit: {},
		shipmentvocab.StatusCompleted: {},
		shipmentvocab.StatusCanceled:  {},
	},
	shipmentvocab.StatusInTransit: {
		shipmentvocab.StatusDelayed:            {},
		shipmentvocab.StatusPartiallyCompleted: {},
		shipmentvocab.StatusCompleted:          {},
		shipmentvocab.StatusCanceled:           {},
	},
	shipmentvocab.StatusDelayed: {
		shipmentvocab.StatusInTransit:          {},
		shipmentvocab.StatusPartiallyCompleted: {},
		shipmentvocab.StatusCompleted:          {},
		shipmentvocab.StatusCanceled:           {},
	},
	shipmentvocab.StatusPartiallyCompleted: {
		shipmentvocab.StatusCompleted: {},
		shipmentvocab.StatusCanceled:  {},
	},
	shipmentvocab.StatusReadyToInvoice: {
		shipmentvocab.StatusInvoiced: {},
		shipmentvocab.StatusCanceled: {},
	},
	shipmentvocab.StatusCompleted: {
		shipmentvocab.StatusReadyToInvoice: {},
	},
	shipmentvocab.StatusInvoiced: {},
	shipmentvocab.StatusCanceled: {},
}

var moveStatusTransitions = map[shipmentvocab.MoveStatus]map[shipmentvocab.MoveStatus]struct{}{
	shipmentvocab.MoveStatusNew: {
		shipmentvocab.MoveStatusAssigned:  {},
		shipmentvocab.MoveStatusInTransit: {},
		shipmentvocab.MoveStatusCompleted: {},
		shipmentvocab.MoveStatusCanceled:  {},
	},
	shipmentvocab.MoveStatusAssigned: {
		shipmentvocab.MoveStatusInTransit: {},
		shipmentvocab.MoveStatusCompleted: {},
		shipmentvocab.MoveStatusCanceled:  {},
	},
	shipmentvocab.MoveStatusInTransit: {
		shipmentvocab.MoveStatusCompleted: {},
		shipmentvocab.MoveStatusCanceled:  {},
	},
	shipmentvocab.MoveStatusCompleted: {},
	shipmentvocab.MoveStatusCanceled:  {},
}

var stopStatusTransitions = map[shipmentvocab.StopStatus]map[shipmentvocab.StopStatus]struct{}{
	shipmentvocab.StopStatusNew: {
		shipmentvocab.StopStatusInTransit: {},
		shipmentvocab.StopStatusCompleted: {},
		shipmentvocab.StopStatusCanceled:  {},
	},
	shipmentvocab.StopStatusInTransit: {
		shipmentvocab.StopStatusCompleted: {},
		shipmentvocab.StopStatusCanceled:  {},
	},
	shipmentvocab.StopStatusCompleted: {},
	shipmentvocab.StopStatusCanceled:  {},
}

func isAllowedShipmentStatusTransition(from, to shipmentvocab.Status) bool {
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

func CanTransitionShipmentStatus(from, to shipmentvocab.Status) bool {
	return isAllowedShipmentStatusTransition(from, to)
}

func isAllowedMoveStatusTransition(from, to shipmentvocab.MoveStatus) bool {
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

func CanTransitionMoveStatus(from, to shipmentvocab.MoveStatus) bool {
	return isAllowedMoveStatusTransition(from, to)
}

func isAllowedStopStatusTransition(from, to shipmentvocab.StopStatus) bool {
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
