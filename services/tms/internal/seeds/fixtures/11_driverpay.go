package fixtures

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer/equipmentmanufacturerdb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype/equipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payadvance"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	payWorkerJohn   = "john.smith@example.com"
	payWorkerJane   = "jane.doe@example.com"
	payWorkerMike   = "mike.j@example.com"
	payWorkerSarah  = "sarah.w@example.com"
	payWorkerRobert = "robert.b@example.com"
	payWorkerCarlos = "carlos.rivera@example.com"
	payWorkerEmily  = "emily.chen@example.com"
	payWorkerDavid  = "david.park@example.com"

	payProfileCompanyMile   = "Company Driver Per-Mile"
	payProfileOwnerOperator = "Owner Operator 75% Revenue"
	payProfileRegionalGuar  = "Regional Guaranteed Mixed"
)

var payMinorFactor = decimal.NewFromInt(100)

type DriverPaySeed struct {
}

const driverPaySeedProPrefix = "SEED-PAY-"

var DriverPay = Seeds.Add(seed.Seed{
	Name:        "driver_pay",
	Kind:        seed.Fixture,
	Description: "Creates pay profiles, completed shipments with assigned moves, pay events, and settlements for development",
	After:       []*seed.Seed{Worker, TestData, Shipment},
	Run:         (&DriverPaySeed{}).Run,
})

type payRefs struct {
	orgID         pulid.ID
	buID          pulid.ID
	adminID       pulid.ID
	locations     map[string]pulid.ID
	customers     map[string]pulid.ID
	serviceTypes  map[string]pulid.ID
	shipmentTypes map[string]pulid.ID
	tractors      map[string]pulid.ID
	trailers      map[string]pulid.ID
	commodities   map[string]pulid.ID
	flatTemplate  *formulatemplate.FormulaTemplate
}

type paidMoveDef struct {
	workerKey string
	miles     float64
	stops     []string
	tractor   string
	trailer   string
}

type paidShipmentDef struct {
	pro           string
	bol           string
	customer      string
	revenue       float64
	daysBeforeEnd int64
	pieces        int64
	weight        int64
	moves         []paidMoveDef
}

type paidMove struct {
	moveID     pulid.ID
	shipmentID pulid.ID
	pro        string
	workerKey  string
	miles      decimal.Decimal
	extraStops int
	revenue    decimal.Decimal
	eventDate  int64
}

func (s *DriverPaySeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	admin, err := userByUsername(ctx, db, "admin")
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}

	refs, err := s.loadRefs(ctx, db, org.ID, org.BusinessUnitID, admin.ID)
	if err != nil {
		return err
	}

	workers, err := s.ensureWorkers(ctx, db, sr, refs)
	if err != nil {
		return fmt.Errorf("ensure workers: %w", err)
	}

	if err = s.ensureOwnerTractors(ctx, db, sr, refs, workers); err != nil {
		return fmt.Errorf("ensure owner tractors: %w", err)
	}

	if err = s.ensurePayProfiles(ctx, db, sr, refs); err != nil {
		return fmt.Errorf("ensure pay profiles: %w", err)
	}

	if err = s.ensurePayAssignments(ctx, db, sr, refs, workers); err != nil {
		return fmt.Errorf("ensure pay assignments: %w", err)
	}

	escrows, err := s.ensureEscrowAccounts(ctx, db, sr, refs, workers)
	if err != nil {
		return fmt.Errorf("ensure escrow accounts: %w", err)
	}

	payCodes, err := ensurePayCodes(ctx, db, refs.orgID, refs.buID)
	if err != nil {
		return fmt.Errorf("ensure pay codes: %w", err)
	}

	deductions, err := s.ensureRecurringDeductions(ctx, db, sr, refs, workers, escrows, payCodes)
	if err != nil {
		return fmt.Errorf("ensure recurring deductions: %w", err)
	}

	if err = s.ensurePayAdvances(ctx, db, sr, refs, workers); err != nil {
		return fmt.Errorf("ensure pay advances: %w", err)
	}

	moves, err := s.createPaidShipments(ctx, db, sr, refs, workers)
	if err != nil {
		return fmt.Errorf("create paid shipments: %w", err)
	}

	backfilled, err := backfillSeededRatingDetails(ctx, db, seededRatingBackfill{
		orgID:     refs.orgID,
		buID:      refs.buID,
		proPrefix: driverPaySeedProPrefix,
		now:       timeutils.NowUnix(),
	})
	if err != nil {
		return fmt.Errorf("backfill paid shipment rating details: %w", err)
	}
	if backfilled > 0 {
	}

	if len(moves) > 0 {
		if err = s.createSettlementData(ctx, db, sr, refs, workers, escrows, deductions, moves); err != nil {
			return fmt.Errorf("create settlement data: %w", err)
		}
	}

	return nil
}

// -- Reference data --

func (s *DriverPaySeed) loadRefs(
	ctx context.Context,
	db *keel.DB,
	orgID, buID, adminID pulid.ID,
) (*payRefs, error) {
	refs := &payRefs{
		orgID:         orgID,
		buID:          buID,
		adminID:       adminID,
		locations:     make(map[string]pulid.ID),
		customers:     make(map[string]pulid.ID),
		serviceTypes:  make(map[string]pulid.ID),
		shipmentTypes: make(map[string]pulid.ID),
		tractors:      make(map[string]pulid.ID),
		trailers:      make(map[string]pulid.ID),
		commodities:   make(map[string]pulid.ID),
	}

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
	for i := range locs {
		refs.locations[locs[i].Code] = locs[i].ID
	}

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
	for i := range custs {
		refs.customers[custs[i].Code] = custs[i].ID
	}

	svcTypes, err := servicetypedb.ServiceTypes.AllTenants().
		Pick(servicetypedb.ServiceTypes.ID, servicetypedb.ServiceTypes.Code).
		Where(
			servicetypedb.ServiceTypes.OrganizationID.Eq(orgID),
			servicetypedb.ServiceTypes.BusinessUnitID.Eq(buID),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get service types: %w", err)
	}
	for i := range svcTypes {
		refs.serviceTypes[svcTypes[i].Code] = svcTypes[i].ID
	}

	shpTypes, err := shipmenttypedb.ShipmentTypes.AllTenants().
		Pick(shipmenttypedb.ShipmentTypes.ID, shipmenttypedb.ShipmentTypes.Code).
		Where(
			shipmenttypedb.ShipmentTypes.OrganizationID.Eq(orgID),
			shipmenttypedb.ShipmentTypes.BusinessUnitID.Eq(buID),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get shipment types: %w", err)
	}
	for i := range shpTypes {
		refs.shipmentTypes[shpTypes[i].Code] = shpTypes[i].ID
	}

	tractors, err := tractordb.Tractors.AllTenants().
		Pick(tractordb.Tractors.ID, tractordb.Tractors.Code).
		Where(
			tractordb.Tractors.OrganizationID.Eq(orgID),
			tractordb.Tractors.BusinessUnitID.Eq(buID),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get tractors: %w", err)
	}
	for i := range tractors {
		refs.tractors[tractors[i].Code] = tractors[i].ID
	}

	trailers, err := trailerdb.Trailers.AllTenants().
		Pick(trailerdb.Trailers.ID, trailerdb.Trailers.Code).
		Where(
			trailerdb.Trailers.OrganizationID.Eq(orgID),
			trailerdb.Trailers.BusinessUnitID.Eq(buID),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get trailers: %w", err)
	}
	for i := range trailers {
		refs.trailers[trailers[i].Code] = trailers[i].ID
	}

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
	for i := range comms {
		refs.commodities[comms[i].Name] = comms[i].ID
	}

	if refs.flatTemplate, err = formulatemplatedb.FormulaTemplates.AllTenants().Pick(formulatemplatedb.FormulaTemplates.ID, formulatemplatedb.FormulaTemplates.Name).Where(
		formulatemplatedb.FormulaTemplates.OrganizationID.Eq(orgID),
		formulatemplatedb.FormulaTemplates.BusinessUnitID.Eq(buID),
		formulatemplatedb.FormulaTemplates.Name.Eq("Flat Rate"),
	).First(ctx, db); err != nil {
		return nil, fmt.Errorf("get flat rate formula template: %w", err)
	}

	return refs, nil
}

// -- Workers --

type payWorkerDef struct {
	firstName  string
	lastName   string
	gender     worker.Gender
	workerType worker.WorkerType
	driverType worker.DriverType
	city       string
	address    string
	postalCode string
	stateAbbr  string
	email      string
	phone      string
	license    string
}

func (s *DriverPaySeed) ensureWorkers(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *payRefs,
) (map[string]*worker.Worker, error) {
	existing, err := workerdb.Workers.AllTenants().Where(
		workerdb.Workers.OrganizationID.Eq(refs.orgID),
		workerdb.Workers.BusinessUnitID.Eq(refs.buID),
	).All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get existing workers: %w", err)
	}

	workers := make(map[string]*worker.Worker, len(existing)+3)
	for i := range existing {
		workers[existing[i].Email] = existing[i]
	}

	defs := []payWorkerDef{
		{
			"Carlos", "Rivera", worker.GenderMale, worker.WorkerTypeContractor,
			worker.DriverTypeOTR, "San Antonio", "918 Mission Trail", "78210", "TX",
			payWorkerCarlos, "+15550106001", "TX44112233",
		},
		{
			"Emily", "Chen", worker.GenderFemale, worker.WorkerTypeEmployee,
			worker.DriverTypeRegional, "Sacramento", "2210 Riverfront Way", "95814", "CA",
			payWorkerEmily, "+15550107001", "CA77665544",
		},
		{
			"David", "Park", worker.GenderMale, worker.WorkerTypeContractor,
			worker.DriverTypeRegional, "Aurora", "77 Fox Valley Dr", "60505", "IL",
			payWorkerDavid, "+15550108001", "IL99001122",
		},
	}

	now := timeutils.NowUnix()
	day := int64(86400)
	created := 0

	for _, def := range defs {
		if _, ok := workers[def.email]; ok {
			continue
		}

		state, err := stateByAbbreviation(ctx, db, def.stateAbbr)
		if err != nil {
			return nil, fmt.Errorf("get state %s: %w", def.stateAbbr, err)
		}

		w := &worker.Worker{
			ID:                   pulid.MustNew("wrk_"),
			BusinessUnitID:       refs.buID,
			OrganizationID:       refs.orgID,
			StateID:              state.ID,
			Status:               domaintypes.StatusActive,
			Type:                 def.workerType,
			DriverType:           def.driverType,
			FirstName:            def.firstName,
			LastName:             def.lastName,
			AddressLine1:         def.address,
			City:                 def.city,
			PostalCode:           def.postalCode,
			Email:                def.email,
			PhoneNumber:          def.phone,
			Gender:               def.gender,
			CanBeAssigned:        true,
			AvailableForDispatch: true,
		}
		if err = workerdb.Workers.Create(ctx, db, w); err != nil {
			return nil, fmt.Errorf("insert worker %s %s: %w", def.firstName, def.lastName, err)
		}

		profile := &worker.WorkerProfile{
			ID:                 pulid.MustNew("wrkp_"),
			WorkerID:           w.ID,
			BusinessUnitID:     refs.buID,
			OrganizationID:     refs.orgID,
			LicenseStateID:     state.ID,
			DOB:                now - (35 * 365 * day),
			LicenseNumber:      def.license,
			CDLClass:           worker.CDLClassA,
			Endorsement:        worker.EndorsementTypeNone,
			LicenseExpiry:      now + (3 * 365 * day),
			HireDate:           now - (3 * 365 * day),
			ComplianceStatus:   worker.ComplianceStatusCompliant,
			IsQualified:        true,
			TrainingHealth:     worker.TrainingHealthCurrent,
			SafetyRating:       worker.SafetyRatingExcellent,
			SafetyScore:        100,
			DrugAlcoholStatus:  worker.DrugAlcoholUnknown,
			ReturnToDutyStatus: worker.ReturnToDutyNotRequired,
		}
		if err = workerdb.WorkerProfiles.Create(ctx, db, profile); err != nil {
			return nil, fmt.Errorf(
				"insert worker profile for %s %s: %w",
				def.firstName,
				def.lastName,
				err,
			)
		}

		workers[def.email] = w
		created++
	}

	if created > 0 {
	}

	return workers, nil
}

// -- Owner-operator tractors --

func (s *DriverPaySeed) ensureOwnerTractors(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *payRefs,
	workers map[string]*worker.Worker,
) error {
	equipType, err := equipmenttypedb.EquipmentTypes.AllTenants().
		Pick(equipmenttypedb.EquipmentTypes.ID).
		Where(
			equipmenttypedb.EquipmentTypes.OrganizationID.Eq(refs.orgID),
			equipmenttypedb.EquipmentTypes.BusinessUnitID.Eq(refs.buID),
			equipmenttypedb.EquipmentTypes.Code.Eq("TRACTOR"),
		).
		First(ctx, db)
	if err != nil {
		return fmt.Errorf("get tractor equipment type: %w", err)
	}

	manufacturer, err := equipmentmanufacturerdb.EquipmentManufacturers.AllTenants().
		Pick(equipmentmanufacturerdb.EquipmentManufacturers.ID).
		Where(
			equipmentmanufacturerdb.EquipmentManufacturers.OrganizationID.Eq(refs.orgID),
			equipmentmanufacturerdb.EquipmentManufacturers.BusinessUnitID.Eq(refs.buID),
		).
		First(ctx, db)
	if err != nil {
		return fmt.Errorf("get equipment manufacturer: %w", err)
	}

	defs := []struct {
		code      string
		workerKey string
		model     string
		make      string
		year      int
		vin       string
	}{
		{"TRC-101", payWorkerMike, "T680", "Kenworth", 2022, "1XKYD49X1NJ405112"},
		{"TRC-102", payWorkerCarlos, "579", "Peterbilt", 2021, "1XPBD49X2MD405783"},
		{"TRC-103", payWorkerDavid, "Cascadia", "Freightliner", 2020, "3AKJHHDR8LSKX9021"},
	}

	created := 0
	for _, def := range defs {
		if _, ok := refs.tractors[def.code]; ok {
			continue
		}
		owner, ok := workers[def.workerKey]
		if !ok {
			return fmt.Errorf("owner worker %s not found", def.workerKey)
		}

		year := def.year
		ownerID := owner.ID
		trac := &tractor.Tractor{
			ID:                      pulid.MustNew("trac_"),
			BusinessUnitID:          refs.buID,
			OrganizationID:          refs.orgID,
			PrimaryWorkerID:         owner.ID,
			EquipmentTypeID:         equipType.ID,
			EquipmentManufacturerID: manufacturer.ID,
			StateID:                 owner.StateID,
			Status:                  domaintypes.EquipmentStatusAvailable,
			Code:                    def.code,
			Model:                   def.model,
			Make:                    def.make,
			Year:                    &year,
			Vin:                     def.vin,
			OwnershipType:           domaintypes.OwnershipTypeOwnerOperator,
			OwnerWorkerID:           &ownerID,
			FuelType:                domaintypes.IFTAFuelTypeDiesel,
		}
		if err := tractordb.Tractors.Create(ctx, db, trac); err != nil {
			return fmt.Errorf("insert tractor %s: %w", def.code, err)
		}
		refs.tractors[def.code] = trac.ID
		created++
	}

	if created > 0 {
	}

	return nil
}

// -- Pay profiles --

func (s *DriverPaySeed) ensurePayProfiles(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *payRefs,
) error {
	profiles := []*driverpay.PayProfile{
		{
			ID:             pulid.MustNew("dpp_"),
			BusinessUnitID: refs.buID,
			OrganizationID: refs.orgID,
			Status:         domaintypes.StatusActive,
			Name:           payProfileCompanyMile,
			Description:    "Banded per-loaded-mile pay with stop pay and detention for company drivers",
			Classification: driverpay.PayeeClassificationCompanyDriver,
			CurrencyCode:   "USD",
			Components: []*driverpay.PayProfileComponent{
				{
					Kind:        driverpay.ComponentKindLinehaul,
					Method:      driverpay.CalcMethodPerLoadedMile,
					Description: "Loaded miles (banded)",
					Bands: []driverpay.MileageBand{
						{MinMiles: 0, MaxMiles: 200, Rate: decimal.NewFromFloat(0.65)},
						{MinMiles: 200, MaxMiles: 500, Rate: decimal.NewFromFloat(0.60)},
						{MinMiles: 500, MaxMiles: 0, Rate: decimal.NewFromFloat(0.56)},
					},
					Sequence: 0,
					IsActive: true,
				},
				{
					Kind:        driverpay.ComponentKindStopPay,
					Method:      driverpay.CalcMethodPerStop,
					Description: "Extra stops beyond pickup and delivery",
					Rate:        decimal.NewFromFloat(25.00),
					Sequence:    1,
					IsActive:    true,
				},
				{
					Kind:            driverpay.ComponentKindDetention,
					Method:          driverpay.CalcMethodPerHour,
					Description:     "Detention after 2 hours free time",
					Rate:            decimal.NewFromFloat(22.00),
					FreeTimeMinutes: 120,
					Sequence:        2,
					IsActive:        true,
				},
			},
		},
		{
			ID:             pulid.MustNew("dpp_"),
			BusinessUnitID: refs.buID,
			OrganizationID: refs.orgID,
			Status:         domaintypes.StatusActive,
			Name:           payProfileOwnerOperator,
			Description:    "75% of total revenue with stop pay for leased owner-operators",
			Classification: driverpay.PayeeClassificationOwnerOperator,
			CurrencyCode:   "USD",
			Components: []*driverpay.PayProfileComponent{
				{
					Kind:         driverpay.ComponentKindLinehaul,
					Method:       driverpay.CalcMethodPercentOfRevenue,
					Description:  "75% of total revenue",
					Rate:         decimal.NewFromInt(75),
					RevenueBasis: driverpay.RevenueBasisTotalRevenue,
					Sequence:     0,
					IsActive:     true,
				},
				{
					Kind:        driverpay.ComponentKindStopPay,
					Method:      driverpay.CalcMethodPerStop,
					Description: "Extra stops beyond pickup and delivery",
					Rate:        decimal.NewFromFloat(30.00),
					Sequence:    1,
					IsActive:    true,
				},
			},
		},
		{
			ID:                           pulid.MustNew("dpp_"),
			BusinessUnitID:               refs.buID,
			OrganizationID:               refs.orgID,
			Status:                       domaintypes.StatusActive,
			Name:                         payProfileRegionalGuar,
			Description:                  "Flat per-total-mile with a $900 guaranteed weekly minimum for regional drivers",
			Classification:               driverpay.PayeeClassificationCompanyDriver,
			CurrencyCode:                 "USD",
			GuaranteedPeriodMinimumMinor: 90000,
			Components: []*driverpay.PayProfileComponent{
				{
					Kind:        driverpay.ComponentKindLinehaul,
					Method:      driverpay.CalcMethodPerTotalMile,
					Description: "All dispatched miles",
					Rate:        decimal.NewFromFloat(0.52),
					Sequence:    0,
					IsActive:    true,
				},
				{
					Kind:        driverpay.ComponentKindStopPay,
					Method:      driverpay.CalcMethodPerStop,
					Description: "Extra stops beyond pickup and delivery",
					Rate:        decimal.NewFromFloat(20.00),
					Sequence:    1,
					IsActive:    true,
				},
				{
					Kind:        driverpay.ComponentKindLayover,
					Method:      driverpay.CalcMethodPerDay,
					Description: "Layover per 24 hours away from home terminal",
					Rate:        decimal.NewFromFloat(75.00),
					Sequence:    2,
					IsActive:    true,
				},
			},
		},
	}

	created := 0
	for _, profile := range profiles {

		components := profile.Components
		profile.Components = nil
		if err := driverpaydb.PayProfiles.Create(ctx, db, profile); err != nil {
			return fmt.Errorf("insert pay profile %s: %w", profile.Name, err)
		}

		for _, comp := range components {
			comp.ID = pulid.MustNew("dppc_")
			comp.BusinessUnitID = refs.buID
			comp.OrganizationID = refs.orgID
			comp.PayProfileID = profile.ID
		}
		if err := driverpaydb.PayProfileComponents.CreateMany(ctx, db, components); err != nil {
			return fmt.Errorf("insert pay profile components for %s: %w", profile.Name, err)
		}
		created++
	}

	if created > 0 {
	}

	return nil
}

// -- Pay assignments --

func (s *DriverPaySeed) ensurePayAssignments(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *payRefs,
	workers map[string]*worker.Worker,
) error {
	profileByWorker := map[string]string{
		payWorkerJohn:   payProfileCompanyMile,
		payWorkerJane:   payProfileCompanyMile,
		payWorkerSarah:  payProfileCompanyMile,
		payWorkerMike:   payProfileOwnerOperator,
		payWorkerCarlos: payProfileOwnerOperator,
		payWorkerDavid:  payProfileOwnerOperator,
		payWorkerRobert: payProfileRegionalGuar,
		payWorkerEmily:  payProfileRegionalGuar,
	}

	profiles, err := driverpaydb.PayProfiles.AllTenants().
		Pick(driverpaydb.PayProfiles.ID, driverpaydb.PayProfiles.Name).
		Where(
			driverpaydb.PayProfiles.OrganizationID.Eq(refs.orgID),
			driverpaydb.PayProfiles.BusinessUnitID.Eq(refs.buID),
		).
		All(ctx, db)
	if err != nil {
		return fmt.Errorf("get pay profiles: %w", err)
	}
	profileIDs := make(map[string]pulid.ID, len(profiles))
	for i := range profiles {
		profileIDs[profiles[i].Name] = profiles[i].ID
	}

	assigned, err := driverpaydb.WorkerPayAssignments.AllTenants().
		Pick(driverpaydb.WorkerPayAssignments.WorkerID).
		Where(
			driverpaydb.WorkerPayAssignments.OrganizationID.Eq(refs.orgID),
			driverpaydb.WorkerPayAssignments.BusinessUnitID.Eq(refs.buID),
			driverpaydb.WorkerPayAssignments.EffectiveTo.IsNull(),
		).
		All(ctx, db)
	if err != nil {
		return fmt.Errorf("get existing pay assignments: %w", err)
	}
	assignedWorkers := make(map[pulid.ID]struct{}, len(assigned))
	for i := range assigned {
		assignedWorkers[assigned[i].WorkerID] = struct{}{}
	}

	now := timeutils.NowUnix()
	created := 0

	for workerKey, profileName := range profileByWorker {
		w, ok := workers[workerKey]
		if !ok {
			continue
		}
		if _, ok = assignedWorkers[w.ID]; ok {
			continue
		}
		profileID, ok := profileIDs[profileName]
		if !ok {
			return fmt.Errorf("pay profile %s not found", profileName)
		}

		assignment := &payassignment.WorkerPayAssignment{
			ID:             pulid.MustNew("wpa_"),
			BusinessUnitID: refs.buID,
			OrganizationID: refs.orgID,
			WorkerID:       w.ID,
			PayProfileID:   profileID,
			EffectiveFrom:  now - (180 * 86400),
			SplitPercent:   decimal.NewFromInt(100),
			CreatedByID:    refs.adminID,
		}
		if err := driverpaydb.WorkerPayAssignments.Create(ctx, db, assignment); err != nil {
			return fmt.Errorf("insert pay assignment for %s: %w", workerKey, err)
		}
		created++
	}

	if created > 0 {
	}

	return nil
}

// -- Escrow accounts --

func (s *DriverPaySeed) ensureEscrowAccounts(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *payRefs,
	workers map[string]*worker.Worker,
) (map[string]*payescrow.EscrowAccount, error) {
	defs := []struct {
		workerKey    string
		targetMinor  int64
		contribMinor int64
		contribCount int
	}{
		{payWorkerMike, 250000, 10000, 4},
		{payWorkerCarlos, 300000, 15000, 3},
	}

	now := timeutils.NowUnix()
	week := int64(7 * 86400)
	accounts := make(map[string]*payescrow.EscrowAccount, len(defs))
	created := 0

	for _, def := range defs {
		w, ok := workers[def.workerKey]
		if !ok {
			continue
		}

		existing, err := driverpaydb.EscrowAccounts.AllTenants().Where(
			driverpaydb.EscrowAccounts.OrganizationID.Eq(refs.orgID),
			driverpaydb.EscrowAccounts.BusinessUnitID.Eq(refs.buID),
			driverpaydb.EscrowAccounts.WorkerID.Eq(w.ID),
		).First(ctx, db)
		if err == nil {
			accounts[def.workerKey] = existing
			continue
		}
		if !errors.Is(err, keel.ErrNotFound) {
			return nil, fmt.Errorf("check escrow account for %s: %w", def.workerKey, err)
		}

		account := &payescrow.EscrowAccount{
			ID:                 pulid.MustNew("escr_"),
			BusinessUnitID:     refs.buID,
			OrganizationID:     refs.orgID,
			WorkerID:           w.ID,
			Status:             payescrow.EscrowAccountStatusActive,
			TargetAmountMinor:  def.targetMinor,
			BalanceMinor:       def.contribMinor * int64(def.contribCount),
			AnnualInterestRate: decimal.NewFromFloat(2.5),
			OpenedDate:         now - (120 * 86400),
			CurrencyCode:       "USD",
		}
		if err = driverpaydb.EscrowAccounts.Create(ctx, db, account); err != nil {
			return nil, fmt.Errorf("insert escrow account for %s: %w", def.workerKey, err)
		}

		txs := make([]*payescrow.EscrowTransaction, 0, def.contribCount)
		for i := range def.contribCount {
			txs = append(txs, &payescrow.EscrowTransaction{
				ID:                pulid.MustNew("esctx_"),
				BusinessUnitID:    refs.buID,
				OrganizationID:    refs.orgID,
				EscrowAccountID:   account.ID,
				Type:              payescrow.EscrowTransactionTypeContribution,
				AmountMinor:       def.contribMinor,
				BalanceAfterMinor: def.contribMinor * int64(i+1),
				OccurredDate:      now - (int64(def.contribCount-i) * week),
				Description:       "Weekly settlement escrow contribution",
				CreatedByID:       refs.adminID,
			})
		}
		if err = driverpaydb.EscrowTransactions.CreateMany(ctx, db, txs); err != nil {
			return nil, fmt.Errorf("insert escrow transactions for %s: %w", def.workerKey, err)
		}

		accounts[def.workerKey] = account
		created++
	}

	if created > 0 {
	}

	return accounts, nil
}

// -- Pay codes --

func ensurePayCodes(
	ctx context.Context,
	db *keel.DB,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	defs := paycode.SystemPayCodes()
	rows := make([]*paycode.PayCode, 0, len(defs))
	for _, def := range defs {
		rows = append(rows, &paycode.PayCode{
			ID:                    pulid.MustNew("payc_"),
			BusinessUnitID:        buID,
			OrganizationID:        orgID,
			Status:                domaintypes.StatusActive,
			Direction:             def.Direction,
			Code:                  def.Code,
			Name:                  def.Name,
			Taxable:               def.Taxable,
			CountsTowardGuarantee: true,
			IsSystem:              true,
		})
	}
	pc := driverpaydb.PayCodes
	if _, err := pc.CreateOrSkip(
		ctx, db, rows, pc.OrganizationID, pc.BusinessUnitID, pc.Direction, pc.Code,
	); err != nil {
		return nil, fmt.Errorf("insert system pay codes: %w", err)
	}

	existing, err := driverpaydb.PayCodes.AllTenants().
		Pick(driverpaydb.PayCodes.ID, driverpaydb.PayCodes.Direction, driverpaydb.PayCodes.Code).
		Where(
			driverpaydb.PayCodes.OrganizationID.Eq(orgID),
			driverpaydb.PayCodes.BusinessUnitID.Eq(buID),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get pay codes: %w", err)
	}

	codes := make(map[string]pulid.ID, len(existing))
	for i := range existing {
		codes[existing[i].Direction.String()+"/"+existing[i].Code] = existing[i].ID
	}
	return codes, nil
}

// -- Recurring deductions --

func (s *DriverPaySeed) ensureRecurringDeductions(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *payRefs,
	workers map[string]*worker.Worker,
	escrows map[string]*payescrow.EscrowAccount,
	payCodes map[string]pulid.ID,
) (map[string]*recurringpay.RecurringDeduction, error) {
	count, err := driverpaydb.RecurringDeductions.AllTenants().Where(
		driverpaydb.RecurringDeductions.OrganizationID.Eq(refs.orgID),
		driverpaydb.RecurringDeductions.BusinessUnitID.Eq(refs.buID),
	).Count(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("check existing recurring deductions: %w", err)
	}
	if count > 0 {
		return make(map[string]*recurringpay.RecurringDeduction), nil
	}

	now := timeutils.NowUnix()
	day := int64(86400)

	type deductionDef struct {
		workerKey     string
		code          string
		status        recurringpay.DeductionStatus
		frequency     recurringpay.DeductionFrequency
		description   string
		amountMinor   int64
		capMinor      int64
		deductedMinor int64
		escrow        bool
	}

	defs := []deductionDef{
		{
			payWorkerMike,
			"TRKLEASE",
			recurringpay.DeductionStatusActive,
			recurringpay.DeductionFrequencyEverySettlement,
			"Kenworth T680 lease-purchase payment",
			65000,
			0,
			585000,
			false,
		},
		{
			payWorkerMike,
			"INSUR",
			recurringpay.DeductionStatusActive,
			recurringpay.DeductionFrequencyEverySettlement,
			"Bobtail and physical damage insurance",
			8600,
			0,
			77400,
			false,
		},
		{
			payWorkerMike,
			"ESCROW",
			recurringpay.DeductionStatusActive,
			recurringpay.DeductionFrequencyEverySettlement,
			"Maintenance escrow contribution",
			10000,
			0,
			40000,
			true,
		},
		{
			payWorkerMike,
			"ELD",
			recurringpay.DeductionStatusActive,
			recurringpay.DeductionFrequencyMonthly,
			"ELD subscription",
			4500,
			0,
			13500,
			false,
		},
		{
			payWorkerCarlos,
			"TRKLEASE",
			recurringpay.DeductionStatusActive,
			recurringpay.DeductionFrequencyEverySettlement,
			"Peterbilt 579 lease-purchase payment",
			72500,
			5800000,
			2175000,
			false,
		},
		{
			payWorkerCarlos,
			"INSUR",
			recurringpay.DeductionStatusActive,
			recurringpay.DeductionFrequencyEverySettlement,
			"Occupational accident insurance",
			9200,
			0,
			276000,
			false,
		},
		{
			payWorkerCarlos,
			"ESCROW",
			recurringpay.DeductionStatusActive,
			recurringpay.DeductionFrequencyEverySettlement,
			"Maintenance escrow contribution",
			15000,
			0,
			45000,
			true,
		},
		{
			payWorkerCarlos,
			"FUELCARD",
			recurringpay.DeductionStatusPaused,
			recurringpay.DeductionFrequencyEverySettlement,
			"Fuel card balance repayment",
			31000,
			0,
			124000,
			false,
		},
		{
			payWorkerDavid,
			"TRLLEASE",
			recurringpay.DeductionStatusActive,
			recurringpay.DeductionFrequencyEverySettlement,
			"Dry van trailer rental",
			24000,
			0,
			96000,
			false,
		},
		{
			payWorkerDavid,
			"INSUR",
			recurringpay.DeductionStatusActive,
			recurringpay.DeductionFrequencyEverySettlement,
			"Occupational accident insurance",
			8800,
			0,
			35200,
			false,
		},
		{
			payWorkerDavid,
			"LOAN",
			recurringpay.DeductionStatusActive,
			recurringpay.DeductionFrequencyEverySettlement,
			"Company loan repayment — tire replacement",
			20000,
			480000,
			140000,
			false,
		},
	}

	deductions := make(map[string]*recurringpay.RecurringDeduction, len(defs))
	rows := make([]*recurringpay.RecurringDeduction, 0, len(defs))

	for _, def := range defs {
		w, ok := workers[def.workerKey]
		if !ok {
			continue
		}

		payCodeID, hasCode := payCodes["Deduction/"+def.code]
		if !hasCode {
			return nil, fmt.Errorf("pay code %s not found", def.code)
		}

		deduction := &recurringpay.RecurringDeduction{
			ID:                  pulid.MustNew("rded_"),
			BusinessUnitID:      refs.buID,
			OrganizationID:      refs.orgID,
			WorkerID:            w.ID,
			PayCodeID:           payCodeID,
			Status:              def.status,
			Frequency:           def.frequency,
			Description:         def.description,
			AmountMinor:         def.amountMinor,
			DeductedToDateMinor: def.deductedMinor,
			StartDate:           now - (270 * day),
			CurrencyCode:        "USD",
			CreatedByID:         refs.adminID,
			Kind:                recurringpay.DeductionKindStandard,
			Priority:            100,
		}
		if def.capMinor > 0 {
			capMinor := def.capMinor
			deduction.TotalCapMinor = &capMinor
		}
		if def.escrow {
			account, hasAccount := escrows[def.workerKey]
			if !hasAccount {
				return nil, fmt.Errorf("escrow account for %s not found", def.workerKey)
			}
			deduction.EscrowAccountID = &account.ID
		}

		rows = append(rows, deduction)
		deductions[def.workerKey+"/"+def.code] = deduction
	}

	if err = driverpaydb.RecurringDeductions.CreateMany(ctx, db, rows); err != nil {
		return nil, fmt.Errorf("insert recurring deductions: %w", err)
	}

	return deductions, nil
}

// -- Pay advances --

func (s *DriverPaySeed) ensurePayAdvances(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *payRefs,
	workers map[string]*worker.Worker,
) error {
	now := timeutils.NowUnix()
	day := int64(86400)

	defs := []struct {
		workerKey      string
		status         payadvance.AdvanceStatus
		source         payadvance.AdvanceSource
		reference      string
		issuedDaysAgo  int64
		amountMinor    int64
		recoveredMinor int64
		notes          string
	}{
		{
			payWorkerJohn, payadvance.AdvanceStatusOutstanding, payadvance.AdvanceSourceEFSMoneyCode,
			"EFS-448211", 4, 20000, 0, "Fuel stop advance issued en route",
		},
		{
			payWorkerCarlos, payadvance.AdvanceStatusPartiallyRecovered, payadvance.AdvanceSourceCash,
			"", 45, 100000, 40000, "Cash advance for emergency trailer tire replacement",
		},
		{
			payWorkerJane, payadvance.AdvanceStatusRecovered, payadvance.AdvanceSourceComdataCode,
			"CMD-90321", 60, 15000, 15000, "",
		},
	}

	rows := make([]*payadvance.PayAdvance, 0, len(defs))
	for _, def := range defs {
		w, ok := workers[def.workerKey]
		if !ok {
			continue
		}
		rows = append(rows, &payadvance.PayAdvance{
			ID:             pulid.MustNew("padv_"),
			BusinessUnitID: refs.buID,
			OrganizationID: refs.orgID,
			WorkerID:       w.ID,
			Status:         def.status,
			Source:         def.source,
			Reference:      def.reference,
			IssuedDate:     now - (def.issuedDaysAgo * day),
			AmountMinor:    def.amountMinor,
			RecoveredMinor: def.recoveredMinor,
			Notes:          def.notes,
			CurrencyCode:   "USD",
			CreatedByID:    refs.adminID,
		})
	}

	if err := driverpaydb.PayAdvances.CreateMany(ctx, db, rows); err != nil {
		return fmt.Errorf("insert pay advances: %w", err)
	}

	return nil
}

// -- Completed shipments with assigned moves --

func (s *DriverPaySeed) createPaidShipments(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *payRefs,
	workers map[string]*worker.Worker,
) ([]paidMove, error) {
	exists, err := shipmentdb.Shipments.AllTenants().Where(
		shipmentdb.Shipments.OrganizationID.Eq(refs.orgID),
		shipmentdb.Shipments.BusinessUnitID.Eq(refs.buID),
		shipmentdb.Shipments.ProNumber.Eq("SEED-PAY-001"),
	).Exists(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("check existing pay shipments: %w", err)
	}
	if exists {
		return nil, nil
	}

	bounds, err := s.resolvePeriod(ctx, db, refs)
	if err != nil {
		return nil, err
	}

	defs := []paidShipmentDef{
		{
			pro: "SEED-PAY-001", bol: "BOL-2026-0101", customer: "ACME",
			revenue: 1450, daysBeforeEnd: 6, pieces: 22, weight: 38000,
			moves: []paidMoveDef{
				{payWorkerJane, 372, []string{"TERM-LA", "COLD-PHX"}, "TRC-003", "TRL-001"},
			},
		},
		{
			pro: "SEED-PAY-002", bol: "BOL-2026-0102", customer: "FRSH",
			revenue: 2300, daysBeforeEnd: 6, pieces: 20, weight: 36000,
			moves: []paidMoveDef{
				{payWorkerSarah, 602, []string{"COLD-PHX", "CUST-DEN"}, "TRC-002", "TRL-002"},
			},
		},
		{
			pro: "SEED-PAY-003", bol: "BOL-2026-0103", customer: "PEAK",
			revenue: 2950, daysBeforeEnd: 5, pieces: 26, weight: 41000,
			moves: []paidMoveDef{
				{
					payWorkerMike,
					925,
					[]string{"WH-DAL", "CUST-DEN", "DC-CHI"},
					"TRC-101",
					"TRL-003",
				},
			},
		},
		{
			pro: "SEED-PAY-004", bol: "BOL-2026-0104", customer: "GLBL",
			revenue: 3400, daysBeforeEnd: 5, pieces: 30, weight: 43000,
			moves: []paidMoveDef{
				{payWorkerCarlos, 1003, []string{"DC-CHI", "CUST-DEN"}, "TRC-102", "TRL-004"},
			},
		},
		{
			pro: "SEED-PAY-005", bol: "BOL-2026-0105", customer: "ACME",
			revenue: 5200, daysBeforeEnd: 4, pieces: 34, weight: 44000,
			moves: []paidMoveDef{
				{payWorkerJohn, 1435, []string{"TERM-LA", "WH-DAL"}, "TRC-004", "TRL-005"},
				{payWorkerRobert, 925, []string{"WH-DAL", "DC-CHI"}, "TRC-001", "TRL-005"},
			},
		},
		{
			pro: "SEED-PAY-006", bol: "BOL-2026-0106", customer: "SUNB",
			revenue: 3150, daysBeforeEnd: 4, pieces: 18, weight: 34000,
			moves: []paidMoveDef{
				{payWorkerDavid, 1120, []string{"MAINT-01", "WH-DAL"}, "TRC-103", "TRL-002"},
			},
		},
		{
			pro: "SEED-PAY-007", bol: "BOL-2026-0107", customer: "RNGE",
			revenue: 3050, daysBeforeEnd: 3, pieces: 24, weight: 39000,
			moves: []paidMoveDef{
				{
					payWorkerEmily,
					1016,
					[]string{"CUST-DEN", "COLD-PHX", "TERM-LA"},
					"TRC-003",
					"TRL-003",
				},
			},
		},
		{
			pro: "SEED-PAY-008", bol: "BOL-2026-0108", customer: "GLBL",
			revenue: 4100, daysBeforeEnd: 3, pieces: 28, weight: 42000,
			moves: []paidMoveDef{
				{payWorkerMike, 1380, []string{"DC-CHI", "MAINT-01"}, "TRC-101", "TRL-004"},
			},
		},
		{
			pro: "SEED-PAY-009", bol: "BOL-2026-0109", customer: "PEAK",
			revenue: 2600, daysBeforeEnd: 2, pieces: 20, weight: 37000,
			moves: []paidMoveDef{
				{payWorkerCarlos, 887, []string{"WH-DAL", "COLD-PHX"}, "TRC-102", "TRL-001"},
			},
		},
		{
			pro: "SEED-PAY-010", bol: "BOL-2026-0110", customer: "FRSH",
			revenue: 5750, daysBeforeEnd: 1, pieces: 32, weight: 43000,
			moves: []paidMoveDef{
				{payWorkerSarah, 887, []string{"COLD-PHX", "WH-DAL"}, "TRC-002", "TRL-005"},
				{payWorkerJane, 1120, []string{"WH-DAL", "MAINT-01"}, "TRC-003", "TRL-005"},
			},
		},
	}

	generalFreightID, ok := refs.commodities["General Freight"]
	if !ok {
		return nil, errors.New("general freight commodity not found")
	}
	stdServiceID, ok := refs.serviceTypes["STD"]
	if !ok {
		return nil, errors.New("STD service type not found")
	}
	ftlTypeID, ok := refs.shipmentTypes["FTL"]
	if !ok {
		return nil, errors.New("FTL shipment type not found")
	}

	day := int64(86400)
	hour := int64(3600)
	result := make([]paidMove, 0, 12)

	for _, def := range defs {
		customerID, hasCustomer := refs.customers[def.customer]
		if !hasCustomer {
			return nil, fmt.Errorf("customer %s not found", def.customer)
		}

		deliveredAt := bounds.PeriodEnd - (def.daysBeforeEnd * day) + (14 * hour)
		shippedAt := deliveredAt - (2 * day)
		revenue := decimal.NewFromFloat(def.revenue)

		shp := &shipment.Shipment{
			ID:                 pulid.MustNew("shp_"),
			BusinessUnitID:     refs.buID,
			OrganizationID:     refs.orgID,
			CustomerID:         customerID,
			ServiceTypeID:      stdServiceID,
			ShipmentTypeID:     ftlTypeID,
			FormulaTemplateID:  refs.flatTemplate.ID,
			Status:             shipment.StatusCompleted,
			ProNumber:          def.pro,
			BOL:                def.bol,
			Pieces:             new(def.pieces),
			Weight:             new(def.weight),
			ActualShipDate:     new(shippedAt),
			ActualDeliveryDate: new(deliveredAt),
			TariffInputs: seededTariffInputs(
				formulatemplate.StandardFlatRate,
				revenue,
			),
			FreightChargeAmount: decimal.NullDecimal{
				Decimal: revenue,
				Valid:   true,
			},
			TotalChargeAmount: decimal.NullDecimal{Decimal: revenue, Valid: true},
			EntryMethod:       shipment.EntryMethodManual,
			FreightTerms:      shipment.FreightTermsPrepaid,
			FreightLines:      []shipment.FreightLine{},
			OtherChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString("0")),
		}
		shp.FreightLines = seededFreightLines(shp)
		shp.RatingDetail = seededRatingDetail(shp, refs.flatTemplate, timeutils.NowUnix())
		if err = shipmentdb.Shipments.Create(ctx, db, shp); err != nil {
			return nil, fmt.Errorf("insert shipment %s: %w", def.pro, err)
		}

		shipCommodity := &shipment.ShipmentCommodity{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: refs.buID,
			OrganizationID: refs.orgID,
			ShipmentID:     shp.ID,
			CommodityID:    generalFreightID,
			Pieces:         def.pieces,
			Weight:         def.weight,
		}
		if err = shipmentdb.ShipmentCommodities.Create(ctx, db, shipCommodity); err != nil {
			return nil, fmt.Errorf("insert shipment commodity for %s: %w", def.pro, err)
		}

		moveWindow := (deliveredAt - shippedAt) / int64(len(def.moves))
		for moveIdx, moveDef := range def.moves {
			w, hasWorker := workers[moveDef.workerKey]
			if !hasWorker {
				return nil, fmt.Errorf("worker %s not found", moveDef.workerKey)
			}

			moveEnd := shippedAt + (int64(moveIdx+1) * moveWindow)
			miles := moveDef.miles
			move := &shipment.ShipmentMove{
				ID:             pulid.MustNew("sm_"),
				BusinessUnitID: refs.buID,
				OrganizationID: refs.orgID,
				ShipmentID:     shp.ID,
				Status:         shipment.MoveStatusCompleted,
				Loaded:         true,
				Sequence:       int64(moveIdx),
				Distance:       &miles,
				DistanceUnits:  "miles",
				CoverageType:   shipment.MoveCoverageTypeDriver,
			}
			if err = shipmentdb.ShipmentMoves.Create(ctx, db, move); err != nil {
				return nil, fmt.Errorf("insert move for %s: %w", def.pro, err)
			}

			if err = s.createMoveStops(ctx, db, sr, refs, move.ID, moveDef.stops, moveEnd); err != nil {
				return nil, fmt.Errorf("create stops for %s: %w", def.pro, err)
			}

			tractorID, hasTractor := refs.tractors[moveDef.tractor]
			if !hasTractor {
				return nil, fmt.Errorf("tractor %s not found", moveDef.tractor)
			}
			trailerID, hasTrailer := refs.trailers[moveDef.trailer]
			if !hasTrailer {
				return nil, fmt.Errorf("trailer %s not found", moveDef.trailer)
			}

			workerID := w.ID
			assignment := &shipment.Assignment{
				ID:              pulid.MustNew("asn_"),
				BusinessUnitID:  refs.buID,
				OrganizationID:  refs.orgID,
				ShipmentMoveID:  move.ID,
				PrimaryWorkerID: &workerID,
				TractorID:       &tractorID,
				TrailerID:       &trailerID,
				Status:          shipment.AssignmentStatusCompleted,
				AckStatus:       shipment.AssignmentAckPending,
			}
			if err = shipmentdb.Assignments.Create(ctx, db, assignment); err != nil {
				return nil, fmt.Errorf("insert assignment for %s: %w", def.pro, err)
			}

			result = append(result, paidMove{
				moveID:     move.ID,
				shipmentID: shp.ID,
				pro:        def.pro,
				workerKey:  moveDef.workerKey,
				miles:      decimal.NewFromFloat(moveDef.miles),
				extraStops: max(len(moveDef.stops)-2, 0),
				revenue:    revenue,
				eventDate:  moveEnd,
			})
		}
	}

	return result, nil
}

func (s *DriverPaySeed) createMoveStops(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *payRefs,
	moveID pulid.ID,
	stopCodes []string,
	moveEnd int64,
) error {
	hour := int64(3600)
	stops := make([]*shipment.Stop, 0, len(stopCodes))

	for idx, code := range stopCodes {
		locationID, ok := refs.locations[code]
		if !ok {
			return fmt.Errorf("location %s not found", code)
		}

		stopType := shipment.StopTypeSplitDelivery
		switch idx {
		case 0:
			stopType = shipment.StopTypePickup
		case len(stopCodes) - 1:
			stopType = shipment.StopTypeDelivery
		}

		arrival := moveEnd - (int64(len(stopCodes)-1-idx) * 6 * hour)
		departure := arrival + hour
		scheduledStart := arrival - (2 * hour)
		scheduledEnd := arrival + (2 * hour)

		stops = append(stops, &shipment.Stop{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       refs.buID,
			OrganizationID:       refs.orgID,
			ShipmentMoveID:       moveID,
			LocationID:           locationID,
			Type:                 stopType,
			Status:               shipment.StopStatusCompleted,
			Sequence:             int64(idx),
			ScheduledWindowStart: scheduledStart,
			ScheduledWindowEnd:   &scheduledEnd,
			ActualArrival:        &arrival,
			ActualDeparture:      &departure,
			ScheduleType:         shipment.StopScheduleTypeOpen,
		})
	}

	if err := shipmentdb.Stops.CreateMany(ctx, db, stops); err != nil {
		return fmt.Errorf("insert stops: %w", err)
	}

	return nil
}

// -- Pay events and settlements --

func (s *DriverPaySeed) resolvePeriod(
	ctx context.Context,
	db *keel.DB,
	refs *payRefs,
) (driversettlementservice.PeriodBounds, error) {
	control, err := tenantdb.SettlementControls.AllTenants().Where(
		tenantdb.SettlementControls.OrganizationID.Eq(refs.orgID),
		tenantdb.SettlementControls.BusinessUnitID.Eq(refs.buID),
	).First(ctx, db)
	if err != nil {
		if !errors.Is(err, keel.ErrNotFound) {
			return driversettlementservice.PeriodBounds{}, fmt.Errorf(
				"get settlement control: %w", err,
			)
		}
		control = &settlementcontrol.SettlementControl{
			PayPeriodFrequency:            settlementcontrol.PayPeriodFrequencyWeekly,
			PeriodEndDayOfWeek:            6,
			PayDelayDays:                  5,
			PayTrigger:                    settlementcontrol.PayTriggerShipmentDelivered,
			VarianceThresholdPct:          decimal.RequireFromString("25"),
			VarianceLookbackWeeks:         8,
			EscrowInterestFrequencyMonths: 3,
		}
	}

	return driversettlementservice.ResolveCurrentPeriod(control, timeutils.NowUnix()), nil
}

func (s *DriverPaySeed) loadWorkerProfiles(
	ctx context.Context,
	db *keel.DB,
	refs *payRefs,
	workers map[string]*worker.Worker,
) (map[pulid.ID]*driverpay.PayProfile, error) {
	workerIDs := make([]pulid.ID, 0, len(workers))
	for _, w := range workers {
		workerIDs = append(workerIDs, w.ID)
	}

	now := timeutils.NowUnix()
	wpa := driverpaydb.WorkerPayAssignments
	assignments, err := wpa.AllTenants().Where(
		wpa.OrganizationID.Eq(refs.orgID),
		wpa.BusinessUnitID.Eq(refs.buID),
		wpa.WorkerID.In(workerIDs...),
		wpa.EffectiveFrom.Lte(now),
		keel.Or(wpa.EffectiveTo.IsNull(), wpa.EffectiveTo.Gt(now)),
	).All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get worker pay assignments: %w", err)
	}

	profileIDs := make([]pulid.ID, 0, len(assignments))
	for i := range assignments {
		profileIDs = append(profileIDs, assignments[i].PayProfileID)
	}

	profiles, err := driverpaydb.PayProfiles.AllTenants().
		Load(driverpaydb.PayProfiles.Components()).
		Where(
			driverpaydb.PayProfiles.OrganizationID.Eq(refs.orgID),
			driverpaydb.PayProfiles.BusinessUnitID.Eq(refs.buID),
			driverpaydb.PayProfiles.ID.In(profileIDs...),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get pay profiles with components: %w", err)
	}

	profilesByID := make(map[pulid.ID]*driverpay.PayProfile, len(profiles))
	for _, profile := range profiles {
		profilesByID[profile.ID] = profile
	}

	byWorker := make(map[pulid.ID]*driverpay.PayProfile, len(assignments))
	for i := range assignments {
		if profile, ok := profilesByID[assignments[i].PayProfileID]; ok {
			byWorker[assignments[i].WorkerID] = profile
		}
	}

	return byWorker, nil
}

func payComponentDescription(comp *driverpay.PayProfileComponent) string {
	if comp.Description != "" {
		return comp.Description
	}
	return comp.Kind.String()
}

func buildPayEventComponents(
	profile *driverpay.PayProfile,
	miles decimal.Decimal,
	extraStops int,
	revenue decimal.Decimal,
) (components []driversettlement.PayEventComponent, grossMinor int64) {
	for _, comp := range profile.Components {
		if comp == nil || !comp.IsActive {
			continue
		}

		var quantity, rate, amount decimal.Decimal
		switch comp.Method {
		case driverpay.CalcMethodPerLoadedMile,
			driverpay.CalcMethodPerEmptyMile,
			driverpay.CalcMethodPerTotalMile:
			quantity = miles
			rate = comp.ResolveMileageRate(miles)
			amount = miles.Mul(rate)
		case driverpay.CalcMethodPercentOfRevenue:
			quantity = revenue
			rate = comp.Rate
			amount = revenue.Mul(rate).Div(payMinorFactor)
		case driverpay.CalcMethodPerStop:
			if extraStops == 0 {
				continue
			}
			quantity = decimal.NewFromInt(int64(extraStops))
			rate = comp.Rate
			amount = quantity.Mul(rate)
		default:
			continue
		}

		amountMinor := amount.Mul(payMinorFactor).Round(0).IntPart()
		if amountMinor == 0 {
			continue
		}

		components = append(components, driversettlement.PayEventComponent{
			Kind:        comp.Kind,
			Method:      comp.Method,
			Description: payComponentDescription(comp),
			Quantity:    quantity,
			Rate:        rate,
			AmountMinor: amountMinor,
		})
		grossMinor += amountMinor
	}

	return components, grossMinor
}

type settlementPlan struct {
	workerKey        string
	number           string
	status           driversettlement.Status
	eventIndexes     []int
	deductionKeys    []string
	applyGuarantee   bool
	exceptions       []driversettlement.Exception
	paymentMethod    string
	paymentReference string
}

//nolint:funlen,gocognit,cyclop // sequential fixture assembly reads clearest as one workflow
func (s *DriverPaySeed) createSettlementData(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *payRefs,
	workers map[string]*worker.Worker,
	escrows map[string]*payescrow.EscrowAccount,
	deductions map[string]*recurringpay.RecurringDeduction,
	moves []paidMove,
) error {
	bounds, err := s.resolvePeriod(ctx, db, refs)
	if err != nil {
		return err
	}

	profilesByWorker, err := s.loadWorkerProfiles(ctx, db, refs, workers)
	if err != nil {
		return err
	}

	events := make([]*driversettlement.PayEvent, 0, len(moves))
	for _, move := range moves {
		w := workers[move.workerKey]
		profile, hasProfile := profilesByWorker[w.ID]
		if !hasProfile {
			return fmt.Errorf("no effective pay profile for %s", move.workerKey)
		}

		components, grossMinor := buildPayEventComponents(
			profile,
			move.miles,
			move.extraStops,
			move.revenue,
		)
		moveID := move.moveID
		profileID := profile.ID
		events = append(events, &driversettlement.PayEvent{
			ID:               pulid.MustNew("dpe_"),
			BusinessUnitID:   refs.buID,
			OrganizationID:   refs.orgID,
			WorkerID:         w.ID,
			ShipmentID:       move.shipmentID,
			MoveID:           &moveID,
			PayProfileID:     &profileID,
			IdempotencyKey:   "pay:" + w.ID.String() + ":" + moveID.String(),
			Status:           driversettlement.PayEventStatusAccrued,
			EventDate:        move.eventDate,
			GrossAmountMinor: grossMinor,
			TotalMiles:       move.miles,
			CurrencyCode:     "USD",
			Components:       components,
			ProNumber:        move.pro,
		})
	}

	holdReasons := map[int]string{
		4: "Awaiting signed BOL from the shipper before pay release",
		9: "Linehaul rate dispute under review with the customer",
	}
	for idx, reason := range holdReasons {
		events[idx].OnHold = true
		events[idx].HoldReason = reason
	}

	plans := []settlementPlan{
		{
			workerKey:    payWorkerJane,
			number:       "SEED-STL-1001",
			status:       driversettlement.StatusDraft,
			eventIndexes: []int{0, 11},
		},
		{
			workerKey:    payWorkerSarah,
			number:       "SEED-STL-1002",
			status:       driversettlement.StatusPendingApproval,
			eventIndexes: []int{1, 10},
		},
		{
			workerKey:    payWorkerMike,
			number:       "SEED-STL-1003",
			status:       driversettlement.StatusApproved,
			eventIndexes: []int{2, 8},
			deductionKeys: []string{
				payWorkerMike + "/TRKLEASE",
				payWorkerMike + "/INSUR",
				payWorkerMike + "/ELD",
				payWorkerMike + "/ESCROW",
			},
		},
		{
			workerKey: payWorkerDavid,
			number:    "SEED-STL-1004",
			status:    driversettlement.StatusDraft,
			deductionKeys: []string{
				payWorkerDavid + "/TRLLEASE",
				payWorkerDavid + "/INSUR",
				payWorkerDavid + "/LOAN",
			},
			exceptions: []driversettlement.Exception{
				{
					Code:     driversettlement.ExceptionCodeNegativeNet,
					Severity: driversettlement.ExceptionSeverityCritical,
					Message:  "Deductions exceed earnings; the negative balance will carry forward",
				},
				{
					Code:     driversettlement.ExceptionCodeNoActivity,
					Severity: driversettlement.ExceptionSeverityWarning,
					Message:  "No completed moves during the pay period",
				},
			},
		},
		{
			workerKey:      payWorkerEmily,
			number:         "SEED-STL-1005",
			status:         driversettlement.StatusPosted,
			eventIndexes:   []int{7},
			applyGuarantee: true,
		},
		{
			workerKey:        payWorkerRobert,
			number:           "SEED-STL-1006",
			status:           driversettlement.StatusPaid,
			eventIndexes:     []int{5},
			applyGuarantee:   true,
			paymentMethod:    "ACH",
			paymentReference: "SEED-ACH-2201",
		},
	}

	now := timeutils.NowUnix()
	hour := int64(3600)

	batch, err := driversettlementdb.SettlementBatches.AllTenants().Where(
		driversettlementdb.SettlementBatches.OrganizationID.Eq(refs.orgID),
		driversettlementdb.SettlementBatches.BusinessUnitID.Eq(refs.buID),
		driversettlementdb.SettlementBatches.PeriodStart.Eq(bounds.PeriodStart),
		driversettlementdb.SettlementBatches.PeriodEnd.Eq(bounds.PeriodEnd),
		driversettlementdb.SettlementBatches.Status.Eq(driversettlement.BatchStatusOpen),
	).First(ctx, db)
	if err != nil {
		if !errors.Is(err, keel.ErrNotFound) {
			return fmt.Errorf("check existing settlement batch: %w", err)
		}
		batch = &driversettlement.SettlementBatch{
			ID:             pulid.MustNew("dstlb_"),
			BusinessUnitID: refs.buID,
			OrganizationID: refs.orgID,
			Status:         driversettlement.BatchStatusOpen,
			Name: "Pay period ending " +
				time.Unix(bounds.PeriodEnd, 0).UTC().AddDate(0, 0, -1).Format("Jan 2, 2006"),
			PeriodStart:   bounds.PeriodStart,
			PeriodEnd:     bounds.PeriodEnd,
			PayDate:       bounds.PayDate,
			CurrencyCode:  "USD",
			GeneratedByID: refs.adminID,
			GeneratedAt:   &now,
		}
		if err = driversettlementdb.SettlementBatches.Create(ctx, db, batch); err != nil {
			return fmt.Errorf("insert settlement batch: %w", err)
		}
	}

	settlements := make([]*driversettlement.Settlement, 0, len(plans))
	allLines := make([]*driversettlement.SettlementLine, 0, 32)

	for _, plan := range plans {
		w := workers[plan.workerKey]
		profile := profilesByWorker[w.ID]
		if profile == nil {
			return fmt.Errorf("no effective pay profile for %s", plan.workerKey)
		}

		profileID := profile.ID
		batchID := batch.ID
		entity := &driversettlement.Settlement{
			ID:               pulid.MustNew("dstl_"),
			BusinessUnitID:   refs.buID,
			OrganizationID:   refs.orgID,
			WorkerID:         w.ID,
			BatchID:          &batchID,
			PayProfileID:     &profileID,
			SettlementNumber: plan.number,
			Status:           plan.status,
			Classification:   profile.Classification,
			PayProfileName:   profile.Name,
			PeriodStart:      bounds.PeriodStart,
			PeriodEnd:        bounds.PeriodEnd,
			PayDate:          bounds.PayDate,
			CurrencyCode:     "USD",
		}

		totalMiles := decimal.Zero
		shipmentIDs := make(map[pulid.ID]struct{}, len(plan.eventIndexes))
		var grossMinor int64

		for _, eventIdx := range plan.eventIndexes {
			event := events[eventIdx]
			event.Status = driversettlement.PayEventStatusSettled
			event.SettlementID = &entity.ID
			event.OnHold = false
			event.HoldReason = ""

			eventID := event.ID
			shipmentID := event.ShipmentID
			shipmentIDs[shipmentID] = struct{}{}
			totalMiles = totalMiles.Add(event.TotalMiles)

			for _, comp := range event.Components {
				entity.Lines = append(entity.Lines, &driversettlement.SettlementLine{
					Category:      driversettlement.LineCategoryEarning,
					ComponentKind: comp.Kind,
					Method:        comp.Method,
					Description:   comp.Description,
					Quantity:      comp.Quantity,
					Rate:          comp.Rate,
					AmountMinor:   comp.AmountMinor,
					ShipmentID:    &shipmentID,
					MoveID:        event.MoveID,
					PayEventID:    &eventID,
					ProNumber:     event.ProNumber,
				})
				grossMinor += comp.AmountMinor
			}
		}

		if plan.applyGuarantee && profile.GuaranteedPeriodMinimumMinor > grossMinor {
			entity.Lines = append(entity.Lines, &driversettlement.SettlementLine{
				Category:    driversettlement.LineCategoryGuaranteeTopUp,
				Description: "Guaranteed weekly minimum top-up",
				Quantity:    decimal.NewFromInt(1),
				AmountMinor: profile.GuaranteedPeriodMinimumMinor - grossMinor,
			})
			entity.AddException(
				driversettlement.ExceptionCodeGuaranteeApplied,
				driversettlement.ExceptionSeverityWarning,
				"Earnings fell below the guaranteed period minimum; a top-up was applied",
			)
		}

		for _, deductionKey := range plan.deductionKeys {
			deduction, hasDeduction := deductions[deductionKey]
			if !hasDeduction {
				continue
			}
			category := driversettlement.LineCategoryDeduction
			var escrowAccountID *pulid.ID
			if deduction.IsEscrowContribution() {
				category = driversettlement.LineCategoryEscrowContribution
				escrowAccountID = deduction.EscrowAccountID
			}
			deductionID := deduction.ID
			linePayCodeID := deduction.PayCodeID
			entity.Lines = append(entity.Lines, &driversettlement.SettlementLine{
				Category:             category,
				Description:          deduction.Description,
				Quantity:             decimal.NewFromInt(1),
				Rate:                 decimal.NewFromInt(deduction.AmountMinor).Div(payMinorFactor),
				AmountMinor:          -deduction.AmountMinor,
				RecurringDeductionID: &deductionID,
				EscrowAccountID:      escrowAccountID,
				PayCodeID:            &linePayCodeID,
			})
		}

		for _, exception := range plan.exceptions {
			entity.AddException(exception.Code, exception.Severity, exception.Message)
		}

		entity.TotalMiles = totalMiles
		entity.ShipmentCount = len(shipmentIDs)
		entity.SyncTotals()

		submittedAt := now - (8 * hour)
		approvedAt := now - (6 * hour)
		postedAt := now - (4 * hour)
		paidAt := now - (2 * hour)
		switch plan.status {
		case driversettlement.StatusPaid:
			entity.PaidAt = &paidAt
			entity.PaidByID = refs.adminID
			entity.PaymentMethod = plan.paymentMethod
			entity.PaymentReference = plan.paymentReference
			fallthrough
		case driversettlement.StatusPosted:
			entity.PostedAt = &postedAt
			entity.PostedByID = refs.adminID
			fallthrough
		case driversettlement.StatusApproved:
			entity.ApprovedAt = &approvedAt
			entity.ApprovedByID = refs.adminID
			fallthrough
		case driversettlement.StatusPendingApproval:
			entity.SubmittedAt = &submittedAt
			entity.SubmittedByID = refs.adminID
		case driversettlement.StatusDraft, driversettlement.StatusVoided:
		}

		if err = driversettlementdb.Settlements.Create(ctx, db, entity); err != nil {
			return fmt.Errorf("insert settlement %s: %w", plan.number, err)
		}

		for _, line := range entity.Lines {
			line.ID = pulid.MustNew("dstll_")
			line.BusinessUnitID = refs.buID
			line.OrganizationID = refs.orgID
			line.SettlementID = entity.ID
			allLines = append(allLines, line)
		}

		batch.SettlementCount++
		if entity.HasExceptions {
			batch.ExceptionCount++
		}
		batch.TotalGrossMinor += entity.GrossEarningsMinor
		batch.TotalNetMinor += entity.NetPayMinor
		settlements = append(settlements, entity)
	}

	if err = driversettlementdb.PayEvents.CreateMany(ctx, db, events); err != nil {
		return fmt.Errorf("insert pay events: %w", err)
	}

	if err = driversettlementdb.SettlementLines.CreateMany(ctx, db, allLines); err != nil {
		return fmt.Errorf("insert settlement lines: %w", err)
	}

	if err = driversettlementdb.SettlementBatches.Save(ctx, db, batch); err != nil {
		return fmt.Errorf("update settlement batch totals: %w", err)
	}

	if err = s.applyEscrowSettlementTx(ctx, db, sr, refs, escrows, deductions, settlements); err != nil {
		return err
	}

	return nil
}

func (s *DriverPaySeed) applyEscrowSettlementTx(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *payRefs,
	escrows map[string]*payescrow.EscrowAccount,
	deductions map[string]*recurringpay.RecurringDeduction,
	settlements []*driversettlement.Settlement,
) error {
	account, hasAccount := escrows[payWorkerMike]
	deduction, hasDeduction := deductions[payWorkerMike+"/ESCROW"]
	if !hasAccount || !hasDeduction {
		return nil
	}

	var mikeSettlement *driversettlement.Settlement
	for _, settlement := range settlements {
		if settlement.Status == driversettlement.StatusApproved {
			mikeSettlement = settlement
			break
		}
	}
	if mikeSettlement == nil {
		return nil
	}

	account.BalanceMinor += deduction.AmountMinor
	if err := driverpaydb.EscrowAccounts.Save(ctx, db, account); err != nil {
		return fmt.Errorf("update escrow balance: %w", err)
	}

	settlementID := mikeSettlement.ID
	escTx := &payescrow.EscrowTransaction{
		ID:                pulid.MustNew("esctx_"),
		BusinessUnitID:    refs.buID,
		OrganizationID:    refs.orgID,
		EscrowAccountID:   account.ID,
		Type:              payescrow.EscrowTransactionTypeContribution,
		AmountMinor:       deduction.AmountMinor,
		BalanceAfterMinor: account.BalanceMinor,
		OccurredDate:      timeutils.NowUnix(),
		Description:       "Escrow contribution from settlement " + mikeSettlement.SettlementNumber,
		SettlementID:      &settlementID,
		CreatedByID:       refs.adminID,
	}
	if err := driverpaydb.EscrowTransactions.Create(ctx, db, escTx); err != nil {
		return fmt.Errorf("insert settlement escrow transaction: %w", err)
	}

	deduction.DeductedToDateMinor += deduction.AmountMinor
	if err := driverpaydb.RecurringDeductions.Save(ctx, db, deduction); err != nil {
		return fmt.Errorf("update escrow deduction total: %w", err)
	}

	return nil
}
