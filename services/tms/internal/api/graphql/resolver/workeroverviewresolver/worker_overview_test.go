package workeroverviewresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeroverviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/internal/testutil/mocks/workerportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func grant(resources ...permission.Resource) map[string]bool {
	granted := make(map[string]bool, len(resources))
	for _, resource := range resources {
		granted[resource.String()+"|"+string(permission.OpRead)] = true
	}
	return granted
}

func userContext(t *testing.T) context.Context {
	t.Helper()
	userID := pulid.MustNew("usr_")
	return gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

type trainingOnly struct {
	summary *workertraining.WorkerTrainingSummary
}

func (f *trainingOnly) SummaryFor(
	_ context.Context, _ pagination.TenantInfo, _ *worker.Worker,
) (*workertraining.WorkerTrainingSummary, error) {
	return f.summary, nil
}

func TestOverviewSections_FollowEachSectionsOwnRead(t *testing.T) {
	t.Parallel()
	engine := &resolvertest.GrantingPermissionEngine{Granted: grant(
		permission.ResourceWorker,
		permission.ResourceWorkerTraining,
		permission.ResourceWorkerDOTTest,
		permission.ResourceProfileChangeRequest,
		permission.ResourceDocument,
	)}
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	ctx := userContext(t)
	authCtx, err := deps.RequirePermission(ctx, permission.ResourceWorker, permission.OpRead)
	require.NoError(t, err)

	sections := deps.OverviewSections(ctx, authCtx)

	assert.Equal(t, workeroverviewservice.Sections{
		Training:    true,
		DrugAlcohol: true,
		HOS:         true,
		Documents:   true,
	}, sections, "profile change requests need driver portal read as well")
	assert.Nil(t, deps.OverviewMemoryActor(ctx, authCtx, sections))
}

func TestOverviewMemoryActor_IsThePersonWhenAssistantIsReadable(t *testing.T) {
	t.Parallel()
	engine := &resolvertest.GrantingPermissionEngine{Granted: grant(
		permission.ResourceWorker,
		permission.ResourceAssistant,
	)}
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	ctx := userContext(t)
	authCtx, err := deps.RequirePermission(ctx, permission.ResourceWorker, permission.OpRead)
	require.NoError(t, err)

	sections := deps.OverviewSections(ctx, authCtx)
	actor := deps.OverviewMemoryActor(ctx, authCtx, sections)

	require.True(t, sections.AgentMemory)
	require.NotNil(t, actor)
	assert.Equal(t, authCtx.UserID, actor.Actor.UserID)
	assert.False(t, actor.MayManageShared)
}

func TestWorkerOverview_ClearanceIsNullWithoutCredentialRead(t *testing.T) {
	t.Parallel()
	engine := &resolvertest.GrantingPermissionEngine{Granted: grant(
		permission.ResourceWorker,
		permission.ResourceWorkerTraining,
	)}
	wrk := &worker.Worker{
		ID:            pulid.MustNew("wrk_"),
		Status:        domaintypes.StatusActive,
		CanBeAssigned: true,
		Profile:       &worker.WorkerProfile{},
	}
	workerRepo := workerportsmock.NewMockWorkerRepository(t)
	workerRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(wrk, nil).Once()
	service := workeroverviewservice.New(workeroverviewservice.Params{
		Logger:     zap.NewNop(),
		Clock:      func() int64 { return 1_800_000_000 },
		Features:   plantest.Without(platformcatalog.FeatureSamsaraIntegration),
		WorkerRepo: workerRepo,
		Training: &trainingOnly{
			summary: &workertraining.WorkerTrainingSummary{Compliant: true},
		},
	})
	q := &QueryResolver{&Deps{
		Core:                  &resolvercore.Core{PermissionEngine: engine},
		WorkerOverviewService: service,
	}}

	overview, err := q.WorkerOverview(userContext(t), wrk.ID.String())

	require.NoError(t, err)
	assert.Nil(t, overview.Clearance)
	assert.Nil(t, overview.Credentials)
	assert.NotNil(t, overview.Training)
}

func TestWorkerOverviewPayRate_RendersTheDecimal(t *testing.T) {
	t.Parallel()
	r := &WorkerOverviewPayRateResolver{&Deps{}}

	rate, err := r.Rate(t.Context(), &workeroverviewservice.PayRate{
		Rate: decimal.RequireFromString("0.565"),
	})

	require.NoError(t, err)
	assert.Equal(t, "0.565", rate)
}
