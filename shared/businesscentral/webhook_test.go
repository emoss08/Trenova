package businesscentral_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/businesscentralwebhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNotifications(t *testing.T) {
	t.Parallel()

	notifications, err := businesscentralwebhook.ParseNotifications(
		fixture(t, "notifications.json"),
	)
	require.NoError(t, err)
	require.Len(t, notifications, 2)
	first := notifications[0]
	assert.Equal(t, testSubscription, first.SubscriptionID)
	assert.Equal(t, "s3cr3t-state", first.ClientState)
	assert.Equal(t, businesscentralwebhook.ChangeTypeUpdated, first.ChangeType)
	assert.Equal(t, "api/v2.0/companies("+testCompany+")/salesInvoices("+testInvoice+")",
		first.Resource)
	assert.Equal(t, time.Date(2026, 10, 6, 10, 15, 0, 250000000, time.UTC), first.LastModified)
	assert.Equal(t, time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC), first.ExpirationDateTime)
	assert.Equal(t, businesscentralwebhook.ChangeTypeCollection, notifications[1].ChangeType)
}

func TestParseNotificationsRejectsBadPayloads(t *testing.T) {
	t.Parallel()

	for _, body := range []string{"", "validationToken=abc", "[]", "{not json", `{"value":"x"}`} {
		_, err := businesscentralwebhook.ParseNotifications([]byte(body))
		require.ErrorIs(t, err, businesscentralerrors.ErrUnexpectedPayload, body)
	}

	var b strings.Builder
	b.WriteString(`{"value":[`)
	for idx := range businesscentralwebhook.MaxNotificationsPerBatch + 1 {
		if idx > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"subscriptionId":"s` + strconv.Itoa(idx) + `"}`)
	}
	b.WriteString(`]}`)
	_, err := businesscentralwebhook.ParseNotifications([]byte(b.String()))
	require.ErrorIs(t, err, businesscentralerrors.ErrTooManyNotifications)
}

func TestValidationToken(t *testing.T) {
	t.Parallel()

	token, ok := businesscentralwebhook.ValidationToken("a1B2-c3_d4.e5~!")
	assert.True(t, ok)
	assert.Equal(t, "a1B2-c3_d4.e5~!", token)

	_, ok = businesscentralwebhook.ValidationToken(strings.Repeat("x", 1024))
	assert.True(t, ok)
	for _, raw := range []string{"", " ", "has space", "tab\t", "<script>\n", "é",
		strings.Repeat("x", 1025)} {
		_, ok = businesscentralwebhook.ValidationToken(raw)
		assert.False(t, ok, raw)
	}
}

func TestDocumentLink(t *testing.T) {
	t.Parallel()

	link := businesscentral.DocumentLink(testRef(), "CRONUS USA, Inc.",
		businesscentral.PagePostedSalesInvoice, "PS-INV103001")
	assert.Equal(t, "https://businesscentral.dynamics.com/"+testTenant+"/Production/"+
		"?company=CRONUS+USA%2C+Inc.&page=132&filter=%27No.%27+IS+%27PS-INV103001%27", link)

	linker := businesscentral.NewLinker(businesscentral.WithWebURL("https://bc.example.test/"))
	assert.Equal(t, "https://bc.example.test/"+testTenant+"/Production/"+
		"?company=A%26B&page=20&filter=%27No.%27+IS+%27X%26Y%27",
		linker.DocumentLink(testRef(), "A&B", businesscentral.PageGeneralLedgerEntries, "X&Y"))

	assert.Empty(t, businesscentral.DocumentLink(testRef(), "", 132, "1"))
	assert.Empty(t, businesscentral.DocumentLink(testRef(), "Co", 132, " "))
	assert.Empty(t, businesscentral.DocumentLink(businesscentral.CompanyRef{}, "Co", 132, "1"))
	assert.Equal(t, 25, businesscentral.PageCustomerLedgerEntries)
	assert.Equal(t, 134, businesscentral.PagePostedSalesCreditMemo)
	assert.Equal(t, 138, businesscentral.PagePostedPurchaseInvoice)
	assert.Equal(t, 140, businesscentral.PagePostedPurchaseCreditMemo)
}

func TestAllowedHosts(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, []string{
		"api.businesscentral.dynamics.com",
		"login.microsoftonline.com",
		"businesscentral.dynamics.com",
	}, businesscentral.AllowedHosts())
}
