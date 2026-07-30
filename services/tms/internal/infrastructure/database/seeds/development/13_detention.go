package development

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
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
	detentionProPattern = "SEED-DET-%"

	detentionMinute = int64(60)
	detentionDay    = int64(86400)
)

type DetentionSeed struct {
	seedhelpers.BaseSeed
}

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
func NewDetentionSeed() *DetentionSeed {
	seed := &DetentionSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"Detention",
		"1.0.0",
		"Creates detention policies, notice recipients, and worked detention occurrences with evidence chains",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)
	seed.SetDependencies(
		seedhelpers.SeedTestData,
		seedhelpers.SeedShipment,
		seedhelpers.SeedDriverPay,
	)
	return seed
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
	formulaID     pulid.ID
	recipients    map[string][]string
}

func (s *DetentionSeed) Run(ctx context.Context, tx bun.Tx) error {
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

			admin, err := sc.GetUserByUsername(ctx, "admin")
			if err != nil {
				return fmt.Errorf("get admin user: %w", err)
			}

			refs, err := s.loadRefs(ctx, tx, org, admin.ID)
			if err != nil {
				return err
			}

			policies, err := s.ensurePolicies(ctx, tx, sc, refs)
			if err != nil {
				return fmt.Errorf("ensure detention policies: %w", err)
			}

			if err = s.enablePolicyEngine(ctx, tx, refs, policies[detentionPolicyStandard]); err != nil {
				return fmt.Errorf("enable detention policy engine: %w", err)
			}

			if err = s.ensureNoticeRecipients(ctx, tx, sc, refs); err != nil {
				return fmt.Errorf("ensure notice recipients: %w", err)
			}

			if err = s.createDetentionBook(ctx, tx, sc, refs, policies); err != nil {
				return fmt.Errorf("create detention book: %w", err)
			}

			return nil
		},
	)
}

// -- Reference data --

//nolint:funlen // one linear pass over the fixtures every detention record needs
func (s *DetentionSeed) loadRefs(
	ctx context.Context,
	tx bun.Tx,
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

	var locs []location.Location
	if err := tx.NewSelect().Model(&locs).Column("id", "code", "name").
		Where("organization_id = ?", refs.orgID).Where("business_unit_id = ?", refs.buID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get locations: %w", err)
	}
	for i := range locs {
		refs.locations[locs[i].Code] = &locs[i]
	}

	var custs []customer.Customer
	if err := tx.NewSelect().Model(&custs).Column("id", "code", "name").
		Where("organization_id = ?", refs.orgID).Where("business_unit_id = ?", refs.buID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get customers: %w", err)
	}
	for i := range custs {
		refs.customers[custs[i].Code] = &custs[i]
	}

	var svcTypes []servicetype.ServiceType
	if err := tx.NewSelect().Model(&svcTypes).Column("id", "code").
		Where("organization_id = ?", refs.orgID).Where("business_unit_id = ?", refs.buID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get service types: %w", err)
	}
	for i := range svcTypes {
		refs.serviceTypes[svcTypes[i].Code] = svcTypes[i].ID
	}

	var shpTypes []shipmenttype.ShipmentType
	if err := tx.NewSelect().Model(&shpTypes).Column("id", "code").
		Where("organization_id = ?", refs.orgID).Where("business_unit_id = ?", refs.buID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get shipment types: %w", err)
	}
	for i := range shpTypes {
		refs.shipmentTypes[shpTypes[i].Code] = shpTypes[i].ID
	}

	var charges []accessorialcharge.AccessorialCharge
	if err := tx.NewSelect().Model(&charges).
		Column("id", "code", "description", "method", "rate_unit", "amount").
		Where("organization_id = ?", refs.orgID).Where("business_unit_id = ?", refs.buID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get accessorial charges: %w", err)
	}
	for i := range charges {
		refs.accessorials[charges[i].Code] = &charges[i]
		refs.chargesByID[charges[i].ID] = &charges[i]
	}
	for _, code := range []string{detentionAccessorial, layoverAccessorial} {
		if _, ok := refs.accessorials[code]; !ok {
			return nil, fmt.Errorf("accessorial charge %s not found", code)
		}
	}

	var tractors []tractor.Tractor
	if err := tx.NewSelect().Model(&tractors).Column("id", "code").
		Where("organization_id = ?", refs.orgID).Where("business_unit_id = ?", refs.buID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get tractors: %w", err)
	}
	for i := range tractors {
		refs.tractors[tractors[i].Code] = tractors[i].ID
	}

	var trailers []trailer.Trailer
	if err := tx.NewSelect().Model(&trailers).Column("id", "code").
		Where("organization_id = ?", refs.orgID).Where("business_unit_id = ?", refs.buID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get trailers: %w", err)
	}
	for i := range trailers {
		refs.trailers[trailers[i].Code] = trailers[i].ID
	}

	var workers []worker.Worker
	if err := tx.NewSelect().Model(&workers).Column("id", "email").
		Where("organization_id = ?", refs.orgID).Where("business_unit_id = ?", refs.buID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get workers: %w", err)
	}
	for i := range workers {
		refs.workers[workers[i].Email] = workers[i].ID
	}

	if err := s.loadDetentionPayRates(ctx, tx, refs); err != nil {
		return nil, err
	}

	var freight commodity.Commodity
	if err := tx.NewSelect().Model(&freight).Column("id").
		Where("organization_id = ?", refs.orgID).Where("business_unit_id = ?", refs.buID).
		Where("name = ?", "General Freight").Limit(1).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get general freight commodity: %w", err)
	}
	refs.commodityID = freight.ID

	var template formulatemplate.FormulaTemplate
	if err := tx.NewSelect().Model(&template).Column("id").
		Where("organization_id = ?", refs.orgID).Where("business_unit_id = ?", refs.buID).
		Where("name = ?", "Flat Rate").Limit(1).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get flat rate formula template: %w", err)
	}
	refs.formulaID = template.ID

	return refs, nil
}

// loadDetentionPayRates resolves the hourly detention rate each driver is paid,
// the same way the service does at runtime, so a seeded occurrence carries a
// truthful AP side and net margin rather than a fabricated one.
func (s *DetentionSeed) loadDetentionPayRates(
	ctx context.Context,
	tx bun.Tx,
	refs *detentionRefs,
) error {
	type workerDetentionRate struct {
		Email string          `bun:"email"`
		Rate  decimal.Decimal `bun:"rate"`
	}

	now := timeutils.NowUnix()
	rows := make([]workerDetentionRate, 0, 8)

	err := tx.NewSelect().
		Model((*driverpay.WorkerPayAssignment)(nil)).
		ColumnExpr("w.email AS email").
		ColumnExpr("dppc.rate AS rate").
		Join("JOIN workers AS w ON w.id = wpa.worker_id AND w.organization_id = wpa.organization_id").
		Join("JOIN driver_pay_profile_components AS dppc ON dppc.pay_profile_id = wpa.pay_profile_id"+
			" AND dppc.organization_id = wpa.organization_id").
		Where("wpa.organization_id = ?", refs.orgID).
		Where("wpa.business_unit_id = ?", refs.buID).
		Where("wpa.effective_from <= ?", now).
		Where("wpa.effective_to IS NULL OR wpa.effective_to >= ?", now).
		Where("dppc.kind = ?", driverpay.ComponentKindDetention).
		Where("dppc.is_active = TRUE").
		Scan(ctx, &rows)
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
	policy *detention.DetentionPolicy
	tiers  []*detention.DetentionPolicyTier
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
			policy: &detention.DetentionPolicy{
				ID:                          pulid.MustNew("dtp_"),
				BusinessUnitID:              refs.buID,
				OrganizationID:              refs.orgID,
				Name:                        "Standard detention terms",
				Code:                        detentionPolicyStandard,
				Description:                 "Two hours free, billed in quarter hours at the detention accessorial rate. Applies wherever no negotiated term is on file.",
				Status:                      detention.PolicyStatusActive,
				IsOrgDefault:                true,
				Priority:                    0,
				ClockStartBasis:             detention.ClockStartBasisLaterOfArrivalOrAppointment,
				LateArrivalRule:             detention.LateArrivalRuleNoEffect,
				LateArrivalGraceMinutes:     15,
				BillingFreeMinutes:          120,
				PayFreeMinutes:              new(int32(120)),
				MinimumBillableMinutes:      15,
				BillingIncrementMinutes:     15,
				RoundingMode:                detention.RoundingModeUp,
				RateSource:                  detention.RateSourceAccessorial,
				AccessorialChargeID:         det.ID,
				MaxChargePerDay:             decimal.NewNullDecimal(decimal.NewFromInt(900)),
				DayBoundaryMode:             detention.CapScopePerDay,
				NotificationRequirement:     detention.NotificationRequirementAdvisory,
				NotificationLeadMinutes:     30,
				NotificationDeadlineMinutes: 0,
				UnnotifiedBehavior:          detention.UnnotifiedBehaviorBill,
				SendDepartureSummary:        true,
				RequireApprovalOverAmount:   decimal.NewNullDecimal(decimal.NewFromInt(750)),
				AutoApproveUnderAmount:      decimal.NewNullDecimal(decimal.NewFromInt(150)),
				Currency:                    "USD",
				Comments:                    "Fallback terms. Anything negotiated with a customer or facility outranks this policy on specificity.",
			},
		},
		{
			policy: &detention.DetentionPolicy{
				ID:                          pulid.MustNew("dtp_"),
				BusinessUnitID:              refs.buID,
				OrganizationID:              refs.orgID,
				Name:                        "Acme Manufacturing — negotiated detention",
				Code:                        detentionPolicyAcme,
				Description:                 "Graduated ladder with split pickup and delivery free time. Written notice is a condition of billing.",
				Status:                      detention.PolicyStatusActive,
				Priority:                    10,
				CustomerID:                  &acmeID,
				ClockStartBasis:             detention.ClockStartBasisLaterOfArrivalOrAppointment,
				LateArrivalRule:             detention.LateArrivalRuleReduceFreeTime,
				LateArrivalGraceMinutes:     15,
				BillingFreeMinutes:          120,
				PickupFreeMinutes:           new(int32(90)),
				DeliveryFreeMinutes:         new(int32(150)),
				PayFreeMinutes:              new(int32(60)),
				MinimumBillableMinutes:      30,
				BillingIncrementMinutes:     15,
				RoundingMode:                detention.RoundingModeUp,
				RateSource:                  detention.RateSourceTiers,
				AccessorialChargeID:         det.ID,
				MaxChargePerStop:            decimal.NewNullDecimal(decimal.NewFromInt(1200)),
				MaxChargePerShipment:        decimal.NewNullDecimal(decimal.NewFromInt(2000)),
				DayBoundaryMode:             detention.CapScopePerStop,
				NotificationRequirement:     detention.NotificationRequirementRequired,
				NotificationLeadMinutes:     30,
				NotificationDeadlineMinutes: 60,
				UnnotifiedBehavior:          detention.UnnotifiedBehaviorFlag,
				AutoSendNotice:              true,
				SendDepartureSummary:        true,
				RequireApprovalOverAmount:   decimal.NewNullDecimal(decimal.NewFromInt(1000)),
				AutoApproveUnderAmount:      decimal.NewNullDecimal(decimal.NewFromInt(250)),
				Currency:                    "USD",
				Comments:                    "Master agreement §7. Notice inside 60 minutes of free-time expiry is a condition precedent to the charge.",
			},
			tiers: []*detention.DetentionPolicyTier{
				{
					FromMinute: 0, ToMinute: new(int32(240)),
					Rate: decimal.NewFromInt(60), RateUnit: detention.TierRateUnitHour,
					Label: "First 4 hours", SortOrder: 0,
				},
				{
					FromMinute: 240, ToMinute: new(int32(720)),
					Rate: decimal.NewFromInt(85), RateUnit: detention.TierRateUnitHour,
					Label: "Hours 4 through 12", SortOrder: 1,
				},
				{
					FromMinute: 720,
					Rate:       decimal.NewFromInt(750), RateUnit: detention.TierRateUnitDay,
					Label: "Beyond 12 hours", SortOrder: 2,
				},
			},
		},
		{
			policy: &detention.DetentionPolicy{
				ID:                          pulid.MustNew("dtp_"),
				BusinessUnitID:              refs.buID,
				OrganizationID:              refs.orgID,
				Name:                        "Phoenix Cold Storage — facility terms",
				Code:                        detentionPolicyCold,
				Description:                 "One hour free from arrival, capped daily, converting to layover after 24 hours on site. Notice is required or the charge is suppressed.",
				Status:                      detention.PolicyStatusActive,
				Priority:                    20,
				LocationID:                  &coldID,
				ClockStartBasis:             detention.ClockStartBasisArrival,
				LateArrivalRule:             detention.LateArrivalRuleNoEffect,
				BillingFreeMinutes:          60,
				PayFreeMinutes:              new(int32(60)),
				MinimumBillableMinutes:      30,
				BillingIncrementMinutes:     30,
				RoundingMode:                detention.RoundingModeNearest,
				RateSource:                  detention.RateSourceAccessorial,
				AccessorialChargeID:         det.ID,
				MaxChargePerDay:             decimal.NewNullDecimal(decimal.NewFromInt(600)),
				DayBoundaryMode:             detention.CapScopePerDay,
				ConvertToLayoverAtMinutes:   new(int32(1440)),
				LayoverAccessorialChargeID:  &layoverCharge,
				NotificationRequirement:     detention.NotificationRequirementRequired,
				NotificationLeadMinutes:     30,
				NotificationDeadlineMinutes: 30,
				UnnotifiedBehavior:          detention.UnnotifiedBehaviorSuppress,
				SendDepartureSummary:        true,
				Currency:                    "USD",
				Comments:                    "Facility rider. The dock refuses charges without a timestamped notice, so an unnotified stop is not worth billing.",
			},
		},
		{
			policy: &detention.DetentionPolicy{
				ID:                        pulid.MustNew("dtp_"),
				BusinessUnitID:            refs.buID,
				OrganizationID:            refs.orgID,
				Name:                      "GlobalTrade drayage — port pickups",
				Code:                      detentionPolicyDray,
				Description:               "Four hours free measured from the appointment. Arriving late forfeits detention entirely.",
				Status:                    detention.PolicyStatusActive,
				Priority:                  15,
				CustomerID:                &glblID,
				StopTypes:                 []shipment.StopType{shipment.StopTypePickup},
				AppointmentStopsOnly:      true,
				ClockStartBasis:           detention.ClockStartBasisAppointment,
				LateArrivalRule:           detention.LateArrivalRuleForfeit,
				LateArrivalGraceMinutes:   30,
				BillingFreeMinutes:        240,
				PayFreeMinutes:            new(int32(180)),
				MinimumBillableMinutes:    60,
				BillingIncrementMinutes:   60,
				RoundingMode:              detention.RoundingModeUp,
				RateSource:                detention.RateSourceAccessorial,
				AccessorialChargeID:       det.ID,
				MaxBillableMinutesPerStop: new(int32(720)),
				DayBoundaryMode:           detention.CapScopePerStop,
				NotificationRequirement:   detention.NotificationRequirementNone,
				UnnotifiedBehavior:        detention.UnnotifiedBehaviorBill,
				Currency:                  "USD",
				Comments:                  "Port terminal rider. The appointment is the clock; a missed appointment is the carrier's cost.",
			},
		},
		{
			policy: &detention.DetentionPolicy{
				ID:                      pulid.MustNew("dtp_"),
				BusinessUnitID:          refs.buID,
				OrganizationID:          refs.orgID,
				Name:                    "2025 legacy detention terms",
				Code:                    detentionPolicyLegacy,
				Description:             "Superseded terms retained so historical occurrences still explain themselves.",
				Status:                  detention.PolicyStatusInactive,
				Priority:                0,
				EffectiveStartDate:      new(now - 455*detentionDay),
				EffectiveEndDate:        new(now - 90*detentionDay),
				ClockStartBasis:         detention.ClockStartBasisArrival,
				LateArrivalRule:         detention.LateArrivalRuleNoEffect,
				BillingFreeMinutes:      180,
				PayFreeMinutes:          new(int32(180)),
				BillingIncrementMinutes: 60,
				RoundingMode:            detention.RoundingModeDown,
				RateSource:              detention.RateSourceAccessorial,
				AccessorialChargeID:     det.ID,
				DayBoundaryMode:         detention.CapScopePerStop,
				NotificationRequirement: detention.NotificationRequirementNone,
				UnnotifiedBehavior:      detention.UnnotifiedBehaviorBill,
				Currency:                "USD",
				Comments:                "Expired. Kept for audit of anything billed before the 2026 renegotiation.",
			},
		},
		{
			policy: &detention.DetentionPolicy{
				ID:                          pulid.MustNew("dtp_"),
				BusinessUnitID:              refs.buID,
				OrganizationID:              refs.orgID,
				Name:                        "2027 proposed detention terms",
				Code:                        detentionPolicyProposed,
				Description:                 "Draft ladder for the 2027 rate cycle. Backtest it against the last quarter before promoting it.",
				Status:                      detention.PolicyStatusDraft,
				Priority:                    5,
				ClockStartBasis:             detention.ClockStartBasisLaterOfArrivalOrAppointment,
				LateArrivalRule:             detention.LateArrivalRuleReduceFreeTime,
				LateArrivalGraceMinutes:     15,
				BillingFreeMinutes:          60,
				PayFreeMinutes:              new(int32(60)),
				MinimumBillableMinutes:      15,
				BillingIncrementMinutes:     15,
				RoundingMode:                detention.RoundingModeUp,
				RateSource:                  detention.RateSourceTiers,
				AccessorialChargeID:         det.ID,
				MaxChargePerStop:            decimal.NewNullDecimal(decimal.NewFromInt(1500)),
				DayBoundaryMode:             detention.CapScopePerStop,
				NotificationRequirement:     detention.NotificationRequirementRequired,
				NotificationLeadMinutes:     30,
				NotificationDeadlineMinutes: 30,
				UnnotifiedBehavior:          detention.UnnotifiedBehaviorFlag,
				AutoSendNotice:              true,
				Currency:                    "USD",
				Comments:                    "Draft. Tighter free time and a steeper second tier; run the backtest before making it active.",
			},
			tiers: []*detention.DetentionPolicyTier{
				{
					FromMinute: 0, ToMinute: new(int32(180)),
					Rate: decimal.NewFromInt(80), RateUnit: detention.TierRateUnitHour,
					Label: "First 3 hours", SortOrder: 0,
				},
				{
					FromMinute: 180,
					Rate:       decimal.NewFromInt(110), RateUnit: detention.TierRateUnitHour,
					Label: "Beyond 3 hours", SortOrder: 1,
				},
			},
		},
	}
}

func (s *DetentionSeed) ensurePolicies(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *detentionRefs,
) (map[string]*detention.DetentionPolicy, error) {
	policies := make(map[string]*detention.DetentionPolicy, 6)
	created := 0

	for _, def := range s.policyDefs(refs) {
		existing, err := s.findPolicy(ctx, tx, refs, def.policy.Code)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			policies[existing.Code] = existing
			continue
		}

		if _, err = tx.NewInsert().Model(def.policy).Exec(ctx); err != nil {
			return nil, fmt.Errorf("insert detention policy %s: %w", def.policy.Code, err)
		}
		if err = sc.TrackCreated(ctx, "detention_policies", def.policy.ID, s.Name()); err != nil {
			return nil, fmt.Errorf("track detention policy: %w", err)
		}

		if len(def.tiers) > 0 {
			for _, tier := range def.tiers {
				tier.ID = pulid.MustNew("dpt_")
				tier.BusinessUnitID = refs.buID
				tier.OrganizationID = refs.orgID
				tier.DetentionPolicyID = def.policy.ID
			}
			if _, err = tx.NewInsert().Model(&def.tiers).Exec(ctx); err != nil {
				return nil, fmt.Errorf("insert tiers for %s: %w", def.policy.Code, err)
			}
			for _, tier := range def.tiers {
				if err = sc.TrackCreated(ctx, "detention_policy_tiers", tier.ID, s.Name()); err != nil {
					return nil, fmt.Errorf("track detention policy tier: %w", err)
				}
			}
			def.policy.Tiers = def.tiers
		}

		policies[def.policy.Code] = def.policy
		created++
	}

	if created > 0 {
		seedhelpers.LogSuccess(
			"Created detention policy library",
			fmt.Sprintf("- Created %d policies covering default, customer, facility, and draft terms", created),
		)
	}

	return policies, nil
}

func (s *DetentionSeed) findPolicy(
	ctx context.Context,
	tx bun.Tx,
	refs *detentionRefs,
	code string,
) (*detention.DetentionPolicy, error) {
	policy := new(detention.DetentionPolicy)
	err := tx.NewSelect().Model(policy).
		Where("organization_id = ?", refs.orgID).
		Where("business_unit_id = ?", refs.buID).
		Where("code = ?", code).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get detention policy %s: %w", code, err)
	}

	if policy.RateSource != detention.RateSourceTiers {
		return policy, nil
	}

	tiers := make([]*detention.DetentionPolicyTier, 0, 4)
	if err = tx.NewSelect().Model(&tiers).
		Where("organization_id = ?", refs.orgID).
		Where("business_unit_id = ?", refs.buID).
		Where("detention_policy_id = ?", policy.ID).
		Order("from_minute ASC").
		Scan(ctx); err != nil {
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
	tx bun.Tx,
	refs *detentionRefs,
	defaultPolicy *detention.DetentionPolicy,
) error {
	if defaultPolicy == nil {
		return errors.New("default detention policy is missing")
	}

	_, err := tx.NewUpdate().
		Model((*tenant.ShipmentControl)(nil)).
		Set("track_detention_time = TRUE").
		Set("use_detention_policy_engine = TRUE").
		Set("default_detention_policy_id = ?", defaultPolicy.ID).
		Set("detention_charge_id = ?", refs.accessorials[detentionAccessorial].ID).
		Set("updated_at = ?", timeutils.NowUnix()).
		Where("organization_id = ?", refs.orgID).
		Where("business_unit_id = ?", refs.buID).
		Exec(ctx)
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
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
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

		existing := new(customer.CustomerEmailProfile)
		err := tx.NewSelect().Model(existing).
			Where("organization_id = ?", refs.orgID).
			Where("business_unit_id = ?", refs.buID).
			Where("customer_id = ?", cust.ID).
			Limit(1).
			Scan(ctx)
		switch {
		case err == nil && existing.ToRecipients != "":
			continue
		case err == nil:
			// A profile with no recipients is worse than none: the notice
			// pipeline finds it, resolves nothing, and refuses to send.
			if _, err = tx.NewUpdate().
				Model((*customer.CustomerEmailProfile)(nil)).
				Set("to_recipients = ?", recipients[0]+","+recipients[1]).
				Set("cc_recipients = ?", "ap@"+def.domain).
				Set("from_email = ?", "billing@trenova.example.com").
				Set("updated_at = ?", timeutils.NowUnix()).
				Where("id = ?", existing.ID).
				Exec(ctx); err != nil {
				return fmt.Errorf("fill email profile for %s: %w", def.customer, err)
			}
			filled++
			continue
		case !errors.Is(err, sql.ErrNoRows):
			return fmt.Errorf("check email profile for %s: %w", def.customer, err)
		}

		profile := &customer.CustomerEmailProfile{
			ID:             pulid.MustNew("cem_"),
			BusinessUnitID: refs.buID,
			OrganizationID: refs.orgID,
			CustomerID:     cust.ID,
			Subject:        "Shipment documentation",
			Comment:        "Detention notices and rate confirmations for " + cust.Name,
			FromEmail:      "billing@trenova.example.com",
			ToRecipients:   recipients[0] + "," + recipients[1],
			CCRecipients:   "ap@" + def.domain,
			AttachmentName: "shipment-documents.pdf",
			ReadReceipt:    true,
		}
		if _, err = tx.NewInsert().Model(profile).Exec(ctx); err != nil {
			return fmt.Errorf("insert email profile for %s: %w", def.customer, err)
		}
		if err = sc.TrackCreated(ctx, "customer_email_profiles", profile.ID, s.Name()); err != nil {
			return fmt.Errorf("track customer email profile: %w", err)
		}
		created++
	}

	if created > 0 || filled > 0 {
		seedhelpers.LogSuccess(
			"Created detention notice recipients",
			fmt.Sprintf("- Created %d customer email profiles, filled %d empty ones",
				created, filled),
		)
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
	kind             detention.NoticeKind
	delivery         detention.NoticeDeliveryStatus
	failureReason    string
	automatic        bool
}

type detentionStopDef struct {
	location        string
	stopType        shipment.StopType
	scheduleType    shipment.StopScheduleType
	apptStartAgo    int64
	apptWindow      int64
	arrivalOffset   int64
	dwell           int64
	arrivalSource   detention.EvidenceSource
	departureSource detention.EvidenceSource
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
	status     shipment.Status
	moveStatus shipment.MoveStatus
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
			service: "STD", shipType: "FTL", status: shipment.StatusInTransit,
			moveStatus: shipment.MoveStatusInTransit,
			revenue:    2850, pieces: 24, weight: 39000,
			workerKey: payWorkerJane, tractor: "TRC-003", trailer: "TRL-001",
			stops: []detentionStopDef{
				{
					location: "TERM-LA", stopType: shipment.StopTypePickup,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 26 * 60, apptWindow: 120, arrivalOffset: 10, dwell: 55,
					arrivalSource:   detention.EvidenceSourceGeofence,
					departureSource: detention.EvidenceSourceGeofence,
				},
				{
					location: "DC-CHI", stopType: shipment.StopTypeDelivery,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 240, apptWindow: 120, arrivalOffset: 15, dwell: 0,
					arrivalSource: detention.EvidenceSourceTelematics,
					notice: &detentionNoticeDef{
						offsetFromExpiry: -20, kind: detention.NoticeKindWarning,
						delivery: detention.NoticeDeliveryStatusOpened, automatic: true,
					},
				},
			},
		},
		{
			pro: "SEED-DET-002", bol: "BOL-2026-0202", customer: "FRSH",
			service: "TEMP", shipType: "REEF", status: shipment.StatusDelayed,
			moveStatus: shipment.MoveStatusInTransit,
			revenue:    3300, pieces: 20, weight: 36000,
			workerKey: payWorkerSarah, tractor: "TRC-002", trailer: "TRL-002",
			stops: []detentionStopDef{
				{
					location: "WH-DAL", stopType: shipment.StopTypePickup,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 30 * 60, apptWindow: 120, arrivalOffset: 5, dwell: 50,
					arrivalSource:   detention.EvidenceSourceDriverApp,
					departureSource: detention.EvidenceSourceDriverApp,
				},
				{
					location: "COLD-PHX", stopType: shipment.StopTypeDelivery,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 300, apptWindow: 60, arrivalOffset: 20, dwell: 0,
					arrivalSource: detention.EvidenceSourceGeofence,
				},
			},
		},
		{
			pro: "SEED-DET-003", bol: "BOL-2026-0203", customer: "PEAK",
			service: "STD", shipType: "FTL", status: shipment.StatusCompleted,
			moveStatus: shipment.MoveStatusCompleted,
			revenue:    3150, pieces: 26, weight: 41000,
			workerKey: payWorkerSarah, tractor: "TRC-001", trailer: "TRL-003",
			stops: []detentionStopDef{
				{
					location: "DC-CHI", stopType: shipment.StopTypePickup,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 4 * 24 * 60, apptWindow: 120, arrivalOffset: 0, dwell: 200,
					arrivalSource:   detention.EvidenceSourceTelematics,
					departureSource: detention.EvidenceSourceTelematics,
					document:        "Signed BOL captured at the dock office with in and out times",
				},
				{
					location: "WH-DAL", stopType: shipment.StopTypeDelivery,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 3 * 24 * 60, apptWindow: 120, arrivalOffset: 20, dwell: 480,
					arrivalSource:   detention.EvidenceSourceGeofence,
					departureSource: detention.EvidenceSourceGeofence,
					document:        "Signed POD showing the receiver's own arrival and release stamps",
					notice: &detentionNoticeDef{
						offsetFromExpiry: -15, kind: detention.NoticeKindStarted,
						delivery: detention.NoticeDeliveryStatusDelivered,
					},
					outcome: detentionOutcomeApprove,
				},
			},
		},
		{
			pro: "SEED-DET-004", bol: "BOL-2026-0204", customer: "ACME",
			service: "STD", shipType: "FTL", status: shipment.StatusReadyToInvoice,
			moveStatus: shipment.MoveStatusCompleted,
			revenue:    4200, pieces: 30, weight: 43000,
			workerKey: payWorkerJane, tractor: "TRC-003", trailer: "TRL-004",
			stops: []detentionStopDef{
				{
					location: "TERM-LA", stopType: shipment.StopTypePickup,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 9 * 24 * 60, apptWindow: 120, arrivalOffset: -10, dwell: 55,
					arrivalSource:   detention.EvidenceSourceGeofence,
					departureSource: detention.EvidenceSourceGeofence,
				},
				{
					location: "CUST-DEN", stopType: shipment.StopTypeDelivery,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 8 * 24 * 60, apptWindow: 120, arrivalOffset: 5, dwell: 840,
					arrivalSource:   detention.EvidenceSourceTelematics,
					departureSource: detention.EvidenceSourceTelematics,
					document:        "Gate log printout from the receiver covering both timestamps",
					notice: &detentionNoticeDef{
						offsetFromExpiry: 90, kind: detention.NoticeKindStarted,
						delivery: detention.NoticeDeliveryStatusDelivered, automatic: true,
					},
					outcome: detentionOutcomeBill,
				},
			},
		},
		{
			pro: "SEED-DET-005", bol: "BOL-2026-0205", customer: "GLBL",
			service: "STD", shipType: "DRAY", status: shipment.StatusCompleted,
			moveStatus: shipment.MoveStatusCompleted,
			revenue:    1950, pieces: 18, weight: 34000,
			workerKey: payWorkerMike, tractor: "TRC-101", trailer: "TRL-005",
			stops: []detentionStopDef{
				{
					location: "TERM-LA", stopType: shipment.StopTypePickup,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 6 * 24 * 60, apptWindow: 60, arrivalOffset: 150, dwell: 300,
					arrivalSource:   detention.EvidenceSourceTelematics,
					departureSource: detention.EvidenceSourceTelematics,
				},
				{
					location: "CUST-DEN", stopType: shipment.StopTypeDelivery,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 5 * 24 * 60, apptWindow: 120, arrivalOffset: 10, dwell: 100,
					arrivalSource:   detention.EvidenceSourceDriverApp,
					departureSource: detention.EvidenceSourceDriverApp,
				},
			},
		},
		{
			pro: "SEED-DET-006", bol: "BOL-2026-0206", customer: "SUNB",
			service: "WHG", shipType: "PART", status: shipment.StatusReadyToInvoice,
			moveStatus: shipment.MoveStatusCompleted,
			revenue:    3600, pieces: 16, weight: 32000,
			workerKey: payWorkerSarah, tractor: "TRC-002", trailer: "TRL-001",
			stops: []detentionStopDef{
				{
					location: "WH-DAL", stopType: shipment.StopTypePickup,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 14 * 24 * 60, apptWindow: 120, arrivalOffset: 0, dwell: 70,
					arrivalSource:   detention.EvidenceSourceDriverApp,
					departureSource: detention.EvidenceSourceDriverApp,
				},
				{
					location: "DC-CHI", stopType: shipment.StopTypeDelivery,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 13 * 24 * 60, apptWindow: 120, arrivalOffset: 30, dwell: 1800,
					arrivalSource:   detention.EvidenceSourceGeofence,
					departureSource: detention.EvidenceSourceGeofence,
					notice: &detentionNoticeDef{
						offsetFromExpiry: -30, kind: detention.NoticeKindStarted,
						delivery: detention.NoticeDeliveryStatusDelivered,
					},
					outcome: detentionOutcomeDispute,
					note:    "Receiver disputes the overnight hold, claiming the trailer was dropped and the tractor released at 19:00.",
				},
			},
		},
		{
			pro: "SEED-DET-007", bol: "BOL-2026-0207", customer: "RNGE",
			service: "STD", shipType: "FTL", status: shipment.StatusCompleted,
			moveStatus: shipment.MoveStatusCompleted,
			revenue:    2450, pieces: 22, weight: 37000,
			workerKey: payWorkerJane, tractor: "TRC-004", trailer: "TRL-002",
			stops: []detentionStopDef{
				{
					location: "DC-CHI", stopType: shipment.StopTypePickup,
					scheduleType: shipment.StopScheduleTypeOpen,
					apptStartAgo: 11 * 24 * 60, apptWindow: 240, arrivalOffset: 30, dwell: 65,
					arrivalSource:   detention.EvidenceSourceManual,
					departureSource: detention.EvidenceSourceManual,
				},
				{
					location: "CUST-DEN", stopType: shipment.StopTypeDelivery,
					scheduleType: shipment.StopScheduleTypeOpen,
					apptStartAgo: 10 * 24 * 60, apptWindow: 240, arrivalOffset: 45, dwell: 240,
					arrivalSource:   detention.EvidenceSourceManual,
					departureSource: detention.EvidenceSourceManual,
				},
			},
		},
		{
			pro: "SEED-DET-008", bol: "BOL-2026-0208", customer: "FRSH",
			service: "TEMP", shipType: "REEF", status: shipment.StatusReadyToInvoice,
			moveStatus: shipment.MoveStatusCompleted,
			revenue:    5100, pieces: 28, weight: 42000,
			workerKey: payWorkerSarah, tractor: "TRC-002", trailer: "TRL-003",
			stops: []detentionStopDef{
				{
					location: "WH-DAL", stopType: shipment.StopTypePickup,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 20 * 24 * 60, apptWindow: 120, arrivalOffset: 15, dwell: 80,
					arrivalSource:   detention.EvidenceSourceGeofence,
					departureSource: detention.EvidenceSourceGeofence,
				},
				{
					location: "COLD-PHX", stopType: shipment.StopTypeDelivery,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 19 * 24 * 60, apptWindow: 60, arrivalOffset: 5, dwell: 2400,
					arrivalSource:   detention.EvidenceSourceGeofence,
					departureSource: detention.EvidenceSourceGeofence,
					document:        "Cold storage gate ticket showing the trailer sat two nights",
					notice: &detentionNoticeDef{
						offsetFromExpiry: -25, kind: detention.NoticeKindStarted,
						delivery: detention.NoticeDeliveryStatusOpened,
					},
					outcome: detentionOutcomeBill,
				},
			},
		},
		{
			pro: "SEED-DET-009", bol: "BOL-2026-0209", customer: "PEAK",
			service: "STD", shipType: "FTL", status: shipment.StatusCompleted,
			moveStatus: shipment.MoveStatusCompleted,
			revenue:    2700, pieces: 20, weight: 35000,
			workerKey: payWorkerJane, tractor: "TRC-003", trailer: "TRL-004",
			stops: []detentionStopDef{
				{
					location: "DC-CHI", stopType: shipment.StopTypePickup,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 7 * 24 * 60, apptWindow: 120, arrivalOffset: 0, dwell: 60,
					arrivalSource:   detention.EvidenceSourceDriverApp,
					departureSource: detention.EvidenceSourceDriverApp,
				},
				{
					location: "WH-DAL", stopType: shipment.StopTypeDelivery,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 6 * 24 * 60, apptWindow: 120, arrivalOffset: 10, dwell: 300,
					arrivalSource:   detention.EvidenceSourceTelematics,
					departureSource: detention.EvidenceSourceTelematics,
					notice: &detentionNoticeDef{
						offsetFromExpiry: -10, kind: detention.NoticeKindStarted,
						delivery: detention.NoticeDeliveryStatusDelivered,
					},
					outcome:      detentionOutcomeWaive,
					waiverReason: detention.WaiverReasonCustomerGoodwill,
					note:         "Waived by the account manager ahead of the Q3 bid; the delay was one shift short-staffed, not a pattern.",
				},
			},
		},
		{
			pro: "SEED-DET-010", bol: "BOL-2026-0210", customer: "ACME",
			service: "STD", shipType: "FTL", status: shipment.StatusCompleted,
			moveStatus: shipment.MoveStatusCompleted,
			revenue:    2200, pieces: 18, weight: 31000,
			workerKey: payWorkerJohn, tractor: "TRC-001", trailer: "TRL-005",
			stops: []detentionStopDef{
				{
					location: "TERM-LA", stopType: shipment.StopTypePickup,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 3 * 24 * 60, apptWindow: 120, arrivalOffset: 5, dwell: 45,
					arrivalSource:   detention.EvidenceSourceGeofence,
					departureSource: detention.EvidenceSourceGeofence,
				},
				{
					location: "DC-CHI", stopType: shipment.StopTypeDelivery,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 2 * 24 * 60, apptWindow: 120, arrivalOffset: 0, dwell: 120,
					arrivalSource:   detention.EvidenceSourceGeofence,
					departureSource: detention.EvidenceSourceGeofence,
					document:        "Signed POD released inside the appointment window",
				},
			},
		},
		{
			pro: "SEED-DET-011", bol: "BOL-2026-0211", customer: "GLBL",
			service: "STD", shipType: "FTL", status: shipment.StatusReadyToInvoice,
			moveStatus: shipment.MoveStatusCompleted,
			revenue:    3050, pieces: 24, weight: 38000,
			workerKey: payWorkerMike, tractor: "TRC-101", trailer: "TRL-002",
			stops: []detentionStopDef{
				{
					location: "CUST-DEN", stopType: shipment.StopTypePickup,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 16 * 24 * 60, apptWindow: 120, arrivalOffset: 10, dwell: 75,
					arrivalSource:   detention.EvidenceSourceDriverApp,
					departureSource: detention.EvidenceSourceDriverApp,
				},
				{
					location: "WH-DAL", stopType: shipment.StopTypeDelivery,
					scheduleType: shipment.StopScheduleTypeAppointment,
					apptStartAgo: 15 * 24 * 60, apptWindow: 120, arrivalOffset: 20, dwell: 420,
					arrivalSource:   detention.EvidenceSourceTelematics,
					departureSource: detention.EvidenceSourceTelematics,
					notice: &detentionNoticeDef{
						offsetFromExpiry: -20, kind: detention.NoticeKindStarted,
						delivery:      detention.NoticeDeliveryStatusBounced,
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
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *detentionRefs,
	policies map[string]*detention.DetentionPolicy,
) error {
	exists, err := tx.NewSelect().Model((*shipment.Shipment)(nil)).
		Where("organization_id = ?", refs.orgID).
		Where("business_unit_id = ?", refs.buID).
		Where("pro_number = ?", detentionFirstPro).
		Exists(ctx)
	if err != nil {
		return fmt.Errorf("check existing detention shipments: %w", err)
	}
	if exists {
		return nil
	}

	candidates := make([]*detention.DetentionPolicy, 0, len(policies))
	for _, policy := range policies {
		candidates = append(candidates, policy)
	}

	defs := detentionShipmentDefs()
	occurrences := 0
	for _, def := range defs {
		count, sErr := s.createDetentionShipment(ctx, tx, sc, refs, candidates, def)
		if sErr != nil {
			return fmt.Errorf("create shipment %s: %w", def.pro, sErr)
		}
		occurrences += count
	}

	seedhelpers.LogSuccess(
		"Created detention occurrences with evidence chains",
		fmt.Sprintf("- Created %d shipments carrying %d detention occurrences",
			len(defs), occurrences),
	)

	return nil
}

//nolint:funlen // one shipment from header through moves, stops, and occurrences
func (s *DetentionSeed) createDetentionShipment(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *detentionRefs,
	candidates []*detention.DetentionPolicy,
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
		ID:                  pulid.MustNew("shp_"),
		BusinessUnitID:      refs.buID,
		OrganizationID:      refs.orgID,
		CustomerID:          cust.ID,
		ServiceTypeID:       serviceTypeID,
		ShipmentTypeID:      shipmentTypeID,
		FormulaTemplateID:   refs.formulaID,
		Status:              def.status,
		ProNumber:           def.pro,
		BOL:                 def.bol,
		Pieces:              new(def.pieces),
		Weight:              new(def.weight),
		ActualShipDate:      new(shippedAt),
		BaseRate:            decimal.NullDecimal{Decimal: revenue, Valid: true},
		FreightChargeAmount: decimal.NullDecimal{Decimal: revenue, Valid: true},
		TotalChargeAmount:   decimal.NullDecimal{Decimal: revenue, Valid: true},
		RatingUnit:          1,
	}
	last := stops[len(stops)-1]
	if last.ActualDeparture != nil {
		shp.ActualDeliveryDate = new(*last.ActualDeparture)
	}
	if _, err := tx.NewInsert().Model(shp).Exec(ctx); err != nil {
		return 0, fmt.Errorf("insert shipment: %w", err)
	}
	if err := sc.TrackCreated(ctx, "shipments", shp.ID, s.Name()); err != nil {
		return 0, fmt.Errorf("track shipment: %w", err)
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
	if _, err := tx.NewInsert().Model(shipCommodity).Exec(ctx); err != nil {
		return 0, fmt.Errorf("insert shipment commodity: %w", err)
	}
	if err := sc.TrackCreated(ctx, "shipment_commodities", shipCommodity.ID, s.Name()); err != nil {
		return 0, fmt.Errorf("track shipment commodity: %w", err)
	}

	move := &shipment.ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		BusinessUnitID: refs.buID,
		OrganizationID: refs.orgID,
		ShipmentID:     shp.ID,
		Status:         def.moveStatus,
		Loaded:         true,
		Sequence:       0,
	}
	if _, err := tx.NewInsert().Model(move).Exec(ctx); err != nil {
		return 0, fmt.Errorf("insert shipment move: %w", err)
	}
	if err := sc.TrackCreated(ctx, "shipment_moves", move.ID, s.Name()); err != nil {
		return 0, fmt.Errorf("track shipment move: %w", err)
	}

	for _, stop := range stops {
		stop.ShipmentMoveID = move.ID
	}
	if _, err := tx.NewInsert().Model(&stops).Exec(ctx); err != nil {
		return 0, fmt.Errorf("insert stops: %w", err)
	}
	for _, stop := range stops {
		if err := sc.TrackCreated(ctx, "stops", stop.ID, s.Name()); err != nil {
			return 0, fmt.Errorf("track stop: %w", err)
		}
	}

	if err := s.createAssignment(ctx, tx, sc, refs, def, move.ID); err != nil {
		return 0, err
	}

	return s.createOccurrences(ctx, tx, sc, createOccurrencesParams{
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
		Status:               shipment.StopStatusInTransit,
		Sequence:             sequence,
		ScheduledWindowStart: apptStart,
		ScheduledWindowEnd:   new(apptEnd),
		ActualArrival:        new(arrival),
	}

	if def.dwell > 0 {
		stop.Status = shipment.StopStatusCompleted
		stop.ActualDeparture = new(arrival + def.dwell*detentionMinute)
	}

	return stop, nil
}

func (s *DetentionSeed) createAssignment(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
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

	status := shipment.AssignmentStatusCompleted
	if def.moveStatus != shipment.MoveStatusCompleted {
		status = shipment.AssignmentStatusInProgress
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
	}
	if _, err := tx.NewInsert().Model(assignment).Exec(ctx); err != nil {
		return fmt.Errorf("insert assignment: %w", err)
	}
	if err := sc.TrackCreated(ctx, "assignments", assignment.ID, s.Name()); err != nil {
		return fmt.Errorf("track assignment: %w", err)
	}

	return nil
}

// -- Occurrences --

type createOccurrencesParams struct {
	refs       *detentionRefs
	candidates []*detention.DetentionPolicy
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
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	p createOccurrencesParams,
) (int, error) {
	payRate := p.refs.payRates[p.def.workerKey]
	dayAccrued := map[string]decimal.Decimal{}
	shipmentAccrued := decimal.Zero
	created := 0

	for i, stop := range p.stops {
		stopDef := p.def.stops[i]

		match := detention.MatchInput{
			CustomerID:     p.shipment.CustomerID,
			LocationID:     stop.LocationID,
			ShipmentTypeID: p.shipment.ShipmentTypeID,
			ServiceTypeID:  p.shipment.ServiceTypeID,
			CommodityIDs:   []pulid.ID{p.refs.commodityID},
			StopType:       stop.Type,
			ScheduleType:   stop.ScheduleType,
			Timestamp:      *stop.ActualArrival,
		}

		resolution := detentionservice.ResolvePolicy(p.candidates, match)
		if !resolution.Found() {
			continue
		}

		policy := resolution.Policy
		accessorial, ok := p.refs.chargesByID[policy.AccessorialChargeID]
		if !ok {
			return created, fmt.Errorf(
				"policy %s references an accessorial the seed did not load", policy.Code)
		}

		snapshot := detention.NewPolicySnapshot(detention.SnapshotInput{
			Policy:          policy,
			StopType:        string(stop.Type),
			FreeMinutes:     policy.FreeMinutesForStopType(stop.Type),
			AccessorialCode: accessorial.Code,
			FlatRate:        accessorial.Amount,
			FlatRateUnit:    detentionTierUnit(accessorial.RateUnit),
			CapturedAt:      p.now,
		})

		input := detentionservice.ComputeInput{
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
		computed := detentionservice.Compute(input)
		var noticeSentAt *int64
		if stopDef.notice != nil && computed.Applicable {
			sentAt := computed.FreeTimeExpiresAt + stopDef.notice.offsetFromExpiry*detentionMinute
			noticeSentAt = &sentAt
			input.NoticeSentAt = noticeSentAt
			computed = detentionservice.Compute(input)
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

		if err := s.persistOccurrence(ctx, tx, sc, persistOccurrenceParams{
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
	policy *detention.DetentionPolicy,
	snapshot detention.PolicySnapshot,
	computed detentionservice.ComputeResult,
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
		DriverPayAmount:    computed.DriverPayAmount,
		NetMargin:          computed.NetMargin,
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
	policy       *detention.DetentionPolicy
	shipment     *shipment.Shipment
	location     *location.Location
	customerCode string
	noticeSent   *int64
	now          int64
}

func (s *DetentionSeed) persistOccurrence(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	p persistOccurrenceParams,
) error {
	entries := s.buildEvidence(p)
	assessment := detention.AssessCollectability(p.occurrence, entries)
	p.occurrence.CollectabilityScore = assessment.Score
	if len(entries) > 0 {
		p.occurrence.EvidenceHead = entries[len(entries)-1].Hash
	}

	// is_open defaults to true in the schema, so a closed clock has to be written
	// explicitly rather than left to bun's zero-value handling.
	if _, err := tx.NewInsert().
		Model(p.occurrence).
		Value("is_open", "?", p.occurrence.IsOpen).
		Exec(ctx); err != nil {
		return fmt.Errorf("insert detention occurrence: %w", err)
	}
	if err := sc.TrackCreated(ctx, "detention_occurrences", p.occurrence.ID, s.Name()); err != nil {
		return fmt.Errorf("track detention occurrence: %w", err)
	}

	// The charge points back at the occurrence, so the occurrence has to exist
	// before it can be posted, and the back-reference lands as an update.
	if p.occurrence.Status == detention.OccurrenceStatusBilled {
		charge, err := s.createDetentionCharge(ctx, tx, sc, p)
		if err != nil {
			return err
		}
		p.occurrence.AdditionalChargeID = &charge.ID

		if _, err = tx.NewUpdate().
			Model((*detention.DetentionOccurrence)(nil)).
			Set("additional_charge_id = ?", charge.ID).
			Set("updated_at = ?", p.now).
			Where("id = ?", p.occurrence.ID).
			Where("organization_id = ?", p.refs.orgID).
			Where("business_unit_id = ?", p.refs.buID).
			Exec(ctx); err != nil {
			return fmt.Errorf("link detention charge to occurrence: %w", err)
		}
	}

	if len(entries) > 0 {
		if _, err := tx.NewInsert().Model(&entries).Exec(ctx); err != nil {
			return fmt.Errorf("insert detention evidence: %w", err)
		}
		for _, entry := range entries {
			if err := sc.TrackCreated(ctx, "detention_evidence", entry.ID, s.Name()); err != nil {
				return fmt.Errorf("track detention evidence: %w", err)
			}
		}
	}

	return s.createNotice(ctx, tx, sc, p)
}

// createDetentionCharge posts the billed detention onto the shipment so the
// invoice and the occurrence agree on one number.
func (s *DetentionSeed) createDetentionCharge(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	p persistOccurrenceParams,
) (*shipment.AdditionalCharge, error) {
	charge := &shipment.AdditionalCharge{
		ID:                    pulid.MustNew("adc_"),
		BusinessUnitID:        p.refs.buID,
		OrganizationID:        p.refs.orgID,
		ShipmentID:            p.shipment.ID,
		AccessorialChargeID:   p.policy.AccessorialChargeID,
		IsSystemGenerated:     true,
		Method:                accessorialcharge.MethodFlat,
		Amount:                p.occurrence.BillableAmount,
		Unit:                  1,
		DetentionOccurrenceID: &p.occurrence.ID,
	}
	if _, err := tx.NewInsert().Model(charge).Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert detention additional charge: %w", err)
	}
	if err := sc.TrackCreated(ctx, "additional_charges", charge.ID, s.Name()); err != nil {
		return nil, fmt.Errorf("track additional charge: %w", err)
	}

	_, err := tx.NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set("other_charge_amount = COALESCE(other_charge_amount, 0) + ?", charge.Amount).
		Set("total_charge_amount = COALESCE(total_charge_amount, 0) + ?", charge.Amount).
		Where("id = ?", p.shipment.ID).
		Where("organization_id = ?", p.refs.orgID).
		Where("business_unit_id = ?", p.refs.buID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("apply detention charge to shipment total: %w", err)
	}

	return charge, nil
}

// -- Evidence --

type detentionEvidenceDef struct {
	kind       detention.EvidenceKind
	source     detention.EvidenceSource
	summary    string
	observedAt int64
	recordedBy *pulid.ID
	payload    map[string]any
}

//nolint:funlen // the claim file is assembled in the order the facts happened
func (s *DetentionSeed) buildEvidence(p persistOccurrenceParams) []*detention.DetentionEvidence {
	occ := p.occurrence
	facility := "the facility"
	if p.location != nil {
		facility = p.location.Name
	}
	admin := p.refs.adminID

	defs := make([]detentionEvidenceDef, 0, 8)

	if occ.AppointmentStart != nil {
		defs = append(defs, detentionEvidenceDef{
			kind:       detention.EvidenceKindAppointment,
			source:     detention.EvidenceSourceSystem,
			summary:    fmt.Sprintf("Appointment booked at %s for %s", facility, formatDetentionStamp(*occ.AppointmentStart, p.refs.timezone)),
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
			kind:       detention.EvidenceKindArrival,
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
			kind:       detention.EvidenceKindNotice,
			source:     detention.EvidenceSourceSystem,
			summary:    s.noticeEvidenceSummary(p),
			observedAt: *p.noticeSent,
			payload: map[string]any{
				"channel":        string(detention.NoticeChannelEmail),
				"deliveryStatus": string(p.stopDef.notice.delivery),
				"automatic":      p.stopDef.notice.automatic,
			},
		})
	}

	if p.stopDef.document != "" && occ.DepartedAt != nil {
		defs = append(defs, detentionEvidenceDef{
			kind:       detention.EvidenceKindDocument,
			source:     detention.EvidenceSourceDocument,
			summary:    p.stopDef.document,
			observedAt: *occ.DepartedAt,
		})
	}

	if occ.DepartedAt != nil {
		defs = append(defs, detentionEvidenceDef{
			kind:       detention.EvidenceKindDeparture,
			source:     p.stopDef.departureSource,
			summary:    fmt.Sprintf("Driver departed %s after %s on site", facility, formatDetentionDuration(occ.RawDwellMinutes)),
			observedAt: *occ.DepartedAt,
			payload: map[string]any{
				"capturedBy":      string(p.stopDef.departureSource),
				"rawDwellMinutes": occ.RawDwellMinutes,
			},
		})
	}

	defs = append(defs, detentionEvidenceDef{
		kind:   detention.EvidenceKindRecalculated,
		source: detention.EvidenceSourceSystem,
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
			kind:       detention.EvidenceKindStatusChange,
			source:     detention.EvidenceSourceManual,
			summary:    "Approved for billing at " + occ.BillableAmount.StringFixed(2) + " " + occ.Currency,
			observedAt: p.now,
			recordedBy: &admin,
		})
	case detention.OccurrenceStatusBilled:
		defs = append(defs, detentionEvidenceDef{
			kind:       detention.EvidenceKindStatusChange,
			source:     detention.EvidenceSourceSystem,
			summary:    "Billed on shipment " + p.shipment.ProNumber,
			observedAt: p.now,
			recordedBy: &admin,
		})
	case detention.OccurrenceStatusWaived:
		defs = append(defs, detentionEvidenceDef{
			kind:   detention.EvidenceKindWaiver,
			source: detention.EvidenceSourceManual,
			summary: fmt.Sprintf("Waived %s %s (%s): %s",
				occ.WaivedAmount.StringFixed(2), occ.Currency, p.stopDef.waiverReason, p.stopDef.note),
			observedAt: p.now,
			recordedBy: &admin,
		})
	case detention.OccurrenceStatusDisputed:
		defs = append(defs, detentionEvidenceDef{
			kind:       detention.EvidenceKindDispute,
			source:     detention.EvidenceSourceManual,
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
) []*detention.DetentionEvidence {
	entries := make([]*detention.DetentionEvidence, 0, len(defs))
	prev := detention.GenesisHash

	for i, def := range defs {
		entry := &detention.DetentionEvidence{
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
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
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

	content, err := detentionservice.RenderBuiltInNotice(ctx, &detentionservice.BuildNoticeParams{
		Occurrence:    p.occurrence,
		Kind:          def.kind,
		FacilityName:  facility,
		ShipmentRef:   p.shipment.ProNumber,
		ArrivalSource: p.stopDef.arrivalSource,
		Location:      p.refs.timezone,
	})
	if err != nil {
		return fmt.Errorf("render detention notice: %w", err)
	}

	recipients := p.refs.recipients[p.customerCode]
	if len(recipients) == 0 {
		return fmt.Errorf("no notice recipients configured for customer %s", p.customerCode)
	}

	sentAt := *p.noticeSent
	notice := &detention.DetentionNotice{
		ID:                    pulid.MustNew("dtn_"),
		BusinessUnitID:        p.refs.buID,
		OrganizationID:        p.refs.orgID,
		DetentionOccurrenceID: p.occurrence.ID,
		Kind:                  def.kind,
		Channel:               detention.NoticeChannelEmail,
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
	case def.delivery == detention.NoticeDeliveryStatusOpened:
		notice.DeliveredAt = new(sentAt + 3*detentionMinute)
		notice.OpenedAt = new(sentAt + 11*detentionMinute)
		notice.ProviderMessageID = "seed-" + notice.ID.String()
	case def.delivery == detention.NoticeDeliveryStatusDelivered:
		notice.DeliveredAt = new(sentAt + 2*detentionMinute)
		notice.ProviderMessageID = "seed-" + notice.ID.String()
	}

	if _, err := tx.NewInsert().Model(notice).Exec(ctx); err != nil {
		return fmt.Errorf("insert detention notice: %w", err)
	}
	if err := sc.TrackCreated(ctx, "detention_notices", notice.ID, s.Name()); err != nil {
		return fmt.Errorf("track detention notice: %w", err)
	}

	return nil
}

// -- Formatting --

// detentionTierUnit maps an accessorial's rate unit onto the tier vocabulary the
// snapshot speaks, mirroring how the service freezes a flat-rate policy.
func detentionTierUnit(unit accessorialcharge.RateUnit) detention.TierRateUnit {
	switch unit {
	case accessorialcharge.RateUnitHour:
		return detention.TierRateUnitHour
	case accessorialcharge.RateUnitDay:
		return detention.TierRateUnitDay
	case accessorialcharge.RateUnitStop, accessorialcharge.RateUnitMile:
		return detention.TierRateUnitFlat
	default:
		return detention.TierRateUnitHour
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

// -- Rollback --

// Down removes everything this seed wrote, addressing the rows by their own
// markers rather than the entity tracker: the tracker only persists when the
// seed runs against a *bun.DB, and rollback hands the seed a transaction.
func (s *DetentionSeed) Down(ctx context.Context, tx bun.Tx) error {
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

			// The engine flag has to come off before the policies go, or the
			// organization is left switched on with nothing to resolve against.
			if _, err = tx.NewUpdate().
				Model((*tenant.ShipmentControl)(nil)).
				Set("use_detention_policy_engine = FALSE").
				Set("default_detention_policy_id = NULL").
				Set("updated_at = ?", timeutils.NowUnix()).
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Exec(ctx); err != nil {
				return fmt.Errorf("reset shipment control: %w", err)
			}

			if err = s.deleteDetentionShipments(ctx, tx, org); err != nil {
				return err
			}

			if err = s.deletePolicies(ctx, tx, org); err != nil {
				return err
			}

			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *DetentionSeed) deleteDetentionShipments(
	ctx context.Context,
	tx bun.Tx,
	org *tenant.Organization,
) error {
	shipmentIDs := make([]pulid.ID, 0, 16)
	if err := tx.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Column("id").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Where("pro_number LIKE ?", detentionProPattern).
		Scan(ctx, &shipmentIDs); err != nil {
		return fmt.Errorf("get seeded detention shipments: %w", err)
	}

	if len(shipmentIDs) == 0 {
		return nil
	}

	moveIDs := make([]pulid.ID, 0, len(shipmentIDs))
	if err := tx.NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		Column("id").
		Where("shipment_id IN (?)", bun.List(shipmentIDs)).
		Scan(ctx, &moveIDs); err != nil {
		return fmt.Errorf("get seeded detention moves: %w", err)
	}

	// Notices and evidence cascade from the occurrence; everything else is
	// removed child-first so no foreign key is ever left dangling.
	byShipment := []string{"additional_charges", "detention_occurrences", "shipment_commodities"}
	for _, table := range byShipment {
		if _, err := tx.NewDelete().
			Table(table).
			Where("shipment_id IN (?)", bun.List(shipmentIDs)).
			Exec(ctx); err != nil {
			return fmt.Errorf("delete %s: %w", table, err)
		}
	}

	if len(moveIDs) > 0 {
		for _, table := range []string{"assignments", "stops"} {
			if _, err := tx.NewDelete().
				Table(table).
				Where("shipment_move_id IN (?)", bun.List(moveIDs)).
				Exec(ctx); err != nil {
				return fmt.Errorf("delete %s: %w", table, err)
			}
		}

		if _, err := tx.NewDelete().
			Table("shipment_moves").
			Where("id IN (?)", bun.List(moveIDs)).
			Exec(ctx); err != nil {
			return fmt.Errorf("delete shipment_moves: %w", err)
		}
	}

	if _, err := tx.NewDelete().
		Table("shipments").
		Where("id IN (?)", bun.List(shipmentIDs)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete shipments: %w", err)
	}

	return nil
}

func (s *DetentionSeed) deletePolicies(
	ctx context.Context,
	tx bun.Tx,
	org *tenant.Organization,
) error {
	codes := []string{
		detentionPolicyStandard,
		detentionPolicyAcme,
		detentionPolicyCold,
		detentionPolicyDray,
		detentionPolicyLegacy,
		detentionPolicyProposed,
	}

	policyIDs := make([]pulid.ID, 0, len(codes))
	if err := tx.NewSelect().
		Model((*detention.DetentionPolicy)(nil)).
		Column("id").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Where("code IN (?)", bun.List(codes)).
		Scan(ctx, &policyIDs); err != nil {
		return fmt.Errorf("get seeded detention policies: %w", err)
	}

	if len(policyIDs) == 0 {
		return nil
	}

	if _, err := tx.NewDelete().
		Table("detention_policy_tiers").
		Where("detention_policy_id IN (?)", bun.List(policyIDs)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete detention_policy_tiers: %w", err)
	}

	if _, err := tx.NewDelete().
		Table("detention_policies").
		Where("id IN (?)", bun.List(policyIDs)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete detention_policies: %w", err)
	}

	return nil
}

func (s *DetentionSeed) CanRollback() bool {
	return true
}
