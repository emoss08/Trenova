package agenttoolpolicy

import "go.uber.org/fx"

var Module = fx.Module("agent-tool-policy",
	fx.Provide(NewCatalog),
	fx.Invoke(func(*Catalog) {}),
)
