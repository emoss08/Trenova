package exchangerateservice

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/exchangerate"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNew_UsesDefaultHTTPClientWithTimeout(t *testing.T) {
	t.Parallel()

	svc := New(Params{
		Logger: zap.NewNop(),
	})

	assert.NotNil(t, svc.httpClient)
	assert.Equal(t, 10*time.Second, svc.httpClient.Timeout)
	assert.NotNil(t, svc.httpClient.Transport)
}

func TestNew_UsesInjectedHTTPClient(t *testing.T) {
	t.Parallel()

	client := &http.Client{Timeout: 2 * time.Second}
	svc := New(Params{
		Logger:     zap.NewNop(),
		HTTPClient: client,
	})

	assert.Same(t, client, svc.httpClient)
}

type stubExchangeRateRepository struct {
	getRateResult *exchangerate.ExchangeRate
	getRateErr    error
	upsertErr     error
	createErr     error
	upsertedRates []*exchangerate.ExchangeRate
	createdQuote  *exchangerate.SettlementQuote
}

func (s *stubExchangeRateRepository) GetRate(
	_ context.Context,
	_ *repositories.GetExchangeRateRequest,
) (*exchangerate.ExchangeRate, error) {
	return s.getRateResult, s.getRateErr
}

func (s *stubExchangeRateRepository) UpsertRates(
	_ context.Context,
	req *repositories.UpsertExchangeRatesRequest,
) error {
	s.upsertedRates = append(s.upsertedRates, req.Rates...)
	return s.upsertErr
}

func (s *stubExchangeRateRepository) CreateSettlementQuote(
	_ context.Context,
	req *repositories.CreateSettlementQuoteRequest,
) (*exchangerate.SettlementQuote, error) {
	s.createdQuote = req.Quote
	return req.Quote, s.createErr
}

func (s *stubExchangeRateRepository) GetLatestDate(
	_ context.Context,
	_ pagination.TenantInfo,
) (*time.Time, error) {
	return nil, nil
}

type stubIntegrationRepo struct {
	record *integration.Integration
}

func (s *stubIntegrationRepo) ListByTenant(
	_ context.Context,
	_ pagination.TenantInfo,
) ([]*integration.Integration, error) {
	return nil, nil
}

func (s *stubIntegrationRepo) ListEnabledByType(
	_ context.Context,
	_ integration.Type,
) ([]*integration.Integration, error) {
	return nil, nil
}

func (s *stubIntegrationRepo) GetByType(
	_ context.Context,
	_ pagination.TenantInfo,
	typ integration.Type,
) (*integration.Integration, error) {
	if s.record == nil || s.record.Type != typ {
		return nil, errortypes.NewNotFoundError("integration not found")
	}
	return s.record, nil
}

func (s *stubIntegrationRepo) Upsert(
	_ context.Context,
	_ *integration.Integration,
) (*integration.Integration, error) {
	return nil, nil
}

func TestConvertFetchesOANDARateWithBearerAuthAndDecimalPrecision(t *testing.T) {
	t.Parallel()

	var sawAuthHeader bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuthHeader = r.Header.Get("Authorization") == "Bearer test-key"
		require.Equal(t, "/v2/rates/spot.json", r.URL.Path)
		require.Equal(t, "USD", r.URL.Query().Get("base"))
		require.Equal(t, "EUR", r.URL.Query().Get("quote"))
		require.Equal(t, "true", r.URL.Query().Get("source_date"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"meta":{"request_time":"2026-05-27T12:00:00Z"},
			"quotes":[{
				"base_currency":"USD",
				"quote_currency":"EUR",
				"date_time":"2026-05-27T12:00:00Z",
				"bid":"1.234567890123",
				"ask":"1.234567890125",
				"midpoint":"1.234567890124",
				"source_date":"2026-05-27T12:00:00Z"
			}]
		}`))
	}))
	defer server.Close()

	repo := &stubExchangeRateRepository{getRateErr: errortypes.NewNotFoundError("missing")}
	svc := newTestService(t, repo, server.URL, "mid")

	result, err := svc.Convert(
		t.Context(),
		pagination.TenantInfo{},
		"usd",
		"eur",
		decimal.NewFromInt(10),
		time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC),
	)

	require.NoError(t, err)
	require.True(t, sawAuthHeader)
	require.Equal(t, "1.234567890124", result.Rate.String())
	require.Equal(t, "OANDA", result.Provider)
	require.Equal(t, "mid", result.RateType)
	require.False(t, result.SettlementEligible)
	require.Len(t, repo.upsertedRates, 1)
	require.Equal(t, "1.234567890124", repo.upsertedRates[0].SelectedRate.String())
}

func TestConvertReturnsRepositoryFailureWithoutFetchingProvider(t *testing.T) {
	t.Parallel()

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requestCount++
	}))
	defer server.Close()

	repo := &stubExchangeRateRepository{getRateErr: errors.New("database unavailable")}
	svc := newTestService(t, repo, server.URL, "mid")

	result, err := svc.Convert(
		t.Context(),
		pagination.TenantInfo{},
		"USD",
		"EUR",
		decimal.NewFromInt(10),
		time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC),
	)

	require.Nil(t, result)
	require.Error(t, err)
	require.Zero(t, requestCount)
}

func TestConvertMapsOANDANonSuccessResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":3,"message":"This request requires authorization"}`))
	}))
	defer server.Close()

	repo := &stubExchangeRateRepository{getRateErr: errortypes.NewNotFoundError("missing")}
	svc := newTestService(t, repo, server.URL, "mid")

	result, err := svc.Convert(
		t.Context(),
		pagination.TenantInfo{},
		"USD",
		"EUR",
		decimal.NewFromInt(10),
		time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC),
	)

	require.Nil(t, result)
	require.ErrorContains(t, err, "authorization")
}

func TestConvertRejectsMalformedOANDAPayload(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":`))
	}))
	defer server.Close()

	repo := &stubExchangeRateRepository{getRateErr: errortypes.NewNotFoundError("missing")}
	svc := newTestService(t, repo, server.URL, "mid")

	result, err := svc.Convert(
		t.Context(),
		pagination.TenantInfo{},
		"USD",
		"EUR",
		decimal.NewFromInt(10),
		time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC),
	)

	require.Nil(t, result)
	require.ErrorContains(t, err, "parse OANDA")
}

func TestCreateSettlementQuoteUsesDefaultMidpointAndPersistsQuote(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"meta":{"request_time":"2026-05-27T12:00:00Z"},
			"quotes":[{
				"base_currency":"USD",
				"quote_currency":"CAD",
				"date_time":"2026-05-27T12:00:00Z",
				"bid":"1.360000000000",
				"ask":"1.380000000000",
				"midpoint":"1.370000000000",
				"source_date":"2026-05-27T12:00:00Z"
			}]
		}`))
	}))
	defer server.Close()

	repo := &stubExchangeRateRepository{}
	svc := newTestService(t, repo, server.URL, "mid")

	quote, err := svc.CreateSettlementQuote(
		t.Context(),
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		&services.CreateSettlementQuoteRequest{
			FromCurrency: "USD",
			ToCurrency:   "CAD",
			Amount:       decimal.NewFromInt(100),
		},
	)

	require.NoError(t, err)
	require.Same(t, repo.createdQuote, quote)
	require.Equal(t, exchangerate.ProviderOANDA, quote.Provider)
	require.Equal(t, exchangerate.RateTypeMid, quote.RateType)
	require.Equal(t, "1.37", quote.Rate.String())
	require.Equal(t, "137", quote.ConvertedAmount.String())
	require.Len(t, repo.upsertedRates, 1)
}

func newTestService(
	t *testing.T,
	repo repositories.ExchangeRateRepository,
	baseURL string,
	defaultRateType string,
) *Service {
	t.Helper()

	encryption := encryptionservice.New(encryptionservice.Params{
		Config: &config.Config{
			Security: config.SecurityConfig{
				Encryption: config.EncryptionConfig{
					Key: "unit-test-encryption-key-with-at-least-32-bytes",
				},
			},
		},
	})
	apiKey, err := encryption.EncryptString("test-key")
	require.NoError(t, err)

	integrationSvc := integrationservice.New(integrationservice.Params{
		Logger: zap.NewNop(),
		Repo: &stubIntegrationRepo{
			record: &integration.Integration{
				Type:    integration.TypeOANDAExchangeRates,
				Enabled: true,
				Configuration: map[string]any{
					"apiKey":          apiKey,
					"baseUrl":         baseURL,
					"defaultRateType": defaultRateType,
				},
			},
		},
		Encryption: encryption,
	})

	return New(Params{
		Logger:             zap.NewNop(),
		IntegrationService: integrationSvc,
		ExchangeRateRepo:   repo,
		HTTPClient:         &http.Client{Timeout: 2 * time.Second},
	})
}
