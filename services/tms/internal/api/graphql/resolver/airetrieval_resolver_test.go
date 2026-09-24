package resolver

import (
	"context"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type grantingPermissionEngine struct {
	mocks.AllowAllPermissionEngine

	mu       sync.Mutex
	granted  map[string]bool
	requests []*services.PermissionCheckRequest
}

func grantOnly(grants ...permission.Operation) *grantingPermissionEngine {
	engine := &grantingPermissionEngine{granted: make(map[string]bool, len(grants))}
	for _, op := range grants {
		engine.granted[permission.ResourceAIProvider.String()+"|"+string(op)] = true
	}

	return engine
}

func (e *grantingPermissionEngine) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.requests = append(e.requests, req)

	return &services.PermissionCheckResult{
		Allowed: e.granted[req.Resource+"|"+string(req.Operation)],
	}, nil
}

type fakeRetrievalStatusService struct {
	mu        sync.Mutex
	calls     []string
	tenants   []pagination.TenantInfo
	update    *services.UpdateAIRetrievalSettingsRequest
	reindex   *services.ReindexAIRetrievalSourceRequest
	estimate  airetrieval.SourceType
	failedReq *services.ListAIRetrievalFailedEntriesRequest
	page      *services.AIRetrievalFailedEntryPage
}

func (f *fakeRetrievalStatusService) record(call string, tenant pagination.TenantInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, call)
	f.tenants = append(f.tenants, tenant)
}

func (f *fakeRetrievalStatusService) Status(
	_ context.Context,
	tenant pagination.TenantInfo,
) (*services.AIRetrievalStatus, error) {
	f.record("Status", tenant)
	return &services.AIRetrievalStatus{}, nil
}

func (f *fakeRetrievalStatusService) UpdateSettings(
	_ context.Context,
	req *services.UpdateAIRetrievalSettingsRequest,
) (*services.AIRetrievalStatus, error) {
	f.record("UpdateSettings", req.TenantInfo)
	f.update = req
	return &services.AIRetrievalStatus{}, nil
}

func (f *fakeRetrievalStatusService) Reindex(
	_ context.Context,
	req *services.ReindexAIRetrievalSourceRequest,
) (*services.AIRetrievalStatus, error) {
	f.record("Reindex", req.TenantInfo)
	f.reindex = req
	return &services.AIRetrievalStatus{}, nil
}

func (f *fakeRetrievalStatusService) ReindexEstimate(
	_ context.Context,
	tenant pagination.TenantInfo,
	sourceType airetrieval.SourceType,
) (*services.AIRetrievalReindexEstimate, error) {
	f.record("ReindexEstimate", tenant)
	f.estimate = sourceType
	return &services.AIRetrievalReindexEstimate{SourceType: sourceType}, nil
}

func (f *fakeRetrievalStatusService) ListFailedEntries(
	_ context.Context,
	req *services.ListAIRetrievalFailedEntriesRequest,
) (*services.AIRetrievalFailedEntryPage, error) {
	f.record("ListFailedEntries", req.TenantInfo)
	f.failedReq = req
	if f.page != nil {
		return f.page, nil
	}
	return &services.AIRetrievalFailedEntryPage{}, nil
}

type retrievalResolverFixture struct {
	orgID   pulid.ID
	buID    pulid.ID
	userID  pulid.ID
	service *fakeRetrievalStatusService
	engine  *grantingPermissionEngine
	query   *queryResolver
	mutate  *mutationResolver
}

func newRetrievalResolverFixture(
	t *testing.T,
	grants ...permission.Operation,
) *retrievalResolverFixture {
	t.Helper()

	f := &retrievalResolverFixture{
		orgID:   pulid.MustNew("org_"),
		buID:    pulid.MustNew("bu_"),
		userID:  pulid.MustNew("usr_"),
		service: &fakeRetrievalStatusService{},
		engine:  grantOnly(grants...),
	}
	root := &Resolver{
		aiRetrievalStatusService: f.service,
		permissionEngine:         f.engine,
	}
	f.query = &queryResolver{root}
	f.mutate = &mutationResolver{root}

	return f
}

func (f *retrievalResolverFixture) ctx(t *testing.T) context.Context {
	t.Helper()

	return gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    f.userID,
		UserID:         f.userID,
		OrganizationID: f.orgID,
		BusinessUnitID: f.buID,
	})
}

type retrievalCall struct {
	name      string
	operation permission.Operation
	run       func(ctx context.Context, f *retrievalResolverFixture) error
}

func retrievalCalls() []retrievalCall {
	return []retrievalCall{
		{
			name:      "aiRetrievalStatus",
			operation: permission.OpRead,
			run: func(ctx context.Context, f *retrievalResolverFixture) error {
				_, err := f.query.AiRetrievalStatus(ctx)
				return err
			},
		},
		{
			name:      "aiRetrievalReindexEstimate",
			operation: permission.OpRead,
			run: func(ctx context.Context, f *retrievalResolverFixture) error {
				_, err := f.query.AiRetrievalReindexEstimate(ctx, airetrieval.SourceTypeDocument)
				return err
			},
		},
		{
			name:      "aiRetrievalFailedEntryConnection",
			operation: permission.OpRead,
			run: func(ctx context.Context, f *retrievalResolverFixture) error {
				_, err := f.query.AiRetrievalFailedEntryConnection(
					ctx,
					nil,
					gqlmodel.DataTableConnectionInput{},
				)
				return err
			},
		},
		{
			name:      "updateAIRetrievalSettings",
			operation: permission.OpUpdate,
			run: func(ctx context.Context, f *retrievalResolverFixture) error {
				enabled := false
				_, err := f.mutate.UpdateAIRetrievalSettings(
					ctx,
					gqlmodel.AIRetrievalSettingsPatchInput{
						MemoryEnabled: graphql.OmittableOf(&enabled),
					},
				)
				return err
			},
		},
		{
			name:      "reindexAIRetrievalSource",
			operation: permission.OpUpdate,
			run: func(ctx context.Context, f *retrievalResolverFixture) error {
				_, err := f.mutate.ReindexAIRetrievalSource(ctx, airetrieval.SourceTypeMemory)
				return err
			},
		},
	}
}

func TestRetrievalResolvers_RefuseWithoutTheAIProviderRight(t *testing.T) {
	t.Parallel()

	for _, call := range retrievalCalls() {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			f := newRetrievalResolverFixture(t)
			err := call.run(f.ctx(t), f)

			require.Error(t, err)
			assert.True(t, errortypes.IsAuthorizationError(err), "got %v", err)
			assert.Empty(t, f.service.calls, "a refused call must never reach the service")
			require.Len(t, f.engine.requests, 1)
			assert.Equal(
				t,
				permission.ResourceAIProvider.String(),
				f.engine.requests[0].Resource,
			)
			assert.Equal(t, call.operation, f.engine.requests[0].Operation)
		})
	}
}

func TestRetrievalResolvers_ReadingDoesNotGrantChanging(t *testing.T) {
	t.Parallel()

	for _, call := range retrievalCalls() {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			f := newRetrievalResolverFixture(t, permission.OpRead)
			err := call.run(f.ctx(t), f)

			if call.operation == permission.OpRead {
				require.NoError(t, err)
				require.Len(t, f.service.calls, 1)
				return
			}

			require.Error(t, err)
			assert.True(t, errortypes.IsAuthorizationError(err), "got %v", err)
			assert.Empty(t, f.service.calls)
		})
	}
}

func TestRetrievalResolvers_ScopeEveryCallToTheCallersTenant(t *testing.T) {
	t.Parallel()

	for _, call := range retrievalCalls() {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			f := newRetrievalResolverFixture(t, permission.OpRead, permission.OpUpdate)
			require.NoError(t, call.run(f.ctx(t), f))

			require.Len(t, f.service.tenants, 1)
			assert.Equal(
				t,
				pagination.TenantInfo{OrgID: f.orgID, BuID: f.buID},
				f.service.tenants[0],
			)
		})
	}
}

func TestRetrievalResolvers_RefuseWithoutASignedInCaller(t *testing.T) {
	t.Parallel()

	for _, call := range retrievalCalls() {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			f := newRetrievalResolverFixture(t, permission.OpRead, permission.OpUpdate)
			err := call.run(t.Context(), f)

			require.Error(t, err)
			assert.Empty(t, f.service.calls)
			assert.Empty(t, f.engine.requests)
		})
	}
}

func TestReindexAIRetrievalSource_PassesTheSourceAndActor(t *testing.T) {
	t.Parallel()

	f := newRetrievalResolverFixture(t, permission.OpUpdate)
	_, err := f.mutate.ReindexAIRetrievalSource(f.ctx(t), airetrieval.SourceTypeInboundMessage)
	require.NoError(t, err)

	require.NotNil(t, f.service.reindex)
	assert.Equal(t, airetrieval.SourceTypeInboundMessage, f.service.reindex.SourceType)
	require.NotNil(t, f.service.reindex.Actor)
	assert.Equal(t, f.userID, f.service.reindex.Actor.UserID)
}

func TestAIRetrievalFailedEntryConnection_NarrowsToTheSourceAndMapsThePage(t *testing.T) {
	t.Parallel()

	f := newRetrievalResolverFixture(t, permission.OpRead)
	total := 3
	sourceID := pulid.MustNew("doc_")
	f.service.page = &services.AIRetrievalFailedEntryPage{
		Edges: []*services.AIRetrievalFailedEntryEdge{
			{Node: &services.AIRetrievalFailedEntry{ID: "a", SourceID: sourceID}, Cursor: "c1"},
			{Node: &services.AIRetrievalFailedEntry{ID: "b", SourceID: sourceID}, Cursor: "c2"},
		},
		HasNextPage: true,
		TotalCount:  &total,
	}
	sourceType := airetrieval.SourceTypeDocument
	first := 2

	conn, err := f.query.AiRetrievalFailedEntryConnection(
		f.ctx(t),
		&sourceType,
		gqlmodel.DataTableConnectionInput{First: &first},
	)
	require.NoError(t, err)

	require.NotNil(t, f.service.failedReq)
	assert.Equal(t, airetrieval.SourceTypeDocument, f.service.failedReq.SourceType)
	assert.Equal(t, 2, f.service.failedReq.Table.First)

	require.Len(t, conn.Edges, 2)
	assert.Equal(t, "b", conn.Edges[1].Node.ID)
	assert.True(t, conn.PageInfo.HasNextPage)
	require.NotNil(t, conn.PageInfo.EndCursor)
	assert.Equal(t, "c2", *conn.PageInfo.EndCursor)
	require.NotNil(t, conn.TotalCount)
	assert.Equal(t, 3, *conn.TotalCount)
}

func TestAIRetrievalSettingsPatchFromInput(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	t.Run("absent fields are left alone", func(t *testing.T) {
		t.Parallel()

		req, err := aiRetrievalSettingsPatchFromInput(
			tenant,
			&gqlmodel.AIRetrievalSettingsPatchInput{},
		)
		require.NoError(t, err)
		assert.True(t, req.Empty())
		assert.Equal(t, tenant, req.TenantInfo)
	})

	t.Run("given fields are copied, false included", func(t *testing.T) {
		t.Parallel()

		off, on := false, true
		budget := "25.50"
		req, err := aiRetrievalSettingsPatchFromInput(tenant, &gqlmodel.AIRetrievalSettingsPatchInput{
			DocumentsEnabled:         graphql.OmittableOf(&off),
			Paused:                   graphql.OmittableOf(&on),
			MonthlyIndexingBudgetUsd: graphql.OmittableOf(&budget),
		})
		require.NoError(t, err)

		require.NotNil(t, req.DocumentsEnabled)
		assert.False(t, *req.DocumentsEnabled)
		require.NotNil(t, req.Paused)
		assert.True(t, *req.Paused)
		require.NotNil(t, req.MonthlyIndexingBudgetUSD)
		assert.True(t, decimal.RequireFromString("25.5").Equal(*req.MonthlyIndexingBudgetUSD))
		assert.Nil(t, req.MemoryEnabled)
		assert.Nil(t, req.InboundMessagesEnabled)

		off = true
		assert.False(t, *req.DocumentsEnabled, "the request must not alias the input")
	})

	t.Run("an explicit null cannot clear a required setting", func(t *testing.T) {
		t.Parallel()

		_, err := aiRetrievalSettingsPatchFromInput(tenant, &gqlmodel.AIRetrievalSettingsPatchInput{
			MemoryEnabled:            graphql.OmittableOf[*bool](nil),
			MonthlyIndexingBudgetUsd: graphql.OmittableOf[*string](nil),
		})
		require.Error(t, err)

		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr)
		fields := make([]string, 0, len(multiErr.Errors))
		for _, item := range multiErr.Errors {
			fields = append(fields, item.Field)
		}
		assert.ElementsMatch(t, []string{"memoryEnabled", "monthlyIndexingBudgetUsd"}, fields)
	})

	t.Run("a budget that is not a number is refused on its field", func(t *testing.T) {
		t.Parallel()

		budget := "ten dollars"
		_, err := aiRetrievalSettingsPatchFromInput(tenant, &gqlmodel.AIRetrievalSettingsPatchInput{
			MonthlyIndexingBudgetUsd: graphql.OmittableOf(&budget),
		})

		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr)
		require.Len(t, multiErr.Errors, 1)
		assert.Equal(t, "monthlyIndexingBudgetUsd", multiErr.Errors[0].Field)
	})
}
