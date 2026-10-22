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
	processTenants = 512
	processCells   = 4096
	cellTTL        = 30 * time.Minute
	keySeparator   = "\x1f"
)

type contextKey struct{}

type Stamp func(ctx context.Context) (string, error)

type Axis struct {
	Position      int16
	Kind          string
	Exact         bool
	Normalization string
	Forward       map[string]string
	Reverse       map[string][]string
}

type Table struct {
	MatrixID pulid.ID
	Code     string
	Axes     []Axis
}

type Entry struct {
	Tables []rules.TariffTable
	Index  []Table
}

func (e *Entry) Table(code string) (*Table, int, bool) {
	for i := range e.Index {
		if e.Index[i].Code == code {
			return &e.Index[i], i, true
		}
	}
	return nil, -1, false
}

type Build func(ctx context.Context, stamp string) (*Entry, error)

type tenantKey struct {
	orgID pulid.ID
	buID  pulid.ID
}

type shapes struct {
	stamp string
	entry *Entry
}

type memo struct {
	mu      sync.Mutex
	entries map[tenantKey]shapes
}

type cellKey struct {
	tenant     tenantKey
	generation uint64
	stamp      string
	table  string
	keys   string
}

var process = struct {
	mu          sync.Mutex
	generations map[tenantKey]uint64
	shapes      lrucache.Cache[tenantKey, shapes]
	cells       lrucache.Cache[cellKey, []rules.TariffCell]
}{
	generations: make(map[tenantKey]uint64, processTenants),
	shapes:      lrucache.NewExpirable[tenantKey, shapes](processTenants, cellTTL),
	cells:       lrucache.NewExpirable[cellKey, []rules.TariffCell](processCells, cellTTL),
}

func keyFor(orgID, buID pulid.ID, stamp, table string, keys []string) cellKey {
	tenant := tenantKey{orgID: orgID, buID: buID}
	return cellKey{
		tenant:     tenant,
		generation: process.generations[tenant],
		stamp:      stamp,
		table:      table,
		keys:       strings.Join(keys, keySeparator),
	}
}

func With(ctx context.Context) context.Context {
	if from(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, &memo{entries: make(map[tenantKey]shapes, 1)})
}

func from(ctx context.Context) *memo {
	m, _ := ctx.Value(contextKey{}).(*memo)
	return m
}

func Tables(
	ctx context.Context,
	orgID, buID pulid.ID,
	stamp Stamp,
	build Build,
) (string, *Entry, error) {
	key := tenantKey{orgID: orgID, buID: buID}
	m := from(ctx)
	if m != nil {
		m.mu.Lock()
		entry, ok := m.entries[key]
		m.mu.Unlock()
		if ok {
			return entry.stamp, entry.entry, nil
		}
	}

	current, err := stamp(ctx)
	if err != nil {
		return "", nil, err
	}

	process.mu.Lock()
	entry, ok := process.shapes.Get(key)
	process.mu.Unlock()
	if !ok || entry.stamp != current || current == "" {
		built, buildErr := build(ctx, current)
		if buildErr != nil {
			return "", nil, buildErr
		}
		entry = shapes{stamp: current, entry: built}
		if current != "" {
			process.mu.Lock()
			process.shapes.Add(key, entry)
			process.mu.Unlock()
		}
	}

	if m != nil {
		m.mu.Lock()
		m.entries[key] = entry
		m.mu.Unlock()
	}

	return entry.stamp, entry.entry, nil
}

func Cells(orgID, buID pulid.ID, stamp, table string, keys []string) ([]rules.TariffCell, bool) {
	if stamp == "" {
		return nil, false
	}
	process.mu.Lock()
	defer process.mu.Unlock()
	return process.cells.Get(keyFor(orgID, buID, stamp, table, keys))
}

func StoreCells(orgID, buID pulid.ID, stamp, table string, keys []string, cells []rules.TariffCell) {
	if stamp == "" {
		return
	}
	process.mu.Lock()
	defer process.mu.Unlock()
	process.cells.Add(keyFor(orgID, buID, stamp, table, keys), cells)
}

func Invalidate(orgID, buID pulid.ID) {
	process.mu.Lock()
	defer process.mu.Unlock()
	tenant := tenantKey{orgID: orgID, buID: buID}
	process.shapes.Remove(tenant)
	process.generations[tenant]++
}
