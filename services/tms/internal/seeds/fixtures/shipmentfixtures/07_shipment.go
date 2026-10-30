package shipmentfixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentprojection"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/pricingfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

type ShipmentSeed struct {
}

const shipmentSeedProPrefix = "SEED-SHP-"

func (s *ShipmentSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	if err = s.createCustomers(ctx, db, sr, org.ID, org.BusinessUnitID); err != nil {
		return fmt.Errorf("create customers: %w", err)
	}

	if err = s.createCommodities(ctx, db, sr, org.ID, org.BusinessUnitID); err != nil {
		return fmt.Errorf("create commodities: %w", err)
	}

	if err = s.createShipments(ctx, db, sr, org.ID, org.BusinessUnitID); err != nil {
		return fmt.Errorf("create shipments: %w", err)
	}

	backfilled, err := pricingfixtures.BackfillSeededRatingDetails(
		ctx,
		db,
		pricingfixtures.SeededRatingBackfill{
			OrgID:     org.ID,
			BuID:      org.BusinessUnitID,
			ProPrefix: shipmentSeedProPrefix,
			Now:       timeutils.NowUnix(),
		},
	)
	if err != nil {
		return fmt.Errorf("backfill shipment rating details: %w", err)
	}
	if backfilled > 0 {
	}

	return nil
}

// -- Customers --

func (s *ShipmentSeed) createCustomers(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) error {
	ilState, err := tenantfixtures.StateByAbbreviation(ctx, db, "IL")
	if err != nil {
		return fmt.Errorf("get IL state: %w", err)
	}

	caState, err := tenantfixtures.StateByAbbreviation(ctx, db, "CA")
	if err != nil {
		return fmt.Errorf("get CA state: %w", err)
	}

	txState, err := tenantfixtures.StateByAbbreviation(ctx, db, "TX")
	if err != nil {
		return fmt.Errorf("get TX state: %w", err)
	}

	azState, err := tenantfixtures.StateByAbbreviation(ctx, db, "AZ")
	if err != nil {
		return fmt.Errorf("get AZ state: %w", err)
	}

	coState, err := tenantfixtures.StateByAbbreviation(ctx, db, "CO")
	if err != nil {
		return fmt.Errorf("get CO state: %w", err)
	}

	flState, err := tenantfixtures.StateByAbbreviation(ctx, db, "FL")
	if err != nil {
		return fmt.Errorf("get FL state: %w", err)
	}

	customers := []customer.Customer{
		{
			ID:                 pulid.MustNew("cus_"),
			BusinessUnitID:     buID,
			OrganizationID:     orgID,
			StateID:            ilState.ID,
			Status:             domaintypes.StatusActive,
			AllowConsolidation: true,
			Code:               "ACME",
			Name:               "Acme Manufacturing",
			AddressLine1:       "400 W Superior St",
			City:               "Chicago",
			PostalCode:         "60654",

			StatusUpdatePreference: customer.StatusUpdateNone,

			ConsolidationPriority: 1,
		},
		{
			ID:                 pulid.MustNew("cus_"),
			BusinessUnitID:     buID,
			OrganizationID:     orgID,
			StateID:            caState.ID,
			Status:             domaintypes.StatusActive,
			AllowConsolidation: true,
			Code:               "GLBL",
			Name:               "GlobalTrade Imports",
			AddressLine1:       "2500 E Olympic Blvd",
			City:               "Los Angeles",
			PostalCode:         "90023",

			StatusUpdatePreference: customer.StatusUpdateNone,

			ConsolidationPriority: 1,
		},
		{
			ID:                 pulid.MustNew("cus_"),
			BusinessUnitID:     buID,
			OrganizationID:     orgID,
			StateID:            txState.ID,
			Status:             domaintypes.StatusActive,
			AllowConsolidation: true,
			Code:               "PEAK",
			Name:               "Peak Distributing",
			AddressLine1:       "1200 Commerce St",
			City:               "Dallas",
			PostalCode:         "75202",

			StatusUpdatePreference: customer.StatusUpdateNone,

			ConsolidationPriority: 1,
		},
		{
			ID:                 pulid.MustNew("cus_"),
			BusinessUnitID:     buID,
			OrganizationID:     orgID,
			StateID:            azState.ID,
			Status:             domaintypes.StatusActive,
			AllowConsolidation: true,
			Code:               "FRSH",
			Name:               "FreshHaul Foods",
			AddressLine1:       "3300 N Central Ave",
			City:               "Phoenix",
			PostalCode:         "85012",

			StatusUpdatePreference: customer.StatusUpdateNone,

			ConsolidationPriority: 1,
		},
		{
			ID:                 pulid.MustNew("cus_"),
			BusinessUnitID:     buID,
			OrganizationID:     orgID,
			StateID:            coState.ID,
			Status:             domaintypes.StatusActive,
			AllowConsolidation: true,
			Code:               "RNGE",
			Name:               "Range Logistics",
			AddressLine1:       "1801 California St",
			City:               "Denver",
			PostalCode:         "80202",

			StatusUpdatePreference: customer.StatusUpdateNone,

			ConsolidationPriority: 1,
		},
		{
			ID:                 pulid.MustNew("cus_"),
			BusinessUnitID:     buID,
			OrganizationID:     orgID,
			StateID:            flState.ID,
			Status:             domaintypes.StatusActive,
			AllowConsolidation: true,
			Code:               "SUNB",
			Name:               "Sunbelt Materials",
			AddressLine1:       "800 Brickell Ave",
			City:               "Miami",
			PostalCode:         "33131",

			StatusUpdatePreference: customer.StatusUpdateNone,

			ConsolidationPriority: 1,
		},
	}

	err = customerdb.Customers.CreateMany(ctx, db, sliceutils.Pointers(customers))
	if err != nil {
		return fmt.Errorf("insert customers: %w", err)
	}

	billingProfiles := make([]customer.CustomerBillingProfile, len(customers))
	for i := range customers {
		billingProfiles[i] = *customer.NewDefaultBillingProfile(orgID, buID, customers[i].ID)
	}

	err = customerdb.CustomerBillingProfiles.CreateMany(
		ctx,
		db,
		sliceutils.Pointers(billingProfiles),
	)
	if err != nil {
		return fmt.Errorf("insert customer billing profiles: %w", err)
	}

	return nil
}

// -- Commodities --

func (s *ShipmentSeed) createCommodities(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) error {
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

	err := commoditydb.Commodities.CreateMany(ctx, db, sliceutils.Pointers(commodities))
	if err != nil {
		return fmt.Errorf("insert commodities: %w", err)
	}

	return nil
}

func (s *ShipmentSeed) createShipments(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) error {
	locs, err := s.getLocations(ctx, db, orgID, buID)
	if err != nil {
		return err
	}

	custs, err := s.getCustomers(ctx, db, orgID, buID)
	if err != nil {
		return err
	}

	svcTypes, err := s.getServiceTypes(ctx, db, orgID, buID)
	if err != nil {
		return err
	}

	shpTypes, err := s.getShipmentTypes(ctx, db, orgID, buID)
	if err != nil {
		return err
	}

	comms, err := s.getCommodities(ctx, db, orgID, buID)
	if err != nil {
		return err
	}

	accCharges, err := s.getAccessorialCharges(ctx, db, orgID, buID)
	if err != nil {
		return err
	}

	flatRateTemplate, err := formulatemplatedb.FormulaTemplates.AllTenants().Where(
		formulatemplatedb.FormulaTemplates.OrganizationID.Eq(orgID),
		formulatemplatedb.FormulaTemplates.BusinessUnitID.Eq(buID),
		formulatemplatedb.FormulaTemplates.Name.Eq("Flat Rate"),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get flat rate formula template: %w", err)
	}

	perMileTemplate, err := formulatemplatedb.FormulaTemplates.AllTenants().Where(
		formulatemplatedb.FormulaTemplates.OrganizationID.Eq(orgID),
		formulatemplatedb.FormulaTemplates.BusinessUnitID.Eq(buID),
		formulatemplatedb.FormulaTemplates.Name.Eq("Per Mile"),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get per mile formula template: %w", err)
	}

	perStopTemplate, err := formulatemplatedb.FormulaTemplates.AllTenants().Where(
		formulatemplatedb.FormulaTemplates.OrganizationID.Eq(orgID),
		formulatemplatedb.FormulaTemplates.BusinessUnitID.Eq(buID),
		formulatemplatedb.FormulaTemplates.Name.Eq("Per Stop"),
	).First(ctx, db)
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
			FormulaTemplateID: flatRateTemplate.ID, Status: shipmentvocab.StatusNew,
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

			EntryMethod: shipmentvocab.EntryMethodManual,

			FreightTerms: shipmentvocab.FreightTermsPrepaid,

			FreightLines: []shipment.FreightLine{},

			OtherChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString("0")),
		},
		{
			ID: shp2ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["PEAK"], ServiceTypeID: svcTypes["STD"], ShipmentTypeID: shpTypes["LTL"],
			FormulaTemplateID: perMileTemplate.ID, Status: shipmentvocab.StatusNew,
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

			EntryMethod: shipmentvocab.EntryMethodManual,

			FreightTerms: shipmentvocab.FreightTermsPrepaid,

			FreightLines: []shipment.FreightLine{},
		},
		{
			ID: shp3ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["GLBL"], ServiceTypeID: svcTypes["EXP"], ShipmentTypeID: shpTypes["FTL"],
			FormulaTemplateID: perMileTemplate.ID, Status: shipmentvocab.StatusNew,
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

			EntryMethod: shipmentvocab.EntryMethodManual,

			FreightTerms: shipmentvocab.FreightTermsPrepaid,

			FreightLines: []shipment.FreightLine{},
		},
		{
			ID: shp4ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["FRSH"], ServiceTypeID: svcTypes["TEMP"], ShipmentTypeID: shpTypes["REEF"],
			FormulaTemplateID: flatRateTemplate.ID, Status: shipmentvocab.StatusNew,
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

			EntryMethod: shipmentvocab.EntryMethodManual,

			FreightTerms: shipmentvocab.FreightTermsPrepaid,

			FreightLines: []shipment.FreightLine{},

			OtherChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString("0")),
		},
		{
			ID: shp5ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["ACME"], ServiceTypeID: svcTypes["STD"], ShipmentTypeID: shpTypes["FTL"],
			FormulaTemplateID: flatRateTemplate.ID, Status: shipmentvocab.StatusCompleted,
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

			EntryMethod: shipmentvocab.EntryMethodManual,

			FreightTerms: shipmentvocab.FreightTermsPrepaid,

			FreightLines: []shipment.FreightLine{},
		},
		{
			ID: shp6ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["GLBL"], ServiceTypeID: svcTypes["TEAM"], ShipmentTypeID: shpTypes["FTL"],
			FormulaTemplateID: perMileTemplate.ID, Status: shipmentvocab.StatusNew,
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

			EntryMethod: shipmentvocab.EntryMethodManual,

			FreightTerms: shipmentvocab.FreightTermsPrepaid,

			FreightLines: []shipment.FreightLine{},
		},
		{
			ID: shp7ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["RNGE"], ServiceTypeID: svcTypes["HAZM"], ShipmentTypeID: shpTypes["HAZM"],
			FormulaTemplateID: flatRateTemplate.ID, Status: shipmentvocab.StatusNew,
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

			EntryMethod: shipmentvocab.EntryMethodManual,

			FreightTerms: shipmentvocab.FreightTermsPrepaid,

			FreightLines: []shipment.FreightLine{},
		},
		{
			ID: shp8ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["SUNB"], ServiceTypeID: svcTypes["WHG"], ShipmentTypeID: shpTypes["PART"],
			FormulaTemplateID: perStopTemplate.ID, Status: shipmentvocab.StatusInTransit,
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

			EntryMethod: shipmentvocab.EntryMethodManual,

			FreightTerms: shipmentvocab.FreightTermsPrepaid,

			FreightLines: []shipment.FreightLine{},

			OtherChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString("0")),
		},
		{
			ID: shp9ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["PEAK"], ServiceTypeID: svcTypes["DTP"], ShipmentTypeID: shpTypes["FTL"],
			FormulaTemplateID: flatRateTemplate.ID, Status: shipmentvocab.StatusReadyToInvoice,
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

			EntryMethod: shipmentvocab.EntryMethodManual,

			FreightTerms: shipmentvocab.FreightTermsPrepaid,

			FreightLines: []shipment.FreightLine{},
		},
		{
			ID: shp10ID, BusinessUnitID: buID, OrganizationID: orgID,
			CustomerID: custs["FRSH"], ServiceTypeID: svcTypes["TEMP"], ShipmentTypeID: shpTypes["REEF"],
			FormulaTemplateID: flatRateTemplate.ID, Status: shipmentvocab.StatusDelayed,
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

			EntryMethod: shipmentvocab.EntryMethodManual,

			FreightTerms: shipmentvocab.FreightTermsPrepaid,

			FreightLines: []shipment.FreightLine{},
		},
	}

	templatesByID := map[pulid.ID]*formulatemplate.FormulaTemplate{
		flatRateTemplate.ID: flatRateTemplate,
		perMileTemplate.ID:  perMileTemplate,
		perStopTemplate.ID:  perStopTemplate,
	}
	for i := range allShipments {
		template, ok := templatesByID[allShipments[i].FormulaTemplateID]
		if !ok {
			return fmt.Errorf(
				"formula template %s for shipment %s not loaded",
				allShipments[i].FormulaTemplateID,
				allShipments[i].ProNumber,
			)
		}
		allShipments[i].TariffInputs = pricingfixtures.SeededTariffInputs(
			template.Name,
			pricingfixtures.SeededRate(template.Name, allShipments[i].FreightChargeAmount.Decimal),
		)
		allShipments[i].FreightLines = pricingfixtures.SeededFreightLines(&allShipments[i])
		allShipments[i].RatingDetail = pricingfixtures.SeededRatingDetail(
			&allShipments[i],
			template,
			now,
		)
	}

	err = shipmentdb.Shipments.CreateMany(ctx, db, sliceutils.Pointers(allShipments))
	if err != nil {
		return fmt.Errorf("insert shipments: %w", err)
	}

	// -- Moves --

	allMoves := []shipment.ShipmentMove{
		// 1: single move
		{
			ID:             shp1M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp1ID,
			Status:         shipmentvocab.MoveStatusNew,
			Loaded:         true,
			Sequence:       0,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		// 2: single move, multi-stop
		{
			ID:             shp2M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp2ID,
			Status:         shipmentvocab.MoveStatusNew,
			Loaded:         true,
			Sequence:       0,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		// 3: two moves
		{
			ID:             shp3M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp3ID,
			Status:         shipmentvocab.MoveStatusNew,
			Loaded:         true,
			Sequence:       0,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		{
			ID:             shp3M2,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp3ID,
			Status:         shipmentvocab.MoveStatusNew,
			Loaded:         true,
			Sequence:       1,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		// 4: single move
		{
			ID:             shp4M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp4ID,
			Status:         shipmentvocab.MoveStatusNew,
			Loaded:         true,
			Sequence:       0,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		// 5: completed
		{
			ID:             shp5M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp5ID,
			Status:         shipmentvocab.MoveStatusCompleted,
			Loaded:         true,
			Sequence:       0,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		// 6: three-move relay
		{
			ID:             shp6M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp6ID,
			Status:         shipmentvocab.MoveStatusNew,
			Loaded:         true,
			Sequence:       0,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		{
			ID:             shp6M2,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp6ID,
			Status:         shipmentvocab.MoveStatusNew,
			Loaded:         true,
			Sequence:       1,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		{
			ID:             shp6M3,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp6ID,
			Status:         shipmentvocab.MoveStatusNew,
			Loaded:         true,
			Sequence:       2,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		// 7: hazmat single move
		{
			ID:             shp7M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp7ID,
			Status:         shipmentvocab.MoveStatusNew,
			Loaded:         true,
			Sequence:       0,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		// 8: in-transit, two moves
		{
			ID:             shp8M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp8ID,
			Status:         shipmentvocab.MoveStatusCompleted,
			Loaded:         true,
			Sequence:       0,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		{
			ID:             shp8M2,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp8ID,
			Status:         shipmentvocab.MoveStatusInTransit,
			Loaded:         true,
			Sequence:       1,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		// 9: ready to invoice, completed
		{
			ID:             shp9M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp9ID,
			Status:         shipmentvocab.MoveStatusCompleted,
			Loaded:         true,
			Sequence:       0,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
		// 10: delayed, in transit
		{
			ID:             shp10M1,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shp10ID,
			Status:         shipmentvocab.MoveStatusInTransit,
			Loaded:         true,
			Sequence:       0,

			CoverageType: shipmentvocab.MoveCoverageTypeUnassigned,
		},
	}

	err = shipmentdb.ShipmentMoves.CreateMany(ctx, db, sliceutils.Pointers(allMoves))
	if err != nil {
		return fmt.Errorf("insert shipment moves: %w", err)
	}

	allStops := []shipment.Stop{
		// SHP 1 — LA → Chicago (1 move, 2 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp1M1,
			LocationID:           locLA,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 1*day,
			ScheduledWindowEnd:   new(now + 1*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp1M1,
			LocationID:           locChicago,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 3*day,
			ScheduledWindowEnd:   new(now + 3*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},

		// SHP 2 — Dallas → Denver → Chicago (1 move, 3 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp2M1,
			LocationID:           locDallas,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 1*day,
			ScheduledWindowEnd:   new(now + 1*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp2M1,
			LocationID:           locDenver,
			Type:                 shipmentvocab.StopTypeSplitDelivery,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 2*day,
			ScheduledWindowEnd:   new(now + 2*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp2M1,
			LocationID:           locChicago,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             2,
			ScheduledWindowStart: now + 3*day,
			ScheduledWindowEnd:   new(now + 3*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},

		// SHP 3 — LA → Dallas (move 1), Dallas → Chicago (move 2) — 2 moves, 4 stops
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp3M1,
			LocationID:           locLA,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 1*day,
			ScheduledWindowEnd:   new(now + 1*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp3M1,
			LocationID:           locDallas,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 2*day,
			ScheduledWindowEnd:   new(now + 2*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp3M2,
			LocationID:           locDallas,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 3*day,
			ScheduledWindowEnd:   new(now + 3*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp3M2,
			LocationID:           locChicago,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 4*day,
			ScheduledWindowEnd:   new(now + 4*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},

		// SHP 4 — Phoenix → Denver (1 move, 2 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp4M1,
			LocationID:           locPhoenix,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 1*day,
			ScheduledWindowEnd:   new(now + 1*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp4M1,
			LocationID:           locDenver,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 2*day,
			ScheduledWindowEnd:   new(now + 2*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},

		// SHP 5 — Dallas → Chicago completed (1 move, 2 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp5M1,
			LocationID:           locDallas,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusCompleted,
			Sequence:             0,
			ScheduledWindowStart: now - 6*day,
			ScheduledWindowEnd:   new(now - 6*day + 7200),
			ActualArrival:        new(now - 5*day),
			ActualDeparture:      new(now - 5*day + 3600),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp5M1,
			LocationID:           locChicago,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusCompleted,
			Sequence:             1,
			ScheduledWindowStart: now - 4*day,
			ScheduledWindowEnd:   new(now - 4*day + 7200),
			ActualArrival:        new(now - 3*day),
			ActualDeparture:      new(now - 3*day + 1800),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},

		// SHP 6 — LA → Dallas → Chicago → Miami relay (3 moves, 6 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp6M1,
			LocationID:           locLA,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 1*day,
			ScheduledWindowEnd:   new(now + 1*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp6M1,
			LocationID:           locDallas,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 2*day,
			ScheduledWindowEnd:   new(now + 2*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp6M2,
			LocationID:           locDallas,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 3*day,
			ScheduledWindowEnd:   new(now + 3*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp6M2,
			LocationID:           locChicago,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 4*day,
			ScheduledWindowEnd:   new(now + 4*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp6M3,
			LocationID:           locChicago,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 5*day,
			ScheduledWindowEnd:   new(now + 5*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp6M3,
			LocationID:           locMiami,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 7*day,
			ScheduledWindowEnd:   new(now + 7*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},

		// SHP 7 — Chicago → Denver hazmat (1 move, 2 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp7M1,
			LocationID:           locChicago,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             0,
			ScheduledWindowStart: now + 2*day,
			ScheduledWindowEnd:   new(now + 2*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp7M1,
			LocationID:           locDenver,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 4*day,
			ScheduledWindowEnd:   new(now + 4*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},

		// SHP 8 — Miami → Dallas (completed) → Dallas → Chicago (in transit) — 2 moves, 4 stops
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp8M1,
			LocationID:           locMiami,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusCompleted,
			Sequence:             0,
			ScheduledWindowStart: now - 2*day,
			ScheduledWindowEnd:   new(now - 2*day + 7200),
			ActualArrival:        new(now - 1*day),
			ActualDeparture:      new(now - 1*day + 2700),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp8M1,
			LocationID:           locDallas,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusCompleted,
			Sequence:             1,
			ScheduledWindowStart: now - 1*day,
			ScheduledWindowEnd:   new(now - 1*day + 7200),
			ActualArrival:        new(now - 12*3600),
			ActualDeparture:      new(now - 11*3600),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp8M2,
			LocationID:           locDallas,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusCompleted,
			Sequence:             0,
			ScheduledWindowStart: now,
			ScheduledWindowEnd:   new(now + 7200),
			ActualArrival:        new(now - 10*3600),
			ActualDeparture:      new(now - 9*3600),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp8M2,
			LocationID:           locChicago,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now + 1*day,
			ScheduledWindowEnd:   new(now + 1*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},

		// SHP 9 — Denver → Phoenix ready-to-invoice (1 move, 2 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp9M1,
			LocationID:           locDenver,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusCompleted,
			Sequence:             0,
			ScheduledWindowStart: now - 8*day,
			ScheduledWindowEnd:   new(now - 8*day + 7200),
			ActualArrival:        new(now - 7*day),
			ActualDeparture:      new(now - 7*day + 3600),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp9M1,
			LocationID:           locPhoenix,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusCompleted,
			Sequence:             1,
			ScheduledWindowStart: now - 5*day,
			ScheduledWindowEnd:   new(now - 5*day + 7200),
			ActualArrival:        new(now - 4*day),
			ActualDeparture:      new(now - 4*day + 1800),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},

		// SHP 10 — Phoenix → LA delayed reefer (1 move, 2 stops)
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp10M1,
			LocationID:           locPhoenix,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusCompleted,
			Sequence:             0,
			ScheduledWindowStart: now - 3*day,
			ScheduledWindowEnd:   new(now - 3*day + 7200),
			ActualArrival:        new(now - 2*day),
			ActualDeparture:      new(now - 2*day + 3600),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			ShipmentMoveID:       shp10M1,
			LocationID:           locLA,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusNew,
			Sequence:             1,
			ScheduledWindowStart: now - 1*day,
			ScheduledWindowEnd:   new(now - 1*day + 7200),

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
	}

	err = shipmentdb.Stops.CreateMany(ctx, db, sliceutils.Pointers(allStops))
	if err != nil {
		return fmt.Errorf("insert stops: %w", err)
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

	err = shipmentdb.ShipmentCommodities.CreateMany(ctx, db, sliceutils.Pointers(allCommodities))
	if err != nil {
		return fmt.Errorf("insert shipment commodities: %w", err)
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

	err = shipmentdb.AdditionalCharges.CreateMany(ctx, db, sliceutils.Pointers(allCharges))
	if err != nil {
		return fmt.Errorf("insert additional charges: %w", err)
	}

	shipmentIDs := make([]pulid.ID, 0, len(allShipments))
	for i := range allShipments {
		shipmentIDs = append(shipmentIDs, allShipments[i].ID)
	}
	if err = shipmentprojection.RefreshAcrossTenants(ctx, db, shipmentIDs...); err != nil {
		return fmt.Errorf("refresh shipment projections: %w", err)
	}

	return nil
}

// -- Lookup helpers --

func (s *ShipmentSeed) getLocations(
	ctx context.Context,
	db *keel.DB,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	locs, err := locationdb.Locations.AllTenants().
		Pick(locationdb.Locations.ID, locationdb.Locations.Code).
		Where(
			locationdb.Locations.OrganizationID.Eq(orgID),
			locationdb.Locations.BusinessUnitID.Eq(buID),
		).
		All(ctx, db)
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
	db *keel.DB,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	custs, err := customerdb.Customers.AllTenants().
		Pick(customerdb.Customers.ID, customerdb.Customers.Code).
		Where(
			customerdb.Customers.OrganizationID.Eq(orgID),
			customerdb.Customers.BusinessUnitID.Eq(buID),
		).
		All(ctx, db)
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
	db *keel.DB,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	sts, err := servicetypedb.ServiceTypes.AllTenants().
		Pick(servicetypedb.ServiceTypes.ID, servicetypedb.ServiceTypes.Code).
		Where(
			servicetypedb.ServiceTypes.OrganizationID.Eq(orgID),
			servicetypedb.ServiceTypes.BusinessUnitID.Eq(buID),
		).
		All(ctx, db)
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
	db *keel.DB,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	sts, err := shipmenttypedb.ShipmentTypes.AllTenants().
		Pick(shipmenttypedb.ShipmentTypes.ID, shipmenttypedb.ShipmentTypes.Code).
		Where(
			shipmenttypedb.ShipmentTypes.OrganizationID.Eq(orgID),
			shipmenttypedb.ShipmentTypes.BusinessUnitID.Eq(buID),
		).
		All(ctx, db)
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
	db *keel.DB,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	comms, err := commoditydb.Commodities.AllTenants().
		Pick(commoditydb.Commodities.ID, commoditydb.Commodities.Name).
		Where(
			commoditydb.Commodities.OrganizationID.Eq(orgID),
			commoditydb.Commodities.BusinessUnitID.Eq(buID),
		).
		All(ctx, db)
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
	db *keel.DB,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	charges, err := accessorialchargedb.AccessorialCharges.AllTenants().
		Pick(accessorialchargedb.AccessorialCharges.ID, accessorialchargedb.AccessorialCharges.Code).
		Where(
			accessorialchargedb.AccessorialCharges.OrganizationID.Eq(orgID),
			accessorialchargedb.AccessorialCharges.BusinessUnitID.Eq(buID),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get accessorial charges: %w", err)
	}

	m := make(map[string]pulid.ID, len(charges))
	for i := range charges {
		m[charges[i].Code] = charges[i].ID
	}
	return m, nil
}
