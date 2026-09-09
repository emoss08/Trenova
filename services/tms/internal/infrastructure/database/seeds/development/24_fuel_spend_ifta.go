package development

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	fuelSpendSeedDay          = int64(86400)
	fuelSpendSeedTractorCount = 2
	fuelSpendSeedCardYears    = 2
	fuelSpendSeedRateNote     = "Development seed — illustrative, not published rates"

	fuelSpendJurisdictionTX = "US_TX"
	fuelSpendJurisdictionOK = "US_OK"
	fuelSpendJurisdictionIN = "US_IN"
	fuelSpendJurisdictionON = "CA_ON"

	fuelSpendCardComdataLastFour = "4412"
	fuelSpendCardWEXLastFour     = "0931"
)

type FuelSpendIFTASeed struct {
	seedhelpers.BaseSeed
}

func NewFuelSpendIFTASeed() *FuelSpendIFTASeed {
	seed := &FuelSpendIFTASeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"FuelSpendIFTA",
		"1.0.0",
		"Seeds fuel cards, fuel purchases, IFTA tax rates and jurisdiction mileage for the last two completed quarters",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(
		seedhelpers.SeedIFTAJurisdictions,
		seedhelpers.SeedTestData,
	)
	return seed
}

type fuelSpendSeedRefs struct {
	orgID         pulid.ID
	buID          pulid.ID
	adminID       pulid.ID
	now           int64
	periods       [2]ifta.Period
	tractors      []*tractor.Tractor
	cards         []*fuelpurchase.FuelCard
	jurisdictions map[string]pulid.ID
	odometers     []int64
}

type fuelPurchasePlan struct {
	tractor      int
	period       int
	day          int
	jurisdiction string
	vendor       string
	city         string
	fuelType     domaintypes.IFTAFuelType
	quantity     string
	unit         fuelpurchase.QuantityUnit
	unitPrice    string
	currency     string
	taxPaid      bool
	useCard      bool
	milesDriven  int64
	notes        string
}

type taxRatePlan struct {
	jurisdiction string
	fuelType     domaintypes.IFTAFuelType
	rate         string
	surcharge    string
}

type mileagePlan struct {
	tractor      int
	period       int
	day          int
	jurisdiction string
	miles        string
	loaded       bool
	notes        string
}

func (s *FuelSpendIFTASeed) Run(ctx context.Context, tx bun.Tx) error {
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

			exists, err := tx.NewSelect().
				Model((*fuelpurchase.FuelCard)(nil)).
				Where(buncolgen.FuelCardColumns.OrganizationID.Eq(), org.ID).
				Where(buncolgen.FuelCardColumns.BusinessUnitID.Eq(), org.BusinessUnitID).
				Exists(ctx)
			if err != nil {
				return fmt.Errorf("check existing fuel cards: %w", err)
			}
			if exists {
				return nil
			}

			current := ifta.PeriodOf(timeutils.NowUnix(), time.UTC)
			refs := &fuelSpendSeedRefs{
				orgID:   org.ID,
				buID:    org.BusinessUnitID,
				adminID: admin.ID,
				now:     timeutils.NowUnix(),
				periods: [2]ifta.Period{current.Previous(), current},
			}

			if err = s.loadTractors(ctx, tx, refs); err != nil {
				return fmt.Errorf("load tractors: %w", err)
			}
			if len(refs.tractors) < fuelSpendSeedTractorCount {
				sc.Logger().Warn(
					"FuelSpendIFTA seed skipped: found %d tractors, need %d",
					len(refs.tractors),
					fuelSpendSeedTractorCount,
				)
				return nil
			}
			if err = s.loadJurisdictions(ctx, tx, refs); err != nil {
				return fmt.Errorf("load ifta jurisdictions: %w", err)
			}

			if err = s.seedCards(ctx, tx, sc, refs); err != nil {
				return fmt.Errorf("seed fuel cards: %w", err)
			}
			if err = s.seedPurchases(ctx, tx, sc, refs); err != nil {
				return fmt.Errorf("seed fuel purchases: %w", err)
			}
			if err = s.seedTaxRates(ctx, tx, sc, refs); err != nil {
				return fmt.Errorf("seed ifta tax rates: %w", err)
			}
			if err = s.seedMileage(ctx, tx, sc, refs); err != nil {
				return fmt.Errorf("seed jurisdiction mileage: %w", err)
			}
			return nil
		},
	)
}

func (s *FuelSpendIFTASeed) loadTractors(
	ctx context.Context,
	tx bun.Tx,
	refs *fuelSpendSeedRefs,
) error {
	tractors := make([]*tractor.Tractor, 0, fuelSpendSeedTractorCount)
	cols := buncolgen.TractorColumns
	if err := tx.NewSelect().
		Model(&tractors).
		Where(cols.OrganizationID.Eq(), refs.orgID).
		Where(cols.BusinessUnitID.Eq(), refs.buID).
		Order(cols.Code.OrderAsc()).
		Limit(fuelSpendSeedTractorCount).
		Scan(ctx); err != nil {
		return err
	}
	refs.tractors = tractors
	refs.odometers = []int64{412_350, 287_910}
	return nil
}

func (s *FuelSpendIFTASeed) loadJurisdictions(
	ctx context.Context,
	tx bun.Tx,
	refs *fuelSpendSeedRefs,
) error {
	wanted := []string{
		fuelSpendJurisdictionTX,
		fuelSpendJurisdictionOK,
		fuelSpendJurisdictionIN,
		fuelSpendJurisdictionON,
	}
	rows := make([]*ifta.Jurisdiction, 0, len(wanted))
	cols := buncolgen.JurisdictionColumns
	if err := tx.NewSelect().
		Model(&rows).
		Where(cols.CountryCode.In(), bun.List([]string{ifta.CountryCodeUS, ifta.CountryCodeCA})).
		Where(cols.Code.In(), bun.List([]string{"TX", "OK", "IN", "ON"})).
		Scan(ctx); err != nil {
		return err
	}

	refs.jurisdictions = make(map[string]pulid.ID, len(wanted))
	for _, row := range rows {
		refs.jurisdictions[row.Key()] = row.ID
	}
	for _, key := range wanted {
		if _, ok := refs.jurisdictions[key]; !ok {
			return fmt.Errorf("ifta jurisdiction %s is not seeded", key)
		}
	}
	return nil
}

func (s *FuelSpendIFTASeed) seedCards(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *fuelSpendSeedRefs,
) error {
	expires := time.Unix(refs.now, 0).UTC().AddDate(fuelSpendSeedCardYears, 0, 0).Unix()
	first, second := refs.tractors[0], refs.tractors[1]
	firstWorker := first.PrimaryWorkerID

	cards := []*fuelpurchase.FuelCard{
		{
			OrganizationID:    refs.orgID,
			BusinessUnitID:    refs.buID,
			Provider:          fuelpurchase.CardProviderComdata,
			LastFour:          fuelSpendCardComdataLastFour,
			Label:             fmt.Sprintf("Comdata •••• %s — Unit %s", fuelSpendCardComdataLastFour, first.Code),
			ExternalCardID:    "CMD-7781-4412",
			AssignedWorkerID:  &firstWorker,
			AssignedTractorID: &first.ID,
			Status:            fuelpurchase.CardStatusActive,
			ExpiresAt:         &expires,
			Notes:             "Primary over-the-road card; diesel and DEF only.",
		},
		{
			OrganizationID:    refs.orgID,
			BusinessUnitID:    refs.buID,
			Provider:          fuelpurchase.CardProviderWEX,
			LastFour:          fuelSpendCardWEXLastFour,
			Label:             fmt.Sprintf("WEX •••• %s — Unit %s", fuelSpendCardWEXLastFour, second.Code),
			ExternalCardID:    "WEX-2210-0931",
			AssignedTractorID: &second.ID,
			Status:            fuelpurchase.CardStatusActive,
			ExpiresAt:         &expires,
		},
	}

	for _, card := range cards {
		card.Normalize()
		if err := validateSeedEntity("fuel card "+card.LastFour, card.Validate); err != nil {
			return err
		}
		if _, err := tx.NewInsert().Model(card).Exec(ctx); err != nil {
			return err
		}
		if err := sc.TrackCreated(ctx, "fuel_cards", card.ID, s.Name()); err != nil {
			return err
		}
	}
	refs.cards = cards
	return nil
}

func (s *FuelSpendIFTASeed) purchasePlan() []fuelPurchasePlan {
	return []fuelPurchasePlan{
		{
			tractor: 0, period: 0, day: 4,
			jurisdiction: fuelSpendJurisdictionTX, vendor: "Pilot Travel Center", city: "Amarillo",
			fuelType: domaintypes.IFTAFuelTypeDiesel, quantity: "132.500", unit: fuelpurchase.QuantityUnitGallon,
			unitPrice: "3.6890", currency: money.DefaultCurrencyCode, taxPaid: true, useCard: true,
			milesDriven: 0,
		},
		{
			tractor: 1, period: 0, day: 6,
			jurisdiction: fuelSpendJurisdictionOK, vendor: "Love's Travel Stop", city: "Oklahoma City",
			fuelType: domaintypes.IFTAFuelTypeDiesel, quantity: "118.200", unit: fuelpurchase.QuantityUnitGallon,
			unitPrice: "3.7590", currency: money.DefaultCurrencyCode, taxPaid: true, useCard: true,
			milesDriven: 0,
		},
		{
			tractor: 0, period: 0, day: 19,
			jurisdiction: fuelSpendJurisdictionOK, vendor: "Love's Travel Stop", city: "Tulsa",
			fuelType: domaintypes.IFTAFuelTypeDiesel, quantity: "96.400", unit: fuelpurchase.QuantityUnitGallon,
			unitPrice: "3.7290", currency: money.DefaultCurrencyCode, taxPaid: true, useCard: true,
			milesDriven: 742,
		},
		{
			tractor: 0, period: 0, day: 33,
			jurisdiction: fuelSpendJurisdictionON, vendor: "Petro-Canada", city: "Windsor",
			fuelType: domaintypes.IFTAFuelTypeDiesel, quantity: "420.000", unit: fuelpurchase.QuantityUnitLitre,
			unitPrice: "1.5590", currency: "CAD", taxPaid: true, useCard: true,
			milesDriven: 688,
			notes:       "Cross-border fill; receipt in litres and CAD.",
		},
		{
			tractor: 1, period: 0, day: 41,
			jurisdiction: fuelSpendJurisdictionTX, vendor: "Flying J", city: "Dallas",
			fuelType: domaintypes.IFTAFuelTypeDiesel, quantity: "145.000", unit: fuelpurchase.QuantityUnitGallon,
			unitPrice: "3.6490", currency: money.DefaultCurrencyCode, taxPaid: true, useCard: true,
			milesDriven: 1_106,
		},
		{
			tractor: 0, period: 0, day: 58,
			jurisdiction: fuelSpendJurisdictionTX, vendor: "Pilot Travel Center", city: "Amarillo",
			fuelType: domaintypes.IFTAFuelTypeDEF, quantity: "15.000", unit: fuelpurchase.QuantityUnitGallon,
			unitPrice: "3.2990", currency: money.DefaultCurrencyCode, taxPaid: false, useCard: true,
			milesDriven: 903,
			notes:       "DEF top-up; not a taxable motor fuel.",
		},
		{
			tractor: 1, period: 1, day: 3,
			jurisdiction: fuelSpendJurisdictionOK, vendor: "Love's Travel Stop", city: "Oklahoma City",
			fuelType: domaintypes.IFTAFuelTypeDiesel, quantity: "122.800", unit: fuelpurchase.QuantityUnitGallon,
			unitPrice: "3.8190", currency: money.DefaultCurrencyCode, taxPaid: true, useCard: true,
			milesDriven: 964,
		},
		{
			tractor: 0, period: 1, day: 9,
			jurisdiction: fuelSpendJurisdictionTX, vendor: "Flying J", city: "Fort Worth",
			fuelType: domaintypes.IFTAFuelTypeDiesel, quantity: "151.300", unit: fuelpurchase.QuantityUnitGallon,
			unitPrice: "3.8990", currency: money.DefaultCurrencyCode, taxPaid: true, useCard: true,
			milesDriven: 1_015,
		},
		{
			tractor: 0, period: 1, day: 22,
			jurisdiction: fuelSpendJurisdictionON, vendor: "Petro-Canada", city: "London",
			fuelType: domaintypes.IFTAFuelTypeDiesel, quantity: "380.000", unit: fuelpurchase.QuantityUnitLitre,
			unitPrice: "1.5790", currency: "CAD", taxPaid: true, useCard: true,
			milesDriven: 811,
		},
		{
			tractor: 1, period: 1, day: 27,
			jurisdiction: fuelSpendJurisdictionTX, vendor: "Yard bulk tank", city: "Houston",
			fuelType: domaintypes.IFTAFuelTypeDiesel, quantity: "160.000", unit: fuelpurchase.QuantityUnitGallon,
			unitPrice: "3.4890", currency: money.DefaultCurrencyCode, taxPaid: false, useCard: false,
			milesDriven: 1_188,
			notes:       "Tax-unpaid bulk fill from the yard tank; tax is settled on the return.",
		},
		{
			tractor: 1, period: 1, day: 47,
			jurisdiction: fuelSpendJurisdictionTX, vendor: "Pilot Travel Center", city: "San Antonio",
			fuelType: domaintypes.IFTAFuelTypeDiesel, quantity: "108.600", unit: fuelpurchase.QuantityUnitGallon,
			unitPrice: "4.0190", currency: money.DefaultCurrencyCode, taxPaid: true, useCard: true,
			milesDriven: 872,
		},
		{
			tractor: 0, period: 1, day: 61,
			jurisdiction: fuelSpendJurisdictionOK, vendor: "Flying J", city: "Oklahoma City",
			fuelType: domaintypes.IFTAFuelTypeDiesel, quantity: "139.900", unit: fuelpurchase.QuantityUnitGallon,
			unitPrice: "4.1290", currency: money.DefaultCurrencyCode, taxPaid: true, useCard: true,
			milesDriven: 1_047,
		},
	}
}

func (s *FuelSpendIFTASeed) seedPurchases(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *fuelSpendSeedRefs,
) error {
	plans := s.purchasePlan()
	for i, plan := range plans {
		trc := refs.tractors[plan.tractor]
		card := refs.cards[plan.tractor]
		workerID := trc.PrimaryWorkerID
		refs.odometers[plan.tractor] += plan.milesDriven
		odometer := refs.odometers[plan.tractor]

		quantity, err := decimal.NewFromString(plan.quantity)
		if err != nil {
			return fmt.Errorf("parse quantity for purchase %d: %w", i+1, err)
		}
		unitPrice, err := decimal.NewFromString(plan.unitPrice)
		if err != nil {
			return fmt.Errorf("parse unit price for purchase %d: %w", i+1, err)
		}

		purchase := &fuelpurchase.FuelPurchase{
			OrganizationID:       refs.orgID,
			BusinessUnitID:       refs.buID,
			TractorID:            trc.ID,
			WorkerID:             &workerID,
			JurisdictionID:       refs.jurisdictions[plan.jurisdiction],
			PurchasedAt:          seedPeriodInstant(refs.periods[plan.period], plan.day, 9),
			Vendor:               plan.vendor,
			VendorCity:           plan.city,
			FuelType:             plan.fuelType,
			Quantity:             quantity,
			QuantityUnit:         plan.unit,
			UnitPrice:            decimal.NewNullDecimal(unitPrice),
			TotalAmountMinor:     money.MinorUnits(quantity.Mul(unitPrice)),
			CurrencyCode:         plan.currency,
			Odometer:             &odometer,
			TransactionReference: fmt.Sprintf("DEV-FP-%04d", i+1),
			Source:               fuelpurchase.PurchaseSourceManual,
			TaxPaid:              plan.taxPaid,
			Notes:                plan.notes,
			CreatedByID:          refs.adminID,
		}
		if plan.useCard {
			purchase.FuelCardID = &card.ID
			purchase.CardLastFour = card.LastFour
		}

		purchase.Normalize()
		if err = validateSeedEntity(
			"fuel purchase "+purchase.TransactionReference,
			purchase.Validate,
		); err != nil {
			return err
		}
		if _, err = tx.NewInsert().Model(purchase).Exec(ctx); err != nil {
			return err
		}
		if err = sc.TrackCreated(ctx, "fuel_purchases", purchase.ID, s.Name()); err != nil {
			return err
		}
	}
	return nil
}

func (s *FuelSpendIFTASeed) taxRatePlan() []taxRatePlan {
	return []taxRatePlan{
		{jurisdiction: fuelSpendJurisdictionTX, fuelType: domaintypes.IFTAFuelTypeDiesel, rate: "0.2000"},
		{jurisdiction: fuelSpendJurisdictionTX, fuelType: domaintypes.IFTAFuelTypeGasoline, rate: "0.2000"},
		{jurisdiction: fuelSpendJurisdictionOK, fuelType: domaintypes.IFTAFuelTypeDiesel, rate: "0.1900"},
		{jurisdiction: fuelSpendJurisdictionOK, fuelType: domaintypes.IFTAFuelTypeGasoline, rate: "0.1900"},
		{jurisdiction: fuelSpendJurisdictionON, fuelType: domaintypes.IFTAFuelTypeDiesel, rate: "0.4300"},
		{jurisdiction: fuelSpendJurisdictionON, fuelType: domaintypes.IFTAFuelTypeGasoline, rate: "0.4100"},
		{jurisdiction: fuelSpendJurisdictionIN, fuelType: domaintypes.IFTAFuelTypeDiesel, rate: "0.5500", surcharge: "0.1100"},
		{jurisdiction: fuelSpendJurisdictionIN, fuelType: domaintypes.IFTAFuelTypeGasoline, rate: "0.3400", surcharge: "0.1100"},
	}
}

func (s *FuelSpendIFTASeed) seedTaxRates(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *fuelSpendSeedRefs,
) error {
	plans := s.taxRatePlan()
	for _, period := range refs.periods {
		for _, plan := range plans {
			rate, err := decimal.NewFromString(plan.rate)
			if err != nil {
				return fmt.Errorf("parse rate for %s %s: %w", plan.jurisdiction, plan.fuelType, err)
			}
			entity := &ifta.TaxRate{
				JurisdictionID: refs.jurisdictions[plan.jurisdiction],
				Year:           period.Year,
				Quarter:        period.Quarter,
				FuelType:       plan.fuelType,
				RatePerGallon:  rate,
				SourceNote:     fuelSpendSeedRateNote,
			}
			if plan.surcharge != "" {
				surcharge, parseErr := decimal.NewFromString(plan.surcharge)
				if parseErr != nil {
					return fmt.Errorf(
						"parse surcharge for %s %s: %w",
						plan.jurisdiction,
						plan.fuelType,
						parseErr,
					)
				}
				entity.SurchargeRatePerGallon = decimal.NewNullDecimal(surcharge)
			}

			entity.Normalize()
			if err = validateSeedEntity(
				fmt.Sprintf("ifta tax rate %s %s %s", plan.jurisdiction, plan.fuelType, period.Key()),
				entity.Validate,
			); err != nil {
				return err
			}

			result, err := tx.NewInsert().
				Model(entity).
				On("CONFLICT (jurisdiction_id, year, quarter, fuel_type) DO NOTHING").
				Exec(ctx)
			if err != nil {
				return err
			}
			inserted, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("read inserted rows: %w", err)
			}
			if inserted == 0 {
				continue
			}
			if err = sc.TrackCreated(ctx, "ifta_tax_rates", entity.ID, s.Name()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *FuelSpendIFTASeed) mileagePlan() []mileagePlan {
	return []mileagePlan{
		{
			tractor: 0, period: 1, day: 9,
			jurisdiction: fuelSpendJurisdictionOK, miles: "212.40", loaded: false,
			notes: "Deadhead from Tulsa back to the Fort Worth yard.",
		},
		{
			tractor: 1, period: 1, day: 27,
			jurisdiction: fuelSpendJurisdictionTX, miles: "640.00", loaded: true,
			notes: "Houston to Amarillo, loaded.",
		},
		{
			tractor: 0, period: 0, day: 33,
			jurisdiction: fuelSpendJurisdictionON, miles: "150.00", loaded: true,
			notes: "Windsor to London and back to the border.",
		},
	}
}

func (s *FuelSpendIFTASeed) seedMileage(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *fuelSpendSeedRefs,
) error {
	for i, plan := range s.mileagePlan() {
		miles, err := decimal.NewFromString(plan.miles)
		if err != nil {
			return fmt.Errorf("parse miles for mileage entry %d: %w", i+1, err)
		}
		entry := &ifta.JurisdictionMileageEntry{
			OrganizationID: refs.orgID,
			BusinessUnitID: refs.buID,
			TractorID:      refs.tractors[plan.tractor].ID,
			JurisdictionID: refs.jurisdictions[plan.jurisdiction],
			TraveledAt:     seedPeriodInstant(refs.periods[plan.period], plan.day, 14),
			Miles:          miles,
			Loaded:         plan.loaded,
			Source:         ifta.MileageSourceManual,
			Notes:          plan.notes,
			CreatedByID:    refs.adminID,
		}
		entry.AssignPeriod(time.UTC)
		entry.Normalize()
		if err = validateSeedEntity(
			fmt.Sprintf("mileage entry %d", i+1),
			entry.Validate,
		); err != nil {
			return err
		}
		if _, err = tx.NewInsert().Model(entry).Exec(ctx); err != nil {
			return err
		}
		if err = sc.TrackCreated(
			ctx,
			"ifta_jurisdiction_mileage_entries",
			entry.ID,
			s.Name(),
		); err != nil {
			return err
		}
	}
	return nil
}

func seedPeriodInstant(period ifta.Period, day, hour int) int64 {
	start, _ := period.Bounds(time.UTC)
	instant := start + int64(day)*fuelSpendSeedDay + int64(hour)*3600

	if now := timeutils.NowUnix(); instant > now {
		return now - fuelSpendSeedDay
	}

	return instant
}

func validateSeedEntity(label string, validate func(*errortypes.MultiError)) error {
	multiErr := errortypes.NewMultiError()
	validate(multiErr)
	if multiErr.HasErrors() {
		return fmt.Errorf("%s is invalid: %w", label, multiErr)
	}
	return nil
}

func (s *FuelSpendIFTASeed) Down(ctx context.Context, tx bun.Tx) error {
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

func (s *FuelSpendIFTASeed) CanRollback() bool {
	return true
}
