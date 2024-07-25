// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package fixtures

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

func loadShipments(ctx context.Context, db *bun.DB, gen *gen.CodeGenerator, orgID, buID uuid.UUID) error {
	count, err := db.NewSelect().Model((*models.Shipment)(nil)).Count(ctx)
	if err != nil {
		return err
	}

	if count < 1 {
		state := new(models.UsState)
		err = db.NewSelect().Model(state).Where("abbreviation = ?", "AL").Scan(ctx)
		if err != nil {
			return err
		}

		customer, err := seedCustomer(ctx, db, gen, state.ID, orgID, buID)
		if err != nil {
			return err
		}

		location, err := seedLocation(ctx, db, state.ID, orgID, buID)
		if err != nil {
			return err
		}

		revCode, err := seedRevenueCode(ctx, db, orgID, buID)
		if err != nil {
			return err
		}

		serviceType, err := seedServiceType(ctx, db, orgID, buID)
		if err != nil {
			return err
		}

		primaryWorker, err := seedPrimaryWorker(ctx, db, gen, state.ID, orgID, buID)
		if err != nil {
			return err
		}

		equipType, err := seedEquipmentType(ctx, db, orgID, buID)
		if err != nil {
			return err
		}

		shipType, err := seedShipmentType(ctx, db, orgID, buID)
		if err != nil {
			return err
		}

		trailer, err := seedTrailer(ctx, db, equipType.ID, orgID, buID)
		if err != nil {
			return err
		}

		tractor, err := seedTractor(ctx, db, primaryWorker.ID, state.ID, equipType.ID, orgID, buID)
		if err != nil {
			return err
		}

		for i := 0; i < 100; i++ {
			input := types.CreateShipmentInput{
				BusinessUnitID:              buID,
				OrganizationID:              orgID,
				ShipmentTypeID:              shipType.ID,
				RevenueCodeID:               &revCode.ID,
				ServiceTypeID:               &serviceType.ID,
				RatingMethod:                property.ShipmentRatingMethodFlatRate,
				RatingUnit:                  1,
				OtherChargeAmount:           decimal.NewFromFloat(50),
				FreightChargeamount:         decimal.NewFromFloat(150),
				TotalChargeAmount:           decimal.NewFromFloat(200),
				CustomerID:                  customer.ID,
				OriginLocationID:            location.ID,
				OriginPlannedArrival:        time.Now(),
				OriginPlannedDeparture:      time.Now(),
				DestinationLocationID:       location.ID,
				DestinationPlannedArrival:   time.Now(),
				DestinationPlannedDeparture: time.Now(),
				PrimaryWorkerID:             primaryWorker.ID,
				TractorID:                   tractor.ID,
				TrailerID:                   trailer.ID,
				TrailerTypeID:               &equipType.ID,
				TractorTypeID:               &equipType.ID,
				BillOfLading:                "123456",
				SpecialInstructions:         "Special Instructions",
				Stops: []types.StopInput{
					{
						LocationID:       location.ID,
						Type:             property.StopTypePickup,
						PlannedArrival:   time.Now(),
						PlannedDeparture: time.Now().Add(2 * time.Hour),
					},
					{
						LocationID:       location.ID,
						Type:             property.StopTypeDelivery,
						PlannedArrival:   time.Now().Add(8 * time.Hour),
						PlannedDeparture: time.Now().Add(9 * time.Hour),
					},
				},
			}

			_, err = create(ctx, db, &input)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func create(ctx context.Context, db *bun.DB, input *types.CreateShipmentInput) (*models.Shipment, error) {
	shipment := &models.Shipment{
		EntryMethod:           "Manual",
		Status:                property.ShipmentStatusNew,
		BusinessUnitID:        input.BusinessUnitID,
		OrganizationID:        input.OrganizationID,
		CustomerID:            input.CustomerID,
		OriginLocationID:      input.OriginLocationID,
		DestinationLocationID: input.DestinationLocationID,
		ShipmentTypeID:        input.ShipmentTypeID,
		RevenueCodeID:         input.RevenueCodeID,
		ServiceTypeID:         input.ServiceTypeID,
		RatingMethod:          input.RatingMethod,
		RatingUnit:            input.RatingUnit,
		OtherChargeAmount:     input.OtherChargeAmount,
		FreightChargeAmount:   input.FreightChargeamount,
		TotalChargeAmount:     input.TotalChargeAmount,
		Pieces:                input.Pieces,
		Weight:                input.Weight,
		TrailerTypeID:         input.TrailerTypeID,
		TractorTypeID:         input.TractorTypeID,
		TemperatureMin:        input.TemperatureMin,
		TemperatureMax:        input.TemperatureMax,
		BillOfLading:          input.BillOfLading,
		SpecialInstructions:   input.SpecialInstructions,
		TrackingNumber:        input.TrackingNumber,
		Priority:              input.Priority,
		TotalDistance:         input.TotalDistance,
	}

	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Insert the shipment.
		if err := shipment.InsertShipment(ctx, db); err != nil {
			return err
		}

		// Create the initial shipment move.
		move := &models.ShipmentMove{
			BusinessUnitID:    input.BusinessUnitID,
			OrganizationID:    input.OrganizationID,
			ShipmentID:        shipment.ID,
			Status:            property.ShipmentMoveStatusNew,
			IsLoaded:          true, // All first movements are loaded by default.
			SequenceNumber:    1,
			TractorID:         input.TractorID,
			TrailerID:         input.TrailerID,
			PrimaryWorkerID:   input.PrimaryWorkerID,
			SecondaryWorkerID: input.SecondaryWorkerID,
		}

		// Insert the move
		if _, err := tx.NewInsert().Model(move).Exec(ctx); err != nil {
			return err
		}

		// Create stops
		stops, err := createStops(ctx, tx, shipment, move, input.Stops)
		if err != nil {
			return err
		}

		move.Stops = stops

		// Validate stop sequence
		if err = move.ValidateStopSequence(); err != nil {
			return err
		}

		// Insert all stops
		if _, err = tx.NewInsert().Model(&stops).Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return shipment, nil
}

func createStops(ctx context.Context, tx bun.Tx, shipment *models.Shipment, move *models.ShipmentMove, stopInputs []types.StopInput) ([]*models.Stop, error) {
	stops := make([]*models.Stop, len(stopInputs))

	for i, input := range stopInputs {
		location, err := getLocation(ctx, tx, input.LocationID)
		if err != nil {
			return nil, err
		}

		consolidatedAddress, err := consolidateAddress(ctx, tx, location)
		if err != nil {
			return nil, err
		}

		stop := &models.Stop{
			ShipmentMoveID:   move.ID,
			LocationID:       input.LocationID,
			Type:             input.Type,
			SequenceNumber:   i + 1,
			Status:           property.ShipmentMoveStatusNew,
			PlannedArrival:   input.PlannedArrival,
			PlannedDeparture: input.PlannedDeparture,
			Weight:           input.Weight,
			Pieces:           input.Pieces,
			BusinessUnitID:   shipment.BusinessUnitID,
			OrganizationID:   shipment.OrganizationID,
			AddressLine:      consolidatedAddress,
		}

		// Validate the stop
		if err = stop.Validate(); err != nil {
			return nil, err
		}

		stops[i] = stop
	}

	// Set origin and destination on shipment
	if len(stops) > 0 {
		shipment.OriginLocationID = stops[0].LocationID
		shipment.DestinationLocationID = stops[len(stops)-1].LocationID
	}

	return stops, nil
}

func getLocation(ctx context.Context, tx bun.Tx, locationID uuid.UUID) (*models.Location, error) {
	location := new(models.Location)
	err := tx.NewSelect().Model(location).Where("id = ?", locationID).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch location: %w", err)
	}
	return location, nil
}

func consolidateAddress(ctx context.Context, tx bun.Tx, location *models.Location) (string, error) {
	state := new(models.UsState)
	err := tx.NewSelect().Model(state).Where("id = ?", location.StateID).Scan(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch state: %w", err)
	}

	addressParts := []string{location.AddressLine1}

	if location.AddressLine2 != "" {
		addressParts = append(addressParts, location.AddressLine2)
	}

	addressParts = append(addressParts,
		location.City,
		state.Abbreviation,
		location.PostalCode)

	return strings.Join(addressParts, ", "), nil
}

func seedShipmentType(ctx context.Context, db *bun.DB, orgID, buID uuid.UUID) (*models.ShipmentType, error) {
	shipType := &models.ShipmentType{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         property.StatusActive,
		Code:           "ST-001",
		Color:          "#000000",
		Description:    "Shipment Type",
	}

	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(shipType).Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return shipType, nil
}

func seedTrailer(ctx context.Context, db *bun.DB, equipTypeID, orgID, buID uuid.UUID) (*models.Trailer, error) {
	trailer := &models.Trailer{
		OrganizationID:             orgID,
		BusinessUnitID:             buID,
		Code:                       "Test-Trailer",
		Status:                     "Available",
		Model:                      "Test",
		Year:                       2021,
		Vin:                        "12345678901234567",
		LicensePlateNumber:         "ABC123",
		LastInspectionDate:         &pgtype.Date{Valid: true, Time: time.Now()},
		RegistrationNumber:         "123456",
		RegistrationExpirationDate: &pgtype.Date{Valid: true, Time: time.Now()},
		EquipmentTypeID:            equipTypeID,
	}

	if _, err := db.NewInsert().Model(trailer).Exec(ctx); err != nil {
		return nil, err
	}

	return trailer, nil
}

func seedTractor(ctx context.Context, db *bun.DB, primaryWorkerID, stateID, equipTypeID, orgID, buID uuid.UUID) (*models.Tractor, error) {
	tractor := &models.Tractor{
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		Code:               "Test-Tractor",
		Status:             "Available",
		EquipmentTypeID:    equipTypeID,
		Year:               2021,
		Vin:                "12345678901234567",
		LicensePlateNumber: "ABC123",
		PrimaryWorkerID:    primaryWorkerID,
		StateID:            &stateID,
		IsLeased:           false,
	}

	if _, err := db.NewInsert().Model(tractor).Exec(ctx); err != nil {
		return nil, err
	}

	return tractor, nil
}

func seedEquipmentType(ctx context.Context, db *bun.DB, orgID, buID uuid.UUID) (*models.EquipmentType, error) {
	equipType := &models.EquipmentType{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         property.StatusActive,
		Code:           "ET",
		EquipmentClass: "Trailer",
		Description:    "Equipment Type",
		Color:          "#000000",
	}

	if _, err := db.NewInsert().Model(equipType).Exec(ctx); err != nil {
		return nil, err
	}

	return equipType, nil
}

func seedPrimaryWorker(ctx context.Context, db *bun.DB, gen *gen.CodeGenerator, stateID, orgID, buID uuid.UUID) (*models.Worker, error) {
	primaryWorker := &models.Worker{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         property.StatusActive,
		WorkerType:     property.WorkerTypeEmployee,
		FirstName:      "John",
		LastName:       "Doe",
		AddressLine1:   "123 Main St",
		AddressLine2:   "Apt 1",
		City:           "Minneapolis",
		StateID:        &stateID,
		WorkerProfile: &models.WorkerProfile{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			LicenseNumber:  "123456",
			StateID:        &stateID,
			Endorsements:   property.WorkerEndorsementNone,
			DateOfBirth:    &pgtype.Date{Valid: true, Time: time.Now()},
		},
	}

	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		mkg, mErr := models.QueryWorkerMasterKeyGenerationByOrgID(ctx, db, orgID)
		if mErr != nil {
			return mErr
		}

		return primaryWorker.InsertWorker(ctx, tx, gen, mkg.Pattern)
	})
	if err != nil {
		return nil, err
	}

	return primaryWorker, nil
}

func seedServiceType(ctx context.Context, db *bun.DB, orgID, buID uuid.UUID) (*models.ServiceType, error) {
	servType := &models.ServiceType{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         property.StatusActive,
		Code:           "ST",
		Description:    "Service Type",
	}

	if _, err := db.NewInsert().Model(servType).Exec(ctx); err != nil {
		return nil, err
	}

	return servType, nil
}

func seedRevenueCode(ctx context.Context, db *bun.DB, orgID, buID uuid.UUID) (*models.RevenueCode, error) {
	revCode := &models.RevenueCode{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         property.StatusActive,
		Code:           "RC",
		Description:    "Revenue Code",
		Color:          "#000000",
	}

	if _, err := db.NewInsert().Model(revCode).Exec(ctx); err != nil {
		return nil, err
	}

	return revCode, nil
}

func seedLocation(ctx context.Context, db *bun.DB, stateID, orgID, buID uuid.UUID) (*models.Location, error) {
	locCategory := &models.LocationCategory{
		BusinessUnitID: buID,
		OrganizationID: orgID,
		Name:           "Category",
		Description:    "Category Description",
		Color:          "#000000",
	}

	if _, err := db.NewInsert().Model(locCategory).Exec(ctx); err != nil {
		return nil, err
	}

	location := &models.Location{
		BusinessUnitID:     buID,
		OrganizationID:     orgID,
		Status:             property.StatusActive,
		LocationCategoryID: locCategory.ID,
		Name:               "Target",
		AddressLine1:       "123 Main St",
		City:               "Minneapolis",
		StateID:            stateID,
		PostalCode:         "55401",
	}

	if _, err := db.NewInsert().Model(location).Exec(ctx); err != nil {
		return nil, err
	}

	return location, nil
}

func seedCustomer(ctx context.Context, db *bun.DB, gen *gen.CodeGenerator, stateID, orgID, buID uuid.UUID) (*models.Customer, error) {
	customer := &models.Customer{
		BusinessUnitID:      buID,
		OrganizationID:      orgID,
		Status:              property.StatusActive,
		Name:                "Target-0001",
		AddressLine1:        "123 Main St",
		City:                "Minneapolis",
		StateID:             stateID,
		PostalCode:          "55401",
		AutoMarkReadyToBill: true,
	}

	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		mkg, mErr := models.QueryCustomerMasterKeyGenerationByOrgID(ctx, db, orgID)
		if mErr != nil {
			return mErr
		}

		return customer.InsertCustomer(ctx, tx, gen, mkg.Pattern)
	})
	if err != nil {
		return nil, err
	}

	return customer, nil
}
