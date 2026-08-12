package carrier_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validCarrier() *carrier.Carrier {
	return &carrier.Carrier{
		Status:           carrier.StatusActive,
		Code:             "CARR001",
		Name:             "Test Carrier",
		CarrierType:      carrier.TypeCommon,
		DOTNumber:        "1234567",
		MCNumber:         "654321",
		SCAC:             "TSTC",
		ComplianceStatus: carrier.ComplianceStatusQualified,
		SafetyRating:     carrier.SafetyRatingSatisfactory,
		PaymentMethod:    carrier.PaymentMethodCheck,
		PaymentTermDays:  30,
	}
}

func TestCarrierValidate(t *testing.T) {
	t.Run("valid carrier passes", func(t *testing.T) {
		multiErr := errortypes.NewMultiError()
		validCarrier().Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), "unexpected errors: %v", multiErr)
	})

	t.Run("defaults are applied before validation", func(t *testing.T) {
		entity := &carrier.Carrier{
			Code: "CARR002",
			Name: "Default Carrier",
		}
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), "unexpected errors: %v", multiErr)
		assert.Equal(t, carrier.StatusActive, entity.Status)
		assert.Equal(t, carrier.TypeCommon, entity.CarrierType)
		assert.Equal(t, carrier.ComplianceStatusPending, entity.ComplianceStatus)
		assert.Equal(t, carrier.SafetyRatingNotRated, entity.SafetyRating)
		assert.Equal(t, carrier.PaymentMethodCheck, entity.PaymentMethod)
		assert.Equal(t, 30, entity.PaymentTermDays)
	})

	t.Run("missing required fields fail", func(t *testing.T) {
		entity := validCarrier()
		entity.Code = ""
		entity.Name = ""
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("invalid SCAC fails", func(t *testing.T) {
		entity := validCarrier()
		entity.SCAC = "ts1"
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("non-numeric DOT number fails", func(t *testing.T) {
		entity := validCarrier()
		entity.DOTNumber = "12A4567"
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("tax ID without type fails", func(t *testing.T) {
		entity := validCarrier()
		entity.TaxID = "12-3456789"
		entity.TaxIDType = nil
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("disqualified without reason fails", func(t *testing.T) {
		entity := validCarrier()
		entity.ComplianceStatus = carrier.ComplianceStatusDisqualified
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("invalid child contact is reported with index", func(t *testing.T) {
		entity := validCarrier()
		entity.Contacts = []*carrier.CarrierContact{
			{Name: ""},
		}
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("rate confirmation recipient requires email", func(t *testing.T) {
		entity := validCarrier()
		entity.Contacts = []*carrier.CarrierContact{
			{Name: "Dispatcher", ReceivesRateConfirmations: true},
		}
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})
}

func TestInsurancePolicyValidate(t *testing.T) {
	now := timeutils.NowUnix()

	t.Run("valid policy passes", func(t *testing.T) {
		policy := &carrier.CarrierInsurancePolicy{
			PolicyType:     carrier.InsurancePolicyTypeAutoLiability,
			PolicyNumber:   "POL-100",
			ProviderName:   "Acme Insurance",
			CoverageAmount: decimal.NewFromInt(1000000),
			EffectiveDate:  now - 86400,
			ExpirationDate: now + 86400*365,
		}
		multiErr := errortypes.NewMultiError()
		policy.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), "unexpected errors: %v", multiErr)
	})

	t.Run("expiration before effective fails", func(t *testing.T) {
		policy := &carrier.CarrierInsurancePolicy{
			PolicyType:     carrier.InsurancePolicyTypeCargoLiability,
			PolicyNumber:   "POL-101",
			ProviderName:   "Acme Insurance",
			CoverageAmount: decimal.NewFromInt(100000),
			EffectiveDate:  now,
			ExpirationDate: now - 1,
		}
		multiErr := errortypes.NewMultiError()
		policy.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})

	t.Run("negative coverage fails", func(t *testing.T) {
		policy := &carrier.CarrierInsurancePolicy{
			PolicyType:     carrier.InsurancePolicyTypeCargoLiability,
			PolicyNumber:   "POL-102",
			ProviderName:   "Acme Insurance",
			CoverageAmount: decimal.NewFromInt(-1),
			EffectiveDate:  now,
			ExpirationDate: now + 86400,
		}
		multiErr := errortypes.NewMultiError()
		policy.Validate(multiErr)
		assert.True(t, multiErr.HasErrors())
	})
}

func TestInsuranceExpiryHelpers(t *testing.T) {
	now := timeutils.NowUnix()
	day := int64(86400)

	expired := &carrier.CarrierInsurancePolicy{ExpirationDate: now - day}
	expiringSoon := &carrier.CarrierInsurancePolicy{ExpirationDate: now + 10*day}
	healthy := &carrier.CarrierInsurancePolicy{ExpirationDate: now + 90*day}

	assert.True(t, expired.IsExpired(now))
	assert.False(t, expiringSoon.IsExpired(now))
	assert.True(t, expiringSoon.ExpiresWithin(now, 30*day))
	assert.False(t, healthy.ExpiresWithin(now, 30*day))
	assert.False(t, expired.ExpiresWithin(now, 30*day))
}

func TestRequiredPolicy(t *testing.T) {
	now := timeutils.NowUnix()
	day := int64(86400)

	entity := validCarrier()
	older := &carrier.CarrierInsurancePolicy{
		PolicyType:     carrier.InsurancePolicyTypeAutoLiability,
		PolicyNumber:   "OLD",
		ExpirationDate: now + 10*day,
	}
	renewal := &carrier.CarrierInsurancePolicy{
		PolicyType:     carrier.InsurancePolicyTypeAutoLiability,
		PolicyNumber:   "NEW",
		ExpirationDate: now + 400*day,
	}
	cargo := &carrier.CarrierInsurancePolicy{
		PolicyType:     carrier.InsurancePolicyTypeCargoLiability,
		PolicyNumber:   "CARGO",
		ExpirationDate: now + 100*day,
	}
	entity.InsurancePolicies = []*carrier.CarrierInsurancePolicy{older, renewal, cargo}

	got := entity.RequiredPolicy(carrier.InsurancePolicyTypeAutoLiability)
	require.NotNil(t, got)
	assert.Equal(t, "NEW", got.PolicyNumber)

	assert.Nil(t, entity.RequiredPolicy(carrier.InsurancePolicyTypeUmbrella))
}
