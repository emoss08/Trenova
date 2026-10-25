package resultcache

import (
	"time"

	"github.com/emoss08/keel"
)

const (
	SettingsTTL  = 5 * time.Minute
	AccessTTL    = 5 * time.Minute
	ReferenceTTL = 30 * time.Minute
	CatalogTTL   = time.Hour
)

func Settings(opts ...keel.CacheOption) []keel.CacheOption {
	return with(SettingsTTL, opts)
}

func Access(opts ...keel.CacheOption) []keel.CacheOption {
	return with(AccessTTL, opts)
}

func Reference(opts ...keel.CacheOption) []keel.CacheOption {
	return with(ReferenceTTL, opts)
}

func Catalog(opts ...keel.CacheOption) []keel.CacheOption {
	return with(CatalogTTL, opts)
}

func with(ttl time.Duration, opts []keel.CacheOption) []keel.CacheOption {
	return append([]keel.CacheOption{keel.TTL(ttl)}, opts...)
}
