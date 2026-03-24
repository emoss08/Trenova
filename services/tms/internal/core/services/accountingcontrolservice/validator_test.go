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
	entity.DefaultCurrencyCode = "sek"

	multiErr := v.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, multiErr)
}

func TestValidateUpdate_RejectsUnknownCurrencyCode(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.DefaultCurrencyCode = "ZZZ"

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "defaultCurrencyCode")
}

func TestValidateUpdate_RejectsDuplicateJournalEntryCriteria(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.AutoCreateJournalEntries = true
	entity.JournalEntryCriteria = []tenant.JournalEntryCriteriaType{
		tenant.JournalEntryCriteriaInvoicePosted,
		tenant.JournalEntryCriteriaInvoicePosted,
	}
	entity.DefaultRevenueAccountID = pulid.MustNew("gla_")
	entity.DefaultExpenseAccountID = pulid.MustNew("gla_")
	entity.DefaultARAccountID = pulid.MustNew("gla_")
	entity.DefaultAPAccountID = pulid.MustNew("gla_")

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "journalEntryCriteria[1]")
}

func TestValidateUpdate_DoesNotAddSecondAccountingMethodErrorWhenEnumIsInvalid(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.AccountingMethod = tenant.AccountingMethodType("Broken")

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	require.Len(t, errorsForField(multiErr, "accountingMethod"), 1)
}

func TestValidateUpdate_SkipsOptionalReferenceChecksForZeroValueIDs(t *testing.T) {
	t.Parallel()

	checker := &capturingReferenceChecker{valid: true}
	v := NewTestValidatorWithReferenceChecker(checker)
	entity := validAccountingControl()

	multiErr := v.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, multiErr)
	assert.Empty(t, checker.requests)
}

func TestValidateUpdate_UsesGLAccountsTableForAccountReferences(t *testing.T) {
	t.Parallel()

	checker := &capturingReferenceChecker{valid: true}
	v := NewTestValidatorWithReferenceChecker(checker)
	entity := validAccountingControl()
	entity.DefaultRevenueAccountID = pulid.MustNew("gla_")
	entity.DefaultExpenseAccountID = pulid.MustNew("gla_")
	entity.DefaultARAccountID = pulid.MustNew("gla_")
	entity.DefaultAPAccountID = pulid.MustNew("gla_")
	entity.DefaultTaxAccountID = pulid.MustNew("gla_")
	entity.DefaultDeferredRevenueAccountID = pulid.MustNew("gla_")
	entity.DefaultCostOfServiceAccountID = pulid.MustNew("gla_")
	entity.DefaultRetainedEarningsAccountID = pulid.MustNew("gla_")
	entity.CurrencyGainAccountID = pulid.MustNew("gla_")
	entity.CurrencyLossAccountID = pulid.MustNew("gla_")

	multiErr := v.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, multiErr)
	require.Len(t, checker.requests, 10)
	for _, req := range checker.requests {
		assert.Equal(t, "gl_accounts", req.TableName)
		assert.Equal(t, entity.OrganizationID, req.OrganizationID)
		assert.Equal(t, entity.BusinessUnitID, req.BusinessUnitID)
	}
}

func TestValidateUpdate_RejectsNonPositiveReconciliationThresholdWhenEnabled(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	entity := validAccountingControl()
	entity.EnableReconciliation = true
	entity.ReconciliationThreshold = decimal.Zero

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "reconciliationThreshold")
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

func errorsForField(multiErr *errortypes.MultiError, field string) []*errortypes.Error {
	if multiErr == nil {
		return nil
	}

	var errs []*errortypes.Error
	for _, err := range multiErr.Errors {
		if err.Field == field {
			errs = append(errs, err)
		}
	}

	return errs
}
