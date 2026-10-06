package bcconnector

import (
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSubscription  = "3a6f0b1c2d4e4f5a8b9c0d1e2f3a4b5c"
	testNotifyURL     = "https://hooks.trenova.example/bc/notify"
	subscriptionsPath = "/subscriptions"
)

func TestSubscriptionResourcesAndRenewWindow(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeBC{})
	resources := conn.SubscriptionResources()
	assert.Equal(t, []string{
		"salesInvoices", "salesCreditMemos", "purchaseInvoices",
		"customers", "vendors", "items", "accounts",
	}, resources)
	resources[0] = "changed"
	assert.Equal(t, "salesInvoices", conn.SubscriptionResources()[0])
	assert.Equal(t, 24*time.Hour, conn.SubscriptionRenewWindow())
}

func TestSubscribeSendsTheClientStateItIsGiven(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		if call.is(http.MethodPost, subscriptionsPath) {
			return http.StatusCreated, readFixture(t, "create_subscription.json")
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	grant, err := conn.Subscribe(t.Context(), &services.AccountingSubscribeRequest{
		Auth:            testAuth(),
		Resource:        "salesInvoices",
		NotificationURL: testNotifyURL,
		ClientState:     "sealed-by-the-service",
	})
	require.NoError(t, err)
	assert.Equal(t, &accountingsync.WebhookSubscriptionGrant{
		ExternalID:      testSubscription,
		NotificationURL: testNotifyURL,
		ETag:            `W/"JzE5OzI="`,
		ExpiresAt:       time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC),
	}, grant)

	call, ok := fake.find(http.MethodPost, subscriptionsPath)
	require.True(t, ok)
	assert.Equal(t, testEnv, call.Env)
	assert.Equal(t, testNotifyURL, call.Body["notificationUrl"])
	assert.Equal(t, "/api/v2.0/companies("+testCompany+")/salesInvoices", call.Body["resource"])
	assert.Equal(t, "sealed-by-the-service", call.Body["clientState"])

	_, err = conn.Subscribe(t.Context(), &services.AccountingSubscribeRequest{
		Auth: testAuth(), Resource: "sales invoices", NotificationURL: testNotifyURL,
	})
	require.ErrorIs(t, err, errUnknownResource)
}

func TestRenewSubscriptionPatchesWithTheETag(t *testing.T) {
	t.Parallel()

	path := subscriptionsPath + "('" + testSubscription + "')"
	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		if call.is(http.MethodPatch, path) {
			return http.StatusOK, `{"@odata.etag":"W/\"next\"","subscriptionId":"` +
				testSubscription + `","notificationUrl":"","resource":"x","clientState":"s"}`
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	grant, err := conn.RenewSubscription(t.Context(), &services.AccountingRenewSubscriptionRequest{
		Auth:            testAuth(),
		Resource:        "customers",
		ExternalID:      testSubscription,
		NotificationURL: testNotifyURL,
		ClientState:     "s",
		ETag:            `W/"JzE5OzI="`,
	})
	require.NoError(t, err)
	assert.Equal(t, `W/"next"`, grant.ETag)
	assert.Equal(t, testNotifyURL, grant.NotificationURL)
	assert.Equal(t, testNow.Add(72*time.Hour), grant.ExpiresAt)

	call, ok := fake.find(http.MethodPatch, path)
	require.True(t, ok)
	assert.Equal(t, `W/"JzE5OzI="`, call.IfMatch)
	assert.Equal(t, "/api/v2.0/companies("+testCompany+")/customers", call.Body["resource"])
}

func TestRenewingAGoneSubscriptionIsReportedAsGone(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeBC{})
	_, err := conn.RenewSubscription(t.Context(), &services.AccountingRenewSubscriptionRequest{
		Auth:            testAuth(),
		Resource:        "items",
		ExternalID:      testSubscription,
		NotificationURL: testNotifyURL,
	})
	require.Error(t, err)
	assert.True(t, conn.IsSubscriptionGone(err))
	assert.False(t, conn.IsSubscriptionGone(errUnknownResource))
}

func TestUnsubscribeDeletesAndToleratesAMissingSubscription(t *testing.T) {
	t.Parallel()

	path := subscriptionsPath + "('" + testSubscription + "')"
	fake := &fakeBC{}
	conn := testConnector(t, fake)
	require.NoError(t, conn.Unsubscribe(t.Context(), &services.AccountingUnsubscribeRequest{
		Auth: testAuth(), ExternalID: testSubscription, ETag: `W/"e"`,
	}))
	call, ok := fake.find(http.MethodDelete, path)
	require.True(t, ok)
	assert.Equal(t, `W/"e"`, call.IfMatch)

	failing := &fakeBC{respond: func(bcCall) (int, string) {
		return http.StatusForbidden, readFixture(t, "error_403.json")
	}}
	conn = testConnector(t, failing)
	require.Error(t, conn.Unsubscribe(t.Context(), &services.AccountingUnsubscribeRequest{
		Auth: testAuth(), ExternalID: testSubscription,
	}))
	require.Error(t, conn.Unsubscribe(t.Context(), &services.AccountingUnsubscribeRequest{
		Auth: services.AccountingDocumentAuth{RealmID: "bad"}, ExternalID: testSubscription,
	}))
}
