package invoiceadjustmentengine

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestApplyCreditBalancePolicy(t *testing.T) {
	svc := &Engine{}
	preview := &invoiceadjustmentports.InvoiceAdjustmentPreview{
		WouldCreateUnappliedCredit: true,
		Errors:                     make(map[string][]string),
	}
	entity := &invoice.Invoice{
		SettlementStatus: invoice.SettlementStatusPaid,
	}

	svc.applyCreditBalancePolicy(
		preview,
		entity,
		&invoiceadjustmentcontrol.InvoiceAdjustmentControl{
			CustomerCreditBalancePolicy: invoiceadjustmentcontrol.CustomerCreditBalancePolicyAllowUnappliedCredit,
			OverCreditPolicy:            invoiceadjustmentcontrol.OverCreditPolicyAllowWithApproval,
		},
	)

	assert.True(t, preview.RequiresApproval)
	assert.True(t, preview.RequiresReconciliationException)
	assert.Empty(t, preview.Errors)
}

func TestApplyCreditBalancePolicyDisallow(t *testing.T) {
	svc := &Engine{}
	preview := &invoiceadjustmentports.InvoiceAdjustmentPreview{
		WouldCreateUnappliedCredit: true,
		Errors:                     make(map[string][]string),
	}
	entity := &invoice.Invoice{
		SettlementStatus: invoice.SettlementStatusUnpaid,
	}

	svc.applyCreditBalancePolicy(
		preview,
		entity,
		&invoiceadjustmentcontrol.InvoiceAdjustmentControl{
			CustomerCreditBalancePolicy: invoiceadjustmentcontrol.CustomerCreditBalancePolicyDisallow,
			OverCreditPolicy:            invoiceadjustmentcontrol.OverCreditPolicyBlock,
		},
	)

	assert.Contains(t, preview.Errors, "creditTotalAmount")
}

func TestApplyReplacementReviewPolicy(t *testing.T) {
	svc := &Engine{}
	preview := &invoiceadjustmentports.InvoiceAdjustmentPreview{
		Kind:                  invoiceadjustment.KindCreditRebill,
		CreditTotalAmount:     decimal.NewFromInt(100),
		RebillTotalAmount:     decimal.NewFromInt(110),
		RerateVariancePercent: decimal.NewFromInt(12),
	}

	svc.applyReplacementReviewPolicy(preview, &invoiceadjustmentcontrol.InvoiceAdjustmentControl{
		ReplacementInvoiceReviewPolicy: invoiceadjustmentcontrol.ReplacementInvoiceReviewPolicyRequireReviewWhenEconomicTermsChange,
		RerateVarianceTolerancePercent: decimal.NewFromInt(5),
	})

	assert.True(t, preview.RequiresReplacementInvoiceReview)
}

func TestValidateSettlementPolicy(t *testing.T) {
	svc := &Engine{}
	preview := &invoiceadjustmentports.InvoiceAdjustmentPreview{
		Errors: make(map[string][]string),
	}

	svc.validateSettlementPolicy(&invoice.Invoice{
		SettlementStatus: invoice.SettlementStatusPartiallyPaid,
		DisputeStatus:    invoice.DisputeStatusDisputed,
	}, &invoiceadjustmentcontrol.InvoiceAdjustmentControl{
		PartiallyPaidInvoiceAdjustmentPolicy: invoiceadjustmentcontrol.AdjustmentEligibilityAllowWithApproval,
		DisputedInvoiceAdjustmentPolicy:      invoiceadjustmentcontrol.AdjustmentEligibilityDisallow,
	}, preview)

	assert.True(t, preview.RequiresApproval)
	assert.Contains(t, preview.Errors, "invoiceId")
	assert.True(t, preview.RequiresReconciliationException)
}

func TestSumInvoiceLines(t *testing.T) {
	lines := []*invoice.InvoiceLine{
		{Type: invoice.InvoiceLineTypeFreight, Amount: decimal.NewFromInt(100)},
		{Type: invoice.InvoiceLineTypeAccessorial, Amount: decimal.NewFromInt(20)},
	}

	assert.True(
		t,
		decimal.NewFromInt(100).Equal(sumInvoiceLines(lines, invoice.InvoiceLineTypeFreight)),
	)
	assert.True(t, decimal.NewFromInt(120).Equal(sumInvoiceLines(lines, "")))
}
