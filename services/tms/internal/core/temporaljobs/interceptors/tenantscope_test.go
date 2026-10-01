package interceptors

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/interceptor"
)

type recordingActivityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	scope dbscope.Scope
}

func (r *recordingActivityInbound) ExecuteActivity(
	ctx context.Context,
	_ *interceptor.ExecuteActivityInput,
) (any, error) {
	r.scope = dbscope.From(ctx)
	return nil, nil
}

func TestTenantScopeInterceptor_BindsTheActivityTenant(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	type payload struct {
		ShipmentID pulid.ID
		TenantInfo pagination.TenantInfo
	}

	next := &recordingActivityInbound{}
	inbound := NewTenantScopeInterceptor().InterceptActivity(t.Context(), next)

	_, err := inbound.ExecuteActivity(t.Context(), &interceptor.ExecuteActivityInput{
		Args: []any{&payload{ShipmentID: pulid.MustNew("shp_"), TenantInfo: tenant}},
	})
	require.NoError(t, err)

	got, ok := next.scope.Tenant()
	require.True(t, ok)
	assert.Equal(t, dbscope.Tenant{OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID, UserID: tenant.UserID}, got)
}

func TestTenantScopeInterceptor_LeavesTenantlessActivitiesUnscoped(t *testing.T) {
	t.Parallel()

	next := &recordingActivityInbound{}
	inbound := NewTenantScopeInterceptor().InterceptActivity(t.Context(), next)

	_, err := inbound.ExecuteActivity(t.Context(), &interceptor.ExecuteActivityInput{
		Args: []any{"cursor", 50},
	})
	require.NoError(t, err)
	assert.Equal(t, dbscope.KindNone, next.scope.Kind())
}
