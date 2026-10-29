package shipmentbillingtransfer

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/stretchr/testify/assert"
)

func TestBillingTransferPolicyViolation(t *testing.T) {
	t.Parallel()

	missingDocument := []shipmentbillingports.ShipmentBillingRequirement{
		{DocumentTypeName: "Proof of Delivery"},
	}
	rateFailure := []shipmentbillingports.ShipmentBillingValidation{
		{Code: "rate_variance_requires_action"},
	}

	tests := []struct {
		name      string
		readiness *shipmentbillingports.ShipmentBillingReadiness
		want      shipmentbillingports.BillingTransferFailureCode
	}{
		{
			name: "clean freight transfers",
			readiness: &shipmentbillingports.ShipmentBillingReadiness{
				Policy: shipmentbillingports.ShipmentBillingReadinessPolicy{
					ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
					RateValidationEnforcement:             tenant.EnforcementLevelBlock,
				},
			},
			want: "",
		},
		{
			name: "blocked requirements",
			readiness: &shipmentbillingports.ShipmentBillingReadiness{
				Policy: shipmentbillingports.ShipmentBillingReadinessPolicy{
					ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
				},
				MissingRequirements: missingDocument,
			},
			want: shipmentbillingports.BillingTransferFailureRequirementsUnmet,
		},
		{
			name: "blocked rate validation",
			readiness: &shipmentbillingports.ShipmentBillingReadiness{
				Policy: shipmentbillingports.ShipmentBillingReadinessPolicy{
					RateValidationEnforcement: tenant.EnforcementLevelBlock,
				},
				ValidationFailures: rateFailure,
			},
			want: shipmentbillingports.BillingTransferFailureRateValidation,
		},
		{
			name: "review items returned to operations",
			readiness: &shipmentbillingports.ShipmentBillingReadiness{
				Policy: shipmentbillingports.ShipmentBillingReadinessPolicy{
					RateValidationEnforcement:   tenant.EnforcementLevelRequireReview,
					BillingExceptionDisposition: billingcontrol.BillingExceptionDispositionReturnToOperations,
				},
				ValidationFailures: rateFailure,
			},
			want: shipmentbillingports.BillingTransferFailureReturnToOperations,
		},
		{
			name: "review items routed to billing review still transfer",
			readiness: &shipmentbillingports.ShipmentBillingReadiness{
				Policy: shipmentbillingports.ShipmentBillingReadinessPolicy{
					RateValidationEnforcement:   tenant.EnforcementLevelRequireReview,
					BillingExceptionDisposition: billingcontrol.BillingExceptionDispositionRouteToBillingReview,
				},
				ValidationFailures: rateFailure,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code, err := billingTransferPolicyViolation(tt.readiness)
			assert.Equal(t, tt.want, code)
			if tt.want == "" {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
		})
	}
}
