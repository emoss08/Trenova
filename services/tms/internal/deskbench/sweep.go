package deskbench

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
)

const (
	sweepAgentPrefix = BenchAgentPrefix + "Sweep · "
	NeverDisclose    = 1_000_000
	AlwaysDisclose   = 0
)

type SweepMode string

const (
	SweepAll       = SweepMode("all")
	SweepDisclosed = SweepMode("disclosed")
)

func (m SweepMode) Threshold() int {
	if m == SweepDisclosed {
		return AlwaysDisclose
	}

	return NeverDisclose
}

type Sweep struct {
	Held int
	Mode SweepMode
}

func (s *Sweep) Label() string {
	return "held " + strconv.Itoa(s.Held) + ", " + string(s.Mode)
}

var ErrSweepSkipped = errors.New("skipped by the tool-count sweep")

func sweepTools(base, expected []string, held int) ([]string, error) {
	base = agentdefinition.WithoutCoreTools(base)
	required := make([]string, 0, len(expected))
	for _, spec := range expected {
		options := alternatives(spec)
		chosen := ""
		for _, option := range options {
			if slices.Contains(base, option) {
				chosen = option
				break
			}
		}
		if chosen == "" {
			if len(options) == 0 || agentdefinition.IsCoreTool(options[0]) || plumbing(options[0]) {
				continue
			}
			chosen = options[0]
		}
		if !slices.Contains(required, chosen) {
			required = append(required, chosen)
		}
	}
	if len(required) > held {
		return nil, fmt.Errorf("%w: the scenario needs %d tools, more than %d", ErrSweepSkipped,
			len(required), held)
	}

	tools := slices.Clone(required)
	for _, name := range base {
		if len(tools) == held {
			break
		}
		if !slices.Contains(tools, name) {
			tools = append(tools, name)
		}
	}
	if len(tools) < held {
		return nil, fmt.Errorf("%w: the agent holds %d tools, fewer than %d", ErrSweepSkipped,
			len(tools), held)
	}
	slices.Sort(tools)

	return tools, nil
}

func sweepAgentName(base *agentdefinition.Definition) string {
	return sweepAgentPrefix + base.Name
}

func (s *Session) sweepAgent(
	ctx context.Context,
	base *agentdefinition.Definition,
	tools []string,
) (*agentdefinition.Definition, error) {
	ctx = s.Context(ctx)
	name := sweepAgentName(base)
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
		Comment:    agentSyncComment,
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
