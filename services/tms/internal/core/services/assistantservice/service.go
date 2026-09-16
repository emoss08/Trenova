package assistantservice

import (
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	Guard       *agentguard.Service
	Completion  serviceports.CompletionService
	QueryTools  serviceports.AgentQueryToolRegistry
	ActionTools serviceports.AgentToolRegistry
}

type Service struct {
	logger      *zap.Logger
	guard       *agentguard.Service
	completion  serviceports.CompletionService
	queryTools  serviceports.AgentQueryToolRegistry
	actionTools serviceports.AgentToolRegistry
}

func New(p Params) *Service {
	return &Service{
		logger:      p.Logger.Named("service.assistant"),
		guard:       p.Guard,
		completion:  p.Completion,
		queryTools:  p.QueryTools,
		actionTools: p.ActionTools,
	}
}

// toolSpecsFor is the list of tools offered to the model for a turn.
//
// Read-only tools are offered unconditionally: they cannot change anything, so
// there is nothing for an organization to restrict. Write tools are offered only
// when the agent was configured with them, and even then the model asking for one
// produces a proposal rather than an action. The offered list is a prompt, so it
// is a convenience rather than the control — dispatch checks membership again
// when a call actually arrives.
func (s *Service) toolSpecsFor(definition *agentdefinition.Definition) []serviceports.ToolSpec {
	queryDescriptors := s.queryTools.Descriptors()
	specs := make([]serviceports.ToolSpec, 0, len(queryDescriptors)+len(definition.ToolNames))

	for _, descriptor := range queryDescriptors {
		specs = append(specs, serviceports.ToolSpec{
			Name:        descriptor.Name,
			Description: descriptor.Description,
			Parameters:  descriptor.Parameters,
		})
	}

	for _, name := range definition.ToolNames {
		tool, ok := s.actionTools.Get(name)
		if !ok {
			// A configured tool that no longer exists is skipped rather than
			// described to the model, which would only invite a call that fails.
			s.logger.Warn("configured tool is not in the registry",
				zap.String("tool", name),
				zap.String("agent", definition.Name),
			)
			continue
		}

		specs = append(specs, serviceports.ToolSpec{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.ParamSchema(),
		})
	}

	return specs
}
