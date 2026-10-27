package tariff

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/pkg/rules"
)

type CatalogEntry struct {
	ModuleVersion string
	Catalog       rules.RawJSON
}

type catalogCache struct {
	mu     sync.Mutex
	module moduleCalls
	entry  *CatalogEntry
}

func newCatalogCache(module moduleCalls) *catalogCache {
	return &catalogCache{module: module}
}

func (c *catalogCache) get(ctx context.Context) (*CatalogEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	info, err := c.module.Version(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the rules module version: %w", err)
	}
	if c.entry != nil && c.entry.ModuleVersion == info.Version {
		return c.entry, nil
	}

	catalog, err := c.module.TariffCatalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the tariff catalog: %w", err)
	}

	c.entry = &CatalogEntry{ModuleVersion: info.Version, Catalog: catalog}
	return c.entry, nil
}

func (c *Checker) Catalog(ctx context.Context) (*CatalogEntry, error) {
	return c.catalog.get(ctx)
}
