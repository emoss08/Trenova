package billingcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*billingcontrol.BillingControl]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(p.DB, tenantdb.BillingControlsSchema())

	return &Validator{
		validator: tenantedvalidation.NewBuilder[*billingcontrol.BillingControl]().
			WithModelName("BillingControl").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			WithCustomRule(createTransferAutomationRule()).
			WithCustomRule(createInvoiceAutomationRule()).
			WithCustomRule(createRequirementEnforcementRule()).
			WithCustomRule(createRateValidationRule()).
			Build(),
	}
}

func createTransferAutomationRule() validationframework.TenantedRule[*billingcontrol.BillingControl] {
	return validationframework.NewTenantedRule[*billingcontrol.BillingControl](
		"transfer_automation",
	).
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *billingcontrol.BillingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.BillingQueueTransferMode != billingcontrol.BillingQueueTransferModeAutomaticWhenReady {
				return nil
			}

			if entity.BillingQueueTransferSchedule == "" {
				multiErr.Add(
					"billingQueueTransferSchedule",
					errortypes.ErrRequired,
					"Billing queue transfer schedule is required when transfer mode is AutomaticWhenReady",
				)
			}
			if entity.BillingQueueTransferBatchSize < 1 {
				multiErr.Add(
					"billingQueueTransferBatchSize",
					errortypes.ErrInvalid,
					"Billing queue transfer batch size must be at least 1 when transfer mode is AutomaticWhenReady",
				)
			}

			return nil
		})
}

func createInvoiceAutomationRule() validationframework.TenantedRule[*billingcontrol.BillingControl] {
	return validationframework.NewTenantedRule[*billingcontrol.BillingControl](
		"invoice_automation",
	).
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *billingcontrol.BillingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.InvoiceDraftCreationMode == billingcontrol.InvoiceDraftCreationModeManualOnly {
				if entity.AutoInvoiceBatchSize != 0 {
					multiErr.Add(
						"autoInvoiceBatchSize",
						errortypes.ErrInvalidOperation,
						"Auto invoice batch size must be empty when invoice draft creation mode is ManualOnly",
					)
				}
				if entity.NotifyOnAutoInvoiceCreation {
					multiErr.Add(
						"notifyOnAutoInvoiceCreation",
						errortypes.ErrInvalidOperation,
						"Auto invoice creation notifications must be disabled when invoice draft creation mode is ManualOnly",
					)
				}
			} else if entity.AutoInvoiceBatchSize < 1 {
				multiErr.Add(
					"autoInvoiceBatchSize",
					errortypes.ErrInvalid,
					"Auto invoice batch size must be at least 1 when invoice draft creation mode is AutomaticWhenTransferred",
				)
			}

			if entity.InvoicePostingMode == billingcontrol.InvoicePostingModeAutomaticWhenNoBlockingExceptions &&
				entity.InvoiceDraftCreationMode != billingcontrol.InvoiceDraftCreationModeAutomaticWhenTransferred {
				multiErr.Add(
					"invoicePostingMode",
					errortypes.ErrInvalidOperation,
					"Automatic invoice posting requires invoice draft creation mode AutomaticWhenTransferred",
				)
			}

			return nil
		})
}

func createRequirementEnforcementRule() validationframework.TenantedRule[*billingcontrol.BillingControl] {
	return validationframework.NewTenantedRule[*billingcontrol.BillingControl](
		"requirement_enforcement",
	).
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *billingcontrol.BillingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			//nolint:exhaustive // only actionable enum states require explicit handling here
			switch entity.ShipmentBillingRequirementEnforcement {
			case tenant.EnforcementLevelIgnore, tenant.EnforcementLevelWarn:
				return nil
			case tenant.EnforcementLevelRequireReview:
				if entity.BillingExceptionDisposition == "" {
					multiErr.Add(
						"billingExceptionDisposition",
						errortypes.ErrRequired,
						"Billing exception disposition is required when billing requirement enforcement is RequireReview",
					)
				}
			}

			return nil
		})
}

func createRateValidationRule() validationframework.TenantedRule[*billingcontrol.BillingControl] {
	return validationframework.NewTenantedRule[*billingcontrol.BillingControl]("rate_validation").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *billingcontrol.BillingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.RateValidationEnforcement == tenant.EnforcementLevelIgnore {
				if !entity.RateVarianceTolerancePercent.IsZero() {
					multiErr.Add(
						"rateVarianceTolerancePercent",
						errortypes.ErrInvalidOperation,
						"Rate variance tolerance percent must be zero when rate validation enforcement is Ignore",
					)
				}
				if entity.RateVarianceAutoResolutionMode != billingcontrol.RateVarianceAutoResolutionModeDisabled {
					multiErr.Add(
						"rateVarianceAutoResolutionMode",
						errortypes.ErrInvalidOperation,
						"Rate variance auto resolution mode must be Disabled when rate validation enforcement is Ignore",
					)
				}
				return nil
			}

			if entity.RateVarianceTolerancePercent.IsNegative() {
				multiErr.Add(
					"rateVarianceTolerancePercent",
					errortypes.ErrInvalid,
					"Rate variance tolerance percent must not be negative",
				)
			}

			if entity.RateVarianceAutoResolutionMode == billingcontrol.RateVarianceAutoResolutionModeBypassReviewWithinTolerance {
				if entity.RateValidationEnforcement != tenant.EnforcementLevelRequireReview {
					multiErr.Add(
						"rateVarianceAutoResolutionMode",
						errortypes.ErrInvalidOperation,
						"BypassReviewWithinTolerance requires rate validation enforcement RequireReview",
					)
				}
				if !entity.RateVarianceTolerancePercent.GreaterThan(decimal.Zero) {
					multiErr.Add(
						"rateVarianceTolerancePercent",
						errortypes.ErrInvalidOperation,
						"Rate variance tolerance percent must be greater than zero when bypass review within tolerance is enabled",
					)
				}
			}

			return nil
		})
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *billingcontrol.BillingControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
