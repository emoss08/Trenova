package tableconfigurationresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tableconfigurationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/tableconfigurationports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/tableconfigurationportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func viewResolver(
	t *testing.T,
	engine permissionports.PermissionEngine,
) (*QueryResolver, *tableconfigurationportsmock.MockTableConfigurationRepository, context.Context) {
	t.Helper()

	repo := tableconfigurationportsmock.NewMockTableConfigurationRepository(t)
	deps := &Deps{
		Core: &resolvercore.Core{L: zap.NewNop(), PermissionEngine: engine},
		TableConfigurationService: tableconfigurationservice.New(tableconfigurationservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
	}
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  string(permissionports.PrincipalTypeUser),
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	return &QueryResolver{deps}, repo, ctx
}

func emptyPage() *pagination.CursorListResult[*tableconfiguration.TableConfiguration] {
	return &pagination.CursorListResult[*tableconfiguration.TableConfiguration]{}
}

func TestTableConfigurationsListsOnlyViewsOfTablesTheReaderCanRead(t *testing.T) {
	t.Parallel()

	resolver, repo, ctx := viewResolver(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceWorker.String()},
	})
	var asked *tableconfigurationports.ListTableConfigurationConnectionRequest
	repo.EXPECT().ListConnection(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *tableconfigurationports.ListTableConfigurationConnectionRequest,
		) (*pagination.CursorListResult[*tableconfiguration.TableConfiguration], error) {
			asked = req
			return emptyPage(), nil
		}).
		Twice()

	worker := permission.ResourceWorker.String()
	_, err := resolver.TableConfigurations(ctx, gqlmodel.DataTableConnectionInput{}, &worker, nil)
	require.NoError(t, err)
	require.NotNil(t, asked)
	assert.True(t, asked.LimitToResources)
	assert.Empty(t, asked.ReadableResources)

	_, err = resolver.TableConfigurations(ctx, gqlmodel.DataTableConnectionInput{}, nil, nil)
	require.NoError(t, err)
	assert.True(t, asked.LimitToResources)
	assert.Contains(t, asked.ReadableResources, permission.ResourceShipment.String())
	assert.NotContains(t, asked.ReadableResources, worker)
}

func TestTableConfigurationHidesAViewOfATableTheReaderCannotRead(t *testing.T) {
	t.Parallel()

	resolver, repo, ctx := viewResolver(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceWorker.String()},
	})
	id := pulid.MustNew("tc_")
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&tableconfiguration.TableConfiguration{
			ID:         id,
			Resource:   permission.ResourceWorker.String(),
			Visibility: tableconfiguration.VisibilityPublic,
		}, nil)

	entity, err := resolver.TableConfiguration(ctx, id.String())

	assert.Nil(t, entity)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestDefaultTableConfigurationIsNoneForATableTheReaderCannotRead(t *testing.T) {
	t.Parallel()

	resolver, _, ctx := viewResolver(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceWorker.String()},
	})

	entity, err := resolver.DefaultTableConfiguration(ctx, permission.ResourceWorker.String())

	require.NoError(t, err)
	assert.Nil(t, entity)
}
