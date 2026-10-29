package detentionfixtures

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol/shipmentcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticerender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentioncollectability"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence/detentionevidencedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice/detentionnoticedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy/detentionpolicydb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment/payassignmentdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/driverpayfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/pricingfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	detentionPolicyStandard = "DET-STD"
	detentionPolicyAcme     = "DET-ACME"
	detentionPolicyCold     = "DET-COLD"
	detentionPolicyDray     = "DET-DRAY"
	detentionPolicyLegacy   = "DET-2025"
	detentionPolicyProposed = "DET-2027P"

	detentionAccessorial = "DET"
	layoverAccessorial   = "LAY"

	detentionFirstPro   = "SEED-DET-001"
	detentionProPrefix  = "SEED-DET-"
	detentionProPattern = detentionProPrefix + "%"

	detentionMinute = int64(60)
	detentionDay    = int64(86400)
)

// DetentionSeed builds a working detention book: the contract library, the org
// switch that turns the engine on, notice-ready customer contacts, and a set of
// shipments whose dwell times exercise every branch the desk has to render —
// open clocks, missed notices, tiered ladders, day caps, layover conversion,
// waivers, and disputes.
//
// Depends on:
//   - TestData: accessorial charges, service and shipment types, equipment
//   - Shipment: customers and commodities
//   - DriverPay: pay profiles, so occurrences carry a real AP side
type DetentionSeed struct {
}

type detentionRefs struct {
	orgID         pulid.ID
	buID          pulid.ID
	adminID       pulid.ID
	timezone      *time.Location
	locations     map[string]*location.Location
	customers     map[string]*customer.Customer
	serviceTypes  map[string]pulid.ID
	shipmentTypes map[string]pulid.ID
	accessorials  map[string]*accessorialcharge.AccessorialCharge
	chargesByID   map[pulid.ID]*accessorialcharge.AccessorialCharge
	workers       map[string]pulid.ID
	payRates      map[string]decimal.NullDecimal
	tractors      map[string]pulid.ID
	trailers      map[string]pulid.ID
	commodityID   pulid.ID
	formula       *formulatemplate.FormulaTemplate
	recipients    map[string][]string
}

func (s *DetentionSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	admin, err := tenantfixtures.UserByUsername(ctx, db, "admin")
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}

	refs, err := s.loadRefs(ctx, db, org, admin.ID)
	if err != nil {
		return err
	}

	policies, err := s.ensurePolicies(ctx, db, sr, refs)
	if err != nil {
		return fmt.Errorf("ensure detention policies: %w", err)
	}

	if err = s.enablePolicyEngine(ctx, db, refs, policies[detentionPolicyStandard]); err != nil {
		return fmt.Errorf("enable detention policy engine: %w", err)
	}

	if err = s.ensureNoticeRecipients(ctx, db, sr, refs); err != nil {
		return fmt.Errorf("ensure notice recipients: %w", err)
	}

	if err = s.createDetentionBook(ctx, db, sr, refs, policies); err != nil {
		return fmt.Errorf("create detention book: %w", err)
	}

	backfilled, err := pricingfixtures.BackfillSeededRatingDetails(
		ctx,
		db,
		pricingfixtures.SeededRatingBackfill{
			OrgID:     refs.orgID,
			BuID:      refs.buID,
			ProPrefix: detentionProPrefix,
			Now:       timeutils.NowUnix(),
		},
	)
	if err != nil {
		return fmt.Errorf("backfill detention shipment rating details: %w", err)
	}
	if backfilled > 0 {
	}

	return nil
}

// -- Reference data --

//nolint:funlen // one linear pass over the fixtures every detention record needs
func (s *DetentionSeed) loadRefs(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
	adminID pulid.ID,
) (*detentionRefs, error) {
	refs := &detentionRefs{
		orgID:         org.ID,
		buID:          org.BusinessUnitID,
		adminID:       adminID,
		timezone:      time.UTC,
		locations:     make(map[string]*location.Location),
		customers:     make(map[string]*customer.Customer),
		serviceTypes:  make(map[string]pulid.ID),
		shipmentTypes: make(map[string]pulid.ID),
		accessorials:  make(map[string]*accessorialcharge.AccessorialCharge),
		chargesByID:   make(map[pulid.ID]*accessorialcharge.AccessorialCharge),
		workers:       make(map[string]pulid.ID),
		payRates:      make(map[string]decimal.NullDecimal),
		tractors:      make(map[string]pulid.ID),
		trailers:      make(map[string]pulid.ID),
		recipients:    make(map[string][]string),
	}

	if loc, err := time.LoadLocation(timeutils.NormalizeTimezone(org.Timezone)); err == nil {
		refs.timezone = loc
	}

	locs, err := locationdb.Locations.AllTenants().
		Pick(locationdb.Locations.ID, locationdb.Locations.Code, locationdb.Locations.Name).
		Where(
			locationdb.Locations.OrganizationID.Eq(refs.orgID),
			locationdb.Locations.BusinessUnitID.Eq(refs.buID),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get locations: %w", err)
	}
	for i := range locs {
		refs.locations[locs[i].Code] = locs[i]
	}

	custs, err := customerdb.Customers.AllTenants().
		Pick(customerdb.Customers.ID, customerdb.Customers.Code, customerdb.Customers.Name).
		Where(
			customerdb.Customers.OrganizationID.Eq(refs.orgID),
			customerdb.Customers.BusinessUnitID.Eq(refs.buID),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get customers: %w", err)
	}
	for i := range custs {
		refs.customers[custs[i].Code] = custs[i]
	}

	svcTypes, err := servicetypedb.ServiceTypes.AllTenants().
		Pick(servicetypedb.ServiceTypes.ID, servicetypedb.ServiceTypes.Code).
		Where(
			servicetypedb.ServiceTypes.OrganizationID.Eq(refs.orgID),
			servicetypedb.ServiceTypes.BusinessUnitID.Eq(refs.buID),
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
			shipmenttypedb.ShipmentTypes.OrganizationID.Eq(refs.orgID),
			shipmenttypedb.ShipmentTypes.BusinessUnitID.Eq(refs.buID),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get shipment types: %w", err)
	}
	for i := range shpTypes {
		refs.shipmentTypes[shpTypes[i].Code] = shpTypes[i].ID
	}

	charges, err := accessorialchargedb.AccessorialCharges.AllTenants().
		Pick(accessorialchargedb.AccessorialCharges.ID, accessorialchargedb.AccessorialCharges.Code, accessorialchargedb.AccessorialCharges.Description, accessorialchargedb.AccessorialCharges.Method, accessorialchargedb.AccessorialCharges.RateUnit, accessorialchargedb.AccessorialCharges.Amount).
		Where(
			accessorialchargedb.AccessorialCharges.OrganizationID.Eq(refs.orgID),
			accessorialchargedb.AccessorialCharges.BusinessUnitID.Eq(refs.buID),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get accessorial charges: %w", err)
	}
	for i := range charges {
		refs.accessorials[charges[i].Code] = charges[i]
		refs.chargesByID[charges[i].ID] = charges[i]
	}
	for _, code := range []string{detentionAccessorial, layoverAccessorial} {
		if _, ok := refs.accessorials[code]; !ok {
			return nil, fmt.Errorf("accessorial charge %s not found", code)
		}
	}

	tractors, err := tractordb.Tractors.AllTenants().
		Pick(tractordb.Tractors.ID, tractordb.Tractors.Code).
		Where(
			tractordb.Tractors.OrganizationID.Eq(refs.orgID),
			tractordb.Tractors.BusinessUnitID.Eq(refs.buID),
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
			trailerdb.Trailers.OrganizationID.Eq(refs.orgID),
			trailerdb.Trailers.BusinessUnitID.Eq(refs.buID),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get trailers: %w", err)
	}
	for i := range trailers {
		refs.trailers[trailers[i].Code] = trailers[i].ID
	}

	workers, err := workerdb.Workers.AllTenants().
		Pick(workerdb.Workers.ID, workerdb.Workers.Email).
		Where(
			workerdb.Workers.OrganizationID.Eq(refs.orgID),
			workerdb.Workers.BusinessUnitID.Eq(refs.buID),
		).
		All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get workers: %w", err)
	}
	for i := range workers {
		refs.workers[workers[i].Email] = workers[i].ID
	}

	if err := s.loadDetentionPayRates(ctx, db, refs); err != nil {
		return nil, err
	}

	freight, err := commoditydb.Commodities.AllTenants().Pick(commoditydb.Commodities.ID).Where(
		commoditydb.Commodities.OrganizationID.Eq(refs.orgID),
		commoditydb.Commodities.BusinessUnitID.Eq(refs.buID),
		commoditydb.Commodities.Name.Eq("General Freight"),
	).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get general freight commodity: %w", err)
	}
	refs.commodityID = freight.ID

	if refs.formula, err = formulatemplatedb.FormulaTemplates.AllTenants().Pick(formulatemplatedb.FormulaTemplates.ID, formulatemplatedb.FormulaTemplates.Name).Where(
		formulatemplatedb.FormulaTemplates.OrganizationID.Eq(refs.orgID),
		formulatemplatedb.FormulaTemplates.BusinessUnitID.Eq(refs.buID),
		formulatemplatedb.FormulaTemplates.Name.Eq("Flat Rate"),
	).First(ctx, db); err != nil {
		return nil, fmt.Errorf("get flat rate formula template: %w", err)
	}

	return refs, nil
}

// loadDetentionPayRates resolves the hourly detention rate each driver is paid,
// the same way the service does at runtime, so a seeded occurrence carries a
// truthful AP side and net margin rather than a fabricated one.
func (s *DetentionSeed) loadDetentionPayRates(
	ctx context.Context,
	db *keel.DB,
	refs *detentionRefs,
) error {
	type workerDetentionRate struct {
		Email string
		Rate  decimal.Decimal
	}

	now := timeutils.NowUnix()
	wpa := payassignmentdb.WorkerPayAssignments
	w := workerdb.Workers
	comp := driverpaydb.PayProfileComponents
	rows, err := keel.Into[workerDetentionRate](ctx, db, wpa.AllTenants().
		Join(w, w.ID.EqCol(wpa.WorkerID), w.OrganizationID.EqCol(wpa.OrganizationID)).
		Join(comp, comp.PayProfileID.EqCol(wpa.PayProfileID), comp.OrganizationID.EqCol(wpa.OrganizationID)).
		Pick(w.Email, comp.Rate).
		Where(
			wpa.OrganizationID.Eq(refs.orgID),
			wpa.BusinessUnitID.Eq(refs.buID),
			wpa.EffectiveFrom.Lte(now),
			keel.Or(wpa.EffectiveTo.IsNull(), wpa.EffectiveTo.Gte(now)),
			comp.Kind.Eq(driverpay.ComponentKindDetention),
			comp.IsActive.IsTrue(),
		))
	if err != nil {
		return fmt.Errorf("get worker detention pay rates: %w", err)
	}

	for _, row := range rows {
		refs.payRates[row.Email] = decimal.NewNullDecimal(row.Rate)
	}

	return nil
}

// -- Policies --

type detentionPolicyDef struct {
	policy *detentionpolicy.DetentionPolicy
	tiers  []*detentionpolicy.DetentionPolicyTier
}

//nolint:funlen // a policy library is a contract catalogue; each term is explicit
func (s *DetentionSeed) policyDefs(refs *detentionRefs) []detentionPolicyDef {
	det := refs.accessorials[detentionAccessorial]
	lay := refs.accessorials[layoverAccessorial]
	now := timeutils.NowUnix()
	layoverCharge := lay.ID

	acmeID := refs.customers["ACME"].ID
	glblID := refs.customers["GLBL"].ID
	coldID := refs.locations["COLD-PHX"].ID

	return []detentionPolicyDef{
		{
			policy: &detentionpolicy.DetentionPolicy{
				ID:                          pulid.MustNew("dtp_"),
				BusinessUnitID:              refs.buID,
				OrganizationID:              refs.orgID,
				Name:                        "Standard detention terms",
				Code:                        detentionPolicyStandard,
				Description:                 "Two hours free, billed in quarter hours at the detention accessorial rate. Applies wherever no negotiated term is on file.",
				Status:                      detentionpolicy.PolicyStatusActive,
				IsOrgDefault:                true,
				Priority:                    0,
				ClockStartBasis:             detentionpolicy.ClockStartBasisLaterOfArrivalOrAppointment,
				LateArrivalRule:             detentionpolicy.LateArrivalRuleNoEffect,
				LateArrivalGraceMinutes:     15,
				BillingFreeMinutes:          120,
				PayFreeMinutes:              new(int32(120)),
				MinimumBillableMinutes:      15,
				BillingIncrementMinutes:     15,
				RoundingMode:                detentionpolicy.RoundingModeUp,
				RateSource:                  detentionpolicy.RateSourceAccessorial,
				AccessorialChargeID:         det.ID,
				MaxChargePerDay:             decimal.NewNullDecimal(decimal.NewFromInt(900)),
				DayBoundaryMode:             detentionpolicy.CapScopePerDay,
				NotificationRequirement:     detentionpolicy.NotificationRequirementAdvisory,
				NotificationLeadMinutes:     30,
				NotificationDeadlineMinutes: 0,
				UnnotifiedBehavior:          detentionpolicy.UnnotifiedBehaviorBill,
				SendDepartureSummary:        true,
				RequireApprovalOverAmount:   decimal.NewNullDecimal(decimal.NewFromInt(750)),
				AutoApproveUnderAmount:      decimal.NewNullDecimal(decimal.NewFromInt(150)),
				Currency:                    "USD",
				Comments:                    "Fallback terms. Anything negotiated with a customer or facility outranks this policy on specificity.",
			},
		},
		{
			policy: &detentionpolicy.DetentionPolicy{
				ID:                          pulid.MustNew("dtp_"),
				BusinessUnitID:              refs.buID,
				OrganizationID:              refs.orgID,
				Name:                        "Acme Manufacturing — negotiated detention",
				Code:                        detentionPolicyAcme,
				Description:                 "Graduated ladder with split pickup and delivery free time. Written notice is a condition of billing.",
				Status:                      detentionpolicy.PolicyStatusActive,
				Priority:                    10,
				CustomerID:                  &acmeID,
				ClockStartBasis:             detentionpolicy.ClockStartBasisLaterOfArrivalOrAppointment,
				LateArrivalRule:             detentionpolicy.LateArrivalRuleReduceFreeTime,
				LateArrivalGraceMinutes:     15,
				BillingFreeMinutes:          120,
				PickupFreeMinutes:           new(int32(90)),
				DeliveryFreeMinutes:         new(int32(150)),
				PayFreeMinutes:              new(int32(60)),
				MinimumBillableMinutes:      30,
				BillingIncrementMinutes:     15,
				RoundingMode:                detentionpolicy.RoundingModeUp,
				RateSource:                  detentionpolicy.RateSourceTiers,
				AccessorialChargeID:         det.ID,
				MaxChargePerStop:            decimal.NewNullDecimal(decimal.NewFromInt(1200)),
				MaxChargePerShipment:        decimal.NewNullDecimal(decimal.NewFromInt(2000)),
				DayBoundaryMode:             detentionpolicy.CapScopePerStop,
				NotificationRequirement:     detentionpolicy.NotificationRequirementRequired,
				NotificationLeadMinutes:     30,
				NotificationDeadlineMinutes: 60,
				UnnotifiedBehavior:          detentionpolicy.UnnotifiedBehaviorFlag,
				AutoSendNotice:              true,
				SendDepartureSummary:        true,
				RequireApprovalOverAmount:   decimal.NewNullDecimal(decimal.NewFromInt(1000)),
				AutoApproveUnderAmount:      decimal.NewNullDecimal(decimal.NewFromInt(250)),
				Currency:                    "USD",
				Comments:                    "Master agreement §7. Notice inside 60 minutes of free-time expiry is a condition precedent to the charge.",
			},
			tiers: []*detentionpolicy.DetentionPolicyTier{
				{
					FromMinute: 0, ToMinute: new(int32(240)),
					Rate: decimal.NewFromInt(60), RateUnit: detentionpolicy.TierRateUnitHour,
					Label: "First 4 hours", SortOrder: 0,
				},
				{
					FromMinute: 240, ToMinute: new(int32(720)),
					Rate: decimal.NewFromInt(85), RateUnit: detentionpolicy.TierRateUnitHour,
					Label: "Hours 4 through 12", SortOrder: 1,
				},
				{
					FromMinute: 720,
					Rate:       decimal.NewFromInt(750), RateUnit: detentionpolicy.TierRateUnitDay,
					Label: "Beyond 12 hours", SortOrder: 2,
				},
			},
		},
		{
			policy: &detentionpolicy.DetentionPolicy{
				ID:                          pulid.MustNew("dtp_"),
				BusinessUnitID:              refs.buID,
				OrganizationID:              refs.orgID,
				Name:                        "Phoenix Cold Storage — facility terms",
				Code:                        detentionPolicyCold,
				Description:                 "One hour free from arrival, capped daily, converting to layover after 24 hours on site. Notice is required or the charge is suppressed.",
				Status:                      detentionpolicy.PolicyStatusActive,
				Priority:                    20,
				LocationID:                  &coldID,
				ClockStartBasis:             detentionpolicy.ClockStartBasisArrival,
				LateArrivalRule:             detentionpolicy.LateArrivalRuleNoEffect,
				BillingFreeMinutes:          60,
				PayFreeMinutes:              new(int32(60)),
				MinimumBillableMinutes:      30,
				BillingIncrementMinutes:     30,
				RoundingMode:                detentionpolicy.RoundingModeNearest,
				RateSource:                  detentionpolicy.RateSourceAccessorial,
				AccessorialChargeID:         det.ID,
				MaxChargePerDay:             decimal.NewNullDecimal(decimal.NewFromInt(600)),
				DayBoundaryMode:             detentionpolicy.CapScopePerDay,
				ConvertToLayoverAtMinutes:   new(int32(1440)),
				LayoverAccessorialChargeID:  &layoverCharge,
				NotificationRequirement:     detentionpolicy.NotificationRequirementRequired,
				NotificationLeadMinutes:     30,
				NotificationDeadlineMinutes: 30,
				UnnotifiedBehavior:          detentionpolicy.UnnotifiedBehaviorSuppress,
				SendDepartureSummary:        true,
				Currency:                    "USD",
				Comments:                    "Facility rider. The dock refuses charges without a timestamped notice, so an unnotified stop is not worth billing.",
			},
		},
		{
			policy: &detentionpolicy.DetentionPolicy{
				ID:                        pulid.MustNew("dtp_"),
				BusinessUnitID:            refs.buID,
				OrganizationID:            refs.orgID,
				Name:                      "GlobalTrade drayage — port pickups",
				Code:                      detentionPolicyDray,
				Description:               "Four hours free measured from the appointment. Arriving late forfeits detention entirely.",
				Status:                    detentionpolicy.PolicyStatusActive,
				Priority:                  15,
				CustomerID:                &glblID,
				StopTypes:                 []shipmentvocab.StopType{shipmentvocab.StopTypePickup},
				AppointmentStopsOnly:      true,
				ClockStartBasis:           detentionpolicy.ClockStartBasisAppointment,
				LateArrivalRule:           detentionpolicy.LateArrivalRuleForfeit,
				LateArrivalGraceMinutes:   30,
				BillingFreeMinutes:        240,
				PayFreeMinutes:            new(int32(180)),
				MinimumBillableMinutes:    60,
				BillingIncrementMinutes:   60,
				RoundingMode:              detentionpolicy.RoundingModeUp,
				RateSource:                detentionpolicy.RateSourceAccessorial,
				AccessorialChargeID:       det.ID,
				MaxBillableMinutesPerStop: new(int32(720)),
				DayBoundaryMode:           detentionpolicy.CapScopePerStop,
				NotificationRequirement:   detentionpolicy.NotificationRequirementNone,
				UnnotifiedBehavior:        detentionpolicy.UnnotifiedBehaviorBill,
				Currency:                  "USD",
				Comments:                  "Port terminal rider. The appointment is the clock; a missed appointment is the carrier's cost.",
				NotificationLeadMinutes:   30,
			},
		},
		{
			policy: &detentionpolicy.DetentionPolicy{
				ID:                      pulid.MustNew("dtp_"),
				BusinessUnitID:          refs.buID,
				OrganizationID:          refs.orgID,
				Name:                    "2025 legacy detention terms",
				Code:                    detentionPolicyLegacy,
				Description:             "Superseded terms retained so historical occurrences still explain themselves.",
				Status:                  detentionpolicy.PolicyStatusInactive,
				Priority:                0,
				EffectiveStartDate:      new(now - 455*detentionDay),
				EffectiveEndDate:        new(now - 90*detentionDay),
				ClockStartBasis:         detentionpolicy.ClockStartBasisArrival,
				LateArrivalRule:         detentionpolicy.LateArrivalRuleNoEffect,
				BillingFreeMinutes:      180,
				PayFreeMinutes:          new(int32(180)),
				BillingIncrementMinutes: 60,
				RoundingMode:            detentionpolicy.RoundingModeDown,
				RateSource:              detentionpolicy.RateSourceAccessorial,
				AccessorialChargeID:     det.ID,
				DayBoundaryMode:         detentionpolicy.CapScopePerStop,
				NotificationRequirement: detentionpolicy.NotificationRequirementNone,
				UnnotifiedBehavior:      detentionpolicy.UnnotifiedBehaviorBill,
				Currency:                "USD",
				Comments:                "Expired. Kept for audit of anything billed before the 2026 renegotiation.",
				NotificationLeadMinutes: 30,
			},
		},
		{
			policy: &detentionpolicy.DetentionPolicy{
				ID:                          pulid.MustNew("dtp_"),
				BusinessUnitID:              refs.buID,
				OrganizationID:              refs.orgID,
				Name:                        "2027 proposed detention terms",
				Code:                        detentionPolicyProposed,
				Description:                 "Draft ladder for the 2027 rate cycle. Backtest it against the last quarter before promoting it.",
				Status:                      detentionpolicy.PolicyStatusDraft,
				Priority:                    5,
				ClockStartBasis:             detentionpolicy.ClockStartBasisLaterOfArrivalOrAppointment,
				LateArrivalRule:             detentionpolicy.LateArrivalRuleReduceFreeTime,
				LateArrivalGraceMinutes:     15,
				BillingFreeMinutes:          60,
				PayFreeMinutes:              new(int32(60)),
				MinimumBillableMinutes:      15,
				BillingIncrementMinutes:     15,
				RoundingMode:                detentionpolicy.RoundingModeUp,
				RateSource:                  detentionpolicy.RateSourceTiers,
				AccessorialChargeID:         det.ID,
				MaxChargePerStop:            decimal.NewNullDecimal(decimal.NewFromInt(1500)),
				DayBoundaryMode:             detentionpolicy.CapScopePerStop,
				NotificationRequirement:     detentionpolicy.NotificationRequirementRequired,
				NotificationLeadMinutes:     30,
				NotificationDeadlineMinutes: 30,
				UnnotifiedBehavior:          detentionpolicy.UnnotifiedBehaviorFlag,
				AutoSendNotice:              true,
				Currency:                    "USD",
				Comments:                    "Draft. Tighter free time and a steeper second tier; run the backtest before making it active.",
			},
			tiers: []*detentionpolicy.DetentionPolicyTier{
				{
					FromMinute: 0, ToMinute: new(int32(180)),
					Rate: decimal.NewFromInt(80), RateUnit: detentionpolicy.TierRateUnitHour,
					Label: "First 3 hours", SortOrder: 0,
				},
				{
					FromMinute: 180,
					Rate:       decimal.NewFromInt(110), RateUnit: detentionpolicy.TierRateUnitHour,
					Label: "Beyond 3 hours", SortOrder: 1,
				},
			},
		},
	}
}

func (s *DetentionSeed) ensurePolicies(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *detentionRefs,
) (map[string]*detentionpolicy.DetentionPolicy, error) {
	policies := make(map[string]*detentionpolicy.DetentionPolicy, 6)
	created := 0

	for _, def := range s.policyDefs(refs) {
		existing, err := s.findPolicy(ctx, db, refs, def.policy.Code)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			policies[existing.Code] = existing
			continue
		}

		if err = detentionpolicydb.DetentionPolicies.Create(ctx, db, def.policy); err != nil {
			return nil, fmt.Errorf("insert detention policy %s: %w", def.policy.Code, err)
		}

		if len(def.tiers) > 0 {
			for _, tier := range def.tiers {
				tier.ID = pulid.MustNew("dpt_")
				tier.BusinessUnitID = refs.buID
				tier.OrganizationID = refs.orgID
				tier.DetentionPolicyID = def.policy.ID
			}
			if err = detentionpolicydb.DetentionPolicyTiers.CreateMany(ctx, db, def.tiers); err != nil {
				return nil, fmt.Errorf("insert tiers for %s: %w", def.policy.Code, err)
			}
			def.policy.Tiers = def.tiers
		}

		policies[def.policy.Code] = def.policy
		created++
	}
	sr.Count(int64(created))

	return policies, nil
}

func (s *DetentionSeed) findPolicy(
	ctx context.Context,
	db *keel.DB,
	refs *detentionRefs,
	code string,
) (*detentionpolicy.DetentionPolicy, error) {
	policy, err := detentionpolicydb.DetentionPolicies.AllTenants().Where(
		detentionpolicydb.DetentionPolicies.OrganizationID.Eq(refs.orgID),
		detentionpolicydb.DetentionPolicies.BusinessUnitID.Eq(refs.buID),
		detentionpolicydb.DetentionPolicies.Code.Eq(code),
	).First(ctx, db)
	if err != nil {
		if errors.Is(err, keel.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get detention policy %s: %w", code, err)
	}

	if policy.RateSource != detentionpolicy.RateSourceTiers {
		return policy, nil
	}

	tiers, err := detentionpolicydb.DetentionPolicyTiers.AllTenants().Where(
		detentionpolicydb.DetentionPolicyTiers.OrganizationID.Eq(refs.orgID),
		detentionpolicydb.DetentionPolicyTiers.BusinessUnitID.Eq(refs.buID),
		detentionpolicydb.DetentionPolicyTiers.DetentionPolicyID.Eq(policy.ID),
	).Order(detentionpolicydb.DetentionPolicyTiers.FromMinute.Asc()).All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get tiers for detention policy %s: %w", code, err)
	}
	policy.Tiers = tiers

	return policy, nil
}

// enablePolicyEngine flips the organization onto the policy engine. Without it
// the legacy flat-threshold path stays in charge and none of this data is
// reachable from the shipment.
func (s *DetentionSeed) enablePolicyEngine(
	ctx context.Context,
	db *keel.DB,
	refs *detentionRefs,
	defaultPolicy *detentionpolicy.DetentionPolicy,
) error {
	if defaultPolicy == nil {
		return errors.New("default detention policy is missing")
	}

	sc := shipmentcontroldb.ShipmentControls
	_, err := sc.AllTenants().Where(
		sc.OrganizationID.Eq(refs.orgID),
		sc.BusinessUnitID.Eq(refs.buID),
	).Update(
		ctx,
		db,
		sc.TrackDetentionTime.To(true),
		sc.UseDetentionPolicyEngine.To(true),
		sc.DefaultDetentionPolicyID.To(defaultPolicy.ID),
		sc.DetentionChargeID.To(refs.accessorials[detentionAccessorial].ID),
	)
	if err != nil {
		return fmt.Errorf("update shipment control: %w", err)
	}

	return nil
}

// -- Notice recipients --

// ensureNoticeRecipients gives every detention customer an email profile. The
// notice pipeline resolves recipients from it and refuses to send without one,
// so the desk's "Send notice" action is dead until this exists.
func (s *DetentionSeed) ensureNoticeRecipients(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *detentionRefs,
) error {
	defs := []struct {
		customer string
		domain   string
	}{
		{"ACME", "acmemfg.example.com"},
		{"GLBL", "globaltrade.example.com"},
		{"PEAK", "peakdist.example.com"},
		{"FRSH", "freshhaul.example.com"},
		{"RNGE", "rangelogistics.example.com"},
		{"SUNB", "sunbeltmaterials.example.com"},
	}

	created := 0
	filled := 0

	for _, def := range defs {
		cust, ok := refs.customers[def.customer]
		if !ok {
			return fmt.Errorf("customer %s not found", def.customer)
		}

		recipients := []string{"billing@" + def.domain, "dock@" + def.domain}
		refs.recipients[def.customer] = recipients

		existing, err := customerdb.CustomerEmailProfiles.AllTenants().Where(
			customerdb.CustomerEmailProfiles.OrganizationID.Eq(refs.orgID),
			customerdb.CustomerEmailProfiles.BusinessUnitID.Eq(refs.buID),
			customerdb.CustomerEmailProfiles.CustomerID.Eq(cust.ID),
		).First(ctx, db)
		switch {
		case err == nil && existing.ToRecipients != "":
			continue
		case err == nil:
			// A profile with no recipients is worse than none: the notice
			// pipeline finds it, resolves nothing, and refuses to send.
			ep := customerdb.CustomerEmailProfiles
			existing.ToRecipients = recipients[0] + "," + recipients[1]
			existing.CCRecipients = "ap@" + def.domain
			if err = ep.Save(ctx, db, existing, ep.ToRecipients, ep.CCRecipients); err != nil {
				return fmt.Errorf("fill email profile for %s: %w", def.customer, err)
			}
			filled++
			continue
		case !errors.Is(err, keel.ErrNotFound):
			return fmt.Errorf("check email profile for %s: %w", def.customer, err)
		}

		profile := &customer.CustomerEmailProfile{
			ID:             pulid.MustNew("cem_"),
			BusinessUnitID: refs.buID,
			OrganizationID: refs.orgID,
			CustomerID:     cust.ID,
			Subject:        "Shipment documentation",
			Comment:        "Detention notices and rate confirmations for " + cust.Name,
			ToRecipients:   recipients[0] + "," + recipients[1],
			CCRecipients:   "ap@" + def.domain,
			AttachmentName: "shipment-documents.pdf",
			ReadReceipt:    true,
		}
		if err = customerdb.CustomerEmailProfiles.Create(ctx, db, profile); err != nil {
			return fmt.Errorf("insert email profile for %s: %w", def.customer, err)
		}
		created++
	}

	if created > 0 || filled > 0 {
	}

	return nil
}

// -- The detention book --

type detentionOutcome string

const (
	detentionOutcomeNone    detentionOutcome = ""
	detentionOutcomeApprove detentionOutcome = "approve"
	detentionOutcomeBill    detentionOutcome = "bill"
	detentionOutcomeWaive   detentionOutcome = "waive"
	detentionOutcomeDispute detentionOutcome = "dispute"
)

// detentionNoticeDef places a notice relative to the moment free time lapses,
// which is the only reference point the contract cares about.
type detentionNoticeDef struct {
	offsetFromExpiry int64
	kind             detentionnotice.NoticeKind
	delivery         detentionnotice.NoticeDeliveryStatus
	failureReason    string
	automatic        bool
}

type detentionStopDef struct {
	location        string
	stopType        shipmentvocab.StopType
	scheduleType    shipmentvocab.StopScheduleType
	apptStartAgo    int64
	apptWindow      int64
	arrivalOffset   int64
	dwell           int64
	arrivalSource   detentionevidence.EvidenceSource
	departureSource detentionevidence.EvidenceSource
	document        string
	notice          *detentionNoticeDef
	outcome         detentionOutcome
	waiverReason    detention.WaiverReason
	note            string
}

type detentionShipmentDef struct {
	pro        string
	bol        string
	customer   string
	service    string
	shipType   string
	status     shipmentvocab.Status
	moveStatus shipmentvocab.MoveStatus
	revenue    float64
	pieces     int64
	weight     int64
	workerKey  string
	tractor    string
	trailer    string
	stops      []detentionStopDef
}

//nolint:funlen // the scenario table is data: every branch of the engine gets a row
func detentionShipmentDefs() []detentionShipmentDef {
	return []detentionShipmentDef{
		{
			pro: detentionFirstPro, bol: "BOL-2026-0201", customer: "ACME",
			service: "STD", shipType: "FTL", status: shipmentvocab.StatusInTransit,
			moveStatus: shipmentvocab.MoveStatusInTransit,
			revenue:    2850, pieces: 24, weight: 39000,
			workerKey: driverpayfixtures.PayWorkerJane, tractor: "TRC-003", trailer: "TRL-001",
			stops: []detentionStopDef{
				{
					location: "TERM-LA", stopType: shipmentvocab.StopTypePickup,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 26 * 60, apptWindow: 120, arrivalOffset: 10, dwell: 55,
					arrivalSource:   detentionevidence.EvidenceSourceGeofence,
					departureSource: detentionevidence.EvidenceSourceGeofence,
				},
				{
					location: "DC-CHI", stopType: shipmentvocab.StopTypeDelivery,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 240, apptWindow: 120, arrivalOffset: 15, dwell: 0,
					arrivalSource: detentionevidence.EvidenceSourceTelematics,
					notice: &detentionNoticeDef{
						offsetFromExpiry: -20, kind: detentionnotice.NoticeKindWarning,
						delivery: detentionnotice.NoticeDeliveryStatusOpened, automatic: true,
					},
				},
			},
		},
		{
			pro: "SEED-DET-002", bol: "BOL-2026-0202", customer: "FRSH",
			service: "TEMP", shipType: "REEF", status: shipmentvocab.StatusDelayed,
			moveStatus: shipmentvocab.MoveStatusInTransit,
			revenue:    3300, pieces: 20, weight: 36000,
			workerKey: driverpayfixtures.PayWorkerSarah, tractor: "TRC-002", trailer: "TRL-002",
			stops: []detentionStopDef{
				{
					location: "WH-DAL", stopType: shipmentvocab.StopTypePickup,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 30 * 60, apptWindow: 120, arrivalOffset: 5, dwell: 50,
					arrivalSource:   detentionevidence.EvidenceSourceDriverApp,
					departureSource: detentionevidence.EvidenceSourceDriverApp,
				},
				{
					location: "COLD-PHX", stopType: shipmentvocab.StopTypeDelivery,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 300, apptWindow: 60, arrivalOffset: 20, dwell: 0,
					arrivalSource: detentionevidence.EvidenceSourceGeofence,
				},
			},
		},
		{
			pro: "SEED-DET-003", bol: "BOL-2026-0203", customer: "PEAK",
			service: "STD", shipType: "FTL", status: shipmentvocab.StatusCompleted,
			moveStatus: shipmentvocab.MoveStatusCompleted,
			revenue:    3150, pieces: 26, weight: 41000,
			workerKey: driverpayfixtures.PayWorkerSarah, tractor: "TRC-001", trailer: "TRL-003",
			stops: []detentionStopDef{
				{
					location: "DC-CHI", stopType: shipmentvocab.StopTypePickup,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 4 * 24 * 60, apptWindow: 120, arrivalOffset: 0, dwell: 200,
					arrivalSource:   detentionevidence.EvidenceSourceTelematics,
					departureSource: detentionevidence.EvidenceSourceTelematics,
					document:        "Signed BOL captured at the dock office with in and out times",
				},
				{
					location: "WH-DAL", stopType: shipmentvocab.StopTypeDelivery,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 3 * 24 * 60, apptWindow: 120, arrivalOffset: 20, dwell: 480,
					arrivalSource:   detentionevidence.EvidenceSourceGeofence,
					departureSource: detentionevidence.EvidenceSourceGeofence,
					document:        "Signed POD showing the receiver's own arrival and release stamps",
					notice: &detentionNoticeDef{
						offsetFromExpiry: -15, kind: detentionnotice.NoticeKindStarted,
						delivery: detentionnotice.NoticeDeliveryStatusDelivered,
					},
					outcome: detentionOutcomeApprove,
				},
			},
		},
		{
			pro: "SEED-DET-004", bol: "BOL-2026-0204", customer: "ACME",
			service: "STD", shipType: "FTL", status: shipmentvocab.StatusReadyToInvoice,
			moveStatus: shipmentvocab.MoveStatusCompleted,
			revenue:    4200, pieces: 30, weight: 43000,
			workerKey: driverpayfixtures.PayWorkerJane, tractor: "TRC-003", trailer: "TRL-004",
			stops: []detentionStopDef{
				{
					location: "TERM-LA", stopType: shipmentvocab.StopTypePickup,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 9 * 24 * 60, apptWindow: 120, arrivalOffset: -10, dwell: 55,
					arrivalSource:   detentionevidence.EvidenceSourceGeofence,
					departureSource: detentionevidence.EvidenceSourceGeofence,
				},
				{
					location: "CUST-DEN", stopType: shipmentvocab.StopTypeDelivery,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 8 * 24 * 60, apptWindow: 120, arrivalOffset: 5, dwell: 840,
					arrivalSource:   detentionevidence.EvidenceSourceTelematics,
					departureSource: detentionevidence.EvidenceSourceTelematics,
					document:        "Gate log printout from the receiver covering both timestamps",
					notice: &detentionNoticeDef{
						offsetFromExpiry: 90, kind: detentionnotice.NoticeKindStarted,
						delivery: detentionnotice.NoticeDeliveryStatusDelivered, automatic: true,
					},
					outcome: detentionOutcomeBill,
				},
			},
		},
		{
			pro: "SEED-DET-005", bol: "BOL-2026-0205", customer: "GLBL",
			service: "STD", shipType: "DRAY", status: shipmentvocab.StatusCompleted,
			moveStatus: shipmentvocab.MoveStatusCompleted,
			revenue:    1950, pieces: 18, weight: 34000,
			workerKey: driverpayfixtures.PayWorkerMike, tractor: "TRC-101", trailer: "TRL-005",
			stops: []detentionStopDef{
				{
					location: "TERM-LA", stopType: shipmentvocab.StopTypePickup,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 6 * 24 * 60, apptWindow: 60, arrivalOffset: 150, dwell: 300,
					arrivalSource:   detentionevidence.EvidenceSourceTelematics,
					departureSource: detentionevidence.EvidenceSourceTelematics,
				},
				{
					location: "CUST-DEN", stopType: shipmentvocab.StopTypeDelivery,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 5 * 24 * 60, apptWindow: 120, arrivalOffset: 10, dwell: 100,
					arrivalSource:   detentionevidence.EvidenceSourceDriverApp,
					departureSource: detentionevidence.EvidenceSourceDriverApp,
				},
			},
		},
		{
			pro: "SEED-DET-006", bol: "BOL-2026-0206", customer: "SUNB",
			service: "WHG", shipType: "PART", status: shipmentvocab.StatusReadyToInvoice,
			moveStatus: shipmentvocab.MoveStatusCompleted,
			revenue:    3600, pieces: 16, weight: 32000,
			workerKey: driverpayfixtures.PayWorkerSarah, tractor: "TRC-002", trailer: "TRL-001",
			stops: []detentionStopDef{
				{
					location: "WH-DAL", stopType: shipmentvocab.StopTypePickup,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 14 * 24 * 60, apptWindow: 120, arrivalOffset: 0, dwell: 70,
					arrivalSource:   detentionevidence.EvidenceSourceDriverApp,
					departureSource: detentionevidence.EvidenceSourceDriverApp,
				},
				{
					location: "DC-CHI", stopType: shipmentvocab.StopTypeDelivery,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 13 * 24 * 60, apptWindow: 120, arrivalOffset: 30, dwell: 1800,
					arrivalSource:   detentionevidence.EvidenceSourceGeofence,
					departureSource: detentionevidence.EvidenceSourceGeofence,
					notice: &detentionNoticeDef{
						offsetFromExpiry: -30, kind: detentionnotice.NoticeKindStarted,
						delivery: detentionnotice.NoticeDeliveryStatusDelivered,
					},
					outcome: detentionOutcomeDispute,
					note:    "Receiver disputes the overnight hold, claiming the trailer was dropped and the tractor released at 19:00.",
				},
			},
		},
		{
			pro: "SEED-DET-007", bol: "BOL-2026-0207", customer: "RNGE",
			service: "STD", shipType: "FTL", status: shipmentvocab.StatusCompleted,
			moveStatus: shipmentvocab.MoveStatusCompleted,
			revenue:    2450, pieces: 22, weight: 37000,
			workerKey: driverpayfixtures.PayWorkerJane, tractor: "TRC-004", trailer: "TRL-002",
			stops: []detentionStopDef{
				{
					location: "DC-CHI", stopType: shipmentvocab.StopTypePickup,
					scheduleType: shipmentvocab.StopScheduleTypeOpen,
					apptStartAgo: 11 * 24 * 60, apptWindow: 240, arrivalOffset: 30, dwell: 65,
					arrivalSource:   detentionevidence.EvidenceSourceManual,
					departureSource: detentionevidence.EvidenceSourceManual,
				},
				{
					location: "CUST-DEN", stopType: shipmentvocab.StopTypeDelivery,
					scheduleType: shipmentvocab.StopScheduleTypeOpen,
					apptStartAgo: 10 * 24 * 60, apptWindow: 240, arrivalOffset: 45, dwell: 240,
					arrivalSource:   detentionevidence.EvidenceSourceManual,
					departureSource: detentionevidence.EvidenceSourceManual,
				},
			},
		},
		{
			pro: "SEED-DET-008", bol: "BOL-2026-0208", customer: "FRSH",
			service: "TEMP", shipType: "REEF", status: shipmentvocab.StatusReadyToInvoice,
			moveStatus: shipmentvocab.MoveStatusCompleted,
			revenue:    5100, pieces: 28, weight: 42000,
			workerKey: driverpayfixtures.PayWorkerSarah, tractor: "TRC-002", trailer: "TRL-003",
			stops: []detentionStopDef{
				{
					location: "WH-DAL", stopType: shipmentvocab.StopTypePickup,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 20 * 24 * 60, apptWindow: 120, arrivalOffset: 15, dwell: 80,
					arrivalSource:   detentionevidence.EvidenceSourceGeofence,
					departureSource: detentionevidence.EvidenceSourceGeofence,
				},
				{
					location: "COLD-PHX", stopType: shipmentvocab.StopTypeDelivery,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 19 * 24 * 60, apptWindow: 60, arrivalOffset: 5, dwell: 2400,
					arrivalSource:   detentionevidence.EvidenceSourceGeofence,
					departureSource: detentionevidence.EvidenceSourceGeofence,
					document:        "Cold storage gate ticket showing the trailer sat two nights",
					notice: &detentionNoticeDef{
						offsetFromExpiry: -25, kind: detentionnotice.NoticeKindStarted,
						delivery: detentionnotice.NoticeDeliveryStatusOpened,
					},
					outcome: detentionOutcomeBill,
				},
			},
		},
		{
			pro: "SEED-DET-009", bol: "BOL-2026-0209", customer: "PEAK",
			service: "STD", shipType: "FTL", status: shipmentvocab.StatusCompleted,
			moveStatus: shipmentvocab.MoveStatusCompleted,
			revenue:    2700, pieces: 20, weight: 35000,
			workerKey: driverpayfixtures.PayWorkerJane, tractor: "TRC-003", trailer: "TRL-004",
			stops: []detentionStopDef{
				{
					location: "DC-CHI", stopType: shipmentvocab.StopTypePickup,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 7 * 24 * 60, apptWindow: 120, arrivalOffset: 0, dwell: 60,
					arrivalSource:   detentionevidence.EvidenceSourceDriverApp,
					departureSource: detentionevidence.EvidenceSourceDriverApp,
				},
				{
					location: "WH-DAL", stopType: shipmentvocab.StopTypeDelivery,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 6 * 24 * 60, apptWindow: 120, arrivalOffset: 10, dwell: 300,
					arrivalSource:   detentionevidence.EvidenceSourceTelematics,
					departureSource: detentionevidence.EvidenceSourceTelematics,
					notice: &detentionNoticeDef{
						offsetFromExpiry: -10, kind: detentionnotice.NoticeKindStarted,
						delivery: detentionnotice.NoticeDeliveryStatusDelivered,
					},
					outcome:      detentionOutcomeWaive,
					waiverReason: detention.WaiverReasonCustomerGoodwill,
					note:         "Waived by the account manager ahead of the Q3 bid; the delay was one shift short-staffed, not a pattern.",
				},
			},
		},
		{
			pro: "SEED-DET-010", bol: "BOL-2026-0210", customer: "ACME",
			service: "STD", shipType: "FTL", status: shipmentvocab.StatusCompleted,
			moveStatus: shipmentvocab.MoveStatusCompleted,
			revenue:    2200, pieces: 18, weight: 31000,
			workerKey: driverpayfixtures.PayWorkerJohn, tractor: "TRC-001", trailer: "TRL-005",
			stops: []detentionStopDef{
				{
					location: "TERM-LA", stopType: shipmentvocab.StopTypePickup,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 3 * 24 * 60, apptWindow: 120, arrivalOffset: 5, dwell: 45,
					arrivalSource:   detentionevidence.EvidenceSourceGeofence,
					departureSource: detentionevidence.EvidenceSourceGeofence,
				},
				{
					location: "DC-CHI", stopType: shipmentvocab.StopTypeDelivery,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 2 * 24 * 60, apptWindow: 120, arrivalOffset: 0, dwell: 120,
					arrivalSource:   detentionevidence.EvidenceSourceGeofence,
					departureSource: detentionevidence.EvidenceSourceGeofence,
					document:        "Signed POD released inside the appointment window",
				},
			},
		},
		{
			pro: "SEED-DET-011", bol: "BOL-2026-0211", customer: "GLBL",
			service: "STD", shipType: "FTL", status: shipmentvocab.StatusReadyToInvoice,
			moveStatus: shipmentvocab.MoveStatusCompleted,
			revenue:    3050, pieces: 24, weight: 38000,
			workerKey: driverpayfixtures.PayWorkerMike, tractor: "TRC-101", trailer: "TRL-002",
			stops: []detentionStopDef{
				{
					location: "CUST-DEN", stopType: shipmentvocab.StopTypePickup,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 16 * 24 * 60, apptWindow: 120, arrivalOffset: 10, dwell: 75,
					arrivalSource:   detentionevidence.EvidenceSourceDriverApp,
					departureSource: detentionevidence.EvidenceSourceDriverApp,
				},
				{
					location: "WH-DAL", stopType: shipmentvocab.StopTypeDelivery,
					scheduleType: shipmentvocab.StopScheduleTypeAppointment,
					apptStartAgo: 15 * 24 * 60, apptWindow: 120, arrivalOffset: 20, dwell: 420,
					arrivalSource:   detentionevidence.EvidenceSourceTelematics,
					departureSource: detentionevidence.EvidenceSourceTelematics,
					notice: &detentionNoticeDef{
						offsetFromExpiry: -20, kind: detentionnotice.NoticeKindStarted,
						delivery:      detentionnotice.NoticeDeliveryStatusBounced,
						failureReason: "550 5.1.1 recipient address rejected: user unknown",
						automatic:     true,
					},
				},
			},
		},
	}
}

func (s *DetentionSeed) createDetentionBook(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *detentionRefs,
	policies map[string]*detentionpolicy.DetentionPolicy,
) error {
	candidates := make([]*detentionpolicy.DetentionPolicy, 0, len(policies))
	for _, policy := range policies {
		candidates = append(candidates, policy)
	}

	defs := detentionShipmentDefs()
	occurrences := 0
	for _, def := range defs {
		count, sErr := s.createDetentionShipment(ctx, db, sr, refs, candidates, def)
		if sErr != nil {
			return fmt.Errorf("create shipment %s: %w", def.pro, sErr)
		}
		occurrences += count
	}

	return nil
}

//nolint:funlen // one shipment from header through moves, stops, and occurrences
func (s *DetentionSeed) createDetentionShipment(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *detentionRefs,
	candidates []*detentionpolicy.DetentionPolicy,
	def detentionShipmentDef,
) (int, error) {
	cust, ok := refs.customers[def.customer]
	if !ok {
		return 0, fmt.Errorf("customer %s not found", def.customer)
	}
	serviceTypeID, ok := refs.serviceTypes[def.service]
	if !ok {
		return 0, fmt.Errorf("service type %s not found", def.service)
	}
	shipmentTypeID, ok := refs.shipmentTypes[def.shipType]
	if !ok {
		return 0, fmt.Errorf("shipment type %s not found", def.shipType)
	}

	now := timeutils.NowUnix()
	stops := make([]*shipment.Stop, 0, len(def.stops))
	for i := range def.stops {
		stop, sErr := s.buildStop(refs, def.stops[i], int64(i), now)
		if sErr != nil {
			return 0, sErr
		}
		stops = append(stops, stop)
	}

	revenue := decimal.NewFromFloat(def.revenue)
	shippedAt := *stops[0].ActualArrival
	shp := &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		BusinessUnitID:    refs.buID,
		OrganizationID:    refs.orgID,
		CustomerID:        cust.ID,
		ServiceTypeID:     serviceTypeID,
		ShipmentTypeID:    shipmentTypeID,
		FormulaTemplateID: refs.formula.ID,
		Status:            def.status,
		ProNumber:         def.pro,
		BOL:               def.bol,
		Pieces:            new(def.pieces),
		Weight:            new(def.weight),
		ActualShipDate:    new(shippedAt),
		TariffInputs: pricingfixtures.SeededTariffInputs(
			formulatemplate.StandardFlatRate,
			revenue,
		),
		FreightChargeAmount: decimal.NullDecimal{Decimal: revenue, Valid: true},
		TotalChargeAmount:   decimal.NullDecimal{Decimal: revenue, Valid: true},
		EntryMethod:         shipmentvocab.EntryMethodManual,
		FreightTerms:        shipmentvocab.FreightTermsPrepaid,
		FreightLines:        []shipment.FreightLine{},
		OtherChargeAmount:   decimal.NewNullDecimal(decimal.RequireFromString("0")),
	}
	last := stops[len(stops)-1]
	if last.ActualDeparture != nil {
		shp.ActualDeliveryDate = new(*last.ActualDeparture)
	}
	shp.FreightLines = pricingfixtures.SeededFreightLines(shp)
	shp.RatingDetail = pricingfixtures.SeededRatingDetail(shp, refs.formula, now)
	if err := shipmentdb.Shipments.Create(ctx, db, shp); err != nil {
		return 0, fmt.Errorf("insert shipment: %w", err)
	}

	shipCommodity := &shipment.ShipmentCommodity{
		ID:             pulid.MustNew("sc_"),
		BusinessUnitID: refs.buID,
		OrganizationID: refs.orgID,
		ShipmentID:     shp.ID,
		CommodityID:    refs.commodityID,
		Pieces:         def.pieces,
		Weight:         def.weight,
	}
	if err := shipmentdb.ShipmentCommodities.Create(ctx, db, shipCommodity); err != nil {
		return 0, fmt.Errorf("insert shipment commodity: %w", err)
	}

	move := &shipment.ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		BusinessUnitID: refs.buID,
		OrganizationID: refs.orgID,
		ShipmentID:     shp.ID,
		Status:         def.moveStatus,
		Loaded:         true,
		Sequence:       0,
		CoverageType:   shipmentvocab.MoveCoverageTypeDriver,
	}
	if err := shipmentdb.ShipmentMoves.Create(ctx, db, move); err != nil {
		return 0, fmt.Errorf("insert shipment move: %w", err)
	}

	for _, stop := range stops {
		stop.ShipmentMoveID = move.ID
	}
	if err := shipmentdb.Stops.CreateMany(ctx, db, stops); err != nil {
		return 0, fmt.Errorf("insert stops: %w", err)
	}

	if err := s.createAssignment(ctx, db, sr, refs, def, move.ID); err != nil {
		return 0, err
	}

	return s.createOccurrences(ctx, db, sr, createOccurrencesParams{
		refs:       refs,
		candidates: candidates,
		def:        def,
		shipment:   shp,
		move:       move,
		stops:      stops,
		now:        now,
	})
}

func (s *DetentionSeed) buildStop(
	refs *detentionRefs,
	def detentionStopDef,
	sequence, now int64,
) (*shipment.Stop, error) {
	loc, ok := refs.locations[def.location]
	if !ok {
		return nil, fmt.Errorf("location %s not found", def.location)
	}

	apptStart := now - def.apptStartAgo*detentionMinute
	apptEnd := apptStart + def.apptWindow*detentionMinute
	arrival := apptStart + def.arrivalOffset*detentionMinute

	stop := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		BusinessUnitID:       refs.buID,
		OrganizationID:       refs.orgID,
		LocationID:           loc.ID,
		Type:                 def.stopType,
		ScheduleType:         def.scheduleType,
		Status:               shipmentvocab.StopStatusInTransit,
		Sequence:             sequence,
		ScheduledWindowStart: apptStart,
		ScheduledWindowEnd:   new(apptEnd),
		ActualArrival:        new(arrival),
	}

	if def.dwell > 0 {
		stop.Status = shipmentvocab.StopStatusCompleted
		stop.ActualDeparture = new(arrival + def.dwell*detentionMinute)
	}

	return stop, nil
}

func (s *DetentionSeed) createAssignment(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *detentionRefs,
	def detentionShipmentDef,
	moveID pulid.ID,
) error {
	workerID, ok := refs.workers[def.workerKey]
	if !ok {
		return fmt.Errorf("worker %s not found", def.workerKey)
	}
	tractorID, ok := refs.tractors[def.tractor]
	if !ok {
		return fmt.Errorf("tractor %s not found", def.tractor)
	}
	trailerID, ok := refs.trailers[def.trailer]
	if !ok {
		return fmt.Errorf("trailer %s not found", def.trailer)
	}

	status := shipmentvocab.AssignmentStatusCompleted
	if def.moveStatus != shipmentvocab.MoveStatusCompleted {
		status = shipmentvocab.AssignmentStatusInProgress
	}

	assignment := &shipment.Assignment{
		ID:              pulid.MustNew("asn_"),
		BusinessUnitID:  refs.buID,
		OrganizationID:  refs.orgID,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: &workerID,
		TractorID:       &tractorID,
		TrailerID:       &trailerID,
		Status:          status,
		AckStatus:       shipmentvocab.AssignmentAckPending,
	}
	if err := shipmentdb.Assignments.Create(ctx, db, assignment); err != nil {
		return fmt.Errorf("insert assignment: %w", err)
	}

	return nil
}

// -- Occurrences --

type createOccurrencesParams struct {
	refs       *detentionRefs
	candidates []*detentionpolicy.DetentionPolicy
	def        detentionShipmentDef
	shipment   *shipment.Shipment
	move       *shipment.ShipmentMove
	stops      []*shipment.Stop
	now        int64
}

// createOccurrences walks a shipment's stops exactly as the live sync does:
// resolve the governing policy per stop, freeze it, compute against immutable
// facts, and accumulate the day and shipment totals the caps key on.
func (s *DetentionSeed) createOccurrences(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	p createOccurrencesParams,
) (int, error) {
	payRate := p.refs.payRates[p.def.workerKey]
	dayAccrued := map[string]decimal.Decimal{}
	shipmentAccrued := decimal.Zero
	created := 0

	for i, stop := range p.stops {
		stopDef := p.def.stops[i]

		match := detentionpolicy.MatchInput{
			CustomerID:     p.shipment.CustomerID,
			LocationID:     stop.LocationID,
			ShipmentTypeID: p.shipment.ShipmentTypeID,
			ServiceTypeID:  p.shipment.ServiceTypeID,
			CommodityIDs:   []pulid.ID{p.refs.commodityID},
			StopType:       stop.Type,
			ScheduleType:   stop.ScheduleType,
			Timestamp:      *stop.ActualArrival,
		}

		resolution := detentionengine.ResolvePolicy(p.candidates, match)
		if !resolution.Found() {
			continue
		}

		policy := resolution.Policy
		accessorial, ok := p.refs.chargesByID[policy.AccessorialChargeID]
		if !ok {
			return created, fmt.Errorf(
				"policy %s references an accessorial the seed did not load", policy.Code)
		}

		snapshot := detentionpolicy.NewPolicySnapshot(detentionpolicy.SnapshotInput{
			Policy:          policy,
			StopType:        string(stop.Type),
			FreeMinutes:     policy.FreeMinutesForStopType(stop.Type),
			AccessorialCode: accessorial.Code,
			FlatRate:        accessorial.Amount,
			FlatRateUnit:    detentionTierUnit(accessorial.RateUnit),
			CapturedAt:      p.now,
		})

		input := detentionengine.ComputeInput{
			Snapshot:         snapshot,
			StopType:         stop.Type,
			ScheduleType:     stop.ScheduleType,
			ArrivedAt:        stop.ActualArrival,
			DepartedAt:       stop.ActualDeparture,
			AppointmentStart: new(stop.ScheduledWindowStart),
			AppointmentEnd:   stop.ScheduledWindowEnd,
			Now:              p.now,
			Location:         p.refs.timezone,
			DriverPayRate:    payRate,
			DayAccrued:       dayAccrued,
			ShipmentAccrued:  shipmentAccrued,
		}

		// The notice is placed relative to free-time expiry, which the engine
		// itself derives, so the first pass exists only to learn that instant.
		computed := detentionengine.Compute(input)
		var noticeSentAt *int64
		if stopDef.notice != nil && computed.Applicable {
			sentAt := computed.FreeTimeExpiresAt + stopDef.notice.offsetFromExpiry*detentionMinute
			noticeSentAt = &sentAt
			input.NoticeSentAt = noticeSentAt
			computed = detentionengine.Compute(input)
		}

		if !computed.Applicable {
			continue
		}

		for day, amount := range computed.DayAllocation {
			dayAccrued[day] = dayAccrued[day].Add(amount)
		}

		occurrence := buildDetentionOccurrence(p, stop, policy, snapshot, computed)
		if stopDef.notice != nil && stopDef.notice.delivery.IsFailure() {
			occurrence.NotificationStatus = detention.NotificationStatusFailed
		} else {
			occurrence.NoticeSentAt = noticeSentAt
		}

		if err := s.applyOutcome(occurrence, stopDef, p.refs, p.now); err != nil {
			return created, err
		}

		shipmentAccrued = shipmentAccrued.Add(occurrence.BillableAmount)

		if err := s.persistOccurrence(ctx, db, sr, persistOccurrenceParams{
			refs:         p.refs,
			occurrence:   occurrence,
			stopDef:      stopDef,
			policy:       policy,
			shipment:     p.shipment,
			location:     p.refs.locations[stopDef.location],
			customerCode: p.def.customer,
			noticeSent:   noticeSentAt,
			now:          p.now,
		}); err != nil {
			return created, err
		}

		created++
	}

	return created, nil
}

func buildDetentionOccurrence(
	p createOccurrencesParams,
	stop *shipment.Stop,
	policy *detentionpolicy.DetentionPolicy,
	snapshot detentionpolicy.PolicySnapshot,
	computed detentionengine.ComputeResult,
) *detention.DetentionOccurrence {
	policyID := policy.ID

	return &detention.DetentionOccurrence{
		ID:                 pulid.MustNew("dto_"),
		BusinessUnitID:     p.refs.buID,
		OrganizationID:     p.refs.orgID,
		ShipmentID:         p.shipment.ID,
		ShipmentMoveID:     p.move.ID,
		StopID:             stop.ID,
		CustomerID:         p.shipment.CustomerID,
		LocationID:         stop.LocationID,
		DetentionPolicyID:  &policyID,
		PolicySnapshot:     &snapshot,
		CalculationTrace:   computed.Trace,
		StopType:           stop.Type,
		ScheduleType:       stop.ScheduleType,
		AppointmentStart:   new(stop.ScheduledWindowStart),
		AppointmentEnd:     stop.ScheduledWindowEnd,
		ArrivedAt:          stop.ActualArrival,
		DepartedAt:         stop.ActualDeparture,
		ClockStartAt:       computed.ClockStartAt,
		ClockStopAt:        computed.ClockStopAt,
		FreeTimeExpiresAt:  computed.FreeTimeExpiresAt,
		NoticeDueAt:        computed.NoticeDueAt,
		NoticeDeadlineAt:   computed.NoticeDeadlineAt,
		IsOpen:             computed.IsOpen,
		ArrivedLate:        computed.ArrivedLate,
		LateByMinutes:      computed.LateByMinutes,
		FreeMinutesGranted: computed.FreeMinutesGranted,
		RawDwellMinutes:    computed.RawDwellMinutes,
		BillableMinutes:    computed.BillableMinutes,
		RoundedMinutes:     computed.RoundedMinutes,
		BillableUnits:      computed.BillableUnits,
		GrossAmount:        computed.GrossAmount,
		BillableAmount:     computed.BillableAmount,
		DriverPayMinutes:   computed.DriverPayMinutes,
		DriverPayAmount:    decimal.NewNullDecimal(computed.DriverPayAmount),
		NetMargin:          decimal.NewNullDecimal(computed.NetMargin),
		CapApplied:         computed.CapApplied,
		ConvertedToLayover: computed.ConvertedToLayover,
		Currency:           snapshot.Currency,
		Status:             computed.Status,
		NotificationStatus: computed.NotificationStatus,
		SuppressedByGate:   computed.SuppressedByGate,
		RequiresApproval:   computed.RequiresApproval,
	}
}

// applyOutcome walks the occurrence through the same lifecycle transitions the
// desk offers, using the domain methods so an invalid seeded transition fails
// loudly here rather than producing a record the UI cannot explain.
func (s *DetentionSeed) applyOutcome(
	occurrence *detention.DetentionOccurrence,
	def detentionStopDef,
	refs *detentionRefs,
	now int64,
) error {
	switch def.outcome {
	case detentionOutcomeNone:
		return nil
	case detentionOutcomeApprove:
		return occurrence.Approve(refs.adminID, now)
	case detentionOutcomeBill:
		if occurrence.Status == detention.OccurrenceStatusPending {
			if err := occurrence.Approve(refs.adminID, now); err != nil {
				return err
			}
		}
		if occurrence.Status != detention.OccurrenceStatusApproved {
			return fmt.Errorf("cannot bill an occurrence in status %s", occurrence.Status)
		}
		occurrence.Status = detention.OccurrenceStatusBilled
		return nil
	case detentionOutcomeWaive:
		return occurrence.Waive(def.waiverReason, def.note, refs.adminID, now)
	case detentionOutcomeDispute:
		return occurrence.Dispute(def.note, now)
	default:
		return fmt.Errorf("unknown detention outcome %q", def.outcome)
	}
}

type persistOccurrenceParams struct {
	refs         *detentionRefs
	occurrence   *detention.DetentionOccurrence
	stopDef      detentionStopDef
	policy       *detentionpolicy.DetentionPolicy
	shipment     *shipment.Shipment
	location     *location.Location
	customerCode string
	noticeSent   *int64
	now          int64
}

func (s *DetentionSeed) persistOccurrence(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	p persistOccurrenceParams,
) error {
	entries := s.buildEvidence(p)
	assessment := detentioncollectability.AssessCollectability(p.occurrence, entries)
	p.occurrence.CollectabilityScore = assessment.Score
	if len(entries) > 0 {
		p.occurrence.EvidenceHead = entries[len(entries)-1].Hash
	}

	if err := detentiondb.DetentionOccurrences.Create(ctx, db, p.occurrence); err != nil {
		return fmt.Errorf("insert detention occurrence: %w", err)
	}

	// The charge points back at the occurrence, so the occurrence has to exist
	// before it can be posted, and the back-reference lands as an update.
	if p.occurrence.Status == detention.OccurrenceStatusBilled {
		charge, err := s.createDetentionCharge(ctx, db, sr, p)
		if err != nil {
			return err
		}
		p.occurrence.AdditionalChargeID = &charge.ID

		occ := detentiondb.DetentionOccurrences
		if err = occ.Save(ctx, db, p.occurrence, occ.AdditionalChargeID); err != nil {
			return fmt.Errorf("link detention charge to occurrence: %w", err)
		}
	}

	if len(entries) > 0 {
		if err := detentionevidencedb.DetentionEvidences.CreateMany(ctx, db, entries); err != nil {
			return fmt.Errorf("insert detention evidence: %w", err)
		}
	}

	return s.createNotice(ctx, db, sr, p)
}

// createDetentionCharge posts the billed detention onto the shipment so the
// invoice and the occurrence agree on one number. A shipment carries one
// detention charge per accessorial, so a second billed stop adds to the row
// the first one created.
func (s *DetentionSeed) createDetentionCharge(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	p persistOccurrenceParams,
) (*shipment.AdditionalCharge, error) {
	ac := shipmentdb.AdditionalCharges
	existing, err := ac.AllTenants().Where(
		ac.ShipmentID.Eq(p.shipment.ID),
		ac.OrganizationID.Eq(p.refs.orgID),
		ac.BusinessUnitID.Eq(p.refs.buID),
		ac.AccessorialChargeID.Eq(p.policy.AccessorialChargeID),
		ac.IsDetention.IsTrue(),
	).Order(ac.CreatedAt.Asc(), ac.ID.Asc()).First(ctx, db)
	if err != nil && !errors.Is(err, keel.ErrNotFound) {
		return nil, fmt.Errorf("find detention additional charge: %w", err)
	}

	charge := existing
	if err == nil {
		charge.Amount = charge.Amount.Add(p.occurrence.BillableAmount)
		if err = ac.Save(ctx, db, charge, ac.Amount); err != nil {
			return nil, fmt.Errorf("extend detention additional charge: %w", err)
		}
	} else {
		charge = &shipment.AdditionalCharge{
			ID:                  pulid.MustNew("adc_"),
			BusinessUnitID:      p.refs.buID,
			OrganizationID:      p.refs.orgID,
			ShipmentID:          p.shipment.ID,
			AccessorialChargeID: p.policy.AccessorialChargeID,
			IsSystemGenerated:   true,
			IsDetention:         true,
			Method:              accessorialcharge.MethodFlat,
			Amount:              p.occurrence.BillableAmount,
			Unit:                1,
		}
		if err = shipmentdb.AdditionalCharges.Create(ctx, db, charge); err != nil {
			return nil, fmt.Errorf("insert detention additional charge: %w", err)
		}
	}

	sh := shipmentdb.Shipments
	_, err = sh.AllTenants().Where(
		sh.ID.Eq(p.shipment.ID),
		sh.OrganizationID.Eq(p.refs.orgID),
		sh.BusinessUnitID.Eq(p.refs.buID),
	).Update(
		ctx,
		db,
		sh.OtherChargeAmount.ToExpr(
			keel.NumericOf(sh.OtherChargeAmount.Coalesce(0)).
				Add(keel.Numeric(p.occurrence.BillableAmount)),
		),
		sh.TotalChargeAmount.ToExpr(
			keel.NumericOf(sh.TotalChargeAmount.Coalesce(0)).
				Add(keel.Numeric(p.occurrence.BillableAmount)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("apply detention charge to shipment total: %w", err)
	}

	return charge, nil
}

// -- Evidence --

type detentionEvidenceDef struct {
	kind       detentionevidence.EvidenceKind
	source     detentionevidence.EvidenceSource
	summary    string
	observedAt int64
	recordedBy *pulid.ID
	payload    map[string]any
}

//
//nolint:funlen // the claim file is assembled in the order the facts happened
func (s *DetentionSeed) buildEvidence(
	p persistOccurrenceParams,
) []*detentionevidence.DetentionEvidence {
	occ := p.occurrence
	facility := "the facility"
	if p.location != nil {
		facility = p.location.Name
	}
	admin := p.refs.adminID

	defs := make([]detentionEvidenceDef, 0, 8)

	if occ.AppointmentStart != nil {
		defs = append(defs, detentionEvidenceDef{
			kind:   detentionevidence.EvidenceKindAppointment,
			source: detentionevidence.EvidenceSourceSystem,
			summary: fmt.Sprintf(
				"Appointment booked at %s for %s",
				facility,
				formatDetentionStamp(*occ.AppointmentStart, p.refs.timezone),
			),
			observedAt: *occ.AppointmentStart - detentionDay,
			payload: map[string]any{
				"windowStart":  *occ.AppointmentStart,
				"windowEnd":    occ.AppointmentEnd,
				"scheduleType": string(occ.ScheduleType),
			},
		})
	}

	if occ.ArrivedAt != nil {
		defs = append(defs, detentionEvidenceDef{
			kind:       detentionevidence.EvidenceKindArrival,
			source:     p.stopDef.arrivalSource,
			summary:    fmt.Sprintf("Driver arrived at %s", facility),
			observedAt: *occ.ArrivedAt,
			payload: map[string]any{
				"capturedBy":  string(p.stopDef.arrivalSource),
				"arrivedLate": occ.ArrivedLate,
			},
		})
	}

	if p.noticeSent != nil && p.stopDef.notice != nil {
		defs = append(defs, detentionEvidenceDef{
			kind:       detentionevidence.EvidenceKindNotice,
			source:     detentionevidence.EvidenceSourceSystem,
			summary:    s.noticeEvidenceSummary(p),
			observedAt: *p.noticeSent,
			payload: map[string]any{
				"channel":        string(detentionnotice.NoticeChannelEmail),
				"deliveryStatus": string(p.stopDef.notice.delivery),
				"automatic":      p.stopDef.notice.automatic,
			},
		})
	}

	if p.stopDef.document != "" && occ.DepartedAt != nil {
		defs = append(defs, detentionEvidenceDef{
			kind:       detentionevidence.EvidenceKindDocument,
			source:     detentionevidence.EvidenceSourceDocument,
			summary:    p.stopDef.document,
			observedAt: *occ.DepartedAt,
		})
	}

	if occ.DepartedAt != nil {
		defs = append(defs, detentionEvidenceDef{
			kind:   detentionevidence.EvidenceKindDeparture,
			source: p.stopDef.departureSource,
			summary: fmt.Sprintf(
				"Driver departed %s after %s on site",
				facility,
				formatDetentionDuration(occ.RawDwellMinutes),
			),
			observedAt: *occ.DepartedAt,
			payload: map[string]any{
				"capturedBy":      string(p.stopDef.departureSource),
				"rawDwellMinutes": occ.RawDwellMinutes,
			},
		})
	}

	defs = append(defs, detentionEvidenceDef{
		kind:   detentionevidence.EvidenceKindRecalculated,
		source: detentionevidence.EvidenceSourceSystem,
		summary: fmt.Sprintf("Detention computed to %s %s (%s) under %s",
			occ.BillableAmount.StringFixed(2), occ.Currency, occ.Status, p.policy.Code),
		observedAt: p.now,
		payload: map[string]any{
			"policyCode":      p.policy.Code,
			"billableMinutes": occ.BillableMinutes,
			"roundedMinutes":  occ.RoundedMinutes,
			"capApplied":      string(occ.CapApplied),
		},
	})

	switch occ.Status {
	case detention.OccurrenceStatusApproved:
		defs = append(defs, detentionEvidenceDef{
			kind:   detentionevidence.EvidenceKindStatusChange,
			source: detentionevidence.EvidenceSourceManual,
			summary: "Approved for billing at " + occ.BillableAmount.StringFixed(
				2,
			) + " " + occ.Currency,
			observedAt: p.now,
			recordedBy: &admin,
		})
	case detention.OccurrenceStatusBilled:
		defs = append(defs, detentionEvidenceDef{
			kind:       detentionevidence.EvidenceKindStatusChange,
			source:     detentionevidence.EvidenceSourceSystem,
			summary:    "Billed on shipment " + p.shipment.ProNumber,
			observedAt: p.now,
			recordedBy: &admin,
		})
	case detention.OccurrenceStatusWaived:
		defs = append(defs, detentionEvidenceDef{
			kind:   detentionevidence.EvidenceKindWaiver,
			source: detentionevidence.EvidenceSourceManual,
			summary: fmt.Sprintf(
				"Waived %s %s (%s): %s",
				occ.WaivedAmount.StringFixed(
					2,
				),
				occ.Currency,
				p.stopDef.waiverReason,
				p.stopDef.note,
			),
			observedAt: p.now,
			recordedBy: &admin,
		})
	case detention.OccurrenceStatusDisputed:
		defs = append(defs, detentionEvidenceDef{
			kind:       detentionevidence.EvidenceKindDispute,
			source:     detentionevidence.EvidenceSourceManual,
			summary:    "Customer disputed the charge: " + p.stopDef.note,
			observedAt: p.now,
			recordedBy: &admin,
		})
	case detention.OccurrenceStatusAccruing, detention.OccurrenceStatusPending,
		detention.OccurrenceStatusNotBillable:
	}

	return sealDetentionEvidence(occ, defs)
}

// sealDetentionEvidence hash-chains the entries so the seeded claim file
// verifies exactly like one the engine wrote.
func sealDetentionEvidence(
	occ *detention.DetentionOccurrence,
	defs []detentionEvidenceDef,
) []*detentionevidence.DetentionEvidence {
	entries := make([]*detentionevidence.DetentionEvidence, 0, len(defs))
	prev := detentionevidence.GenesisHash

	for i, def := range defs {
		entry := &detentionevidence.DetentionEvidence{
			ID:                    pulid.MustNew("dte_"),
			BusinessUnitID:        occ.BusinessUnitID,
			OrganizationID:        occ.OrganizationID,
			DetentionOccurrenceID: occ.ID,
			Sequence:              int32(i),
			Kind:                  def.kind,
			Source:                def.source,
			Summary:               def.summary,
			ObservedAt:            def.observedAt,
			RecordedAt:            def.observedAt,
			RecordedByID:          def.recordedBy,
			Payload:               def.payload,
			PrevHash:              prev,
		}
		entry.Seal()
		prev = entry.Hash
		entries = append(entries, entry)
	}

	return entries
}

func (s *DetentionSeed) noticeEvidenceSummary(p persistOccurrenceParams) string {
	notice := p.stopDef.notice

	switch {
	case notice.delivery.IsFailure():
		return fmt.Sprintf("%s notice failed to send: %s", notice.kind, notice.failureReason)
	case p.occurrence.NotificationStatus == detention.NotificationStatusLate:
		return fmt.Sprintf("%s notice sent after the contractual deadline", notice.kind)
	default:
		return fmt.Sprintf("%s notice sent within the contractual window", notice.kind)
	}
}

// -- Notices --

func (s *DetentionSeed) createNotice(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	p persistOccurrenceParams,
) error {
	if p.stopDef.notice == nil || p.noticeSent == nil {
		return nil
	}

	def := p.stopDef.notice
	facility := ""
	if p.location != nil {
		facility = p.location.Name
	}

	content, err := detentionnoticerender.RenderBuiltInNotice(
		ctx,
		&detentionnoticerender.BuildNoticeParams{
			Occurrence:    p.occurrence,
			Kind:          def.kind,
			FacilityName:  facility,
			ShipmentRef:   p.shipment.ProNumber,
			ArrivalSource: p.stopDef.arrivalSource,
			Location:      p.refs.timezone,
		},
	)
	if err != nil {
		return fmt.Errorf("render detention notice: %w", err)
	}

	recipients := p.refs.recipients[p.customerCode]
	if len(recipients) == 0 {
		return fmt.Errorf("no notice recipients configured for customer %s", p.customerCode)
	}

	sentAt := *p.noticeSent
	notice := &detentionnotice.DetentionNotice{
		ID:                    pulid.MustNew("dtn_"),
		BusinessUnitID:        p.refs.buID,
		OrganizationID:        p.refs.orgID,
		DetentionOccurrenceID: p.occurrence.ID,
		Kind:                  def.kind,
		Channel:               detentionnotice.NoticeChannelEmail,
		DeliveryStatus:        def.delivery,
		Recipients:            recipients,
		Subject:               content.Subject,
		Body:                  content.Text,
		BodyHTML:              content.HTML,
		ScheduledFor:          sentAt - 5*detentionMinute,
		SentAt:                &sentAt,
		WasAutomatic:          def.automatic,
		QuotedFreeMinutes:     p.occurrence.FreeMinutesGranted,
		QuotedAmount:          decimal.NewNullDecimal(p.occurrence.BillableAmount),
		SatisfiesRequirement:  p.occurrence.NotificationStatus == detention.NotificationStatusSent,
	}

	if p.occurrence.PolicySnapshot != nil {
		notice.QuotedRate = decimal.NewNullDecimal(p.occurrence.PolicySnapshot.FlatRate)
	}
	if !def.automatic {
		notice.SentByID = &p.refs.adminID
	}

	switch {
	case def.delivery.IsFailure():
		notice.FailedAt = &sentAt
		notice.FailureReason = def.failureReason
		notice.SatisfiesRequirement = false
	case def.delivery == detentionnotice.NoticeDeliveryStatusOpened:
		notice.DeliveredAt = new(sentAt + 3*detentionMinute)
		notice.OpenedAt = new(sentAt + 11*detentionMinute)
		notice.ProviderMessageID = "seed-" + notice.ID.String()
	case def.delivery == detentionnotice.NoticeDeliveryStatusDelivered:
		notice.DeliveredAt = new(sentAt + 2*detentionMinute)
		notice.ProviderMessageID = "seed-" + notice.ID.String()
	}

	if err := detentionnoticedb.DetentionNotices.Create(ctx, db, notice); err != nil {
		return fmt.Errorf("insert detention notice: %w", err)
	}

	return nil
}

// -- Formatting --

// detentionTierUnit maps an accessorial's rate unit onto the tier vocabulary the
// snapshot speaks, mirroring how the service freezes a flat-rate policy.
func detentionTierUnit(unit accessorialcharge.RateUnit) detentionpolicy.TierRateUnit {
	switch unit {
	case accessorialcharge.RateUnitHour:
		return detentionpolicy.TierRateUnitHour
	case accessorialcharge.RateUnitDay:
		return detentionpolicy.TierRateUnitDay
	case accessorialcharge.RateUnitStop, accessorialcharge.RateUnitMile:
		return detentionpolicy.TierRateUnitFlat
	default:
		return detentionpolicy.TierRateUnitHour
	}
}

func formatDetentionStamp(ts int64, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return time.Unix(ts, 0).In(loc).Format("Mon 02 Jan 2006 15:04 MST")
}

func formatDetentionDuration(minutes int32) string {
	if minutes <= 0 {
		return "0 min"
	}
	if minutes < 60 {
		return fmt.Sprintf("%d min", minutes)
	}
	hours := minutes / 60
	rem := minutes % 60
	if rem == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, rem)
}
