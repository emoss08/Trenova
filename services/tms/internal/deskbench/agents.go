package deskbench

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"gopkg.in/yaml.v3"
)

const (
	BenchAgentPrefix = "Bench · "
	agentSyncComment = "Synced from deskbench/agents.yaml by the desk bench"
)

type AgentSpec struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Template     string   `yaml:"template"`
	Instructions string   `yaml:"instructions"`
	Tools        []string `yaml:"tools"`
	Delegates    []string `yaml:"delegates"`
	MaxToolCalls int      `yaml:"maxToolCalls"`
	Ceiling      string   `yaml:"autonomyCeiling"`
}

type AgentSync struct {
	Created   []string
	Updated   []string
	Unchanged []string
}

var ErrAgentSpecUnnamed = errors.New("a bench agent has no name")

func LoadAgentSpecs(path string) ([]AgentSpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var doc struct {
		Agents []AgentSpec `yaml:"agents"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err = decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	for idx := range doc.Agents {
		spec := &doc.Agents[idx]
		if strings.TrimSpace(spec.Name) == "" {
			return nil, fmt.Errorf("%s: agent %d: %w", path, idx+1, ErrAgentSpecUnnamed)
		}
		if !strings.HasPrefix(spec.Name, BenchAgentPrefix) {
			spec.Name = BenchAgentPrefix + spec.Name
		}
	}

	return doc.Agents, nil
}

func (s *Session) SyncAgents(ctx context.Context, specs []AgentSpec) (*AgentSync, error) {
	ctx = s.Context(ctx)
	outcome := &AgentSync{}
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

func (s *Session) benchAgent(ctx context.Context, name string) (*agentdefinition.Definition, error) {
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

func (s *Session) syncAgent(ctx context.Context, spec *AgentSpec, outcome *AgentSync) error {
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
		_, err = s.bench.AgentDefinitions.Create(ctx, &serviceports.SaveAgentDefinitionRequest{
			Name:            spec.Name,
			Description:     spec.Description,
			Template:        agentdefinition.Template(spec.Template),
			Instructions:    spec.Instructions,
			ToolNames:       tools,
			AutonomyCeiling: ceiling,
			Enabled:         true,
			TriggerMode:     agentdefinition.TriggerChat,
			MaxToolCalls:    spec.MaxToolCalls,
			TenantInfo:      s.Tenant,
		}, &s.Actor)
		if err != nil {
			return err
		}
		outcome.Created = append(outcome.Created, spec.Name)

		return nil
	}

	if agentMatches(existing, spec, tools, ceiling) {
		outcome.Unchanged = append(outcome.Unchanged, spec.Name)
		return nil
	}

	_, err = s.bench.AgentDefinitions.Patch(ctx, &serviceports.PatchAgentDefinitionRequest{
		ID:         existing.ID,
		TenantInfo: s.Tenant,
		Version:    existing.Version,
		Comment:    agentSyncComment,
		Edit: func(definition *agentdefinition.Definition) error {
			definition.Description = spec.Description
			definition.Template = agentdefinition.Template(spec.Template)
			definition.Instructions = spec.Instructions
			definition.ToolNames = tools
			definition.AutonomyCeiling = ceiling
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

func agentMatches(
	existing *agentdefinition.Definition,
	spec *AgentSpec,
	tools []string,
	ceiling agent.AutonomyTier,
) bool {
	held := slices.Clone(existing.ToolNames)
	wanted := slices.Clone(tools)
	slices.Sort(held)
	slices.Sort(wanted)

	return existing.Enabled &&
		existing.Description == spec.Description &&
		string(existing.Template) == spec.Template &&
		existing.Instructions == spec.Instructions &&
		existing.AutonomyCeiling == ceiling &&
		(spec.MaxToolCalls == 0 || existing.MaxToolCalls == spec.MaxToolCalls) &&
		slices.Equal(held, wanted)
}

func (s *Session) syncDelegates(ctx context.Context, spec *AgentSpec) error {
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

	_, err = s.bench.AgentDefinitions.Patch(ctx, &serviceports.PatchAgentDefinitionRequest{
		ID:         existing.ID,
		TenantInfo: s.Tenant,
		Version:    existing.Version,
		Comment:    agentSyncComment,
		Edit: func(definition *agentdefinition.Definition) error {
			definition.DelegateIDs = ids
			return nil
		},
	}, &s.Actor)

	return err
}
