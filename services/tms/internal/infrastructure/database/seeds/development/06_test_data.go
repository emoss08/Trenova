package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type TestDataSeed struct {
	seedhelpers.BaseSeed
}

func NewTestDataSeed() *TestDataSeed {
	seed := &TestDataSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"TestData",
		"1.0.0",
		"Creates test data for development environment",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)
	seed.SetDependencies(seedhelpers.SeedAdminAccount, seedhelpers.SeedWorker)
	return seed
}

const (
	TestTrailerEquipmentTypeCode         = "TRAILER"
	TestTractorEquipmentTypeCode         = "TRACTOR"
	TestTrailerEquipmentManufacturerName = "Freightliner"
)

func (s *TestDataSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}

			if err = s.createAccessorialCharges(ctx, tx, sc, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("create accessorial charges: %w", err)
			}

			if err = s.createEquipmentManufacturers(ctx, tx, sc, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("create equipment manufacturers: %w", err)
			}

			if err = s.createEquipmentTypes(ctx, tx, sc, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("create equipment types: %w", err)
			}

			if err = s.createTestTrailers(ctx, tx, sc, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("create test trailers: %w", err)
			}

			if err = s.createTestTractors(ctx, tx, sc, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("create test tractors: %w", err)
			}

			if err = s.createTestFleetCodes(ctx, tx, sc, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("create test fleet codes: %w", err)
			}

			if err = s.createTestShipmentTypes(ctx, tx, sc, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("create test shipment types: %w", err)
			}

			if err = s.createTestServiceTypes(ctx, tx, sc, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("create test service types: %w", err)
			}

			return nil
		},
	)
}

func (s *TestDataSeed) createAccessorialCharges(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) error {
	count, err := tx.NewSelect().
		Model((*accessorialcharge.AccessorialCharge)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check existing accessorial charges: %w", err)
	}

	if count > 0 {
		return nil
	}

	charges := []accessorialcharge.AccessorialCharge{
		{
			ID:             pulid.MustNew("acc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "DET",
			Description:    "Detention Fee",
			Method:         accessorialcharge.MethodPerUnit,
			RateUnit:       accessorialcharge.RateUnitHour,
			Amount:         decimal.NewFromFloat(75.00),
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("acc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "FUEL",
			Description:    "Fuel Surcharge",
			Method:         accessorialcharge.MethodPercentage,
			Amount:         decimal.NewFromFloat(12.50),
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("acc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "HAZ",
			Description:    "Hazardous Material Handling",
			Method:         accessorialcharge.MethodFlat,
			Amount:         decimal.NewFromFloat(350.00),
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("acc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "TONU",
			Description:    "Truck Ordered Not Used",
			Method:         accessorialcharge.MethodFlat,
			Amount:         decimal.NewFromFloat(250.00),
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("acc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "LAY",
			Description:    "Layover",
			Method:         accessorialcharge.MethodFlat,
			Amount:         decimal.NewFromFloat(200.00),
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("acc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "STOP",
			Description:    "Stop-Off Charge",
			Method:         accessorialcharge.MethodPerUnit,
			RateUnit:       accessorialcharge.RateUnitStop,
			Amount:         decimal.NewFromFloat(50.00),
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("acc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "TEAM",
			Description:    "Team Driver Surcharge",
			Method:         accessorialcharge.MethodPerUnit,
			RateUnit:       accessorialcharge.RateUnitMile,
			Amount:         decimal.NewFromFloat(0.25),
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("acc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "STORE",
			Description:    "Storage Fee",
			Method:         accessorialcharge.MethodPerUnit,
			RateUnit:       accessorialcharge.RateUnitDay,
			Amount:         decimal.NewFromFloat(100.00),
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("acc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "TARP",
			Description:    "Tarping Fee",
			Method:         accessorialcharge.MethodFlat,
			Amount:         decimal.NewFromFloat(75.00),
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("acc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "RESDL",
			Description:    "Residential Delivery",
			Method:         accessorialcharge.MethodFlat,
			Amount:         decimal.NewFromFloat(125.00),
			Status:         domaintypes.StatusActive,
		},
	}

	_, err = tx.NewInsert().
		Model(&charges).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert accessorial charges: %w", err)
	}

	for i := range charges {
		if err = sc.TrackCreated(ctx, "accessorial_charges", charges[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track accessorial charge: %w", err)
		}
	}

	seedhelpers.LogSuccess(
		"Created accessorial charge fixtures",
		fmt.Sprintf("- Created %d accessorial charges", len(charges)),
	)

	return nil
}

func (s *TestDataSeed) createEquipmentManufacturers(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) error {
	count, err := tx.NewSelect().
		Model((*equipmentmanufacturer.EquipmentManufacturer)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check existing equipment manufacturers: %w", err)
	}

	if count > 0 {
		return nil
	}

	manufacturers := []equipmentmanufacturer.EquipmentManufacturer{
		{
			ID:             pulid.MustNew("em_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Name:           TestTrailerEquipmentManufacturerName,
			Description:    "Freightliner LLC — Class 5-8 trucks and chassis",
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("em_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Name:           "Kenworth",
			Description:    "Kenworth Truck Company — Class 5-8 trucks",
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("em_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Name:           "Peterbilt",
			Description:    "Peterbilt Motors Company — Medium and heavy-duty trucks",
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("em_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Name:           "Volvo",
			Description:    "Volvo Trucks — Heavy-duty trucks and powertrain systems",
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("em_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Name:           "International",
			Description:    "International Trucks — Medium and heavy-duty commercial vehicles",
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("em_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Name:           "Mack",
			Description:    "Mack Trucks — Heavy-duty and vocational trucks",
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("em_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Name:           "Great Dane",
			Description:    "Great Dane Trailers — Dry van, refrigerated, and flatbed trailers",
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("em_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Name:           "Wabash",
			Description:    "Wabash National — Dry van, refrigerated, and platform trailers",
			Status:         domaintypes.StatusActive,
		},
	}

	_, err = tx.NewInsert().
		Model(&manufacturers).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert equipment manufacturers: %w", err)
	}

	for i := range manufacturers {
		if err = sc.TrackCreated(ctx, "equipment_manufacturers", manufacturers[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track equipment manufacturer: %w", err)
		}
	}

	seedhelpers.LogSuccess(
		"Created equipment manufacturer fixtures",
		fmt.Sprintf("- Created %d equipment manufacturers", len(manufacturers)),
	)

	return nil
}

func (s *TestDataSeed) createEquipmentTypes(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) error {
	count, err := tx.NewSelect().
		Model((*equipmenttype.EquipmentType)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check existing equipment types: %w", err)
	}

	if count > 0 {
		return nil
	}

	types := []equipmenttype.EquipmentType{
		{
			ID:             pulid.MustNew("et_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           TestTrailerEquipmentTypeCode,
			Description:    "Standard semi-trailer",
			Class:          equipmenttype.ClassTrailer,
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("et_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           TestTractorEquipmentTypeCode,
			Description:    "Standard day cab or sleeper tractor",
			Class:          equipmenttype.ClassTractor,
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("et_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "REEFER",
			Description:    "Refrigerated trailer",
			Class:          equipmenttype.ClassTrailer,
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("et_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "FLAT",
			Description:    "Flatbed trailer",
			Class:          equipmenttype.ClassTrailer,
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("et_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "TANK",
			Description:    "Tanker trailer for liquid freight",
			Class:          equipmenttype.ClassTrailer,
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("et_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "CONT",
			Description:    "Intermodal container",
			Class:          equipmenttype.ClassContainer,
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("et_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "LOWBOY",
			Description:    "Low-profile heavy haul trailer",
			Class:          equipmenttype.ClassTrailer,
			Status:         domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("et_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "OTHER",
			Description:    "Miscellaneous equipment",
			Class:          equipmenttype.ClassOther,
			Status:         domaintypes.StatusActive,
		},
	}

	_, err = tx.NewInsert().
		Model(&types).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert equipment types: %w", err)
	}

	for i := range types {
		if err = sc.TrackCreated(ctx, "equipment_types", types[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track equipment type: %w", err)
		}
	}

	seedhelpers.LogSuccess(
		"Created equipment type fixtures",
		fmt.Sprintf("- Created %d equipment types", len(types)),
	)

	return nil
}

func (s *TestDataSeed) createTestTrailers(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) error {
	count, err := tx.NewSelect().
		Model((*trailer.Trailer)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check existing trailers: %w", err)
	}

	if count > 0 {
		return nil
	}

	var trailerType equipmenttype.EquipmentType
	err = tx.NewSelect().
		Model(&trailerType).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("code = ?", TestTrailerEquipmentTypeCode).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get trailer equipment type: %w", err)
	}

	var reeferType equipmenttype.EquipmentType
	err = tx.NewSelect().
		Model(&reeferType).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("code = ?", "REEFER").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get reefer equipment type: %w", err)
	}

	var flatType equipmenttype.EquipmentType
	err = tx.NewSelect().
		Model(&flatType).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("code = ?", "FLAT").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get flatbed equipment type: %w", err)
	}

	var freightliner equipmentmanufacturer.EquipmentManufacturer
	err = tx.NewSelect().
		Model(&freightliner).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("name = ?", TestTrailerEquipmentManufacturerName).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get freightliner manufacturer: %w", err)
	}

	var greatDane equipmentmanufacturer.EquipmentManufacturer
	err = tx.NewSelect().
		Model(&greatDane).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("name = ?", "Great Dane").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get great dane manufacturer: %w", err)
	}

	var wabash equipmentmanufacturer.EquipmentManufacturer
	err = tx.NewSelect().
		Model(&wabash).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("name = ?", "Wabash").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get wabash manufacturer: %w", err)
	}

	trailers := []trailer.Trailer{
		{
			ID:                      pulid.MustNew("tr_"),
			BusinessUnitID:          buID,
			OrganizationID:          orgID,
			EquipmentTypeID:         trailerType.ID,
			EquipmentManufacturerID: greatDane.ID,
			Status:                  domaintypes.EquipmentStatusAvailable,
			Code:                    "TRL-001",
			Make:                    "Great Dane",
			Model:                   "Champion SE",
			Year:                    new(2023),
		},
		{
			ID:                      pulid.MustNew("tr_"),
			BusinessUnitID:          buID,
			OrganizationID:          orgID,
			EquipmentTypeID:         trailerType.ID,
			EquipmentManufacturerID: wabash.ID,
			Status:                  domaintypes.EquipmentStatusAvailable,
			Code:                    "TRL-002",
			Make:                    "Wabash",
			Model:                   "DuraPlate HD",
			Year:                    new(2022),
		},
		{
			ID:                      pulid.MustNew("tr_"),
			BusinessUnitID:          buID,
			OrganizationID:          orgID,
			EquipmentTypeID:         reeferType.ID,
			EquipmentManufacturerID: greatDane.ID,
			Status:                  domaintypes.EquipmentStatusAvailable,
			Code:                    "TRL-003",
			Make:                    "Great Dane",
			Model:                   "Everest TL",
			Year:                    new(2024),
		},
		{
			ID:                      pulid.MustNew("tr_"),
			BusinessUnitID:          buID,
			OrganizationID:          orgID,
			EquipmentTypeID:         flatType.ID,
			EquipmentManufacturerID: wabash.ID,
			Status:                  domaintypes.EquipmentStatusAvailable,
			Code:                    "TRL-004",
			Make:                    "Wabash",
			Model:                   "FreightPro",
			Year:                    new(2021),
		},
		{
			ID:                      pulid.MustNew("tr_"),
			BusinessUnitID:          buID,
			OrganizationID:          orgID,
			EquipmentTypeID:         trailerType.ID,
			EquipmentManufacturerID: greatDane.ID,
			Status:                  domaintypes.EquipmentStatusOOS,
			Code:                    "TRL-005",
			Make:                    "Great Dane",
			Model:                   "Champion CL",
			Year:                    new(2019),
		},
	}

	_, err = tx.NewInsert().
		Model(&trailers).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert trailers: %w", err)
	}

	for i := range trailers {
		if err = sc.TrackCreated(ctx, "trailers", trailers[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track trailer: %w", err)
		}
	}

	seedhelpers.LogSuccess(
		"Created trailer fixtures",
		fmt.Sprintf("- Created %d trailers", len(trailers)),
	)

	return nil
}

func (s *TestDataSeed) createTestTractors(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) error {
	count, err := tx.NewSelect().
		Model((*tractor.Tractor)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check existing tractors: %w", err)
	}
	if count > 0 {
		return nil
	}

	var equipType equipmenttype.EquipmentType
	err = tx.NewSelect().
		Model(&equipType).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("code = ?", TestTractorEquipmentTypeCode).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get tractor equipment type: %w", err)
	}

	var freightliner equipmentmanufacturer.EquipmentManufacturer
	err = tx.NewSelect().
		Model(&freightliner).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("name = ?", "Freightliner").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get freightliner manufacturer: %w", err)
	}

	var kenworth equipmentmanufacturer.EquipmentManufacturer
	err = tx.NewSelect().
		Model(&kenworth).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("name = ?", "Kenworth").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get kenworth manufacturer: %w", err)
	}

	var peterbilt equipmentmanufacturer.EquipmentManufacturer
	err = tx.NewSelect().
		Model(&peterbilt).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("name = ?", "Peterbilt").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get peterbilt manufacturer: %w", err)
	}

	var workers []worker.Worker
	err = tx.NewSelect().
		Model(&workers).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("type = ?", worker.WorkerTypeEmployee).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get workers: %w", err)
	}

	if len(workers) < 4 {
		return fmt.Errorf("need at least 4 employee workers, found %d", len(workers))
	}

	state, err := sc.GetState(ctx, "IL")
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	tractors := []tractor.Tractor{
		{
			ID:                      pulid.MustNew("trac_"),
			BusinessUnitID:          buID,
			OrganizationID:          orgID,
			EquipmentTypeID:         equipType.ID,
			EquipmentManufacturerID: freightliner.ID,
			Status:                  domaintypes.EquipmentStatusAvailable,
			PrimaryWorkerID:         workers[0].ID,
			Code:                    "TRC-001",
			Make:                    "Freightliner",
			Model:                   "Cascadia",
			Year:                    new(2023),
			LicensePlateNumber:      "IL-TRC-1001",
			StateID:                 state.ID,
			RegistrationNumber:      "REG-FL-001",
		},
		{
			ID:                      pulid.MustNew("trac_"),
			BusinessUnitID:          buID,
			OrganizationID:          orgID,
			EquipmentTypeID:         equipType.ID,
			EquipmentManufacturerID: kenworth.ID,
			Status:                  domaintypes.EquipmentStatusAvailable,
			PrimaryWorkerID:         workers[1].ID,
			Code:                    "TRC-002",
			Make:                    "Kenworth",
			Model:                   "T680",
			Year:                    new(2022),
			LicensePlateNumber:      "IL-TRC-1002",
			StateID:                 state.ID,
			RegistrationNumber:      "REG-KW-002",
		},
		{
			ID:                      pulid.MustNew("trac_"),
			BusinessUnitID:          buID,
			OrganizationID:          orgID,
			EquipmentTypeID:         equipType.ID,
			EquipmentManufacturerID: peterbilt.ID,
			Status:                  domaintypes.EquipmentStatusAvailable,
			PrimaryWorkerID:         workers[2].ID,
			Code:                    "TRC-003",
			Make:                    "Peterbilt",
			Model:                   "579",
			Year:                    new(2024),
			LicensePlateNumber:      "IL-TRC-1003",
			StateID:                 state.ID,
			RegistrationNumber:      "REG-PB-003",
		},
		{
			ID:                      pulid.MustNew("trac_"),
			BusinessUnitID:          buID,
			OrganizationID:          orgID,
			EquipmentTypeID:         equipType.ID,
			EquipmentManufacturerID: freightliner.ID,
			Status:                  domaintypes.EquipmentStatusAtMaintenance,
			PrimaryWorkerID:         workers[3].ID,
			Code:                    "TRC-004",
			Make:                    "Freightliner",
			Model:                   "Cascadia",
			Year:                    new(2020),
			LicensePlateNumber:      "IL-TRC-1004",
			StateID:                 state.ID,
			RegistrationNumber:      "REG-FL-004",
		},
	}

	_, err = tx.NewInsert().
		Model(&tractors).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert tractors: %w", err)
	}

	for i := range tractors {
		if err = sc.TrackCreated(ctx, "tractors", tractors[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track tractor: %w", err)
		}
	}

	seedhelpers.LogSuccess(
		"Created tractor fixtures",
		fmt.Sprintf("- Created %d tractors", len(tractors)),
	)

	return nil
}

func (s *TestDataSeed) createTestFleetCodes(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) error {
	count, err := tx.NewSelect().
		Model((*fleetcode.FleetCode)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check existing fleet codes: %w", err)
	}
	if count > 0 {
		return nil
	}

	manager, err := sc.GetUserByUsername(ctx, "admin")
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}

	fleetCodes := []fleetcode.FleetCode{
		{
			ID:             pulid.MustNew("fc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "OTR",
			Description:    "Over the Road — long-haul interstate freight",
			ManagerID:      manager.ID,
			RevenueGoal:    new(250000.00),
			DeadheadGoal:   new(12.00),
			MileageGoal:    new(10000.00),
		},
		{
			ID:             pulid.MustNew("fc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "LOCAL",
			Description:    "Local — same-day metro area delivery",
			ManagerID:      manager.ID,
			RevenueGoal:    new(80000.00),
			DeadheadGoal:   new(8.00),
			MileageGoal:    new(3000.00),
		},
		{
			ID:             pulid.MustNew("fc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "REG",
			Description:    "Regional — multi-state short-haul routes",
			ManagerID:      manager.ID,
			RevenueGoal:    new(150000.00),
			DeadheadGoal:   new(10.00),
			MileageGoal:    new(6000.00),
		},
		{
			ID:             pulid.MustNew("fc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "DED",
			Description:    "Dedicated — contracted lanes for a single customer",
			ManagerID:      manager.ID,
			RevenueGoal:    new(200000.00),
			DeadheadGoal:   new(5.00),
			MileageGoal:    new(8000.00),
		},
		{
			ID:             pulid.MustNew("fc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "INTM",
			Description:    "Intermodal — rail and dray container moves",
			ManagerID:      manager.ID,
			RevenueGoal:    new(120000.00),
			DeadheadGoal:   new(15.00),
			MileageGoal:    new(4000.00),
		},
	}

	_, err = tx.NewInsert().
		Model(&fleetCodes).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert fleet codes: %w", err)
	}

	for i := range fleetCodes {
		if err = sc.TrackCreated(ctx, "fleet_codes", fleetCodes[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track fleet code: %w", err)
		}
	}

	seedhelpers.LogSuccess(
		"Created fleet code fixtures",
		fmt.Sprintf("- Created %d fleet codes", len(fleetCodes)),
	)

	return nil
}

func (s *TestDataSeed) createTestShipmentTypes(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) error {
	count, err := tx.NewSelect().
		Model((*shipmenttype.ShipmentType)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check existing shipment types: %w", err)
	}
	if count > 0 {
		return nil
	}

	shipmentTypes := []shipmenttype.ShipmentType{
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "FTL",
			Description:    "Full Truckload",
			Color:          "#2563EB",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "LTL",
			Description:    "Less Than Truckload",
			Color:          "#7C3AED",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "PART",
			Description:    "Partial Truckload",
			Color:          "#0891B2",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "DRAY",
			Description:    "Drayage — port/rail container pickup and delivery",
			Color:          "#EA580C",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "INTM",
			Description:    "Intermodal — rail and truck combination move",
			Color:          "#0D9488",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "EXP",
			Description:    "Expedited — time-critical guaranteed delivery",
			Color:          "#DC2626",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "HAZM",
			Description:    "Hazmat — shipment requiring hazardous materials handling",
			Color:          "#CA8A04",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "TANK",
			Description:    "Tanker — bulk liquid or gas freight",
			Color:          "#4F46E5",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "FLAT",
			Description:    "Flatbed — oversized or non-containerized freight",
			Color:          "#9333EA",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "REEF",
			Description:    "Refrigerated — temperature-controlled freight",
			Color:          "#0284C7",
		},
	}

	_, err = tx.NewInsert().
		Model(&shipmentTypes).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert shipment types: %w", err)
	}

	for i := range shipmentTypes {
		if err = sc.TrackCreated(ctx, "shipment_types", shipmentTypes[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track shipment type: %w", err)
		}
	}

	seedhelpers.LogSuccess(
		"Created shipment type fixtures",
		fmt.Sprintf("- Created %d shipment types", len(shipmentTypes)),
	)

	return nil
}

func (s *TestDataSeed) createTestServiceTypes(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) error {
	count, err := tx.NewSelect().
		Model((*servicetype.ServiceType)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check existing service types: %w", err)
	}
	if count > 0 {
		return nil
	}

	serviceTypes := []servicetype.ServiceType{
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "STD",
			Description:    "Standard — regular transit time, no special handling",
			Status:         domaintypes.StatusActive,
			Color:          "#6B7280",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "EXP",
			Description:    "Expedited — guaranteed faster transit window",
			Status:         domaintypes.StatusActive,
			Color:          "#DC2626",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "TEAM",
			Description:    "Team — dual-driver non-stop relay for maximum speed",
			Status:         domaintypes.StatusActive,
			Color:          "#7C3AED",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "WHG",
			Description:    "White Glove — inside delivery with special care handling",
			Status:         domaintypes.StatusActive,
			Color:          "#2563EB",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "TEMP",
			Description:    "Temperature Controlled — monitored climate throughout transit",
			Status:         domaintypes.StatusActive,
			Color:          "#0891B2",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "HAZM",
			Description:    "Hazmat — certified hazardous materials transport",
			Status:         domaintypes.StatusActive,
			Color:          "#CA8A04",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "LIFT",
			Description:    "Liftgate — hydraulic lift at pickup or delivery",
			Status:         domaintypes.StatusActive,
			Color:          "#EA580C",
		},
		{
			ID:             pulid.MustNew("st_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "DTP",
			Description:    "Drop Trailer — pre-positioned trailer for shipper loading",
			Status:         domaintypes.StatusActive,
			Color:          "#0D9488",
		},
	}

	_, err = tx.NewInsert().
		Model(&serviceTypes).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert service types: %w", err)
	}

	for i := range serviceTypes {
		if err = sc.TrackCreated(ctx, "service_types", serviceTypes[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track service type: %w", err)
		}
	}

	seedhelpers.LogSuccess(
		"Created service type fixtures",
		fmt.Sprintf("- Created %d service types", len(serviceTypes)),
	)

	return nil
}

func (s *TestDataSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *TestDataSeed) CanRollback() bool {
	return true
}
