package detentionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The contract's price is what makes the rate confirmation and the invoice
// agree on detention: both read it from the same place rather than one reading
// the organization default.
func TestContractPrices_PricedRowBeatsTheOrganizationDefault(t *testing.T) {
	t.Parallel()

	charge := &accessorialcharge.AccessorialCharge{
		ID:       pulid.MustNew("acc_"),
		Code:     "DET",
		Method:   accessorialcharge.MethodPerUnit,
		RateUnit: accessorialcharge.RateUnitHour,
		Amount:   decimal.NewFromInt(50),
	}

	prices := &contractPrices{
		byAccessorial: map[pulid.ID]*rateagreement.RateAgreementAccessorial{
			charge.ID: {
				AccessorialChargeID: charge.ID,
				Method:              accessorialcharge.MethodPerUnit,
				RateUnit:            accessorialcharge.RateUnitHour,
				Amount:              decimal.NewFromInt(65),
			},
		},
	}

	rate, unit, priced := prices.priceFor(charge)

	require.True(t, priced)
	assert.True(t, decimal.NewFromInt(65).Equal(rate), "got %s", rate.String())
	assert.Equal(t, accessorialcharge.RateUnitHour, unit)
}

// A contract giving detention away is a stated term. Falling through to the
// organization default would bill the customer for something they were promised
// they would not see.
func TestContractPrices_WaivedRowStillCountsAsPriced(t *testing.T) {
	t.Parallel()

	charge := &accessorialcharge.AccessorialCharge{
		ID:     pulid.MustNew("acc_"),
		Amount: decimal.NewFromInt(50),
	}

	prices := &contractPrices{
		byAccessorial: map[pulid.ID]*rateagreement.RateAgreementAccessorial{
			charge.ID: {
				AccessorialChargeID: charge.ID,
				Waived:              true,
				Amount:              decimal.Zero,
			},
		},
	}

	rate, _, priced := prices.priceFor(charge)

	require.True(t, priced, "a waiver is a price of zero, not an absence of one")
	assert.True(t, rate.IsZero())
}

// A schedule row that does not state its own unit inherits the accessorial's,
// so a contract only has to say what it actually renegotiated.
func TestContractPrices_RowWithoutAUnitInheritsTheAccessorialsOwn(t *testing.T) {
	t.Parallel()

	charge := &accessorialcharge.AccessorialCharge{
		ID:       pulid.MustNew("acc_"),
		RateUnit: accessorialcharge.RateUnitDay,
		Amount:   decimal.NewFromInt(50),
	}

	prices := &contractPrices{
		byAccessorial: map[pulid.ID]*rateagreement.RateAgreementAccessorial{
			charge.ID: {AccessorialChargeID: charge.ID, Amount: decimal.NewFromInt(400)},
		},
	}

	_, unit, priced := prices.priceFor(charge)

	require.True(t, priced)
	assert.Equal(t, accessorialcharge.RateUnitDay, unit)
}

func TestContractPrices_AccessorialTheContractDoesNotNameIsUnpriced(t *testing.T) {
	t.Parallel()

	prices := &contractPrices{
		byAccessorial: map[pulid.ID]*rateagreement.RateAgreementAccessorial{
			pulid.MustNew("acc_"): {Amount: decimal.NewFromInt(65)},
		},
	}

	_, _, priced := prices.priceFor(&accessorialcharge.AccessorialCharge{
		ID:     pulid.MustNew("acc_"),
		Amount: decimal.NewFromInt(50),
	})

	assert.False(t, priced, "an accessorial the contract is silent about keeps the org default")
}

// A shipment with no contract has to keep working exactly as it did before rate
// agreements existed, which means the nil case cannot panic.
func TestContractPrices_NoContractLeavesTheOrganizationDefault(t *testing.T) {
	t.Parallel()

	var prices *contractPrices

	_, _, priced := prices.priceFor(&accessorialcharge.AccessorialCharge{
		ID:     pulid.MustNew("acc_"),
		Amount: decimal.NewFromInt(50),
	})

	assert.False(t, priced)
}
