package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/domain/ratezone"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	// rateSeedEffectiveFrom is a fixed moment well before any seeded shipment,
	// so a seeded contract is always in force for the loads it should price.
	rateSeedEffectiveFrom = int64(1_700_000_000)

	rateMatrixCode   = "ZONE-TL"
	densityScaleCode = "NMFC-2025"
)

type RateAgreementSeed struct {
	seedhelpers.BaseSeed
}

// RateAgreementSeed builds a working rate book: the geography contracts are
// written against, a zone-by-zone rate matrix, the 2025 density scale, and a
// customer agreement whose lanes deliberately overlap.
//
// The overlap is the point. Three rules cover the Dallas outbound lane at
// different widths — a location pair, a city pair, and a zone pair — so the
// resolver has to choose, the trace has losers to explain, and the "why this
// rate" panel has something to show.
//
// Depends on:
//   - Shipment: customers, and the locations its lanes run between
//   - TestData: accessorial charges and service types
//   - FuelSurcharge: the program the agreement binds its fuel terms to
func NewRateAgreementSeed() *RateAgreementSeed {
	seed := &RateAgreementSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"RateAgreement",
		"1.0.0",
		"Creates rate zones, a zone rate matrix, the 2025 density scale, and a customer rate agreement with overlapping lanes",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(
		seedhelpers.SeedShipment,
		seedhelpers.SeedTestData,
		seedhelpers.SeedFuelSurcharge,
		seedhelpers.SeedFormulaTemplate,
	)

	return seed
}

// rateZoneDef is a named region and the states it covers. States are the widest
// members worth seeding: a real tariff narrows to postal prefixes, but a demo
// that needs three hundred rows to show one zone teaches nothing.
type rateZoneDef struct {
	code        string
	name        string
	description string
	states      []string
}

func rateZoneDefs() []rateZoneDef {
	return []rateZoneDef{
		{
			code:        "SE",
			name:        "Southeast",
			description: "Atlantic and Gulf states from Virginia through Louisiana",
			states: []string{
				"VA", "NC", "SC", "GA", "FL", "AL", "MS", "TN", "KY", "LA", "AR",
			},
		},
		{
			code:        "MW",
			name:        "Midwest",
			description: "The industrial Midwest and upper plains",
			states: []string{
				"OH", "MI", "IN", "IL", "WI", "MN", "IA", "MO", "ND", "SD", "NE", "KS",
			},
		},
		{
			code:        "NE",
			name:        "Northeast",
			description: "New England and the Mid-Atlantic",
			states: []string{
				"ME", "NH", "VT", "MA", "RI", "CT", "NY", "NJ", "PA", "DE", "MD", "WV",
			},
		},
		{
			code:        "SW",
			name:        "Southwest",
			description: "Texas, Oklahoma and the desert Southwest",
			states:      []string{"TX", "OK", "NM", "AZ", "NV", "UT", "CO"},
		},
		{
			code:        "WC",
			name:        "West Coast",
			description: "Pacific states and the mountain Northwest",
			states:      []string{"CA", "OR", "WA", "ID", "MT", "WY", "AK", "HI"},
		},
	}
}

type rateSeedRefs struct {
	orgID      pulid.ID
	buID       pulid.ID
	now        int64
	zones      map[string]pulid.ID
	states     map[string]pulid.ID
	templates  map[string]pulid.ID
	customer   *customer.Customer
	accessorls map[string]pulid.ID
	programID  *pulid.ID
	matrixID   pulid.ID
	densityID  pulid.ID
}

func (s *RateAgreementSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return fmt.Errorf("get organization: %w", err)
			}

			count, err := tx.NewSelect().
				Model((*rateagreement.RateAgreement)(nil)).
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("check existing rate agreements: %w", err)
			}

			if count > 0 {
				return nil
			}

			refs, err := s.loadRefs(ctx, tx, org)
			if err != nil {
				return err
			}

			if err = s.createZones(ctx, tx, sc, refs); err != nil {
				return err
			}

			if err = s.createDensityScale(ctx, tx, sc, refs); err != nil {
				return err
			}

			if err = s.createMatrix(ctx, tx, sc, refs); err != nil {
				return err
			}

			return s.createAgreement(ctx, tx, sc, refs)
		},
	)
}

func (s *RateAgreementSeed) loadRefs(
	ctx context.Context,
	tx bun.Tx,
	org *tenant.Organization,
) (*rateSeedRefs, error) {
	refs := &rateSeedRefs{
		orgID:      org.ID,
		buID:       org.BusinessUnitID,
		now:        timeutils.NowUnix(),
		zones:      make(map[string]pulid.ID),
		states:     make(map[string]pulid.ID),
		templates:  make(map[string]pulid.ID),
		accessorls: make(map[string]pulid.ID),
	}

	type stateRow struct {
		ID           pulid.ID `bun:"id"`
		Abbreviation string   `bun:"abbreviation"`
	}

	var states []stateRow
	if err := tx.NewSelect().
		Table("us_states").
		Column("id", "abbreviation").
		Scan(ctx, &states); err != nil {
		return nil, fmt.Errorf("load us states: %w", err)
	}

	for _, state := range states {
		refs.states[state.Abbreviation] = state.ID
	}

	var templates []formulatemplate.FormulaTemplate
	if err := tx.NewSelect().
		Model(&templates).
		Column("id", "name").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Where("status = ?", formulatemplate.StatusActive).
		Where("type = ?", formulatemplate.TemplateTypeFreightCharge).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load formula templates: %w", err)
	}

	for _, template := range templates {
		refs.templates[template.Name] = template.ID
	}

	cust := new(customer.Customer)
	if err := tx.NewSelect().
		Model(cust).
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Order("created_at ASC").
		Limit(1).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load customer: %w", err)
	}
	refs.customer = cust

	var charges []accessorialcharge.AccessorialCharge
	if err := tx.NewSelect().
		Model(&charges).
		Column("id", "code").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load accessorial charges: %w", err)
	}

	for _, charge := range charges {
		refs.accessorls[charge.Code] = charge.ID
	}

	program := new(fuelsurcharge.FuelSurchargeProgram)
	if err := tx.NewSelect().
		Model(program).
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Order("created_at ASC").
		Limit(1).
		Scan(ctx); err == nil {
		programID := program.ID
		refs.programID = &programID
	}

	return refs, nil
}

func (s *RateAgreementSeed) createZones(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *rateSeedRefs,
) error {
	for _, def := range rateZoneDefs() {
		zone := &ratezone.RateZone{
			ID:             pulid.MustNew("rzn_"),
			OrganizationID: refs.orgID,
			BusinessUnitID: refs.buID,
			Code:           def.code,
			Name:           def.name,
			Description:    def.description,
			Kind:           ratezone.ZoneKindRegional,
			Status:         domaintypes.StatusActive,
			CreatedAt:      refs.now,
			UpdatedAt:      refs.now,
		}

		if _, err := tx.NewInsert().Model(zone).Exec(ctx); err != nil {
			return fmt.Errorf("insert rate zone %s: %w", def.code, err)
		}

		if err := sc.TrackCreated(ctx, "rate_zones", zone.ID, s.Name()); err != nil {
			return fmt.Errorf("track rate zone: %w", err)
		}

		refs.zones[def.code] = zone.ID

		members := make([]*ratezone.RateZoneMember, 0, len(def.states))
		for _, abbreviation := range def.states {
			stateID, ok := refs.states[abbreviation]
			if !ok {
				continue
			}

			member := &ratezone.RateZoneMember{
				ID:             pulid.MustNew("rzm_"),
				OrganizationID: refs.orgID,
				BusinessUnitID: refs.buID,
				RateZoneID:     zone.ID,
				ScopeType:      rategeo.ScopeTypeState,
				ScopeValue:     stateID.String(),
				CreatedAt:      refs.now,
				UpdatedAt:      refs.now,
			}
			member.ApplyMatchKey()

			members = append(members, member)
		}

		if len(members) == 0 {
			return fmt.Errorf("rate zone %s resolved no states", def.code)
		}

		if _, err := tx.NewInsert().Model(&members).Exec(ctx); err != nil {
			return fmt.Errorf("insert rate zone members for %s: %w", def.code, err)
		}

		sc.Logger().Info("Created rate zone %s with %d states", def.code, len(members))
	}

	return nil
}

// createDensityScale seeds the 2025 NMFC density ladder as the organization's
// default, which is what a density-rated LTL rule classifies against.
func (s *RateAgreementSeed) createDensityScale(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *rateSeedRefs,
) error {
	scale := &ratematrix.DensityScale{
		ID:             pulid.MustNew("rds_"),
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		Code:           densityScaleCode,
		Name:           "NMFC 2025 Density Scale",
		Description:    "The thirteen-tier density ladder the July 2025 NMFC docket moved most commodities onto",
		Status:         domaintypes.StatusActive,
		EffectiveFrom:  rateSeedEffectiveFrom,
		IsOrgDefault:   true,
		CreatedAt:      refs.now,
		UpdatedAt:      refs.now,
	}

	if _, err := tx.NewInsert().Model(scale).Exec(ctx); err != nil {
		return fmt.Errorf("insert density scale: %w", err)
	}

	if err := sc.TrackCreated(ctx, "rate_density_scales", scale.ID, s.Name()); err != nil {
		return fmt.Errorf("track density scale: %w", err)
	}

	refs.densityID = scale.ID

	tiers := ratematrix.StandardDensityTiers(refs.orgID, refs.buID)
	for _, tier := range tiers {
		tier.ID = pulid.MustNew("rdt_")
		tier.RateDensityScaleID = scale.ID
		tier.CreatedAt = refs.now
		tier.UpdatedAt = refs.now
	}

	if _, err := tx.NewInsert().Model(&tiers).Exec(ctx); err != nil {
		return fmt.Errorf("insert density scale tiers: %w", err)
	}

	sc.Logger().Info("Created density scale %s with %d tiers", densityScaleCode, len(tiers))

	return nil
}

// zoneMatrixCell is one priced intersection of the seeded zone matrix.
type zoneMatrixCell struct {
	origin      string
	destination string
	ratePerMile string
}

func zoneMatrixCells() []zoneMatrixCell {
	return []zoneMatrixCell{
		{origin: "SW", destination: "MW", ratePerMile: "2.35"},
		{origin: "SW", destination: "SE", ratePerMile: "2.15"},
		{origin: "SW", destination: "WC", ratePerMile: "2.55"},
		{origin: "SW", destination: "NE", ratePerMile: "2.75"},
		{origin: "MW", destination: "SW", ratePerMile: "2.20"},
		{origin: "MW", destination: "SE", ratePerMile: "2.30"},
		{origin: "MW", destination: "NE", ratePerMile: "2.45"},
		{origin: "WC", destination: "MW", ratePerMile: "2.85"},
		{origin: "WC", destination: "SW", ratePerMile: "2.40"},
		{origin: "SE", destination: "MW", ratePerMile: "2.25"},
		{origin: "SE", destination: "NE", ratePerMile: "2.60"},
		{origin: "NE", destination: "SE", ratePerMile: "2.50"},
	}
}

// createMatrix builds a zone-by-zone rate table, which is the shape a truckload
// tariff most often arrives in: a grid of origin region against destination
// region, priced per mile.
func (s *RateAgreementSeed) createMatrix(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *rateSeedRefs,
) error {
	perMileID, ok := refs.templates[formulatemplate.StandardPerMile]
	if !ok {
		return fmt.Errorf(
			"the %q formula template is not seeded; the zone matrix cannot say what its cells mean",
			formulatemplate.StandardPerMile,
		)
	}

	matrix := &ratematrix.RateMatrix{
		ID:                pulid.MustNew("rmx_"),
		OrganizationID:    refs.orgID,
		BusinessUnitID:    refs.buID,
		Code:              rateMatrixCode,
		Name:              "Zone to Zone Truckload",
		Description:       "Per-mile linehaul by origin region and destination region",
		Status:            domaintypes.StatusActive,
		Currency:          "USD",
		FormulaTemplateID: perMileID,
		CreatedAt:         refs.now,
		UpdatedAt:         refs.now,
	}

	if _, err := tx.NewInsert().Model(matrix).Exec(ctx); err != nil {
		return fmt.Errorf("insert rate matrix: %w", err)
	}

	if err := sc.TrackCreated(ctx, "rate_matrices", matrix.ID, s.Name()); err != nil {
		return fmt.Errorf("track rate matrix: %w", err)
	}

	refs.matrixID = matrix.ID

	dimensions := []*ratematrix.RateMatrixDimension{
		{
			ID:             pulid.MustNew("rmd_"),
			OrganizationID: refs.orgID,
			BusinessUnitID: refs.buID,
			RateMatrixID:   matrix.ID,
			Position:       0,
			Kind:           ratematrix.DimensionKindZone,
			MatchMode:      ratematrix.MatchModeExact,
			Label:          "Origin region",
			CreatedAt:      refs.now,
			UpdatedAt:      refs.now,
		},
		{
			ID:             pulid.MustNew("rmd_"),
			OrganizationID: refs.orgID,
			BusinessUnitID: refs.buID,
			RateMatrixID:   matrix.ID,
			Position:       1,
			Kind:           ratematrix.DimensionKindZone,
			MatchMode:      ratematrix.MatchModeExact,
			Label:          "Destination region",
			CreatedAt:      refs.now,
			UpdatedAt:      refs.now,
		},
	}

	if _, err := tx.NewInsert().Model(&dimensions).Exec(ctx); err != nil {
		return fmt.Errorf("insert rate matrix dimensions: %w", err)
	}

	defs := zoneMatrixCells()
	cells := make([]*ratematrix.RateMatrixCell, 0, len(defs))

	for _, def := range defs {
		originID, originOK := refs.zones[def.origin]
		destinationID, destinationOK := refs.zones[def.destination]
		if !originOK || !destinationOK {
			continue
		}

		cells = append(cells, &ratematrix.RateMatrixCell{
			ID:             pulid.MustNew("rmc_"),
			OrganizationID: refs.orgID,
			BusinessUnitID: refs.buID,
			RateMatrixID:   matrix.ID,
			D0Key:          originID.String(),
			D1Key:          destinationID.String(),
			Value:          decimal.RequireFromString(def.ratePerMile),
			CreatedAt:      refs.now,
			UpdatedAt:      refs.now,
		})
	}

	if _, err := tx.NewInsert().Model(&cells).Exec(ctx); err != nil {
		return fmt.Errorf("insert rate matrix cells: %w", err)
	}

	sc.Logger().Info("Created rate matrix %s with %d cells", rateMatrixCode, len(cells))

	return nil
}

// laneDef is one priced lane of the seeded agreement.
//
// The three Dallas outbound lanes deliberately overlap. Each is written at a
// different width, so a shipment from the Dallas yard to Chicago matches all
// three and the resolver has to pick — which is the behaviour the trace exists
// to explain and the panel exists to show.
type laneDef struct {
	label       string
	originType  rategeo.ScopeType
	originValue string
	originCity  string
	destType    rategeo.ScopeType
	destValue   string
	destCity    string
	// template names the standard formula template that prices the lane, and
	// useMatrix points the lane at the zone matrix instead — a lane carries
	// exactly one of the two.
	template  string
	useMatrix bool
	rate      string
	minCharge string
	minMiles  string
	priority  int16
}

// zoneRef and stateRef mark a lane value that has to be resolved to an id once
// the zones exist, since a seed cannot know them in advance.
type laneScopeSource string

const (
	scopeSourceZone  laneScopeSource = "zone"
	scopeSourceState laneScopeSource = "state"
)

type resolvedLane struct {
	def         laneDef
	originSrc   laneScopeSource
	destSrc     laneScopeSource
	originToken string
	destToken   string
}

// seededLanes lists the agreement's rates from the narrowest lane to the
// widest, which is the order the resolver will rank them in.
//
// A zone outranks a state deliberately: organizations write zones for metros
// and market areas far more often than for whole regions, so the narrower
// reading is the useful default. The two zone lanes tie, which is correct —
// they cover different geography and never compete with each other.
func seededLanes() []resolvedLane {
	return []resolvedLane{
		{
			def: laneDef{
				label:      "Dallas to Chicago, city pair",
				originType: rategeo.ScopeTypeCityState,
				originCity: "Dallas",
				destType:   rategeo.ScopeTypeCityState,
				destCity:   "Chicago",
				template:   formulatemplate.StandardPerMile,
				rate:       "2.55",
				minCharge:  "850.00",
				minMiles:   "250",
			},
			originSrc:   scopeSourceState,
			destSrc:     scopeSourceState,
			originToken: "TX",
			destToken:   "IL",
		},
		{
			def: laneDef{
				label:      "Southwest to Midwest, zone pair",
				originType: rategeo.ScopeTypeZone,
				destType:   rategeo.ScopeTypeZone,
				useMatrix:  true,
				minCharge:  "750.00",
			},
			originSrc:   scopeSourceZone,
			destSrc:     scopeSourceZone,
			originToken: "SW",
			destToken:   "MW",
		},
		{
			def: laneDef{
				label:      "West Coast to Midwest, zone pair",
				originType: rategeo.ScopeTypeZone,
				destType:   rategeo.ScopeTypeZone,
				useMatrix:  true,
				minCharge:  "950.00",
			},
			originSrc:   scopeSourceZone,
			destSrc:     scopeSourceZone,
			originToken: "WC",
			destToken:   "MW",
		},
		{
			def: laneDef{
				label:      "Texas to Illinois, state pair",
				originType: rategeo.ScopeTypeState,
				destType:   rategeo.ScopeTypeState,
				template:   formulatemplate.StandardPerMile,
				rate:       "2.40",
				minCharge:  "800.00",
			},
			originSrc:   scopeSourceState,
			destSrc:     scopeSourceState,
			originToken: "TX",
			destToken:   "IL",
		},
		{
			def: laneDef{
				label:      "Anywhere to anywhere, contract floor",
				originType: rategeo.ScopeTypeAny,
				destType:   rategeo.ScopeTypeAny,
				template:   formulatemplate.StandardPerMile,
				rate:       "2.10",
				minCharge:  "600.00",
				priority:   -10,
			},
		},
	}
}

// accessorialDef is one row of the agreement's negotiated accessorial schedule.
type accessorialDef struct {
	code      string
	method    accessorialcharge.Method
	rateUnit  accessorialcharge.RateUnit
	amount    string
	autoApply bool
	condition string
}

func seededAccessorials() []accessorialDef {
	return []accessorialDef{
		{
			// Detention is priced by the contract rather than by the
			// organization default, which is what makes the rate confirmation
			// and the invoice agree on it.
			code:      "DET",
			method:    accessorialcharge.MethodPerUnit,
			rateUnit:  accessorialcharge.RateUnitHour,
			amount:    "65.00",
			autoApply: false,
		},
		{
			// A multi-stop fee the contract applies on its own, which is what
			// makes the rate confirmation and the invoice agree without anybody
			// remembering to add it.
			code:      "STP",
			method:    accessorialcharge.MethodFlat,
			amount:    "75.00",
			autoApply: true,
			condition: "totalStops > 2",
		},
	}
}

func (s *RateAgreementSeed) createAgreement(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *rateSeedRefs,
) error {
	customerID := refs.customer.ID

	agreement := &rateagreement.RateAgreement{
		ID:                pulid.MustNew("rag_"),
		OrganizationID:    refs.orgID,
		BusinessUnitID:    refs.buID,
		PartyType:         rateagreement.PartyTypeCustomer,
		CustomerID:        &customerID,
		Code:              "SEED-TL-2026",
		Name:              refs.customer.Name + " Truckload Agreement",
		Description:       "Seeded contract with overlapping lanes so rate resolution can be demonstrated",
		AgreementType:     rateagreement.AgreementTypeContract,
		Status:            rateagreement.StatusActive,
		Priority:          10,
		EffectiveFrom:     rateSeedEffectiveFrom,
		Currency:          "USD",
		RoundingMode:      ratetypes.RoundingModeHalfUp,
		RoundingPrecision: 2,
		CreatedAt:         refs.now,
		UpdatedAt:         refs.now,
	}

	if _, err := tx.NewInsert().Model(agreement).Exec(ctx); err != nil {
		return fmt.Errorf("insert rate agreement: %w", err)
	}

	if err := sc.TrackCreated(ctx, "rate_agreements", agreement.ID, s.Name()); err != nil {
		return fmt.Errorf("track rate agreement: %w", err)
	}

	if err := s.createRules(ctx, tx, refs, agreement); err != nil {
		return err
	}

	if err := s.createAccessorials(ctx, tx, refs, agreement); err != nil {
		return err
	}

	if err := s.createFuelBinding(ctx, tx, refs, agreement); err != nil {
		return err
	}

	sc.Logger().Info("Created rate agreement %s for %s", agreement.Code, refs.customer.Name)

	return nil
}

func (s *RateAgreementSeed) createRules(
	ctx context.Context,
	tx bun.Tx,
	refs *rateSeedRefs,
	agreement *rateagreement.RateAgreement,
) error {
	lanes := seededLanes()
	rules := make([]*rateagreement.RateAgreementRule, 0, len(lanes))

	for _, lane := range lanes {
		rule, ok := s.buildRule(refs, agreement, lane)
		if !ok {
			continue
		}
		rules = append(rules, rule)
	}

	if len(rules) == 0 {
		return fmt.Errorf("no seeded lanes resolved to a rate rule")
	}

	if _, err := tx.NewInsert().Model(&rules).Exec(ctx); err != nil {
		return fmt.Errorf("insert rate agreement rules: %w", err)
	}

	return nil
}

func (s *RateAgreementSeed) buildRule(
	refs *rateSeedRefs,
	agreement *rateagreement.RateAgreement,
	lane resolvedLane,
) (*rateagreement.RateAgreementRule, bool) {
	originValue, originOK := s.resolveScopeValue(refs, lane.originSrc, lane.originToken)
	destValue, destOK := s.resolveScopeValue(refs, lane.destSrc, lane.destToken)

	if !originOK || !destOK {
		return nil, false
	}

	rule := &rateagreement.RateAgreementRule{
		ID:                    pulid.MustNew("ragr_"),
		OrganizationID:        refs.orgID,
		BusinessUnitID:        refs.buID,
		RateAgreementID:       agreement.ID,
		PartyType:             agreement.PartyType,
		PartyID:               agreement.PartyID(),
		Label:                 lane.def.label,
		Status:                rateagreement.RuleStatusActive,
		OriginScopeType:       lane.def.originType,
		OriginScopeValue:      originValue,
		OriginCity:            lane.def.originCity,
		DestinationScopeType:  lane.def.destType,
		DestinationScopeValue: destValue,
		DestinationCity:       lane.def.destCity,
		Direction:             rateagreement.DirectionDirectional,
		Priority:              lane.def.priority,
		EffectiveFrom:         rateSeedEffectiveFrom,
		CreatedAt:             refs.now,
		UpdatedAt:             refs.now,
	}

	if lane.def.rate != "" {
		rule.Rate = decimal.NewNullDecimal(decimal.RequireFromString(lane.def.rate))
	}

	if lane.def.minCharge != "" {
		rule.MinCharge = decimal.NewNullDecimal(decimal.RequireFromString(lane.def.minCharge))
	}

	if lane.def.minMiles != "" {
		rule.MinBillableDistance = decimal.NewNullDecimal(
			decimal.RequireFromString(lane.def.minMiles),
		)
	}

	if lane.def.useMatrix {
		matrixID := refs.matrixID
		rule.RateMatrixID = &matrixID
	} else {
		templateID, ok := refs.templates[lane.def.template]
		if !ok {
			return nil, false
		}
		rule.FormulaTemplateID = &templateID
	}

	// The lane key and specificity are what the database indexes and orders by,
	// so they are derived here rather than left to a write path the seed skips.
	rule.ApplyLaneKey()
	rule.SpecificityScore = rule.ComputeSpecificity()

	return rule, true
}

// resolveScopeValue turns a lane's token into the identifier the rule stores.
// A lane that names nothing — an Any scope — resolves to an empty value, which
// is exactly what the key encoder expects.
func (s *RateAgreementSeed) resolveScopeValue(
	refs *rateSeedRefs,
	source laneScopeSource,
	token string,
) (string, bool) {
	switch source {
	case scopeSourceZone:
		zoneID, ok := refs.zones[token]
		if !ok {
			return "", false
		}
		return zoneID.String(), true
	case scopeSourceState:
		stateID, ok := refs.states[token]
		if !ok {
			return "", false
		}
		return stateID.String(), true
	default:
		return "", true
	}
}

func (s *RateAgreementSeed) createAccessorials(
	ctx context.Context,
	tx bun.Tx,
	refs *rateSeedRefs,
	agreement *rateagreement.RateAgreement,
) error {
	defs := seededAccessorials()
	rows := make([]*rateagreement.RateAgreementAccessorial, 0, len(defs))
	effectiveFrom := int64(rateSeedEffectiveFrom)

	for _, def := range defs {
		chargeID, ok := refs.accessorls[def.code]
		if !ok {
			continue
		}

		rows = append(rows, &rateagreement.RateAgreementAccessorial{
			ID:                  pulid.MustNew("raga_"),
			OrganizationID:      refs.orgID,
			BusinessUnitID:      refs.buID,
			RateAgreementID:     agreement.ID,
			AccessorialChargeID: chargeID,
			Method:              def.method,
			RateUnit:            def.rateUnit,
			Amount:              decimal.RequireFromString(def.amount),
			AutoApply:           def.autoApply,
			ApplyCondition:      def.condition,
			EffectiveFrom:       &effectiveFrom,
			CreatedAt:           refs.now,
			UpdatedAt:           refs.now,
		})
	}

	if len(rows) == 0 {
		return nil
	}

	if _, err := tx.NewInsert().Model(&rows).Exec(ctx); err != nil {
		return fmt.Errorf("insert rate agreement accessorials: %w", err)
	}

	return nil
}

// createFuelBinding points the contract at the organization's fuel program and
// caps the surcharge, which is the term customers negotiate most often and the
// one their current systems most often cannot express.
func (s *RateAgreementSeed) createFuelBinding(
	ctx context.Context,
	tx bun.Tx,
	refs *rateSeedRefs,
	agreement *rateagreement.RateAgreement,
) error {
	if refs.programID == nil {
		return nil
	}

	binding := &rateagreement.RateAgreementFuelBinding{
		ID:                     pulid.MustNew("ragf_"),
		OrganizationID:         refs.orgID,
		BusinessUnitID:         refs.buID,
		RateAgreementID:        agreement.ID,
		FuelSurchargeProgramID: *refs.programID,
		CapAmount:              decimal.NewNullDecimal(decimal.RequireFromString("450.00")),
		CreatedAt:              refs.now,
		UpdatedAt:              refs.now,
	}

	if _, err := tx.NewInsert().Model(binding).Exec(ctx); err != nil {
		return fmt.Errorf("insert rate agreement fuel binding: %w", err)
	}

	return nil
}
