package agentrunhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type grantEngine struct {
	services.PermissionEngine

	allowed bool
	asked   []*services.PermissionCheckRequest
}

func (e *grantEngine) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	e.asked = append(e.asked, req)

	return &services.PermissionCheckResult{
		Allowed: e.allowed &&
			req.Resource == permission.ResourceAgentRun.String() &&
			req.Operation == permission.OpRead,
	}, nil
}

type stubTranscripts struct {
	requests []repositories.GetAgentRunByIDRequest
	scopes   []dbscope.Tenant
	err      error
}

func (s *stubTranscripts) RunTranscript(
	ctx context.Context,
	req repositories.GetAgentRunByIDRequest,
) (*services.TranscriptFile, error) {
	s.requests = append(s.requests, req)
	if tenant, ok := dbscope.TenantFrom(ctx); ok {
		s.scopes = append(s.scopes, tenant)
	}
	if s.err != nil {
		return nil, s.err
	}

	return &services.TranscriptFile{
		FileName: "overnight-check-run-0run0001.md",
		Body:     "# Overnight check run\n",
	}, nil
}

type transcriptCall struct {
	engine      *grantEngine
	transcripts *stubTranscripts
	orgID       pulid.ID
	buID        pulid.ID
}

func (tc *transcriptCall) do(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})
	handler := New(Params{
		Transcripts: tc.transcripts,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: tc.engine,
				ErrorHandler:     errorHandler,
			},
		),
		ErrorHandler: errorHandler,
	})

	router := gin.New()
	router.Use(func(c *gin.Context) {
		authctx.SetAuthContext(c, pulid.MustNew("usr_"), tc.buID, tc.orgID)
		c.Next()
	})
	handler.RegisterRoutes(router.Group(""))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, http.NoBody))

	return recorder
}

func newTranscriptCall(allowed bool) *transcriptCall {
	return &transcriptCall{
		engine:      &grantEngine{allowed: allowed},
		transcripts: &stubTranscripts{},
		orgID:       pulid.MustNew("org_"),
		buID:        pulid.MustNew("bu_"),
	}
}

func TestDownloadTranscript_HandsTheRunOverAsMarkdownUnderTheCallersTenant(t *testing.T) {
	t.Parallel()

	tc := newTranscriptCall(true)
	runID := pulid.MustNew("ar_")

	recorder := tc.do(t, "/agent-runs/"+runID.String()+"/transcript/")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/markdown; charset=utf-8", recorder.Header().Get("Content-Type"))
	assert.Equal(
		t,
		`attachment; filename="overnight-check-run-0run0001.md"`,
		recorder.Header().Get("Content-Disposition"),
	)
	assert.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	assert.Equal(t, "# Overnight check run\n", recorder.Body.String())

	require.Len(t, tc.transcripts.requests, 1)
	request := tc.transcripts.requests[0]
	assert.Equal(t, runID, request.ID)
	require.NotNil(t, request.TenantInfo)
	assert.Equal(t, tc.orgID, request.TenantInfo.OrgID)
	assert.Equal(t, tc.buID, request.TenantInfo.BuID)

	require.Len(t, tc.transcripts.scopes, 1)
	assert.Equal(t, tc.orgID, tc.transcripts.scopes[0].OrganizationID)
	assert.Equal(t, tc.buID, tc.transcripts.scopes[0].BusinessUnitID)

	require.NotEmpty(t, tc.engine.asked)
	assert.Equal(t, permission.ResourceAgentRun.String(), tc.engine.asked[0].Resource)
	assert.Equal(t, permission.OpRead, tc.engine.asked[0].Operation)
}

func TestDownloadTranscript_NeedsAgentRunRead(t *testing.T) {
	t.Parallel()

	tc := newTranscriptCall(false)

	recorder := tc.do(t, "/agent-runs/"+pulid.MustNew("ar_").String()+"/transcript/")

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Empty(t, tc.transcripts.requests)
}

func TestDownloadTranscript_AnotherTenantsRunIsNotFound(t *testing.T) {
	t.Parallel()

	tc := newTranscriptCall(true)
	tc.transcripts.err = errortypes.NewNotFoundError("Agent run not found")

	recorder := tc.do(t, "/agent-runs/"+pulid.MustNew("ar_").String()+"/transcript/")

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Empty(t, recorder.Header().Get("Content-Disposition"))
}

func TestDownloadTranscript_RefusesAMalformedRunID(t *testing.T) {
	t.Parallel()

	tc := newTranscriptCall(true)

	recorder := tc.do(t, "/agent-runs/not-an-id/transcript/")

	assert.NotEqual(t, http.StatusOK, recorder.Code)
	assert.Empty(t, tc.transcripts.requests)
}
