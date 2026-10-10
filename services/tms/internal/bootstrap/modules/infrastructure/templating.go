package infrastructure

import (
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/infrastructure/templating/assetinliner"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/templateengine"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// TemplatingModule provides the pieces every template render shares.
//
// All three are singletons on purpose: the registry is immutable after
// construction, the engine owns a compilation cache that is worthless if each
// caller gets its own, and the inliner holds a pooled SSRF-safe HTTP client.
var TemplatingModule = fx.Module("templating", fx.Provide(
	documenttemplate.NewRegistry,
	newTemplateEngine,
	newAssetInliner,
))

// newTemplateEngine builds the engine on its defaults.
//
// The limits bound what an organization can author and produce — source size,
// output size, cache entries — and the defaults are sized for the documents this
// system actually sends. They are not configuration because raising them is a
// decision about what a tenant can make the renderer do, not a deployment knob.
func newTemplateEngine() (*templateengine.Engine, error) {
	return templateengine.New(templateengine.Config{})
}

type assetInlinerParams struct {
	fx.In

	Storage kernelports.Client
	Logger  *zap.Logger
}

func newAssetInliner(p assetInlinerParams) documenttemplateports.AssetInliner {
	return assetinliner.New(assetinliner.Params{
		Storage: p.Storage,
		Logger:  p.Logger,
	})
}
