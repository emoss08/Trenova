package modeprofile

import (
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MatchOrgDefault   = "organizationDefault"
	MatchCustomer     = "customer"
	MatchServiceType  = "serviceType"
	MatchShipmentType = "shipmentType"
	MatchEquipmentTyp = "equipmentType"
)

type Provenance struct {
	ProfileID          pulid.ID                `json:"profileId"`
	ProfileCode        string                  `json:"profileCode"`
	ProfileName        string                  `json:"profileName"`
	IsOrgDefault       bool                    `json:"isOrgDefault"`
	Priority           int16                   `json:"priority"`
	SpecificityScore   int32                   `json:"specificityScore"`
	MatchedOn          []string                `json:"matchedOn"`
	RuleKey            RuleKey                 `json:"ruleKey"`
	Capability         Capability              `json:"capability"`
	CapabilityLabel    string                  `json:"capabilityLabel"`
	Enforcement        tenant.EnforcementLevel `json:"enforcement"`
	DefaultEnforcement tenant.EnforcementLevel `json:"defaultEnforcement"`
	Overridden         bool                    `json:"overridden"`
	OverrideReason     string                  `json:"overrideReason,omitempty"`
	Rationale          string                  `json:"rationale"`
}

type ResolvedRule struct {
	Key         RuleKey                 `json:"key"`
	Capability  Capability              `json:"capability"`
	Label       string                  `json:"label"`
	Enforcement tenant.EnforcementLevel `json:"enforcement"`
	Enabled     bool                    `json:"enabled"`

	// Fields the rule targets, for explaining why a field behaves as it does.
	// Not the same as the fields it makes mandatory — see RequiredFields.
	Fields []string `json:"fields,omitempty"`

	// Fields this rule makes mandatory under its resolved parameters. Narrower
	// than the definition's list where a parameter turns a requirement off.
	RequiredFields []string `json:"requiredFields,omitempty"`

	Parameters map[string]any `json:"parameters,omitempty"`
	Provenance Provenance     `json:"provenance"`
}

// RequiresField reports whether this rule makes the given field mandatory.
//
// Blocking is part of the question: a rule that only warns records a deviation
// rather than refusing the save, so the field is not actually required.
func (r *ResolvedRule) RequiresField(field string) bool {
	if !r.Blocks() {
		return false
	}

	return slices.Contains(r.RequiredFields, field)
}

func (r *ResolvedRule) Blocks() bool {
	return r.Enabled && r.Enforcement == tenant.EnforcementLevelBlock
}

func (r *ResolvedRule) Records() bool {
	if !r.Enabled {
		return false
	}
	return r.Enforcement == tenant.EnforcementLevelWarn ||
		r.Enforcement == tenant.EnforcementLevelRequireReview
}

func (r *ResolvedRule) Applies() bool {
	return r.Enabled && r.Enforcement != tenant.EnforcementLevelIgnore
}

type ProfileCandidate struct {
	ProfileID        pulid.ID `json:"profileId"`
	ProfileCode      string   `json:"profileCode"`
	ProfileName      string   `json:"profileName"`
	Priority         int16    `json:"priority"`
	SpecificityScore int32    `json:"specificityScore"`
	MatchedOn        []string `json:"matchedOn"`
	Selected         bool     `json:"selected"`
	RejectionReason  string   `json:"rejectionReason,omitempty"`
}

type ResolvedPolicy struct {
	ProfileID      pulid.ID                 `json:"profileId"`
	ProfileCode    string                   `json:"profileCode"`
	ProfileName    string                   `json:"profileName"`
	ServiceModel   ServiceModel             `json:"serviceModel"`
	EquipmentClass EquipmentClass           `json:"equipmentClass"`
	ExecutionParty ExecutionParty           `json:"executionParty"`
	Capabilities   []Capability             `json:"capabilities"`
	Rules          map[RuleKey]ResolvedRule `json:"rules"`
	Candidates     []ProfileCandidate       `json:"candidates,omitempty"`
	ResolvedAt     int64                    `json:"resolvedAt"`
}

func (p *ResolvedPolicy) Rule(key RuleKey) (ResolvedRule, bool) {
	if p == nil {
		return ResolvedRule{}, false
	}
	rule, ok := p.Rules[key]
	return rule, ok
}

func (p *ResolvedPolicy) Blocks(key RuleKey) bool {
	rule, ok := p.Rule(key)
	return ok && rule.Blocks()
}

func (p *ResolvedPolicy) Records(key RuleKey) bool {
	rule, ok := p.Rule(key)
	return ok && rule.Records()
}

func (p *ResolvedPolicy) Applies(key RuleKey) bool {
	rule, ok := p.Rule(key)
	return ok && rule.Applies()
}

func (p *ResolvedPolicy) EnforcementFor(key RuleKey) tenant.EnforcementLevel {
	rule, ok := p.Rule(key)
	if !ok || !rule.Enabled {
		return tenant.EnforcementLevelIgnore
	}
	return rule.Enforcement
}

func (p *ResolvedPolicy) HasCapability(capability Capability) bool {
	if p == nil {
		return false
	}
	for _, c := range p.Capabilities {
		if c == capability {
			return true
		}
	}
	return false
}

func (p *ResolvedPolicy) IntParam(key RuleKey, name string) (int64, bool) {
	rule, ok := p.Rule(key)
	if !ok {
		return 0, false
	}

	if value, present := rule.Parameters[name]; present && value != nil {
		switch typed := value.(type) {
		case int64:
			return typed, true
		case int32:
			return int64(typed), true
		case int:
			return int64(typed), true
		case float64:
			return int64(typed), true
		case float32:
			return int64(typed), true
		}
	}

	def, err := RuleDefinitionFor(key)
	if err != nil {
		return 0, false
	}
	for _, param := range def.Parameters {
		if param.Name != name || param.Default == nil {
			continue
		}
		switch typed := param.Default.(type) {
		case int64:
			return typed, true
		case int:
			return int64(typed), true
		case float64:
			return int64(typed), true
		}
	}

	return 0, false
}

func (p *ResolvedPolicy) BoolParam(key RuleKey, name string) (enabled, found bool) {
	rule, ok := p.Rule(key)
	if !ok {
		return false, false
	}

	if value, present := rule.Parameters[name]; present && value != nil {
		if typed, isBool := value.(bool); isBool {
			return typed, true
		}
	}

	def, err := RuleDefinitionFor(key)
	if err != nil {
		return false, false
	}
	for _, param := range def.Parameters {
		if param.Name != name || param.Default == nil {
			continue
		}
		if typed, isBool := param.Default.(bool); isBool {
			return typed, true
		}
	}

	return false, false
}
