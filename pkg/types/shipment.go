// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
