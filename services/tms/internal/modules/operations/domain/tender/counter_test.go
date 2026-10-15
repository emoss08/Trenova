package tender_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestACounterThatClearsTheFloorIsTakenOnlyWhenAsked(t *testing.T) {
	terms := tender.CounterTerms{
		Sell:  decimal.NewFromInt(2000),
		Buy:   decimal.NewFromInt(1700),
		Floor: decimal.NewFromInt(10),
	}

	verdict := tender.EvaluateCounter(terms)
	assert.False(t, verdict.BelowFloor)
	assert.False(t, verdict.NeedsApproval)
	assert.False(t, verdict.AutoAccept, "without the setting a person still decides")
	assert.True(t, decimal.NewFromInt(15).Equal(verdict.MarginPercent.Decimal))

	terms.AutoAccept = true
	verdict = tender.EvaluateCounter(terms)
	assert.True(t, verdict.AutoAccept)
	assert.False(t, verdict.NeedsApproval)
}

func TestACounterBelowTheFloorNeedsApproval(t *testing.T) {
	verdict := tender.EvaluateCounter(tender.CounterTerms{
		Sell:       decimal.NewFromInt(2000),
		Buy:        decimal.NewFromInt(1850),
		Floor:      decimal.NewFromInt(10),
		AutoAccept: true,
	})

	assert.True(t, verdict.BelowFloor)
	assert.True(t, verdict.NeedsApproval)
	assert.False(t, verdict.AutoAccept, "auto-accept never reaches below the floor")
	assert.True(t, decimal.NewFromFloat(7.5).Equal(verdict.MarginPercent.Decimal))
	assert.Equal(t, "the counter leaves a margin of 7.5%, below the 10% floor", verdict.Explanation)
}

func TestACounterExactlyAtTheFloorClearsIt(t *testing.T) {
	verdict := tender.EvaluateCounter(tender.CounterTerms{
		Sell:       decimal.NewFromInt(2000),
		Buy:        decimal.NewFromInt(1800),
		Floor:      decimal.NewFromInt(10),
		AutoAccept: true,
	})

	assert.False(t, verdict.BelowFloor)
	assert.True(t, verdict.AutoAccept)
}

func TestACounterOnALoadWithNoSellPriceWaitsForAPerson(t *testing.T) {
	verdict := tender.EvaluateCounter(tender.CounterTerms{
		Sell:       decimal.Zero,
		Buy:        decimal.NewFromInt(1800),
		Floor:      decimal.NewFromInt(10),
		AutoAccept: true,
	})

	assert.True(t, verdict.NeedsApproval)
	assert.False(t, verdict.AutoAccept)
	assert.False(t, verdict.MarginPercent.Valid)
}
