package dispatchstyleresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchlearningports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingService struct {
	got   *dispatchlearningports.GetStyleRequest
	reset *dispatchlearningports.ResetStyleRequest
}

func (s *recordingService) Get(
	_ context.Context,
	req *dispatchlearningports.GetStyleRequest,
) (*dispatchlearningports.Style, error) {
	s.got = req
	return &dispatchlearningports.Style{Enabled: true}, nil
}

func (s *recordingService) Reset(
	_ context.Context,
	req *dispatchlearningports.ResetStyleRequest,
) (*dispatchlearningports.Style, error) {
	s.reset = req
	return &dispatchlearningports.Style{}, nil
}

func TestDispatchStyle_ChecksTheDispatchControlPermission(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"dispatchStyle",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).DispatchStyle(ctx)
				return err
			},
		},
		{
			"resetDispatchStyle",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ResetDispatchStyle(ctx)
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			service := &recordingService{}
			deps := &Deps{
				Core:                 &resolvercore.Core{PermissionEngine: engine},
				DispatchStyleService: service,
			}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceDispatchControl,
				call.operation,
			)
			assert.Nil(t, service.got)
			assert.Nil(t, service.reset, "a refused caller never reaches the service")
		})
	}
}

func TestResetDispatchStyle_ActsForTheCallerInTheirOrganization(t *testing.T) {
	t.Parallel()

	service := &recordingService{}
	deps := &Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &mocks.AllowAllPermissionEngine{},
		},
		DispatchStyleService: service,
	}
	authCtx := resolvertest.UserAuthContext()

	_, err := (&MutationResolver{deps}).ResetDispatchStyle(resolvertest.AuthedContext(t, authCtx))

	require.NoError(t, err)
	require.NotNil(t, service.reset)
	assert.Equal(t, authCtx.OrganizationID, service.reset.TenantInfo.OrgID)
	assert.Equal(t, authCtx.BusinessUnitID, service.reset.TenantInfo.BuID)
	require.NotNil(t, service.reset.Actor)
	assert.Equal(t, authCtx.UserID, service.reset.Actor.UserID)
}
