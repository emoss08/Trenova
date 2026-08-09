package carrierassignmentservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
)

const insuranceWarningWindowSeconds = int64(30 * 24 * 60 * 60)

type EligibilityResult struct {
	Blockers []string `json:"blockers"`
	Warnings []string `json:"warnings"`
}

func (r EligibilityResult) IsBlocked() bool {
	return len(r.Blockers) > 0
}

func (r EligibilityResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// EvaluateCarrierEligibility applies the coverage gate: an inactive,
// unqualified, or uninsured carrier hard-blocks assignment, while required
// coverage expiring inside the warning window is dispatcher-overridable.
func EvaluateCarrierEligibility(entity *carrier.Carrier, now int64) EligibilityResult {
	result := EligibilityResult{}

	if !entity.IsActive() {
		result.Blockers = append(result.Blockers,
			fmt.Sprintf("Carrier status is %s", entity.Status))
	}

	if !entity.IsQualified() {
		result.Blockers = append(result.Blockers,
			fmt.Sprintf("Carrier compliance status is %s", entity.ComplianceStatus))
	}

	for _, policyType := range carrier.RequiredInsurancePolicyTypes {
		policy := entity.RequiredPolicy(policyType)
		switch {
		case policy == nil:
			result.Blockers = append(result.Blockers,
				fmt.Sprintf("Carrier has no %s insurance policy on file", policyType))
		case policy.IsExpired(now):
			result.Blockers = append(result.Blockers,
				fmt.Sprintf("Carrier %s insurance policy %s is expired",
					policyType, policy.PolicyNumber))
		case policy.ExpiresWithin(now, insuranceWarningWindowSeconds):
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Carrier %s insurance policy %s expires within 30 days",
					policyType, policy.PolicyNumber))
		}
	}

	return result
}
