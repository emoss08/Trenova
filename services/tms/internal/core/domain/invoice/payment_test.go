package invoice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestApplyPaymentMinorUpdatesSettlementState(t *testing.T) {
	t.Parallel()

	entity := &Invoice{BillType: billingqueue.BillTypeInvoice, TotalAmount: decimal.NewFromInt(100), TotalAmountMinor: 10000, AppliedAmount: decimal.Zero, AppliedAmountMinor: 0, SettlementStatus: SettlementStatusUnpaid}

	entity.ApplyPaymentMinor(2500)
	assert.Equal(t, int64(2500), entity.AppliedAmountMinor)
	assert.True(t, decimal.RequireFromString("25.00").Equal(entity.AppliedAmount))
	assert.Equal(t, SettlementStatusPartiallyPaid, entity.SettlementStatus)
	assert.Equal(t, int64(7500), entity.OpenBalanceMinor())

	entity.ApplyPaymentMinor(7500)
	assert.Equal(t, int64(10000), entity.AppliedAmountMinor)
	assert.Equal(t, SettlementStatusPaid, entity.SettlementStatus)
	assert.Equal(t, int64(0), entity.OpenBalanceMinor())
}

func TestRemovePaymentMinorUpdatesSettlementState(t *testing.T) {
	t.Parallel()

	entity := &Invoice{BillType: billingqueue.BillTypeInvoice, TotalAmount: decimal.NewFromInt(100), TotalAmountMinor: 10000, AppliedAmount: decimal.RequireFromString("100.00"), AppliedAmountMinor: 10000, SettlementStatus: SettlementStatusPaid}

	entity.RemovePaymentMinor(2500)
	assert.Equal(t, int64(7500), entity.AppliedAmountMinor)
	assert.True(t, decimal.RequireFromString("75.00").Equal(entity.AppliedAmount))
	assert.Equal(t, SettlementStatusPartiallyPaid, entity.SettlementStatus)

	entity.RemovePaymentMinor(7500)
	assert.Equal(t, int64(0), entity.AppliedAmountMinor)
	assert.Equal(t, SettlementStatusUnpaid, entity.SettlementStatus)
	assert.Equal(t, int64(10000), entity.OpenBalanceMinor())
}
