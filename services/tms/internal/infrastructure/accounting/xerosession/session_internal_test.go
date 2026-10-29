package xerosession

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	sessionRevenue  = "e2bc5c1b-9a60-4f3f-8d4a-3a0f5c1e2d01"
	sessionWriteOff = "e2bc5c1b-9a60-4f3f-8d4a-3a0f5c1e2d02"
	sessionUncoded  = "e2bc5c1b-9a60-4f3f-8d4a-3a0f5c1e2d04"
)

const sessionAccountsBody = `{"Accounts":[
 {"AccountID":"` + sessionRevenue + `","Code":"200","Name":"Freight Revenue","Type":"REVENUE","Status":"ACTIVE","Class":"REVENUE"},
 {"AccountID":"` + sessionWriteOff + `","Code":"499","Name":"Short Pay Write-off","Type":"EXPENSE","Status":"ACTIVE","Class":"EXPENSE"},
 {"AccountID":"` + sessionUncoded + `","Name":"Clearing","Type":"CURRLIAB","Status":"ACTIVE","Class":"LIABILITY"}
]}`

func TestAccountCodeLookupRefetchesAStaleIndex(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	codes := NewAccountCodes(func() time.Time { return now })
	var mu sync.Mutex
	reads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/Accounts") {
			mu.Lock()
			reads++
			mu.Unlock()
			_, _ = w.Write([]byte(sessionAccountsBody))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"Type":"NotFound","Message":"not found"}`))
	}))
	t.Cleanup(server.Close)
	session := NewSession(
		[]xero.Option{xero.WithBaseURL(server.URL), xero.WithRetry(restx.RetryConfig{})},
		server.Client(),
		codes,
	)
	client, err := session.Client(accountingconnectorports.AccountingDocumentAuth{
		RealmID:     "4c7e1f20-3b5a-4d8e-9f10-2a3b4c5d6e7f",
		AccessToken: "access-token",
	})
	require.NoError(t, err)

	code, ok, err := codes.codeFor(t.Context(), client, sessionRevenue)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "200", code)

	_, ok, err = codes.codeFor(t.Context(), client, sessionUncoded)
	require.NoError(t, err)
	assert.False(t, ok)

	now = now.Add(accountCodesRefetch + time.Second)
	_, ok, err = codes.codeFor(t.Context(), client, "e2bc5c1b-9a60-4f3f-8d4a-3a0f5c1e2dff")
	require.NoError(t, err)
	assert.False(t, ok)

	id, ok, err := codes.IDFor(t.Context(), client, "499")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, sessionWriteOff, id)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, reads)
}
