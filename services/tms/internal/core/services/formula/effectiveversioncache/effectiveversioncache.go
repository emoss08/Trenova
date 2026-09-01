package effectiveversioncache

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/shared/pulid"
)

type contextKey struct{}

type Load func(ctx context.Context) ([]*formulatemplate.FormulaTemplateVersion, error)

type cache struct {
	mu      sync.Mutex
	entries map[pulid.ID][]*formulatemplate.FormulaTemplateVersion
}

func With(ctx context.Context) context.Context {
	if from(ctx) != nil {
		return ctx
	}

	return context.WithValue(ctx, contextKey{}, &cache{
		entries: make(map[pulid.ID][]*formulatemplate.FormulaTemplateVersion, 1),
	})
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
