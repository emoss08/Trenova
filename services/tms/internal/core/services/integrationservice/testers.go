package integrationservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	sharedsamsara "github.com/emoss08/trenova/shared/samsara"
	"github.com/emoss08/trenova/shared/samsara/drivers"
)

type connectionTester interface {
	Test(ctx context.Context, config map[string]string) error
}

var connectionTesters = map[integration.Type]connectionTester{
	integration.TypeSamsara:        &samsaraConnectionTester{},
	integration.TypeExchangeRateAPI: &exchangeRateAPIConnectionTester{},
}

type exchangeRateAPIConnectionTester struct{}

func (t *exchangeRateAPIConnectionTester) Test(ctx context.Context, cfg map[string]string) error {
	apiKey := cfg["apiKey"]
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	url := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/latest/USD", apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to ExchangeRate-API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Result    string `json:"result"`
		ErrorType string `json:"error-type,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("invalid API response: %w", err)
	}

	if result.Result != "success" {
		return fmt.Errorf("ExchangeRate-API error: %s", result.ErrorType)
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
