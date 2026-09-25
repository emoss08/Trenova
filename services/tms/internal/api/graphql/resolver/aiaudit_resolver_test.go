package resolver

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resourceGrants struct {
	grantingPermissionEngine
}

func grantResources(grants ...string) *resourceGrants {
	engine := &resourceGrants{}
	engine.granted = make(map[string]bool, len(grants))
	for _, grant := range grants {
		engine.granted[grant] = true
	}

	return engine
}

func grantKey(resource permission.Resource, op permission.Operation) string {
	return resource.String() + "|" + string(op)
}

type fakeAIAuditService struct {
	services.AIAuditService

	mu    sync.Mutex
	calls []string
}

func (f *fakeAIAuditService) record(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, name)
}

func (f *fakeAIAuditService) ListEvents(
	context.Context,
	*services.ListAIAuditEventsRequest,
) (*pagination.CursorListResult[*aiaudit.AIAuditEvent], error) {
	f.record("ListEvents")

	return &pagination.CursorListResult[*aiaudit.AIAuditEvent]{}, nil
}

func (f *fakeAIAuditService) GetEvent(
	context.Context,
	repositories.GetAIAuditEventRequest,
) (*aiaudit.AIAuditEvent, error) {
	f.record("GetEvent")

	return &aiaudit.AIAuditEvent{}, nil
}

func (f *fakeAIAuditService) ChainStatus(
	context.Context,
	pagination.TenantInfo,
) (*services.AIAuditChainStatus, error) {
	f.record("ChainStatus")

	return &services.AIAuditChainStatus{}, nil
}

func (f *fakeAIAuditService) RequestVerification(
	context.Context,
	*services.RequestActor,
) (*services.AIAuditChainStatus, error) {
	f.record("RequestVerification")

	return &services.AIAuditChainStatus{Verifying: true}, nil
}

func (f *fakeAIAuditService) RequestExport(
	context.Context,
	*services.RequestAIAuditExportRequest,
) (*aiaudit.AIAuditExport, error) {
	f.record("RequestExport")

	return &aiaudit.AIAuditExport{}, nil
}

func (f *fakeAIAuditService) GetExport(
	context.Context,
	repositories.GetAIAuditExportRequest,
) (*aiaudit.AIAuditExport, error) {
	f.record("GetExport")

	return &aiaudit.AIAuditExport{}, nil
}

func (f *fakeAIAuditService) ListExports(
	context.Context,
	*repositories.ListAIAuditExportsRequest,
) (*pagination.CursorListResult[*aiaudit.AIAuditExport], error) {
	f.record("ListExports")

	return &pagination.CursorListResult[*aiaudit.AIAuditExport]{}, nil
}

func (f *fakeAIAuditService) ExportDownload(
	context.Context,
	*services.GetAIAuditExportDownloadRequest,
) (*services.AIAuditExportDownload, error) {
	f.record("ExportDownload")

	return &services.AIAuditExportDownload{}, nil
}

type aiAuditFixture struct {
	service *fakeAIAuditService
	root    *Resolver
	userID  pulid.ID
}

func newAIAuditFixture(grants ...string) *aiAuditFixture {
	service := &fakeAIAuditService{}

	return &aiAuditFixture{
		service: service,
		root: &Resolver{
			aiAuditService:   service,
			permissionEngine: grantResources(grants...),
		},
		userID: pulid.MustNew("usr_"),
	}
}

func (f *aiAuditFixture) ctx(t *testing.T) context.Context {
	t.Helper()

	ctx := graphql.WithOperationContext(t.Context(), &graphql.OperationContext{})
	ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{})

	return gqlctx.WithAuthContext(ctx, &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    f.userID,
		UserID:         f.userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

func TestAIAuditRootResolvers_RequireTheTrailPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew(aiaudit.EventIDPrefix).String()
	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, root *Resolver) error
	}{
		{"aiAuditEvents", permission.OpRead, func(ctx context.Context, root *Resolver) error {
			_, err := (&queryResolver{root}).AiAuditEvents(ctx, gqlmodel.DataTableConnectionInput{})
			return err
		}},
		{"aiAuditEvent", permission.OpRead, func(ctx context.Context, root *Resolver) error {
			_, err := (&queryResolver{root}).AiAuditEvent(ctx, id)
			return err
		}},
		{"aiAuditChainStatus", permission.OpRead, func(ctx context.Context, root *Resolver) error {
			_, err := (&queryResolver{root}).AiAuditChainStatus(ctx)
			return err
		}},
		{"aiAuditExports", permission.OpRead, func(ctx context.Context, root *Resolver) error {
			_, err := (&queryResolver{root}).AiAuditExports(
				ctx,
				gqlmodel.DataTableConnectionInput{},
			)
			return err
		}},
		{"aiAuditExport", permission.OpRead, func(ctx context.Context, root *Resolver) error {
			_, err := (&queryResolver{root}).AiAuditExport(ctx, id)
			return err
		}},
		{"verifyAIAuditChain", permission.OpRead, func(ctx context.Context, root *Resolver) error {
			_, err := (&mutationResolver{root}).VerifyAIAuditChain(ctx)
			return err
		}},
		{
			"requestAIAuditExport",
			permission.OpExport,
			func(ctx context.Context, root *Resolver) error {
				_, err := (&mutationResolver{root}).RequestAIAuditExport(
					ctx,
					gqlmodel.RequestAIAuditExportInput{
						Format: aiaudit.ExportFormatCSV,
						From:   1,
						To:     2,
					},
				)
				return err
			},
		},
		{
			"aiAuditExportDownload",
			permission.OpExport,
			func(ctx context.Context, root *Resolver) error {
				_, err := (&mutationResolver{root}).AiAuditExportDownload(ctx, id)
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name+"/refused", func(t *testing.T) {
			t.Parallel()

			wrong := permission.OpExport
			if call.operation == permission.OpExport {
				wrong = permission.OpRead
			}
			fixture := newAIAuditFixture(
				grantKey(permission.ResourceAIAuditTrail, wrong),
				grantKey(permission.ResourceAgentRun, permission.OpRead),
				grantKey(permission.ResourceAuditLog, permission.OpRead),
			)

			err := call.run(fixture.ctx(t), fixture.root)

			var authzErr *errortypes.AuthorizationError
			require.True(t, errors.As(err, &authzErr), "%s must be refused: %v", call.name, err)
			assert.Empty(t, fixture.service.calls)
		})

		t.Run(call.name+"/allowed", func(t *testing.T) {
			t.Parallel()

			fixture := newAIAuditFixture(grantKey(permission.ResourceAIAuditTrail, call.operation))

			require.NoError(t, call.run(fixture.ctx(t), fixture.root))
			assert.Len(t, fixture.service.calls, 1)
		})
	}
}

func TestAIAuditEventAuditEntries_NeedTheAuditLog(t *testing.T) {
	t.Parallel()

	fixture := newAIAuditFixture(grantKey(permission.ResourceAIAuditTrail, permission.OpRead))

	entries, err := (&aIAuditEventResolver{fixture.root}).AuditEntries(
		fixture.ctx(t),
		&aiaudit.AIAuditEvent{ID: pulid.MustNew(aiaudit.EventIDPrefix)},
	)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestAIAuditExportDownloadable_OnlyForTheRequester(t *testing.T) {
	t.Parallel()

	fixture := newAIAuditFixture(grantKey(permission.ResourceAIAuditTrail, permission.OpRead))
	export := &aiaudit.AIAuditExport{
		RequestedByUserID: fixture.userID,
		Status:            aiaudit.ExportStatusSucceeded,
		ArtifactKey:       "ai-audit-exports/x.csv",
	}
	resolver := &aIAuditExportResolver{fixture.root}

	mine, err := resolver.Downloadable(fixture.ctx(t), export)
	require.NoError(t, err)
	assert.True(t, mine)

	export.RequestedByUserID = pulid.MustNew("usr_")
	theirs, err := resolver.Downloadable(fixture.ctx(t), export)
	require.NoError(t, err)
	assert.False(t, theirs)
}
