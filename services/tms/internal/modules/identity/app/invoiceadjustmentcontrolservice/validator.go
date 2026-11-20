package invoiceadjustmentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/shopspring/decimal"
)

var maxRerateVarianceTolerancePercent = decimal.RequireFromString("999.999999")

type Validator struct {
	validator tenantedvalidation.EntityValidator[*invoiceadjustmentcontrol.InvoiceAdjustmentControl]
}

func NewValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*invoiceadjustmentcontrol.InvoiceAdjustmentControl]().
			WithModelName("InvoiceAdjustmentControl").
			WithCustomRule(createClosedPeriodRule()).
			WithCustomRule(createApprovalThresholdRule()).
			WithCustomRule(createCreditBalanceRule()).
			Build(),
	}
}

func createClosedPeriodRule() validationframework.Rule[*invoiceadjustmentcontrol.InvoiceAdjustmentControl] {
	return validationframework.NewRule[*invoiceadjustmentcontrol.InvoiceAdjustmentControl](
		"closed_period",
	).
		OnUpdate().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			_ context.Context,
			entity *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
			_ *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
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

func createApprovalThresholdRule() validationframework.Rule[*invoiceadjustmentcontrol.InvoiceAdjustmentControl] {
	return validationframework.NewRule[*invoiceadjustmentcontrol.InvoiceAdjustmentControl](
		"approval_thresholds",
	).
		OnUpdate().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			_ context.Context,
			entity *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
			_ *validationframework.Context,
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
			if entity.RerateVarianceTolerancePercent.GreaterThan(
				maxRerateVarianceTolerancePercent,
			) {
				multiErr.Add(
					"rerateVarianceTolerancePercent",
					errortypes.ErrInvalid,
					"Rerate variance tolerance percent must be at most {0}",
					maxRerateVarianceTolerancePercent.String(),
				)
			}

			return nil
		})
}

func createCreditBalanceRule() validationframework.Rule[*invoiceadjustmentcontrol.InvoiceAdjustmentControl] {
	return validationframework.NewRule[*invoiceadjustmentcontrol.InvoiceAdjustmentControl](
		"credit_balance",
	).
		OnUpdate().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			_ context.Context,
			entity *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
			_ *validationframework.Context,
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
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
