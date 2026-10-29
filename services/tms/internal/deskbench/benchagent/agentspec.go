package benchagent

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"gopkg.in/yaml.v3"
)

const (
	sweepAgentPrefix = BenchAgentPrefix + "Sweep · "
)

const (
	BenchAgentPrefix = "Bench · "
	AgentSyncComment = "Synced from deskbench/agents.yaml by the desk bench"
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
	DataAccess   string   `yaml:"dataAccess"`
}

func (s *AgentSpec) DataAccessCeiling() agentdefinition.DataAccessCeiling {
	if s.DataAccess == "" {
		return agentdefinition.DataAccessInternal
	}

	return agentdefinition.DataAccessCeiling(s.DataAccess)
}

type AgentSync struct {
	Created   []string
	Updated   []string
	Unchanged []string
}

var (
	ErrAgentSpecUnnamed    = errors.New("a bench agent has no name")
	ErrAgentSpecDataAccess = errors.New("dataAccess must be Internal or Restricted")
)

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
		if !spec.DataAccessCeiling().IsValid() {
			return nil, fmt.Errorf("%s: %s: %w", path, spec.Name, ErrAgentSpecDataAccess)
		}
		if !strings.HasPrefix(spec.Name, BenchAgentPrefix) {
			spec.Name = BenchAgentPrefix + spec.Name
		}
	}

	return doc.Agents, nil
}

func AgentMatches(
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
		existing.DataAccessCeiling == spec.DataAccessCeiling() &&
		(spec.MaxToolCalls == 0 || existing.MaxToolCalls == spec.MaxToolCalls) &&
		slices.Equal(held, wanted)
}

func IsBenchAgent(definition *agentdefinition.Definition) bool {
	return definition != nil && strings.HasPrefix(definition.Name, BenchAgentPrefix)
}

func SweepAgentName(base *agentdefinition.Definition) string {
	return sweepAgentPrefix + base.Name
}
