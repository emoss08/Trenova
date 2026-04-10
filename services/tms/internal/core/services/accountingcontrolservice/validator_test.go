package accountingcontrolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingReferenceChecker struct {
	valid    bool
	requests []*validationframework.ReferenceRequest
}

func (c *capturingReferenceChecker) CheckReference(
	_ context.Context,
	req *validationframework.ReferenceRequest,
) (bool, error) {
	c.requests = append(c.requests, req)
	return c.valid, nil
}

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

	checker := &capturingReferenceChecker{valid: true}
	v := NewTestValidatorWithReferenceChecker(checker)
	entity := validAccountingControl()
	entity.DefaultTaxLiabilityAccountID = pulid.ID("")
	entity.DefaultWriteOffAccountID = pulid.ID("")
	entity.DefaultRetainedEarningsAccountID = pulid.ID("")

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.Nil(t, multiErr)
	assert.Len(t, checker.requests, 4)
}

func TestValidateUpdate_UsesGLAccountsTableForAccountReferences(t *testing.T) {
	t.Parallel()

	checker := &capturingReferenceChecker{valid: true}
	v := NewTestValidatorWithReferenceChecker(checker)
	entity := validAccountingControl()
	entity.RealizedFXGainAccountID = pulid.MustNew("gla_")
	entity.RealizedFXLossAccountID = pulid.MustNew("gla_")
	entity.CurrencyMode = tenant.CurrencyModeMultiCurrency
	entity.ExchangeRateOverridePolicy = tenant.ExchangeRateOverrideRequireApproval

	multiErr := v.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, multiErr)
	require.Len(t, checker.requests, 9)
	for _, req := range checker.requests {
		assert.Equal(t, "gl_accounts", req.TableName)
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

func assertErrorField(t *testing.T, multiErr *errortypes.MultiError, field string) {
	t.Helper()

	for _, err := range multiErr.Errors {
		if err.Field == field {
			return
		}
	}

	t.Fatalf("expected validation error for field %q, got %#v", field, multiErr.Errors)
}
