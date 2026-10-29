package factoringremit_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/pkg/factoringremit"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fundingReport = "Schedule #,Invoice No,Debtor Name,Invoice Amount,Advance Amount,Fee,Reserve,Funding Date,Type\n" +
	"FS-1,INV-1001,Acme Foods,\"1,000.00\",900.00,30.00,70.00,10/07/2026,Purchase\n" +
	"FS-1,INV-1002,Acme Foods,500.00,450.00,15.00,35.00,10/07/2026,Declined\n" +
	"FS-1,INV-1003,Bolt Logistics,,(85.00),,,10/07/2026,Chargeback\n" +
	",,,,,,,,\n" +
	"Total,,,\"1,500.00\",\"1,265.00\",45.00,105.00,,\n"

func readCSV(t *testing.T, body string) *rateimport.Sheet {
	t.Helper()
	sheet, err := factoringremit.ReadSheet(
		[]byte(body),
		factoringremittance.RemittanceFormatCsv,
		rateimport.Limits{},
	)
	require.NoError(t, err)
	return sheet
}

func TestGuessMappingPlacesAFactorsUsualHeaders(t *testing.T) {
	t.Parallel()

	sheet := readCSV(t, fundingReport)
	mapping := factoringremit.GuessMapping(sheet.Headers)

	assert.Equal(t, factoringremit.Mapping{
		factoringremit.FieldFactorReference: 0,
		factoringremit.FieldInvoiceNumber:   1,
		factoringremit.FieldDebtor:          2,
		factoringremit.FieldFace:            3,
		factoringremit.FieldAdvance:         4,
		factoringremit.FieldFee:             5,
		factoringremit.FieldReserve:         6,
		factoringremit.FieldDate:            7,
		factoringremit.FieldKind:            8,
	}, mapping)
	assert.Empty(t, factoringremit.Validate(mapping))
	assert.Empty(t, factoringremit.UnmappedHeaders(sheet.Headers, mapping))
}

func TestValidateAsksForAnInvoiceColumnAndAnAmount(t *testing.T) {
	t.Parallel()

	problems := factoringremit.Validate(factoringremit.Mapping{factoringremit.FieldDebtor: 0})
	require.Len(t, problems, 2)
	assert.Equal(t, factoringremit.FieldInvoiceNumber, problems[0].Field)
	assert.Equal(t, factoringremit.FieldCash, problems[1].Field)

	clash := factoringremit.Validate(factoringremit.Mapping{
		factoringremit.FieldInvoiceNumber: 0,
		factoringremit.FieldCash:          1,
		factoringremit.FieldAdvance:       1,
	})
	require.Len(t, clash, 1)
	assert.Equal(t, "Amount paid and advance use the same column", clash[0].Message)
}

func TestParseReadsEachRowAsTheFactorWroteIt(t *testing.T) {
	t.Parallel()

	sheet := readCSV(t, fundingReport)
	lines := factoringremit.Parse(
		sheet,
		factoringremit.GuessMapping(sheet.Headers),
		factoringremit.Options{
			DefaultKind: factoringremittance.RemittanceLineFunding,
			Location:    time.UTC,
		},
	)
	require.Len(t, lines, 3, "the blank row and the totals row are not lines")

	funded := lines[0]
	assert.Equal(t, 2, funded.RowNumber)
	assert.Equal(t, factoringremittance.RemittanceLineFunding, funded.Kind)
	assert.Equal(t, "INV-1001", funded.InvoiceNumber)
	assert.Equal(t, "FS-1", funded.FactorReference)
	assert.Equal(t, "Acme Foods", funded.Debtor)
	assert.Equal(t, int64(100_000), *funded.FaceMinor)
	assert.Equal(t, int64(90_000), *funded.AdvanceMinor)
	assert.Equal(t, int64(3_000), *funded.FeeMinor)
	assert.Equal(t, int64(7_000), *funded.ReserveMinor)
	assert.Nil(t, funded.CashMinor)
	assert.Equal(t, time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC).Unix(), *funded.OccurredOn)
	assert.Empty(t, funded.Err)

	assert.Equal(t, factoringremittance.RemittanceLineRejection, lines[1].Kind)

	chargeback := lines[2]
	assert.Equal(t, factoringremittance.RemittanceLineChargeback, chargeback.Kind)
	assert.Equal(t, int64(-8_500), *chargeback.AdvanceMinor, "parenthesized amounts are negative")
	assert.Nil(t, chargeback.FaceMinor)
}

func TestParseKeepsABadRowAsAnErrorLine(t *testing.T) {
	t.Parallel()

	sheet := readCSV(
		t,
		"Invoice,Amount Paid,Date\nINV-9,twelve,10/07/2026\n,40.00,\nINV-10,15.00,someday\n",
	)
	lines := factoringremit.Parse(
		sheet,
		factoringremit.GuessMapping(sheet.Headers),
		factoringremit.Options{
			DefaultKind: factoringremittance.RemittanceLineReserveRelease,
		},
	)
	require.Len(t, lines, 3)
	assert.Equal(t, `Amount paid "twelve" is not an amount`, lines[0].Err)
	assert.Equal(t, "The row has no invoice number or reference", lines[1].Err)
	assert.Equal(t, `The date "someday" could not be read`, lines[2].Err)
	assert.Equal(
		t,
		factoringremittance.RemittanceLineReserveRelease,
		lines[2].Kind,
		"no type column: the report's kind",
	)

	domain := factoringremit.ToDomain(lines)
	require.Len(t, domain, 3)
	assert.Equal(t, factoringremittance.RemittanceLineError, domain[0].MatchStatus)
	assert.Equal(t, "INV9", domain[0].NormalizedInvoice)
}

func TestKindFromTextReadsTheWordsFactorsUse(t *testing.T) {
	t.Parallel()

	cases := map[string]factoringremittance.RemittanceLineKind{
		"Charge Back":       factoringremittance.RemittanceLineChargeback,
		"Recourse":          factoringremittance.RemittanceLineChargeback,
		"Reserve Release":   factoringremittance.RemittanceLineReserveRelease,
		"Rebate":            factoringremittance.RemittanceLineReserveRelease,
		"DECLINED":          factoringremittance.RemittanceLineRejection,
		"Non-Purchase":      factoringremittance.RemittanceLineRejection,
		"Purchase":          factoringremittance.RemittanceLineFunding,
		"Advance":           factoringremittance.RemittanceLineFunding,
		"Wire fee":          factoringremittance.RemittanceLineOther,
		"Something unusual": factoringremittance.RemittanceLineReserveRelease,
	}
	for text, want := range cases {
		assert.Equal(
			t,
			want,
			factoringremit.KindFromText(text, factoringremittance.RemittanceLineReserveRelease),
			text,
		)
	}
}

func TestMappingRoundTripsThroughItsStoredForm(t *testing.T) {
	t.Parallel()

	mapping := factoringremit.Mapping{
		factoringremit.FieldInvoiceNumber: 2,
		factoringremit.FieldCash:          5,
	}
	back, err := factoringremit.MappingFromStrings(mapping.Strings())
	require.NoError(t, err)
	assert.Equal(t, mapping, back)

	_, err = factoringremit.MappingFromStrings(map[string]int{"tip": 1})
	require.ErrorIs(t, err, factoringremit.ErrUnknownField)
}

func TestHeaderSignatureIgnoresCaseAndPunctuation(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		factoringremit.HeaderSignature([]string{"Invoice No.", "Amount Paid"}),
		factoringremit.HeaderSignature([]string{"invoice no", "AMOUNT PAID"}),
	)
	assert.NotEqual(t,
		factoringremit.HeaderSignature([]string{"Invoice", "Amount"}),
		factoringremit.HeaderSignature([]string{"Amount", "Invoice"}),
	)
}

func TestFormatForReadsTheExtensionThenTheContentType(t *testing.T) {
	t.Parallel()

	format, err := factoringremit.FormatFor("Remit 10-07.XLSX", "")
	require.NoError(t, err)
	assert.Equal(t, factoringremittance.RemittanceFormatXlsx, format)

	format, err = factoringremit.FormatFor("remit", "application/pdf")
	require.NoError(t, err)
	assert.Equal(t, factoringremittance.RemittanceFormatPdf, format)

	_, err = factoringremit.FormatFor("remit.docx", "application/msword")
	require.ErrorIs(t, err, factoringremit.ErrUnsupportedFile)
}
