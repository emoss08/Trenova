package aitraining

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rateConfirmation = `ACME FREIGHT BROKERAGE, LLC
Rate Confirmation  Load # AFB-7731902
Carrier: Northstar Hauling Inc  MC# 884213  USDOT 2931177
Dispatcher: Dana Whitfield  (214) 555-0199  dana.whitfield@northstarhauling.com
Shipper: Pacific Coast Produce Co.
1450 Harbor Industrial Road, Oakland, CA 94607
Pickup 03/14/2026 08:00-14:00  PU# 55120934
Consignee: Rocky Mountain Grocers
8800 E 40th Ave, Denver, CO 80239-1102
Deliver 03/16/2026  Appt required
Linehaul: $2,450.00   Fuel: $310.50   Total: $2,760.50 USD
Weight 42000 lbs  Pieces 24  Commodity: Fresh Produce
Remit to PO Box 88120, visit www.acmefreight.com or acmefreight.com
Tax ID 12-3456789`

func sampleSource() *Source {
	target := &aicorrection.Snapshot{
		Fields: map[string]string{
			aicorrection.FieldReference: "AFB-7731902",
			aicorrection.FieldRate:      "2760.50",
			aicorrection.FieldShipper:   "Pacific Coast Produce",
			aicorrection.FieldConsignee: "Rocky Mountain Grocers",
			aicorrection.FieldWeight:    "42000",
			aicorrection.FieldPieces:    "24",
			aicorrection.FieldCommodity: "Fresh Produce",
		},
		Stops: []aicorrection.StopSnapshot{
			{
				Role:         aicorrection.RolePickup,
				Name:         "Pacific Coast Produce",
				AddressLine1: "1450 Harbor Industrial Rd",
				City:         "Oakland",
				State:        "CA",
				PostalCode:   "94607",
				Date:         "2026-03-14",
			},
			{
				Role:         aicorrection.RoleDelivery,
				Name:         "Rocky Mountain Grocers",
				AddressLine1: "8800 East 40th Avenue",
				City:         "Denver",
				State:        "CO",
				PostalCode:   "80239",
				Date:         "2026-03-16",
			},
		},
	}
	prediction := &aicorrection.Snapshot{
		Fields: map[string]string{
			"loadNumber":                "AFB-7731902",
			"pickupNumber":              "55120934",
			"carrierName":               "Northstar Hauling Inc",
			aicorrection.FieldRate:      "$2,450.00",
			aicorrection.FieldShipper:   "Pacific Coast Produce Co.",
			aicorrection.FieldConsignee: "Rocky Mountain Grocers",
		},
		Stops: target.Stops,
	}

	return &Source{
		FileName:   "AFB-7731902 Rate Con Northstar.PDF",
		Pages:      []ExamplePage{{Number: 1, Text: rateConfirmation}},
		Target:     target,
		Prediction: prediction,
		Issuer:     "Acme Freight Brokerage",
	}
}

func sampleIdentity() *Identity {
	return NewIdentity(&IdentityValues{
		Parties:     []string{"Northstar Hauling, Inc."},
		People:      []string{"Dana Whitfield"},
		Identifiers: []string{"NSHL", "2931177"},
		Streets:     []string{"3100 Commerce Street"},
		Cities:      []string{"Irving"},
		PostalCodes: []string{"75062"},
	})
}

var decimalCent = decimal.RequireFromString("0.01")

func newRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
}

func TestAnonymizeRemovesKnownAndPatternIdentifiers(t *testing.T) {
	t.Parallel()

	out, err := Anonymize(sampleSource(), sampleIdentity(), newRNG(1))
	require.NoError(t, err)

	text := out.Pages[0].Text
	lower := strings.ToLower(text)
	for _, leaked := range []string{
		"acme freight", "northstar", "pacific coast produce", "rocky mountain grocers",
		"dana whitfield", "whitfield@", "northstarhauling", "acmefreight",
		"afb-7731902", "7731902", "884213", "2931177", "55120934", "214) 555-0199",
		"1450 harbor", "8800 e 40th", "oakland", "denver", "94607", "80239", "88120",
		"2,450.00", "310.50", "2,760.50", "12-3456789", "3456789",
	} {
		assert.NotContains(t, lower, leaked)
	}

	assert.Contains(t, text, "Rate Confirmation")
	assert.Contains(t, text, "03/14/2026")
	assert.Contains(t, text, "Weight 42000 lbs")
	assert.Contains(t, text, "Fresh Produce")
	assert.Contains(t, text, ", CA ")
	assert.Equal(t, "document.pdf", out.FileName)
}

func TestAnonymizeKeepsTargetConsistentWithText(t *testing.T) {
	t.Parallel()

	out, err := Anonymize(sampleSource(), sampleIdentity(), newRNG(7))
	require.NoError(t, err)
	text := out.Pages[0].Text

	reference := out.Target.Fields[aicorrection.FieldReference]
	require.NotEqual(t, "AFB-7731902", reference)
	assert.Contains(t, text, reference)
	assert.Equal(t, reference, out.Prediction.Fields["loadNumber"])

	shipper := out.Target.Fields[aicorrection.FieldShipper]
	assert.Contains(t, strings.ToLower(text), strings.ToLower(shipper))
	assert.Equal(t, shipper, out.Target.Stops[0].Name)
	assert.True(t, strings.HasPrefix(out.Prediction.Fields[aicorrection.FieldShipper], shipper))

	rate := out.Target.Fields[aicorrection.FieldRate]
	require.NotEqual(t, "2760.50", rate)
	assert.Contains(t, strings.ReplaceAll(text, ",", ""), rate)

	assert.Contains(t, text, out.Target.Stops[0].PostalCode)
	assert.Contains(t, text, out.Target.Stops[1].PostalCode)
	assert.Contains(t, text, out.Target.Stops[0].City)
	assert.Contains(t, strings.ToLower(text), strings.ToLower(out.Target.Stops[1].AddressLine1))

	assert.Equal(t, "42000", out.Target.Fields[aicorrection.FieldWeight])
	assert.Equal(t, "24", out.Target.Fields[aicorrection.FieldPieces])
	assert.Equal(t, "2026-03-14", out.Target.Stops[0].Date)
	assert.Equal(t, "CA", out.Target.Stops[0].State)
}

func TestAnonymizeVariesBetweenExamples(t *testing.T) {
	t.Parallel()

	first, err := Anonymize(sampleSource(), sampleIdentity(), newRNG(11))
	require.NoError(t, err)
	second, err := Anonymize(sampleSource(), sampleIdentity(), newRNG(12))
	require.NoError(t, err)

	assert.NotEqual(t, first.Pages[0].Text, second.Pages[0].Text)
}

func TestAnonymizeDropsWhenAKnownValueSurvives(t *testing.T) {
	t.Parallel()

	src := sampleSource()
	src.Pages = []ExamplePage{{Number: 1, Text: "Shipper: Pacific-Coast/Produce loads at dock"}}
	src.Target.Stops[0].Name = "Pacific Coast Produce"
	src.Target.Fields[aicorrection.FieldShipper] = "PacificCoastProduce Incorporated Holdings"

	_, err := Anonymize(src, nil, newRNG(3))
	require.NoError(t, err)

	src.Pages = []ExamplePage{{Number: 1, Text: "Ship from Pacific Coast Produce"}}
	src.Target.Fields["billTo"] = "Coastline"
	identity := NewIdentity(&IdentityValues{Cities: []string{"Coast Produce Line"}})
	out, err := Anonymize(src, identity, newRNG(3))
	require.NoError(t, err)
	assert.NotContains(t, strings.ToLower(out.Pages[0].Text), "pacific coast produce")
}

func TestResidualCheckRejectsSurvivors(t *testing.T) {
	t.Parallel()

	known := []*knownValue{partyKnown("Pacific Coast Produce"), identifierKnown("AFB-7731902")}
	assert.True(t, residual(&Anonymized{
		Pages: []ExamplePage{{Text: "pacific  coast produce"}},
	}, known))
	assert.True(t, residual(&Anonymized{
		Pages: []ExamplePage{{Text: "ref AFB 7731902"}},
	}, known))
	assert.False(t, residual(&Anonymized{
		Pages: []ExamplePage{{Text: "Harbor Supply"}},
	}, known))
}

func TestMoneyScalingKeepsFormat(t *testing.T) {
	t.Parallel()

	p := newPseudonymizer(newRNG(5))
	amount := moneyKnown("1500")
	require.NotNil(t, amount)

	grouped := p.money("$1,500", amount.amount)
	assert.Regexp(t, `^\$\d?,?\d{3}(\.\d{2})?$`, grouped)
	fixed := p.money("1500.00", amount.amount)
	assert.Regexp(t, `^\d+\.\d{2}$`, fixed)

	groupedValue, err := decimalutils.ParseMoneyText(grouped)
	require.NoError(t, err)
	fixedValue, err := decimalutils.ParseMoneyText(fixed)
	require.NoError(t, err)
	assert.True(t, groupedValue.Decimal.Equal(fixedValue.Decimal))
}

func TestIdentifierLike(t *testing.T) {
	t.Parallel()

	assert.True(t, identifierLike("TRLU4481203"))
	assert.True(t, identifierLike("AB12CD34"))
	assert.False(t, identifierLike("42000lbs"))
	assert.False(t, identifierLike("Produce"))
	assert.False(t, identifierLike("53FT"))
}

func TestCompactDatesAreKept(t *testing.T) {
	t.Parallel()

	out, err := Anonymize(&Source{
		Pages: []ExamplePage{{Number: 1, Text: "Ship date 20260314 ref 44871203"}},
	}, nil, newRNG(9))
	require.NoError(t, err)
	assert.Contains(t, out.Pages[0].Text, "20260314")
	assert.NotContains(t, out.Pages[0].Text, "44871203")
}

func TestAnonymousFileName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "document.pdf", anonymousFileName("Load 7731902 ACME.PDF"))
	assert.Equal(t, "document", anonymousFileName("scan"))
	assert.Equal(t, "document", anonymousFileName("weird.ext-with-dash"))
}

func TestMoneyScalingKeepsSums(t *testing.T) {
	t.Parallel()

	p := newPseudonymizer(newRNG(21))
	linehaul := p.scale(moneyKnown("2450.00").amount)
	fuel := p.scale(moneyKnown("310.50").amount)
	total := p.scale(moneyKnown("2760.50").amount)
	assert.True(t, linehaul.Add(fuel).Sub(total).Abs().LessThanOrEqual(decimalCent))
}

func TestEmailsAreReplacedWhole(t *testing.T) {
	t.Parallel()

	out, err := Anonymize(sampleSource(), sampleIdentity(), newRNG(7))
	require.NoError(t, err)
	assert.Regexp(t, `[a-z]+\.[a-z]+\d*@example\.com`, out.Pages[0].Text)
}
