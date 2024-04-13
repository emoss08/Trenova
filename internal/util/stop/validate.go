package stop

import (
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/internal/util/types"
)

// ValidateStop performs all necessary validations for a stop.
func ValidateStop(m *ent.StopMutation, shipmentMove *ent.ShipmentMove) ([]types.ValidationErrorDetail, error) {
	var errs []types.ValidationErrorDetail

	validateWorkerTractorAndTimes(m, shipmentMove, &errs)
	validateMovementStatus(m, shipmentMove, &errs)
	validateLocation(m, &errs)
	validateAppointmentTimes(m, &errs)

	return errs, nil
}

// validateWorkerTractorAndTimes checks if a shipment move lacking assigned tractor or worker has an arrival time set.
func validateWorkerTractorAndTimes(
	m *ent.StopMutation, shipmentMove *ent.ShipmentMove, validationErrors *[]types.ValidationErrorDetail,
) {
	tractorID := shipmentMove.TractorID
	workerID := shipmentMove.PrimaryWorkerID
	_, ok := m.ArrivalTime()

	if tractorID == nil && workerID == nil && ok {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "invalidShipmentMoveTimes",
			Detail: "Shipment move does not have a tractor or worker assigned, but arrival time is set in the stop.",
			Attr:   "arrivalTime",
		})
	}
}

// validateMovementStatus ensures that movement status transitions are valid.
func validateMovementStatus(
	m *ent.StopMutation, shipmentMove *ent.ShipmentMove, validationErrors *[]types.ValidationErrorDetail,
) {
	stopStatus, _ := m.Status()

	tractorID := shipmentMove.TractorID
	workerID := shipmentMove.PrimaryWorkerID

	if tractorID == nil && workerID == nil {
		if stopStatus == "InProgress" || stopStatus == "Completed" {
			*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
				Code:   "invalidShipmentMoveStatus",
				Detail: "Shipment move does not have a tractor or worker assigned, but status is set to 'InProgress' or 'Completed'.",
				Attr:   "status",
			})
		}
	}
}

// validateLocation ensures that a location address or code is set for a stop.
func validateLocation(
	m *ent.StopMutation, validationErrors *[]types.ValidationErrorDetail,
) {
	_, exists := m.AddressLine()
	_, codeExists := m.LocationID()

	if !exists && !codeExists {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "invalidStopLocation",
			Detail: "Stop must have either an address or a location code.",
			Attr:   "address",
		})
	}
}

// validateAppointmentTimes ensures that appointment times are set correctly.
func validateAppointmentTimes(
	m *ent.StopMutation, validationErrors *[]types.ValidationErrorDetail,
) {
	arrivalTime, arrivalExists := m.ArrivalTime()
	departureTime, departureExists := m.DepartureTime()

	if departureExists && !arrivalExists {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "invalidStopTimes",
			Detail: "Stop must have an arrival time before a departure time.",
			Attr:   "arrivalTime",
		})
	}

	if arrivalExists && departureExists && departureTime.Before(arrivalTime) {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "invalidStopTimes",
			Detail: "Stop arrival time must be before departure time.",
			Attr:   "arrivalTime",
		})
	}

	// TODO: Add validation that validates the appointment time must be before the next stop. This will require a query to get the next stop.
}
