package carrier_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
)

const day = int64(86400)

func qualifiedCarrier(now int64) *carrier.Carrier {
	return &carrier.Carrier{
		Status:           carrier.StatusActive,
		ComplianceStatus: carrier.ComplianceStatusQualified,
		InsurancePolicies: []*carrier.CarrierInsurancePolicy{
			{
				PolicyType:     carrier.InsurancePolicyTypeAutoLiability,
				PolicyNumber:   "AUTO-1",
				ExpirationDate: now + 365*day,
			},
			{
				PolicyType:     carrier.InsurancePolicyTypeCargoLiability,
				PolicyNumber:   "CARGO-1",
				ExpirationDate: now + 365*day,
			},
		},
	}
}

func TestEvaluateCarrierEligibility(t *testing.T) {
	now := timeutils.NowUnix()

	t.Run("qualified carrier with healthy insurance passes", func(t *testing.T) {
		result := carrier.EvaluateCarrierEligibility(qualifiedCarrier(now), now)
		assert.False(t, result.IsBlocked())
		assert.False(t, result.HasWarnings())
	})

	t.Run("inactive carrier is blocked", func(t *testing.T) {
		entity := qualifiedCarrier(now)
		entity.Status = carrier.StatusDoNotUse
		result := carrier.EvaluateCarrierEligibility(entity, now)
		assert.True(t, result.IsBlocked())
	})

	t.Run("unqualified carrier is blocked", func(t *testing.T) {
		entity := qualifiedCarrier(now)
		entity.ComplianceStatus = carrier.ComplianceStatusPending
		result := carrier.EvaluateCarrierEligibility(entity, now)
		assert.True(t, result.IsBlocked())
	})

	t.Run("missing required policy is blocked", func(t *testing.T) {
		entity := qualifiedCarrier(now)
		entity.InsurancePolicies = entity.InsurancePolicies[:1]
		result := carrier.EvaluateCarrierEligibility(entity, now)
		assert.True(t, result.IsBlocked())
	})

	t.Run("expired required policy is blocked", func(t *testing.T) {
		entity := qualifiedCarrier(now)
		entity.InsurancePolicies[0].ExpirationDate = now - day
		result := carrier.EvaluateCarrierEligibility(entity, now)
		assert.True(t, result.IsBlocked())
	})

	t.Run("policy expiring inside the window only warns", func(t *testing.T) {
		entity := qualifiedCarrier(now)
		entity.InsurancePolicies[1].ExpirationDate = now + 10*day
		result := carrier.EvaluateCarrierEligibility(entity, now)
		assert.False(t, result.IsBlocked())
		assert.True(t, result.HasWarnings())
	})

	t.Run("renewal supersedes an expiring policy", func(t *testing.T) {
		entity := qualifiedCarrier(now)
		entity.InsurancePolicies = append(entity.InsurancePolicies,
			&carrier.CarrierInsurancePolicy{
				PolicyType:     carrier.InsurancePolicyTypeAutoLiability,
				PolicyNumber:   "AUTO-OLD",
				ExpirationDate: now + 5*day,
			})
		result := carrier.EvaluateCarrierEligibility(entity, now)
		assert.False(t, result.IsBlocked())
		assert.False(t, result.HasWarnings())
	})
}

func TestEvaluateEligibility_IntelGate(t *testing.T) {
	now := timeutils.NowUnix()

	evaluate := func(gate *carrier.IntelGate) carrier.EligibilityResult {
		return carrier.EvaluateEligibility(carrier.EligibilityInput{
			Carrier: qualifiedCarrier(now),
			Now:     now,
			Intel:   gate,
		})
	}

	t.Run("nil gate leaves the record result unchanged", func(t *testing.T) {
		result := evaluate(nil)
		assert.False(t, result.IsBlocked())
		assert.False(t, result.HasAdvisories())
		assert.Empty(t, result.Findings)
	})

	t.Run("blocking finding blocks and is attributed to intelligence", func(t *testing.T) {
		result := evaluate(&carrier.IntelGate{
			Evaluated: true,
			FetchedAt: now,
			Findings: []carrier.IntelFinding{
				{Code: "authority.revoked", Action: carrier.IntelActionBlock, Message: "revoked"},
			},
		})
		assert.True(t, result.IsBlocked())
		assert.False(t, result.HasWarnings())
		assert.Equal(t, carrier.EligibilitySourceIntelligence, result.Findings[0].Source)
		assert.Equal(t, "authority.revoked", result.Findings[0].Code)
	})

	t.Run("warn and overridden block become advisories that never need an override", func(t *testing.T) {
		result := evaluate(&carrier.IntelGate{
			Evaluated: true,
			FetchedAt: now,
			Findings: []carrier.IntelFinding{
				{Code: "safety.iss_high", Action: carrier.IntelActionWarn, Message: "iss"},
				{Code: "safety.oos_order", Action: carrier.IntelActionBlock, Message: "oos", Overridden: true},
				{Code: "operations.mcs150_stale", Action: carrier.IntelActionNotify, Message: "mcs"},
			},
		})
		assert.False(t, result.IsBlocked())
		assert.False(t, result.HasWarnings())
		assert.Len(t, result.Advisories, 2)
		for _, f := range result.Findings {
			assert.False(t, f.RequiresOverride)
		}
	})

	t.Run("fail closed blocks past the hard max age", func(t *testing.T) {
		result := evaluate(&carrier.IntelGate{
			Evaluated:      true,
			FetchedAt:      now - 30*day,
			Stale:          true,
			PastHardMaxAge: true,
			OutagePolicy:   carrier.IntelOutagePolicyFailClosed,
		})
		assert.True(t, result.IsBlocked())
	})

	t.Run("fail open only advises when stale", func(t *testing.T) {
		result := evaluate(&carrier.IntelGate{
			Evaluated:      true,
			FetchedAt:      now - 30*day,
			Stale:          true,
			PastHardMaxAge: true,
			OutagePolicy:   carrier.IntelOutagePolicyFailOpen,
		})
		assert.False(t, result.IsBlocked())
		assert.True(t, result.HasAdvisories())
	})

	t.Run("fail closed with no intelligence ever fetched blocks when provider is down", func(t *testing.T) {
		result := evaluate(&carrier.IntelGate{
			Evaluated:           true,
			ProviderUnavailable: true,
			OutagePolicy:        carrier.IntelOutagePolicyFailClosed,
		})
		assert.True(t, result.IsBlocked())
	})

	t.Run("last known blocks persist during an outage", func(t *testing.T) {
		result := evaluate(&carrier.IntelGate{
			Evaluated:           true,
			FetchedAt:           now - day,
			ProviderUnavailable: true,
			OutagePolicy:        carrier.IntelOutagePolicyFailOpen,
			Findings: []carrier.IntelFinding{
				{Code: "insurance.none_on_file", Action: carrier.IntelActionBlock, Message: "none"},
			},
		})
		assert.True(t, result.IsBlocked())
		assert.True(t, result.HasAdvisories())
	})
}
