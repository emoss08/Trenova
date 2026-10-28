package shipmentrating

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// Clearing an override is a deliberate act, never a side effect of saving
// something else. A client that round-trips a shipment without the rating
// fields must not thereby unlock an invoiced load or wipe a negotiated rate.
func TestRestoreRateOwnedFields_CallersCannotClearAnOverride(t *testing.T) {
	t.Parallel()

	setBy := pulid.MustNew("usr_")
	setAt := int64(500)
	quoteID := pulid.MustNew("rqt_")
	agreementID := pulid.MustNew("ragr_")
	ruleID := pulid.MustNew("ragr_")

	original := &shipment.Shipment{
		RateOverrideAmount:  decimal.NewNullDecimal(decimal.NewFromInt(1400)),
		RateOverrideReason:  "Spot rate agreed with the customer",
		RateOverrideByID:    &setBy,
		RateOverrideAt:      &setAt,
		RateLocked:          true,
		RateQuoteID:         &quoteID,
		RateAgreementID:     &agreementID,
		RateAgreementRuleID: &ruleID,
	}

	updated := &shipment.Shipment{}

	RestoreRateOwnedFields(original, updated)

	require.True(t, updated.HasRateOverride())
	require.True(t, decimal.NewFromInt(1400).Equal(updated.RateOverrideAmount.Decimal))
	require.Equal(t, "Spot rate agreed with the customer", updated.RateOverrideReason)
	require.Equal(t, setBy, *updated.RateOverrideByID)
	require.Equal(t, setAt, *updated.RateOverrideAt)
	require.True(t, updated.RateLocked)
	require.Equal(t, quoteID, *updated.RateQuoteID)
	require.Equal(t, agreementID, *updated.RateAgreementID)
	require.Equal(t, ruleID, *updated.RateAgreementRuleID)
}

// The rating detail explains an invoice. A client echoing back the handful of
// fields it renders must not thereby erase the contract and the bill that
// priced a locked shipment, which is never re-rated.
func TestRestoreRateOwnedFields_RatingDetailComesFromTheRaterNotThePayload(t *testing.T) {
	t.Parallel()

	original := &shipment.Shipment{
		RateLocked: true,
		RatingDetail: &shipment.RatingDetail{
			FormulaTemplateID: "ft_01JT",
			AgreementName:     "Acme Master Contract",
			RuleLabel:         "DFW → LAX dry van",
			Source:            "Rated",
			Explanation:       "Priced from the Acme contract",
			Bill: &tarifftypes.Bill{
				Lines: []tarifftypes.Line{
					{
						Kind:   tarifftypes.LineCharge,
						Label:  "Linehaul",
						Amount: decimal.NewFromInt(1200),
					},
					{Kind: tarifftypes.LineCharge, Code: "FSC", Amount: decimal.NewFromInt(200)},
				},
				Total: decimal.NewFromInt(1400),
			},
		},
	}

	updated := &shipment.Shipment{
		RatingDetail: &shipment.RatingDetail{FormulaTemplateID: "ft_01JT"},
	}

	RestoreRateOwnedFields(original, updated)

	require.Equal(t, original.RatingDetail, updated.RatingDetail)
	require.NotNil(t, updated.RatingDetail.Bill)
	require.Len(t, updated.RatingDetail.Bill.Lines, 2)
	require.Equal(t, "Acme Master Contract", updated.RatingDetail.AgreementName)
}

// A forged override on a create — where there is no original to compare
// against — is left alone here; the create path never calls this.
func TestRestoreRateOwnedFields_IgnoresAMissingSide(t *testing.T) {
	t.Parallel()

	updated := &shipment.Shipment{RateLocked: true}

	RestoreRateOwnedFields(nil, updated)
	RestoreRateOwnedFields(&shipment.Shipment{}, nil)

	require.True(t, updated.RateLocked)
}
