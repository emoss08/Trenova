package billingcontrolservice

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
	validator *validationframework.TenantedValidator[*tenant.BillingControl]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*tenant.BillingControl]().
			WithModelName("BillingControl").
			WithUniquenessChecker(
				validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() }),
			).
			WithReferenceChecker(
				validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() }),
			).
			WithCustomRule(createTransferAutomationRule()).
			WithCustomRule(createInvoiceAutomationRule()).
			WithCustomRule(createRequirementEnforcementRule()).
			WithCustomRule(createRateValidationRule()).
			Build(),
	}
}

func createTransferAutomationRule() validationframework.TenantedRule[*tenant.BillingControl] {
	return validationframework.NewTenantedRule[*tenant.BillingControl]("transfer_automation").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *tenant.BillingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.BillingQueueTransferMode == tenant.BillingQueueTransferModeManualOnly {
				if entity.BillingQueueTransferSchedule != "" {
					multiErr.Add("billingQueueTransferSchedule", errortypes.ErrInvalidOperation, "Billing queue transfer schedule must be empty when transfer mode is ManualOnly")
				}
				if entity.BillingQueueTransferBatchSize != 0 {
					multiErr.Add("billingQueueTransferBatchSize", errortypes.ErrInvalidOperation, "Billing queue transfer batch size must be empty when transfer mode is ManualOnly")
				}
				return nil
			}

			if entity.BillingQueueTransferSchedule == "" {
				multiErr.Add("billingQueueTransferSchedule", errortypes.ErrRequired, "Billing queue transfer schedule is required when transfer mode is AutomaticWhenReady")
			}
			if entity.BillingQueueTransferBatchSize < 1 {
				multiErr.Add("billingQueueTransferBatchSize", errortypes.ErrInvalid, "Billing queue transfer batch size must be at least 1 when transfer mode is AutomaticWhenReady")
			}

			return nil
		})
}

func createInvoiceAutomationRule() validationframework.TenantedRule[*tenant.BillingControl] {
	return validationframework.NewTenantedRule[*tenant.BillingControl]("invoice_automation").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *tenant.BillingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.InvoiceDraftCreationMode == tenant.InvoiceDraftCreationModeManualOnly {
				if entity.AutoInvoiceBatchSize != 0 {
					multiErr.Add("autoInvoiceBatchSize", errortypes.ErrInvalidOperation, "Auto invoice batch size must be empty when invoice draft creation mode is ManualOnly")
				}
				if entity.NotifyOnAutoInvoiceCreation {
					multiErr.Add("notifyOnAutoInvoiceCreation", errortypes.ErrInvalidOperation, "Auto invoice creation notifications must be disabled when invoice draft creation mode is ManualOnly")
				}
			} else if entity.AutoInvoiceBatchSize < 1 {
				multiErr.Add("autoInvoiceBatchSize", errortypes.ErrInvalid, "Auto invoice batch size must be at least 1 when invoice draft creation mode is AutomaticWhenTransferred")
			}

			if entity.InvoicePostingMode == tenant.InvoicePostingModeAutomaticWhenNoBlockingExceptions &&
				entity.InvoiceDraftCreationMode != tenant.InvoiceDraftCreationModeAutomaticWhenTransferred {
				multiErr.Add("invoicePostingMode", errortypes.ErrInvalidOperation, "Automatic invoice posting requires invoice draft creation mode AutomaticWhenTransferred")
			}

			return nil
		})
}

func createRequirementEnforcementRule() validationframework.TenantedRule[*tenant.BillingControl] {
	return validationframework.NewTenantedRule[*tenant.BillingControl]("requirement_enforcement").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *tenant.BillingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			switch entity.ShipmentBillingRequirementEnforcement {
			case tenant.EnforcementLevelIgnore, tenant.EnforcementLevelWarn:
				return nil
			case tenant.EnforcementLevelRequireReview:
				if entity.BillingExceptionDisposition == "" {
					multiErr.Add("billingExceptionDisposition", errortypes.ErrRequired, "Billing exception disposition is required when billing requirement enforcement is RequireReview")
				}
			}

			return nil
		})
}

func createRateValidationRule() validationframework.TenantedRule[*tenant.BillingControl] {
	return validationframework.NewTenantedRule[*tenant.BillingControl]("rate_validation").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *tenant.BillingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.RateValidationEnforcement == tenant.EnforcementLevelIgnore {
				if !entity.RateVarianceTolerancePercent.IsZero() {
					multiErr.Add("rateVarianceTolerancePercent", errortypes.ErrInvalidOperation, "Rate variance tolerance percent must be zero when rate validation enforcement is Ignore")
				}
				if entity.RateVarianceAutoResolutionMode != tenant.RateVarianceAutoResolutionModeDisabled {
					multiErr.Add("rateVarianceAutoResolutionMode", errortypes.ErrInvalidOperation, "Rate variance auto resolution mode must be Disabled when rate validation enforcement is Ignore")
				}
				return nil
			}

			if entity.RateVarianceTolerancePercent.IsNegative() {
				multiErr.Add("rateVarianceTolerancePercent", errortypes.ErrInvalid, "Rate variance tolerance percent must not be negative")
			}

			if entity.RateVarianceAutoResolutionMode == tenant.RateVarianceAutoResolutionModeBypassReviewWithinTolerance {
				if entity.RateValidationEnforcement != tenant.EnforcementLevelRequireReview {
					multiErr.Add("rateVarianceAutoResolutionMode", errortypes.ErrInvalidOperation, "BypassReviewWithinTolerance requires rate validation enforcement RequireReview")
				}
				if !entity.RateVarianceTolerancePercent.GreaterThan(decimal.Zero) {
					multiErr.Add("rateVarianceTolerancePercent", errortypes.ErrInvalidOperation, "Rate variance tolerance percent must be greater than zero when bypass review within tolerance is enabled")
				}
			}

			return nil
		})
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *tenant.BillingControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
