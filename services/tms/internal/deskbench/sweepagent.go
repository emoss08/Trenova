package deskbench

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/deskbench/benchagent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
)

func (s *Session) SweepAgent(
	ctx context.Context,
	base *agentdefinition.Definition,
	tools []string,
) (*agentdefinition.Definition, error) {
	ctx = s.Context(ctx)
	name := benchagent.SweepAgentName(base)
	agents, err := s.allAgents(ctx)
	if err != nil {
		return nil, err
	}
	idx := slices.IndexFunc(agents, func(definition *agentdefinition.Definition) bool {
		return definition.Name == name
	})
	if idx < 0 {
		learningOff := true
		if _, err = s.bench.AgentDefinitions.Create(ctx,
			&agentdefinitionports.SaveAgentDefinitionRequest{
				Name:              name,
				Description:       base.Description,
				Template:          base.Template,
				Instructions:      base.Instructions,
				ToolNames:         tools,
				AutonomyCeiling:   base.AutonomyCeiling,
				DataAccessCeiling: base.DataAccessCeiling,
				Enabled:           true,
				TriggerMode:       agentdefinition.TriggerChat,
				MaxToolCalls:      base.MaxToolCalls,
				LearningOff:       &learningOff,
				TenantInfo:        s.Tenant,
			}, &s.Actor); err != nil {
			return nil, fmt.Errorf("create %s: %w", name, err)
		}

		return s.Agent(ctx, name)
	}

	existing := agents[idx]
	if existing.Enabled && slices.Equal(sortedCopy(existing.ToolNames), tools) &&
		existing.Instructions == base.Instructions && existing.Template == base.Template {
		return existing, nil
	}
	patch := &agentdefinitionports.PatchAgentDefinitionRequest{
		ID:         existing.ID,
		TenantInfo: s.Tenant,
		Version:    existing.Version,
		Comment:    benchagent.AgentSyncComment,
		Edit: func(definition *agentdefinition.Definition) error {
			definition.Template = base.Template
			definition.Instructions = base.Instructions
			definition.Description = base.Description
			definition.ToolNames = tools
			definition.AutonomyCeiling = base.AutonomyCeiling
			definition.DataAccessCeiling = base.DataAccessCeiling
			definition.MaxToolCalls = base.MaxToolCalls
			definition.LearningOff = true
			definition.Enabled = true

			return nil
		},
	}
	if _, err = s.bench.AgentDefinitions.Patch(ctx, patch, &s.Actor); err != nil {
		return nil, fmt.Errorf("give %s its tools: %w", name, err)
	}

	return s.Agent(ctx, name)
}

func sortedCopy(values []string) []string {
	sorted := slices.Clone(values)
	slices.Sort(sorted)

	return sorted
}
