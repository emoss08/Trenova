package accountingcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func newTestValidator() *Validator {
	return newTestValidatorWith(validationtest.NewIntegrity(), configuredOANDARepo())
}

func newTestValidatorWithIntegrity(integrity *validationtest.Integrity) *Validator {
	return newTestValidatorWith(integrity, configuredOANDARepo())
}

func newTestValidatorWithIntegrationRepository(
	integrationRepo integrationports.IntegrationRepository,
) *Validator {
	return newTestValidatorWith(validationtest.NewIntegrity(), integrationRepo)
}

func newTestValidatorWith(
	integrity *validationtest.Integrity,
	integrationRepo integrationports.IntegrationRepository,
) *Validator {
	return NewValidator(ValidatorParams{
		Integrity:       integrity,
		IntegrationRepo: integrationRepo,
	})
}

func configuredOANDARepo() *stubAccountingControlIntegrationRepo {
	return &stubAccountingControlIntegrationRepo{
		record: validOANDAIntegration(true, "encrypted-key"),
	}
}

type stubAccountingControlIntegrationRepo struct {
	record *integration.Integration
	err    error
}

func (s *stubAccountingControlIntegrationRepo) ListByTenant(
	_ context.Context,
	_ pagination.TenantInfo,
) ([]*integration.Integration, error) {
	return nil, nil
}

func (s *stubAccountingControlIntegrationRepo) ListEnabledByType(
	_ context.Context,
	_ integration.Type,
) ([]*integration.Integration, error) {
	return nil, nil
}

func (s *stubAccountingControlIntegrationRepo) GetByType(
	_ context.Context,
	_ pagination.TenantInfo,
	_ integration.Type,
) (*integration.Integration, error) {
	return s.record, s.err
}

func (s *stubAccountingControlIntegrationRepo) Upsert(
	_ context.Context,
	_ *integration.Integration,
) (*integration.Integration, error) {
	return nil, nil
}

var _ integrationports.IntegrationRepository = (*stubAccountingControlIntegrationRepo)(nil)

func validOANDAIntegration(enabled bool, apiKey string) *integration.Integration {
	return &integration.Integration{
		Type:    integration.TypeOANDAExchangeRates,
		Enabled: enabled,
		Configuration: map[string]any{
			"apiKey":          apiKey,
			"baseUrl":         "https://exchange-rates-api.oanda.com",
			"defaultRateType": "mid",
		},
	}
}

func validAccountingControl() *accountingcontrol.AccountingControl {
	return &accountingcontrol.AccountingControl{
		ID:                       pulid.MustNew("ac_"),
		BusinessUnitID:           pulid.MustNew("bu_"),
		OrganizationID:           pulid.MustNew("org_"),
		AccountingBasis:          accountingcontrol.AccountingBasisAccrual,
		RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnInvoicePost,
		ExpenseRecognitionPolicy: accountingcontrol.ExpenseRecognitionOnVendorBillPost,
		JournalPostingMode:       accountingcontrol.JournalPostingModeAutomatic,
		AutoPostSourceEvents: []accountingcontrol.JournalSourceEventType{
			accountingcontrol.JournalSourceEventInvoicePosted,
			accountingcontrol.JournalSourceEventCreditMemoPosted,
			accountingcontrol.JournalSourceEventDebitMemoPosted,
			accountingcontrol.JournalSourceEventVendorBillPosted,
		},
		ManualJournalEntryPolicy:         accountingcontrol.ManualJournalEntryPolicyAdjustmentOnly,
		RequireManualJEApproval:          true,
		JournalReversalPolicy:            accountingcontrol.JournalReversalPolicyNextOpenPeriod,
		PeriodCloseMode:                  accountingcontrol.PeriodCloseModeManualOnly,
		RequirePeriodCloseApproval:       true,
		LockedPeriodPostingPolicy:        accountingcontrol.LockedPeriodPostingPolicyBlockSubledgerAllowManualJe,
		ClosedPeriodPostingPolicy:        accountingcontrol.ClosedPeriodPostingPolicyRequireReopen,
		ReconciliationMode:               accountingcontrol.ReconciliationModeWarnOnly,
		ReconciliationToleranceAmount:    decimal.RequireFromString("0.0050"),
		NotifyOnReconciliationException:  true,
		CurrencyMode:                     accountingcontrol.CurrencyModeSingleCurrency,
		FunctionalCurrencyCode:           "USD",
		ExchangeRateDatePolicy:           accountingcontrol.ExchangeRateDatePolicyDocumentDate,
		ExchangeRateOverridePolicy:       accountingcontrol.ExchangeRateOverrideDisallow,
		DefaultRevenueAccountID:          pulid.MustNew("gla_"),
		DefaultCashAccountID:             pulid.MustNew("gla_"),
		DefaultUnappliedCashAccountID:    pulid.MustNew("gla_"),
		DefaultExpenseAccountID:          pulid.MustNew("gla_"),
		DefaultARAccountID:               pulid.MustNew("gla_"),
		DefaultAPAccountID:               pulid.MustNew("gla_"),
		DefaultTaxLiabilityAccountID:     pulid.MustNew("gla_"),
		DefaultWriteOffAccountID:         pulid.MustNew("gla_"),
		DefaultRetainedEarningsAccountID: pulid.MustNew("gla_"),
	}
}

func validMultiCurrencyAccountingControl() *accountingcontrol.AccountingControl {
	entity := validAccountingControl()
	entity.CurrencyMode = accountingcontrol.CurrencyModeMultiCurrency
	entity.ExchangeRateOverridePolicy = accountingcontrol.ExchangeRateOverrideRequireApproval
	entity.RealizedFXGainAccountID = pulid.MustNew("gla_")
	entity.RealizedFXLossAccountID = pulid.MustNew("gla_")
	return entity
}
