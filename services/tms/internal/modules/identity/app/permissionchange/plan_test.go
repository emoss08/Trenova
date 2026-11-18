package permissionchange

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordedChanges struct {
	organizations []pulid.ID
}

func (r *recordedChanges) OrganizationChanged(_ context.Context, tenant pagination.TenantInfo) {
	r.organizations = append(r.organizations, tenant.OrgID)
}

func (*recordedChanges) UserChanged(context.Context, pagination.TenantInfo, pulid.ID) {}

type pagedTenants struct {
	tenants []pagination.TenantInfo
}

func (p pagedTenants) ListTenantsPage(
	_ context.Context,
	after pulid.ID,
	limit int,
) ([]pagination.TenantInfo, error) {
	start := 0
	for i, tenant := range p.tenants {
		if tenant.OrgID == after {
			start = i + 1
		}
	}
	end := min(start+limit, len(p.tenants))
	return p.tenants[start:end], nil
}

func tenants(count int) []pagination.TenantInfo {
	out := make([]pagination.TenantInfo, count)
	for i := range out {
		out[i] = pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		}
	}
	return out
}

func TestAPlanChangeHasEveryOrganizationReadItsPermissionsAgain(t *testing.T) {
	t.Parallel()

	all := tenants(130)
	changes := &recordedChanges{}
	observer := &PlanObserver{notifier: changes, tenants: pagedTenants{tenants: all}, l: zap.NewNop()}

	err := observer.PlanChanged(t.Context(), managedfeatures.Replace(
		&managedfeatures.State{ID: managedfeatures.CurrentStateID},
		&managedfeatures.Received{
			Features: []platformcatalog.FeatureKey{platformcatalog.FeatureAIDesk},
		},
		1,
	))
	require.NoError(t, err)

	want := make([]pulid.ID, len(all))
	for i := range all {
		want[i] = all[i].OrgID
	}
	assert.Equal(t, want, changes.organizations)
}

func TestAnUnchangedPlanDisturbsNobody(t *testing.T) {
	t.Parallel()

	changes := &recordedChanges{}
	observer := &PlanObserver{notifier: changes, tenants: pagedTenants{tenants: tenants(3)}, l: zap.NewNop()}

	held := &managedfeatures.State{
		ID:       managedfeatures.CurrentStateID,
		Features: []platformcatalog.FeatureKey{platformcatalog.FeatureAIDesk},
	}
	err := observer.PlanChanged(t.Context(), managedfeatures.Replace(
		held,
		&managedfeatures.Received{
			Features: []platformcatalog.FeatureKey{platformcatalog.FeatureAIDesk},
		},
		2,
	))
	require.NoError(t, err)
	assert.Empty(t, changes.organizations)
}

func TestAPlanChangeThatOnlySchedulesAnEndDisturbsNobody(t *testing.T) {
	t.Parallel()

	changes := &recordedChanges{}
	observer := &PlanObserver{notifier: changes, tenants: pagedTenants{tenants: tenants(3)}, l: zap.NewNop()}

	held := &managedfeatures.State{
		ID:       managedfeatures.CurrentStateID,
		Features: []platformcatalog.FeatureKey{platformcatalog.FeatureAccounting},
	}
	endings := []managedfeatures.AddOnEnding{{AddOn: platformcatalog.AddOnAccounting, EndsAt: 99}}
	err := observer.PlanChanged(t.Context(), managedfeatures.Replace(
		held,
		&managedfeatures.Received{
			Features:     []platformcatalog.FeatureKey{platformcatalog.FeatureAccounting},
			AddOnEndings: &endings,
		},
		2,
	))
	require.NoError(t, err)
	assert.Empty(t, changes.organizations)
}
