package ratetablecache

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/lrucache"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	processTenants  = 512
	processVersions = 2048
	processCells    = 4096
	entryTTL        = 30 * time.Minute
	keySeparator    = "\x1f"
)

type contextKey struct{}

type VersionKey struct {
	MatrixID pulid.ID
	Version  int64
}

type Axis struct {
	Position      int16
	Kind          string
	Exact         bool
	Postal        bool
	Normalization string
	Forward       map[string]string
	Reverse       map[string][]string
	Ranges        func(key string) []string
}

func (a *Axis) Keyed() bool {
	return a.Exact || a.Postal
}

type Table struct {
	Key  VersionKey
	Code string
	Axes []Axis
}

type Version struct {
	Shape rules.TariffTable
	Index Table
}

type Served struct {
	Versions []*Version
}

type Stamp func(ctx context.Context) (string, error)

type BuildServed func(ctx context.Context, stamp string) (*Served, error)

type LoadVersions func(ctx context.Context, keys []VersionKey) ([]*Version, error)

type tenantKey struct {
	orgID pulid.ID
	buID  pulid.ID
}

type served struct {
	stamp  string
	served *Served
}

type memo struct {
	mu      sync.Mutex
	entries map[tenantKey]served
}

type versionKey struct {
	tenant tenantKey
	key    VersionKey
}

type cellKey struct {
	tenant tenantKey
	key    VersionKey
	keys   string
}

var process = struct {
	mu       sync.Mutex
	served   lrucache.Cache[tenantKey, served]
	versions lrucache.Cache[versionKey, *Version]
	cells    lrucache.Cache[cellKey, []rules.TariffCell]
}{
	served:   lrucache.NewExpirable[tenantKey, served](processTenants, entryTTL),
	versions: lrucache.NewExpirable[versionKey, *Version](processVersions, entryTTL),
	cells:    lrucache.NewExpirable[cellKey, []rules.TariffCell](processCells, entryTTL),
}

func With(ctx context.Context) context.Context {
	if from(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, &memo{entries: make(map[tenantKey]served, 1)})
}

func from(ctx context.Context) *memo {
	m, _ := ctx.Value(contextKey{}).(*memo)
	return m
}

func ServedTables(
	ctx context.Context,
	orgID, buID pulid.ID,
	stamp Stamp,
	build BuildServed,
) (string, *Served, error) {
	key := tenantKey{orgID: orgID, buID: buID}
	m := from(ctx)
	if m != nil {
		m.mu.Lock()
		entry, ok := m.entries[key]
		m.mu.Unlock()
		if ok {
			return entry.stamp, entry.served, nil
		}
	}

	current, err := stamp(ctx)
	if err != nil {
		return "", nil, err
	}

	process.mu.Lock()
	entry, ok := process.served.Get(key)
	process.mu.Unlock()
	if !ok || entry.stamp != current || current == "" {
		built, buildErr := build(ctx, current)
		if buildErr != nil {
			return "", nil, buildErr
		}
		entry = served{stamp: current, served: built}
		if current != "" {
			process.mu.Lock()
			process.served.Add(key, entry)
			process.mu.Unlock()
		}
	}

	if m != nil {
		m.mu.Lock()
		m.entries[key] = entry
		m.mu.Unlock()
	}

	return entry.stamp, entry.served, nil
}

func Versions(
	ctx context.Context,
	orgID, buID pulid.ID,
	keys []VersionKey,
	load LoadVersions,
) (map[VersionKey]*Version, error) {
	tenant := tenantKey{orgID: orgID, buID: buID}
	found := make(map[VersionKey]*Version, len(keys))
	missing := make([]VersionKey, 0, len(keys))

	process.mu.Lock()
	for _, key := range keys {
		if _, seen := found[key]; seen {
			continue
		}
		if version, ok := process.versions.Get(versionKey{tenant: tenant, key: key}); ok {
			found[key] = version
			continue
		}
		missing = append(missing, key)
	}
	process.mu.Unlock()

	if len(missing) == 0 {
		return found, nil
	}

	loaded, err := load(ctx, missing)
	if err != nil {
		return nil, err
	}

	process.mu.Lock()
	defer process.mu.Unlock()
	for _, version := range loaded {
		if version == nil {
			continue
		}
		found[version.Index.Key] = version
		process.versions.Add(versionKey{tenant: tenant, key: version.Index.Key}, version)
	}
	return found, nil
}

func Cells(orgID, buID pulid.ID, key VersionKey, keys []string) ([]rules.TariffCell, bool) {
	process.mu.Lock()
	defer process.mu.Unlock()
	return process.cells.Get(cellKeyFor(orgID, buID, key, keys))
}

func StoreCells(orgID, buID pulid.ID, key VersionKey, keys []string, cells []rules.TariffCell) {
	process.mu.Lock()
	defer process.mu.Unlock()
	process.cells.Add(cellKeyFor(orgID, buID, key, keys), cells)
}

func cellKeyFor(orgID, buID pulid.ID, key VersionKey, keys []string) cellKey {
	return cellKey{
		tenant: tenantKey{orgID: orgID, buID: buID},
		key:    key,
		keys:   strings.Join(keys, keySeparator),
	}
}

func Invalidate(orgID, buID pulid.ID) {
	process.mu.Lock()
	defer process.mu.Unlock()
	process.served.Remove(tenantKey{orgID: orgID, buID: buID})
}
