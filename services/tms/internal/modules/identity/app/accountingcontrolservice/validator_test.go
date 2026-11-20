package accountingcontrolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUpdate_AllowsAValidAccountingControl(t *testing.T) {
	t.Parallel()

	err := newTestValidator().ValidateUpdate(t.Context(), validAccountingControl())

	validationtest.Outcome(t, err).Valid()
}

func TestValidateUpdate_AllowsRecognizedISOCurrencyCode(t *testing.T) {
	t.Parallel()

	entity := validAccountingControl()
	entity.FunctionalCurrencyCode = "sek"

	err := newTestValidator().ValidateUpdate(t.Context(), entity)

	validationtest.Outcome(t, err).Valid()
}

func TestValidateUpdate_RejectsUnknownCurrencyCode(t *testing.T) {
	t.Parallel()

	entity := validAccountingControl()
	entity.FunctionalCurrencyCode = "ZZZ"

	err := newTestValidator().ValidateUpdate(t.Context(), entity)

	validationtest.Refused(t, err).Refuses("functionalCurrencyCode", errortypes.ErrInvalid)
}

func TestValidateUpdate_RejectsMultiCurrencyWhenOANDAConfigMissing(t *testing.T) {
	t.Parallel()

	v := newTestValidatorWithIntegrationRepository(&stubAccountingControlIntegrationRepo{
		err: errortypes.NewNotFoundError("integration not found"),
	})

	err := v.ValidateUpdate(t.Context(), validMultiCurrencyAccountingControl())

	validationtest.Refused(t, err).Exactly(
		validationtest.At("currencyMode", errortypes.ErrInvalidOperation),
	)
}

func TestValidateUpdate_RejectsMultiCurrencyWhenOANDADisabled(t *testing.T) {
	t.Parallel()

	v := newTestValidatorWithIntegrationRepository(&stubAccountingControlIntegrationRepo{
		record: validOANDAIntegration(false, "encrypted-key"),
	})

	err := v.ValidateUpdate(t.Context(), validMultiCurrencyAccountingControl())

	validationtest.Refused(t, err).Exactly(
		validationtest.At("currencyMode", errortypes.ErrInvalidOperation),
	)
}

func TestValidateUpdate_RejectsMultiCurrencyWhenOANDAAPIKeyMissing(t *testing.T) {
	t.Parallel()

	v := newTestValidatorWithIntegrationRepository(&stubAccountingControlIntegrationRepo{
		record: validOANDAIntegration(true, ""),
	})

	err := v.ValidateUpdate(t.Context(), validMultiCurrencyAccountingControl())

	validationtest.Refused(t, err).Exactly(
		validationtest.At("currencyMode", errortypes.ErrInvalidOperation),
	)
}

func TestValidateUpdate_AllowsMultiCurrencyWhenOANDAEnabledAndConfigured(t *testing.T) {
	t.Parallel()

	v := newTestValidatorWithIntegrationRepository(configuredOANDARepo())

	err := v.ValidateUpdate(t.Context(), validMultiCurrencyAccountingControl())

	validationtest.Outcome(t, err).Valid()
}

func TestValidateUpdate_FailsWhenTheIntegrationLookupFails(t *testing.T) {
	t.Parallel()

	v := newTestValidatorWithIntegrationRepository(&stubAccountingControlIntegrationRepo{
		err: assert.AnError,
	})

	err := v.ValidateUpdate(t.Context(), validMultiCurrencyAccountingControl())

	assert.ErrorIs(t, validationtest.Failed(t, err), assert.AnError)
}

func TestValidateUpdate_RequiresDisallowedExchangeRateOverridesForSingleCurrency(t *testing.T) {
	t.Parallel()

	entity := validAccountingControl()
	entity.ExchangeRateOverridePolicy = accountingcontrol.ExchangeRateOverrideRequireApproval

	err := newTestValidator().ValidateUpdate(t.Context(), entity)

	validationtest.Refused(t, err).Exactly(
		validationtest.At("exchangeRateOverridePolicy", errortypes.ErrInvalidOperation),
	)
}

func TestValidateUpdate_RejectsDuplicateAutoPostEvents(t *testing.T) {
	t.Parallel()

	entity := validAccountingControl()
	entity.AutoPostSourceEvents = []accountingcontrol.JournalSourceEventType{
		accountingcontrol.JournalSourceEventInvoicePosted,
		accountingcontrol.JournalSourceEventInvoicePosted,
		accountingcontrol.JournalSourceEventVendorBillPosted,
	}

	err := newTestValidator().ValidateUpdate(t.Context(), entity)

	validationtest.Refused(t, err).Refuses("autoPostSourceEvents[1]", errortypes.ErrDuplicate)
}

func TestValidateUpdate_RejectsIncompatibleAccountingBasisCombination(t *testing.T) {
	t.Parallel()

	entity := validAccountingControl()
	entity.AccountingBasis = accountingcontrol.AccountingBasisCash
	entity.RevenueRecognitionPolicy = accountingcontrol.RevenueRecognitionOnInvoicePost

	err := newTestValidator().ValidateUpdate(t.Context(), entity)

	validationtest.Refused(t, err).
		Refuses("revenueRecognitionPolicy", errortypes.ErrInvalidOperation)
}

func TestValidateUpdate_SkipsOptionalReferenceChecksForZeroValueIDs(t *testing.T) {
	t.Parallel()

	integrity := validationtest.NewIntegrity()
	entity := validAccountingControl()
	entity.DefaultTaxLiabilityAccountID = pulid.Nil
	entity.DefaultWriteOffAccountID = pulid.Nil
	entity.DefaultRetainedEarningsAccountID = pulid.Nil

	err := newTestValidatorWithIntegrity(integrity).ValidateUpdate(t.Context(), entity)

	validationtest.Outcome(t, err).Valid()
	request := onlyIntegrityRequest(t, integrity)
	assert.Len(t, request.References, 6)
}

func TestValidateUpdate_ChecksEveryGLAccountReferenceInTheEntityTenant(t *testing.T) {
	t.Parallel()

	integrity := validationtest.NewIntegrity()
	entity := validMultiCurrencyAccountingControl()

	err := newTestValidatorWithIntegrity(integrity).ValidateUpdate(t.Context(), entity)

	validationtest.Outcome(t, err).Valid()
	request := onlyIntegrityRequest(t, integrity)
	assert.Equal(t, entity.OrganizationID, request.OrganizationID)
	assert.Equal(t, entity.BusinessUnitID, request.BusinessUnitID)
	require.Len(t, request.References, 11)
	for _, reference := range request.References {
		assert.Equal(t, glAccountsTable, reference.Table)
		assert.NotEmpty(t, reference.ID)
	}
}

func TestValidateUpdate_ChecksDriverPayAccountReferencesInTheEntityTenant(t *testing.T) {
	t.Parallel()

	integrity := validationtest.NewIntegrity()
	entity := validAccountingControl()
	for _, account := range driverPayAccounts() {
		account.set(entity, pulid.MustNew("gla_"))
	}

	err := newTestValidatorWithIntegrity(integrity).ValidateUpdate(t.Context(), entity)

	validationtest.Outcome(t, err).Valid()
	request := onlyIntegrityRequest(t, integrity)
	assert.Equal(t, entity.OrganizationID, request.OrganizationID)
	assert.Equal(t, entity.BusinessUnitID, request.BusinessUnitID)
	require.Len(t, request.References, 17)
	checked := make(map[pulid.ID]struct{}, len(request.References))
	for _, reference := range request.References {
		assert.Equal(t, glAccountsTable, reference.Table)
		checked[reference.ID] = struct{}{}
	}
	for _, account := range driverPayAccounts() {
		assert.Contains(t, checked, account.get(entity), account.field)
	}
}

func TestValidateUpdate_RejectsDriverPayAccountOutsideTheTenant(t *testing.T) {
	t.Parallel()

	for _, account := range driverPayAccounts() {
		t.Run(account.field, func(t *testing.T) {
			t.Parallel()

			foreign := pulid.MustNew("gla_")
			integrity := validationtest.NewIntegrity().Missing(glAccountsTable, foreign)
			entity := validAccountingControl()
			account.set(entity, foreign)

			err := newTestValidatorWithIntegrity(integrity).ValidateUpdate(t.Context(), entity)

			validationtest.Refused(t, err).Exactly(
				validationtest.At(account.field, errortypes.ErrInvalidReference),
			)
		})
	}
}

func TestValidateUpdate_RejectsStandardAccountOutsideTheTenant(t *testing.T) {
	t.Parallel()

	foreign := pulid.MustNew("gla_")
	integrity := validationtest.NewIntegrity().Missing(glAccountsTable, foreign)
	entity := validAccountingControl()
	entity.DefaultCashAccountID = foreign

	err := newTestValidatorWithIntegrity(integrity).ValidateUpdate(t.Context(), entity)

	validationtest.Refused(t, err).Exactly(
		validationtest.At("defaultCashAccountId", errortypes.ErrInvalidReference),
	)
}

func TestValidateUpdate_FailsWhenTheReferenceCheckFails(t *testing.T) {
	t.Parallel()

	integrity := validationtest.NewIntegrity().FailWith(assert.AnError)

	err := newTestValidatorWithIntegrity(integrity).
		ValidateUpdate(t.Context(), validAccountingControl())

	assert.ErrorIs(t, validationtest.Failed(t, err), assert.AnError)
}

func TestValidateUpdate_RejectsZeroReconciliationToleranceWhenEnabled(t *testing.T) {
	t.Parallel()

	entity := validAccountingControl()
	entity.ReconciliationMode = accountingcontrol.ReconciliationModeWarnOnly
	entity.ReconciliationToleranceAmount = decimal.Zero

	err := newTestValidator().ValidateUpdate(t.Context(), entity)

	validationtest.Refused(t, err).Exactly(
		validationtest.At("reconciliationToleranceAmount", errortypes.ErrInvalidOperation),
	)
}

func TestValidateUpdate_RequiresCreditAndDebitMemoEventsForInvoicePostingRecognition(t *testing.T) {
	t.Parallel()

	entity := validAccountingControl()
	entity.AutoPostSourceEvents = []accountingcontrol.JournalSourceEventType{
		accountingcontrol.JournalSourceEventInvoicePosted,
		accountingcontrol.JournalSourceEventVendorBillPosted,
	}

	err := newTestValidator().ValidateUpdate(t.Context(), entity)

	validationtest.Refused(t, err).Exactly(
		validationtest.At("autoPostSourceEvents", errortypes.ErrInvalidOperation),
		validationtest.At("autoPostSourceEvents", errortypes.ErrInvalidOperation),
	)
}

func TestValidateUpdate_RejectsInvoiceEventsForCashReceiptRecognition(t *testing.T) {
	t.Parallel()

	entity := validAccountingControl()
	entity.RevenueRecognitionPolicy = accountingcontrol.RevenueRecognitionOnCashReceipt
	entity.AutoPostSourceEvents = []accountingcontrol.JournalSourceEventType{
		accountingcontrol.JournalSourceEventCustomerPaymentPosted,
		accountingcontrol.JournalSourceEventInvoicePosted,
		accountingcontrol.JournalSourceEventVendorBillPosted,
	}

	err := newTestValidator().ValidateUpdate(t.Context(), entity)

	validationtest.Refused(t, err).Refuses("autoPostSourceEvents", errortypes.ErrInvalidOperation)
}

func TestValidateUpdate_RejectsScheduledCloseWithApproval(t *testing.T) {
	t.Parallel()

	entity := validAccountingControl()
	entity.PeriodCloseMode = accountingcontrol.PeriodCloseModeSystemScheduled
	entity.RequirePeriodCloseApproval = true

	err := newTestValidator().ValidateUpdate(t.Context(), entity)

	validationtest.Refused(t, err).Exactly(
		validationtest.At("requirePeriodCloseApproval", errortypes.ErrInvalidOperation),
	)
}

func TestValidateUpdate_RequiresCashAccountForCustomerPaymentPosting(t *testing.T) {
	t.Parallel()

	entity := validAccountingControl()
	entity.DefaultCashAccountID = pulid.Nil
	entity.AutoPostSourceEvents = append(
		entity.AutoPostSourceEvents,
		accountingcontrol.JournalSourceEventCustomerPaymentPosted,
	)

	err := newTestValidator().ValidateUpdate(t.Context(), entity)

	validationtest.Refused(t, err).Exactly(
		validationtest.At("defaultCashAccountId", errortypes.ErrRequired),
	)
}

func TestValidateUpdate_RequiresUnappliedCashAccountForCustomerPaymentPosting(t *testing.T) {
	t.Parallel()

	entity := validAccountingControl()
	entity.DefaultUnappliedCashAccountID = pulid.Nil
	entity.AutoPostSourceEvents = append(
		entity.AutoPostSourceEvents,
		accountingcontrol.JournalSourceEventCustomerPaymentPosted,
	)

	err := newTestValidator().ValidateUpdate(t.Context(), entity)

	validationtest.Refused(t, err).Exactly(
		validationtest.At("defaultUnappliedCashAccountId", errortypes.ErrRequired),
	)
}

type driverPayAccount struct {
	field string
	get   func(*accountingcontrol.AccountingControl) pulid.ID
	set   func(*accountingcontrol.AccountingControl, pulid.ID)
}

func driverPayAccounts() []driverPayAccount {
	return []driverPayAccount{
		{
			field: "defaultDriverPayExpenseAccountId",
			get: func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultDriverPayExpenseAccountID
			},
			set: func(ac *accountingcontrol.AccountingControl, id pulid.ID) {
				ac.DefaultDriverPayExpenseAccountID = id
			},
		},
		{
			field: "defaultPurchasedTransportationAccountId",
			get: func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultPurchasedTransportationAccountID
			},
			set: func(ac *accountingcontrol.AccountingControl, id pulid.ID) {
				ac.DefaultPurchasedTransportationAccountID = id
			},
		},
		{
			field: "defaultSettlementsPayableAccountId",
			get: func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultSettlementsPayableAccountID
			},
			set: func(ac *accountingcontrol.AccountingControl, id pulid.ID) {
				ac.DefaultSettlementsPayableAccountID = id
			},
		},
		{
			field: "defaultDriverAdvanceAccountId",
			get: func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultDriverAdvanceAccountID
			},
			set: func(ac *accountingcontrol.AccountingControl, id pulid.ID) {
				ac.DefaultDriverAdvanceAccountID = id
			},
		},
		{
			field: "defaultDriverReimbursementAccountId",
			get: func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultDriverReimbursementAccountID
			},
			set: func(ac *accountingcontrol.AccountingControl, id pulid.ID) {
				ac.DefaultDriverReimbursementAccountID = id
			},
		},
		{
			field: "defaultEscrowLiabilityAccountId",
			get: func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultEscrowLiabilityAccountID
			},
			set: func(ac *accountingcontrol.AccountingControl, id pulid.ID) {
				ac.DefaultEscrowLiabilityAccountID = id
			},
		},
		{
			field: "defaultEscrowInterestExpenseAccountId",
			get: func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultEscrowInterestExpenseAccountID
			},
			set: func(ac *accountingcontrol.AccountingControl, id pulid.ID) {
				ac.DefaultEscrowInterestExpenseAccountID = id
			},
		},
		{
			field: "defaultEscrowAdjustmentAccountId",
			get: func(ac *accountingcontrol.AccountingControl) pulid.ID {
				return ac.DefaultEscrowAdjustmentAccountID
			},
			set: func(ac *accountingcontrol.AccountingControl, id pulid.ID) {
				ac.DefaultEscrowAdjustmentAccountID = id
			},
		},
	}
}

func onlyIntegrityRequest(
	t *testing.T,
	integrity *validationtest.Integrity,
) validationframework.IntegrityRequest {
	t.Helper()

	requests := integrity.Requests()
	require.Len(t, requests, 1)

	return requests[0]
}
