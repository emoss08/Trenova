// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package assignmentvalidator

import "github.com/emoss08/trenova/internal/core/domain/shipment"

var assignableMoveStatuses = map[shipment.MoveStatus]bool{
	shipment.MoveStatusNew:       true,  // Can assign to new moves
	shipment.MoveStatusAssigned:  true,  // Can reassign to assigned moves
	shipment.MoveStatusInTransit: false, // Can't reassign to in transit moves
	shipment.MoveStatusCompleted: false, // Can't assign to completed moves
	shipment.MoveStatusCanceled:  false, // Can't assign to canceled moves
}
