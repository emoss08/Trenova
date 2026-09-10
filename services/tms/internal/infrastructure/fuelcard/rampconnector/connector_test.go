package rampconnector_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/fuelcard/rampconnector"
	"github.com/emoss08/trenova/pkg/fuelimport"
	"github.com/stretchr/testify/require"
)

type rampServer struct {
	server      *httptest.Server
	tokenCalls  int
	pageCalls   int
	basicUser   string
	basicPass   string
	fromDate    string
	toDate      string
	pages       []string
	tokenStatus int
}

func newRampServer(t *testing.T, pages ...string) *rampServer {
	t.Helper()

	stub := &rampServer{pages: pages, tokenStatus: http.StatusOK}
	mux := http.NewServeMux()

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		stub.tokenCalls++
		stub.basicUser, stub.basicPass, _ = r.BasicAuth()

		if stub.tokenStatus != http.StatusOK {
			w.WriteHeader(stub.tokenStatus)
			return
		}

		_, _ = w.Write([]byte(`{"access_token":"tok-1","token_type":"Bearer","expires_in":86400}`))
	})

	mux.HandleFunc("/transactions", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok-1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if stub.pageCalls == 0 {
			stub.fromDate = r.URL.Query().Get("from_date")
			stub.toDate = r.URL.Query().Get("to_date")
		}

		page := stub.pages[min(stub.pageCalls, len(stub.pages)-1)]
		stub.pageCalls++
		_, _ = w.Write([]byte(page))
	})

	stub.server = httptest.NewServer(mux)
	t.Cleanup(stub.server.Close)

	return stub
}

func (s *rampServer) config() map[string]string {
	return map[string]string{
		integration.ConfigKeyFuelClientID:        "client-1",
		integration.ConfigKeyFuelClientSecret:    "secret-1",
		integration.ConfigKeyFuelBaseURL:         s.server.URL,
		integration.ConfigKeyFuelDefaultFuelType: "Diesel",
	}
}

const oneTransaction = `{"data":[{
  "id":"txn-1",
  "amount":128.45,
  "currency_code":"USD",
  "merchant_name":"Loves Travel Stop",
  "user_transaction_time":"2026-01-05T14:22:00Z",
  "card_id":"card-1",
  "card_last_four":"4411",
  "merchant_location":{"city":"Dallas","state":"TX"},
  "card_holder":{"first_name":"Dana","last_name":"Ruiz"}
}],"page":{"next":""}}`

func fetch(
	t *testing.T,
	stub *rampServer,
	since, until int64,
) *services.FetchFuelTransactionsResult {
	t.Helper()

	result, err := rampconnector.New().FetchTransactions(
		t.Context(),
		&services.FetchFuelTransactionsRequest{
			Provider: fuelpurchase.CardProviderRamp,
			Config:   stub.config(),
			Since:    since,
			Until:    until,
		},
	)
	require.NoError(t, err)

	return result
}

func TestFetchAuthenticatesWithTheClientCredentials(t *testing.T) {
	t.Parallel()

	stub := newRampServer(t, oneTransaction)
	fetch(t, stub, 1_767_225_600, 1_767_312_000)

	require.Equal(t, 1, stub.tokenCalls)
	require.Equal(t, "client-1", stub.basicUser)
	require.Equal(t, "secret-1", stub.basicPass)
}

func TestFetchAsksForTheRequestedWindow(t *testing.T) {
	t.Parallel()

	stub := newRampServer(t, oneTransaction)
	fetch(t, stub, 1_767_225_600, 1_767_312_000)

	require.Equal(t, "2026-01-01T00:00:00Z", stub.fromDate)
	require.Equal(t, "2026-01-02T00:00:00Z", stub.toDate)
}

// The headers the connector emits have to be ones the shared header dictionary
// recognises, or a Ramp row would arrive with nothing mapped and no way to say
// what it was.
func TestFetchMapsRampHeadersOntoImportFields(t *testing.T) {
	t.Parallel()

	stub := newRampServer(t, oneTransaction)
	result := fetch(t, stub, 0, 1_767_312_000)

	require.False(t, result.Staged.HasProblems())
	require.Empty(t, result.Staged.Unmapped)
	for _, field := range []fuelimport.Field{
		fuelimport.FieldPurchasedAt,
		fuelimport.FieldVendor,
		fuelimport.FieldCity,
		fuelimport.FieldJurisdiction,
		fuelimport.FieldTotalAmount,
		fuelimport.FieldCurrency,
		fuelimport.FieldCardLastFour,
		fuelimport.FieldTransactionReference,
		fuelimport.FieldDriverName,
	} {
		require.Contains(t, result.Staged.Mapping, field, "unmapped field %s", field)
	}
	require.Equal(t, fuelpurchase.SourceFormatAPI, result.Format)
}

func TestFetchLaysCellsOutInHeaderOrder(t *testing.T) {
	t.Parallel()

	stub := newRampServer(t, oneTransaction)
	result := fetch(t, stub, 0, 1_767_312_000)

	require.Len(t, result.Staged.Rows, 1)
	require.Equal(t, []string{
		"2026-01-05T14:22:00Z",
		"Loves Travel Stop",
		"Dallas",
		"TX",
		"128.45",
		"USD",
		"4411",
		"txn-1",
		"Dana Ruiz",
		"",
	}, result.Staged.Rows[0].Cells)
}

// Ramp is a commercial Visa product: an authorization carries the merchant and
// the amount, never the gallons. A row without gallons cannot become a tax
// record, which is what the SpendOnly capability warns callers about. The row
// still has to reach the review queue rather than vanish, so a person can key
// the pump detail in.
func TestFetchHoldsEveryRowForReview(t *testing.T) {
	t.Parallel()

	stub := newRampServer(t, oneTransaction)
	result := fetch(t, stub, 0, 1_767_312_000)

	require.Len(t, result.Staged.Rows, 1)
	require.False(t, result.Staged.HasProblems())

	row := result.Staged.Rows[0]
	require.True(t, row.Failed())
	require.ErrorContains(t, row.Err, "quantity")
	require.Equal(t, services.FuelFeedCapabilitySpendOnly, rampconnector.New().Capability())
}

// Without a fuel type to fall back on, staging rejects the sheet and the entire
// window is lost. Saying so beats a feed that silently reads nothing every hour.
func TestFetchRefusesAConnectionWithoutADefaultFuelType(t *testing.T) {
	t.Parallel()

	stub := newRampServer(t, oneTransaction)
	config := stub.config()
	delete(config, integration.ConfigKeyFuelDefaultFuelType)

	_, err := rampconnector.New().FetchTransactions(
		t.Context(),
		&services.FetchFuelTransactionsRequest{
			Provider: fuelpurchase.CardProviderRamp,
			Config:   config,
		},
	)
	require.ErrorContains(t, err, "default fuel type is required")
}

func TestFetchWalksEveryPage(t *testing.T) {
	t.Parallel()

	first := `{"data":[{"id":"txn-1","amount":10,"currency_code":"USD","merchant_name":"A",` +
		`"user_transaction_time":"2026-01-05T14:22:00Z","card_last_four":"4411",` +
		`"merchant_location":{"city":"Dallas","state":"TX"}}],"page":{"next":"PLACEHOLDER"}}`
	second := `{"data":[{"id":"txn-2","amount":20,"currency_code":"USD","merchant_name":"B",` +
		`"user_transaction_time":"2026-01-06T14:22:00Z","card_last_four":"4412",` +
		`"merchant_location":{"city":"Tulsa","state":"OK"}}],"page":{"next":""}}`

	stub := newRampServer(t, "", second)
	stub.pages[0] = strings.Replace(first, "PLACEHOLDER", stub.server.URL+"/transactions", 1)

	result := fetch(t, stub, 0, 1_767_312_000)

	require.Equal(t, 2, stub.pageCalls)
	require.Len(t, result.Staged.Rows, 2)
	require.Len(t, result.Cards, 2)
}

func TestFetchReportsEachCardOnce(t *testing.T) {
	t.Parallel()

	stub := newRampServer(t, oneTransaction)
	result := fetch(t, stub, 0, 1_767_312_000)

	require.Len(t, result.Cards, 1)
	require.Equal(t, "4411", result.Cards[0].LastFour)
	require.Equal(t, "card-1", result.Cards[0].ExternalCardID)
	require.Equal(t, fuelpurchase.CardProviderRamp, result.Cards[0].Provider)
}

func TestFetchReturnsNothingForAQuietWindow(t *testing.T) {
	t.Parallel()

	stub := newRampServer(t, `{"data":[],"page":{"next":""}}`)
	result := fetch(t, stub, 0, 1_767_312_000)

	require.Empty(t, result.Staged.Rows)
	require.Empty(t, result.Cards)
}

func TestFetchSurfacesAnAuthenticationFailure(t *testing.T) {
	t.Parallel()

	stub := newRampServer(t, oneTransaction)
	stub.tokenStatus = http.StatusUnauthorized

	_, err := rampconnector.New().FetchTransactions(
		t.Context(),
		&services.FetchFuelTransactionsRequest{
			Provider: fuelpurchase.CardProviderRamp,
			Config:   stub.config(),
		},
	)
	require.ErrorContains(t, err, "HTTP 401")
}

func TestFetchRefusesAConnectionWithoutCredentials(t *testing.T) {
	t.Parallel()

	_, err := rampconnector.New().FetchTransactions(
		t.Context(),
		&services.FetchFuelTransactionsRequest{
			Provider: fuelpurchase.CardProviderRamp,
			Config:   map[string]string{integration.ConfigKeyFuelClientID: "client-1"},
		},
	)
	require.ErrorContains(t, err, "client secret is required")
}

func TestTestConnectionExchangesAToken(t *testing.T) {
	t.Parallel()

	stub := newRampServer(t, oneTransaction)
	require.NoError(t, rampconnector.New().TestConnection(t.Context(), stub.config()))
	require.Equal(t, 1, stub.tokenCalls)
}

func TestIntegrationTypeIsRampFuel(t *testing.T) {
	t.Parallel()

	require.Equal(t, integration.TypeRampFuel, rampconnector.New().IntegrationType())
}
