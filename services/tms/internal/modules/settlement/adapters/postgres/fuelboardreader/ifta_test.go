package fuelboardreader

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelboard"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetTaxReadsOwedCreditAndAMissingRate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	owed := netTaxReading(ctx, &lineRow{
		LineTotalMinor: 61032, TaxPaidGallons: decimal.NewFromInt(456), Currency: "USD",
	})
	assert.Equal(t, "{0} owed", owed.Standing.Text)
	assert.Equal(t, "610.32", owed.Standing.Value.Number.StringFixed(2))
	assert.Equal(t, "610.32", owed.Centred.Signed.StringFixed(2))
	assert.Equal(t, "{0} gal bought", owed.Fact.Text)
	assert.Equal(t, "456", owed.Fact.Value.Number.String())

	credit := netTaxReading(ctx, &lineRow{LineTotalMinor: -135240, Currency: "USD"})
	assert.Equal(t, "{0} credit", credit.Standing.Text)
	assert.Equal(t, "1352.40", credit.Standing.Value.Number.StringFixed(2))
	assert.True(t, credit.Centred.Signed.IsNegative())
	assert.True(t, credit.Value.Number.IsNegative(), "a credit adds up as a credit")

	missing := netTaxReading(ctx, &lineRow{RateMissing: true, Currency: "USD"})
	assert.Equal(t, "Rate missing", missing.Standing.Text)
	assert.True(t, missing.Centred.Missing)
	assert.Equal(t, boardcatalog.ToneBad, missing.Tone)
	assert.True(t, missing.Value.Number.IsZero(), "an unpriced line adds nothing")
}

func TestALineOpensItsReturn(t *testing.T) {
	t.Parallel()

	returnID := pulid.MustNew("ifr_")
	summary := lineSummary(context.Background(), &lineRow{
		ReturnID: returnID, Jurisdiction: "Oklahoma", FuelType: "Diesel",
		TotalMiles: decimal.NewFromInt(21340),
	})
	assert.Equal(t, "Oklahoma", summary.Label)
	assert.Equal(t, "Diesel, 21,340 mi", summary.Subtitle)
	require.NotNil(t, summary.Opens)
	assert.Equal(t, recordkind.IFTAReturn, summary.Opens.Kind)
	assert.Equal(t, returnID, summary.Opens.ID)
}

func draftReturn(loc *time.Location) *ifta.Return {
	start, end := ifta.NewPeriod(2026, 3).Bounds(loc)

	return &ifta.Return{
		Year: 2026, Quarter: 3, Status: ifta.ReturnStatusDraft, Timezone: loc.String(),
		PeriodStart: start, PeriodEnd: end, NetDueMinor: 25826, CurrencyCode: "USD",
		UnattributedMiles: decimal.NewFromInt(412),
		Problems: []ifta.Problem{
			{
				Code:             ifta.ProblemMissingRate,
				Message:          "No Diesel tax rate is on file for Oklahoma in Q3 2026",
				JurisdictionCode: "US_OK",
			},
			{
				Code:    ifta.ProblemUnattributedMiles,
				Message: "3 completed moves totalling 412.00 miles have no jurisdiction breakdown",
				Amount:  "412.00",
			},
		},
	}
}

func TestTheChecksPutWhatBlocksFirstAndSayWhatIsClear(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	now := time.Date(2026, 11, 23, 9, 41, 0, 0, loc).Unix()
	line := pulid.MustNew("ifrl_")
	read := checksRead(
		context.Background(),
		&boardreaderports.MetricRequest{Now: now},
		&returnFacts{
			ret:     draftReturn(loc),
			missing: []namedLine{{ID: line, Jurisdiction: "Oklahoma"}},
			nMissed: 1,
			credit:  &namedLine{Jurisdiction: "Texas", LineTotalMinor: -135240},
		},
	)

	require.NotEmpty(t, read.Checks)
	first := read.Checks[0]
	assert.Equal(t, "Missing tax rate", first.Title)
	assert.Equal(t, boardcatalog.ToneBad, first.Tone)
	assert.Equal(t, []boardcatalog.Ref{{Kind: recordkind.IFTAReturnLine, ID: line}}, first.Refs)

	titles := make(map[string]boardcatalog.Check, len(read.Checks))
	for _, check := range read.Checks {
		titles[check.Title] = check
	}
	unplaced := titles["Miles not attributed to a jurisdiction"]
	assert.Equal(t, boardcatalog.ToneWarn, unplaced.Tone)
	assert.Equal(t, "412 mi", unplaced.Count.Text)
	late := titles["Past the filing date"]
	assert.Equal(t, boardcatalog.ToneWarn, late.Tone)
	assert.Equal(t, "23", late.Count.Number.String(), "Oct 31 to Nov 23")
	assert.Contains(t, late.Detail, "Oct 31.")
	assert.Equal(t, boardcatalog.ToneOK, titles["Every move has a tractor"].Tone)
	assert.NotContains(t, titles, "The quarter is still running")

	assert.Equal(t, "1", read.Figures[fuelboard.FigureFilingBlockers].Number.String())
	assert.Equal(t, "258.26", read.Figures[fuelboard.FigureNetTaxDue].Number.StringFixed(2))
	assert.Equal(t, "1352.40", read.Figures[fuelboard.FigureLargestCredit].Number.StringFixed(2))
	assert.Equal(t, "Texas", read.Figures[fuelboard.FigureLargestCreditFor].Text)
	assert.Equal(t, "Oct 31", read.Figures[fuelboard.FigureFilingDue].Text)
	assert.Equal(t, "Q3 2026 IFTA return", read.Title)
}

func TestAnOpenQuarterIsStillRunningAndNotLate(t *testing.T) {
	t.Parallel()

	loc := time.UTC
	ret := draftReturn(loc)
	ret.Quarter = 4
	ret.PeriodStart, ret.PeriodEnd = ifta.NewPeriod(2026, 4).Bounds(loc)
	ret.Problems = nil
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, loc).Unix()
	read := checksRead(context.Background(), &boardreaderports.MetricRequest{Now: now},
		&returnFacts{ret: ret})

	titles := map[string]boardcatalog.Check{}
	for _, check := range read.Checks {
		titles[check.Title] = check
	}
	running := titles["The quarter is still running"]
	assert.Equal(t, boardcatalog.ToneWarn, running.Tone)
	assert.Equal(t, "37", running.Count.Number.String(), "Nov 24 to Dec 31")
	assert.NotContains(t, titles, "Past the filing date")
	assert.Equal(t, "0", read.Figures[fuelboard.FigureFilingBlockers].Number.String())
	assert.Nil(t, read.Figures[fuelboard.FigureLargestCreditFor], "no line is a credit")
	assert.Equal(t, "Jan 31, 2027", read.Figures[fuelboard.FigureFilingDue].Text)
}

func TestAReturnNotReadReadsAsNothing(t *testing.T) {
	t.Parallel()

	read, err := (&IFTAChecks{}).Read(context.Background(), &boardreaderports.MetricRequest{})
	require.NoError(t, err)
	assert.Nil(t, read)
}
