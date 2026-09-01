package effectiveversioncache

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/shared/pulid"
)

type contextKey struct{}

type Load func(ctx context.Context) ([]*formulatemplate.FormulaTemplateVersion, error)

// LoadOne fetches a single snapshot. It may return nil, nil for "no such
// version"; that answer is memoized too, so a batch of shipments that all
// predate any snapshot asks the database once.
type LoadOne func(ctx context.Context) (*formulatemplate.FormulaTemplateVersion, error)

// latestActiveKey is the version-number slot reserved for "the newest Active
// snapshot"; real version numbers start at one.
const latestActiveKey int64 = 0

type singleKey struct {
	templateID    pulid.ID
	versionNumber int64
}

type cache struct {
	mu      sync.Mutex
	entries map[pulid.ID][]*formulatemplate.FormulaTemplateVersion
	singles map[singleKey]*formulatemplate.FormulaTemplateVersion
}

func With(ctx context.Context) context.Context {
	if from(ctx) != nil {
		return ctx
	}

	return context.WithValue(ctx, contextKey{}, &cache{
		entries: make(map[pulid.ID][]*formulatemplate.FormulaTemplateVersion, 1),
		singles: make(map[singleKey]*formulatemplate.FormulaTemplateVersion, 4),
	})
}

// GetVersion returns one numbered snapshot of a template, loading it once per
// context. A backtest over hundreds of shipments rated by the same handful of
// versions resolves each version a single time.
func GetVersion(
	ctx context.Context,
	templateID pulid.ID,
	versionNumber int64,
	load LoadOne,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return getSingle(ctx, singleKey{templateID: templateID, versionNumber: versionNumber}, load)
}

// GetLatestActive returns the newest snapshot the template took while Active,
// loading it once per context.
func GetLatestActive(
	ctx context.Context,
	templateID pulid.ID,
	load LoadOne,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return getSingle(ctx, singleKey{templateID: templateID, versionNumber: latestActiveKey}, load)
}

func getSingle(
	ctx context.Context,
	key singleKey,
	load LoadOne,
) (*formulatemplate.FormulaTemplateVersion, error) {
	c := from(ctx)
	if c == nil {
		return load(ctx)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if version, ok := c.singles[key]; ok {
		return version, nil
	}

	version, err := load(ctx)
	if err != nil {
		return nil, err
	}

	c.singles[key] = version

	return version, nil
}

func GetVersions(
	ctx context.Context,
	templateID pulid.ID,
	load Load,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	c := from(ctx)
	if c == nil {
		return load(ctx)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if versions, ok := c.entries[templateID]; ok {
		return versions, nil
	}

	versions, err := load(ctx)
	if err != nil {
		return nil, err
	}

	c.entries[templateID] = versions

	return versions, nil
}

func EffectiveAsOf(
	versions []*formulatemplate.FormulaTemplateVersion,
	asOf int64,
) *formulatemplate.FormulaTemplateVersion {
	var best *formulatemplate.FormulaTemplateVersion
	for _, version := range versions {
		if version == nil || version.EffectiveFrom == nil || *version.EffectiveFrom > asOf {
			continue
		}

		if best == nil ||
			*version.EffectiveFrom > *best.EffectiveFrom ||
			(*version.EffectiveFrom == *best.EffectiveFrom &&
				version.VersionNumber > best.VersionNumber) {
			best = version
		}
	}

	return best
}

func from(ctx context.Context) *cache {
	c, _ := ctx.Value(contextKey{}).(*cache)

	return c
}
