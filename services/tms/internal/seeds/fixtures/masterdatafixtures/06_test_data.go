package masterdatafixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer/equipmentmanufacturerdb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype/equipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/shopspring/decimal"
)

type TestDataSeed struct {
}

const (
	TestTrailerEquipmentTypeCode         = "TRAILER"
	TestTractorEquipmentTypeCode         = "TRACTOR"
	TestTrailerEquipmentManufacturerName = "Freightliner"
)

func (s *TestDataSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	if err = s.createAccessorialCharges(ctx, db, sr, org.ID, org.BusinessUnitID); err != nil {
		return fmt.Errorf("create accessorial charges: %w", err)
	}

	if err = s.createEquipmentManufacturers(ctx, db, sr, org.ID, org.BusinessUnitID); err != nil {
		return fmt.Errorf("create equipment manufacturers: %w", err)
	}

	if err = s.createEquipmentTypes(ctx, db, sr, org.ID, org.BusinessUnitID); err != nil {
		return fmt.Errorf("create equipment types: %w", err)
	}

	if err = s.createTestTrailers(ctx, db, sr, org.ID, org.BusinessUnitID); err != nil {
		return fmt.Errorf("create test trailers: %w", err)
	}

	if err = s.createTestTractors(ctx, db, sr, org.ID, org.BusinessUnitID); err != nil {
		return fmt.Errorf("create test tractors: %w", err)
	}

	if err = s.createTestFleetCodes(ctx, db, sr, org.ID, org.BusinessUnitID); err != nil {
		return fmt.Errorf("create test fleet codes: %w", err)
	}

	if err = s.createTestShipmentTypes(ctx, db, sr, org.ID, org.BusinessUnitID); err != nil {
		return fmt.Errorf("create test shipment types: %w", err)
	}

	if err = s.createTestServiceTypes(ctx, db, sr, org.ID, org.BusinessUnitID); err != nil {
		return fmt.Errorf("create test service types: %w", err)
	}

	return nil
}

func (s *TestDataSeed) createAccessorialCharges(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) error {
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
			Code:           "LUMP",
			Description:    "Lumper Fee",
			Method:         accessorialcharge.MethodFlat,
			Amount:         decimal.NewFromFloat(150.00),
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

	err := accessorialchargedb.AccessorialCharges.CreateMany(ctx, db, sliceutils.Pointers(charges))
	if err != nil {
		return fmt.Errorf("insert accessorial charges: %w", err)
	}

	return nil
}

func (s *TestDataSeed) createEquipmentManufacturers(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) error {
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

	err := equipmentmanufacturerdb.EquipmentManufacturers.CreateMany(
		ctx,
		db,
		sliceutils.Pointers(manufacturers),
	)
	if err != nil {
		return fmt.Errorf("insert equipment manufacturers: %w", err)
	}

	return nil
}

func (s *TestDataSeed) createEquipmentTypes(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) error {
	types := []equipmenttype.EquipmentType{
		{
			ID:             pulid.MustNew("et_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           TestTrailerEquipmentTypeCode,
			Description:    "Dry van (standard enclosed semi-trailer)",
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

	err := equipmenttypedb.EquipmentTypes.CreateMany(ctx, db, sliceutils.Pointers(types))
	if err != nil {
		return fmt.Errorf("insert equipment types: %w", err)
	}

	return nil
}

func (s *TestDataSeed) createTestTrailers(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) error {
	trailerType, err := equipmenttypedb.EquipmentTypes.AllTenants().Where(
		equipmenttypedb.EquipmentTypes.OrganizationID.Eq(orgID),
		equipmenttypedb.EquipmentTypes.BusinessUnitID.Eq(buID),
		equipmenttypedb.EquipmentTypes.Code.Eq(TestTrailerEquipmentTypeCode),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get trailer equipment type: %w", err)
	}

	reeferType, err := equipmenttypedb.EquipmentTypes.AllTenants().Where(
		equipmenttypedb.EquipmentTypes.OrganizationID.Eq(orgID),
		equipmenttypedb.EquipmentTypes.BusinessUnitID.Eq(buID),
		equipmenttypedb.EquipmentTypes.Code.Eq("REEFER"),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get reefer equipment type: %w", err)
	}

	flatType, err := equipmenttypedb.EquipmentTypes.AllTenants().Where(
		equipmenttypedb.EquipmentTypes.OrganizationID.Eq(orgID),
		equipmenttypedb.EquipmentTypes.BusinessUnitID.Eq(buID),
		equipmenttypedb.EquipmentTypes.Code.Eq("FLAT"),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get flatbed equipment type: %w", err)
	}

	greatDane, err := equipmentmanufacturerdb.EquipmentManufacturers.AllTenants().Where(
		equipmentmanufacturerdb.EquipmentManufacturers.OrganizationID.Eq(orgID),
		equipmentmanufacturerdb.EquipmentManufacturers.BusinessUnitID.Eq(buID),
		equipmentmanufacturerdb.EquipmentManufacturers.Name.Eq("Great Dane"),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get great dane manufacturer: %w", err)
	}

	wabash, err := equipmentmanufacturerdb.EquipmentManufacturers.AllTenants().Where(
		equipmentmanufacturerdb.EquipmentManufacturers.OrganizationID.Eq(orgID),
		equipmentmanufacturerdb.EquipmentManufacturers.BusinessUnitID.Eq(buID),
		equipmentmanufacturerdb.EquipmentManufacturers.Name.Eq("Wabash"),
	).First(ctx, db)
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

			OwnershipType: domaintypes.OwnershipTypeCompanyOwned,
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

			OwnershipType: domaintypes.OwnershipTypeCompanyOwned,
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

			OwnershipType: domaintypes.OwnershipTypeCompanyOwned,
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

			OwnershipType: domaintypes.OwnershipTypeCompanyOwned,
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

			OwnershipType: domaintypes.OwnershipTypeCompanyOwned,
		},
		{
			ID:                      pulid.MustNew("tr_"),
			BusinessUnitID:          buID,
			OrganizationID:          orgID,
			EquipmentTypeID:         trailerType.ID,
			EquipmentManufacturerID: wabash.ID,
			Status:                  domaintypes.EquipmentStatusAvailable,
			Code:                    "TRL-006",
			Make:                    "Wabash",
			Model:                   "DuraPlate",
			Year:                    new(2024),

			OwnershipType: domaintypes.OwnershipTypeCompanyOwned,
		},
	}

	err = trailerdb.Trailers.CreateMany(ctx, db, sliceutils.Pointers(trailers))
	if err != nil {
		return fmt.Errorf("insert trailers: %w", err)
	}

	return nil
}

func (s *TestDataSeed) createTestTractors(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) error {
	equipType, err := equipmenttypedb.EquipmentTypes.AllTenants().Where(
		equipmenttypedb.EquipmentTypes.OrganizationID.Eq(orgID),
		equipmenttypedb.EquipmentTypes.BusinessUnitID.Eq(buID),
		equipmenttypedb.EquipmentTypes.Code.Eq(TestTractorEquipmentTypeCode),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get tractor equipment type: %w", err)
	}

	freightliner, err := equipmentmanufacturerdb.EquipmentManufacturers.AllTenants().Where(
		equipmentmanufacturerdb.EquipmentManufacturers.OrganizationID.Eq(orgID),
		equipmentmanufacturerdb.EquipmentManufacturers.BusinessUnitID.Eq(buID),
		equipmentmanufacturerdb.EquipmentManufacturers.Name.Eq("Freightliner"),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get freightliner manufacturer: %w", err)
	}

	kenworth, err := equipmentmanufacturerdb.EquipmentManufacturers.AllTenants().Where(
		equipmentmanufacturerdb.EquipmentManufacturers.OrganizationID.Eq(orgID),
		equipmentmanufacturerdb.EquipmentManufacturers.BusinessUnitID.Eq(buID),
		equipmentmanufacturerdb.EquipmentManufacturers.Name.Eq("Kenworth"),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get kenworth manufacturer: %w", err)
	}

	peterbilt, err := equipmentmanufacturerdb.EquipmentManufacturers.AllTenants().Where(
		equipmentmanufacturerdb.EquipmentManufacturers.OrganizationID.Eq(orgID),
		equipmentmanufacturerdb.EquipmentManufacturers.BusinessUnitID.Eq(buID),
		equipmentmanufacturerdb.EquipmentManufacturers.Name.Eq("Peterbilt"),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get peterbilt manufacturer: %w", err)
	}

	workers, err := workerdb.Workers.AllTenants().Where(
		workerdb.Workers.OrganizationID.Eq(orgID),
		workerdb.Workers.BusinessUnitID.Eq(buID),
		workerdb.Workers.Type.Eq(worker.WorkerTypeEmployee),
	).Order(workerdb.Workers.CreatedAt.Asc()).All(ctx, db)
	if err != nil {
		return fmt.Errorf("get workers: %w", err)
	}

	if len(workers) < 4 {
		return fmt.Errorf("need at least 4 employee workers, found %d", len(workers))
	}

	state, err := tenantfixtures.StateByAbbreviation(ctx, db, "IL")
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

			OwnershipType: domaintypes.OwnershipTypeCompanyOwned,

			FuelType: domaintypes.IFTAFuelTypeDiesel,
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

			OwnershipType: domaintypes.OwnershipTypeCompanyOwned,

			FuelType: domaintypes.IFTAFuelTypeDiesel,
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

			OwnershipType: domaintypes.OwnershipTypeCompanyOwned,

			FuelType: domaintypes.IFTAFuelTypeDiesel,
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

			OwnershipType: domaintypes.OwnershipTypeCompanyOwned,

			FuelType: domaintypes.IFTAFuelTypeDiesel,
		},
	}

	for i := range tractors {
		tractors[i].IFTAQualified = true
	}

	err = tractordb.Tractors.CreateMany(ctx, db, sliceutils.Pointers(tractors))
	if err != nil {
		return fmt.Errorf("insert tractors: %w", err)
	}

	return nil
}

func (s *TestDataSeed) createTestFleetCodes(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) error {
	manager, err := tenantfixtures.UserByUsername(ctx, db, "admin")
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

			Status: domaintypes.StatusActive,
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

			Status: domaintypes.StatusActive,
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

			Status: domaintypes.StatusActive,
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

			Status: domaintypes.StatusActive,
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

			Status: domaintypes.StatusActive,
		},
	}

	err = fleetcodedb.FleetCodes.CreateMany(ctx, db, sliceutils.Pointers(fleetCodes))
	if err != nil {
		return fmt.Errorf("insert fleet codes: %w", err)
	}

	return nil
}

func (s *TestDataSeed) createTestShipmentTypes(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) error {
	shipmentTypes := []shipmenttype.ShipmentType{
		{
			ID:             pulid.MustNew("sht_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "FTL",
			Description:    "Full Truckload",
			Color:          "#2563EB",

			Status: domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("sht_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "LTL",
			Description:    "Less Than Truckload",
			Color:          "#7C3AED",

			Status: domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("sht_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "PART",
			Description:    "Partial Truckload",
			Color:          "#0891B2",

			Status: domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("sht_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "DRAY",
			Description:    "Drayage — port/rail container pickup and delivery",
			Color:          "#EA580C",

			Status: domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("sht_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "INTM",
			Description:    "Intermodal — rail and truck combination move",
			Color:          "#0D9488",

			Status: domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("sht_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "EXP",
			Description:    "Expedited — time-critical guaranteed delivery",
			Color:          "#DC2626",

			Status: domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("sht_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "HAZM",
			Description:    "Hazmat — shipment requiring hazardous materials handling",
			Color:          "#CA8A04",

			Status: domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("sht_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "TANK",
			Description:    "Tanker — bulk liquid or gas freight",
			Color:          "#4F46E5",

			Status: domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("sht_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "FLAT",
			Description:    "Flatbed — oversized or non-containerized freight",
			Color:          "#9333EA",

			Status: domaintypes.StatusActive,
		},
		{
			ID:             pulid.MustNew("sht_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Code:           "REEF",
			Description:    "Refrigerated — temperature-controlled freight",
			Color:          "#0284C7",

			Status: domaintypes.StatusActive,
		},
	}

	err := shipmenttypedb.ShipmentTypes.CreateMany(ctx, db, sliceutils.Pointers(shipmentTypes))
	if err != nil {
		return fmt.Errorf("insert shipment types: %w", err)
	}

	return nil
}

func (s *TestDataSeed) createTestServiceTypes(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) error {
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

	err := servicetypedb.ServiceTypes.CreateMany(ctx, db, sliceutils.Pointers(serviceTypes))
	if err != nil {
		return fmt.Errorf("insert service types: %w", err)
	}

	return nil
}
