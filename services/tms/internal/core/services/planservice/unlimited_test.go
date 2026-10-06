package planservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/services/planservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func TestUnlimitedPlanService(t *testing.T) {
	t.Parallel()

	svc := planservice.NewUnlimited()
	ti := tenantInfo()

	assert.False(t, svc.EnforcesPlans())
	resolved, err := svc.Resolve(t.Context(), ti.OrgID, ti.BuID)
	require.NoError(t, err)
	assert.Equal(t, platformplan.OriginSelfHosted, resolved.Origin)
	assert.True(t, resolved.Plan.IsUnlimited())
	require.NoError(t, svc.RequireCapability(t.Context(), ti, platformplan.CapabilitySMS))
	require.NoError(t, svc.RequireWritable(t.Context(), ti))
	svc.Invalidate(ti.OrgID)
}

func TestNewIsUnlimited(t *testing.T) {
	t.Parallel()

	assert.IsType(t, &planservice.UnlimitedPlanService{}, planservice.New())
}
