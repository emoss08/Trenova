package tooldispatch

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"go.uber.org/zap"
)

type DispatcherParams struct {
	Logger      *zap.Logger
	Access      *toolaccess.Access
	QueryTools  agenttoolports.AgentQueryToolRegistry
	ActionTools agenttoolports.AgentToolRegistry
	Permissions permissionports.PermissionEngine
	// Versions is optional. Without it proposals are still made, just without
	// the record pinned; with it the executor can refuse a change to a record
	// that moved on since the proposal.
	Versions agenttoolports.RecordVersionReader
	// Budgets is optional. With it an automatic write past its tool's daily
	// cap is refused before it runs.
	Budgets agentbudgetports.AgentBudgetService
	Trust   agenttoolports.AgentToolTrustRepository
	// Previews is optional. With it a held write's target is pinned, and
	// what it would do kept as its baseline, in one read-only snapshot;
	// without it the target is pinned alone.
	Previews agentruntimeports.ProposalPreviewService
	// Rules is optional. With it a tool an organization holds lower than its
	// declared rule is decided and tainted by the organization's rule.
	Rules agenttoolports.ToolRuleOverrides
	// CodeMode decides which models may run scripts.
	CodeMode codemode.Gate
	// Shadow is optional. With it a write by an agent in shadow is held.
	Shadow ShadowVerdicts
	// Scripts is optional. With it the code-mode gate reads the chat
	// providers' script settings.
	Scripts llmports.ScriptProviders
	// Brokers opens the broker a script's calls go through.
	Brokers ScriptBrokers
}

// Dispatcher runs one tool call the loop has decided to make: it checks the
// call's arguments and the person's permission, decides whether a write runs,
// waits for a person or is refused, and runs scripts through their broker.
type Dispatcher struct {
	logger      *zap.Logger
	access      *toolaccess.Access
	queryTools  agenttoolports.AgentQueryToolRegistry
	actionTools agenttoolports.AgentToolRegistry
	permissions permissionports.PermissionEngine
	versions    agenttoolports.RecordVersionReader
	budgets     agentbudgetports.AgentBudgetService
	trust       agenttoolports.AgentToolTrustRepository
	previews    agentruntimeports.ProposalPreviewService
	rules       agenttoolports.ToolRuleOverrides
	// arguments holds each tool's compiled schema, against which every
	// call's arguments are checked before the tool sees them.
	arguments *toolschema.Validator
	codeMode  codemode.Gate
	scripts   llmports.ScriptProviders
	shadow    ShadowVerdicts
	brokers   ScriptBrokers
}

func NewDispatcher(p DispatcherParams) *Dispatcher {
	return &Dispatcher{
		logger:      p.Logger.Named("service.agentruntime"),
		access:      p.Access,
		queryTools:  p.QueryTools,
		actionTools: p.ActionTools,
		permissions: p.Permissions,
		versions:    p.Versions,
		budgets:     p.Budgets,
		trust:       p.Trust,
		previews:    p.Previews,
		rules:       p.Rules,
		arguments:   toolschema.NewValidator(),
		codeMode:    p.CodeMode,
		scripts:     p.Scripts,
		shadow:      p.Shadow,
		brokers:     p.Brokers,
	}
}

// Access is the tool access the dispatcher checks calls against.
func (s *Dispatcher) Access() *toolaccess.Access { return s.access }

// QueryTools are the read tools the dispatcher runs.
func (s *Dispatcher) QueryTools() agenttoolports.AgentQueryToolRegistry { return s.queryTools }

// ActionTools are the write tools the dispatcher runs or proposes.
func (s *Dispatcher) ActionTools() agenttoolports.AgentToolRegistry { return s.actionTools }

// ScriptBrokers opens the broker every call a script makes goes through.
type ScriptBrokers func(ctx context.Context, d *Dispatcher, p *ScriptBrokerParams) ScriptBroker

// ScriptBroker is a broker as the tool that runs a script holds it: the
// script calls through it, and once the script ends it reports what was
// read and settles what was prepared.
type ScriptBroker interface {
	codemode.Broker
	TaintMarks() []agent.TaintMark
	Settle(ctx context.Context, settlement *ScriptSettlement) []agentruntimeports.PendingAction
}
