package models

import (
	"context"
	"errors"

	gen "github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/equipmenttype"
	"github.com/emoss08/trenova/internal/ent/shipment"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/google/uuid"
)

func ValidateShipment(
	ctx context.Context, m *gen.ShipmentMutation, shipmentControl *gen.ShipmentControl, billingControl *gen.BillingControl, dispatchControl *gen.DispatchControl,
) ([]types.ValidationErrorDetail, error) {
	var validationErrors []types.ValidationErrorDetail

	validateRatingMethod(m, &validationErrors)

	validateShipmentLocation(m, &validationErrors)

	if err := validateReadyToBill(m, &validationErrors, billingControl); err != nil {
		return nil, err
	}

	// Validate shipment control.
	if err := validateShipmentControl(m, &validationErrors, shipmentControl); err != nil {
		return nil, err
	}

	// Validate duplicate BOL.
	if err := validateDuplicateShipmentBOL(ctx, m, &validationErrors); err != nil {
		return nil, err
	}

	// Validate Appointtime windows.
	if err := validateAppointmentWindows(m, &validationErrors); err != nil {
		return nil, err
	}

	// Validate shipment weight limit
	if err := validateShipmentWeightLimit(m, &validationErrors, dispatchControl); err != nil {
		return nil, err
	}

	// Validate trailer and tractor type.
	if err := validateTrailerAndTractorType(ctx, m, &validationErrors); err != nil {
		return nil, err
	}

	return validationErrors, nil
}

func validateRatingMethod(m *gen.ShipmentMutation, validationErrors *[]types.ValidationErrorDetail) {
	ratingMethod, _ := m.RatingMethod()
	_, chargeExists := m.FreightChargeAmount()
	_, mileageExists := m.Mileage()
	_, weightExists := m.Weight()

	if ratingMethod == "FlatRate" && !chargeExists {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "missingFreightChargeAmount",
			Detail: "Freight charge amount is required when the rating method is Flat. Please try again.",
			Attr:   "freightChargeAmount",
		})
	}

	if ratingMethod == "PerMile" && !mileageExists {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "missingMileage",
			Detail: "Mileage is required when the rating method is PerMile. Please try again.",
			Attr:   "mileage",
		})
	}

	if ratingMethod == "PerPound" && !weightExists {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "missingWeight",
			Detail: "Weight is required when the rating method is PerPound. Please try again.",
			Attr:   "weight",
		})
	}
}

func validateShipmentControl(
	m *gen.ShipmentMutation, validationErrors *[]types.ValidationErrorDetail, shipmentControl *gen.ShipmentControl,
) error {
	originLocationID, originLocationExists := m.OriginLocationID()
	destinationLocationID, destinationLocationExists := m.DestinationLocationID()

	if !originLocationExists || !destinationLocationExists {
		return errors.New("origin and destination locations are required for the shipment")
	}

	// Validate compare origin and destination are not the same.
	if shipmentControl.EnforceOriginDestination && originLocationID == destinationLocationID {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "invalidOriginDestination",
			Detail: "The origin and destination locations cannot be the same. Please try again.",
			Attr:   "originLocationId",
		})
	}

	// Validate revenue code is entered if shipment control requires it for the organization.
	revenueCodeID, revenueCodeExists := m.RevenueCodeID()
	if !revenueCodeExists {
		return errors.New("revenue code is required for the shipment")
	}

	if shipmentControl.EnforceRevCode && revenueCodeID == uuid.Nil {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "missingRevenueCode",
			Detail: "The revenue code is required. Please try again.",
			Attr:   "revenueCodeId",
		})
	}

	// Validate voiced comment is entered if the shipment control requires it for the organization.
	voidedComment, voidedCommentExists := m.VoidedComment()
	shipmentStatus, statusExists := m.Status()

	if !voidedCommentExists || !statusExists {
		return errors.New("voided comment and status are required for the shipment")
	}

	if shipmentControl.EnforceVoidedComm && shipmentStatus == "Voided" && voidedComment == "" {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "missingVoidedComment",
			Detail: "The voided comment is required for voided shipments. Please try again.",
			Attr:   "voidedComment",
		})
	}

	return nil
}

func validateReadyToBill(
	m *gen.ShipmentMutation, validationErrors *[]types.ValidationErrorDetail, billingControl *gen.BillingControl,
) error {
	readyToBill, readyToBillExists := m.ReadyToBill()
	shipmentStatus, shipmentStatusExists := m.Status()

	if !readyToBillExists || !shipmentStatusExists {
		return errors.New("ready to bill and status are required for the shipment")
	}

	if billingControl.ShipmentTransferCriteria == "ReadyAndCompleted" && readyToBill && shipmentStatus != "Completed" {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "invalidReadyToBill",
			Detail: "The shipment must be completed to be ready to bill. Please try again.",
			Attr:   "readyToBill",
		})
	}

	return nil
}

// validateShipmentLocation checks if either the origin or destination locations or address lines are provided for the shipment.
// It appends validation errors to the provided slice if any of the required fields are missing.
func validateShipmentLocation(
	m *gen.ShipmentMutation, validationErrors *[]types.ValidationErrorDetail,
) {
	_, originLocationExists := m.OriginLocationID()
	_, destinationLocationExists := m.DestinationLocationID()

	_, originAddressLineExists := m.OriginAddressLine()
	_, destinationAddressLineExists := m.DestinationAddressLine()

	if !originAddressLineExists && !originLocationExists {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "missingOriginLocation",
			Detail: "The origin location or address line is required. Please try again.",
			Attr:   "originLocationId",
		})
	}

	if !destinationAddressLineExists && !destinationLocationExists {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "missingDestinationLocation",
			Detail: "The destination location or address line is required. Please try again.",
			Attr:   "destinationLocationId",
		})
	}
}

// validateDupliceShipmentBOL checks if the bill of lading number is already in use by another shipment.
// It appends validation errors to the provided slice if the bill of lading number is already in use.
func validateDuplicateShipmentBOL(
	ctx context.Context, m *gen.ShipmentMutation, validationErrors *[]types.ValidationErrorDetail,
) error {
	billOfLadingNumber, bolExists := m.BillOfLadingNumber()
	orgID, orgIDExists := m.OrganizationID()
	buID, buIDExists := m.BusinessUnitID()

	if !bolExists || !orgIDExists || !buIDExists {
		return errors.New("bill of lading number, organization ID, and business unit ID are required for the shipment")
	}

	// Current shipment ID.
	shipmentID, idExists := m.ID()
	if !idExists {
		return errors.New("shipment ID is required for the shipment")
	}

	duplicateBOLs, err := m.Client().Shipment.Query().Where(
		shipment.BillOfLadingNumberEQ(billOfLadingNumber),
		shipment.OrganizationIDEQ(orgID),
		shipment.BusinessUnitIDEQ(buID),
		shipment.StatusIn("New", "InProgress"),
		shipment.IDNEQ(shipmentID),
	).All(ctx)
	if err != nil {
		return err
	}

	// Combine all of the pro_numbers into a single string.
	var shipmentProNumbers string
	for _, shipment := range duplicateBOLs {
		shipmentProNumbers += shipment.ProNumber
	}

	if len(duplicateBOLs) > 0 {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "duplicateBOL",
			Detail: "The bill of lading number is already in use by the following shipments: " + shipmentProNumbers,
			Attr:   "billOfLadingNumber",
		})
	}

	return nil
}

// validateAppointmentWindows checks if the appointment windows are valid for the shipment.
// It appends validation errors to the provided slice if the appointment windows are invalid.
func validateAppointmentWindows(
	m *gen.ShipmentMutation, validationErrors *[]types.ValidationErrorDetail,
) error {
	originStart, originStartExists := m.OriginAppointmentStart()
	originEnd, originEndExists := m.OriginAppointmentEnd()

	destinationStart, destinationStartExists := m.DestinationAppointmentStart()
	destinationEnd, destinationEndExists := m.DestinationAppointmentEnd()

	if !originStartExists || !originEndExists || !destinationStartExists || !destinationEndExists {
		return errors.New("appointment windows are required for the shipment")
	}

	if originStart.After(originEnd) {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "invalidOriginAppointmentWindow",
			Detail: "The origin appointment start date must be before the end date. Please try again.",
			Attr:   "originAppointmentStart",
		})
	}

	if destinationStart.After(destinationEnd) {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "invalidDestinationAppointmentWindow",
			Detail: "The destination appointment start date must be before the end date. Please try again.",
			Attr:   "destinationAppointmentStart",
		})
	}

	return nil
}

// validateShipmentWeightLimit checks if the weight of the shipment exceeds the maximum weight limit.
// It appends validation errors to the provided slice if the weight exceeds the limit.
func validateShipmentWeightLimit(
	m *gen.ShipmentMutation, validationErrors *[]types.ValidationErrorDetail, dispatchControl *gen.DispatchControl,
) error {
	weight, weightExists := m.Weight()

	if !weightExists {
		return errors.New("weight is required for the shipment")
	}

	if weight > float64(dispatchControl.MaxShipmentWeightLimit) {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "invalidWeight",
			Detail: "The weight of the shipment exceeds the maximum weight limit. Please try again.",
			Attr:   "weight",
		})
	}

	return nil
}

// validateTrailerAndTractorType checks if the trailer and tractor types are valid for the shipment.
// It appends validation errors to the provided slice if the types are invalid.
func validateTrailerAndTractorType(
	ctx context.Context, m *gen.ShipmentMutation, validationErrors *[]types.ValidationErrorDetail,
) error {
	trailerTypeID, trailerTypeExists := m.TrailerTypeID()
	tractorTypeID, tractorTypeExists := m.TractorTypeID()

	// Check if either trailerTypeID or tractorTypeID is provided before querying the database.
	if trailerTypeExists {
		trailerType, err := m.Client().EquipmentType.Query().Where(
			equipmenttype.IDEQ(trailerTypeID),
		).Only(ctx)
		if err != nil {
			return err // Handle not found error or other DB errors appropriately.
		}
		if trailerType.EquipmentClass != "Trailer" {
			*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
				Code:   "invalidTrailerType",
				Detail: "The trailer type must be a trailer. Please try again.",
				Attr:   "trailerTypeId",
			})
		}
	}

	if tractorTypeExists {
		tractorType, err := m.Client().EquipmentType.Query().Where(
			equipmenttype.IDEQ(tractorTypeID),
		).Only(ctx)
		if err != nil {
			return err // Handle not found error or other DB errors appropriately.
		}
		if tractorType.EquipmentClass != "Tractor" {
			*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
				Code:   "invalidTractorType",
				Detail: "The tractor type must be a tractor. Please try again.",
				Attr:   "tractorTypeId",
			})
		}
	}

	// If neither ID exists, append a validation error indicating the need for at least one type.
	if !trailerTypeExists && !tractorTypeExists {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Code:   "missingEquipmentType",
			Detail: "Either a trailer or tractor type must be provided. Please try again.",
			Attr:   "equipmentType",
		})
	}

	return nil
}
