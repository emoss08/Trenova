package accesstest

import (
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimecontext"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"go.uber.org/zap"
)

func New(p *toolaccess.AccessParams) *toolaccess.Access {
	filled := *p
	if filled.Logger == nil {
		filled.Logger = zap.NewNop()
	}
	if filled.QueryTools == nil {
		filled.QueryTools = &agenttest.StubQueryRegistry{}
	}
	if filled.ActionTools == nil {
		filled.ActionTools = &agenttest.StubActionRegistry{}
	}
	if filled.Catalog == nil {
		filled.Catalog = agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
			QueryTools:  filled.QueryTools,
			ActionTools: filled.ActionTools,
		})
	}
	if filled.Extensions == nil {
		filled.Extensions = runtimetest.NoExtensions{}
	}
	if filled.Vectorizer == nil {
		filled.Vectorizer = runtimetest.NoVectors{}
	}
	if filled.CatalogVectors == nil {
		filled.CatalogVectors = runtimetest.NoVectors{}
	}
	if filled.Definitions == nil {
		filled.Definitions = &runtimetest.ListedDefinitions{}
	}
	if filled.Disclosures == nil {
		filled.Disclosures = toolaccess.NewIgnoredDisclosures()
	}
	if filled.Scripts == nil {
		filled.Scripts = runtimetest.NoScripts{}
	}
	if filled.Registry == nil {
		filled.Registry = permissioncatalog.NewRegistry()
	}

	return toolaccess.NewAccess(&filled)
}

//nolint:gocritic // mirrors runtimecontext.NewBuilder, which fx calls by value
func ContextBuilder(
	p runtimecontext.BuilderParams,
) agentruntimeports.RuntimeContextBuilder {
	if p.Memories == nil {
		p.Memories = runtimetest.UnusedMemories{}
	}
	if p.Guide == nil {
		p.Guide = runtimetest.NoGuide{}
	}
	if p.Definitions == nil {
		p.Definitions = &runtimetest.ListedDefinitions{}
	}
	if p.Permissions == nil {
		p.Permissions = &agenttest.StubPermissions{}
	}
	if p.Vectorizer == nil {
		p.Vectorizer = runtimetest.NoVectors{}
	}
	if p.Profiles == nil {
		p.Profiles = runtimetest.NoProfiles{}
	}

	return runtimecontext.NewBuilder(p)
}
