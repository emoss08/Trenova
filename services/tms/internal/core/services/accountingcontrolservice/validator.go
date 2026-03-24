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
	"github.com/emoss08/trenova/shared/stringutils"
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
		WithCustomRule(createAccountingMethodCoherenceRule()).
		WithCustomRule(createFlagCoherenceRule()).
		WithCustomRule(createJournalEntryCriteriaDuplicateRule()).
		WithCustomRule(createReconciliationThresholdRule()).
		WithCustomRule(createCurrencyCodeRule())
}

func addGLAccountReferenceChecks(
	builder *validationframework.TenantedValidatorBuilder[*accountingcontrol.AccountingControl],
) {
	builder.
		WithOptionalReferenceCheck(
			"defaultRevenueAccountId",
			"gl_accounts",
			"Default revenue account does not exist in your organization",
			func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultRevenueAccountID
			},
		).
		WithOptionalReferenceCheck(
			"defaultExpenseAccountId",
			"gl_accounts",
			"Default expense account does not exist in your organization",
			func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultExpenseAccountID
			},
		).
		WithOptionalReferenceCheck(
			"defaultArAccountId",
			"gl_accounts",
			"Default AR account does not exist in your organization",
			func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultARAccountID
			},
		).
		WithOptionalReferenceCheck(
			"defaultApAccountId",
			"gl_accounts",
			"Default AP account does not exist in your organization",
			func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultAPAccountID
			},
		).
		WithOptionalReferenceCheck(
			"defaultTaxAccountId",
			"gl_accounts",
			"Default tax account does not exist in your organization",
			func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultTaxAccountID
			},
		).
		WithOptionalReferenceCheck(
			"defaultDeferredRevenueAccountId",
			"gl_accounts",
			"Default deferred revenue account does not exist in your organization",
			func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultDeferredRevenueAccountID
			},
		).
		WithOptionalReferenceCheck(
			"defaultCostOfServiceAccountId",
			"gl_accounts",
			"Default cost of service account does not exist in your organization",
			func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultCostOfServiceAccountID
			},
		).
		WithOptionalReferenceCheck(
			"defaultRetainedEarningsAccountId",
			"gl_accounts",
			"Default retained earnings account does not exist in your organization",
			func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultRetainedEarningsAccountID
			},
		).
		WithOptionalReferenceCheck(
			"currencyGainAccountId",
			"gl_accounts",
			"Currency gain account does not exist in your organization",
			func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.CurrencyGainAccountID
			},
		).
		WithOptionalReferenceCheck(
			"currencyLossAccountId",
			"gl_accounts",
			"Currency loss account does not exist in your organization",
			func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.CurrencyLossAccountID
			},
		)
}

func createAccountingMethodCoherenceRule() validationframework.TenantedRule[*accountingcontrol.AccountingControl] {
	return validationframework.NewTenantedRule[*accountingcontrol.AccountingControl](
		"accounting_method_coherence",
	).
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *accountingcontrol.AccountingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if !entity.AccountingMethod.IsValid() {
				// The domain layer already reports invalid enum membership for this field.
				return nil
			}

			// Revenue recognition coherence
			validRevMethods := entity.AccountingMethod.ValidRevenueRecognitionMethods()
			if !slices.Contains(validRevMethods, entity.RevenueRecognitionMethod) {
				multiErr.Add(
					"revenueRecognitionMethod",
					errortypes.ErrInvalidOperation,
					fmt.Sprintf(
						"%s accounting does not permit revenue recognition method %q; valid options: %s",
						entity.AccountingMethod,
						entity.RevenueRecognitionMethod,
						stringutils.JoinMethods(validRevMethods),
					),
				)
			}

			validExpMethods := entity.AccountingMethod.ValidExpenseRecognitionMethods()
			if !slices.Contains(validExpMethods, entity.ExpenseRecognitionMethod) {
				multiErr.Add(
					"expenseRecognitionMethod",
					errortypes.ErrInvalidOperation,
					fmt.Sprintf(
						"%s accounting does not permit expense recognition method %q; valid options: %s",
						entity.AccountingMethod,
						entity.ExpenseRecognitionMethod,
						stringutils.JoinMethods(validExpMethods),
					),
				)
			}

			return nil
		})
}

func createFlagCoherenceRule() validationframework.TenantedRule[*accountingcontrol.AccountingControl] {
	return validationframework.NewTenantedRule[*accountingcontrol.AccountingControl](
		"flag_coherence",
	).
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *accountingcontrol.AccountingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.DeferRevenueUntilPaid &&
				entity.AccountingMethod == accountingcontrol.AccountingMethodCash {
				multiErr.Add(
					"deferRevenueUntilPaid",
					errortypes.ErrInvalidOperation,
					"Defer revenue until paid is not applicable under cash basis accounting",
				)
			}

			switch entity.AccountingMethod { //nolint:exhaustive // this is fine
			case accountingcontrol.AccountingMethodCash:
				if entity.AccrueExpenses {
					multiErr.Add(
						"accrueExpenses",
						errortypes.ErrInvalidOperation,
						"Accrue expenses is not applicable under cash basis accounting",
					)
				}
			case accountingcontrol.AccountingMethodHybrid:
				if entity.AccrueExpenses {
					multiErr.Add(
						"accrueExpenses",
						errortypes.ErrInvalidOperation,
						"Accrue expenses is not applicable under hybrid accounting (expenses are recognized on a cash basis)",
					)
				}
			}

			return nil
		})
}

func createJournalEntryCriteriaDuplicateRule() validationframework.TenantedRule[*accountingcontrol.AccountingControl] {
	return validationframework.NewTenantedRule[*accountingcontrol.AccountingControl](
		"journal_entry_criteria_duplicates",
	).
		OnUpdate().
		WithStage(validationframework.ValidationStageDataIntegrity).
		WithPriority(validationframework.ValidationPriorityMedium).
		WithValidation(func(
			_ context.Context,
			entity *accountingcontrol.AccountingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			seen := make(
				map[accountingcontrol.JournalEntryCriteriaType]struct{},
				len(entity.JournalEntryCriteria),
			)
			for i, criteria := range entity.JournalEntryCriteria {
				if _, exists := seen[criteria]; exists {
					multiErr.Add(
						fmt.Sprintf("journalEntryCriteria[%d]", i),
						errortypes.ErrInvalid,
						fmt.Sprintf("Duplicate journal entry criteria: %s", criteria),
					)
				}
				seen[criteria] = struct{}{}
			}

			return nil
		})
}

func createReconciliationThresholdRule() validationframework.TenantedRule[*accountingcontrol.AccountingControl] {
	return validationframework.NewTenantedRule[*accountingcontrol.AccountingControl](
		"reconciliation_threshold",
	).
		OnUpdate().
		WithStage(validationframework.ValidationStageDataIntegrity).
		WithPriority(validationframework.ValidationPriorityMedium).
		WithValidation(func(
			_ context.Context,
			entity *accountingcontrol.AccountingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if !entity.EnableReconciliation {
				return nil
			}

			if entity.ReconciliationThreshold.IsNegative() ||
				entity.ReconciliationThreshold.IsZero() {
				multiErr.Add(
					"reconciliationThreshold",
					errortypes.ErrInvalid,
					"Reconciliation threshold must be greater than zero when reconciliation is enabled",
				)
			}

			return nil
		})
}

func createCurrencyCodeRule() validationframework.TenantedRule[*accountingcontrol.AccountingControl] {
	return validationframework.NewTenantedRule[*accountingcontrol.AccountingControl](
		"currency_code_iso4217",
	).
		OnUpdate().
		WithStage(validationframework.ValidationStageDataIntegrity).
		WithPriority(validationframework.ValidationPriorityMedium).
		WithValidation(func(
			_ context.Context,
			entity *accountingcontrol.AccountingControl,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.DefaultCurrencyCode == "" {
				return nil
			}

			if _, err := currency.ParseISO(strings.ToUpper(entity.DefaultCurrencyCode)); err != nil {
				multiErr.Add(
					"defaultCurrencyCode",
					errortypes.ErrInvalid,
					fmt.Sprintf(
						"Unrecognized ISO 4217 currency code: %q",
						entity.DefaultCurrencyCode,
					),
				)
			}

			return nil
		})
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *accountingcontrol.AccountingControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
