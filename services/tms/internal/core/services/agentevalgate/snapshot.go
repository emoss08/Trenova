package agentevalgate

import (
	"sort"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
)

type CatalogSnapshot struct {
	Tools []ToolSnapshot `json:"tools"`
}

type ToolSnapshot struct {
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Parameters    map[string]any `json:"parameters,omitempty"`
	SearchTerms   []string       `json:"searchTerms,omitempty"`
	Prerequisites []string       `json:"prerequisites,omitempty"`
	Policy        PolicySnapshot `json:"policy"`
}

type PolicySnapshot struct {
	Kind                string   `json:"kind"`
	Resource            string   `json:"resource"`
	Operation           string   `json:"operation"`
	Scope               string   `json:"scope"`
	DefaultTier         string   `json:"defaultTier"`
	MaxTier             string   `json:"maxTier"`
	PromotableTo        string   `json:"promotableTo"`
	Egress              []string `json:"egress"`
	ClassifiesEachCall  bool     `json:"classifiesEachCall"`
	Condition           string   `json:"condition,omitempty"`
	PersonalRunsUnasked bool     `json:"personalRunsUnasked"`
	HeldWhenTainted     bool     `json:"heldWhenTainted"`
	TaintHold           string   `json:"taintHold,omitempty"`
	Effect              string   `json:"effect"`
	Artifact            string   `json:"artifact,omitempty"`
	Reversible          bool     `json:"reversible"`
	Idempotent          bool     `json:"idempotent"`
	ReadsExternal       string   `json:"readsExternal"`
	Source              string   `json:"source,omitempty"`
	CarriesTaint        bool     `json:"carriesTaint"`
	Rationale           string   `json:"rationale"`
}

func (k *Kit) Snapshot() CatalogSnapshot {
	descriptors := make(
		map[string]serviceports.AgentToolDescriptor,
		len(k.Queries.All())+len(k.Actions.All()),
	)
	for _, descriptor := range k.Queries.Descriptors() {
		descriptors[descriptor.Name] = descriptor
	}
	for _, descriptor := range k.Actions.Descriptors() {
		descriptors[descriptor.Name] = descriptor
	}

	policies := make([]serviceports.ToolPolicy, 0, len(descriptors)+len(k.Runtime))
	for _, tool := range k.Queries.All() {
		policies = append(policies, tool.Policy())
	}
	for _, tool := range k.Actions.All() {
		policies = append(policies, tool.Policy())
	}
	policies = append(policies, k.Runtime...)

	tools := make([]ToolSnapshot, 0, len(policies))
	for idx := range policies {
		policy := policies[idx]
		descriptor := descriptors[policy.Name]
		tools = append(tools, ToolSnapshot{
			Name:          policy.Name,
			Description:   descriptor.Description,
			Parameters:    descriptor.Parameters,
			SearchTerms:   descriptor.SearchTerms,
			Prerequisites: descriptor.Prerequisites,
			Policy:        snapshotPolicy(&policy),
		})
	}
	sort.SliceStable(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })

	return CatalogSnapshot{Tools: tools}
}

func snapshotPolicy(policy *serviceports.ToolPolicy) PolicySnapshot {
	egress := make([]string, 0, len(policy.Egress))
	for _, class := range policy.Egress {
		egress = append(egress, class.String())
	}

	condition := ""
	if policy.Condition != nil {
		condition = policy.Condition.Description
	}
	taintHold := ""
	if policy.TaintHold != nil {
		taintHold = policy.TaintHold.Description
	}

	return PolicySnapshot{
		Kind:                policy.Kind.String(),
		Resource:            policy.Resource.String(),
		Operation:           string(policy.Operation),
		Scope:               policy.Scope.String(),
		DefaultTier:         string(policy.DefaultTier),
		MaxTier:             string(policy.MaxTier),
		PromotableTo:        string(agenttoolpolicy.Promotable(*policy)),
		Egress:              egress,
		ClassifiesEachCall:  policy.Classify != nil,
		Condition:           condition,
		PersonalRunsUnasked: policy.PersonalRunsUnasked,
		HeldWhenTainted:     policy.HeldWhenTainted(),
		TaintHold:           taintHold,
		Effect:              string(policy.EffectiveEffect()),
		Artifact:            policy.Artifact,
		Reversible:          policy.Reversible,
		Idempotent:          policy.Idempotent,
		ReadsExternal:       policy.ReadsExternal.String(),
		Source:              policy.Source.String(),
		CarriesTaint:        policy.CarriesTaint,
		Rationale:           policy.Rationale,
	}
}
