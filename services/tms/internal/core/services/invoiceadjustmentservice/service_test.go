package invoiceadjustmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestApplyCreditBalancePolicy(t *testing.T) {
	svc := &Service{}
	preview := &serviceports.InvoiceAdjustmentPreview{
		WouldCreateUnappliedCredit: true,
		Errors:                     make(map[string][]string),
	}
	entity := &invoice.Invoice{
		SettlementStatus: invoice.SettlementStatusPaid,
	}

	svc.applyCreditBalancePolicy(preview, entity, &tenant.InvoiceAdjustmentControl{
		CustomerCreditBalancePolicy: tenant.CustomerCreditBalancePolicyAllowUnappliedCredit,
		OverCreditPolicy:            tenant.OverCreditPolicyAllowWithApproval,
	})

	assert.True(t, preview.RequiresApproval)
	assert.True(t, preview.RequiresReconciliationException)
	assert.Empty(t, preview.Errors)
}

func TestApplyCreditBalancePolicyDisallow(t *testing.T) {
	svc := &Service{}
	preview := &serviceports.InvoiceAdjustmentPreview{
		WouldCreateUnappliedCredit: true,
		Errors:                     make(map[string][]string),
	}
	entity := &invoice.Invoice{
		SettlementStatus: invoice.SettlementStatusUnpaid,
	}

	svc.applyCreditBalancePolicy(preview, entity, &tenant.InvoiceAdjustmentControl{
		CustomerCreditBalancePolicy: tenant.CustomerCreditBalancePolicyDisallow,
		OverCreditPolicy:            tenant.OverCreditPolicyBlock,
	})

	assert.Contains(t, preview.Errors, "creditTotalAmount")
}

func TestApplyReplacementReviewPolicy(t *testing.T) {
	svc := &Service{}
	preview := &serviceports.InvoiceAdjustmentPreview{
		Kind:                  invoiceadjustment.KindCreditRebill,
		CreditTotalAmount:     decimal.NewFromInt(100),
		RebillTotalAmount:     decimal.NewFromInt(110),
		RerateVariancePercent: decimal.NewFromInt(12),
	}

	svc.applyReplacementReviewPolicy(preview, &tenant.InvoiceAdjustmentControl{
		ReplacementInvoiceReviewPolicy: tenant.ReplacementInvoiceReviewPolicyRequireReviewWhenEconomicTermsChange,
		RerateVarianceTolerancePercent: decimal.NewFromInt(5),
	})

	assert.True(t, preview.RequiresReplacementInvoiceReview)
}

func TestValidateSettlementPolicy(t *testing.T) {
	svc := &Service{}
	preview := &serviceports.InvoiceAdjustmentPreview{
		Errors: make(map[string][]string),
	}

	svc.validateSettlementPolicy(&invoice.Invoice{
		SettlementStatus: invoice.SettlementStatusPartiallyPaid,
		DisputeStatus:    invoice.DisputeStatusDisputed,
	}, &tenant.InvoiceAdjustmentControl{
		PartiallyPaidInvoiceAdjustmentPolicy: tenant.AdjustmentEligibilityAllowWithApproval,
		DisputedInvoiceAdjustmentPolicy:      tenant.AdjustmentEligibilityDisallow,
	}, preview)

	assert.True(t, preview.RequiresApproval)
	assert.Contains(t, preview.Errors, "invoiceId")
	assert.True(t, preview.RequiresReconciliationException)
}

func TestSumInvoiceLines(t *testing.T) {
	lines := []*invoice.InoviceLine{
		{Type: invoice.InvoiceLineTypeFreight, Amount: decimal.NewFromInt(100)},
		{Type: invoice.InvoiceLineTypeAccessorial, Amount: decimal.NewFromInt(20)},
	}

	assert.True(
		t,
		decimal.NewFromInt(100).Equal(sumInvoiceLines(lines, invoice.InvoiceLineTypeFreight)),
	)
	assert.True(t, decimal.NewFromInt(120).Equal(sumInvoiceLines(lines, "")))
}
