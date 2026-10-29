package detentionresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A policy without a rate ladder has zero tiers, not an unknown number of them.
// Emitting nil put a null on the wire that every consumer had to guard.
func TestDetentionPolicyToModel_EmptyListsAreNotNull(t *testing.T) {
	t.Parallel()

	model := detentionPolicyToModel(&detentionpolicy.DetentionPolicy{
		ID:                  pulid.MustNew("dtp_"),
		OrganizationID:      pulid.MustNew("org_"),
		BusinessUnitID:      pulid.MustNew("bu_"),
		Name:                "Standard detention terms",
		Code:                "DET-STD",
		Status:              detentionpolicy.PolicyStatusActive,
		RateSource:          detentionpolicy.RateSourceAccessorial,
		AccessorialChargeID: pulid.MustNew("acc_"),
		Currency:            "USD",
	})

	require.NotNil(t, model)
	assert.NotNil(t, model.Tiers)
	assert.Empty(t, model.Tiers)
	assert.NotNil(t, model.StopTypes)
	assert.Empty(t, model.StopTypes)
}

func TestDetentionPolicyToModel_PopulatedLists(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	policyID := pulid.MustNew("dtp_")

	model := detentionPolicyToModel(&detentionpolicy.DetentionPolicy{
		ID:                  policyID,
		OrganizationID:      orgID,
		BusinessUnitID:      buID,
		Name:                "Acme negotiated detention",
		Code:                "DET-ACME",
		Status:              detentionpolicy.PolicyStatusActive,
		RateSource:          detentionpolicy.RateSourceTiers,
		AccessorialChargeID: pulid.MustNew("acc_"),
		Currency:            "USD",
		StopTypes:           []shipment.StopType{shipment.StopTypeDelivery},
		Tiers: []*detentionpolicy.DetentionPolicyTier{
			nil,
			{
				ID:                pulid.MustNew("dpt_"),
				OrganizationID:    orgID,
				BusinessUnitID:    buID,
				DetentionPolicyID: policyID,
				FromMinute:        0,
				Rate:              decimal.NewFromInt(60),
				RateUnit:          detentionpolicy.TierRateUnitHour,
				Label:             "First 4 hours",
			},
		},
	})

	require.NotNil(t, model)
	require.Len(t, model.Tiers, 1)
	assert.Equal(t, "First 4 hours", model.Tiers[0].Label)
	require.Len(t, model.StopTypes, 1)
}
