package effectiveversioncache

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/shared/pulid"
)

type contextKey struct{}

type Load func(ctx context.Context) ([]*formulatemplate.FormulaTemplateVersion, error)

type LoadOne func(ctx context.Context) (*formulatemplate.FormulaTemplateVersion, error)

type LoadTemplate func(ctx context.Context) (*formulatemplate.FormulaTemplate, error)

type LoadLibraries func(ctx context.Context) ([]*formulalibrary.FormulaLibraryVersion, error)

type singleKey struct {
	templateID    pulid.ID
	versionNumber int64
}

type cache struct {
	mu        sync.Mutex
	entries   map[pulid.ID][]*formulatemplate.FormulaTemplateVersion
	singles   map[singleKey]*formulatemplate.FormulaTemplateVersion
	templates map[pulid.ID]*formulatemplate.FormulaTemplate
	libraries map[string][]*formulalibrary.FormulaLibraryVersion
}

func With(ctx context.Context) context.Context {
	if from(ctx) != nil {
		return ctx
	}

	return context.WithValue(ctx, contextKey{}, &cache{
		entries:   make(map[pulid.ID][]*formulatemplate.FormulaTemplateVersion, 1),
		singles:   make(map[singleKey]*formulatemplate.FormulaTemplateVersion, 4),
		templates: make(map[pulid.ID]*formulatemplate.FormulaTemplate, 1),
		libraries: make(map[string][]*formulalibrary.FormulaLibraryVersion, 1),
	})
}

func GetVersion(
	ctx context.Context,
	templateID pulid.ID,
	versionNumber int64,
	load LoadOne,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return getSingle(ctx, singleKey{templateID: templateID, versionNumber: versionNumber}, load)
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

func GetTemplate(
	ctx context.Context,
	templateID pulid.ID,
	load LoadTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	c := from(ctx)
	if c == nil {
		return load(ctx)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if template, ok := c.templates[templateID]; ok {
		return template, nil
	}

	template, err := load(ctx)
	if err != nil {
		return nil, err
	}

	c.templates[templateID] = template

	return template, nil
}

func GetLibraries(
	ctx context.Context,
	key string,
	load LoadLibraries,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	c := from(ctx)
	if c == nil {
		return load(ctx)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if libraries, ok := c.libraries[key]; ok {
		return libraries, nil
	}

	libraries, err := load(ctx)
	if err != nil {
		return nil, err
	}

	c.libraries[key] = libraries

	return libraries, nil
}

func EffectiveAsOf(
	approved []*formulatemplate.FormulaTemplateVersion,
	asOf, now int64,
) (inForce, scheduled *formulatemplate.FormulaTemplateVersion) {
	var earliest *formulatemplate.FormulaTemplateVersion
	for _, version := range approved {
		if version == nil {
			continue
		}

		start := version.EffectiveStart()
		if earliest == nil ||
			start < earliest.EffectiveStart() ||
			(start == earliest.EffectiveStart() && version.VersionNumber < earliest.VersionNumber) {
			earliest = version
		}

		if start > asOf {
			continue
		}
		if inForce == nil ||
			start > inForce.EffectiveStart() ||
			(start == inForce.EffectiveStart() && version.VersionNumber > inForce.VersionNumber) {
			inForce = version
		}
	}

	if inForce != nil {
		return inForce, nil
	}
	if earliest == nil || earliest.EffectiveStart() <= now {
		return earliest, nil
	}
	return nil, earliest
}

func from(ctx context.Context) *cache {
	c, _ := ctx.Value(contextKey{}).(*cache)

	return c
}
