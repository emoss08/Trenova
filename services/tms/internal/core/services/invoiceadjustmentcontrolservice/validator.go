package invoiceadjustmentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*tenant.InvoiceAdjustmentControl]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*tenant.InvoiceAdjustmentControl]().
			WithModelName("InvoiceAdjustmentControl").
			WithUniquenessChecker(
				validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() }),
			).
			WithReferenceChecker(
				validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() }),
			).
			WithCustomRule(createEligibilityRule()).
			WithCustomRule(createApprovalThresholdRule()).
			WithCustomRule(createCreditBalanceRule()).
			Build(),
	}
}

func createEligibilityRule() validationframework.TenantedRule[*tenant.InvoiceAdjustmentControl] {
	return validationframework.NewTenantedRule[*tenant.InvoiceAdjustmentControl]("eligibility").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *tenant.InvoiceAdjustmentControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.PaidInvoiceAdjustmentPolicy == tenant.AdjustmentEligibilityAllowWithoutApproval {
				multiErr.Add("paidInvoiceAdjustmentPolicy", errortypes.ErrInvalidOperation, "Paid invoice adjustments must not be allowed without approval")
			}
			if entity.PartiallyPaidInvoiceAdjustmentPolicy == tenant.AdjustmentEligibilityAllowWithoutApproval {
				multiErr.Add("partiallyPaidInvoiceAdjustmentPolicy", errortypes.ErrInvalidOperation, "Partially paid invoice adjustments must not be allowed without approval")
			}
			if entity.DisputedInvoiceAdjustmentPolicy == tenant.AdjustmentEligibilityAllowWithoutApproval {
				multiErr.Add("disputedInvoiceAdjustmentPolicy", errortypes.ErrInvalidOperation, "Disputed invoice adjustments must not be allowed without approval")
			}
			if entity.ClosedPeriodAdjustmentPolicy == tenant.ClosedPeriodAdjustmentPolicyRequireReopen &&
				entity.AdjustmentAccountingDatePolicy != tenant.AdjustmentAccountingDateUseOriginalIfOpenElseNextOpen {
				multiErr.Add("adjustmentAccountingDatePolicy", errortypes.ErrInvalidOperation, "Closed period adjustment policy RequireReopen requires accounting date policy UseOriginalIfOpenElseNextOpen")
			}
			if entity.ClosedPeriodAdjustmentPolicy == tenant.ClosedPeriodAdjustmentPolicyPostInNextOpenPeriodWithApproval &&
				entity.StandardAdjustmentApprovalPolicy == tenant.ApprovalPolicyNone {
				multiErr.Add("standardAdjustmentApprovalPolicy", errortypes.ErrInvalidOperation, "Closed period adjustments posted in the next open period must require approval")
			}

			return nil
		})
}

func createApprovalThresholdRule() validationframework.TenantedRule[*tenant.InvoiceAdjustmentControl] {
	return validationframework.NewTenantedRule[*tenant.InvoiceAdjustmentControl]("approval_thresholds").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *tenant.InvoiceAdjustmentControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.StandardAdjustmentApprovalPolicy == tenant.ApprovalPolicyAmountThreshold &&
				!entity.StandardAdjustmentApprovalThreshold.GreaterThan(decimal.Zero) {
				multiErr.Add("standardAdjustmentApprovalThreshold", errortypes.ErrInvalidOperation, "Standard adjustment approval threshold must be greater than zero when approval policy is AmountThreshold")
			}

			if entity.WriteOffApprovalPolicy == tenant.WriteOffApprovalPolicyRequireApprovalAboveThreshold &&
				!entity.WriteOffApprovalThreshold.GreaterThan(decimal.Zero) {
				multiErr.Add("writeOffApprovalThreshold", errortypes.ErrInvalidOperation, "Write-off approval threshold must be greater than zero when write-off approval policy is RequireApprovalAboveThreshold")
			}

			if entity.RerateVarianceTolerancePercent.IsNegative() {
				multiErr.Add("rerateVarianceTolerancePercent", errortypes.ErrInvalid, "Rerate variance tolerance percent must not be negative")
			}

			return nil
		})
}

func createCreditBalanceRule() validationframework.TenantedRule[*tenant.InvoiceAdjustmentControl] {
	return validationframework.NewTenantedRule[*tenant.InvoiceAdjustmentControl]("credit_balance").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *tenant.InvoiceAdjustmentControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.CustomerCreditBalancePolicy == tenant.CustomerCreditBalancePolicyDisallow &&
				entity.OverCreditPolicy != tenant.OverCreditPolicyBlock {
				multiErr.Add("overCreditPolicy", errortypes.ErrInvalidOperation, "Over-credit policy must be Block when customer credit balances are disallowed")
			}
			if entity.OverCreditPolicy == tenant.OverCreditPolicyAllowWithApproval &&
				entity.CustomerCreditBalancePolicy != tenant.CustomerCreditBalancePolicyAllowUnappliedCredit {
				multiErr.Add("customerCreditBalancePolicy", errortypes.ErrInvalidOperation, "Allowing over-credit with approval requires unapplied customer credits to be allowed")
			}

			return nil
		})
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *tenant.InvoiceAdjustmentControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
