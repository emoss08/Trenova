package accountingcontrolservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	accountingcontrol "github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"golang.org/x/text/currency"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*accountingcontrol.AccountingControl]
}

func NewValidator(p ValidatorParams) *Validator {
	builder := newValidatorBuilder()

	if p.DB != nil {
		builder.
			WithUniquenessChecker(
				validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() }),
			).
			WithReferenceChecker(
				validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() }),
			)
		addGLAccountReferenceChecks(builder)
	}

	return &Validator{
		validator: builder.Build(),
	}
}

func newValidatorBuilder() *validationframework.TenantedValidatorBuilder[*accountingcontrol.AccountingControl] {
	return validationframework.
		NewTenantedValidatorBuilder[*accountingcontrol.AccountingControl]().
		WithModelName("AccountingControl").
		WithCustomRule(createAccountingBasisRule()).
		WithCustomRule(createJournalPostingRule()).
		WithCustomRule(createManualJournalEntryRule()).
		WithCustomRule(createCurrencyRule()).
		WithCustomRule(createReconciliationRule()).
		WithCustomRule(createPeriodCloseRule())
}

func addGLAccountReferenceChecks(
	builder *validationframework.TenantedValidatorBuilder[*accountingcontrol.AccountingControl],
) {
	builder.
		WithOptionalReferenceCheck("defaultRevenueAccountId", "gl_accounts", "Default revenue account does not exist in your organization", func(ac *accountingcontrol.AccountingControl) pulid.ID {
			return ac.DefaultRevenueAccountID
		}).
		WithOptionalReferenceCheck("defaultCashAccountId", "gl_accounts", "Default cash account does not exist in your organization", func(ac *accountingcontrol.AccountingControl) pulid.ID {
			return ac.DefaultCashAccountID
		}).
		WithOptionalReferenceCheck("defaultUnappliedCashAccountId", "gl_accounts", "Default unapplied cash account does not exist in your organization", func(ac *accountingcontrol.AccountingControl) pulid.ID {
			return ac.DefaultUnappliedCashAccountID
		}).
		WithOptionalReferenceCheck("defaultExpenseAccountId", "gl_accounts", "Default expense account does not exist in your organization", func(ac *accountingcontrol.AccountingControl) pulid.ID {
			return ac.DefaultExpenseAccountID
		}).
		WithOptionalReferenceCheck("defaultArAccountId", "gl_accounts", "Default AR account does not exist in your organization", func(ac *accountingcontrol.AccountingControl) pulid.ID {
			return ac.DefaultARAccountID
		}).
		WithOptionalReferenceCheck("defaultApAccountId", "gl_accounts", "Default AP account does not exist in your organization", func(ac *accountingcontrol.AccountingControl) pulid.ID {
			return ac.DefaultAPAccountID
		}).
		WithOptionalReferenceCheck("defaultTaxLiabilityAccountId", "gl_accounts", "Default tax liability account does not exist in your organization", func(ac *accountingcontrol.AccountingControl) pulid.ID {
			return ac.DefaultTaxLiabilityAccountID
		}).
		WithOptionalReferenceCheck("defaultWriteOffAccountId", "gl_accounts", "Default write-off account does not exist in your organization", func(ac *accountingcontrol.AccountingControl) pulid.ID {
			return ac.DefaultWriteOffAccountID
		}).
		WithOptionalReferenceCheck("defaultRetainedEarningsAccountId", "gl_accounts", "Default retained earnings account does not exist in your organization", func(ac *accountingcontrol.AccountingControl) pulid.ID {
			return ac.DefaultRetainedEarningsAccountID
		}).
		WithOptionalReferenceCheck("realizedFxGainAccountId", "gl_accounts", "Realized FX gain account does not exist in your organization", func(ac *accountingcontrol.AccountingControl) pulid.ID {
			return ac.RealizedFXGainAccountID
		}).
		WithOptionalReferenceCheck("realizedFxLossAccountId", "gl_accounts", "Realized FX loss account does not exist in your organization", func(ac *accountingcontrol.AccountingControl) pulid.ID {
			return ac.RealizedFXLossAccountID
		})
}

func createAccountingBasisRule() validationframework.TenantedRule[*accountingcontrol.AccountingControl] {
	return validationframework.NewTenantedRule[*accountingcontrol.AccountingControl]("accounting_basis").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *accountingcontrol.AccountingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if !entity.AccountingBasis.IsValid() {
				return nil
			}

			validRevenuePolicies := entity.AccountingBasis.ValidRevenueRecognitionPolicies()
			if !slices.Contains(validRevenuePolicies, entity.RevenueRecognitionPolicy) {
				multiErr.Add("revenueRecognitionPolicy", errortypes.ErrInvalidOperation, fmt.Sprintf("Accounting basis %s does not allow revenue recognition policy %s", entity.AccountingBasis, entity.RevenueRecognitionPolicy))
			}

			validExpensePolicies := entity.AccountingBasis.ValidExpenseRecognitionPolicies()
			if !slices.Contains(validExpensePolicies, entity.ExpenseRecognitionPolicy) {
				multiErr.Add("expenseRecognitionPolicy", errortypes.ErrInvalidOperation, fmt.Sprintf("Accounting basis %s does not allow expense recognition policy %s", entity.AccountingBasis, entity.ExpenseRecognitionPolicy))
			}

			return nil
		})
}

func createJournalPostingRule() validationframework.TenantedRule[*accountingcontrol.AccountingControl] {
	return validationframework.NewTenantedRule[*accountingcontrol.AccountingControl]("journal_posting").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *accountingcontrol.AccountingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.JournalPostingMode == accountingcontrol.JournalPostingModeManual {
				if len(entity.AutoPostSourceEvents) > 0 {
					multiErr.Add("autoPostSourceEvents", errortypes.ErrInvalidOperation, "Auto-post source events must be empty when journal posting mode is Manual")
				}
				return nil
			}

			if len(entity.AutoPostSourceEvents) == 0 {
				multiErr.Add("autoPostSourceEvents", errortypes.ErrRequired, "At least one auto-post source event is required when journal posting mode is Automatic")
			}

			seen := make(map[accountingcontrol.JournalSourceEventType]struct{}, len(entity.AutoPostSourceEvents))
			for i, event := range entity.AutoPostSourceEvents {
				if !event.IsValid() {
					multiErr.Add(fmt.Sprintf("autoPostSourceEvents[%d]", i), errortypes.ErrInvalid, "Invalid journal source event")
					continue
				}
				if _, ok := seen[event]; ok {
					multiErr.Add(fmt.Sprintf("autoPostSourceEvents[%d]", i), errortypes.ErrDuplicate, "Duplicate journal source event")
				}
				seen[event] = struct{}{}
			}

			requiredEvents := make([]accountingcontrol.JournalSourceEventType, 0, 2)
			switch entity.RevenueRecognitionPolicy {
			case accountingcontrol.RevenueRecognitionOnInvoicePost:
				requiredEvents = append(requiredEvents,
					accountingcontrol.JournalSourceEventInvoicePosted,
					accountingcontrol.JournalSourceEventCreditMemoPosted,
					accountingcontrol.JournalSourceEventDebitMemoPosted,
				)
			case accountingcontrol.RevenueRecognitionOnCashReceipt:
				requiredEvents = append(requiredEvents, accountingcontrol.JournalSourceEventCustomerPaymentPosted)
			}
			switch entity.ExpenseRecognitionPolicy {
			case accountingcontrol.ExpenseRecognitionOnVendorBillPost:
				requiredEvents = append(requiredEvents, accountingcontrol.JournalSourceEventVendorBillPosted)
			case accountingcontrol.ExpenseRecognitionOnCashDisbursement:
				requiredEvents = append(requiredEvents, accountingcontrol.JournalSourceEventVendorPaymentPosted)
			}

			for _, requiredEvent := range requiredEvents {
				if !slices.Contains(entity.AutoPostSourceEvents, requiredEvent) {
					multiErr.Add("autoPostSourceEvents", errortypes.ErrInvalidOperation, fmt.Sprintf("Auto-post source events must include %s", requiredEvent))
				}
			}

			if entity.RevenueRecognitionPolicy == accountingcontrol.RevenueRecognitionOnCashReceipt {
				for _, blockedEvent := range []accountingcontrol.JournalSourceEventType{
					accountingcontrol.JournalSourceEventInvoicePosted,
					accountingcontrol.JournalSourceEventCreditMemoPosted,
					accountingcontrol.JournalSourceEventDebitMemoPosted,
				} {
					if slices.Contains(entity.AutoPostSourceEvents, blockedEvent) {
						multiErr.Add("autoPostSourceEvents", errortypes.ErrInvalidOperation, fmt.Sprintf("Auto-post source events must not include %s when revenue recognition is OnCashReceipt", blockedEvent))
					}
				}
			}

			requireGLAccount(multiErr, entity.DefaultRevenueAccountID, "defaultRevenueAccountId", "Default revenue account is required when journal posting mode is Automatic")
			if slices.Contains(entity.AutoPostSourceEvents, accountingcontrol.JournalSourceEventCustomerPaymentPosted) || entity.RevenueRecognitionPolicy == accountingcontrol.RevenueRecognitionOnCashReceipt {
				requireGLAccount(multiErr, entity.DefaultCashAccountID, "defaultCashAccountId", "Default cash account is required when customer payment posting is enabled")
				requireGLAccount(multiErr, entity.DefaultUnappliedCashAccountID, "defaultUnappliedCashAccountId", "Default unapplied cash account is required when customer payment posting is enabled")
			}
			requireGLAccount(multiErr, entity.DefaultExpenseAccountID, "defaultExpenseAccountId", "Default expense account is required when journal posting mode is Automatic")
			requireGLAccount(multiErr, entity.DefaultARAccountID, "defaultArAccountId", "Default AR account is required when journal posting mode is Automatic")
			requireGLAccount(multiErr, entity.DefaultAPAccountID, "defaultApAccountId", "Default AP account is required when journal posting mode is Automatic")

			return nil
		})
}

func createManualJournalEntryRule() validationframework.TenantedRule[*accountingcontrol.AccountingControl] {
	return validationframework.NewTenantedRule[*accountingcontrol.AccountingControl]("manual_journal_entry").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *accountingcontrol.AccountingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.ManualJournalEntryPolicy == accountingcontrol.ManualJournalEntryPolicyDisallow &&
				entity.RequireManualJEApproval {
				multiErr.Add("requireManualJeApproval", errortypes.ErrInvalidOperation, "Manual journal entry approval must be disabled when manual journal entries are disallowed")
			}

			return nil
		})
}

func createCurrencyRule() validationframework.TenantedRule[*accountingcontrol.AccountingControl] {
	return validationframework.NewTenantedRule[*accountingcontrol.AccountingControl]("currency").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *accountingcontrol.AccountingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if _, err := currency.ParseISO(strings.ToUpper(entity.FunctionalCurrencyCode)); err != nil {
				multiErr.Add("functionalCurrencyCode", errortypes.ErrInvalid, fmt.Sprintf("Unrecognized ISO 4217 currency code: %q", entity.FunctionalCurrencyCode))
			}

			if entity.CurrencyMode == accountingcontrol.CurrencyModeSingleCurrency {
				if !entity.RealizedFXGainAccountID.IsNil() {
					multiErr.Add("realizedFxGainAccountId", errortypes.ErrInvalidOperation, "Realized FX gain account must be empty when currency mode is SingleCurrency")
				}
				if !entity.RealizedFXLossAccountID.IsNil() {
					multiErr.Add("realizedFxLossAccountId", errortypes.ErrInvalidOperation, "Realized FX loss account must be empty when currency mode is SingleCurrency")
				}
				if entity.ExchangeRateOverridePolicy != accountingcontrol.ExchangeRateOverrideDisallow {
					multiErr.Add("exchangeRateOverridePolicy", errortypes.ErrInvalidOperation, "Exchange rate override policy must be Disallow when currency mode is SingleCurrency")
				}
				return nil
			}

			requireGLAccount(multiErr, entity.RealizedFXGainAccountID, "realizedFxGainAccountId", "Realized FX gain account is required when currency mode is MultiCurrency")
			requireGLAccount(multiErr, entity.RealizedFXLossAccountID, "realizedFxLossAccountId", "Realized FX loss account is required when currency mode is MultiCurrency")

			return nil
		})
}

func createReconciliationRule() validationframework.TenantedRule[*accountingcontrol.AccountingControl] {
	return validationframework.NewTenantedRule[*accountingcontrol.AccountingControl]("reconciliation").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *accountingcontrol.AccountingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			switch entity.ReconciliationMode {
			case accountingcontrol.ReconciliationModeDisabled:
				if !entity.ReconciliationToleranceAmount.IsZero() {
					multiErr.Add("reconciliationToleranceAmount", errortypes.ErrInvalidOperation, "Reconciliation tolerance amount must be zero when reconciliation mode is Disabled")
				}
			case accountingcontrol.ReconciliationModeWarnOnly, accountingcontrol.ReconciliationModeBlockPosting:
				if !entity.ReconciliationToleranceAmount.GreaterThan(decimal.Zero) {
					multiErr.Add("reconciliationToleranceAmount", errortypes.ErrInvalidOperation, "Reconciliation tolerance amount must be greater than zero when reconciliation mode is enabled")
				}
			}

			if entity.RequireReconciliationToClose && entity.ReconciliationMode == accountingcontrol.ReconciliationModeDisabled {
				multiErr.Add("requireReconciliationToClose", errortypes.ErrInvalidOperation, "Require reconciliation to close cannot be enabled when reconciliation mode is Disabled")
			}

			return nil
		})
}

func createPeriodCloseRule() validationframework.TenantedRule[*accountingcontrol.AccountingControl] {
	return validationframework.NewTenantedRule[*accountingcontrol.AccountingControl]("period_close").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *accountingcontrol.AccountingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.PeriodCloseMode == accountingcontrol.PeriodCloseModeSystemScheduled &&
				entity.RequirePeriodCloseApproval {
				multiErr.Add("requirePeriodCloseApproval", errortypes.ErrInvalidOperation, "Period close approval must be disabled when period close mode is SystemScheduled")
			}

			return nil
		})
}

func requireGLAccount(multiErr *errortypes.MultiError, id pulid.ID, field, message string) {
	if id.IsNil() {
		multiErr.Add(field, errortypes.ErrRequired, message)
	}
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *accountingcontrol.AccountingControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
