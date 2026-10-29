package pricingfixtures

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge/fuelsurchargedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type FuelSurchargeSeed struct {
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

func (s *FuelSurchargeSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	chargeID, err := s.ensureFuelAccessorial(ctx, db, sr, org.ID, org.BusinessUnitID)
	if err != nil {
		return err
	}

	monday := currentSeedMonday()
	var usIndexID pulid.ID

	for _, def := range fuelsurcharge.EIASeriesRegistry() {
		indexID, iErr := s.ensureIndexWithPrices(
			ctx, db, sr, org.ID, org.BusinessUnitID, def, monday)
		if iErr != nil {
			return iErr
		}

		if def.Code == "DOE_US" {
			usIndexID = indexID
		}
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
		PercentBasis:         fuelsurcharge.PercentBasisLinehaul,
	}
	if err = fuelsurchargedb.FuelSurchargePrograms.Create(ctx, db, program); err != nil {
		return fmt.Errorf("insert fuel surcharge program: %w", err)
	}

	return nil
}

func (s *FuelSurchargeSeed) ensureFuelAccessorial(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
) (pulid.ID, error) {
	existing, err := accessorialchargedb.AccessorialCharges.AllTenants().Where(
		accessorialchargedb.AccessorialCharges.OrganizationID.Eq(orgID),
		accessorialchargedb.AccessorialCharges.BusinessUnitID.Eq(buID),
		accessorialchargedb.AccessorialCharges.Code.Eq("FUEL"),
	).First(ctx, db)
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
		Status:         domaintypes.StatusActive,
	}
	if err = accessorialchargedb.AccessorialCharges.Create(ctx, db, charge); err != nil {
		return pulid.Nil, fmt.Errorf("insert fuel accessorial charge: %w", err)
	}

	return charge.ID, nil
}

func (s *FuelSurchargeSeed) ensureIndexWithPrices(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
	def fuelsurcharge.EIASeriesDef,
	monday time.Time,
) (pulid.ID, error) {
	existing, err := fuelsurchargedb.FuelIndexes.AllTenants().Where(
		fuelsurchargedb.FuelIndexes.OrganizationID.Eq(orgID),
		fuelsurchargedb.FuelIndexes.BusinessUnitID.Eq(buID),
		fuelsurchargedb.FuelIndexes.Code.Eq(def.Code),
	).First(ctx, db)
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
	if err = fuelsurchargedb.FuelIndexes.Create(ctx, db, index); err != nil {
		return pulid.Nil, fmt.Errorf("insert fuel index %s: %w", def.Code, err)
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
	if err = fuelsurchargedb.FuelIndexPrices.CreateMany(ctx, db, prices); err != nil {
		return pulid.Nil, fmt.Errorf("insert fuel index prices %s: %w", def.Code, err)
	}

	return index.ID, nil
}

func currentSeedMonday() time.Time {
	now := time.Now().UTC()
	daysSinceMonday := (int(now.Weekday()) - int(time.Monday) + 7) % 7
	monday := now.AddDate(0, 0, -daysSinceMonday)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}
