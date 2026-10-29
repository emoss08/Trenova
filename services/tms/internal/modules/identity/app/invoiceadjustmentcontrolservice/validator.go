package invoiceadjustmentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator validationframework.TenantedEntityValidator[*invoiceadjustmentcontrol.InvoiceAdjustmentControl]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := validationframework.NewKeelChecker(
		p.DB,
		tenantdb.InvoiceAdjustmentControlsSchema(),
	)

	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*invoiceadjustmentcontrol.InvoiceAdjustmentControl]().
			WithModelName("InvoiceAdjustmentControl").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			WithCustomRule(createEligibilityRule()).
			WithCustomRule(createApprovalThresholdRule()).
			WithCustomRule(createCreditBalanceRule()).
			Build(),
	}
}

func createEligibilityRule() validationframework.TenantedRule[*invoiceadjustmentcontrol.InvoiceAdjustmentControl] {
	return validationframework.NewTenantedRule[*invoiceadjustmentcontrol.InvoiceAdjustmentControl](
		"eligibility",
	).
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.PaidInvoiceAdjustmentPolicy == invoiceadjustmentcontrol.AdjustmentEligibilityAllowWithoutApproval {
				multiErr.Add(
					"paidInvoiceAdjustmentPolicy",
					errortypes.ErrInvalidOperation,
					"Paid invoice adjustments must not be allowed without approval",
				)
			}
			if entity.PartiallyPaidInvoiceAdjustmentPolicy == invoiceadjustmentcontrol.AdjustmentEligibilityAllowWithoutApproval {
				multiErr.Add(
					"partiallyPaidInvoiceAdjustmentPolicy",
					errortypes.ErrInvalidOperation,
					"Partially paid invoice adjustments must not be allowed without approval",
				)
			}
			if entity.DisputedInvoiceAdjustmentPolicy == invoiceadjustmentcontrol.AdjustmentEligibilityAllowWithoutApproval {
				multiErr.Add(
					"disputedInvoiceAdjustmentPolicy",
					errortypes.ErrInvalidOperation,
					"Disputed invoice adjustments must not be allowed without approval",
				)
			}
			if entity.ClosedPeriodAdjustmentPolicy == invoiceadjustmentcontrol.ClosedPeriodAdjustmentPolicyRequireReopen &&
				entity.AdjustmentAccountingDatePolicy != invoiceadjustmentcontrol.AdjustmentAccountingDateUseOriginalIfOpenElseNextOpen {
				multiErr.Add(
					"adjustmentAccountingDatePolicy",
					errortypes.ErrInvalidOperation,
					"Closed period adjustment policy RequireReopen requires accounting date policy UseOriginalIfOpenElseNextOpen",
				)
			}
			if entity.ClosedPeriodAdjustmentPolicy == invoiceadjustmentcontrol.ClosedPeriodAdjustmentPolicyPostInNextOpenPeriodWithApproval &&
				entity.StandardAdjustmentApprovalPolicy == invoiceadjustmentcontrol.ApprovalPolicyNone {
				multiErr.Add(
					"standardAdjustmentApprovalPolicy",
					errortypes.ErrInvalidOperation,
					"Closed period adjustments posted in the next open period must require approval",
				)
			}

			return nil
		})
}

func createApprovalThresholdRule() validationframework.TenantedRule[*invoiceadjustmentcontrol.InvoiceAdjustmentControl] {
	return validationframework.NewTenantedRule[*invoiceadjustmentcontrol.InvoiceAdjustmentControl](
		"approval_thresholds",
	).
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.StandardAdjustmentApprovalPolicy == invoiceadjustmentcontrol.ApprovalPolicyAmountThreshold &&
				!entity.StandardAdjustmentApprovalThreshold.GreaterThan(decimal.Zero) {
				multiErr.Add(
					"standardAdjustmentApprovalThreshold",
					errortypes.ErrInvalidOperation,
					"Standard adjustment approval threshold must be greater than zero when approval policy is AmountThreshold",
				)
			}

			if entity.WriteOffApprovalPolicy == invoiceadjustmentcontrol.WriteOffApprovalPolicyRequireApprovalAboveThreshold &&
				!entity.WriteOffApprovalThreshold.GreaterThan(decimal.Zero) {
				multiErr.Add(
					"writeOffApprovalThreshold",
					errortypes.ErrInvalidOperation,
					"Write-off approval threshold must be greater than zero when write-off approval policy is RequireApprovalAboveThreshold",
				)
			}

			if entity.RerateVarianceTolerancePercent.IsNegative() {
				multiErr.Add(
					"rerateVarianceTolerancePercent",
					errortypes.ErrInvalid,
					"Rerate variance tolerance percent must not be negative",
				)
			}

			return nil
		})
}

func createCreditBalanceRule() validationframework.TenantedRule[*invoiceadjustmentcontrol.InvoiceAdjustmentControl] {
	return validationframework.NewTenantedRule[*invoiceadjustmentcontrol.InvoiceAdjustmentControl](
		"credit_balance",
	).
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.CustomerCreditBalancePolicy == invoiceadjustmentcontrol.CustomerCreditBalancePolicyDisallow &&
				entity.OverCreditPolicy != invoiceadjustmentcontrol.OverCreditPolicyBlock {
				multiErr.Add(
					"overCreditPolicy",
					errortypes.ErrInvalidOperation,
					"Over-credit policy must be Block when customer credit balances are disallowed",
				)
			}
			if entity.OverCreditPolicy == invoiceadjustmentcontrol.OverCreditPolicyAllowWithApproval &&
				entity.CustomerCreditBalancePolicy != invoiceadjustmentcontrol.CustomerCreditBalancePolicyAllowUnappliedCredit {
				multiErr.Add(
					"customerCreditBalancePolicy",
					errortypes.ErrInvalidOperation,
					"Allowing over-credit with approval requires unapplied customer credits to be allowed",
				)
			}

			return nil
		})
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
