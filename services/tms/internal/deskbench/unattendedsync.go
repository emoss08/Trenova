package deskbench

import (
	"context"
	"fmt"
	"maps"

	"github.com/emoss08/trenova/internal/deskbench/benchagent"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Session) allAgents(ctx context.Context) ([]*agentdefinition.Definition, error) {
	result, err := s.bench.Definitions.List(
		s.Context(ctx),
		&agentdefinitionports.ListAgentDefinitionRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: s.Tenant,
				Pagination: pagination.Info{Limit: pagination.MaxLimit},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	return result.Items, nil
}

func (s *Session) UnattendedAgent(
	ctx context.Context,
	template agenttemplate.Template,
) (*agentdefinition.Definition, error) {
	agents, err := s.allAgents(ctx)
	if err != nil {
		return nil, err
	}
	name := benchagent.UnattendedAgentName(template)
	for _, definition := range agents {
		if definition.Name == name {
			return definition, nil
		}
	}

	return nil, fmt.Errorf("the bench has no %s agent; run `trenova desk run` to sync it", name)
}

func (s *Session) SyncUnattendedAgents(ctx context.Context) (*benchagent.AgentSync, error) {
	ctx = s.Context(ctx)
	outcome := &benchagent.AgentSync{}
	agents, err := s.allAgents(ctx)
	if err != nil {
		return outcome, err
	}
	byName := make(map[string]*agentdefinition.Definition, len(agents))
	for _, definition := range agents {
		byName[definition.Name] = definition
	}

	for _, template := range benchagent.UnattendedTemplates() {
		want := benchagent.UnattendedShape(template)
		existing := byName[want.Name]
		switch {
		case existing == nil:
			err = s.createUnattended(ctx, want)
			outcome.Created = append(outcome.Created, want.Name)
		case benchagent.UnattendedMatches(existing, want):
			outcome.Unchanged = append(outcome.Unchanged, want.Name)
		default:
			err = s.reshapeUnattended(ctx, existing, want)
			outcome.Updated = append(outcome.Updated, want.Name)
		}
		if err != nil {
			return outcome, fmt.Errorf("sync %s: %w", want.Name, err)
		}
	}

	return outcome, nil
}

func (s *Session) createUnattended(ctx context.Context, want *agentdefinition.Definition) error {
	learningOff := true
	_, err := s.bench.AgentDefinitions.Create(ctx, &agentdefinitionports.SaveAgentDefinitionRequest{
		Name:              want.Name,
		Description:       want.Description,
		Template:          want.Template,
		Instructions:      want.Instructions,
		ToolNames:         want.ToolNames,
		AutonomyCeiling:   want.AutonomyCeiling,
		DataAccessCeiling: want.DataAccessCeiling,
		Enabled:           false,
		TriggerMode:       want.TriggerMode,
		CronExpression:    want.CronExpression,
		EventKinds:        want.EventKinds,
		ContextProviders:  want.ContextProviders,
		OutputMode:        want.OutputMode,
		LearningOff:       &learningOff,
		TenantInfo:        s.Tenant,
	}, &s.Actor)

	return err
}

func (s *Session) reshapeUnattended(
	ctx context.Context,
	existing, want *agentdefinition.Definition,
) error {
	_, err := s.bench.AgentDefinitions.Patch(ctx, &agentdefinitionports.PatchAgentDefinitionRequest{
		ID:         existing.ID,
		TenantInfo: s.Tenant,
		Version:    existing.Version,
		Comment:    benchagent.AgentSyncComment,
		Edit: func(definition *agentdefinition.Definition) error {
			definition.Description = want.Description
			definition.Template = want.Template
			definition.Instructions = want.Instructions
			definition.ToolNames = want.ToolNames
			definition.AutonomyCeiling = want.AutonomyCeiling
			definition.DataAccessCeiling = want.DataAccessCeiling
			definition.TriggerMode = want.TriggerMode
			definition.CronExpression = want.CronExpression
			definition.EventKinds = want.EventKinds
			definition.OutputMode = want.OutputMode
			definition.ContextProviders = want.ContextProviders
			definition.ToolTiers = nil
			definition.LearningOff = true
			definition.ShadowMode = false
			definition.Enabled = false

			return nil
		},
	}, &s.Actor)

	return err
}

func (s *Session) ShapeForRun(
	ctx context.Context,
	definition *agentdefinition.Definition,
	fire *benchscenario.Fire,
	enabled bool,
) error {
	ceiling := agentpreset.StarterCeiling(definition.Template)
	var tiers map[string]agent.AutonomyTier
	if enabled {
		ceiling = fire.AutonomyCeiling(definition.Template)
		tiers = maps.Clone(fire.ToolTiers)
	}

	_, err := s.bench.AgentDefinitions.Patch(
		s.Context(ctx),
		&agentdefinitionports.PatchAgentDefinitionRequest{
			ID:         definition.ID,
			TenantInfo: s.Tenant,
			Comment:    benchagent.AgentSyncComment,
			Edit: func(edited *agentdefinition.Definition) error {
				edited.Enabled = enabled
				edited.AutonomyCeiling = ceiling
				edited.ToolTiers = tiers
				return nil
			},
		},
		&s.Actor,
	)

	return err
}
