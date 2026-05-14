package exchangerateservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

type Params struct {
	fx.In

	Logger             *zap.Logger
	IntegrationService *integrationservice.Service
	ExchangeRateRepo   repositories.ExchangeRateRepository
}

type Service struct {
	l                  *zap.Logger
	integrationService *integrationservice.Service
	repo               repositories.ExchangeRateRepository
}

func New(p Params) *Service {
	return &Service{
		l:                  p.Logger.Named("service.exchange-rate"),
		integrationService: p.IntegrationService,
		repo:               p.ExchangeRateRepo,
	}
}

func (s *Service) Convert(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fromCurrency, toCurrency string,
	amount decimal.Decimal,
	date time.Time,
) (*services.RateConversionResult, error) {
	fromCurrency = strings.ToUpper(strings.TrimSpace(fromCurrency))
	toCurrency = strings.ToUpper(strings.TrimSpace(toCurrency))

	if fromCurrency == toCurrency {
		return &services.RateConversionResult{
			FromCurrency: fromCurrency,
			ToCurrency:   toCurrency,
			Amount:       amount,
			Rate:         decimal.NewFromInt(1),
			Converted:    amount,
			Date:         date.Format("2006-01-02"),
		}, nil
	}

	rate, err := s.getRate(ctx, tenantInfo, fromCurrency, toCurrency, date)
	if err != nil {
		return nil, err
	}

	converted := amount.Mul(rate)

	return &services.RateConversionResult{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Amount:       amount,
		Rate:         rate,
		Converted:    converted,
		Date:         date.Format("2006-01-02"),
	}, nil
}

func (s *Service) GetLatestRates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	baseCurrency string,
) (*services.LatestRatesResult, error) {
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	return s.fetchAndCacheRates(ctx, tenantInfo, baseCurrency)
}

func (s *Service) RefreshRates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	baseCurrency string,
) error {
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	_, err := s.fetchAndCacheRates(ctx, tenantInfo, baseCurrency)
	return err
}

func (s *Service) getRate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fromCurrency, toCurrency string,
	date time.Time,
) (decimal.Decimal, error) {
	cached, err := s.repo.GetRate(ctx, &repositories.GetExchangeRateRequest{
		TenantInfo:   tenantInfo,
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Date:         date,
	})
	if err == nil && cached != nil {
		return cached.Rate, nil
	}

	cfg, apiErr := s.integrationService.GetRuntimeConfig(ctx, tenantInfo, integration.TypeExchangeRateAPI)
	if apiErr != nil {
		return decimal.Zero, apiErr
	}

	rate, apiErr := s.fetchPairRate(ctx, cfg.Config["apiKey"], fromCurrency, toCurrency)
	if apiErr != nil {
		return decimal.Zero, apiErr
	}

	upsertErr := s.repo.UpsertRates(ctx, &repositories.UpsertExchangeRatesRequest{
		TenantInfo: tenantInfo,
		Rates: []*exchangerate.ExchangeRate{
			{
				BusinessUnitID: tenantInfo.BuID,
				OrganizationID: tenantInfo.OrgID,
				FromCurrency:   fromCurrency,
				ToCurrency:     toCurrency,
				Rate:           rate,
				Date:           date.Format("2006-01-02"),
			},
		},
	})
	if upsertErr != nil {
		s.l.Error("failed to cache exchange rate", zap.Error(upsertErr))
	}

	return rate, nil
}

type exchangeRateAPIResponse struct {
	Result            string             `json:"result"`
	BaseCode          string             `json:"base_code"`
	ConversionRates   map[string]float64 `json:"conversion_rates"`
	ConversionRate    float64            `json:"conversion_rate"`
	TargetCode        string             `json:"target_code"`
	TimeLastUpdateUTC string             `json:"time_last_update_utc"`
	ErrorType         string             `json:"error-type,omitempty"`
}

func (s *Service) fetchAndCacheRates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	baseCurrency string,
) (*services.LatestRatesResult, error) {
	cfg, err := s.integrationService.GetRuntimeConfig(ctx, tenantInfo, integration.TypeExchangeRateAPI)
	if err != nil {
		return nil, err
	}

	apiKey := cfg.Config["apiKey"]
	url := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/latest/%s", apiKey, baseCurrency)

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if reqErr != nil {
		return nil, errortypes.NewBusinessError("failed to create exchange rate request").WithInternal(reqErr)
	}

	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		return nil, errortypes.NewBusinessError("failed to fetch exchange rates").WithInternal(respErr)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, errortypes.NewBusinessError("failed to read exchange rate response").WithInternal(readErr)
	}

	var apiResp exchangeRateAPIResponse
	if jsonErr := json.Unmarshal(body, &apiResp); jsonErr != nil {
		return nil, errortypes.NewBusinessError("failed to parse exchange rate response").WithInternal(jsonErr)
	}

	if apiResp.Result != "success" {
		return nil, errortypes.NewBusinessError(
			fmt.Sprintf("exchange rate API error: %s", apiResp.ErrorType),
		)
	}

	today := time.Now().Format("2006-01-02")
	rates := make([]*exchangerate.ExchangeRate, 0, len(apiResp.ConversionRates))
	for currency, rate := range apiResp.ConversionRates {
		rates = append(rates, &exchangerate.ExchangeRate{
			BusinessUnitID: tenantInfo.BuID,
			OrganizationID: tenantInfo.OrgID,
			FromCurrency:   baseCurrency,
			ToCurrency:     currency,
			Rate:           decimal.NewFromFloat(rate),
			Date:           today,
		})
	}

	if upsertErr := s.repo.UpsertRates(ctx, &repositories.UpsertExchangeRatesRequest{
		TenantInfo: tenantInfo,
		Rates:      rates,
	}); upsertErr != nil {
		s.l.Error("failed to cache exchange rates", zap.Error(upsertErr))
	}

	floatRates := make(map[string]float64, len(apiResp.ConversionRates))
	for k, v := range apiResp.ConversionRates {
		floatRates[k] = v
	}

	return &services.LatestRatesResult{
		BaseCurrency: baseCurrency,
		Date:         today,
		Rates:        floatRates,
	}, nil
}

func (s *Service) fetchPairRate(
	ctx context.Context,
	apiKey, fromCurrency, toCurrency string,
) (decimal.Decimal, error) {
	url := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/pair/%s/%s", apiKey, fromCurrency, toCurrency)

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if reqErr != nil {
		return decimal.Zero, errortypes.NewBusinessError("failed to create pair rate request").WithInternal(reqErr)
	}

	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		return decimal.Zero, errortypes.NewBusinessError("failed to fetch pair rate").WithInternal(respErr)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return decimal.Zero, errortypes.NewBusinessError("failed to read pair rate response").WithInternal(readErr)
	}

	var apiResp exchangeRateAPIResponse
	if jsonErr := json.Unmarshal(body, &apiResp); jsonErr != nil {
		return decimal.Zero, errortypes.NewBusinessError("failed to parse pair rate response").WithInternal(jsonErr)
	}

	if apiResp.Result != "success" {
		return decimal.Zero, errortypes.NewBusinessError(
			fmt.Sprintf("exchange rate API error: %s", apiResp.ErrorType),
		)
	}

	return decimal.NewFromFloat(apiResp.ConversionRate), nil
}


