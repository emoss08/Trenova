package watchtowerservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type scopedControlRepo struct {
	dispatchcontrolports.DispatchControlRepository

	control *dispatchcontrol.DispatchControl
	err     error
	scopes  []dbscope.Scope
}

func (s *scopedControlRepo) GetByOrgID(
	ctx context.Context,
	_ dispatchcontrolports.GetDispatchControlRequest,
) (*dispatchcontrol.DispatchControl, error) {
	s.scopes = append(s.scopes, dbscope.From(ctx))

	return s.control, s.err
}

func TestPolicy_ReadsTheTenantsDispatchControlUnderItsOwnScope(t *testing.T) {
	t.Parallel()

	controls := &scopedControlRepo{
		control: &dispatchcontrol.DispatchControl{ExceptionDueCriticalMinutes: 45},
	}
	router := NewRouter(RouterParams{Logger: zap.NewNop(), Controls: controls})
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	sweep := dbscope.WithSystem(t.Context(), "sweep watchtower attention")

	for range 2 {
		policy := router.Policy(sweep, tenant)
		assert.Equal(t, int32(45), policy.DueCriticalMinutes)
	}

	require.Len(t, controls.scopes, 2, "every call reads the repository's cached row")
	for _, scope := range controls.scopes {
		scoped, ok := scope.Tenant()
		require.True(t, ok, "the read runs in the tenant's scope, not the caller's system scope")
		assert.Equal(t, tenant.OrgID, scoped.OrganizationID)
		assert.Equal(t, tenant.BuID, scoped.BusinessUnitID)
	}
}

func TestPolicy_FallsBackToTheDefaultsWhenTheControlCannotBeRead(t *testing.T) {
	t.Parallel()

	router := NewRouter(RouterParams{
		Logger:   zap.NewNop(),
		Controls: &scopedControlRepo{err: errors.New("database unavailable")},
	})

	policy := router.Policy(t.Context(), pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	})

	assert.Equal(t, (*dispatchcontrol.DispatchControl)(nil).ExceptionPolicy(), policy)
}
