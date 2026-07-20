package fuelsurchargeservice

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	defaultEIABaseURL     = "https://api.eia.gov/v2"
	eiaDieselPricePath    = "/petroleum/pri/gnd/data/"
	eiaTrailingWeeks      = 8
	fuelPriceEventType    = "fuel.price.updated"
	fuelPriceNotifySource = "fuel-price-ingest"
)

type RefreshEIAPricesResult struct {
	Skipped     bool
	SeriesCount int
	NewRows     int
}

type eiaRuntimeConfig struct {
	APIKey  string
	BaseURL string
}

func (s *Service) eiaConfig(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*eiaRuntimeConfig, error) {
	cfg, err := s.integrationService.GetRuntimeConfig(
		ctx,
		tenantInfo,
		integration.TypeEIAFuelPrices,
	)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.Config["baseUrl"]), "/")
	if baseURL == "" {
		baseURL = defaultEIABaseURL
	}

	return &eiaRuntimeConfig{
		APIKey:  strings.TrimSpace(cfg.Config["apiKey"]),
		BaseURL: baseURL,
	}, nil
}

type eiaResponse struct {
	Response struct {
		Data []struct {
			Period string `json:"period"`
			Series string `json:"series"`
			Value  string `json:"value"`
			Units  string `json:"units"`
		} `json:"data"`
	} `json:"response"`
	Error string `json:"error"`
}

func currentPriceMonday(now time.Time) time.Time {
	now = now.UTC()
	daysSinceMonday := (int(now.Weekday()) - int(time.Monday) + daysPerWeek) % daysPerWeek
	monday := now.AddDate(0, 0, -daysSinceMonday)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *Service) RefreshEIAPrices(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*RefreshEIAPricesResult, error) {
	log := s.l.With(
		zap.String("operation", "RefreshEIAPrices"),
		zap.String("orgId", tenantInfo.OrgID.String()),
	)

	indices, err := s.indexRepo.ListActiveEIA(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	if len(indices) == 0 {
		seriesIDs := make([]string, 0, len(fuelsurcharge.EIASeriesRegistry()))
		for _, def := range fuelsurcharge.EIASeriesRegistry() {
			seriesIDs = append(seriesIDs, def.SeriesID)
		}

		if err = EnsureEIAIndices(ctx, s.indexRepo, tenantInfo, seriesIDs); err != nil {
			return nil, err
		}

		if indices, err = s.indexRepo.ListActiveEIA(ctx, tenantInfo); err != nil {
			return nil, err
		}

		if len(indices) == 0 {
			return &RefreshEIAPricesResult{Skipped: true}, nil
		}

		log.Info("provisioned EIA fuel indices for tenant", zap.Int("count", len(indices)))
	}

	currentMonday := currentPriceMonday(time.Unix(s.now(), 0)).
		Format(fuelsurcharge.PriceDateLayout)

	allCurrent := true
	for _, index := range indices {
		has, hErr := s.priceRepo.HasPriceForDate(ctx, &repositories.HasFuelPriceForDateRequest{
			FuelIndexID: index.ID,
			TenantInfo:  tenantInfo,
			Date:        currentMonday,
		})
		if hErr != nil {
			return nil, hErr
		}
		if !has {
			allCurrent = false
			break
		}
	}

	if allCurrent {
		return &RefreshEIAPricesResult{Skipped: true, SeriesCount: len(indices)}, nil
	}

	cfg, err := s.eiaConfig(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("EIA fuel prices integration is not configured")
	}

	indexBySeries := make(map[string]*fuelsurcharge.FuelIndex, len(indices))
	for _, index := range indices {
		indexBySeries[index.EIASeriesID] = index
	}

	response, err := s.fetchEIAPrices(ctx, cfg, indices)
	if err != nil {
		return nil, err
	}

	prices := make([]*fuelsurcharge.FuelIndexPrice, 0, len(response.Response.Data))
	for _, row := range response.Response.Data {
		index, ok := indexBySeries[row.Series]
		if !ok {
			continue
		}

		value, vErr := decimal.NewFromString(row.Value)
		if vErr != nil || value.LessThanOrEqual(decimal.Zero) {
			log.Warn("skipping unparseable EIA price value",
				zap.String("series", row.Series),
				zap.String("period", row.Period),
				zap.String("value", row.Value),
			)
			continue
		}

		prices = append(prices, &fuelsurcharge.FuelIndexPrice{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			FuelIndexID:    index.ID,
			PriceDate:      row.Period,
			Price:          value,
			Currency:       index.Currency,
			SourceRaw:      row.Value,
		})
	}

	newRows, err := s.priceRepo.UpsertPrices(ctx, prices)
	if err != nil {
		return nil, err
	}

	log.Info("EIA fuel prices refreshed",
		zap.Int("seriesCount", len(indices)),
		zap.Int("rowsFetched", len(prices)),
		zap.Int("newRows", newRows),
	)

	if newRows > 0 && hasCurrentWeekRow(prices, currentMonday) {
		s.notifyPriceUpdate(ctx, tenantInfo, indices, currentMonday)
	}

	return &RefreshEIAPricesResult{
		SeriesCount: len(indices),
		NewRows:     newRows,
	}, nil
}

func hasCurrentWeekRow(prices []*fuelsurcharge.FuelIndexPrice, currentMonday string) bool {
	for _, price := range prices {
		if price.PriceDate == currentMonday {
			return true
		}
	}
	return false
}

func (s *Service) fetchEIAPrices(
	ctx context.Context,
	cfg *eiaRuntimeConfig,
	indices []*fuelsurcharge.FuelIndex,
) (*eiaResponse, error) {
	endpoint, err := url.Parse(cfg.BaseURL + eiaDieselPricePath)
	if err != nil {
		return nil, fmt.Errorf("invalid EIA base URL: %w", err)
	}

	query := endpoint.Query()
	query.Set("api_key", cfg.APIKey)
	query.Set("frequency", "weekly")
	query.Set("data[0]", "value")
	for _, index := range indices {
		query.Add("facets[series][]", index.EIASeriesID)
	}
	query.Set("sort[0][column]", "period")
	query.Set("sort[0][direction]", "desc")
	query.Set("length", strconv.Itoa(len(indices)*eiaTrailingWeeks))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create EIA request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call EIA API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read EIA response: %w", err)
	}

	var parsed eiaResponse
	if uErr := sonic.Unmarshal(body, &parsed); uErr != nil {
		return nil, fmt.Errorf("invalid EIA response: %w", uErr)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if parsed.Error != "" {
			return nil, fmt.Errorf("EIA returned status %d: %s", resp.StatusCode, parsed.Error)
		}
		return nil, fmt.Errorf("EIA returned status %d", resp.StatusCode)
	}

	return &parsed, nil
}

func (s *Service) notifyPriceUpdate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	indices []*fuelsurcharge.FuelIndex,
	currentMonday string,
) {
	log := s.l.With(zap.String("operation", "notifyPriceUpdate"))

	exists, err := s.notificationService.ExistsRecent(
		ctx,
		repositories.ExistsRecentNotificationRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			EventType:      fuelPriceEventType,
			CorrelationID:  currentMonday,
			Since:          s.now() - int64((daysPerWeek * 24 * time.Hour).Seconds()),
		},
	)
	if err != nil {
		log.Warn("failed to check for recent fuel price notification", zap.Error(err))
		return
	}
	if exists {
		return
	}

	title, message := s.buildPriceUpdateMessage(ctx, tenantInfo, indices, currentMonday)

	correlationID := currentMonday
	buID := tenantInfo.BuID
	if _, err = s.notificationService.Create(ctx, &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		EventType:      fuelPriceEventType,
		Priority:       notification.PriorityMedium,
		Channel:        notification.ChannelGlobal,
		Title:          title,
		Message:        message,
		Source:         fuelPriceNotifySource,
		CorrelationID:  &correlationID,
		Data: map[string]any{
			"priceDate": currentMonday,
		},
	}); err != nil {
		log.Warn("failed to create fuel price notification", zap.Error(err))
	}
}

func (s *Service) buildPriceUpdateMessage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	indices []*fuelsurcharge.FuelIndex,
	currentMonday string,
) (string, string) {
	title := "DOE diesel price updated"
	message := fmt.Sprintf(
		"Weekly DOE diesel prices for the week of %s are now available.",
		currentMonday,
	)

	var usIndex *fuelsurcharge.FuelIndex
	for _, index := range indices {
		if index.EIASeriesID == "EMD_EPD2D_PTE_NUS_DPG" {
			usIndex = index
			break
		}
	}
	if usIndex == nil {
		usIndex = indices[0]
	}

	prices, err := s.priceRepo.GetLatestOnOrBefore(ctx, &repositories.GetLatestFuelPricesRequest{
		FuelIndexID: usIndex.ID,
		TenantInfo:  tenantInfo,
		Date:        currentMonday,
		Limit:       2,
	})
	if err != nil || len(prices) == 0 {
		return title, message
	}

	latest := prices[0]
	if len(prices) > 1 {
		delta := latest.Price.Sub(prices[1].Price)
		direction := "▲"
		if delta.IsNegative() {
			direction = "▼"
		}
		message = fmt.Sprintf(
			"%s: $%s/gal (%s $%s vs last week). Fuel surcharge rates update on each program's effective day.",
			usIndex.Name,
			latest.Price.StringFixed(3),
			direction,
			delta.Abs().StringFixed(3),
		)
	} else {
		message = fmt.Sprintf(
			"%s: $%s/gal for the week of %s.",
			usIndex.Name,
			latest.Price.StringFixed(3),
			currentMonday,
		)
	}

	return title, message
}

func EnsureEIAIndices(
	ctx context.Context,
	repo repositories.FuelIndexRepository,
	tenantInfo pagination.TenantInfo,
	seriesIDs []string,
) error {
	existing, err := repo.ListActiveEIA(ctx, tenantInfo)
	if err != nil {
		return err
	}

	have := make(map[string]struct{}, len(existing))
	for _, index := range existing {
		have[index.EIASeriesID] = struct{}{}
	}

	for _, seriesID := range seriesIDs {
		if _, ok := have[seriesID]; ok {
			continue
		}

		def, found := fuelsurcharge.EIASeriesByID(seriesID)
		if !found {
			continue
		}

		if _, cErr := repo.Create(ctx, &fuelsurcharge.FuelIndex{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Name:           def.Name,
			Code:           def.Code,
			Source:         fuelsurcharge.IndexSourceEIA,
			FuelType:       def.FuelType,
			Region:         def.Region,
			EIASeriesID:    def.SeriesID,
			Currency:       "USD",
			IsActive:       true,
		}); cErr != nil {
			return cErr
		}
	}

	return nil
}
