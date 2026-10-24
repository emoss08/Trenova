package accountingcontrolservice

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type glAccountCheck struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	AccountID      pulid.ID
}

type capturingGLAccountCheck struct {
	mu       sync.Mutex
	valid    bool
	requests []glAccountCheck
}

func (c *capturingGLAccountCheck) check(
	_ context.Context,
	orgID, buID, accountID pulid.ID,
) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = append(c.requests, glAccountCheck{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		AccountID:      accountID,
	})
	return c.valid, nil
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

func TestNewTestValidator(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	require.NotNil(t, v)
}

func TestValidateUpdate_AllowsRecognizedISOCurrencyCode(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.FunctionalCurrencyCode = "sek"

	multiErr := v.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, multiErr)
}

func TestValidateUpdate_RejectsUnknownCurrencyCode(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.FunctionalCurrencyCode = "ZZZ"

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "functionalCurrencyCode")
}

func TestValidateUpdate_RejectsMultiCurrencyWhenOANDAConfigMissing(t *testing.T) {
	t.Parallel()

	v := NewTestValidatorWithIntegrationRepository(&stubAccountingControlIntegrationRepo{
		err: errortypes.NewNotFoundError("integration not found"),
	})
	entity := validMultiCurrencyAccountingControl()

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "currencyMode")
}

func TestValidateUpdate_RejectsMultiCurrencyWhenOANDADisabled(t *testing.T) {
	t.Parallel()

	v := NewTestValidatorWithIntegrationRepository(&stubAccountingControlIntegrationRepo{
		record: validOANDAIntegration(false, "encrypted-key"),
	})
	entity := validMultiCurrencyAccountingControl()

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "currencyMode")
}

func TestValidateUpdate_RejectsMultiCurrencyWhenOANDAAPIKeyMissing(t *testing.T) {
	t.Parallel()

	v := NewTestValidatorWithIntegrationRepository(&stubAccountingControlIntegrationRepo{
		record: validOANDAIntegration(true, ""),
	})
	entity := validMultiCurrencyAccountingControl()

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "currencyMode")
}

func TestValidateUpdate_AllowsMultiCurrencyWhenOANDAEnabledAndConfigured(t *testing.T) {
	t.Parallel()

	v := NewTestValidatorWithIntegrationRepository(&stubAccountingControlIntegrationRepo{
		record: validOANDAIntegration(true, "encrypted-key"),
	})
	entity := validMultiCurrencyAccountingControl()

	multiErr := v.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, multiErr)
}

func TestValidateUpdate_RequiresDisallowedExchangeRateOverridesForSingleCurrency(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.ExchangeRateOverridePolicy = tenant.ExchangeRateOverrideRequireApproval

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "exchangeRateOverridePolicy")
}

func TestValidateUpdate_RejectsDuplicateAutoPostEvents(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.AutoPostSourceEvents = []tenant.JournalSourceEventType{
		tenant.JournalSourceEventInvoicePosted,
		tenant.JournalSourceEventInvoicePosted,
		tenant.JournalSourceEventVendorBillPosted,
	}

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "autoPostSourceEvents[1]")
}

func TestValidateUpdate_RejectsIncompatibleAccountingBasisCombination(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.AccountingBasis = tenant.AccountingBasisCash
	entity.RevenueRecognitionPolicy = tenant.RevenueRecognitionOnInvoicePost

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "revenueRecognitionPolicy")
}

func TestValidateUpdate_SkipsOptionalReferenceChecksForZeroValueIDs(t *testing.T) {
	t.Parallel()

	checker := &capturingGLAccountCheck{valid: true}
	v := NewTestValidatorWithGLAccountCheck(checker.check)
	entity := validAccountingControl()
	entity.DefaultTaxLiabilityAccountID = pulid.ID("")
	entity.DefaultWriteOffAccountID = pulid.ID("")
	entity.DefaultRetainedEarningsAccountID = pulid.ID("")

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.Nil(t, multiErr)
	assert.Len(t, checker.requests, 6)
}

func TestValidateUpdate_ChecksEveryGLAccountReferenceInTheEntityTenant(t *testing.T) {
	t.Parallel()

	checker := &capturingGLAccountCheck{valid: true}
	v := NewTestValidatorWithGLAccountCheck(checker.check)
	entity := validAccountingControl()
	entity.RealizedFXGainAccountID = pulid.MustNew("gla_")
	entity.RealizedFXLossAccountID = pulid.MustNew("gla_")
	entity.CurrencyMode = tenant.CurrencyModeMultiCurrency
	entity.ExchangeRateOverridePolicy = tenant.ExchangeRateOverrideRequireApproval

	multiErr := v.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, multiErr)
	require.Len(t, checker.requests, 11)
	for _, req := range checker.requests {
		assert.NotEmpty(t, req.AccountID)
		assert.Equal(t, entity.OrganizationID, req.OrganizationID)
		assert.Equal(t, entity.BusinessUnitID, req.BusinessUnitID)
	}
}

func TestValidateUpdate_RejectsZeroReconciliationToleranceWhenEnabled(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.ReconciliationMode = tenant.ReconciliationModeWarnOnly
	entity.ReconciliationToleranceAmount = decimal.Zero

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "reconciliationToleranceAmount")
}

func TestValidateUpdate_RequiresCreditAndDebitMemoEventsForInvoicePostingRecognition(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.AutoPostSourceEvents = []tenant.JournalSourceEventType{
		tenant.JournalSourceEventInvoicePosted,
		tenant.JournalSourceEventVendorBillPosted,
	}

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "autoPostSourceEvents")
}

func TestValidateUpdate_RejectsInvoiceEventsForCashReceiptRecognition(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.RevenueRecognitionPolicy = tenant.RevenueRecognitionOnCashReceipt
	entity.AutoPostSourceEvents = []tenant.JournalSourceEventType{
		tenant.JournalSourceEventCustomerPaymentPosted,
		tenant.JournalSourceEventInvoicePosted,
		tenant.JournalSourceEventVendorBillPosted,
	}

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "autoPostSourceEvents")
}

func TestValidateUpdate_RejectsScheduledCloseWithApproval(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.PeriodCloseMode = tenant.PeriodCloseModeSystemScheduled
	entity.RequirePeriodCloseApproval = true

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "requirePeriodCloseApproval")
}

func TestValidateUpdate_RequiresCashAccountForCustomerPaymentPosting(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.DefaultCashAccountID = pulid.Nil
	entity.AutoPostSourceEvents = append(
		entity.AutoPostSourceEvents,
		tenant.JournalSourceEventCustomerPaymentPosted,
	)

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "defaultCashAccountId")
}

func TestValidateUpdate_RequiresUnappliedCashAccountForCustomerPaymentPosting(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.DefaultUnappliedCashAccountID = pulid.Nil
	entity.AutoPostSourceEvents = append(
		entity.AutoPostSourceEvents,
		tenant.JournalSourceEventCustomerPaymentPosted,
	)

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "defaultUnappliedCashAccountId")
}

func assertErrorField(t *testing.T, multiErr *errortypes.MultiError, field string) {
	t.Helper()

	for _, err := range multiErr.Errors {
		if err.Field == field {
			return
		}
	}

	t.Fatalf("expected validation error for field %q, got %#v", field, multiErr.Errors)
}

func validMultiCurrencyAccountingControl() *tenant.AccountingControl {
	entity := validAccountingControl()
	entity.CurrencyMode = tenant.CurrencyModeMultiCurrency
	entity.ExchangeRateOverridePolicy = tenant.ExchangeRateOverrideRequireApproval
	entity.RealizedFXGainAccountID = pulid.MustNew("gla_")
	entity.RealizedFXLossAccountID = pulid.MustNew("gla_")
	return entity
}

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
