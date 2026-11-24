package carrierfixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentprojection"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/pricingfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	CoverLapsingFirstCode = "RDGL"
	CoverLapsingProPrefix = "SEED-CAR-"
	coverLapsingHour      = int64(3600)
	coverLapsingRenewal   = 185
)

// CoverLapsingSeed gives four carriers that haul for us required cover ending in
// 6, 10, 18 and 27 days and open loads on their trucks, one of which delivers
// after its carrier's cover ends, so the carrier risk desk has insurance about to
// lapse against real loads. Bench scenarios re-place the days relative to now.
type CoverLapsingSeed struct{}

type coverLapsingCarrier struct {
	def       carrierDef
	soonest   carrier.InsurancePolicyType
	endsIn    int64
	loadsDays []int64
}

type coverLapsingRefs struct {
	orgID         pulid.ID
	buID          pulid.ID
	customerID    pulid.ID
	pickupID      pulid.ID
	deliveryID    pulid.ID
	serviceTypeID pulid.ID
	shipTypeID    pulid.ID
	commodityID   pulid.ID
	formula       *formulatemplate.FormulaTemplate
	stateID       pulid.ID
	now           int64
	pro           int
}

func coverLapsingCarrierDef(code, name, dot, mc, city, zip string) carrierDef {
	return carrierDef{
		code:              code,
		name:              name,
		carrierType:       carrier.TypeCommon,
		status:            carrier.StatusActive,
		complianceStatus:  carrier.ComplianceStatusQualified,
		safetyRating:      carrier.SafetyRatingSatisfactory,
		dotNumber:         dot,
		mcNumber:          mc,
		addressLine1:      "400 Freight Way",
		city:              city,
		postalCode:        zip,
		stateAbbr:         "TX",
		phone:             "+12145550190",
		email:             "dispatch@" + code + ".example.com",
		paymentMethod:     carrier.PaymentMethodACHManual,
		paymentTermDays:   30,
		remitToName:       name,
		remitAddressLine1: "PO Box 9100",
		remitCity:         city,
		remitPostalCode:   zip,
		remitStateAbbr:    "TX",
		notes:             "Hauls brokered loads on the Dallas and Chicago lanes.",
	}
}

func coverLapsingCarriers() []coverLapsingCarrier {
	return []coverLapsingCarrier{
		{
			def: coverLapsingCarrierDef(CoverLapsingFirstCode, "Ridgeline Freight",
				"3310428", "910442", "Fort Worth", "76106"),
			soonest:   carrier.InsurancePolicyTypeAutoLiability,
			endsIn:    6,
			loadsDays: []int64{1, 2, 4, 8},
		},
		{
			def: coverLapsingCarrierDef("CPHD", "Copperhead Transport",
				"3310511", "910518", "Waco", "76701"),
			soonest:   carrier.InsurancePolicyTypeCargoLiability,
			endsIn:    10,
			loadsDays: []int64{7},
		},
		{
			def: coverLapsingCarrierDef("BLUL", "Blue Line Freight",
				"3310637", "910663", "Arlington", "76010"),
			soonest:   carrier.InsurancePolicyTypeCargoLiability,
			endsIn:    18,
			loadsDays: []int64{1, 3},
		},
		{
			def: coverLapsingCarrierDef("SNDV", "Sandoval Trucking",
				"3310752", "910779", "Laredo", "78040"),
			soonest:   carrier.InsurancePolicyTypeAutoLiability,
			endsIn:    27,
			loadsDays: nil,
		},
	}
}

func CoverLapsingPolicyNumber(code string, policyType carrier.InsurancePolicyType) string {
	suffix := "AL"
	if policyType == carrier.InsurancePolicyTypeCargoLiability {
		suffix = "CL"
	}

	return "SEED-INS-" + code + "-" + suffix
}

func (entry *coverLapsingCarrier) policies() []carrierPolicyDef {
	out := make([]carrierPolicyDef, 0, len(carrier.RequiredInsurancePolicyTypes))
	for _, policyType := range carrier.RequiredInsurancePolicyTypes {
		days := int64(coverLapsingRenewal)
		if policyType == entry.soonest {
			days = entry.endsIn
		}
		provider, amount := "Great West Casualty", "1000000"
		if policyType == carrier.InsurancePolicyTypeCargoLiability {
			provider, amount = "Northland Insurance", "250000"
		}
		out = append(out, carrierPolicyDef{
			policyType:     policyType,
			policyNumber:   CoverLapsingPolicyNumber(entry.def.code, policyType),
			providerName:   provider,
			coverageAmount: amount,
			effectiveDays:  days - 365,
			expirationDays: days,
			isVerified:     true,
		})
	}

	return out
}

func (s *CoverLapsingSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	c := carrierdb.Carriers
	seeded, err := c.AllTenants().Where(
		c.OrganizationID.Eq(org.ID),
		c.BusinessUnitID.Eq(org.BusinessUnitID),
		c.Code.Eq(CoverLapsingFirstCode),
	).Exists(ctx, db)
	if err != nil || seeded {
		return err
	}
	refs, err := loadCoverLapsingRefs(ctx, db, org.ID, org.BusinessUnitID)
	if err != nil {
		return err
	}

	for _, entry := range coverLapsingCarriers() {
		entry.def.policies = entry.policies()
		ident := carrierSeedIdentity{
			orgID:        refs.orgID,
			buID:         refs.buID,
			carrierID:    pulid.MustNew("car_"),
			stateID:      refs.stateID,
			remitStateID: refs.stateID,
			now:          refs.now,
		}
		if err = carrierdb.Carriers.Create(ctx, db, buildCarrier(entry.def, ident)); err != nil {
			return fmt.Errorf("insert carrier %s: %w", entry.def.name, err)
		}
		if err = (&CarrierSeed{}).insertPolicies(ctx, db, sr, entry.def, ident); err != nil {
			return err
		}
		for _, days := range entry.loadsDays {
			if err = createCoverLapsingLoad(ctx, db, refs, ident.carrierID, days); err != nil {
				return fmt.Errorf("insert a load for carrier %s: %w", entry.def.name, err)
			}
		}
	}

	return nil
}

//nolint:funlen // one reference per column a shipment needs, each looked up once
func loadCoverLapsingRefs(
	ctx context.Context,
	db *keel.DB,
	orgID, buID pulid.ID,
) (*coverLapsingRefs, error) {
	refs := &coverLapsingRefs{orgID: orgID, buID: buID, now: timeutils.NowUnix()}

	cust := customerdb.Customers
	customer, err := cust.AllTenants().Pick(cust.ID).Where(
		cust.OrganizationID.Eq(orgID), cust.BusinessUnitID.Eq(buID), cust.Code.Eq("ACME"),
	).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get customer ACME: %w", err)
	}
	refs.customerID = customer.ID

	loc := locationdb.Locations
	for code, dst := range map[string]*pulid.ID{"TERM-LA": &refs.pickupID, "DC-CHI": &refs.deliveryID} {
		found, lErr := loc.AllTenants().Pick(loc.ID).Where(
			loc.OrganizationID.Eq(orgID), loc.BusinessUnitID.Eq(buID), loc.Code.Eq(code),
		).First(ctx, db)
		if lErr != nil {
			return nil, fmt.Errorf("get location %s: %w", code, lErr)
		}
		*dst = found.ID
	}

	svc := servicetypedb.ServiceTypes
	service, err := svc.AllTenants().Pick(svc.ID).Where(
		svc.OrganizationID.Eq(orgID), svc.BusinessUnitID.Eq(buID), svc.Code.Eq("STD"),
	).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get service type STD: %w", err)
	}
	refs.serviceTypeID = service.ID

	st := shipmenttypedb.ShipmentTypes
	shipType, err := st.AllTenants().Pick(st.ID).Where(
		st.OrganizationID.Eq(orgID), st.BusinessUnitID.Eq(buID), st.Code.Eq("FTL"),
	).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get shipment type FTL: %w", err)
	}
	refs.shipTypeID = shipType.ID

	com := commoditydb.Commodities
	freight, err := com.AllTenants().Pick(com.ID).Where(
		com.OrganizationID.Eq(orgID), com.BusinessUnitID.Eq(buID),
		com.Name.Eq("General Freight"),
	).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get general freight commodity: %w", err)
	}
	refs.commodityID = freight.ID

	ft := formulatemplatedb.FormulaTemplates
	if refs.formula, err = ft.AllTenants().Pick(ft.ID, ft.Name).Where(
		ft.OrganizationID.Eq(orgID), ft.BusinessUnitID.Eq(buID), ft.Name.Eq("Flat Rate"),
	).First(ctx, db); err != nil {
		return nil, fmt.Errorf("get flat rate formula template: %w", err)
	}

	state, err := tenantfixtures.StateByAbbreviation(ctx, db, "TX")
	if err != nil {
		return nil, fmt.Errorf("get state TX: %w", err)
	}
	refs.stateID = state.ID

	return refs, nil
}

//nolint:funlen // one open load from header through its move, stops and carrier
func createCoverLapsingLoad(
	ctx context.Context,
	db *keel.DB,
	refs *coverLapsingRefs,
	carrierID pulid.ID,
	deliveryDays int64,
) error {
	refs.pro++
	pro := fmt.Sprintf("%s%03d", CoverLapsingProPrefix, refs.pro)
	revenue := decimal.NewFromInt(2400)
	shp := &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		BusinessUnitID:    refs.buID,
		OrganizationID:    refs.orgID,
		CustomerID:        refs.customerID,
		ServiceTypeID:     refs.serviceTypeID,
		ShipmentTypeID:    refs.shipTypeID,
		FormulaTemplateID: refs.formula.ID,
		Status:            shipmentvocab.StatusAssigned,
		ProNumber:         pro,
		BOL:               fmt.Sprintf("BOL-CAR-%03d", refs.pro),
		Pieces:            new(int64(22)),
		Weight:            new(int64(38000)),
		TariffInputs: pricingfixtures.SeededTariffInputs(
			formulatemplate.StandardFlatRate,
			revenue,
		),
		FreightChargeAmount: decimal.NullDecimal{Decimal: revenue, Valid: true},
		TotalChargeAmount:   decimal.NullDecimal{Decimal: revenue, Valid: true},
		EntryMethod:         shipmentvocab.EntryMethodManual,
		FreightTerms:        shipmentvocab.FreightTermsPrepaid,
		OtherChargeAmount:   decimal.NewNullDecimal(decimal.Zero),
	}
	shp.FreightLines = pricingfixtures.SeededFreightLines(shp)
	shp.RatingDetail = pricingfixtures.SeededRatingDetail(shp, refs.formula, refs.now)
	if err := shipmentdb.Shipments.Create(ctx, db, shp); err != nil {
		return fmt.Errorf("insert shipment %s: %w", pro, err)
	}

	if err := shipmentdb.ShipmentCommodities.Create(ctx, db, &shipment.ShipmentCommodity{
		ID:             pulid.MustNew("sc_"),
		BusinessUnitID: refs.buID,
		OrganizationID: refs.orgID,
		ShipmentID:     shp.ID,
		CommodityID:    refs.commodityID,
		Pieces:         22,
		Weight:         38000,
	}); err != nil {
		return fmt.Errorf("insert shipment commodity: %w", err)
	}

	move := &shipment.ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		BusinessUnitID: refs.buID,
		OrganizationID: refs.orgID,
		ShipmentID:     shp.ID,
		Status:         shipmentvocab.MoveStatusAssigned,
		Loaded:         true,
		CoverageType:   shipmentvocab.MoveCoverageTypeCarrier,
	}
	if err := shipmentdb.ShipmentMoves.Create(ctx, db, move); err != nil {
		return fmt.Errorf("insert shipment move: %w", err)
	}

	deliveryStart := refs.now + deliveryDays*timeutils.SecondsPerDay
	pickupStart := deliveryStart - 30*coverLapsingHour
	stops := []*shipment.Stop{
		coverLapsingStop(
			refs,
			move.ID,
			refs.pickupID,
			shipmentvocab.StopTypePickup,
			0,
			pickupStart,
		),
		coverLapsingStop(
			refs, move.ID, refs.deliveryID, shipmentvocab.StopTypeDelivery, 1, deliveryStart,
		),
	}
	if err := shipmentdb.Stops.CreateMany(ctx, db, stops); err != nil {
		return fmt.Errorf("insert stops: %w", err)
	}

	cost := decimal.NewFromInt(1950)
	confirmed := refs.now
	if err := shipmentdb.CarrierAssignments.Create(ctx, db, &shipment.CarrierAssignment{
		ID:             pulid.MustNew("casn_"),
		BusinessUnitID: refs.buID,
		OrganizationID: refs.orgID,
		ShipmentMoveID: move.ID,
		CarrierID:      carrierID,
		Status:         shipmentvocab.CarrierAssignmentStatusConfirmed,
		RateMethod:     shipmentvocab.CarrierRateMethodFlat,
		BaseRate:       cost,
		BaseAmount:     cost,
		TotalCost:      cost,
		CurrencyCode:   "USD",
		ConfirmedAt:    &confirmed,
	}); err != nil {
		return fmt.Errorf("insert carrier assignment: %w", err)
	}

	return shipmentprojection.Refresh(ctx, db, shp.ID)
}

func coverLapsingStop(
	refs *coverLapsingRefs,
	moveID, locationID pulid.ID,
	stopType shipmentvocab.StopType,
	sequence, windowStart int64,
) *shipment.Stop {
	return &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		BusinessUnitID:       refs.buID,
		OrganizationID:       refs.orgID,
		ShipmentMoveID:       moveID,
		LocationID:           locationID,
		Type:                 stopType,
		ScheduleType:         shipmentvocab.StopScheduleTypeAppointment,
		Handling:             shipmentvocab.StopHandlingLive,
		Status:               shipmentvocab.StopStatusNew,
		Sequence:             sequence,
		ScheduledWindowStart: windowStart,
		ScheduledWindowEnd:   new(windowStart + 2*coverLapsingHour),
	}
}
