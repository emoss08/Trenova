package aiauditresolver

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resourceGrants struct {
	resolvertest.GrantingPermissionEngine
}

func grantResources(grants ...string) *resourceGrants {
	engine := &resourceGrants{}
	engine.Granted = make(map[string]bool, len(grants))
	for _, grant := range grants {
		engine.Granted[grant] = true
	}

	return engine
}

func grantKey(resource permission.Resource, op permission.Operation) string {
	return resource.String() + "|" + string(op)
}

type fakeAIAuditService struct {
	aiauditports.AIAuditService

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
	*aiauditports.ListAIAuditEventsRequest,
) (*pagination.CursorListResult[*aiaudit.AIAuditEvent], error) {
	f.record("ListEvents")

	return &pagination.CursorListResult[*aiaudit.AIAuditEvent]{}, nil
}

func (f *fakeAIAuditService) GetEvent(
	context.Context,
	aiauditports.GetAIAuditEventRequest,
) (*aiaudit.AIAuditEvent, error) {
	f.record("GetEvent")

	return &aiaudit.AIAuditEvent{}, nil
}

func (f *fakeAIAuditService) ChainStatus(
	context.Context,
	pagination.TenantInfo,
) (*aiauditports.AIAuditChainStatus, error) {
	f.record("ChainStatus")

	return &aiauditports.AIAuditChainStatus{}, nil
}

func (f *fakeAIAuditService) RequestVerification(
	context.Context,
	*permissionports.RequestActor,
) (*aiauditports.AIAuditChainStatus, error) {
	f.record("RequestVerification")

	return &aiauditports.AIAuditChainStatus{Verifying: true}, nil
}

func (f *fakeAIAuditService) RequestExport(
	context.Context,
	*aiauditports.RequestAIAuditExportRequest,
) (*aiaudit.AIAuditExport, error) {
	f.record("RequestExport")

	return &aiaudit.AIAuditExport{}, nil
}

func (f *fakeAIAuditService) GetExport(
	context.Context,
	aiauditports.GetAIAuditExportRequest,
) (*aiaudit.AIAuditExport, error) {
	f.record("GetExport")

	return &aiaudit.AIAuditExport{}, nil
}

func (f *fakeAIAuditService) ListExports(
	context.Context,
	*aiauditports.ListAIAuditExportsRequest,
) (*pagination.CursorListResult[*aiaudit.AIAuditExport], error) {
	f.record("ListExports")

	return &pagination.CursorListResult[*aiaudit.AIAuditExport]{}, nil
}

func (f *fakeAIAuditService) ExportDownload(
	context.Context,
	*aiauditports.GetAIAuditExportDownloadRequest,
) (*aiauditports.AIAuditExportDownload, error) {
	f.record("ExportDownload")

	return &aiauditports.AIAuditExportDownload{}, nil
}

type aiAuditFixture struct {
	service *fakeAIAuditService
	root    *Deps
	userID  pulid.ID
}

func newAIAuditFixture(grants ...string) *aiAuditFixture {
	service := &fakeAIAuditService{}

	return &aiAuditFixture{
		service: service,
		root: &Deps{
			Core:           &base.Core{PermissionEngine: grantResources(grants...)},
			AiAuditService: service,
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
		run       func(ctx context.Context, root *Deps) error
	}{
		{"aiAuditEvents", permission.OpRead, func(ctx context.Context, root *Deps) error {
			_, err := (&QueryResolver{root}).AiAuditEvents(ctx, gqlmodel.DataTableConnectionInput{})
			return err
		}},
		{"aiAuditEvent", permission.OpRead, func(ctx context.Context, root *Deps) error {
			_, err := (&QueryResolver{root}).AiAuditEvent(ctx, id)
			return err
		}},
		{"aiAuditChainStatus", permission.OpRead, func(ctx context.Context, root *Deps) error {
			_, err := (&QueryResolver{root}).AiAuditChainStatus(ctx)
			return err
		}},
		{"aiAuditExports", permission.OpRead, func(ctx context.Context, root *Deps) error {
			_, err := (&QueryResolver{root}).AiAuditExports(
				ctx,
				gqlmodel.DataTableConnectionInput{},
			)
			return err
		}},
		{"aiAuditExport", permission.OpRead, func(ctx context.Context, root *Deps) error {
			_, err := (&QueryResolver{root}).AiAuditExport(ctx, id)
			return err
		}},
		{"verifyAIAuditChain", permission.OpRead, func(ctx context.Context, root *Deps) error {
			_, err := (&MutationResolver{root}).VerifyAIAuditChain(ctx)
			return err
		}},
		{
			"requestAIAuditExport",
			permission.OpExport,
			func(ctx context.Context, root *Deps) error {
				_, err := (&MutationResolver{root}).RequestAIAuditExport(
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
			func(ctx context.Context, root *Deps) error {
				_, err := (&MutationResolver{root}).AiAuditExportDownload(ctx, id)
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

	entries, err := (&AIAuditEventResolver{fixture.root}).AuditEntries(
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
	resolver := &AIAuditExportResolver{fixture.root}

	mine, err := resolver.Downloadable(fixture.ctx(t), export)
	require.NoError(t, err)
	assert.True(t, mine)

	export.RequestedByUserID = pulid.MustNew("usr_")
	theirs, err := resolver.Downloadable(fixture.ctx(t), export)
	require.NoError(t, err)
	assert.False(t, theirs)
}
