package audithandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubAuditService struct {
	listResp *pagination.ListResult[*audit.Entry]
	listErr  error
	getResp  *audit.Entry
	getErr   error

	lastListReq *repositories.ListAuditEntriesRequest
	lastGetReq  repositories.GetAuditEntryByIDOptions
}

func (s *stubAuditService) List(
	_ context.Context,
	req *repositories.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	s.lastListReq = req
	return s.listResp, s.listErr
}

func (s *stubAuditService) ListByResourceID(
	_ context.Context,
	_ *repositories.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return nil, nil
}

func (s *stubAuditService) GetByID(
	_ context.Context,
	req repositories.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	s.lastGetReq = req
	return s.getResp, s.getErr
}

func (s *stubAuditService) LogAction(
	_ *servicesport.LogActionParams,
	_ ...servicesport.LogOption,
) error {
	return nil
}

func (s *stubAuditService) LogActions(_ []servicesport.BulkLogEntry) error { return nil }

func (s *stubAuditService) RegisterSensitiveFields(
	_ permission.Resource,
	_ []servicesport.SensitiveField,
) error {
	return nil
}

type allowAllPermissionEngine struct{}

func (e *allowAllPermissionEngine) Check(
	_ context.Context,
	_ *servicesport.PermissionCheckRequest,
) (*servicesport.PermissionCheckResult, error) {
	return &servicesport.PermissionCheckResult{Allowed: true}, nil
}

func (e *allowAllPermissionEngine) CheckBatch(
	_ context.Context,
	_ *servicesport.BatchPermissionCheckRequest,
) (*servicesport.BatchPermissionCheckResult, error) {
	return &servicesport.BatchPermissionCheckResult{}, nil
}

func (e *allowAllPermissionEngine) GetLightManifest(
	_ context.Context,
	_ pulid.ID,
	_ pulid.ID,
) (*servicesport.LightPermissionManifest, error) {
	return nil, nil
}

func (e *allowAllPermissionEngine) GetResourcePermissions(
	_ context.Context,
	_ pulid.ID,
	_ pulid.ID,
	_ string,
) (*servicesport.ResourcePermissionDetail, error) {
	return nil, nil
}

func (e *allowAllPermissionEngine) InvalidateUser(
	_ context.Context,
	_ pulid.ID,
	_ pulid.ID,
) error {
	return nil
}

func (e *allowAllPermissionEngine) GetEffectivePermissions(
	_ context.Context,
	_ pulid.ID,
	_ pulid.ID,
) (*servicesport.EffectivePermissions, error) {
	return nil, nil
}

func (e *allowAllPermissionEngine) SimulatePermissions(
	_ context.Context,
	_ *servicesport.SimulatePermissionsRequest,
) (*servicesport.EffectivePermissions, error) {
	return nil, nil
}

func newAuditTestHandler(
	t *testing.T,
	auditSvc servicesport.AuditService,
) (*Handler, *gin.Engine) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		App: config.AppConfig{Debug: true},
	}
	logger := zap.NewNop()

	eh := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: cfg,
	})
	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: &allowAllPermissionEngine{},
		ErrorHandler:     eh,
	})

	h := New(Params{
		Service:              auditSvc,
		ErrorHandler:         eh,
		PermissionMiddleware: pm,
	})

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		authctx.SetAuthContext(
			c,
			pulid.MustNew("usr_"),
			pulid.MustNew("bu_"),
			pulid.MustNew("org_"),
		)
		c.Next()
	})

	h.RegisterRoutes(api)

	return h, r
}

func TestList_Success(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{
		ID:         pulid.MustNew("ae_"),
		ResourceID: pulid.MustNew("usr_").String(),
	}
	svc := &stubAuditService{
		listResp: &pagination.ListResult[*audit.Entry]{
			Items: []*audit.Entry{entry},
			Total: 1,
		},
	}

	_, router := newAuditTestHandler(t, svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-entries/?limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, svc.lastListReq)
	require.NotNil(t, svc.lastListReq.Filter)
	assert.Equal(t, 10, svc.lastListReq.Filter.Pagination.Limit)
}

func TestList_ServiceError(t *testing.T) {
	t.Parallel()

	svc := &stubAuditService{
		listErr: errors.New("list failed"),
	}

	_, router := newAuditTestHandler(t, svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-entries/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGet_Success(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{
		ID:         pulid.MustNew("ae_"),
		ResourceID: pulid.MustNew("usr_").String(),
	}
	svc := &stubAuditService{
		getResp: entry,
	}

	_, router := newAuditTestHandler(t, svc)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/audit-entries/"+entry.ID.String()+"/",
		nil,
	)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body audit.Entry
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, body.ID)
	assert.Equal(t, entry.ID, svc.lastGetReq.EntryID)
}

func TestGet_InvalidID(t *testing.T) {
	t.Parallel()

	svc := &stubAuditService{}
	_, router := newAuditTestHandler(t, svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-entries/not-a-pulid/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGet_ServiceError(t *testing.T) {
	t.Parallel()

	svc := &stubAuditService{
		getErr: errors.New("get failed"),
	}
	entryID := pulid.MustNew("ae_")

	_, router := newAuditTestHandler(t, svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-entries/"+entryID.String()+"/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
