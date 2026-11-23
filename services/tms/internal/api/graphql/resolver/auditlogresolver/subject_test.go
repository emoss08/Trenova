package auditlogresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type scopedEngine struct {
	mocks.AllowAllPermissionEngine

	scopes map[string]permission.DataScope
}

func (e *scopedEngine) result(resource string) permissionports.PermissionCheckResult {
	scope, listed := e.scopes[resource]
	if !listed {
		scope = permission.DataScopeOrganization
	}
	if scope == "" {
		return permissionports.PermissionCheckResult{Allowed: false}
	}
	return permissionports.PermissionCheckResult{Allowed: true, DataScope: scope}
}

func (e *scopedEngine) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	result := e.result(req.Resource)
	return &result, nil
}

func (e *scopedEngine) CheckBatch(
	_ context.Context,
	req *permissionports.BatchPermissionCheckRequest,
) (*permissionports.BatchPermissionCheckResult, error) {
	results := make([]permissionports.PermissionCheckResult, len(req.Checks))
	for i, check := range req.Checks {
		results[i] = e.result(check.Resource)
	}
	return &permissionports.BatchPermissionCheckResult{Results: results}, nil
}

type subjectFixture struct {
	resolver *QueryResolver
	service  *auditportsmock.MockAuditService
	ctx      context.Context
}

func newSubjectFixture(t *testing.T, scopes map[string]permission.DataScope) *subjectFixture {
	t.Helper()

	service := auditportsmock.NewMockAuditService(t)
	deps := &Deps{
		Core: &resolvercore.Core{
			L:                zap.NewNop(),
			PermissionEngine: &scopedEngine{scopes: scopes},
		},
		AuditService: service,
	}
	return &subjectFixture{
		resolver: &QueryResolver{deps},
		service:  service,
		ctx:      resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
	}
}

func (f *subjectFixture) capture() *auditports.ListAuditEntriesRequest {
	var captured auditports.ListAuditEntriesRequest
	f.service.EXPECT().
		ListConnection(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *auditports.ListAuditEntriesRequest,
		) (*pagination.CursorListResult[*audit.Entry], error) {
			captured = *req
			return &pagination.CursorListResult[*audit.Entry]{Items: []*audit.Entry{}}, nil
		}).
		Once()
	return &captured
}

func TestAuditEntriesBySubject_RefusesAReaderWithoutTheSubjectsKind(t *testing.T) {
	t.Parallel()

	f := newSubjectFixture(t, map[string]permission.DataScope{
		permission.ResourceWorker.String(): "",
	})

	_, err := f.resolver.AuditEntriesBySubject(
		f.ctx,
		gqlmodel.DataTableConnectionInput{},
		pulid.MustNew("wrk_").String(),
	)

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "the read must be refused: %v", err)
	f.service.AssertNotCalled(t, "ListConnection", mock.Anything, mock.Anything)
}

func TestAuditEntriesBySubject_RefusesOwnOrTeamScopeOnTheSubjectsKind(t *testing.T) {
	t.Parallel()

	for _, scope := range []permission.DataScope{permission.DataScopeOwn, permission.DataScopeTeam} {
		f := newSubjectFixture(t, map[string]permission.DataScope{
			permission.ResourceWorker.String(): scope,
		})

		_, err := f.resolver.AuditEntriesBySubject(
			f.ctx,
			gqlmodel.DataTableConnectionInput{},
			pulid.MustNew("wrk_").String(),
		)

		var authzErr *errortypes.AuthorizationError
		require.True(t, errors.As(err, &authzErr), "%s scope must be refused: %v", scope, err)
	}
}

func TestAuditEntriesBySubject_RefusesAKindThatKeepsNoFile(t *testing.T) {
	t.Parallel()

	f := newSubjectFixture(t, nil)

	_, err := f.resolver.AuditEntriesBySubject(
		f.ctx,
		gqlmodel.DataTableConnectionInput{},
		pulid.MustNew("shp_").String(),
	)

	var validationErr *errortypes.Error
	require.True(t, errors.As(err, &validationErr), "a shipment keeps no file: %v", err)
	f.service.AssertNotCalled(t, "ListConnection", mock.Anything, mock.Anything)
}

func TestAuditEntriesBySubject_ShowsOnlyTheSubRecordKindsTheReaderMayRead(t *testing.T) {
	t.Parallel()

	f := newSubjectFixture(t, map[string]permission.DataScope{
		permission.ResourceWorkerDOTTest.String():     "",
		permission.ResourceWorkerSafetyEvent.String(): permission.DataScopeTeam,
	})
	req := f.capture()
	workerID := pulid.MustNew("wrk_")

	_, err := f.resolver.AuditEntriesBySubject(
		f.ctx,
		gqlmodel.DataTableConnectionInput{},
		workerID.String(),
	)
	require.NoError(t, err)

	assert.Equal(t, workerID, req.SubjectID)
	assert.Empty(t, req.ResourceID)
	assert.False(t, req.Access.AllResources)
	assert.Contains(t, req.Access.Resources, permission.ResourceWorker)
	assert.Contains(t, req.Access.Resources, permission.ResourceWorkerCredential)
	assert.NotContains(t, req.Access.Resources, permission.ResourceWorkerDOTTest,
		"a reader without worker_dot_test read sees none of the worker's tests")
	assert.NotContains(t, req.Access.Resources, permission.ResourceWorkerSafetyEvent,
		"team scope on a sub-record's kind hides its entries")
}

func TestAuditEntriesBySubject_EstimatesATotalOnTheFirstPageOnly(t *testing.T) {
	t.Parallel()

	f := newSubjectFixture(t, nil)
	req := f.capture()

	_, err := f.resolver.AuditEntriesBySubject(
		f.ctx,
		gqlmodel.DataTableConnectionInput{},
		pulid.MustNew("wrk_").String(),
	)
	require.NoError(t, err)
	assert.True(t, req.Cursor.IncludeTotalCount, "the first page estimates its total")
	assert.True(t, req.Access.AllResources)

	later := countFirstPageOnly(
		pagination.CursorInfo{Limit: 50, After: "next", IncludeTotalCount: true},
	)
	assert.False(t, later.IncludeTotalCount, "a later page counts nothing")
	assert.Equal(t, 50, later.Limit)
	assert.Equal(t, "next", later.After)
}

func TestAuditEntrySubject_IsNullWithoutASubject(t *testing.T) {
	t.Parallel()

	resolver, ctx := entryResolver(t)

	resource, err := resolver.SubjectResource(ctx, &audit.Entry{})
	require.NoError(t, err)
	assert.Nil(t, resource)
	id, err := resolver.SubjectID(ctx, &audit.Entry{})
	require.NoError(t, err)
	assert.Nil(t, id)

	workerID := pulid.MustNew("wrk_")
	entry := &audit.Entry{SubjectResource: permission.ResourceWorker, SubjectID: workerID}
	resource, err = resolver.SubjectResource(ctx, entry)
	require.NoError(t, err)
	require.NotNil(t, resource)
	assert.Equal(t, "worker", *resource)
	id, err = resolver.SubjectID(ctx, entry)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, workerID.String(), *id)
}
