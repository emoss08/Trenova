package jurisdictionrulehandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/jurisdictionrulehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type serviceStub struct {
	listReq   *services.ListJurisdictionRulesRequest
	updated   *jurisdictionrule.JurisdictionRule
	verifyReq *services.VerifyJurisdictionRuleRequest
}

func (s *serviceStub) List(
	_ context.Context, req *services.ListJurisdictionRulesRequest,
) (*pagination.ListResult[*jurisdictionrule.JurisdictionRule], error) {
	s.listReq = req
	return &pagination.ListResult[*jurisdictionrule.JurisdictionRule]{
		Items: []*jurisdictionrule.JurisdictionRule{},
		Total: 0,
	}, nil
}

func (s *serviceStub) GetByID(
	context.Context, pulid.ID,
) (*jurisdictionrule.JurisdictionRule, error) {
	return &jurisdictionrule.JurisdictionRule{}, nil
}

func (s *serviceStub) Create(
	_ context.Context, e *jurisdictionrule.JurisdictionRule, _ *services.RequestActor,
) (*jurisdictionrule.JurisdictionRule, error) {
	return e, nil
}

func (s *serviceStub) Update(
	_ context.Context, e *jurisdictionrule.JurisdictionRule, _ *services.RequestActor,
) (*jurisdictionrule.JurisdictionRule, error) {
	s.updated = e
	return e, nil
}

func (s *serviceStub) Verify(
	_ context.Context, req *services.VerifyJurisdictionRuleRequest,
) (*jurisdictionrule.JurisdictionRule, error) {
	s.verifyReq = req
	return &jurisdictionrule.JurisdictionRule{}, nil
}

func setupHandler(t *testing.T, service services.JurisdictionRuleService) *jurisdictionrulehandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})

	return jurisdictionrulehandler.New(jurisdictionrulehandler.Params{
		Service:      service,
		ErrorHandler: errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: &mocks.AllowAllPermissionEngine{},
				ErrorHandler:     errorHandler,
			},
		),
		Logger: logger,
	})
}

// The path owns identity. A payload naming a different row must not redirect
// the write — this table is global, so a misdirected update changes what every
// organization is told is legal for a state nobody intended to touch.
func TestUpdate_TakesTheRuleIDFromThePathNotTheBody(t *testing.T) {
	t.Parallel()

	ruleID := pulid.MustNew("jrl_")
	service := &serviceStub{}
	handler := setupHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/jurisdiction-rules/" + ruleID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"id":              "jrl_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"stateId":         "us_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"status":          "Active",
			"maxWidthFeet":    8.5,
			"maxHeightFeet":   13.5,
			"maxLengthFeet":   53,
			"maxWeightPounds": 80000,
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	require.NotNil(t, service.updated)
	assert.Equal(t, ruleID, service.updated.ID)
}

func TestVerify_PassesTheStateAndNoteThrough(t *testing.T) {
	t.Parallel()

	ruleID := pulid.MustNew("jrl_")
	service := &serviceStub{}
	handler := setupHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/jurisdiction-rules/" + ruleID.String() + "/verify/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"verificationState": "Verified",
			"sourceNote":        "Checked against the state permit office handbook, rev 2026-01",
			"sourceUrl":         "https://example.gov/oversize",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	require.NotNil(t, service.verifyReq)
	assert.Equal(t, ruleID, service.verifyReq.RuleID)
	assert.Equal(t, jurisdictionrule.VerificationVerified, service.verifyReq.State)
	assert.Contains(t, service.verifyReq.SourceNote, "state permit office handbook")
	// The acting user has to reach the service, since the audit entry is the
	// record of who confirmed this.
	require.NotNil(t, service.verifyReq.Actor)
	assert.Equal(t, testutil.TestUserID, service.verifyReq.Actor.UserID)
}

func TestList_PassesTheVerificationFilterThrough(t *testing.T) {
	t.Parallel()

	service := &serviceStub{}
	handler := setupHandler(t, service)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/jurisdiction-rules/").
		WithQuery(map[string]string{"verificationState": "Unverified"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	require.NotNil(t, service.listReq)
	assert.Equal(t, jurisdictionrule.VerificationUnverified, service.listReq.VerificationState)
}

func TestVerify_RejectsAnInvalidRuleID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, &serviceStub{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/jurisdiction-rules/not-a-pulid/verify/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"verificationState": "Verified", "sourceNote": "checked it"})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}
