package agentplanhandler_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/agentplanhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingPlans struct {
	agentports.AgentPlanService

	plan *agentplan.AgentPlan
	sent *agentports.DecideAgentPlanRequest
}

func (p *recordingPlans) Decide(
	_ context.Context,
	req *agentports.DecideAgentPlanRequest,
	_ *permissionports.RequestActor,
) (*agentplan.AgentPlan, error) {
	p.sent = req

	return p.plan, nil
}

func (p *recordingPlans) GetByID(
	context.Context,
	agentports.GetAgentPlanByIDRequest,
) (*agentplan.AgentPlan, error) {
	return p.plan, nil
}

type planPreviews struct {
	agentruntimeports.ProposalPreviewService
}

func (planPreviews) ForPlan(
	_ context.Context,
	req *agentruntimeports.PlanPreviewRequest,
) (*agentpreview.PlanPreview, error) {
	return &agentpreview.PlanPreview{PlanID: req.Plan.ID, Digest: strings.Repeat("c", 64)}, nil
}

func newHandler(plans agentports.AgentPlanService) *agentplanhandler.Handler {
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    zap.NewNop(),
		AppConfig: &appconfig.Config{Debug: true},
	})

	return agentplanhandler.New(agentplanhandler.Params{
		PlanService:    plans,
		PreviewService: planPreviews{},
		ErrorHandler:   errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: &mocks.AllowAllPermissionEngine{},
				ErrorHandler:     errorHandler,
			},
		),
	})
}

func TestResolve_EchoesThePlanPreviewDigest(t *testing.T) {
	t.Parallel()

	plans := &recordingPlans{plan: &agentplan.AgentPlan{ID: pulid.MustNew("apl_")}}
	digest := strings.Repeat("d", 64)
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/agent-plans/" + plans.plan.ID.String() + "/resolve/").
		WithJSONBody(map[string]any{
			"decision":      agentdecision.DecisionAccepted,
			"reasonCode":    "looks_right",
			"note":          "All three are right",
			"previewDigest": digest,
		}).
		WithDefaultAuthContext()
	newHandler(plans).RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))
	require.NotNil(t, plans.sent)
	assert.Equal(t, digest, plans.sent.PreviewDigest)
	assert.Equal(t, "All three are right", plans.sent.Note)
}

func TestPreview_ReturnsThePlanDigestToEcho(t *testing.T) {
	t.Parallel()

	plans := &recordingPlans{plan: &agentplan.AgentPlan{ID: pulid.MustNew("apl_")}}
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/agent-plans/" + plans.plan.ID.String() + "/preview/").
		WithDefaultAuthContext()
	newHandler(plans).RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))
	var body map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&body))
	assert.Equal(t, strings.Repeat("c", 64), body["digest"])
}
