package exchangerateservice

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/exchangerate"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultOANDABaseURL = "https://exchange-rates-api.oanda.com"
	defaultDateLayout   = "2006-01-02"
	settlementQuoteTTL  = 15 * time.Minute
	oandaSpotRatesPath  = "/v2/rates/spot.json"
)

var defaultRefreshTargetCurrencies = []string{
	"USD", "EUR", "GBP", "CAD", "MXN", "AUD", "JPY", "CHF", "BRL", "CNY",
}

type Params struct {
	fx.In

	Logger             *zap.Logger
	IntegrationService *integrationservice.Service
	ExchangeRateRepo   repositories.ExchangeRateRepository
	HTTPClient         *http.Client `optional:"true"`
}

type Service struct {
	l                  *zap.Logger
	integrationService *integrationservice.Service
	repo               repositories.ExchangeRateRepository
	httpClient         *http.Client
}

func New(p Params) *Service {
	httpClient := p.HTTPClient
	if httpClient == nil {
		httpClient = newDefaultHTTPClient()
	}

	return &Service{
		l:                  p.Logger.Named("service.exchange-rate"),
		integrationService: p.IntegrationService,
		repo:               p.ExchangeRateRepo,
		httpClient:         httpClient,
	}
}

func newDefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: time.Second,
		},
	}
}

func (s *Service) Convert(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fromCurrency, toCurrency string,
	amount decimal.Decimal,
	date time.Time,
) (*services.RateConversionResult, error) {
	fromCurrency = normalizeCurrency(fromCurrency)
	toCurrency = normalizeCurrency(toCurrency)

	if fromCurrency == toCurrency {
		return &services.RateConversionResult{
			FromCurrency:       fromCurrency,
			ToCurrency:         toCurrency,
			Amount:             amount,
			Rate:               decimal.NewFromInt(1),
			Converted:          amount,
			Date:               date.Format(defaultDateLayout),
			SettlementEligible: false,
		}, nil
	}

	rate, err := s.getRate(ctx, tenantInfo, fromCurrency, toCurrency, date)
	if err != nil {
		return nil, err
	}

	fetchedAt := rate.FetchedAt
	sourceTimestamp := rate.SourceTimestamp
	return &services.RateConversionResult{
		FromCurrency:       fromCurrency,
		ToCurrency:         toCurrency,
		Amount:             amount,
		Rate:               rate.SelectedRate,
		Converted:          amount.Mul(rate.SelectedRate),
		Date:               rate.Date,
		Provider:           string(rate.Provider),
		RateType:           string(rate.RateType),
		SourceTimestamp:    &sourceTimestamp,
		FetchedAt:          &fetchedAt,
		SettlementEligible: false,
	}, nil
}

func (s *Service) GetLatestRates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	baseCurrency string,
) (*services.LatestRatesResult, error) {
	baseCurrency = normalizeCurrency(baseCurrency)
	return s.fetchAndCacheRates(ctx, tenantInfo, baseCurrency, nil, time.Time{})
}

func (s *Service) RefreshRates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	baseCurrency string,
) error {
	baseCurrency = normalizeCurrency(baseCurrency)
	targets := refreshTargetsForBase(baseCurrency)
	_, err := s.fetchAndCacheRates(ctx, tenantInfo, baseCurrency, targets, time.Time{})
	return err
}

func (s *Service) CreateSettlementQuote(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *services.CreateSettlementQuoteRequest,
) (*exchangerate.SettlementQuote, error) {
	if req == nil {
		return nil, errortypes.NewBusinessError("settlement quote request is required")
	}

	fromCurrency := normalizeCurrency(req.FromCurrency)
	toCurrency := normalizeCurrency(req.ToCurrency)
	if fromCurrency == "" || toCurrency == "" {
		return nil, errortypes.NewBusinessError("fromCurrency and toCurrency are required")
	}
	if !req.Amount.IsPositive() {
		return nil, errortypes.NewBusinessError("amount must be greater than zero")
	}

	cfg, err := s.oandaConfig(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	rateType := normalizeRateType(req.RateType, cfg.DefaultRateType)
	quoteDate, err := parseOptionalDate(req.Date)
	if err != nil {
		return nil, err
	}

	var selectedRate decimal.Decimal
	var sourceTimestamp time.Time
	var fetchedAt time.Time
	if fromCurrency == toCurrency {
		selectedRate = decimal.NewFromInt(1)
		fetchedAt = time.Now().UTC()
		sourceTimestamp = fetchedAt
	} else {
		snapshot, fetchErr := s.fetchOANDAPair(ctx, cfg, fromCurrency, toCurrency, rateType, quoteDate)
		if fetchErr != nil {
			return nil, fetchErr
		}
		if upsertErr := s.repo.UpsertRates(ctx, &repositories.UpsertExchangeRatesRequest{
			TenantInfo: tenantInfo,
			Rates:      []*exchangerate.ExchangeRate{snapshot},
		}); upsertErr != nil {
			return nil, errortypes.NewBusinessError("failed to persist exchange rate snapshot").
				WithInternal(upsertErr)
		}
		selectedRate = snapshot.SelectedRate
		sourceTimestamp = snapshot.SourceTimestamp
		fetchedAt = snapshot.FetchedAt
	}

	quote := &exchangerate.SettlementQuote{
		BusinessUnitID:  tenantInfo.BuID,
		OrganizationID:  tenantInfo.OrgID,
		Provider:        exchangerate.ProviderOANDA,
		FromCurrency:    fromCurrency,
		ToCurrency:      toCurrency,
		Amount:          req.Amount,
		Rate:            selectedRate,
		ConvertedAmount: req.Amount.Mul(selectedRate),
		RateType:        rateType,
		SourceTimestamp: sourceTimestamp,
		FetchedAt:       fetchedAt,
		ExpiresAt:       fetchedAt.Add(settlementQuoteTTL),
	}

	return s.repo.CreateSettlementQuote(ctx, &repositories.CreateSettlementQuoteRequest{
		TenantInfo: tenantInfo,
		Quote:      quote,
	})
}

func (s *Service) getRate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fromCurrency, toCurrency string,
	date time.Time,
) (*exchangerate.ExchangeRate, error) {
	cfg, err := s.oandaConfig(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	rateType := normalizeRateType("", cfg.DefaultRateType)
	cached, err := s.repo.GetRate(ctx, &repositories.GetExchangeRateRequest{
		TenantInfo:   tenantInfo,
		Provider:     exchangerate.ProviderOANDA,
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		RateType:     rateType,
		Date:         date,
	})
	if err == nil {
		return cached, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, errortypes.NewBusinessError("failed to retrieve cached exchange rate").
			WithInternal(err)
	}

	rate, err := s.fetchOANDAPair(ctx, cfg, fromCurrency, toCurrency, rateType, date)
	if err != nil {
		return nil, err
	}

	if upsertErr := s.repo.UpsertRates(ctx, &repositories.UpsertExchangeRatesRequest{
		TenantInfo: tenantInfo,
		Rates:      []*exchangerate.ExchangeRate{rate},
	}); upsertErr != nil {
		return nil, errortypes.NewBusinessError("failed to persist exchange rate snapshot").
			WithInternal(upsertErr)
	}

	return rate, nil
}

type oandaRuntimeConfig struct {
	APIKey          string
	BaseURL         string
	DefaultRateType exchangerate.RateType
}

func (s *Service) oandaConfig(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*oandaRuntimeConfig, error) {
	cfg, err := s.integrationService.GetRuntimeConfig(
		ctx,
		tenantInfo,
		integration.TypeOANDAExchangeRates,
	)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.Config["baseUrl"]), "/")
	if baseURL == "" {
		baseURL = defaultOANDABaseURL
	}

	return &oandaRuntimeConfig{
		APIKey:          strings.TrimSpace(cfg.Config["apiKey"]),
		BaseURL:         baseURL,
		DefaultRateType: normalizeRateType(cfg.Config["defaultRateType"], exchangerate.RateTypeMid),
	}, nil
}

type oandaSpotResponse struct {
	Meta   oandaMeta        `json:"meta"`
	Quotes []oandaSpotQuote `json:"quotes"`
}

type oandaMeta struct {
	RequestTime string `json:"request_time"`
}

type oandaSpotQuote struct {
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	DateTime      string `json:"date_time"`
	Bid           string `json:"bid"`
	Ask           string `json:"ask"`
	Midpoint      string `json:"midpoint"`
	SourceDate    string `json:"source_date"`
}

type oandaErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Service) fetchAndCacheRates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	baseCurrency string,
	targetCurrencies []string,
	date time.Time,
) (*services.LatestRatesResult, error) {
	cfg, err := s.oandaConfig(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	rateType := normalizeRateType("", cfg.DefaultRateType)
	snapshots, err := s.fetchOANDARates(ctx, cfg, baseCurrency, targetCurrencies, rateType, date)
	if err != nil {
		return nil, err
	}

	if len(snapshots) > 0 {
		if upsertErr := s.repo.UpsertRates(ctx, &repositories.UpsertExchangeRatesRequest{
			TenantInfo: tenantInfo,
			Rates:      snapshots,
		}); upsertErr != nil {
			return nil, errortypes.NewBusinessError("failed to persist exchange rate snapshots").
				WithInternal(upsertErr)
		}
	}

	rates := make(map[string]decimal.Decimal, len(snapshots))
	resultDate := time.Now().UTC().Format(defaultDateLayout)
	for idx := range snapshots {
		rate := snapshots[idx]
		rates[rate.ToCurrency] = rate.SelectedRate
		resultDate = rate.Date
	}

	return &services.LatestRatesResult{
		BaseCurrency: baseCurrency,
		Date:         resultDate,
		Provider:     string(exchangerate.ProviderOANDA),
		RateType:     string(rateType),
		Rates:        rates,
	}, nil
}

func (s *Service) fetchOANDAPair(
	ctx context.Context,
	cfg *oandaRuntimeConfig,
	fromCurrency, toCurrency string,
	rateType exchangerate.RateType,
	date time.Time,
) (*exchangerate.ExchangeRate, error) {
	rates, err := s.fetchOANDARates(ctx, cfg, fromCurrency, []string{toCurrency}, rateType, date)
	if err != nil {
		return nil, err
	}
	if len(rates) == 0 {
		return nil, errortypes.NewBusinessError("OANDA returned no exchange rate for currency pair")
	}
	return rates[0], nil
}

func (s *Service) fetchOANDARates(
	ctx context.Context,
	cfg *oandaRuntimeConfig,
	baseCurrency string,
	targetCurrencies []string,
	rateType exchangerate.RateType,
	date time.Time,
) ([]*exchangerate.ExchangeRate, error) {
	endpoint, err := url.Parse(cfg.BaseURL + oandaSpotRatesPath)
	if err != nil {
		return nil, errortypes.NewBusinessError("invalid OANDA base URL").WithInternal(err)
	}

	query := endpoint.Query()
	query.Add("base", baseCurrency)
	for _, targetCurrency := range targetCurrencies {
		targetCurrency = normalizeCurrency(targetCurrency)
		if targetCurrency != "" && targetCurrency != baseCurrency {
			query.Add("quote", targetCurrency)
		}
	}
	query.Set("source_date", "true")
	if !date.IsZero() {
		query.Set("date_time", date.UTC().Format(defaultDateLayout))
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to create OANDA exchange rate request").
			WithInternal(err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to fetch OANDA exchange rates").
			WithInternal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to read OANDA exchange rate response").
			WithInternal(err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, mapOANDAError(resp.StatusCode, body)
	}

	var apiResp oandaSpotResponse
	if err = sonic.Unmarshal(body, &apiResp); err != nil {
		return nil, errortypes.NewBusinessError("failed to parse OANDA exchange rate response").
			WithInternal(err)
	}

	fetchedAt := parseOANDATimestamp(apiResp.Meta.RequestTime, time.Now().UTC())
	rates := make([]*exchangerate.ExchangeRate, 0, len(apiResp.Quotes))
	for idx := range apiResp.Quotes {
		rate, parseErr := buildExchangeRate(apiResp.Quotes[idx], rateType, fetchedAt)
		if parseErr != nil {
			return nil, parseErr
		}
		rates = append(rates, rate)
	}

	return rates, nil
}

func buildExchangeRate(
	quote oandaSpotQuote,
	rateType exchangerate.RateType,
	fetchedAt time.Time,
) (*exchangerate.ExchangeRate, error) {
	bid, err := decimal.NewFromString(quote.Bid)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to parse OANDA bid rate").WithInternal(err)
	}
	ask, err := decimal.NewFromString(quote.Ask)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to parse OANDA ask rate").WithInternal(err)
	}
	mid, err := decimal.NewFromString(quote.Midpoint)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to parse OANDA midpoint rate").
			WithInternal(err)
	}

	selectedRate := mid
	switch rateType {
	case exchangerate.RateTypeBid:
		selectedRate = bid
	case exchangerate.RateTypeAsk:
		selectedRate = ask
	}

	sourceTimestamp := parseOANDATimestamp(quote.DateTime, time.Time{})
	if sourceTimestamp.IsZero() {
		sourceTimestamp = parseOANDATimestamp(quote.SourceDate, fetchedAt)
	}

	return &exchangerate.ExchangeRate{
		Provider:           exchangerate.ProviderOANDA,
		FromCurrency:       normalizeCurrency(quote.BaseCurrency),
		ToCurrency:         normalizeCurrency(quote.QuoteCurrency),
		RateType:           rateType,
		Bid:                bid,
		Ask:                ask,
		Mid:                mid,
		SelectedRate:       selectedRate,
		Date:               sourceTimestamp.Format(defaultDateLayout),
		SourceTimestamp:    sourceTimestamp,
		FetchedAt:          fetchedAt,
		SettlementEligible: true,
	}, nil
}

func mapOANDAError(statusCode int, body []byte) error {
	var apiErr oandaErrorResponse
	if err := sonic.Unmarshal(body, &apiErr); err != nil {
		return errortypes.NewBusinessError(
			fmt.Sprintf("OANDA exchange rate request failed with status %d", statusCode),
		).WithInternal(err)
	}

	message := strings.TrimSpace(apiErr.Message)
	if message == "" {
		message = fmt.Sprintf("OANDA exchange rate request failed with status %d", statusCode)
	}

	return errortypes.NewBusinessError(message)
}

func parseOptionalDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse(defaultDateLayout, value)
	if err != nil {
		return time.Time{}, errortypes.NewBusinessError("invalid date format, use YYYY-MM-DD").
			WithInternal(err)
	}
	return parsed, nil
}

func parseOANDATimestamp(value string, fallback time.Time) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	layouts := [...]string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		defaultDateLayout,
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC()
		}
	}
	return fallback
}

func normalizeCurrency(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeRateType(
	value any,
	defaultRateType exchangerate.RateType,
) exchangerate.RateType {
	rateType := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
	switch rateType {
	case string(exchangerate.RateTypeBid):
		return exchangerate.RateTypeBid
	case string(exchangerate.RateTypeAsk):
		return exchangerate.RateTypeAsk
	case string(exchangerate.RateTypeMid), "midpoint", "":
		if defaultRateType != "" {
			return defaultRateType
		}
		return exchangerate.RateTypeMid
	default:
		return exchangerate.RateTypeMid
	}
}

func refreshTargetsForBase(baseCurrency string) []string {
	targets := make([]string, 0, len(defaultRefreshTargetCurrencies)-1)
	for _, currency := range defaultRefreshTargetCurrencies {
		if currency != baseCurrency {
			targets = append(targets, currency)
		}
	}
	return targets
}
