package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type ShipmentSeed struct {
	seedhelpers.BaseSeed
}

func NewShipmentSeed() *ShipmentSeed {
	seed := &ShipmentSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"Shipment",
		"1.0.0",
		"Creates sample shipments with moves, stops, commodities, and charges for development",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)
	seed.SetDependencies(
		seedhelpers.SeedTestData,
		seedhelpers.SeedFormulaTemplate,
		seedhelpers.SeedLocation,
	)
	return seed
}

func (s *ShipmentSeed) Run(ctx context.Context, tx bun.Tx) error {
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

			if err = s.createCustomers(ctx, tx, sc, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("create customers: %w", err)
			}

			if err = s.createCommodities(ctx, tx, sc, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("create commodities: %w", err)
			}

			if err = s.createShipments(ctx, tx, sc, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("create shipments: %w", err)
			}

			return nil
		},
	)
}

// -- Customers --

func (s *ShipmentSeed) createCustomers(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) error {
	count, err := tx.NewSelect().
		Model((*customer.Customer)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check existing customers: %w", err)
	}

	if count > 0 {
		return nil
	}

	ilState, err := sc.GetState(ctx, "IL")
	if err != nil {
		return fmt.Errorf("get IL state: %w", err)
	}

	caState, err := sc.GetState(ctx, "CA")
	if err != nil {
		return fmt.Errorf("get CA state: %w", err)
	}

	txState, err := sc.GetState(ctx, "TX")
	if err != nil {
		return fmt.Errorf("get TX state: %w", err)
	}

	azState, err := sc.GetState(ctx, "AZ")
	if err != nil {
		return fmt.Errorf("get AZ state: %w", err)
	}

	coState, err := sc.GetState(ctx, "CO")
	if err != nil {
		return fmt.Errorf("get CO state: %w", err)
	}

	flState, err := sc.GetState(ctx, "FL")
	if err != nil {
		return fmt.Errorf("get FL state: %w", err)
	}

	customers := []customer.Customer{
		{
			ID:             pulid.MustNew("cus_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			StateID:        ilState.ID,
			Status:         domaintypes.StatusActive,
			Code:           "ACME",
			Name:           "Acme Manufacturing",
			AddressLine1:   "400 W Superior St",
			City:           "Chicago",
			PostalCode:     "60654",
		},
		{
			ID:             pulid.MustNew("cus_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			StateID:        caState.ID,
			Status:         domaintypes.StatusActive,
			Code:           "GLBL",
			Name:           "GlobalTrade Imports",
			AddressLine1:   "2500 E Olympic Blvd",
			City:           "Los Angeles",
			PostalCode:     "90023",
		},
		{
			ID:             pulid.MustNew("cus_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			StateID:        txState.ID,
			Status:         domaintypes.StatusActive,
			Code:           "PEAK",
			Name:           "Peak Distributing",
			AddressLine1:   "1200 Commerce St",
			City:           "Dallas",
			PostalCode:     "75202",
		},
		{
			ID:             pulid.MustNew("cus_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			StateID:        azState.ID,
			Status:         domaintypes.StatusActive,
			Code:           "FRSH",
			Name:           "FreshHaul Foods",
			AddressLine1:   "3300 N Central Ave",
			City:           "Phoenix",
			PostalCode:     "85012",
		},
		{
			ID:             pulid.MustNew("cus_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			StateID:        coState.ID,
			Status:         domaintypes.StatusActive,
			Code:           "RNGE",
			Name:           "Range Logistics",
			AddressLine1:   "1801 California St",
			City:           "Denver",
			PostalCode:     "80202",
		},
		{
			ID:             pulid.MustNew("cus_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			StateID:        flState.ID,
			Status:         domaintypes.StatusActive,
			Code:           "SUNB",
			Name:           "Sunbelt Materials",
			AddressLine1:   "800 Brickell Ave",
			City:           "Miami",
			PostalCode:     "33131",
		},
	}

	_, err = tx.NewInsert().
		Model(&customers).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert customers: %w", err)
	}

	billingProfiles := make([]customer.CustomerBillingProfile, len(customers))
	for i := range customers {
		billingProfiles[i] = *customer.NewDefaultBillingProfile(orgID, buID, customers[i].ID)
	}

	_, err = tx.NewInsert().
		Model(&billingProfiles).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert customer billing profiles: %w", err)
	}

	for i := range customers {
		if err = sc.TrackCreated(ctx, "customers", customers[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track customer: %w", err)
		}
	}

	for i := range billingProfiles {
		if err = sc.TrackCreated(ctx, "customer_billing_profiles", billingProfiles[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track customer billing profile: %w", err)
		}
	}

	seedhelpers.LogSuccess(
		"Created customer fixtures",
		fmt.Sprintf("- Created %d customers with billing profiles", len(customers)),
	)

	return nil
}

// -- Commodities --

func (s *ShipmentSeed) createCommodities(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) error {
	count, err := tx.NewSelect().
		Model((*commodity.Commodity)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check existing commodities: %w", err)
	}

	if count > 0 {
		return nil
	}

	commodities := []commodity.Commodity{
		{
			ID:             pulid.MustNew("com_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			Name:           "General Freight",
			Description:    "Standard palletized dry goods",
			FreightClass:   commodity.FreightClass70,
			WeightPerUnit:  new(1500.00),
			Stackable:      true,
		},
		{
			ID:             pulid.MustNew("com_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			Name:           "Electronics",
			Description:    "Consumer electronics on pallets",
			FreightClass:   commodity.FreightClass92_5,
			WeightPerUnit:  new(800.00),
			Fragile:        true,
		},
		{
			ID:             pulid.MustNew("com_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			Name:           "Frozen Produce",
			Description:    "Temperature-controlled produce shipments",
			FreightClass:   commodity.FreightClass85,
			WeightPerUnit:  new(2000.00),
			MinTemperature: new(28),
			MaxTemperature: new(34),
			Stackable:      true,
		},
		{
			ID:             pulid.MustNew("com_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			Name:           "Steel Coils",
			Description:    "Rolled steel coils for manufacturing",
			FreightClass:   commodity.FreightClass50,
			WeightPerUnit:  new(5000.00),
		},
		{
			ID:                pulid.MustNew("com_"),
			BusinessUnitID:    buID,
			OrganizationID:    orgID,
			Status:            domaintypes.StatusActive,
			Name:              "Lumber",
			Description:       "Dimensional lumber and plywood",
			FreightClass:      commodity.FreightClass60,
			WeightPerUnit:     new(3000.00),
			LinearFeetPerUnit: new(8.00),
			Stackable:         true,
		},
		{
			ID:             pulid.MustNew("com_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			Name:           "Auto Parts",
			Description:    "Automotive parts and components",
			FreightClass:   commodity.FreightClass85,
			WeightPerUnit:  new(1200.00),
			Fragile:        true,
		},
	}

	_, err = tx.NewInsert().
		Model(&commodities).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert commodities: %w", err)
	}

	for i := range commodities {
		if err = sc.TrackCreated(ctx, "commodities", commodities[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track commodity: %w", err)
		}
	}

	seedhelpers.LogSuccess(
		"Created commodity fixtures",
		fmt.Sprintf("- Created %d commodities", len(commodities)),
	)

	return nil
}

func (s *ShipmentSeed) createShipments(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) error {
	count, err := tx.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check existing shipments: %w", err)
	}

	if count > 0 {
		return nil
	}

	locs, err := s.getLocations(ctx, tx, orgID, buID)
	if err != nil {
		return err
	}

	custs, err := s.getCustomers(ctx, tx, orgID, buID)
	if err != nil {
		return err
	}

	svcTypes, err := s.getServiceTypes(ctx, tx, orgID, buID)
	if err != nil {
		return err
	}

	shpTypes, err := s.getShipmentTypes(ctx, tx, orgID, buID)
	if err != nil {
		return err
	}

	comms, err := s.getCommodities(ctx, tx, orgID, buID)
	if err != nil {
		return err
	}

	accCharges, err := s.getAccessorialCharges(ctx, tx, orgID, buID)
	if err != nil {
		return err
	}

	var flatRateTemplate formulatemplate.FormulaTemplate
	err = tx.NewSelect().
		Model(&flatRateTemplate).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("name = ?", "Flat Rate").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get flat rate formula template: %w", err)
	}

	var perMileTemplate formulatemplate.FormulaTemplate
	err = tx.NewSelect().
		Model(&perMileTemplate).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("name = ?", "Per Mile").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get per mile formula template: %w", err)
	}

	var perStopTemplate formulatemplate.FormulaTemplate
	err = tx.NewSelect().
		Model(&perStopTemplate).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("name = ?", "Per Stop").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("get per stop formula template: %w", err)
	}

	now := timeutils.NowUnix()
	day := int64(86400)

	locLA := locs["TERM-LA"]
	locDallas := locs["WH-DAL"]
	locChicago := locs["DC-CHI"]
	locPhoenix := locs["COLD-PHX"]
	locDenver := locs["CUST-DEN"]
	locMiami := locs["MAINT-01"]

	// -- Pre-generate IDs --

	shp1ID := pulid.MustNew("shp_")
	shp1M1 := pulid.MustNew("sm_")

	shp2ID := pulid.MustNew("shp_")
	shp2M1 := pulid.MustNew("sm_")

	shp3ID := pulid.MustNew("shp_")
	shp3M1 := pulid.MustNew("sm_")
	shp3M2 := pulid.MustNew("sm_")

	shp4ID := pulid.MustNew("shp_")
	shp4M1 := pulid.MustNew("sm_")

	shp5ID := pulid.MustNew("shp_")
	shp5M1 := pulid.MustNew("sm_")

	shp6ID := pulid.MustNew("shp_")
	shp6M1 := pulid.MustNew("sm_")
	shp6M2 := pulid.MustNew("sm_")
	shp6M3 := pulid.MustNew("sm_")

	shp7ID := pulid.MustNew("shp_")
	shp7M1 := pulid.MustNew("sm_")

	shp8ID := pulid.MustNew("shp_")
	shp8M1 := pulid.MustNew("sm_")
	shp8M2 := pulid.MustNew("sm_")

	shp9ID := pulid.MustNew("shp_")
	shp9M1 := pulid.MustNew("sm_")

	shp10ID := pulid.MustNew("shp_")
	shp10M1 := pulid.MustNew("sm_")

	// -- Shipments --

	allShipments := []shipment.Shipment{
		{
			ID: shp1ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["ACME"], ServiceTypeID: svcTypes["STD"], ShipmentTypeID: shpTypes["FTL"],
			FormulaTemplateID: flatRateTemplate.ID, Status: shipment.StatusNew,
			ProNumber: "SEED-SHP-001", BOL: "BOL-2026-0001",
			Pieces: new(int64(24)), Weight: new(int64(42000)),
			FreightChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(3200.00),
				Valid:   true,
			},
			TotalChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(3200.00),
				Valid:   true,
			},
			RatingUnit: 1,
		},
		{
			ID: shp2ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["PEAK"], ServiceTypeID: svcTypes["STD"], ShipmentTypeID: shpTypes["LTL"],
			FormulaTemplateID: perMileTemplate.ID, Status: shipment.StatusNew,
			ProNumber: "SEED-SHP-002", BOL: "BOL-2026-0002",
			Pieces: new(int64(8)), Weight: new(int64(12000)),
			FreightChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(1850.00),
				Valid:   true,
			},
			OtherChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(50.00),
				Valid:   true,
			},
			TotalChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(1900.00),
				Valid:   true,
			},
			RatingUnit: 1,
		},
		{
			ID: shp3ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["GLBL"], ServiceTypeID: svcTypes["EXP"], ShipmentTypeID: shpTypes["FTL"],
			FormulaTemplateID: perMileTemplate.ID, Status: shipment.StatusNew,
			ProNumber: "SEED-SHP-003", BOL: "BOL-2026-0003",
			Pieces: new(int64(30)), Weight: new(int64(44000)),
			FreightChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(4500.00),
				Valid:   true,
			},
			OtherChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(562.50),
				Valid:   true,
			},
			TotalChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(5062.50),
				Valid:   true,
			},
			RatingUnit: 1,
		},
		{
			ID: shp4ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["FRSH"], ServiceTypeID: svcTypes["TEMP"], ShipmentTypeID: shpTypes["REEF"],
			FormulaTemplateID: flatRateTemplate.ID, Status: shipment.StatusNew,
			ProNumber: "SEED-SHP-004", BOL: "BOL-2026-0004",
			Pieces: new(int64(20)), Weight: new(int64(38000)),
			TemperatureMin: new(int16(34)), TemperatureMax: new(int16(38)),
			FreightChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(4100.00),
				Valid:   true,
			},
			TotalChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(4100.00),
				Valid:   true,
			},
			RatingUnit: 1,
		},
		{
			ID: shp5ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["ACME"], ServiceTypeID: svcTypes["STD"], ShipmentTypeID: shpTypes["FTL"],
			FormulaTemplateID: flatRateTemplate.ID, Status: shipment.StatusCompleted,
			ProNumber: "SEED-SHP-005", BOL: "BOL-2026-0005",
			Pieces: new(int64(26)), Weight: new(int64(40000)),
			ActualShipDate: new(now - 5*day), ActualDeliveryDate: new(now - 3*day),
			FreightChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(2800.00),
				Valid:   true,
			},
			OtherChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(150.00),
				Valid:   true,
			},
			TotalChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(2950.00),
				Valid:   true,
			},
			RatingUnit: 1,
		},
		{
			ID: shp6ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["GLBL"], ServiceTypeID: svcTypes["TEAM"], ShipmentTypeID: shpTypes["FTL"],
			FormulaTemplateID: perMileTemplate.ID, Status: shipment.StatusNew,
			ProNumber: "SEED-SHP-006", BOL: "BOL-2026-0006",
			Pieces: new(int64(40)), Weight: new(int64(43000)),
			FreightChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(6200.00),
				Valid:   true,
			},
			OtherChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(775.00),
				Valid:   true,
			},
			TotalChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(6975.00),
				Valid:   true,
			},
			RatingUnit: 1,
		},
		{
			ID: shp7ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["RNGE"], ServiceTypeID: svcTypes["HAZM"], ShipmentTypeID: shpTypes["HAZM"],
			FormulaTemplateID: flatRateTemplate.ID, Status: shipment.StatusNew,
			ProNumber: "SEED-SHP-007", BOL: "BOL-2026-0007",
			Pieces: new(int64(10)), Weight: new(int64(22000)),
			FreightChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(3800.00),
				Valid:   true,
			},
			OtherChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(350.00),
				Valid:   true,
			},
			TotalChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(4150.00),
				Valid:   true,
			},
			RatingUnit: 1,
		},
		{
			ID: shp8ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["SUNB"], ServiceTypeID: svcTypes["WHG"], ShipmentTypeID: shpTypes["PART"],
			FormulaTemplateID: perStopTemplate.ID, Status: shipment.StatusInTransit,
			ProNumber: "SEED-SHP-008", BOL: "BOL-2026-0008",
			Pieces: new(int64(14)), Weight: new(int64(18000)),
			ActualShipDate: new(now - 1*day),
			FreightChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(2400.00),
				Valid:   true,
			},
			TotalChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(2400.00),
				Valid:   true,
			},
			RatingUnit: 1,
		},
		{
			ID: shp9ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["PEAK"], ServiceTypeID: svcTypes["DTP"], ShipmentTypeID: shpTypes["FTL"],
			FormulaTemplateID: flatRateTemplate.ID, Status: shipment.StatusReadyToInvoice,
			ProNumber: "SEED-SHP-009", BOL: "BOL-2026-0009",
			Pieces: new(int64(32)), Weight: new(int64(44000)),
			ActualShipDate: new(now - 7*day), ActualDeliveryDate: new(now - 4*day),
			FreightChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(3500.00),
				Valid:   true,
			},
			OtherChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(200.00),
				Valid:   true,
			},
			TotalChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(3700.00),
				Valid:   true,
			},
			RatingUnit: 1,
		},
		{
			ID: shp10ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["FRSH"], ServiceTypeID: svcTypes["TEMP"], ShipmentTypeID: shpTypes["REEF"],
			FormulaTemplateID: flatRateTemplate.ID, Status: shipment.StatusDelayed,
			ProNumber: "SEED-SHP-010", BOL: "BOL-2026-0010",
			Pieces: new(int64(18)), Weight: new(int64(36000)),
			TemperatureMin: new(int16(0)), TemperatureMax: new(int16(10)),
			ActualShipDate: new(now - 2*day),
			FreightChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(4800.00),
				Valid:   true,
			},
			OtherChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(125.00),
				Valid:   true,
			},
			TotalChargeAmount: decimal.NullDecimal{
				Decimal: decimal.NewFromFloat(4925.00),
				Valid:   true,
			},
			RatingUnit: 1,
		},
	}

	_, err = tx.NewInsert().Model(&allShipments).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert shipments: %w", err)
	}
	for i := range allShipments {
		if err = sc.TrackCreated(ctx, "shipments", allShipments[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track shipment: %w", err)
		}
	}

	// -- Moves --

	allMoves := []shipment.ShipmentMove{
		// 1: single move
		{
			ID:             shp1M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp1ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         true,
			Sequence:       0,
		},
		// 2: single move, multi-stop
		{
			ID:             shp2M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp2ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         true,
			Sequence:       0,
		},
		// 3: two moves
		{
			ID:             shp3M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp3ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         true,
			Sequence:       0,
		},
		{
			ID:             shp3M2,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp3ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         true,
			Sequence:       1,
		},
		// 4: single move
		{
			ID:             shp4M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp4ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         true,
			Sequence:       0,
		},
		// 5: completed
		{
			ID:             shp5M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp5ID,
			Status:         shipment.MoveStatusCompleted,
			Loaded:         true,
			Sequence:       0,
		},
		// 6: three-move relay
		{
			ID:             shp6M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp6ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         true,
			Sequence:       0,
		},
		{
			ID:             shp6M2,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp6ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         true,
			Sequence:       1,
		},
		{
			ID:             shp6M3,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp6ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         true,
			Sequence:       2,
		},
		// 7: hazmat single move
		{
			ID:             shp7M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp7ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         true,
			Sequence:       0,
		},
		// 8: in-transit, two moves
		{
			ID:             shp8M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp8ID,
			Status:         shipment.MoveStatusCompleted,
			Loaded:         true,
			Sequence:       0,
		},
		{
			ID:             shp8M2,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp8ID,
			Status:         shipment.MoveStatusInTransit,
			Loaded:         true,
			Sequence:       1,
		},
		// 9: ready to invoice, completed
		{
			ID:             shp9M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp9ID,
			Status:         shipment.MoveStatusCompleted,
			Loaded:         true,
			Sequence:       0,
		},
		// 10: delayed, in transit
		{
			ID:             shp10M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp10ID,
			Status:         shipment.MoveStatusInTransit,
			Loaded:         true,
			Sequence:       0,
		},
	}

	_, err = tx.NewInsert().Model(&allMoves).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert shipment moves: %w", err)
	}
	for i := range allMoves {
		if err = sc.TrackCreated(ctx, "shipment_moves", allMoves[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track shipment move: %w", err)
		}
	}

	allStops := []shipment.Stop{
		// SHP 1 — LA → Chicago (1 move, 2 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp1M1,
			LocationID:           locLA,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 1*day,
			ScheduledWindowEnd:   new(now + 1*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp1M1,
			LocationID:           locChicago,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 3*day,
			ScheduledWindowEnd:   new(now + 3*day + 7200),
		},

		// SHP 2 — Dallas → Denver → Chicago (1 move, 3 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp2M1,
			LocationID:           locDallas,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 1*day,
			ScheduledWindowEnd:   new(now + 1*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp2M1,
			LocationID:           locDenver,
			Type:                 shipment.StopTypeSplitDelivery,
			Status:               shipment.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 2*day,
			ScheduledWindowEnd:   new(now + 2*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp2M1,
			LocationID:           locChicago,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusNew,
			Sequence:             2,
			ScheduledWindowStart: now + 3*day,
			ScheduledWindowEnd:   new(now + 3*day + 7200),
		},

		// SHP 3 — LA → Dallas (move 1), Dallas → Chicago (move 2) — 2 moves, 4 stops
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp3M1,
			LocationID:           locLA,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 1*day,
			ScheduledWindowEnd:   new(now + 1*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp3M1,
			LocationID:           locDallas,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 2*day,
			ScheduledWindowEnd:   new(now + 2*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp3M2,
			LocationID:           locDallas,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 3*day,
			ScheduledWindowEnd:   new(now + 3*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp3M2,
			LocationID:           locChicago,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 4*day,
			ScheduledWindowEnd:   new(now + 4*day + 7200),
		},

		// SHP 4 — Phoenix → Denver (1 move, 2 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp4M1,
			LocationID:           locPhoenix,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 1*day,
			ScheduledWindowEnd:   new(now + 1*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp4M1,
			LocationID:           locDenver,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 2*day,
			ScheduledWindowEnd:   new(now + 2*day + 7200),
		},

		// SHP 5 — Dallas → Chicago completed (1 move, 2 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp5M1,
			LocationID:           locDallas,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusCompleted,
			Sequence:             0,
			ScheduledWindowStart: now - 6*day,
			ScheduledWindowEnd:   new(now - 6*day + 7200),
			ActualArrival:        new(now - 5*day),
			ActualDeparture:      new(now - 5*day + 3600),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp5M1,
			LocationID:           locChicago,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusCompleted,
			Sequence:             1,
			ScheduledWindowStart: now - 4*day,
			ScheduledWindowEnd:   new(now - 4*day + 7200),
			ActualArrival:        new(now - 3*day),
			ActualDeparture:      new(now - 3*day + 1800),
		},

		// SHP 6 — LA → Dallas → Chicago → Miami relay (3 moves, 6 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp6M1,
			LocationID:           locLA,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 1*day,
			ScheduledWindowEnd:   new(now + 1*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp6M1,
			LocationID:           locDallas,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 2*day,
			ScheduledWindowEnd:   new(now + 2*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp6M2,
			LocationID:           locDallas,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 3*day,
			ScheduledWindowEnd:   new(now + 3*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp6M2,
			LocationID:           locChicago,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 4*day,
			ScheduledWindowEnd:   new(now + 4*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp6M3,
			LocationID:           locChicago,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 5*day,
			ScheduledWindowEnd:   new(now + 5*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp6M3,
			LocationID:           locMiami,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 7*day,
			ScheduledWindowEnd:   new(now + 7*day + 7200),
		},

		// SHP 7 — Chicago → Denver hazmat (1 move, 2 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp7M1,
			LocationID:           locChicago,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 2*day,
			ScheduledWindowEnd:   new(now + 2*day + 7200),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp7M1,
			LocationID:           locDenver,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 4*day,
			ScheduledWindowEnd:   new(now + 4*day + 7200),
		},

		// SHP 8 — Miami → Dallas (completed) → Dallas → Chicago (in transit) — 2 moves, 4 stops
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp8M1,
			LocationID:           locMiami,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusCompleted,
			Sequence:             0,
			ScheduledWindowStart: now - 2*day,
			ScheduledWindowEnd:   new(now - 2*day + 7200),
			ActualArrival:        new(now - 1*day),
			ActualDeparture:      new(now - 1*day + 2700),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp8M1,
			LocationID:           locDallas,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusCompleted,
			Sequence:             1,
			ScheduledWindowStart: now - 1*day,
			ScheduledWindowEnd:   new(now - 1*day + 7200),
			ActualArrival:        new(now - 12*3600),
			ActualDeparture:      new(now - 11*3600),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp8M2,
			LocationID:           locDallas,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusCompleted,
			Sequence:             0,
			ScheduledWindowStart: now,
			ScheduledWindowEnd:   new(now + 7200),
			ActualArrival:        new(now - 10*3600),
			ActualDeparture:      new(now - 9*3600),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp8M2,
			LocationID:           locChicago,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 1*day,
			ScheduledWindowEnd:   new(now + 1*day + 7200),
		},

		// SHP 9 — Denver → Phoenix ready-to-invoice (1 move, 2 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp9M1,
			LocationID:           locDenver,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusCompleted,
			Sequence:             0,
			ScheduledWindowStart: now - 8*day,
			ScheduledWindowEnd:   new(now - 8*day + 7200),
			ActualArrival:        new(now - 7*day),
			ActualDeparture:      new(now - 7*day + 3600),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp9M1,
			LocationID:           locPhoenix,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusCompleted,
			Sequence:             1,
			ScheduledWindowStart: now - 5*day,
			ScheduledWindowEnd:   new(now - 5*day + 7200),
			ActualArrival:        new(now - 4*day),
			ActualDeparture:      new(now - 4*day + 1800),
		},

		// SHP 10 — Phoenix → LA delayed reefer (1 move, 2 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp10M1,
			LocationID:           locPhoenix,
			Type:                 shipment.StopTypePickup,
			Status:               shipment.StopStatusCompleted,
			Sequence:             0,
			ScheduledWindowStart: now - 3*day,
			ScheduledWindowEnd:   new(now - 3*day + 7200),
			ActualArrival:        new(now - 2*day),
			ActualDeparture:      new(now - 2*day + 3600),
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp10M1,
			LocationID:           locLA,
			Type:                 shipment.StopTypeDelivery,
			Status:               shipment.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now - 1*day,
			ScheduledWindowEnd:   new(now - 1*day + 7200),
		},
	}

	_, err = tx.NewInsert().Model(&allStops).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert stops: %w", err)
	}
	for i := range allStops {
		if err = sc.TrackCreated(ctx, "stops", allStops[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track stop: %w", err)
		}
	}

	allCommodities := []shipment.ShipmentCommodity{
		// SHP 1 — general freight
		{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp1ID,
			CommodityID:    comms["General Freight"],
			Pieces:         24,
			Weight:         42000,
		},
		// SHP 2 — electronics (LTL partial)
		{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp2ID,
			CommodityID:    comms["Electronics"],
			Pieces:         8,
			Weight:         12000,
		},
		// SHP 3 — mixed: general freight + auto parts
		{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp3ID,
			CommodityID:    comms["General Freight"],
			Pieces:         18,
			Weight:         27000,
		},
		{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp3ID,
			CommodityID:    comms["Auto Parts"],
			Pieces:         12,
			Weight:         17000,
		},
		// SHP 4 — frozen produce (reefer)
		{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp4ID,
			CommodityID:    comms["Frozen Produce"],
			Pieces:         20,
			Weight:         38000,
		},
		// SHP 5 — steel coils (completed)
		{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp5ID,
			CommodityID:    comms["Steel Coils"],
			Pieces:         8,
			Weight:         40000,
		},
		// SHP 6 — lumber relay
		{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp6ID,
			CommodityID:    comms["Lumber"],
			Pieces:         40,
			Weight:         43000,
		},
		// SHP 7 — steel coils (hazmat)
		{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp7ID,
			CommodityID:    comms["Steel Coils"],
			Pieces:         10,
			Weight:         22000,
		},
		// SHP 8 — electronics + general freight
		{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp8ID,
			CommodityID:    comms["Electronics"],
			Pieces:         6,
			Weight:         8000,
		},
		{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp8ID,
			CommodityID:    comms["General Freight"],
			Pieces:         8,
			Weight:         10000,
		},
		// SHP 9 — general freight
		{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp9ID,
			CommodityID:    comms["General Freight"],
			Pieces:         32,
			Weight:         44000,
		},
		// SHP 10 — frozen produce (delayed reefer)
		{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp10ID,
			CommodityID:    comms["Frozen Produce"],
			Pieces:         18,
			Weight:         36000,
		},
	}

	_, err = tx.NewInsert().Model(&allCommodities).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert shipment commodities: %w", err)
	}
	for i := range allCommodities {
		if err = sc.TrackCreated(ctx, "shipment_commodities", allCommodities[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track shipment commodity: %w", err)
		}
	}

	// -- Additional Charges --

	fuelID := accCharges["FUEL"]
	detID := accCharges["DET"]
	hazID := accCharges["HAZ"]
	stopID := accCharges["STOP"]
	layID := accCharges["LAY"]
	resdlID := accCharges["RESDL"]

	allCharges := []shipment.AdditionalCharge{
		// SHP 2 — stop-off charge
		{
			ID:                  pulid.MustNew("ac_"),
			BusinessUnitID:      buID,
			OrganizationID:      orgID,
			ShipmentID:          shp2ID,
			AccessorialChargeID: stopID,
			Method:              accessorialcharge.MethodPerUnit,
			Amount:              decimal.NewFromFloat(50.00),
			Unit:                1,
		},
		// SHP 3 — fuel surcharge (12.5% of 4500 = 562.50)
		{
			ID:                  pulid.MustNew("ac_"),
			BusinessUnitID:      buID,
			OrganizationID:      orgID,
			ShipmentID:          shp3ID,
			AccessorialChargeID: fuelID,
			Method:              accessorialcharge.MethodPercentage,
			Amount:              decimal.NewFromFloat(562.50),
			Unit:                1,
		},
		// SHP 5 — detention (2 hrs) + layover
		{
			ID:                  pulid.MustNew("ac_"),
			BusinessUnitID:      buID,
			OrganizationID:      orgID,
			ShipmentID:          shp5ID,
			AccessorialChargeID: detID,
			Method:              accessorialcharge.MethodPerUnit,
			Amount:              decimal.NewFromFloat(75.00),
			Unit:                2,
		},
		// SHP 6 — fuel surcharge + stop-off × 2
		{
			ID:                  pulid.MustNew("ac_"),
			BusinessUnitID:      buID,
			OrganizationID:      orgID,
			ShipmentID:          shp6ID,
			AccessorialChargeID: fuelID,
			Method:              accessorialcharge.MethodPercentage,
			Amount:              decimal.NewFromFloat(675.00),
			Unit:                1,
		},
		{
			ID:                  pulid.MustNew("ac_"),
			BusinessUnitID:      buID,
			OrganizationID:      orgID,
			ShipmentID:          shp6ID,
			AccessorialChargeID: stopID,
			Method:              accessorialcharge.MethodPerUnit,
			Amount:              decimal.NewFromFloat(50.00),
			Unit:                2,
		},
		// SHP 7 — hazmat flat fee
		{
			ID:                  pulid.MustNew("ac_"),
			BusinessUnitID:      buID,
			OrganizationID:      orgID,
			ShipmentID:          shp7ID,
			AccessorialChargeID: hazID,
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromFloat(350.00),
			Unit:                1,
		},
		// SHP 9 — layover + residential delivery
		{
			ID:                  pulid.MustNew("ac_"),
			BusinessUnitID:      buID,
			OrganizationID:      orgID,
			ShipmentID:          shp9ID,
			AccessorialChargeID: layID,
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromFloat(200.00),
			Unit:                1,
		},
		// SHP 10 — residential delivery
		{
			ID:                  pulid.MustNew("ac_"),
			BusinessUnitID:      buID,
			OrganizationID:      orgID,
			ShipmentID:          shp10ID,
			AccessorialChargeID: resdlID,
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromFloat(125.00),
			Unit:                1,
		},
	}

	_, err = tx.NewInsert().Model(&allCharges).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert additional charges: %w", err)
	}
	for i := range allCharges {
		if err = sc.TrackCreated(ctx, "additional_charges", allCharges[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track additional charge: %w", err)
		}
	}

	seedhelpers.LogSuccess(
		"Created shipment fixtures",
		fmt.Sprintf(
			"- Created %d shipments, %d moves, %d stops, %d commodities, %d additional charges",
			len(allShipments),
			len(allMoves),
			len(allStops),
			len(allCommodities),
			len(allCharges),
		),
	)

	return nil
}

// -- Lookup helpers --

func (s *ShipmentSeed) getLocations(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	var locs []location.Location
	err := tx.NewSelect().
		Model(&locs).
		Column("id", "code").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get locations: %w", err)
	}

	m := make(map[string]pulid.ID, len(locs))
	for i := range locs {
		m[locs[i].Code] = locs[i].ID
	}
	return m, nil
}

func (s *ShipmentSeed) getCustomers(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	var custs []customer.Customer
	err := tx.NewSelect().
		Model(&custs).
		Column("id", "code").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get customers: %w", err)
	}

	m := make(map[string]pulid.ID, len(custs))
	for i := range custs {
		m[custs[i].Code] = custs[i].ID
	}
	return m, nil
}

func (s *ShipmentSeed) getServiceTypes(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	var sts []servicetype.ServiceType
	err := tx.NewSelect().
		Model(&sts).
		Column("id", "code").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get service types: %w", err)
	}

	m := make(map[string]pulid.ID, len(sts))
	for i := range sts {
		m[sts[i].Code] = sts[i].ID
	}
	return m, nil
}

func (s *ShipmentSeed) getShipmentTypes(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	var sts []shipmenttype.ShipmentType
	err := tx.NewSelect().
		Model(&sts).
		Column("id", "code").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get shipment types: %w", err)
	}

	m := make(map[string]pulid.ID, len(sts))
	for i := range sts {
		m[sts[i].Code] = sts[i].ID
	}
	return m, nil
}

func (s *ShipmentSeed) getCommodities(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	var comms []commodity.Commodity
	err := tx.NewSelect().
		Model(&comms).
		Column("id", "name").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get commodities: %w", err)
	}

	m := make(map[string]pulid.ID, len(comms))
	for i := range comms {
		m[comms[i].Name] = comms[i].ID
	}
	return m, nil
}

func (s *ShipmentSeed) getAccessorialCharges(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	var charges []accessorialcharge.AccessorialCharge
	err := tx.NewSelect().
		Model(&charges).
		Column("id", "code").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get accessorial charges: %w", err)
	}

	m := make(map[string]pulid.ID, len(charges))
	for i := range charges {
		m[charges[i].Code] = charges[i].ID
	}
	return m, nil
}

func (s *ShipmentSeed) Down(ctx context.Context, tx bun.Tx) error {
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

func (s *ShipmentSeed) CanRollback() bool {
	return true
}
