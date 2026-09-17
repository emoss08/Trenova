package carrierintel

import (
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type RuleCode string

const (
	RuleUSDOTInactive          = RuleCode("authority.usdot_inactive")
	RuleAuthorityInactive      = RuleCode("authority.inactive")
	RuleAuthorityRevoked       = RuleCode("authority.revoked")
	RuleAuthorityPendingRevoke = RuleCode("authority.pending_revocation")
	RuleInsuranceNoneOnFile    = RuleCode("insurance.none_on_file")
	RuleInsuranceBIPDBelow     = RuleCode("insurance.bipd_below_required")
	RuleInsuranceCargoBelow    = RuleCode("insurance.cargo_below_required")
	RuleInsuranceBondBelow     = RuleCode("insurance.bond_below_required")
	RuleInsurancePendingCancel = RuleCode("insurance.pending_cancellation")
	RuleSafetyOOSOrder         = RuleCode("safety.oos_order")
	RuleSafetyUnsatisfactory   = RuleCode("safety.rating_unsatisfactory")
	RuleSafetyConditional      = RuleCode("safety.rating_conditional")
	RuleBasicsAlert            = RuleCode("basics.alert")
	RuleSafetyISSHigh          = RuleCode("safety.iss_high")
	RuleSafetyRiskScoreHigh    = RuleCode("safety.risk_score_high")
	RuleInspectionsOOSHigh     = RuleCode("inspections.oos_rate_high")
	RuleIdentityNewEntrant     = RuleCode("identity.new_entrant")
	RuleOperationsMCS150Stale  = RuleCode("operations.mcs150_stale")
	RuleFraudNetworkSharing    = RuleCode("fraud.network_sharing")
	RuleFraudContactChurn      = RuleCode("fraud.contact_churn")
	RuleBenchmarksAnomaly      = RuleCode("benchmarks.anomaly")
	RuleIdentityNotFound       = RuleCode("identity.not_found")
)

func (c RuleCode) String() string { return string(c) }

func (c RuleCode) IsValid() bool {
	_, ok := RuleByCode(c)
	return ok
}

type RuleParamType string

const (
	RuleParamTypeInteger     = RuleParamType("Integer")
	RuleParamTypeDecimal     = RuleParamType("Decimal")
	RuleParamTypeNumber      = RuleParamType("Number")
	RuleParamTypeSelect      = RuleParamType("Select")
	RuleParamTypeMultiSelect = RuleParamType("MultiSelect")
)

type RuleParamSpec struct {
	Key      string        `json:"key"`
	Label    string        `json:"label"`
	Type     RuleParamType `json:"type"`
	Default  string        `json:"default"`
	Min      *float64      `json:"min,omitempty"`
	Max      *float64      `json:"max,omitempty"`
	Options  []string      `json:"options,omitempty"`
	HelpText string        `json:"helpText,omitempty"`
}

type RuleSetting struct {
	Action RuleAction        `json:"action"`
	Params map[string]string `json:"params,omitempty"`
}

type RuleSettings map[RuleCode]RuleSetting

type RuleContext struct {
	Profile         *Profile
	Now             int64
	Subject         SubjectType
	BrokerAuthority bool
	AuthorityExempt bool
	NotFound        bool
	params          map[string]string
	spec            *RuleDefinition
}

type evaluateFunc func(ctx *RuleContext) (hit bool, message string)

type RuleDefinition struct {
	Code              RuleCode        `json:"code"`
	Label             string          `json:"label"`
	Description       string          `json:"description"`
	Category          Section         `json:"category"`
	DefaultAction     RuleAction      `json:"defaultAction"`
	RecommendedAction RuleAction      `json:"recommendedAction"`
	RequiredSections  []Section       `json:"requiredSections"`
	Params            []RuleParamSpec `json:"params"`
	Subjects          []SubjectType   `json:"subjects"`
	GateRelevant      bool            `json:"gateRelevant"`
	evaluate          evaluateFunc
}

func (d *RuleDefinition) AppliesTo(subject SubjectType) bool {
	return len(d.Subjects) == 0 || slices.Contains(d.Subjects, subject)
}

func (d *RuleDefinition) paramSpec(key string) *RuleParamSpec {
	for idx := range d.Params {
		if d.Params[idx].Key == key {
			return &d.Params[idx]
		}
	}
	return nil
}

type Finding struct {
	Code              RuleCode   `json:"code"`
	Category          Section    `json:"category"`
	Action            RuleAction `json:"action"`
	Severity          Severity   `json:"severity"`
	Message           string     `json:"message"`
	Unverifiable      bool       `json:"unverifiable"`
	Unconfirmed       bool       `json:"unconfirmed"`
	Overridden        bool       `json:"overridden"`
	OverrideID        pulid.ID   `json:"overrideId,omitempty"`
	OverrideExpiresAt *int64     `json:"overrideExpiresAt,omitempty"`
}

func (f *Finding) IsBlocking() bool {
	return f.Action == RuleActionBlock && !f.Unverifiable && !f.Overridden
}

func (f *Finding) IsAdvisory() bool {
	if f.Unverifiable {
		return false
	}
	return f.Action == RuleActionWarn || (f.Action == RuleActionBlock && f.Overridden)
}

type OverrideRef struct {
	ID        pulid.ID `json:"id"`
	RuleCode  RuleCode `json:"ruleCode"`
	ExpiresAt int64    `json:"expiresAt"`
}

func (c *RuleContext) paramString(key string) string {
	if v, ok := c.params[key]; ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if spec := c.spec.paramSpec(key); spec != nil {
		return spec.Default
	}
	return ""
}

func (c *RuleContext) paramInt(key string) int {
	v, err := strconv.Atoi(c.paramString(key))
	if err != nil {
		if spec := c.spec.paramSpec(key); spec != nil {
			fallback, _ := strconv.Atoi(spec.Default)
			return fallback
		}
		return 0
	}
	return v
}

func (c *RuleContext) paramFloat(key string) float64 {
	v, err := strconv.ParseFloat(c.paramString(key), 64)
	if err != nil {
		return 0
	}
	return v
}

func (c *RuleContext) paramDecimal(key string) (decimal.Decimal, bool) {
	raw := c.paramString(key)
	if raw == "" {
		return decimal.Zero, false
	}
	v, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, false
	}
	return v, true
}

func (c *RuleContext) paramList(key string) []string {
	raw := c.paramString(key)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func ResolveRuleSetting(settings RuleSettings, def *RuleDefinition) RuleSetting {
	if settings != nil {
		if setting, ok := settings[def.Code]; ok && setting.Action.IsValid() {
			return setting
		}
	}
	return RuleSetting{Action: def.DefaultAction}
}

type EvaluateInput struct {
	Profile         *Profile
	Now             int64
	Subject         SubjectType
	BrokerAuthority bool
	AuthorityExempt bool
	NotFound        bool
	Settings        RuleSettings
	Overrides       []OverrideRef
}

func EvaluateFindings(in *EvaluateInput) []Finding {
	catalog := Catalog()
	findings := make([]Finding, 0, 4)

	for idx := range catalog {
		def := &catalog[idx]
		if !def.AppliesTo(in.Subject) {
			continue
		}
		setting := ResolveRuleSetting(in.Settings, def)
		if setting.Action == RuleActionOff {
			continue
		}

		ctx := &RuleContext{
			Profile:         in.Profile,
			Now:             in.Now,
			Subject:         in.Subject,
			BrokerAuthority: in.BrokerAuthority,
			AuthorityExempt: in.AuthorityExempt,
			NotFound:        in.NotFound,
			params:          setting.Params,
			spec:            def,
		}

		finding := Finding{
			Code:     def.Code,
			Category: def.Category,
			Action:   setting.Action,
			Severity: setting.Action.Severity(),
		}

		if def.Code != RuleIdentityNotFound {
			if in.NotFound {
				continue
			}
			if !in.Profile.CoversAll(def.RequiredSections) {
				finding.Unverifiable = true
				finding.Severity = SeverityInfo
				finding.Message = def.Label + " could not be checked with the current provider"
				findings = append(findings, finding)
				continue
			}
		}

		hit, message := def.evaluate(ctx)
		if !hit {
			continue
		}
		finding.Message = message
		applyOverride(&finding, in.Overrides, in.Now)
		findings = append(findings, finding)
	}

	return findings
}

func applyOverride(finding *Finding, overrides []OverrideRef, now int64) {
	if finding.Action != RuleActionBlock && finding.Action != RuleActionWarn {
		return
	}
	for idx := range overrides {
		override := &overrides[idx]
		if override.RuleCode != finding.Code || override.ExpiresAt <= now {
			continue
		}
		finding.Overridden = true
		finding.OverrideID = override.ID
		expires := override.ExpiresAt
		finding.OverrideExpiresAt = &expires
		return
	}
}

func (f *Finding) MarkUnconfirmed(confirmingProvider string) {
	if f.Action != RuleActionBlock {
		return
	}
	f.Action = RuleActionWarn
	f.Severity = RuleActionWarn.Severity()
	f.Unconfirmed = true
	f.Message += " (not confirmed by " + confirmingProvider + ")"
}

func CarryUnconfirmed(stored, recomputed []Finding) []Finding {
	unconfirmed := make(map[RuleCode]Finding, len(stored))
	for idx := range stored {
		if stored[idx].Unconfirmed {
			unconfirmed[stored[idx].Code] = stored[idx]
		}
	}
	if len(unconfirmed) == 0 {
		return recomputed
	}
	for idx := range recomputed {
		prior, ok := unconfirmed[recomputed[idx].Code]
		if !ok || recomputed[idx].Action != RuleActionBlock {
			continue
		}
		recomputed[idx].Action = RuleActionWarn
		recomputed[idx].Severity = prior.Severity
		recomputed[idx].Message = prior.Message
		recomputed[idx].Unconfirmed = true
	}
	return recomputed
}

func BlockingCodes(findings []Finding) []string {
	codes := make([]string, 0, len(findings))
	for idx := range findings {
		if findings[idx].IsBlocking() {
			codes = append(codes, findings[idx].Code.String())
		}
	}
	return codes
}

func AdvisoryCodes(findings []Finding) []string {
	codes := make([]string, 0, len(findings))
	for idx := range findings {
		if findings[idx].IsAdvisory() {
			codes = append(codes, findings[idx].Code.String())
		}
	}
	return codes
}

func NewBlockingCodes(prior, current []Finding) []Finding {
	priorCodes := make(map[RuleCode]struct{}, len(prior))
	for idx := range prior {
		if prior[idx].IsBlocking() {
			priorCodes[prior[idx].Code] = struct{}{}
		}
	}
	added := make([]Finding, 0, len(current))
	for idx := range current {
		if !current[idx].IsBlocking() {
			continue
		}
		if _, ok := priorCodes[current[idx].Code]; !ok {
			added = append(added, current[idx])
		}
	}
	return added
}

func NewlyRaised(prior, current []Finding) []Finding {
	priorCodes := make(map[RuleCode]RuleAction, len(prior))
	for idx := range prior {
		if !prior[idx].Unverifiable {
			priorCodes[prior[idx].Code] = prior[idx].Action
		}
	}
	raised := make([]Finding, 0, len(current))
	for idx := range current {
		finding := current[idx]
		if finding.Unverifiable || finding.Overridden {
			continue
		}
		if action, ok := priorCodes[finding.Code]; ok && action == finding.Action {
			continue
		}
		raised = append(raised, finding)
	}
	return raised
}
