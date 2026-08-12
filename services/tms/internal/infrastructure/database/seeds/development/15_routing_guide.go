package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type RoutingGuideSeed struct {
	seedhelpers.BaseSeed
}

func NewRoutingGuideSeed() *RoutingGuideSeed {
	seed := &RoutingGuideSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"RoutingGuide",
		"1.0.0",
		"Creates routing guides on the seeded shipment lanes so waterfall tenders can be demonstrated",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(seedhelpers.SeedCarrier, seedhelpers.SeedLocation)

	return seed
}

type routingGuideEntryDef struct {
	carrierCode     string
	rank            int16
	rate            string
	offerTTLSeconds int32
}

// routingGuideDef expresses a lane the way RoutingGuide.ComputeSpecificity
// reads it: either a city pair or a state pair, so the stored specificity lands
// on exactly one tier of MatchLane.
type routingGuideDef struct {
	name             string
	description      string
	originCity       string
	originState      string
	destinationCity  string
	destinationState string
	entries          []routingGuideEntryDef
}

// routingGuideDefs targets lanes the shipment seed actually creates so the
// dispatch console matches against seeded moves. Dallas -> Chicago is the
// city-pair lane behind four seeded moves; CA -> IL is the state-pair fallback
// that catches the Los Angeles -> Chicago move one tier down. The primary
// guide's ranks walk the whole screening path: an eligible carrier, a carrier
// blocked on an expired certificate (skipped with an audit entry), a carrier
// inside the 30-day insurance warning window, then a clean fallback.
func routingGuideDefs() []routingGuideDef {
	return []routingGuideDef{
		{
			name:             "Dallas to Chicago Primary",
			description:      "Ranked contract capacity on the Dallas outbound lane into Chicago",
			originCity:       "Dallas",
			originState:      "TX",
			destinationCity:  "Chicago",
			destinationState: "IL",
			entries: []routingGuideEntryDef{
				{carrierCode: "BRT", rank: 1, rate: "2150.00", offerTTLSeconds: 1800},
				{carrierCode: "IHF", rank: 2, rate: "2240.00", offerTTLSeconds: 1800},
				{carrierCode: "SPL", rank: 3, rate: "2310.00", offerTTLSeconds: 3600},
				{carrierCode: "GPC", rank: 4, rate: "2475.00", offerTTLSeconds: 3600},
			},
		},
		{
			name:             "California to Illinois Fallback",
			description:      "State-level fallback capacity for origins without a city-pair guide",
			originState:      "CA",
			destinationState: "IL",
			entries: []routingGuideEntryDef{
				{carrierCode: "PCT", rank: 1, rate: "3120.00", offerTTLSeconds: 3600},
				{carrierCode: "GPC", rank: 2, rate: "3280.00", offerTTLSeconds: 3600},
			},
		},
	}
}

// routingGuideSeedIdentity carries the tenant keys a seeded guide row needs so
// the lane builder stays free of database access.
type routingGuideSeedIdentity struct {
	orgID   pulid.ID
	buID    pulid.ID
	guideID pulid.ID
	now     int64
}

func buildRoutingGuide(
	def routingGuideDef,
	ident routingGuideSeedIdentity,
) *tender.RoutingGuide {
	guide := &tender.RoutingGuide{
		ID:               ident.guideID,
		BusinessUnitID:   ident.buID,
		OrganizationID:   ident.orgID,
		Name:             def.name,
		Description:      def.description,
		Status:           domaintypes.StatusActive,
		OriginCity:       def.originCity,
		OriginState:      def.originState,
		DestinationCity:  def.destinationCity,
		DestinationState: def.destinationState,
		CreatedAt:        ident.now,
		UpdatedAt:        ident.now,
	}
	guide.Specificity = guide.ComputeSpecificity()

	return guide
}

func (s *RoutingGuideSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetOrganization("default_org")
			if err != nil {
				org, err = sc.GetDefaultOrganization(ctx)
				if err != nil {
					return fmt.Errorf("get organization: %w", err)
				}
			}

			count, err := tx.NewSelect().
				Model((*tender.RoutingGuide)(nil)).
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("check existing routing guides: %w", err)
			}

			if count > 0 {
				return nil
			}

			carrierByCode, err := s.loadCarriers(ctx, tx, org.ID, org.BusinessUnitID)
			if err != nil {
				return err
			}

			now := timeutils.NowUnix()

			for _, def := range routingGuideDefs() {
				guideID := pulid.MustNew("rg_")

				guide := buildRoutingGuide(def, routingGuideSeedIdentity{
					orgID:   org.ID,
					buID:    org.BusinessUnitID,
					guideID: guideID,
					now:     now,
				})
				if guide.Specificity == 0 {
					return fmt.Errorf("routing guide %s does not define a matchable lane", def.name)
				}

				if _, err = tx.NewInsert().Model(guide).Exec(ctx); err != nil {
					return fmt.Errorf("insert routing guide %s: %w", def.name, err)
				}

				if err = sc.TrackCreated(ctx, "routing_guides", guideID, s.Name()); err != nil {
					return fmt.Errorf("track routing guide: %w", err)
				}

				if err = s.insertEntries(ctx, tx, sc, insertGuideEntriesParams{
					org:     org.ID,
					bu:      org.BusinessUnitID,
					guideID: guideID,
					now:     now,
					def:     def,
					byCode:  carrierByCode,
				}); err != nil {
					return err
				}

				sc.Logger().Info(
					"Created routing guide %s with %d ranked entries",
					def.name,
					len(def.entries),
				)
			}

			return nil
		},
	)
}

func (s *RoutingGuideSeed) loadCarriers(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) (map[string]pulid.ID, error) {
	var carriers []carrier.Carrier
	if err := tx.NewSelect().
		Model(&carriers).
		Column("id", "code").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("get carriers: %w", err)
	}

	byCode := make(map[string]pulid.ID, len(carriers))
	for _, c := range carriers {
		byCode[c.Code] = c.ID
	}

	return byCode, nil
}

type insertGuideEntriesParams struct {
	org     pulid.ID
	bu      pulid.ID
	guideID pulid.ID
	now     int64
	def     routingGuideDef
	byCode  map[string]pulid.ID
}

func (s *RoutingGuideSeed) insertEntries(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	params insertGuideEntriesParams,
) error {
	entries := make([]*tender.RoutingGuideEntry, 0, len(params.def.entries))
	for _, entryDef := range params.def.entries {
		carrierID, ok := params.byCode[entryDef.carrierCode]
		if !ok {
			return fmt.Errorf(
				"routing guide %s references unknown carrier code %s",
				params.def.name,
				entryDef.carrierCode,
			)
		}

		rate, err := decimal.NewFromString(entryDef.rate)
		if err != nil {
			return fmt.Errorf(
				"parse rate %s for routing guide %s: %w",
				entryDef.rate,
				params.def.name,
				err,
			)
		}

		entries = append(entries, &tender.RoutingGuideEntry{
			ID:              pulid.MustNew("rge_"),
			BusinessUnitID:  params.bu,
			OrganizationID:  params.org,
			RoutingGuideID:  params.guideID,
			CarrierID:       carrierID,
			Rank:            entryDef.rank,
			RateMethod:      shipment.CarrierRateMethodFlat,
			Rate:            rate,
			OfferTTLSeconds: entryDef.offerTTLSeconds,
			Channel:         tender.ChannelEmail,
			CreatedAt:       params.now,
			UpdatedAt:       params.now,
		})
	}

	if _, err := tx.NewInsert().Model(&entries).Exec(ctx); err != nil {
		return fmt.Errorf("insert entries for routing guide %s: %w", params.def.name, err)
	}

	for _, entry := range entries {
		if err := sc.TrackCreated(ctx, "routing_guide_entries", entry.ID, s.Name()); err != nil {
			return fmt.Errorf("track routing guide entry: %w", err)
		}
	}

	return nil
}

func (s *RoutingGuideSeed) Down(ctx context.Context, tx bun.Tx) error {
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

func (s *RoutingGuideSeed) CanRollback() bool {
	return true
}
