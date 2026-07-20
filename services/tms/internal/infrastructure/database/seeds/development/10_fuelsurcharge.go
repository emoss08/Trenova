package development

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type FuelSurchargeSeed struct {
	seedhelpers.BaseSeed
}

func NewFuelSurchargeSeed() *FuelSurchargeSeed {
	seed := &FuelSurchargeSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"FuelSurcharge",
		"1.2.0",
		"Creates all DOE regional diesel indices with recent weekly prices and a demo fuel surcharge program",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(seedhelpers.SeedTestOrganizations)

	return seed
}

var seedWeeklyPrices = []string{
	"3.612", "3.587", "3.641", "3.702", "3.688", "3.725", "3.691", "3.756",
}

var seedRegionOffsets = map[string]string{
	"DOE_US":               "0",
	"DOE_EAST_COAST":       "0.08",
	"DOE_NEW_ENGLAND":      "0.21",
	"DOE_CENTRAL_ATLANTIC": "0.24",
	"DOE_LOWER_ATLANTIC":   "-0.02",
	"DOE_MIDWEST":          "-0.05",
	"DOE_GULF_COAST":       "-0.18",
	"DOE_ROCKY_MOUNTAIN":   "-0.01",
	"DOE_WEST_COAST":       "0.62",
	"DOE_WEST_COAST_NO_CA": "0.28",
	"DOE_CALIFORNIA":       "0.98",
}

func (s *FuelSurchargeSeed) Run(ctx context.Context, tx bun.Tx) error {
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

			chargeID, err := s.ensureFuelAccessorial(ctx, tx, sc, org.ID, org.BusinessUnitID)
			if err != nil {
				return err
			}

			monday := currentSeedMonday()
			var usIndexID pulid.ID

			for _, def := range fuelsurcharge.EIASeriesRegistry() {
				indexID, iErr := s.ensureIndexWithPrices(
					ctx, tx, sc, org.ID, org.BusinessUnitID, def, monday)
				if iErr != nil {
					return iErr
				}

				if def.Code == "DOE_US" {
					usIndexID = indexID
				}
			}

			count, err := tx.NewSelect().
				Model((*fuelsurcharge.FuelSurchargeProgram)(nil)).
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("check existing fuel surcharge programs: %w", err)
			}

			if count > 0 {
				return nil
			}

			program := &fuelsurcharge.FuelSurchargeProgram{
				ID:                   pulid.MustNew("fsp_"),
				OrganizationID:       org.ID,
				BusinessUnitID:       org.BusinessUnitID,
				Name:                 "Standard DOE Program",
				Code:                 "FSC-DOE-STD",
				Description:          "Peg $1.20, one cent per mile for each $0.05 above the peg, DOE national average",
				Status:               fuelsurcharge.ProgramStatusActive,
				FuelIndexID:          usIndexID,
				AccessorialChargeID:  chargeID,
				Method:               fuelsurcharge.ProgramMethodPerMileStep,
				PegPrice:             decimal.NewNullDecimal(decimal.RequireFromString("1.20")),
				Increment:            decimal.NewNullDecimal(decimal.RequireFromString("0.05")),
				IncrementRate:        decimal.NewNullDecimal(decimal.RequireFromString("0.01")),
				StepRounding:         fuelsurcharge.StepRoundingUp,
				RateRounding:         fuelsurcharge.RateRoundingHalfUp,
				RatePrecision:        4,
				DateBasis:            fuelsurcharge.DateBasisPickupDate,
				PriceEffectiveDay:    3,
				MissingPriceFallback: fuelsurcharge.FallbackUseLatestAvailable,
			}
			if _, err = tx.NewInsert().Model(program).Exec(ctx); err != nil {
				return fmt.Errorf("insert fuel surcharge program: %w", err)
			}
			if err = sc.TrackCreated(ctx, "fuel_surcharge_programs", program.ID, s.Name()); err != nil {
				return fmt.Errorf("track fuel surcharge program: %w", err)
			}

			return nil
		},
	)
}

func (s *FuelSurchargeSeed) ensureFuelAccessorial(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) (pulid.ID, error) {
	existing := new(accessorialcharge.AccessorialCharge)
	err := tx.NewSelect().
		Model(existing).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("code = ?", "FUEL").
		Limit(1).
		Scan(ctx)
	if err == nil {
		return existing.ID, nil
	}

	charge := &accessorialcharge.AccessorialCharge{
		ID:             pulid.MustNew("acc_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           "FUEL",
		Description:    "Fuel surcharge (auto-applied from the customer's fuel surcharge program)",
		Method:         accessorialcharge.MethodFlat,
		Amount:         decimal.NewFromInt(0),
	}
	if _, err = tx.NewInsert().Model(charge).Exec(ctx); err != nil {
		return pulid.Nil, fmt.Errorf("insert fuel accessorial charge: %w", err)
	}
	if err = sc.TrackCreated(ctx, "accessorial_charges", charge.ID, s.Name()); err != nil {
		return pulid.Nil, fmt.Errorf("track fuel accessorial charge: %w", err)
	}

	return charge.ID, nil
}

func (s *FuelSurchargeSeed) ensureIndexWithPrices(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
	def fuelsurcharge.EIASeriesDef,
	monday time.Time,
) (pulid.ID, error) {
	existing := new(fuelsurcharge.FuelIndex)
	err := tx.NewSelect().
		Model(existing).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("code = ?", def.Code).
		Limit(1).
		Scan(ctx)
	if err == nil {
		return existing.ID, nil
	}

	index := &fuelsurcharge.FuelIndex{
		ID:             pulid.MustNew("fidx_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Name:           def.Name,
		Code:           def.Code,
		Description:    "DOE/EIA weekly on-highway diesel retail price",
		Source:         fuelsurcharge.IndexSourceEIA,
		FuelType:       def.FuelType,
		Region:         def.Region,
		EIASeriesID:    def.SeriesID,
		Currency:       "USD",
		IsActive:       true,
	}
	if _, err = tx.NewInsert().Model(index).Exec(ctx); err != nil {
		return pulid.Nil, fmt.Errorf("insert fuel index %s: %w", def.Code, err)
	}
	if err = sc.TrackCreated(ctx, "fuel_indices", index.ID, s.Name()); err != nil {
		return pulid.Nil, fmt.Errorf("track fuel index: %w", err)
	}

	offset := decimal.RequireFromString(seedRegionOffsets[def.Code])
	prices := make([]*fuelsurcharge.FuelIndexPrice, 0, len(seedWeeklyPrices))
	for i, priceValue := range seedWeeklyPrices {
		weeksBack := len(seedWeeklyPrices) - 1 - i
		price := decimal.RequireFromString(priceValue).Add(offset)
		prices = append(prices, &fuelsurcharge.FuelIndexPrice{
			ID:             pulid.MustNew("fip_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			FuelIndexID:    index.ID,
			PriceDate: monday.AddDate(0, 0, -7*weeksBack).
				Format(fuelsurcharge.PriceDateLayout),
			Price:     price,
			Currency:  "USD",
			SourceRaw: price.StringFixed(3),
		})
	}
	if _, err = tx.NewInsert().Model(&prices).Exec(ctx); err != nil {
		return pulid.Nil, fmt.Errorf("insert fuel index prices %s: %w", def.Code, err)
	}
	for _, price := range prices {
		if err = sc.TrackCreated(ctx, "fuel_index_prices", price.ID, s.Name()); err != nil {
			return pulid.Nil, fmt.Errorf("track fuel index price: %w", err)
		}
	}

	return index.ID, nil
}

func currentSeedMonday() time.Time {
	now := time.Now().UTC()
	daysSinceMonday := (int(now.Weekday()) - int(time.Monday) + 7) % 7
	monday := now.AddDate(0, 0, -daysSinceMonday)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *FuelSurchargeSeed) Down(ctx context.Context, tx bun.Tx) error {
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

func (s *FuelSurchargeSeed) CanRollback() bool {
	return true
}
