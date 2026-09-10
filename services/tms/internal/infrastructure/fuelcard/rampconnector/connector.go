package rampconnector

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/fuelimport"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	defaultBaseURL = "https://api.ramp.com/developer/v1"
	requestTimeout = 30 * time.Second
	pageSize       = 100
	maxPages       = 50
)

// Headers are chosen to match the Ramp entries in the fuel import header
// dictionary, so an API row and a Ramp CSV export resolve through exactly the
// same mapping.
var sheetHeaders = []string{
	"User Transaction Time",
	"Merchant Name",
	"Merchant City",
	"Merchant State",
	"Amount",
	"Currency Code",
	"Card Last Four",
	"ID",
	"Card Holder Name",
	// Ramp never sends gallons, but the column has to exist. Staging refuses a
	// sheet with no quantity column at all, which would drop the whole window on
	// the floor; an empty column instead fails each row on its own, so every
	// transaction lands in the review queue saying what it is missing.
	"Gallons",
}

// Connector reads card transactions from Ramp.
//
// Ramp is a commercial Visa product rather than a fleet network: an
// authorization carries the merchant and the amount, and there is no Level III
// pump detail behind it. That is why this connector reports SpendOnly. Rows it
// produces have no gallons and no fuel type, so they cannot become tax records
// on their own and wait in the review queue for somebody to complete them.
type Connector struct {
	client *http.Client
}

func New() *Connector {
	return NewWithClient(&http.Client{Timeout: requestTimeout})
}

func NewWithClient(client *http.Client) *Connector {
	return &Connector{client: client}
}

func (c *Connector) IntegrationType() integration.Type { return integration.TypeRampFuel }

func (c *Connector) Capability() services.FuelFeedCapability {
	return services.FuelFeedCapabilitySpendOnly
}

func (c *Connector) TestConnection(ctx context.Context, config map[string]string) error {
	settings, err := settingsFrom(config)
	if err != nil {
		return err
	}

	_, err = c.token(ctx, settings)

	return err
}

func (c *Connector) FetchTransactions(
	ctx context.Context,
	req *services.FetchFuelTransactionsRequest,
) (*services.FetchFuelTransactionsResult, error) {
	settings, err := settingsFrom(req.Config)
	if err != nil {
		return nil, err
	}

	token, err := c.token(ctx, settings)
	if err != nil {
		return nil, err
	}

	transactions, err := c.transactions(ctx, settings, token, req.Since, req.Until)
	if err != nil {
		return nil, err
	}

	if len(transactions) == 0 {
		return &services.FetchFuelTransactionsResult{
			Staged: &fuelimport.StageResult{},
			Format: fuelpurchase.SourceFormatAPI,
		}, nil
	}

	rows := make([][]string, 0, len(transactions))
	for _, transaction := range transactions {
		rows = append(rows, transaction.cells())
	}

	staged := fuelimport.Stage(&rateimport.Sheet{
		Headers:      sheetHeaders,
		Rows:         rows,
		FirstDataRow: 1,
	}, fuelimport.StageOptions{
		Provider:        fuelpurchase.CardProviderRamp,
		DefaultFuelType: settings.defaultFuelType,
		DefaultCurrency: settings.defaultCurrency,
	})

	return &services.FetchFuelTransactionsResult{
		Staged:    staged,
		Reference: windowReference(req.Since, req.Until),
		Format:    fuelpurchase.SourceFormatAPI,
		Cards:     cardsFrom(transactions),
	}, nil
}

func cardsFrom(transactions []transaction) []services.ProviderCard {
	seen := make(map[string]struct{}, 8)
	cards := make([]services.ProviderCard, 0, 8)

	for _, item := range transactions {
		lastFour := item.lastFour()
		if lastFour == "" {
			continue
		}
		if _, ok := seen[lastFour]; ok {
			continue
		}

		seen[lastFour] = struct{}{}
		cards = append(cards, services.ProviderCard{
			LastFour:       lastFour,
			ExternalCardID: item.CardID,
			Provider:       fuelpurchase.CardProviderRamp,
			Label:          strings.TrimSpace("Ramp " + lastFour),
		})
	}

	return cards
}

func windowReference(since, until int64) string {
	return "ramp:" + strconv.FormatInt(since, 10) + "-" + strconv.FormatInt(until, 10)
}

type settings struct {
	baseURL         string
	clientID        string
	clientSecret    string
	defaultFuelType domaintypes.IFTAFuelType
	defaultCurrency string
}

func settingsFrom(config map[string]string) (*settings, error) {
	clientID := strings.TrimSpace(config[integration.ConfigKeyFuelClientID])
	clientSecret := strings.TrimSpace(config[integration.ConfigKeyFuelClientSecret])

	switch {
	case clientID == "":
		return nil, errors.New("a Ramp client ID is required")
	case clientSecret == "":
		return nil, errors.New("a Ramp client secret is required")
	}

	resolved := &settings{
		baseURL: strings.TrimRight(
			stringutils.WithDefault(config[integration.ConfigKeyFuelBaseURL], defaultBaseURL),
			"/",
		),
		clientID:        clientID,
		clientSecret:    clientSecret,
		defaultCurrency: stringutils.WithDefault(config[integration.ConfigKeyFuelCurrency], "USD"),
	}

	// Ramp never names a product, so without a fuel type to fall back on the
	// staging pipeline rejects the sheet outright and the whole window is lost.
	// Refusing here says why, instead of reporting a feed that quietly reads
	// nothing every hour.
	fuelType := domaintypes.IFTAFuelType(
		strings.TrimSpace(config[integration.ConfigKeyFuelDefaultFuelType]),
	)
	if !fuelType.IsValid() {
		return nil, errors.New(
			"a default fuel type is required, because Ramp transactions do not name a product",
		)
	}
	resolved.defaultFuelType = fuelType

	return resolved, nil
}
