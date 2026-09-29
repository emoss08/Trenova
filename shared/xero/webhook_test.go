package xero_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testWebhookKey = "o0yVXl5Qq8hRk2mz3wXDk0x4Yb0M7x6cPzqf0vIYDtGxw+q9nq0f1yI2WbT3r5mK1Qq8hRk2mz3wXDk0x4Yb0M=="

func signWebhook(body []byte) string {
	mac := hmac.New(sha256.New, []byte(testWebhookKey))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	t.Parallel()

	body := fixture(t, "webhook_events.json")
	signature := signWebhook(body)
	require.NoError(t, xero.VerifySignature(testWebhookKey, body, signature))
	require.NoError(t, xero.VerifySignature(testWebhookKey, body, " "+signature+" "))

	tampered := append([]byte{}, body...)
	tampered[20] = 'X'
	require.ErrorIs(t, xero.VerifySignature(testWebhookKey, tampered, signature), xero.ErrInvalidSignature)
	require.ErrorIs(t, xero.VerifySignature("other-key", body, signature), xero.ErrInvalidSignature)
	require.ErrorIs(t, xero.VerifySignature(testWebhookKey, body, ""), xero.ErrMissingSignature)
	require.ErrorIs(t, xero.VerifySignature(testWebhookKey, body, "%%%"), xero.ErrMissingSignature)
	require.ErrorIs(t, xero.VerifySignature("", body, signature), xero.ErrWebhookKeyRequired)
	assert.Equal(t, "x-xero-signature", xero.SignatureHeader)
}

func TestParseWebhookReadsEventsAndTenants(t *testing.T) {
	t.Parallel()

	payload, err := xero.ParseWebhook(fixture(t, "webhook_events.json"))
	require.NoError(t, err)
	assert.Equal(t, int64(1), payload.FirstEventSequence)
	assert.Equal(t, int64(3), payload.LastEventSequence)
	assert.Equal(t, "S0m3r4Nd0mt3xt", payload.Entropy)
	require.Len(t, payload.Events, 3)

	event := payload.Events[0]
	assert.Equal(t, "https://api.xero.com/api.xro/2.0/Invoices/"+testInvoice, event.ResourceURL)
	assert.Equal(t, testInvoice, event.ResourceID)
	assert.Equal(t, "UPDATE", event.EventType)
	assert.Equal(t, "INVOICE", event.EventCategory)
	assert.Equal(t, testTenant, event.TenantID)
	assert.Equal(t, "ORGANISATION", event.TenantType)
	assert.Equal(t, time.Date(2026, 9, 25, 1, 15, 39, 902_000_000, time.UTC), event.EventDate)

	assert.Equal(t, []string{testTenant, "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d"}, payload.TenantIDs(),
		"tenants are listed once, in the order they first appear")
}

func TestParseWebhookIntentToReceive(t *testing.T) {
	t.Parallel()

	payload, err := xero.ParseWebhook(fixture(t, "webhook_intent.json"))
	require.NoError(t, err)
	assert.Empty(t, payload.Events)
	assert.Empty(t, payload.TenantIDs())
	assert.Equal(t, "YSXCMKAQBJOEMGUZEPFZ", payload.Entropy)
}

func TestParseWebhookRejectsGarbage(t *testing.T) {
	t.Parallel()

	for _, body := range []string{"", "  ", "not json", "[]", `{"events":[`} {
		_, err := xero.ParseWebhook([]byte(body))
		require.ErrorIs(t, err, xero.ErrUnexpectedPayload, body)
	}

	var nilPayload *xero.WebhookPayload
	assert.Empty(t, nilPayload.TenantIDs())
}
