package aiproviderservice

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	openRouterCatalogURL = "https://openrouter.ai/api/v1/models"
	openRouterSource     = "openrouter"
	// priceCatalogTimeout bounds one read of the catalog, apart from the model
	// list that asked for it, so a slow reference never holds a list up for long.
	priceCatalogTimeout = 15 * time.Second
	// priceCatalogRetryGap is how long a failed read is remembered, so a
	// reference that is down is not asked again on every model list.
	priceCatalogRetryGap = 10 * time.Minute
	// priceCatalogSharedRefresh caps how long a copy read from the shared store
	// is held in memory, since how much of its lifetime is left is not known.
	priceCatalogSharedRefresh = time.Hour
)

// datedSuffix is a release date a provider appends to a model ID, which a
// catalog that names the model without it would otherwise miss.
var datedSuffix = regexp.MustCompile(`-(\d{8}|\d{4}-\d{2}-\d{2})$`)

type ModelPriceReferenceParams struct {
	fx.In

	Logger *zap.Logger
	Config *config.Config
	Cache  repositories.ModelPriceCacheRepository `optional:"true"`
}

// openRouterPrices estimates a model's price from OpenRouter's public catalog.
// The catalog is read without credentials, once per refresh for every replica
// and organization: in memory first, then the shared store, and only then from
// OpenRouter, with one read in flight however many lists ask at once.
type openRouterPrices struct {
	l      *zap.Logger
	cfg    *config.AIConfig
	cache  repositories.ModelPriceCacheRepository
	client *http.Client
	group  singleflight.Group

	mu      sync.RWMutex
	index   *priceIndex
	expires time.Time
}

func NewModelPriceReference(p ModelPriceReferenceParams) services.ModelPriceReference {
	return &openRouterPrices{
		l:      p.Logger.Named("service.aiprovider.price-reference"),
		cfg:    p.Config.GetAIConfig(),
		cache:  p.Cache,
		client: &http.Client{Timeout: priceCatalogTimeout},
	}
}

func (r *openRouterPrices) Prices(
	ctx context.Context,
	kind aiprovider.Kind,
	modelIDs []string,
) map[string]services.ReferencePrice {
	if !r.cfg.ModelPriceReferenceEnabled() || kind == aiprovider.KindOllama || len(modelIDs) == 0 {
		return nil
	}

	index := r.catalog(ctx)
	if index == nil {
		return nil
	}

	prices := make(map[string]services.ReferencePrice, len(modelIDs))
	for _, id := range modelIDs {
		if price, ok := index.lookup(kind, id); ok {
			prices[id] = price
		}
	}

	return prices
}

func (r *openRouterPrices) catalog(ctx context.Context) *priceIndex {
	r.mu.RLock()
	index, fresh := r.index, time.Now().Before(r.expires)
	r.mu.RUnlock()
	if fresh {
		return index
	}

	loaded, _, _ := r.group.Do(openRouterSource, func() (any, error) {
		return r.load(context.WithoutCancel(ctx)), nil
	})

	result, _ := loaded.(*priceIndex)
	return result
}

func (r *openRouterPrices) load(ctx context.Context) *priceIndex {
	ctx, cancel := context.WithTimeout(ctx, priceCatalogTimeout)
	defer cancel()

	ttl := r.cfg.GetModelPriceReferenceTTL()
	if shared := r.readShared(ctx); shared != nil {
		index := newPriceIndex(shared)
		r.remember(index, min(ttl, priceCatalogSharedRefresh))
		return index
	}

	models, err := modeladapter.ListReferencePrices(ctx, r.client, openRouterCatalogURL)
	if err != nil {
		r.l.Warn("could not read the model price reference", zap.Error(err))
		r.remember(nil, priceCatalogRetryGap)
		return nil
	}

	prices := make(map[string]repositories.CachedModelPrice, len(models))
	for idx := range models {
		model := &models[idx]
		price := repositories.CachedModelPrice{Input: model.InputCostPerMillion.String()}
		if model.OutputCostPerMillion != nil {
			price.Output = model.OutputCostPerMillion.String()
		}
		prices[model.ID] = price
	}

	if r.cache != nil {
		if err = r.cache.Set(ctx, openRouterSource, prices, ttl); err != nil {
			r.l.Debug("could not share the model price reference", zap.Error(err))
		}
	}

	index := newPriceIndex(prices)
	r.remember(index, ttl)
	return index
}

func (r *openRouterPrices) readShared(ctx context.Context) map[string]repositories.CachedModelPrice {
	if r.cache == nil {
		return nil
	}

	prices, err := r.cache.Get(ctx, openRouterSource)
	if err != nil {
		r.l.Debug("could not read the shared model price reference", zap.Error(err))
		return nil
	}

	return prices
}

func (r *openRouterPrices) remember(index *priceIndex, ttl time.Duration) {
	r.mu.Lock()
	r.index = index
	r.expires = time.Now().Add(ttl)
	r.mu.Unlock()
}

// priceIndex finds a catalog price for a model ID as a provider lists it, which
// rarely matches the catalog's spelling exactly.
type priceIndex struct {
	byID   map[string]services.ReferencePrice
	byName map[string][]services.ReferencePrice
}

func newPriceIndex(prices map[string]repositories.CachedModelPrice) *priceIndex {
	index := &priceIndex{
		byID:   make(map[string]services.ReferencePrice, len(prices)),
		byName: make(map[string][]services.ReferencePrice, len(prices)),
	}

	for id, cached := range prices {
		if strings.Contains(id, ":") {
			continue
		}
		input, err := decimal.NewFromString(cached.Input)
		if err != nil || !input.IsPositive() {
			continue
		}
		price := services.ReferencePrice{InputCostPerMillion: input}
		if output, outputErr := decimal.NewFromString(cached.Output); outputErr == nil &&
			output.IsPositive() {
			price.OutputCostPerMillion = &output
		}

		key := normalizeModelID(id)
		index.byID[key] = price
		name := key[strings.LastIndex(key, "/")+1:]
		index.byName[name] = append(index.byName[name], price)
	}

	return index
}

func (x *priceIndex) lookup(kind aiprovider.Kind, modelID string) (services.ReferencePrice, bool) {
	key := normalizeModelID(modelID)
	undated := datedSuffix.ReplaceAllString(key, "")
	vendor := catalogVendor(kind)

	for _, candidate := range []string{key, vendor + "/" + key, vendor + "/" + undated} {
		if price, ok := x.byID[candidate]; ok {
			return price, true
		}
	}

	name := undated[strings.LastIndex(undated, "/")+1:]
	if matches := x.byName[name]; len(matches) == 1 {
		return matches[0], true
	}

	return services.ReferencePrice{}, false
}

// catalogVendor is the prefix the catalog files a provider's own models under.
func catalogVendor(kind aiprovider.Kind) string {
	if kind == aiprovider.KindAnthropicMessages {
		return "anthropic"
	}

	return "openai"
}

// normalizeModelID makes the spellings a provider and a catalog use for the
// same model compare equal: case, and a version written with dots or dashes.
func normalizeModelID(id string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(id)), ".", "-")
}
