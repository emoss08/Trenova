package agentruntime

import (
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	fingerprintPromptVersion   = "agent-definition/v2"
	canonicalOrganizationName  = "Canonical organization"
	canonicalBusinessUnitName  = "Canonical business unit"
	canonicalTimezone          = "UTC"
	canonicalNow               = int64(1_700_000_000)
	fingerprintToolSpecVersion = "tools/v1"
)

type fingerprintPolicy struct {
	Kind                agent.ToolKind       `json:"kind"`
	Resource            permission.Resource  `json:"resource"`
	Operation           permission.Operation `json:"operation"`
	Scope               agent.ToolScope      `json:"scope"`
	DefaultTier         agent.AutonomyTier   `json:"defaultTier"`
	MaxTier             agent.AutonomyTier   `json:"maxTier"`
	Egress              []agent.EgressClass  `json:"egress"`
	Classified          bool                 `json:"classified"`
	Condition           string               `json:"condition,omitempty"`
	PersonalRunsUnasked bool                 `json:"personalRunsUnasked"`
	Effect              agent.ToolEffect     `json:"effect"`
	Artifact            string               `json:"artifact,omitempty"`
	Reversible          bool                 `json:"reversible"`
	Idempotent          bool                 `json:"idempotent"`
	ReadsExternal       agent.ExternalRead   `json:"readsExternal"`
	Source              agent.TaintSource    `json:"source,omitempty"`
	CarriesTaint        bool                 `json:"carriesTaint"`
	Rationale           string               `json:"rationale"`
}

type fingerprintTool struct {
	Name        string             `json:"name"`
	Registered  bool               `json:"registered"`
	Description string             `json:"description,omitempty"`
	Parameters  map[string]any     `json:"parameters,omitempty"`
	Tier        agent.AutonomyTier `json:"tier,omitempty"`
	SetTier     agent.AutonomyTier `json:"setTier,omitempty"`
	Policy      *fingerprintPolicy `json:"policy,omitempty"`
}

type fingerprintToolSet struct {
	Version string            `json:"version"`
	Tools   []fingerprintTool `json:"tools"`
}

func projectPolicy(policy serviceports.ToolPolicy) *fingerprintPolicy {
	projected := &fingerprintPolicy{
		Kind:                policy.Kind,
		Resource:            policy.Resource,
		Operation:           policy.Operation,
		Scope:               policy.Scope,
		DefaultTier:         policy.DefaultTier,
		MaxTier:             policy.MaxTier,
		Egress:              slices.Clone(policy.Egress),
		Classified:          policy.Classify != nil,
		PersonalRunsUnasked: policy.PersonalRunsUnasked,
		Effect:              policy.Effect,
		Artifact:            policy.Artifact,
		Reversible:          policy.Reversible,
		Idempotent:          policy.Idempotent,
		ReadsExternal:       policy.ReadsExternal,
		Source:              policy.Source,
		CarriesTaint:        policy.CarriesTaint,
		Rationale:           policy.Rationale,
	}
	if policy.Condition != nil {
		projected.Condition = policy.Condition.Description
	}
	slices.Sort(projected.Egress)

	return projected
}

func (s *Service) fingerprintTool(
	definition *agentdefinition.Definition,
	name string,
) fingerprintTool {
	if tool, ok := s.queryTools.Get(name); ok {
		return fingerprintTool{
			Name:        name,
			Registered:  true,
			Description: tool.Description(),
			Parameters:  tool.ParamSchema(),
			Policy:      projectPolicy(tool.Policy()),
		}
	}
	if tool, ok := s.actionTools.Get(name); ok {
		policy := tool.Policy()
		projected := fingerprintTool{
			Name:        name,
			Registered:  true,
			Description: tool.Description(),
			Parameters:  tool.ParamSchema(),
			Tier:        agenttoolpolicy.StaticTier(definition, policy),
			Policy:      projectPolicy(policy),
		}
		if definition != nil && definition.SetsToolTier(name) {
			projected.SetTier = definition.ToolTiers[name]
		}

		return projected
	}
	if policy, ok := runtimePolicyNamed(name); ok {
		return fingerprintTool{Name: name, Registered: true, Policy: projectPolicy(policy)}
	}

	return fingerprintTool{Name: name}
}

func (s *Service) toolSpecHash(definition *agentdefinition.Definition, held []string) string {
	set := fingerprintToolSet{
		Version: fingerprintToolSpecVersion,
		Tools:   make([]fingerprintTool, 0, len(held)),
	}
	for _, name := range held {
		set.Tools = append(set.Tools, s.fingerprintTool(definition, name))
	}

	encoded, err := sonic.ConfigStd.Marshal(set)
	if err != nil {
		s.logger.Warn("could not encode an agent's tools for its fingerprint", zap.Error(err))

		return hashutils.SHA256Hex(fingerprintToolSpecVersion + "|" + strings.Join(held, "\n"))
	}

	return hashutils.SHA256BytesHex(encoded)
}

func canonicalRuntimeContext(tools []agentdefinition.ToolSummary) agentdefinition.RuntimeContext {
	return agentdefinition.RuntimeContext{
		OrganizationName: canonicalOrganizationName,
		BusinessUnitName: canonicalBusinessUnitName,
		Timezone:         canonicalTimezone,
		Now:              canonicalNow,
		Tools:            tools,
	}
}

func (s *Service) Fingerprint(
	definition *agentdefinition.Definition,
	providerID pulid.ID,
	model string,
) *agent.Fingerprint {
	if definition == nil {
		return nil
	}

	held := s.heldTools(definition)
	sorted := slices.Clone(held)
	slices.Sort(sorted)
	sorted = slices.Compact(sorted)

	prompt := definition.BuildSystemPrompt(
		canonicalRuntimeContext(s.summarize(definition, held)),
	)

	return &agent.Fingerprint{
		DefinitionVersion: definition.Version,
		PromptHash:        hashutils.SHA256Hex(prompt),
		ToolSpecHash:      s.toolSpecHash(definition, sorted),
		Model:             model,
		ProviderID:        providerID,
		PromptVersion:     fingerprintPromptVersion,
		InstructionsHash:  hashutils.SHA256Hex(definition.Instructions),
		Tools:             sorted,
	}
}
