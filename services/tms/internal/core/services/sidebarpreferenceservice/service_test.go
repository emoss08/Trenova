package sidebarpreferenceservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/sidebarpreference"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestService(
	t *testing.T,
) (*Service, *mocks.MockSidebarPreferenceRepository, *mocks.MockPermissionEngine) {
	t.Helper()

	repo := mocks.NewMockSidebarPreferenceRepository(t)
	engine := mocks.NewMockPermissionEngine(t)
	svc := New(Params{
		Logger:           zap.NewNop(),
		Repo:             repo,
		PermissionEngine: engine,
	})

	return svc, repo, engine
}

func testRequest() *Request {
	return &Request{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		Principal: services.PrincipalInfo{UserID: pulid.MustNew("usr_")},
	}
}

func allowBatch(engine *mocks.MockPermissionEngine, denied map[string]struct{}) {
	engine.EXPECT().
		CheckBatch(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *services.BatchPermissionCheckRequest) (*services.BatchPermissionCheckResult, error) {
			results := make([]services.PermissionCheckResult, 0, len(req.Checks))
			for _, check := range req.Checks {
				_, deny := denied[check.Resource]
				results = append(results, services.PermissionCheckResult{Allowed: !deny})
			}
			return &services.BatchPermissionCheckResult{Results: results}, nil
		})
}

func TestService_GetEffective_DefaultsWhenNoRow(t *testing.T) {
	t.Parallel()

	svc, repo, engine := newTestService(t)
	req := testRequest()

	repo.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, false, nil)
	allowBatch(engine, nil)

	effective, err := svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, int64(0), effective.Version)
	assert.Equal(t, sidebarpreference.DefaultDocument(), effective.Document)
}

func TestService_GetEffective_FiltersUnpermittedEntries(t *testing.T) {
	t.Parallel()

	svc, repo, engine := newTestService(t)
	req := testRequest()

	repo.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, false, nil)
	allowBatch(engine, map[string]struct{}{
		"billing_queue": {},
		"shipment":      {},
	})

	effective, err := svc.GetEffective(t.Context(), req)
	require.NoError(t, err)

	assert.NotContains(t, effective.Document.AttentionMetrics, "billingQueue")
	assert.Contains(t, effective.Document.AttentionMetrics, "serviceFailures")
	assert.NotContains(t, effective.Document.QuickActionIDs, "create-shipment")
	assert.Contains(t, effective.Document.QuickActionIDs, "create-worker")
}

func TestService_GetOptions_FiltersCatalog(t *testing.T) {
	t.Parallel()

	svc, _, engine := newTestService(t)
	req := testRequest()

	allowBatch(engine, map[string]struct{}{
		"edi":    {},
		"worker": {},
	})

	options, err := svc.GetOptions(t.Context(), req)
	require.NoError(t, err)

	assert.Len(t, options.Sections, len(sidebarpreference.SectionCatalog()))
	assert.Equal(t, sidebarpreference.MaxQuickActions, options.MaxQuickActions)

	for _, metric := range options.AttentionMetrics {
		assert.NotEqual(t, "ediAttention", metric.Key)
	}
	for _, action := range options.QuickActions {
		assert.NotEqual(t, "create-worker", action.ID)
	}
}

func TestService_Update_RejectsInvalidDocument(t *testing.T) {
	t.Parallel()

	svc, _, _ := newTestService(t)

	doc := sidebarpreference.DefaultDocument()
	doc.AttentionMetrics = []string{"bogusMetric"}

	_, err := svc.Update(t.Context(), &UpdateRequest{
		Request:  *testRequest(),
		Document: doc,
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "attentionMetrics[0]", multiErr.Errors[0].Field)
}

func TestService_Update_CreatesRowWhenNoneExists(t *testing.T) {
	t.Parallel()

	svc, repo, engine := newTestService(t)
	req := testRequest()

	repo.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, false, nil)
	repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *sidebarpreference.SidebarPreference) (*sidebarpreference.SidebarPreference, error) {
			assert.Equal(t, req.TenantInfo.OrgID, entity.OrganizationID)
			assert.Equal(t, req.TenantInfo.BuID, entity.BusinessUnitID)
			assert.Equal(t, req.TenantInfo.UserID, entity.UserID)
			assert.Equal(t, int64(1), entity.Version)
			return entity, nil
		})
	allowBatch(engine, nil)

	doc := sidebarpreference.DefaultDocument()
	doc.Activity.PageSize = 10

	effective, err := svc.Update(t.Context(), &UpdateRequest{
		Request:  *req,
		Document: doc,
		Version:  0,
	})
	require.NoError(t, err)

	assert.Equal(t, 10, effective.Document.Activity.PageSize)
}

func TestService_Update_VersionMismatchWhenNoRowButVersionSet(t *testing.T) {
	t.Parallel()

	svc, repo, _ := newTestService(t)

	repo.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, false, nil)

	_, err := svc.Update(t.Context(), &UpdateRequest{
		Request:  *testRequest(),
		Document: sidebarpreference.DefaultDocument(),
		Version:  3,
	})

	var valErr *errortypes.Error
	require.True(t, errors.As(err, &valErr))
	assert.Equal(t, errortypes.ErrVersionMismatch, valErr.Code)
}

func TestService_Update_UpdatesExistingRowWithClientVersion(t *testing.T) {
	t.Parallel()

	svc, repo, engine := newTestService(t)
	req := testRequest()

	existing := &sidebarpreference.SidebarPreference{
		ID:             pulid.MustNew("sbp_"),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		UserID:         req.TenantInfo.UserID,
		Preferences:    sidebarpreference.DefaultDocument(),
		Version:        4,
	}

	repo.EXPECT().Get(mock.Anything, mock.Anything).Return(existing, true, nil)
	repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *sidebarpreference.SidebarPreference) (*sidebarpreference.SidebarPreference, error) {
			assert.Equal(t, int64(4), entity.Version)
			entity.Version++
			return entity, nil
		})
	allowBatch(engine, nil)

	doc := sidebarpreference.DefaultDocument()
	doc.Sections[0].Hidden = true

	effective, err := svc.Update(t.Context(), &UpdateRequest{
		Request:  *req,
		Document: doc,
		Version:  4,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(5), effective.Version)
	assert.True(t, effective.Document.Sections[0].Hidden)
}
