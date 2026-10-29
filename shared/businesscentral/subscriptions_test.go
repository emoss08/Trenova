package businesscentral_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/businesscentralwebhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSubscription = "3a6f0b1c2d4e4f5a8b9c0d1e2f3a4b5c"

func subscriptionInput(clientState string) *businesscentralwebhook.SubscriptionInput {
	return &businesscentralwebhook.SubscriptionInput{
		NotificationURL: "https://hooks.trenova.example/bc/notify",
		Resource:        businesscentralwebhook.SubscriptionResource(testRef(), "salesInvoices"),
		ClientState:     clientState,
	}
}

func TestSubscriptionResource(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/api/v2.0/companies("+testCompany+")/salesInvoices",
		businesscentralwebhook.SubscriptionResource(testRef(), "salesInvoices"))
	assert.Empty(t, businesscentralwebhook.SubscriptionResource(testRef(), "sales/../x"))
	assert.Empty(
		t,
		businesscentralwebhook.SubscriptionResource(businesscentral.CompanyRef{}, "items"),
	)
}

func TestSubscriptions(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testEnvPath+"/subscriptions")
		assert.Empty(t, r.URL.RawQuery, "the subscriptions API takes no paging options")
		_, _ = w.Write(fixture(t, "subscriptions.json"))
	})

	subs, err := businesscentralwebhook.Subscriptions(t.Context(), client)
	require.NoError(t, err)
	require.Len(t, subs, 1)
	assert.Equal(t, testSubscription, subs[0].ID)
	assert.Equal(t, "https://hooks.trenova.example/bc/notify", subs[0].NotificationURL)
	assert.Equal(t, time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC), subs[0].ExpirationDateTime)
	assert.Equal(t, int64(8812), subs[0].Timestamp)
}

func TestCreateSubscription(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, testEnvPath+"/subscriptions")
		assert.Equal(t, map[string]any{
			"notificationUrl": "https://hooks.trenova.example/bc/notify",
			"resource":        "/api/v2.0/companies(" + testCompany + ")/salesInvoices",
			"clientState":     "s3cr3t-state",
		}, readJSON(t, r))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(fixture(t, "create_subscription.json"))
	})

	sub, err := businesscentralwebhook.CreateSubscription(
		t.Context(),
		client,
		subscriptionInput("s3cr3t-state"),
	)
	require.NoError(t, err)
	assert.Equal(t, testSubscription, sub.ID)
	assert.Equal(t, `W/"JzE5OzI="`, sub.ETag)
	assert.Equal(t, sub.CreatedAt.Add(businesscentral.SubscriptionLifetime), sub.ExpirationDateTime)
}

func TestCreateSubscriptionValidatesInput(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("invalid input is never sent")
	})
	in := subscriptionInput("")
	in.NotificationURL = "http://hooks.trenova.example/bc"
	_, err := businesscentralwebhook.CreateSubscription(t.Context(), client, in)
	require.ErrorIs(t, err, businesscentralerrors.ErrNotificationURL)

	in = subscriptionInput("")
	in.Resource = ""
	_, err = businesscentralwebhook.CreateSubscription(t.Context(), client, in)
	require.ErrorIs(t, err, businesscentralerrors.ErrResourceRequired)

	_, err = businesscentralwebhook.CreateSubscription(t.Context(), client,
		subscriptionInput(strings.Repeat("s", businesscentral.MaxClientStateLength+1)))
	require.ErrorIs(t, err, businesscentralerrors.ErrClientStateTooLong)
}

func TestRenewSubscriptionOmitsAnUnknownClientState(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPatch, testEnvPath+"/subscriptions('"+testSubscription+"')")
		assert.Contains(t, r.URL.EscapedPath(), "subscriptions('"+testSubscription+"')")
		assert.Equal(t, `W/"JzE5OzI="`, r.Header.Get("If-Match"))
		body := readJSON(t, r)
		assert.NotContains(t, body, "clientState")
		assert.Equal(t, "https://hooks.trenova.example/bc/notify", body["notificationUrl"])
		_, _ = w.Write(fixture(t, "create_subscription.json"))
	})

	sub, err := businesscentralwebhook.RenewSubscription(
		t.Context(),
		client,
		testSubscription,
		`W/"JzE5OzI="`,
		subscriptionInput(""),
	)
	require.NoError(t, err)
	assert.Equal(t, testSubscription, sub.ID)

	_, err = businesscentralwebhook.RenewSubscription(
		t.Context(),
		client,
		"abc')/x",
		"",
		subscriptionInput(""),
	)
	require.ErrorIs(t, err, businesscentralerrors.ErrInvalidSubscriptionID)
}

func TestDeleteSubscription(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodDelete, testEnvPath+"/subscriptions('"+testSubscription+"')")
		assert.Equal(t, "*", r.Header.Get("If-Match"))
		w.WriteHeader(http.StatusNoContent)
	})

	require.NoError(
		t,
		businesscentralwebhook.DeleteSubscription(t.Context(), client, testSubscription, ""),
	)
	require.ErrorIs(
		t,
		businesscentralwebhook.DeleteSubscription(
			t.Context(),
			client,
			strings.Repeat("a", 101),
			"",
		),
		businesscentralerrors.ErrInvalidSubscriptionID,
	)
}
