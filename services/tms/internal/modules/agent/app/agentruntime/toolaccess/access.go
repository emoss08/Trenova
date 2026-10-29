package toolaccess

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"go.uber.org/zap"
)

type AccessParams struct {
	Logger      *zap.Logger
	QueryTools  agenttoolports.AgentQueryToolRegistry
	ActionTools agenttoolports.AgentToolRegistry
	Permissions permissionports.PermissionEngine
	Catalog     *agenttoolcatalog.Catalog
	// Extensions is optional. Without it no extension's tools are offered.
	Extensions     agentextensionports.AgentExtensionGate
	Vectorizer     retrievalports.QueryVectorizer
	CatalogVectors retrievalports.CatalogVectorIndex
	// Definitions is optional. With it a search for a tool the agent does not
	// hold names the person's other agents that hold it.
	Definitions agentdefinitionports.AgentDefinitionRepository
	// Threshold is how many tools a turn is given before the rest are found
	// on demand; nil keeps the default.
	Threshold   *int
	Disclosures agenttoolports.ToolDisclosureObserver
	// Scripts is optional. With it the code-mode gate reads the chat
	// providers' script settings; without it no model is offered scripts
	// unless the bench forces them.
	Scripts llmports.ScriptProviders
}

// Access decides which tools an agent holds and may call, builds a turn's
// tool set and answers its find_tools searches.
type Access struct {
	logger      *zap.Logger
	queryTools  agenttoolports.AgentQueryToolRegistry
	actionTools agenttoolports.AgentToolRegistry
	permissions permissionports.PermissionEngine
	definitions agentdefinitionports.AgentDefinitionRepository
	catalog     *agenttoolcatalog.Catalog
	extensions  agentextensionports.AgentExtensionGate
	vectorizer  retrievalports.QueryVectorizer
	vectors     retrievalports.CatalogVectorIndex
	disclosures agenttoolports.ToolDisclosureObserver
	threshold   *int
	registry    *permissioncatalog.Registry
	scripts     llmports.ScriptProviders
}

func NewAccess(p AccessParams) *Access {
	return &Access{
		logger:      p.Logger.Named("service.agentruntime"),
		queryTools:  p.QueryTools,
		actionTools: p.ActionTools,
		permissions: p.Permissions,
		definitions: p.Definitions,
		catalog:     p.Catalog,
		extensions:  p.Extensions,
		vectorizer:  p.Vectorizer,
		vectors:     p.CatalogVectors,
		disclosures: p.Disclosures,
		threshold:   p.Threshold,
		registry:    permissioncatalog.NewRegistry(),
		scripts:     p.Scripts,
	}
}
