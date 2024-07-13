package types

import (
	"time"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type StopInput struct {
	LocationID       uuid.UUID           `json:"locationId"`
	Type             property.StopType   `json:"type"`
	PlannedArrival   time.Time           `json:"plannedArrival"`
	PlannedDeparture time.Time           `json:"plannedDeparture"`
	Weight           decimal.NullDecimal `json:"weight"`
	Pieces           decimal.NullDecimal `json:"pieces"`
}

type CreateShipmentInput struct {
	BusinessUnitID              uuid.UUID                     `json:"businessUnitId"`
	OrganizationID              uuid.UUID                     `json:"organizationId"`
	CustomerID                  uuid.UUID                     `json:"customerId"`
	OriginLocationID            uuid.UUID                     `json:"originLocationId"`
	OriginPlannedArrival        time.Time                     `json:"originPlannedArrival"`
	OriginPlannedDeparture      time.Time                     `json:"originPlannedDeparture"`
	DestinationLocationID       uuid.UUID                     `json:"destinationLocationId"`
	DestinationPlannedArrival   time.Time                     `json:"destinationPlannedArrival"`
	DestinationPlannedDeparture time.Time                     `json:"destinationPlannedDeparture"`
	ShipmentTypeID              uuid.UUID                     `json:"shipmentTypeId"`
	RevenueCodeID               *uuid.UUID                    `json:"revenueCodeId"`
	ServiceTypeID               *uuid.UUID                    `json:"serviceTypeId"`
	RatingMethod                property.ShipmentRatingMethod `json:"ratingMethod"`
	RatingUnit                  int                           `json:"ratingUnit"`
	OtherChargeAmount           decimal.Decimal               `json:"otherChargeAmount"`
	FreightChargeamount         decimal.Decimal               `json:"freightChargeAmount"`
	TotalChargeAmount           decimal.Decimal               `json:"totalChargeAmount"`
	Pieces                      decimal.NullDecimal           `json:"pieces"`
	Weight                      decimal.NullDecimal           `json:"weight"`
	TractorID                   uuid.UUID                     `json:"tractorId"`
	TrailerID                   uuid.UUID                     `json:"trailerId"`
	PrimaryWorkerID             uuid.UUID                     `json:"primaryWorkerId"`
	SecondaryWorkerID           *uuid.UUID                    `json:"secondaryWorkerId"`
	TrailerTypeID               *uuid.UUID                    `json:"trailerTypeId"`
	TractorTypeID               *uuid.UUID                    `json:"tractorTypeId"`
	TemperatureMin              int                           `json:"temperatureMin"`
	TemperatureMax              int                           `json:"temperatureMax"`
	BillOfLading                string                        `json:"billOfLading"`
	SpecialInstructions         string                        `json:"specialInstructions"`
	TrackingNumber              string                        `json:"trackingNumber"`
	Priority                    int                           `json:"priority"`
	TotalDistance               decimal.NullDecimal           `json:"totalDistance"`
	Stops                       []StopInput                   `json:"stops"`
}

type AssignTractorInput struct {
	TractorID   uuid.UUID                  `json:"tractorId"`
	Assignments []models.TractorAssignment `json:"assignments"`
}
