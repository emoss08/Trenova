package iftaservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/iftaservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	tractorA = pulid.MustNew("trac_")
	tractorB = pulid.MustNew("trac_")
	diesel   = domaintypes.IFTAFuelTypeDiesel
	gasoline = domaintypes.IFTAFuelTypeGasoline
	period   = ifta.NewPeriod(2026, 2)
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

type world struct {
	jurisdictions map[pulid.ID]*ifta.Jurisdiction
	tractors      map[pulid.ID]iftaservice.TractorProfile
	miles         *repositories.MileAccumulation
	fuel          []*repositories.FuelAccumulationRow
	rates         map[ifta.RateKey]*ifta.TaxRate
	byCode        map[string]*ifta.Jurisdiction
}

func newWorld() *world {
	w := &world{
		jurisdictions: map[pulid.ID]*ifta.Jurisdiction{},
		tractors: map[pulid.ID]iftaservice.TractorProfile{
			tractorA: {FuelType: diesel, IFTAQualified: true},
			tractorB: {FuelType: diesel, IFTAQualified: false},
		},
		miles:  &repositories.MileAccumulation{},
		rates:  map[ifta.RateKey]*ifta.TaxRate{},
		byCode: map[string]*ifta.Jurisdiction{},
	}
	w.jurisdiction("TX", "Texas", true, false)
	w.jurisdiction("OK", "Oklahoma", true, false)
	w.jurisdiction("IN", "Indiana", true, true)
	w.jurisdiction("AK", "Alaska", false, false)
	w.jurisdiction("AL", "Alabama", true, false)
	return w
}

func (w *world) jurisdiction(code, name string, member, surcharge bool) *ifta.Jurisdiction {
	j := &ifta.Jurisdiction{
		ID:           pulid.MustNew("ifj_"),
		CountryCode:  ifta.CountryCodeUS,
		Code:         code,
		Name:         name,
		IsIftaMember: member,
		HasSurcharge: surcharge,
		Status:       ifta.JurisdictionStatusActive,
	}
	w.jurisdictions[j.ID] = j
	w.byCode[code] = j
	return j
}

func (w *world) route(tractor pulid.ID, code, loaded, empty string) {
	j := w.byCode[code]
	w.miles.RouteRows = append(w.miles.RouteRows, &repositories.MileRow{
		TractorID:        tractor,
		CountryCode:      j.CountryCode,
		JurisdictionCode: j.Code,
		JurisdictionID:   j.ID,
		Miles:            dec(loaded).Add(dec(empty)),
		LoadedMiles:      dec(loaded),
		EmptyMiles:       dec(empty),
		MoveCount:        1,
	})
}

func (w *world) manual(tractor pulid.ID, code, miles string) {
	j := w.byCode[code]
	w.miles.ManualRows = append(w.miles.ManualRows, &repositories.MileRow{
		TractorID:        tractor,
		CountryCode:      j.CountryCode,
		JurisdictionCode: j.Code,
		JurisdictionID:   j.ID,
		Miles:            dec(miles),
		LoadedMiles:      dec(miles),
		MoveCount:        1,
	})
}

func (w *world) purchase(
	tractor pulid.ID,
	code string,
	fuelType domaintypes.IFTAFuelType,
	gallons, taxPaid string,
) {
	w.fuel = append(w.fuel, &repositories.FuelAccumulationRow{
		TractorID:      tractor,
		JurisdictionID: w.byCode[code].ID,
		FuelType:       fuelType,
		Gallons:        dec(gallons),
		TaxPaidGallons: dec(taxPaid),
		PurchaseCount:  1,
	})
}

func (w *world) rate(code string, fuelType domaintypes.IFTAFuelType, rate, surcharge string) {
	j := w.byCode[code]
	r := &ifta.TaxRate{
		ID:             pulid.MustNew("iftr_"),
		JurisdictionID: j.ID,
		Year:           period.Year,
		Quarter:        period.Quarter,
		FuelType:       fuelType,
		RatePerGallon:  dec(rate),
	}
	if surcharge != "" {
		r.SurchargeRatePerGallon = decimal.NewNullDecimal(dec(surcharge))
	}
	w.rates[r.Key()] = r
}

func (w *world) compute() iftaservice.ComputeResult {
	return iftaservice.Compute(iftaservice.ComputeInput{
		Period:        period,
		Jurisdictions: w.jurisdictions,
		Tractors:      w.tractors,
		Miles:         w.miles,
		Fuel:          w.fuel,
		Rates:         w.rates,
	})
}

func lineFor(
	t *testing.T,
	result iftaservice.ComputeResult,
	w *world,
	code string,
	fuelType domaintypes.IFTAFuelType,
) *ifta.ReturnLine {
	t.Helper()
	for _, line := range result.Lines {
		if line.JurisdictionID == w.byCode[code].ID && line.FuelType == fuelType {
			return line
		}
	}
	require.Failf(t, "line missing", "no %s line for %s", fuelType, code)
	return nil
}

func problemCodes(result iftaservice.ComputeResult) []ifta.ProblemCode {
	codes := make([]ifta.ProblemCode, 0, len(result.Problems))
	for _, p := range result.Problems {
		codes = append(codes, p.Code)
	}
	return codes
}

func mpgOf(t *testing.T, result iftaservice.ComputeResult, fuelType domaintypes.IFTAFuelType) ifta.FleetMPG {
	t.Helper()
	for _, entry := range result.FleetMPG {
		if entry.FuelType == fuelType {
			return entry
		}
	}
	require.Failf(t, "fleet mpg missing", "no fleet MPG for %s", fuelType)
	return ifta.FleetMPG{}
}

func TestCompute_CreditAndDebitLinesCarryTheirSign(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "TX", "600", "0")
	w.route(tractorA, "OK", "400", "0")
	w.purchase(tractorA, "TX", diesel, "150", "150")
	w.rate("TX", diesel, "0.2000", "")
	w.rate("OK", diesel, "0.1900", "")

	result := w.compute()

	assert.Equal(t, "6.67", mpgOf(t, result, diesel).MPG)

	tx := lineFor(t, result, w, "TX", diesel)
	assert.Equal(t, "90", tx.TaxableGallons.String())
	assert.Equal(t, "150", tx.TaxPaidGallons.String())
	assert.Equal(t, "-60", tx.NetTaxableGallons.String())
	assert.Equal(t, int64(-1200), tx.TaxDueMinor, "credit is negative")
	assert.True(t, tx.IsCredit())

	ok := lineFor(t, result, w, "OK", diesel)
	assert.Equal(t, "60", ok.TaxableGallons.String())
	assert.Equal(t, int64(1140), ok.TaxDueMinor)

	assert.Equal(t, "0", result.Totals.NetTaxableGallons.String())
	assert.Equal(t, int64(-60), result.Totals.TaxDueMinor)
	assert.Equal(t, int64(-60), result.Totals.NetDueMinor)
	assert.Equal(t, "1000", result.Totals.TotalMiles.String())
	assert.Equal(t, "150", result.Totals.TotalGallons.String())
	assert.Empty(t, problemCodes(result))
}

func TestCompute_SurchargeIsChargedOnTaxableGallonsEvenOnACreditLine(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "IN", "600", "0")
	w.purchase(tractorA, "IN", diesel, "100", "100")
	w.rate("IN", diesel, "0.5500", "0.1100")

	result := w.compute()

	in := lineFor(t, result, w, "IN", diesel)
	assert.Equal(t, "6.00", mpgOf(t, result, diesel).MPG)
	assert.Equal(t, "100", in.TaxableGallons.String())
	assert.Equal(t, "0", in.NetTaxableGallons.String())
	assert.Equal(t, int64(0), in.TaxDueMinor)
	assert.Equal(t, int64(1100), in.SurchargeDueMinor, "surcharge on gallons consumed")
	assert.Equal(t, int64(1100), in.LineTotalMinor)
	assert.True(t, in.SurchargeRatePerGallon.Valid)

	w2 := newWorld()
	w2.route(tractorA, "IN", "300", "0")
	w2.route(tractorA, "TX", "300", "0")
	w2.purchase(tractorA, "IN", diesel, "100", "100")
	w2.rate("IN", diesel, "0.5500", "0.1100")
	w2.rate("TX", diesel, "0.2000", "")
	credit := lineFor(t, w2.compute(), w2, "IN", diesel)
	assert.Equal(t, "-50", credit.NetTaxableGallons.String())
	assert.Equal(t, int64(-2750), credit.TaxDueMinor)
	assert.Equal(t, int64(550), credit.SurchargeDueMinor, "surcharge never becomes a credit")
	assert.Equal(t, int64(-2200), credit.LineTotalMinor)
}

func TestCompute_SurchargeJurisdictionWithoutSurchargeRateBlocks(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "IN", "600", "0")
	w.purchase(tractorA, "IN", diesel, "100", "100")
	w.rate("IN", diesel, "0.5500", "")

	result := w.compute()

	in := lineFor(t, result, w, "IN", diesel)
	assert.False(t, in.RateMissing)
	assert.Equal(t, int64(0), in.SurchargeDueMinor)
	assert.Contains(t, problemCodes(result), ifta.ProblemMissingRate)
}

func TestCompute_DeadheadMilesCount(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "OK", "250", "150")
	w.purchase(tractorA, "OK", diesel, "40", "40")
	w.rate("OK", diesel, "0.1900", "")

	result := w.compute()

	ok := lineFor(t, result, w, "OK", diesel)
	assert.Equal(t, "400", ok.TotalMiles.String())
	assert.Equal(t, "400", ok.TaxableMiles.String())
	assert.Equal(t, "250.00", ok.LoadedMiles.StringFixed(2))
	assert.Equal(t, "150.00", ok.EmptyMiles.StringFixed(2))
	assert.Equal(t, "400.00", ok.RouteMiles.StringFixed(2))
	assert.Equal(t, "10.00", mpgOf(t, result, diesel).MPG)
}

func TestCompute_MissingRateFlagsTheLineAndBlocksFinalize(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "TX", "500", "0")
	w.route(tractorA, "OK", "500", "0")
	w.purchase(tractorA, "TX", diesel, "100", "100")
	w.rate("TX", diesel, "0.2000", "")

	result := w.compute()

	ok := lineFor(t, result, w, "OK", diesel)
	assert.True(t, ok.RateMissing)
	assert.False(t, ok.RatePerGallon.Valid)
	assert.Equal(t, int64(0), ok.TaxDueMinor)
	assert.Equal(t, "50", ok.NetTaxableGallons.String(), "gallons still computed")

	tx := lineFor(t, result, w, "TX", diesel)
	assert.False(t, tx.RateMissing)
	assert.Equal(t, int64(-1000), tx.TaxDueMinor)

	ret := &ifta.Return{Status: ifta.ReturnStatusDraft, Problems: result.Problems}
	assert.False(t, ret.CanFinalize())
	assert.Len(t, ret.BlockingProblems(), 1)
	assert.Equal(t, "US_OK", ret.BlockingProblems()[0].JurisdictionCode)
}

func TestCompute_NonMemberMilesFeedMPGButCarryNoTax(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "TX", "800", "0")
	w.route(tractorA, "AK", "200", "0")
	w.purchase(tractorA, "TX", diesel, "100", "100")
	w.purchase(tractorA, "AK", diesel, "20", "20")
	w.rate("TX", diesel, "0.2000", "")

	result := w.compute()

	assert.Equal(t, "8.33", mpgOf(t, result, diesel).MPG, "1000 miles / 120 gallons")
	tx := lineFor(t, result, w, "TX", diesel)
	assert.Equal(t, "96", tx.TaxableGallons.String())

	ak := lineFor(t, result, w, "AK", diesel)
	assert.False(t, ak.IsIftaMember)
	assert.Equal(t, "200", ak.TotalMiles.String())
	assert.Equal(t, "0", ak.TaxableMiles.String())
	assert.Equal(t, "0", ak.TaxableGallons.String())
	assert.Equal(t, "0", ak.TaxPaidGallons.String())
	assert.Equal(t, "20.000", ak.TaxPaidGallonsRaw.StringFixed(3))
	assert.Equal(t, int64(0), ak.LineTotalMinor)
	assert.False(t, ak.RateMissing)

	assert.Equal(t, "1000", result.Totals.TotalMiles.String())
	assert.Equal(t, "800", result.Totals.TotalTaxableMiles.String())
	assert.Equal(t, "120", result.Totals.TotalGallons.String())
	assert.Equal(t, "100", result.Totals.TotalTaxPaidGallons.String())
	assert.Contains(t, problemCodes(result), ifta.ProblemNonMemberActivity)
	assert.NotContains(t, problemCodes(result), ifta.ProblemMissingRate)
}

func TestCompute_UntaxedGallonsCountForMPGButEarnNoCredit(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "TX", "1000", "0")
	w.purchase(tractorA, "TX", diesel, "100", "60")
	w.rate("TX", diesel, "0.2000", "")

	result := w.compute()

	assert.Equal(t, "10.00", mpgOf(t, result, diesel).MPG)
	tx := lineFor(t, result, w, "TX", diesel)
	assert.Equal(t, "100", tx.TaxableGallons.String())
	assert.Equal(t, "60", tx.TaxPaidGallons.String())
	assert.Equal(t, "40", tx.NetTaxableGallons.String())
	assert.Equal(t, int64(800), tx.TaxDueMinor)
	assert.Equal(t, "100", result.Totals.TotalGallons.String())
}

func TestCompute_RoundsHalfAwayFromZero(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "TX", "1001", "0")
	w.purchase(tractorA, "TX", diesel, "200", "195")
	w.rate("TX", diesel, "0.2010", "")

	result := w.compute()

	assert.Equal(t, "5.01", mpgOf(t, result, diesel).MPG, "5.005 rounds up, not to even")
	tx := lineFor(t, result, w, "TX", diesel)
	assert.Equal(t, "200", tx.TaxableGallons.String(), "1001 / 5.01 = 199.80")
	assert.Equal(t, "5", tx.NetTaxableGallons.String())
	assert.Equal(t, int64(101), tx.TaxDueMinor, "1.005 rounds to 1.01, not 1.00")

	w2 := newWorld()
	w2.route(tractorA, "TX", "1001", "0")
	w2.purchase(tractorA, "TX", diesel, "200", "205")
	w2.rate("TX", diesel, "0.2010", "")
	credit := lineFor(t, w2.compute(), w2, "TX", diesel)
	assert.Equal(t, "-5", credit.NetTaxableGallons.String())
	assert.Equal(t, int64(-101), credit.TaxDueMinor, "-1.005 rounds away from zero")
}

func TestCompute_ZeroGallonsForAFuelTypeIsFlagged(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "TX", "500", "0")
	w.rate("TX", diesel, "0.2000", "")

	result := w.compute()

	tx := lineFor(t, result, w, "TX", diesel)
	assert.Equal(t, "0", tx.TaxableGallons.String())
	assert.Equal(t, "0", tx.NetTaxableGallons.String())
	assert.Equal(t, int64(0), tx.TaxDueMinor)
	assert.Empty(t, mpgOf(t, result, diesel).MPG)
	assert.Contains(t, problemCodes(result), ifta.ProblemNoFuelForType)
	assert.NotContains(t, problemCodes(result), ifta.ProblemMissingRate)
}

func TestCompute_FuelForAnotherFuelTypeLandsOnItsOwnLine(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "TX", "500", "0")
	w.purchase(tractorA, "TX", diesel, "50", "50")
	w.purchase(tractorA, "TX", gasoline, "20", "20")
	w.rate("TX", diesel, "0.2000", "")
	w.rate("TX", gasoline, "0.2000", "")

	result := w.compute()

	gas := lineFor(t, result, w, "TX", gasoline)
	assert.Equal(t, "0", gas.TotalMiles.String())
	assert.Equal(t, "20", gas.TaxPaidGallons.String())
	assert.Equal(t, "-20", gas.NetTaxableGallons.String())
	assert.Empty(t, mpgOf(t, result, gasoline).MPG)
	assert.Contains(t, problemCodes(result), ifta.ProblemNoFuelForType)
	assert.Equal(t, "70", result.Totals.TotalGallons.String())
}

func TestCompute_LitrePurchasesArriveAsGallons(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "TX", "264.17", "0")
	w.purchase(tractorA, "TX", diesel, "26.417", "26.417")
	w.rate("TX", diesel, "0.2000", "")

	result := w.compute()

	assert.Equal(t, "10.00", mpgOf(t, result, diesel).MPG)
	tx := lineFor(t, result, w, "TX", diesel)
	assert.Equal(t, "26.417", tx.TaxPaidGallonsRaw.StringFixed(3))
	assert.Equal(t, "26", tx.TaxPaidGallons.String())
	assert.Equal(t, "264", tx.TaxableMiles.String())
	assert.Equal(t, "26", tx.TaxableGallons.String())
	assert.Equal(t, "26.417", result.Totals.TotalGallons.StringFixed(3))
}

func TestCompute_NonQualifiedTractorsAreExcludedAndReported(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "TX", "500", "0")
	w.route(tractorB, "TX", "700", "0")
	w.manual(tractorB, "OK", "100")
	w.purchase(tractorA, "TX", diesel, "50", "50")
	w.purchase(tractorB, "TX", diesel, "80", "80")
	w.rate("TX", diesel, "0.2000", "")

	result := w.compute()

	require.Len(t, result.Lines, 1)
	tx := lineFor(t, result, w, "TX", diesel)
	assert.Equal(t, "500", tx.TotalMiles.String())
	assert.Equal(t, "50", tx.TaxPaidGallons.String())
	assert.Equal(t, "50", result.Totals.TotalGallons.String())
	assert.Equal(t, "800.00", result.NonQualifiedMiles.StringFixed(2))
	assert.Equal(t, "80.000", result.NonQualifiedGallons.StringFixed(3))
	assert.Contains(t, problemCodes(result), ifta.ProblemNonQualifiedActivity)
}

func TestCompute_FleetMPGMatchesStoredTotals(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "TX", "750", "0")
	w.manual(tractorA, "OK", "500")
	w.purchase(tractorA, "TX", diesel, "120", "120")
	w.purchase(tractorA, "OK", diesel, "80", "80")
	w.rate("TX", diesel, "0.2000", "")
	w.rate("OK", diesel, "0.1900", "")

	result := w.compute()

	fleet := mpgOf(t, result, diesel)
	assert.Equal(t, "1250.00", fleet.TotalMiles)
	assert.Equal(t, "200.000", fleet.TotalGallons)
	assert.Equal(t, "6.25", fleet.MPG)
	ratio := dec(fleet.TotalMiles).Div(dec(fleet.TotalGallons))
	assert.Equal(t, ratio.StringFixed(4), dec(fleet.MPG).StringFixed(4))

	ok := lineFor(t, result, w, "OK", diesel)
	assert.Equal(t, "500.00", ok.ManualMiles.StringFixed(2))
	assert.Equal(t, "0.00", ok.RouteMiles.StringFixed(2))
}

func TestCompute_LinesSortByFuelTypeThenJurisdiction(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.tractors[tractorB] = iftaservice.TractorProfile{FuelType: gasoline, IFTAQualified: true}
	w.route(tractorA, "TX", "100", "0")
	w.route(tractorA, "OK", "100", "0")
	w.route(tractorB, "AL", "100", "0")
	w.purchase(tractorA, "TX", diesel, "20", "20")
	w.purchase(tractorB, "AL", gasoline, "10", "10")
	w.rate("TX", diesel, "0.2000", "")
	w.rate("OK", diesel, "0.1900", "")
	w.rate("AL", gasoline, "0.1800", "")

	result := w.compute()

	require.Len(t, result.Lines, 3)
	assert.Equal(t, w.byCode["OK"].ID, result.Lines[0].JurisdictionID)
	assert.Equal(t, w.byCode["TX"].ID, result.Lines[1].JurisdictionID)
	assert.Equal(t, w.byCode["AL"].ID, result.Lines[2].JurisdictionID)
	assert.Equal(t, gasoline, result.Lines[2].FuelType)
	for i, line := range result.Lines {
		assert.Equal(t, i, line.SortOrder)
	}
}

func TestCompute_DiagnosticsBecomeProblems(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "TX", "100", "0")
	w.miles.RouteRows = append(w.miles.RouteRows, &repositories.MileRow{
		CountryCode:      "US",
		JurisdictionCode: "TX",
		JurisdictionID:   w.byCode["TX"].ID,
		Miles:            dec("50"),
		LoadedMiles:      dec("50"),
		MoveCount:        1,
	})
	w.miles.RouteRows = append(w.miles.RouteRows, &repositories.MileRow{
		TractorID:        tractorA,
		CountryCode:      "MX",
		JurisdictionCode: "CH",
		Miles:            dec("30"),
		LoadedMiles:      dec("30"),
		MoveCount:        1,
	})
	w.miles.Unattributed = repositories.MoveAggregate{Miles: dec("300"), MoveCount: 2}
	w.miles.NoTractor = repositories.MoveAggregate{Miles: dec("50"), MoveCount: 1}
	w.miles.Mismatches = []repositories.MoveMismatch{{MoveID: pulid.MustNew("sm_")}}
	w.purchase(tractorA, "TX", diesel, "10", "10")
	w.rate("TX", diesel, "0.2000", "")

	result := w.compute()

	codes := problemCodes(result)
	assert.Contains(t, codes, ifta.ProblemUnattributedMiles)
	assert.Contains(t, codes, ifta.ProblemNoTractorMiles)
	assert.Contains(t, codes, ifta.ProblemMileageMismatch)
	assert.Contains(t, codes, ifta.ProblemNonMemberActivity)
	assert.Equal(t, 2, result.Unattributed.MoveCount)
	assert.Equal(t, "300.00", result.Unattributed.Miles.StringFixed(2))
	assert.Equal(t, 1, result.MismatchCount)

	tx := lineFor(t, result, w, "TX", diesel)
	assert.Equal(t, "100", tx.TotalMiles.String(), "no-tractor and unknown rows stay off the line")

	ret := &ifta.Return{Status: ifta.ReturnStatusDraft, Problems: result.Problems}
	assert.True(t, ret.CanFinalize(), "diagnostics warn but do not block")
}

func TestCompute_EveryLineValidates(t *testing.T) {
	t.Parallel()
	w := newWorld()
	w.route(tractorA, "TX", "600", "40")
	w.route(tractorA, "IN", "300", "0")
	w.route(tractorA, "AK", "100", "0")
	w.purchase(tractorA, "TX", diesel, "150", "150")
	w.purchase(tractorA, "IN", diesel, "20", "0")
	w.rate("TX", diesel, "0.2000", "")
	w.rate("IN", diesel, "0.5500", "0.1100")

	result := w.compute()

	require.Len(t, result.Lines, 3)
	for _, line := range result.Lines {
		line.ReturnID = pulid.MustNew("ifr_")
		multiErr := errortypes.NewMultiError()
		line.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	}
}
