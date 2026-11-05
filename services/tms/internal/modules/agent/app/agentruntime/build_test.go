package agentruntime

import (
	"sync"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/accesstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/tooldispatch"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"go.uber.org/zap"
)

// runtimeParts is what a test runtime and its tool access are built from.
type runtimeParts struct {
	Logger         *zap.Logger
	Completion     llmports.CompletionService
	QueryTools     agenttoolports.AgentQueryToolRegistry
	ActionTools    agenttoolports.AgentToolRegistry
	Permissions    permissionports.PermissionEngine
	Versions       agenttoolports.RecordVersionReader
	Budgets        agentbudgetports.AgentBudgetService
	Trust          agenttoolports.AgentToolTrustRepository
	Memories       agentmemoryports.AgentMemoryService
	Previews       agentruntimeports.ProposalPreviewService
	Rules          agenttoolports.ToolRuleOverrides
	Changes        realtimeports.RecordChangeFeed
	Metrics        *metrics.Registry
	AppConfig      *appconfig.Config
	Shadow         *agentshadow.Resolver
	Scripts        llmports.ScriptProviders
	Catalog        *agenttoolcatalog.Catalog
	Extensions     agentextensionports.AgentExtensionGate
	Vectorizer     retrievalports.QueryVectorizer
	CatalogVectors retrievalports.CatalogVectorIndex
	Definitions    agentdefinitionports.AgentDefinitionRepository
	AIConfig       *aiconfig.Config
	Disclosures    agenttoolports.ToolDisclosureObserver
	Registry       *permissioncatalog.Registry
}

func (p *runtimeParts) runtime() Params {
	built := Params{
		Logger:      p.Logger,
		Completion:  p.Completion,
		QueryTools:  p.QueryTools,
		ActionTools: p.ActionTools,
		Permissions: p.Permissions,
		Versions:    p.Versions,
		Budgets:     p.Budgets,
		Trust:       p.Trust,
		Memories:    p.Memories,
		Previews:    p.Previews,
		Rules:       p.Rules,
		Changes:     p.Changes,
		Metrics:     p.Metrics,
		AppConfig:   p.AppConfig,
		Shadow:      p.Shadow,
		Scripts:     p.Scripts,
	}
	if built.Memories == nil {
		built.Memories = runtimetest.UnusedMemories{}
	}
	if built.Changes == nil {
		built.Changes = runtimetest.NoChanges{}
	}
	if built.Scripts == nil {
		built.Scripts = runtimetest.NoScripts{}
	}

	return built
}

func (p *runtimeParts) access() toolaccess.AccessParams {
	return toolaccess.AccessParams{
		Logger:         p.Logger,
		QueryTools:     p.QueryTools,
		ActionTools:    p.ActionTools,
		Permissions:    p.Permissions,
		Catalog:        p.Catalog,
		Extensions:     p.Extensions,
		Vectorizer:     p.Vectorizer,
		CatalogVectors: p.CatalogVectors,
		Definitions:    p.Definitions,
		Threshold:      toolaccess.ConfiguredThreshold(p.AIConfig),
		Disclosures:    p.Disclosures,
		Scripts:        p.Scripts,
		Registry:       p.Registry,
	}
}

// runtimeBuild is what a test runtime was built from, kept so a test can
// change a part and build it again in place.
type runtimeBuild struct {
	parts runtimeParts
	tunes []func(*toolaccess.AccessParams, *tooldispatch.DispatcherParams)
}

var runtimeBuilds sync.Map

// buildRuntime builds a runtime as New does, remembering what it was built
// from.
func buildRuntime(p runtimeParts) *Service {
	if p.Logger == nil {
		p.Logger = zap.NewNop()
	}
	b := &runtimeBuild{parts: p}
	rt := b.build()
	runtimeBuilds.Store(rt, b)

	return rt
}

func (b *runtimeBuild) build() *Service {
	params := b.parts.runtime()
	accessParts, dispatchParts := b.parts.access(), dispatcherParams(&params)
	for _, tune := range b.tunes {
		tune(&accessParts, &dispatchParts)
	}

	return assemble(&params, accesstest.New(&accessParts), &dispatchParts)
}

func buildOf(rt *Service) *runtimeBuild {
	b, ok := runtimeBuilds.Load(rt)
	if !ok {
		panic("runtime was not built by buildRuntime")
	}

	return b.(*runtimeBuild)
}

// paramsOf is what a test runtime was built from.
func paramsOf(rt *Service) runtimeParts { return buildOf(rt).parts }

// rewire builds rt again, in place, with change made to its dependencies.
func rewire(rt *Service, change func(*runtimeParts)) {
	b := buildOf(rt)
	change(&b.parts)
	*rt = *b.build()
}

// retune builds rt again, in place, with change made to the parts New
// derives from its dependencies.
func retune(rt *Service, change func(*toolaccess.AccessParams, *tooldispatch.DispatcherParams)) {
	b := buildOf(rt)
	b.tunes = append(b.tunes, change)
	*rt = *b.build()
}
