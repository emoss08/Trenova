package deskbench

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/deskbench/benchagent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Session) SyncAgents(
	ctx context.Context,
	specs []benchagent.AgentSpec,
) (*benchagent.AgentSync, error) {
	ctx = s.Context(ctx)
	outcome := &benchagent.AgentSync{}
	if len(specs) == 0 {
		return outcome, nil
	}

	for idx := range specs {
		if err := s.syncAgent(ctx, &specs[idx], outcome); err != nil {
			return outcome, fmt.Errorf("sync %s: %w", specs[idx].Name, err)
		}
	}
	for idx := range specs {
		if err := s.syncDelegates(ctx, &specs[idx]); err != nil {
			return outcome, fmt.Errorf("sync %s's delegates: %w", specs[idx].Name, err)
		}
	}

	return outcome, nil
}

func (s *Session) benchAgent(
	ctx context.Context,
	name string,
) (*agentdefinition.Definition, error) {
	agents, err := s.Agents(ctx)
	if err != nil {
		return nil, err
	}
	for _, definition := range agents {
		if definition.Name == name {
			return definition, nil
		}
	}

	return nil, nil
}

func (s *Session) syncAgent(
	ctx context.Context,
	spec *benchagent.AgentSpec,
	outcome *benchagent.AgentSync,
) error {
	existing, err := s.benchAgent(ctx, spec.Name)
	if err != nil {
		return err
	}

	ceiling := agent.AutonomyTier(spec.Ceiling)
	if ceiling == "" {
		ceiling = agent.TierActWithApproval
	}
	tools := agentdefinition.WithoutCoreTools(spec.Tools)

	if existing == nil {
		_, err = s.bench.AgentDefinitions.Create(
			ctx,
			&agentdefinitionports.SaveAgentDefinitionRequest{
				Name:              spec.Name,
				Description:       spec.Description,
				Template:          agenttemplate.Template(spec.Template),
				Instructions:      spec.Instructions,
				ToolNames:         tools,
				AutonomyCeiling:   ceiling,
				DataAccessCeiling: spec.DataAccessCeiling(),
				Enabled:           true,
				TriggerMode:       agentdefinition.TriggerChat,
				MaxToolCalls:      spec.MaxToolCalls,
				TenantInfo:        s.Tenant,
			},
			&s.Actor,
		)
		if err != nil {
			return err
		}
		outcome.Created = append(outcome.Created, spec.Name)

		return nil
	}

	if benchagent.AgentMatches(existing, spec, tools, ceiling) {
		outcome.Unchanged = append(outcome.Unchanged, spec.Name)
		return nil
	}

	_, err = s.bench.AgentDefinitions.Patch(ctx, &agentdefinitionports.PatchAgentDefinitionRequest{
		ID:         existing.ID,
		TenantInfo: s.Tenant,
		Version:    existing.Version,
		Comment:    benchagent.AgentSyncComment,
		Edit: func(definition *agentdefinition.Definition) error {
			definition.Description = spec.Description
			definition.Template = agenttemplate.Template(spec.Template)
			definition.Instructions = spec.Instructions
			definition.ToolNames = tools
			definition.AutonomyCeiling = ceiling
			definition.DataAccessCeiling = spec.DataAccessCeiling()
			definition.Enabled = true
			if spec.MaxToolCalls > 0 {
				definition.MaxToolCalls = spec.MaxToolCalls
			}

			return nil
		},
	}, &s.Actor)
	if err != nil {
		return err
	}
	outcome.Updated = append(outcome.Updated, spec.Name)

	return nil
}

func (s *Session) syncDelegates(ctx context.Context, spec *benchagent.AgentSpec) error {
	if spec.Delegates == nil {
		return nil
	}

	existing, err := s.benchAgent(ctx, spec.Name)
	if err != nil || existing == nil {
		return err
	}

	ids := make([]pulid.ID, 0, len(spec.Delegates))
	for _, name := range spec.Delegates {
		delegate, findErr := s.Agent(ctx, name)
		if findErr != nil {
			return findErr
		}
		ids = append(ids, delegate.ID)
	}
	if slices.Equal(existing.DelegateIDs, ids) {
		return nil
	}

	_, err = s.bench.AgentDefinitions.Patch(ctx, &agentdefinitionports.PatchAgentDefinitionRequest{
		ID:         existing.ID,
		TenantInfo: s.Tenant,
		Version:    existing.Version,
		Comment:    benchagent.AgentSyncComment,
		Edit: func(definition *agentdefinition.Definition) error {
			definition.DelegateIDs = ids
			return nil
		},
	}, &s.Actor)

	return err
}
