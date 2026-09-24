package formulatemplatehandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/formulatemplatehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type grantedEngine struct {
	serviceports.PermissionEngine

	granted map[string]bool
}

func (e *grantedEngine) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	return &serviceports.PermissionCheckResult{
		Allowed: e.granted[req.Resource+":"+string(req.Operation)],
	}, nil
}

type formulaPageAssistant struct {
	opened []*serviceports.OpenPageThreadRequest
}

func (p *formulaPageAssistant) OpenPageThread(
	_ context.Context,
	req *serviceports.OpenPageThreadRequest,
	_ *serviceports.RequestActor,
) (*serviceports.PageThread, error) {
	p.opened = append(p.opened, req)
	thread := &conversation.Thread{ID: pulid.MustNew("athr_"), Origin: req.Origin}
	if req.SubjectID.IsNotNil() {
		thread.SubjectType = req.SubjectType
		thread.SubjectID = req.SubjectID
	}

	return &serviceports.PageThread{
		Thread: thread,
		Agent:  serviceports.PageAgent{Name: "Formula assistant", SystemKey: "formula_assistant"},
	}, nil
}

func openFormulaThread(
	t *testing.T,
	granted map[string]bool,
	repo *mockFormulaTemplateRepo,
	assistant *formulaPageAssistant,
	body map[string]any,
) *testutil.GinTestContext {
	t.Helper()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})
	engine := &grantedEngine{granted: granted}
	handler := formulatemplatehandler.New(formulatemplatehandler.Params{
		Service: formulatemplateservice.New(formulatemplateservice.Params{
			Logger:         logger,
			DB:             testDBConnection{},
			Repo:           repo,
			VersionRepo:    &mockVersionRepo{},
			TestCaseRepo:   &stubTestCaseRepo{},
			ReviewRepo:     &stubReviewRepo{},
			ShipmentRepo:   &mockShipmentRepo{},
			FormulaService: newTestFormulaService(t),
			AuditService:   &mocks.NoopAuditService{},
		}),
		PageAssistant: assistant,
		ErrorHandler:  errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: engine,
				ErrorHandler:     errorHandler,
			},
		),
		PermissionEngine: engine,
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/formula-templates/ai/thread/").
		WithJSONBody(body).
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	return ginCtx
}

func allowedFormulaAssistant() map[string]bool {
	return map[string]bool{"formula_template:read": true, "assistant:create": true}
}

func TestOpenAssistantThread_OpensTheTemplatesConversation(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(
			_ context.Context,
			req repositories.GetFormulaTemplateByIDRequest,
		) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{ID: req.TemplateID, Name: "Per mile"}, nil
		},
	}
	assistant := &formulaPageAssistant{}

	ginCtx := openFormulaThread(t, allowedFormulaAssistant(), repo, assistant,
		map[string]any{"templateId": templateID.String()})

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))
	require.Len(t, assistant.opened, 1)
	assert.Equal(t, conversation.ThreadOriginFormula, assistant.opened[0].Origin)
	assert.Equal(t, agent.SubjectFormulaTemplate, assistant.opened[0].SubjectType)
	assert.Equal(t, templateID, assistant.opened[0].SubjectID)
}

func TestOpenAssistantThread_ANewTemplateHasNoSubject(t *testing.T) {
	t.Parallel()

	assistant := &formulaPageAssistant{}
	ginCtx := openFormulaThread(t, allowedFormulaAssistant(), &mockFormulaTemplateRepo{},
		assistant, map[string]any{})

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))
	require.Len(t, assistant.opened, 1)
	assert.True(t, assistant.opened[0].SubjectID.IsNil())
}

func TestOpenAssistantThread_RefusesATemplateItCannotFind(t *testing.T) {
	t.Parallel()

	assistant := &formulaPageAssistant{}
	ginCtx := openFormulaThread(t, allowedFormulaAssistant(), &mockFormulaTemplateRepo{},
		assistant, map[string]any{"templateId": pulid.MustNew("ft_").String()})

	assert.GreaterOrEqual(t, ginCtx.ResponseCode(), http.StatusBadRequest)
	assert.Empty(t, assistant.opened)

	bad := openFormulaThread(t, allowedFormulaAssistant(), &mockFormulaTemplateRepo{},
		assistant, map[string]any{"templateId": "not-an-id"})
	assert.Equal(t, http.StatusBadRequest, bad.ResponseCode())
	assert.Empty(t, assistant.opened)
}

func TestOpenAssistantThread_NeedsTheFormulaAndTheAssistant(t *testing.T) {
	t.Parallel()

	for name, granted := range map[string]map[string]bool{
		"no formula":   {"assistant:create": true},
		"no assistant": {"formula_template:read": true, "assistant:read": true},
	} {
		assistant := &formulaPageAssistant{}
		ginCtx := openFormulaThread(t, granted, &mockFormulaTemplateRepo{}, assistant,
			map[string]any{})

		assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode(), name)
		assert.Empty(t, assistant.opened, name)
	}
}
