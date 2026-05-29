package integrationservice

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/shared/pcmiler"
	sharedsamsara "github.com/emoss08/trenova/shared/samsara"
	"github.com/emoss08/trenova/shared/samsara/drivers"
)

type connectionTester interface {
	Test(ctx context.Context, config map[string]string) error
}

var connectionTesters = map[integration.Type]connectionTester{
	integration.TypeSamsara:            &samsaraConnectionTester{},
	integration.TypeOANDAExchangeRates: &oandaExchangeRatesConnectionTester{},
	integration.TypePCMiler:            &pcmilerConnectionTester{},
}

type oandaExchangeRatesConnectionTester struct{}

func (t *oandaExchangeRatesConnectionTester) Test(ctx context.Context, cfg map[string]string) error {
	apiKey := cfg["apiKey"]
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg["baseUrl"]), "/")
	if baseURL == "" {
		baseURL = "https://exchange-rates-api.oanda.com"
	}

	endpoint, err := url.Parse(baseURL + "/v2/rates/spot.json")
	if err != nil {
		return fmt.Errorf("invalid OANDA base URL: %w", err)
	}
	query := endpoint.Query()
	query.Add("base", "USD")
	query.Add("quote", "EUR")
	query.Set("source_date", "true")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to OANDA Exchange Rates: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var result struct {
			Message string `json:"message"`
		}
		if err = sonic.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("OANDA returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("OANDA returned status %d: %s", resp.StatusCode, result.Message)
	}

	var result struct {
		Quotes []struct {
			BaseCurrency  string `json:"base_currency"`
			QuoteCurrency string `json:"quote_currency"`
			Bid           string `json:"bid"`
			Ask           string `json:"ask"`
			Midpoint      string `json:"midpoint"`
		} `json:"quotes"`
	}
	if err := sonic.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("invalid API response: %w", err)
	}

	if len(result.Quotes) == 0 {
		return fmt.Errorf("OANDA returned no USD/EUR quote")
	}

	return nil
}

type samsaraConnectionTester struct{}

func (t *samsaraConnectionTester) Test(ctx context.Context, cfg map[string]string) error {
	client, err := sharedsamsara.New(
		cfg["token"],
		sharedsamsara.WithBaseURL(cfg["baseUrl"]),
		sharedsamsara.WithTimeout(15*time.Second),
	)
	if err != nil {
		return err
	}

	_, err = client.Drivers.List(ctx, drivers.ListParams{Limit: 1})
	return err
}

type pcmilerConnectionTester struct{}

func (t *pcmilerConnectionTester) Test(ctx context.Context, cfg map[string]string) error {
	client, err := pcmiler.New(pcmiler.Config{
		APIKey:  cfg["apiKey"],
		BaseURL: cfg["baseUrl"],
	})
	if err != nil {
		return err
	}

	options := pcmiler.RouteOptions{
		DataVersion:         "Current",
		Region:              "NA",
		RoutingType:         "Practical",
		DistanceUnits:       "Miles",
		VehicleType:         "Truck",
		LocationGranularity: "PostalCode",
		TollRoads:           true,
		BordersOpen:         true,
	}
	if err = testPCMilerMileage(ctx, client, options); err != nil {
		return err
	}

	return nil
}

func testPCMilerMileage(
	ctx context.Context,
	client *pcmiler.Client,
	options pcmiler.RouteOptions,
) error {
	_, err := client.Mileage(ctx, []pcmiler.RouteRequest{
		{
			RouteID: "connection-test",
			Stops: []pcmiler.Stop{
				{City: "Princeton", State: "NJ", PostalCode: "08540"},
				{City: "Philadelphia", State: "PA", PostalCode: "19104"},
			},
			Options: options,
		},
	})
	return err
}
