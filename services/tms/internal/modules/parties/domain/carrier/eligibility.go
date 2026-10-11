package carrier

import "fmt"

const insuranceWarningWindowSeconds = int64(30 * 24 * 60 * 60)

type IntelAction string

const (
	IntelActionBlock  = IntelAction("Block")
	IntelActionWarn   = IntelAction("Warn")
	IntelActionNotify = IntelAction("Notify")
)

type IntelOutagePolicy string

const (
	IntelOutagePolicyFailOpen   = IntelOutagePolicy("FailOpen")
	IntelOutagePolicyFailClosed = IntelOutagePolicy("FailClosed")
)

type IntelFinding struct {
	Code              string      `json:"code"`
	Category          string      `json:"category"`
	Action            IntelAction `json:"action"`
	Message           string      `json:"message"`
	Overridden        bool        `json:"overridden"`
	OverrideExpiresAt *int64      `json:"overrideExpiresAt,omitempty"`
}

type IntelGate struct {
	Evaluated           bool              `json:"evaluated"`
	Provider            string            `json:"provider"`
	FetchedAt           int64             `json:"fetchedAt"`
	Stale               bool              `json:"stale"`
	PastHardMaxAge      bool              `json:"pastHardMaxAge"`
	ProviderUnavailable bool              `json:"providerUnavailable"`
	NotFound            bool              `json:"notFound"`
	OutagePolicy        IntelOutagePolicy `json:"outagePolicy"`
	Findings            []IntelFinding    `json:"findings"`
}

type EligibilityFinding struct {
	Code             string `json:"code"`
	Source           string `json:"source"`
	Severity         string `json:"severity"`
	Message          string `json:"message"`
	RequiresOverride bool   `json:"requiresOverride"`
}

const (
	EligibilitySourceRecord       = "Record"
	EligibilitySourceIntelligence = "Intelligence"

	EligibilitySeverityBlocker  = "Blocker"
	EligibilitySeverityWarning  = "Warning"
	EligibilitySeverityAdvisory = "Advisory"
)

type EligibilityResult struct {
	Blockers   []string             `json:"blockers"`
	Warnings   []string             `json:"warnings"`
	Advisories []string             `json:"advisories"`
	Findings   []EligibilityFinding `json:"findings"`
}

func (r EligibilityResult) IsBlocked() bool {
	return len(r.Blockers) > 0
}

func (r EligibilityResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

func (r EligibilityResult) HasAdvisories() bool {
	return len(r.Advisories) > 0
}

func (r *EligibilityResult) addBlocker(code, source, message string) {
	r.Blockers = append(r.Blockers, message)
	r.Findings = append(r.Findings, EligibilityFinding{
		Code:     code,
		Source:   source,
		Severity: EligibilitySeverityBlocker,
		Message:  message,
	})
}

func (r *EligibilityResult) addWarning(code, source, message string) {
	r.Warnings = append(r.Warnings, message)
	r.Findings = append(r.Findings, EligibilityFinding{
		Code:             code,
		Source:           source,
		Severity:         EligibilitySeverityWarning,
		Message:          message,
		RequiresOverride: true,
	})
}

func (r *EligibilityResult) addAdvisory(code, source, message string) {
	r.Advisories = append(r.Advisories, message)
	r.Findings = append(r.Findings, EligibilityFinding{
		Code:     code,
		Source:   source,
		Severity: EligibilitySeverityAdvisory,
		Message:  message,
	})
}

type EligibilityInput struct {
	Carrier *Carrier
	Now     int64
	Intel   *IntelGate
}

func EvaluateCarrierEligibility(entity *Carrier, now int64) EligibilityResult {
	return EvaluateEligibility(EligibilityInput{Carrier: entity, Now: now})
}

func EvaluateEligibility(in EligibilityInput) EligibilityResult {
	result := EligibilityResult{
		Blockers:   []string{},
		Warnings:   []string{},
		Advisories: []string{},
		Findings:   []EligibilityFinding{},
	}
	entity := in.Carrier

	if !entity.IsActive() {
		result.addBlocker("record.status", EligibilitySourceRecord,
			fmt.Sprintf("Carrier status is %s", entity.Status))
	}

	if !entity.IsQualified() {
		result.addBlocker("record.compliance_status", EligibilitySourceRecord,
			fmt.Sprintf("Carrier compliance status is %s", entity.ComplianceStatus))
	}

	for _, policyType := range RequiredInsurancePolicyTypes {
		policy := entity.RequiredPolicy(policyType)
		code := "record.insurance." + policyType.String()
		switch {
		case policy == nil:
			result.addBlocker(code, EligibilitySourceRecord,
				fmt.Sprintf("Carrier has no %s insurance policy on file", policyType))
		case policy.IsExpired(in.Now):
			result.addBlocker(code, EligibilitySourceRecord,
				fmt.Sprintf("Carrier %s insurance policy %s is expired",
					policyType, policy.PolicyNumber))
		case policy.ExpiresWithin(in.Now, insuranceWarningWindowSeconds):
			result.addWarning(code, EligibilitySourceRecord,
				fmt.Sprintf("Carrier %s insurance policy %s expires within 30 days",
					policyType, policy.PolicyNumber))
		}
	}

	applyIntelGate(&result, in.Intel)

	return result
}

func applyIntelGate(result *EligibilityResult, gate *IntelGate) {
	if gate == nil || !gate.Evaluated {
		return
	}

	failClosed := gate.OutagePolicy == IntelOutagePolicyFailClosed

	switch {
	case gate.PastHardMaxAge && failClosed:
		result.addBlocker("intel.snapshot_stale", EligibilitySourceIntelligence,
			"Carrier intelligence is older than the maximum allowed age")
	case gate.ProviderUnavailable && failClosed && gate.FetchedAt == 0:
		result.addBlocker("intel.provider_unavailable", EligibilitySourceIntelligence,
			"Carrier intelligence could not be retrieved and the organization requires it")
	case gate.ProviderUnavailable:
		result.addAdvisory("intel.provider_unavailable", EligibilitySourceIntelligence,
			"Carrier intelligence provider is unavailable; showing the last known result")
	case gate.Stale:
		result.addAdvisory("intel.snapshot_stale", EligibilitySourceIntelligence,
			"Carrier intelligence has not been refreshed recently")
	}

	for _, finding := range gate.Findings {
		switch {
		case finding.Action == IntelActionBlock && !finding.Overridden:
			result.addBlocker(finding.Code, EligibilitySourceIntelligence, finding.Message)
		case finding.Action == IntelActionBlock && finding.Overridden:
			result.addAdvisory(finding.Code, EligibilitySourceIntelligence,
				finding.Message+" (overridden)")
		case finding.Action == IntelActionWarn:
			result.addAdvisory(finding.Code, EligibilitySourceIntelligence, finding.Message)
		}
	}
}
