package nsconnector

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureError(t *testing.T, status int, fixture string) error {
	t.Helper()
	conn := testConnector(t, &fakeNS{respond: func(nsCall) reply {
		return fixtureReply(t, status, fixture)
	}})
	s, err := conn.open(testAuth())
	require.NoError(t, err)
	err = s.client.Delete(t.Context(), recordInvoice, testInvoice)
	require.Error(t, err)
	return err
}

func TestClassifyDocumentErrorMapsEveryNetSuiteCode(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	cases := []struct {
		name     string
		status   int
		fixture  string
		category accountingsync.SyncErrorCategory
		code     string
	}{
		{
			"invalid login",
			401,
			"error_401_invalid_login.json",
			accountingsync.SyncErrorAuth,
			"INVALID_LOGIN",
		},
		{
			"forbidden",
			403,
			"error_403_forbidden.json",
			accountingsync.SyncErrorConfiguration,
			"INSUFFICIENT_PERMISSION",
		},
		{
			"permission",
			400,
			"error_permission.json",
			accountingsync.SyncErrorConfiguration,
			"USER_ERROR",
		},
		{
			"not found",
			404,
			"error_404_nonexistent.json",
			accountingsync.SyncErrorNotFound,
			"NONEXISTENT_ID",
		},
		{
			"reference",
			400,
			"error_invalid_reference.json",
			accountingsync.SyncErrorMapping,
			"INVALID_CONTENT",
		},
		{
			"content",
			400,
			"error_invalid_content.json",
			accountingsync.SyncErrorValidation,
			"INVALID_CONTENT",
		},
		{
			"request",
			400,
			"error_invalid_request.json",
			accountingsync.SyncErrorValidation,
			"INVALID_REQUEST",
		},
		{
			"business rule",
			400,
			"error_user_error.json",
			accountingsync.SyncErrorValidation,
			"USER_ERROR",
		},
		{
			"closed period",
			400,
			"error_period_closed.json",
			accountingsync.SyncErrorClosedPeriod,
			"USER_ERROR",
		},
		{
			"idempotency",
			409,
			"error_idempotency.json",
			accountingsync.SyncErrorDuplicate,
			"IDEMPOTENCY_ERROR",
		},
		{
			"external id",
			400,
			"error_duplicate_external_id.json",
			accountingsync.SyncErrorDuplicate,
			"USER_ERROR",
		},
		{
			"unexpected",
			500,
			"error_500.json",
			accountingsync.SyncErrorTransient,
			"UNEXPECTED_ERROR",
		},
	}
	for _, tc := range cases {
		err := fixtureError(t, tc.status, tc.fixture)
		classified := conn.ClassifyDocumentError(err)
		require.NotNil(t, classified, tc.name)
		assert.Equal(t, tc.category, classified.Category, tc.name)
		assert.Equal(t, tc.code, classified.Code, tc.name)
		assert.NotEmpty(t, classified.Message, tc.name)
	}
}

func TestClassifyDocumentErrorExplainsPermissionsAndDetails(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	permission := conn.ClassifyDocumentError(fixtureError(t, 400, "error_permission.json"))
	assert.Equal(
		t,
		"The role lacks permission for Transactions -> Invoice in NetSuite",
		permission.Message,
	)
	assert.Contains(t, permission.Resolution, "Transactions -> Invoice")

	forbidden := conn.ClassifyDocumentError(
		&netsuiteerrors.APIError{Status: http.StatusForbidden, Title: "Forbidden"},
	)
	assert.Equal(t, "The role lacks permission for this record in NetSuite", forbidden.Message)

	content := conn.ClassifyDocumentError(fixtureError(t, 400, "error_invalid_content.json"))
	assert.Contains(t, content.Message, "invalid field value 2026-13-45")
	assert.Equal(t, "Correct the record in Trenova or NetSuite, then retry", content.Resolution)
}

func TestClassifyDocumentErrorWaitsTenSecondsOnTheConcurrencyLimit(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	classified := conn.ClassifyDocumentError(fixtureError(t, 429, "error_429_concurrency.json"))
	assert.Equal(t, accountingsync.SyncErrorRateLimited, classified.Category)
	assert.Equal(t, 10*time.Second, classified.RetryAfter)
	assert.Equal(t, "CONCURRENCY_LIMIT_EXCEEDED", classified.Code)
}

func TestClassifyDocumentErrorPassesThroughAndHandlesLocalFailures(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	assert.Nil(t, conn.ClassifyDocumentError(nil))
	own := &accountingsync.SyncError{Category: accountingsync.SyncErrorConflict, Message: "x"}
	assert.Same(t, own, conn.ClassifyDocumentError(fmt.Errorf("wrapped: %w", own)))

	cases := map[error]accountingsync.SyncErrorCategory{
		ErrNotConfigured:                    accountingsync.SyncErrorConfiguration,
		errNoRedirect:                       accountingsync.SyncErrorConfiguration,
		errLinesRequired:                    accountingsync.SyncErrorValidation,
		netsuiteerrors.ErrInvalidExternalID: accountingsync.SyncErrorValidation,
		&netsuiteerrors.OAuthError{Code: "invalid_grant"}: accountingsync.SyncErrorAuth,
		context.DeadlineExceeded:                          accountingsync.SyncErrorTransient,
		errors.New("boom"):                                accountingsync.SyncErrorTransient,
		fmt.Errorf("%w: item", errDuplicateName):          accountingsync.SyncErrorDuplicate,
	}
	for err, want := range cases {
		assert.Equal(t, want, conn.ClassifyDocumentError(err).Category, err.Error())
	}
	assert.Equal(t, "invalid_grant", errorCode(&netsuiteerrors.OAuthError{Code: "invalid_grant"}))
	assert.Equal(t, "400", errorCode(&netsuiteerrors.APIError{Status: 400}))
}

func TestUnreachableCoversNotFoundAndDNSFailures(t *testing.T) {
	t.Parallel()

	assert.True(t, isUnreachable(&net.DNSError{Err: "no such host", Name: "x"}))
	assert.True(t, isUnreachable(&netsuiteerrors.APIError{Status: http.StatusNotFound}))
	assert.False(t, isUnreachable(errors.New("boom")))
	assert.True(t, deleteRefused(fixtureError(t, 400, "error_user_error.json")))
	assert.True(t, deleteRefused(fixtureError(t, 400, "error_period_closed.json")))
	assert.False(t, deleteRefused(fixtureError(t, 400, "error_permission.json")))
	assert.False(t, deleteRefused(fixtureError(t, 500, "error_500.json")))
}
