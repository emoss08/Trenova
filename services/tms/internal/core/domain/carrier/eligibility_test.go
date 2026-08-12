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
