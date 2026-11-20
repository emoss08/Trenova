package detentionresolver

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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
	}, pulid.Nil)

	require.NotNil(t, model)
	assert.False(t, model.IsOrganizationDefault)
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
		StopTypes:           []shipmentvocab.StopType{shipmentvocab.StopTypeDelivery},
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
	}, policyID)

	require.NotNil(t, model)
	assert.True(t, model.IsOrganizationDefault, "shipment control names this policy")
	require.Len(t, model.Tiers, 1)
	assert.Equal(t, "First 4 hours", model.Tiers[0].Label)
	require.Len(t, model.StopTypes, 1)
}

func TestDetentionPolicyFromInput_RefusesANumberThatWouldWrap(t *testing.T) {
	t.Parallel()

	priority := math.MaxInt16 + 1
	_, err := detentionPolicyFromInput(&gqlmodel.DetentionPolicyInput{
		Name:                "Standard detention terms",
		Code:                "DET-STD",
		AccessorialChargeID: pulid.MustNew("acc_").String(),
		Priority:            &priority,
	}, pulid.MustNew("dtp_"), pagination.TenantInfo{})

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "priority", valErr.Field())
	assert.Equal(t, errortypes.ErrInvalid, valErr.Code)
	assert.Equal(t, "Must be between {0} and {1}", valErr.Message)
	assert.Equal(t, []any{int64(math.MinInt16), int64(math.MaxInt16)}, valErr.Args)
}

func TestDetentionTiersFromInput_NamesTheTierThatWouldWrap(t *testing.T) {
	t.Parallel()

	tiers, err := detentionTiersFromInput([]*gqlmodel.DetentionTierInput{
		{FromMinute: 0, Rate: "50"},
		{FromMinute: 4294967311, Rate: "75"},
	}, pagination.TenantInfo{})

	assert.Nil(t, tiers)
	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "tiers[1].fromMinute", valErr.Field())
	assert.Equal(t, []any{int64(math.MinInt32), int64(math.MaxInt32)}, valErr.Args)
}
