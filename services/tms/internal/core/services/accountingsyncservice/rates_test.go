package accountingsyncservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *harness) multicurrencyBooks() {
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.ExternalMultiCurrencyEnabled = true
	})
}

func TestTheRateStampedAtPostingIsSentWithoutALookup(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.multicurrencyBooks()
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	inv := h.postedInvoice(customerID, invoiceSpec{currency: "CAD"})
	stamped := aprilTenth
	inv.ExchangeRate = decimal.NewNullDecimal(decimal.RequireFromString("0.7401"))
	inv.ExchangeRateDate = &stamped
	record := h.enqueueInvoice(t, inv)

	h.drain(t)

	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(record.ID).Status)
	decimalEqual(t, "0.7401", onlySalesDoc(t, h).ExchangeRate)
	assert.Empty(t, h.rates.requests(), "a stamped document never asks the rate table")
}

func TestAMissingRateIsLookedUpOnTheDocumentDateAndStampedOnTheInvoice(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.multicurrencyBooks()
	h.rates.set("CAD", "USD", decimal.RequireFromString("0.7312"))
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	inv := h.postedInvoice(customerID, invoiceSpec{currency: "CAD"})
	record := h.enqueueInvoice(t, inv)

	h.drain(t)

	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(record.ID).Status)
	assert.Equal(t, []string{"CAD/USD@2026-04-10"}, h.rates.requests())
	require.True(t, inv.ExchangeRate.Valid, "the rate the books received is kept on the invoice")
	decimalEqual(t, "0.7312", inv.ExchangeRate.Decimal)
	require.NotNil(t, inv.ExchangeRateDate)
	assert.Equal(t, aprilTenth, *inv.ExchangeRateDate)
}

func TestTheAccountingDatePolicyPicksThePostingDayForTheRate(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.multicurrencyBooks()
	h.controls.set(func(control *tenant.AccountingControl) {
		control.ExchangeRateDatePolicy = tenant.ExchangeRateDatePolicyAccountingDate
	})
	h.rates.set("CAD", "USD", decimal.RequireFromString("0.7312"))
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	posted := aprilTenth + 3*86400
	inv := h.postedInvoice(customerID, invoiceSpec{currency: "CAD"})
	inv.PostedAt = &posted
	h.enqueueInvoice(t, inv)

	h.drain(t)

	assert.Equal(t, []string{"CAD/USD@2026-04-13"}, h.rates.requests())
}

func TestNoRateBlocksTheDocumentAsACurrencyProblem(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.multicurrencyBooks()
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	inv := h.postedInvoice(customerID, invoiceSpec{currency: "CAD"})
	record := h.enqueueInvoice(t, inv)

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorCurrency, blocked.ErrorCategory)
	assert.Contains(t, blocked.ErrorMessage, "no CAD to USD exchange rate for 2026-04-10")
	assert.Contains(t, blocked.Resolution, "OANDA")
	assert.Empty(t, h.writer.callsTo("CreateSalesDocument"))
	assert.False(t, inv.ExchangeRate.Valid, "nothing is stamped when no rate was sent")
}

func TestBooksKeptInAnotherCurrencyThanTrenovaConvertToTheBooksCurrency(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.multicurrencyBooks()
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.ExternalHomeCurrency = "EUR"
	})
	h.rates.set("CAD", "EUR", decimal.RequireFromString("0.6805"))
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	inv := h.postedInvoice(customerID, invoiceSpec{currency: "CAD"})
	stamped := aprilTenth
	inv.ExchangeRate = decimal.NewNullDecimal(decimal.RequireFromString("0.7401"))
	inv.ExchangeRateDate = &stamped
	record := h.enqueueInvoice(t, inv)

	h.drain(t)

	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(record.ID).Status)
	decimalEqual(t, "0.6805", onlySalesDoc(t, h).ExchangeRate)
	assert.Equal(t, []string{"CAD/EUR@2026-04-10"}, h.rates.requests())
	decimalEqual(t, "0.7401", inv.ExchangeRate.Decimal)
}

func TestADocumentInTheBooksCurrencyCarriesNoRate(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.multicurrencyBooks()
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{currency: "usd"}))

	h.drain(t)

	assert.True(t, onlySalesDoc(t, h).ExchangeRate.IsZero())
	assert.Empty(t, h.rates.requests())
}

func TestACustomerPaymentSendsItsRateAndKeepsIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.multicurrencyBooks()
	h.rates.set("CAD", "USD", decimal.RequireFromString("0.7288"))
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	inv := h.postedInvoice(customerID, invoiceSpec{currency: "CAD"})
	h.syncedInvoiceRecord(inv, "qb-inv-1")
	h.mapPaymentAccounts(customerpayment.MethodCheck)
	payment := h.postedPayment(customerID, paymentSpec{
		amountMinor: 150000,
		applications: []*customerpayment.Application{{
			ID:                 pulid.MustNew("cpa_"),
			InvoiceID:          inv.ID,
			AppliedAmountMinor: 150000,
		}},
	})
	payment.CurrencyCode = "CAD"
	record := h.enqueuePayment(t, payment, accountingsync.SyncOperationCreate)

	h.drain(t)

	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(record.ID).Status)
	docs := paymentDocs(t, h)
	require.Len(t, docs, 1)
	decimalEqual(t, "0.7288", docs[0].ExchangeRate)
	assert.Equal(t, []string{"CAD/USD@2026-05-09"}, h.rates.requests(), "dated by the payment date")
	require.True(t, payment.ExchangeRate.Valid)
	decimalEqual(t, "0.7288", payment.ExchangeRate.Decimal)
}

func TestABillSendsThePostedRateAndAPaymentThePaidRate(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.multicurrencyBooks()
	h.rates.set("CAD", "USD", decimal.RequireFromString("0.7312"))
	accts := newPayableAccounts()
	settlement := h.paidCarrierSettlement(accts)
	settlement.CurrencyCode = "CAD"
	posted := aprilTenth
	settlement.ExchangeRate = decimal.NewNullDecimal(decimal.RequireFromString("0.7450"))
	settlement.ExchangeRateDate = &posted
	h.payables.put(h.tenant, settlement)
	h.mapCarrierBill(accts, settlement)
	bill := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBill, accountingsync.SyncOperationCreate,
		settlement)

	h.drain(t)

	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(bill.ID).Status)
	decimalEqual(t, "0.7450", onlyPurchaseDoc(t, h).ExchangeRate)
	assert.Empty(t, h.rates.requests())

	pay := h.enqueueSettlement(t, accountingsync.SyncObjectCarrierBillPay, accountingsync.SyncOperationCreate,
		settlement)

	h.drain(t)

	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(pay.ID).Status)
	decimalEqual(t, "0.7312", onlyBillPayment(t, h).ExchangeRate)
	assert.Equal(t, []string{"CAD/USD@2026-04-18"}, h.rates.requests())
	stored, err := h.payables.GetSettlement(t.Context(), &repositories.GetPayableSettlementRequest{
		TenantInfo: h.tenant,
		Kind:       settlement.Kind,
		ID:         settlement.ID,
	})
	require.NoError(t, err)
	require.True(t, stored.PaidExchangeRate.Valid, "the paid rate is kept on the settlement")
	decimalEqual(t, "0.7312", stored.PaidExchangeRate.Decimal)
	decimalEqual(t, "0.7450", stored.ExchangeRate.Decimal)
}
