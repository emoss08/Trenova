package documenthandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type grantedEngine struct {
	serviceports.PermissionEngine

	granted map[string]bool
	asked   []string
}

func (e *grantedEngine) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	key := req.Resource + ":" + string(req.Operation)
	e.asked = append(e.asked, key)

	return &serviceports.PermissionCheckResult{Allowed: e.granted[key]}, nil
}

type pageAssistant struct {
	opened []*serviceports.OpenPageThreadRequest
	actor  *serviceports.RequestActor
}

func (p *pageAssistant) OpenPageThread(
	_ context.Context,
	req *serviceports.OpenPageThreadRequest,
	actor *serviceports.RequestActor,
) (*serviceports.PageThread, error) {
	p.opened = append(p.opened, req)
	p.actor = actor

	agentID := pulid.MustNew("agdef_")

	return &serviceports.PageThread{
		Thread: &conversation.Thread{
			ID:                pulid.MustNew("athr_"),
			AgentDefinitionID: agentID,
			Origin:            req.Origin,
			SubjectType:       req.SubjectType,
			SubjectID:         req.SubjectID,
			CanContinue:       true,
		},
		Agent: serviceports.PageAgent{
			ID:        agentID,
			Name:      "Shipment import assistant",
			SystemKey: "import_assistant",
		},
	}, nil
}

func openImportThread(
	t *testing.T,
	engine *grantedEngine,
	assistant *pageAssistant,
	documentID string,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})
	handler := New(Params{
		PageAssistant: assistant,
		ErrorHandler:  errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: engine,
				ErrorHandler:     errorHandler,
			},
		),
		Logger: zap.NewNop(),
	})

	router := gin.New()
	router.Use(func(c *gin.Context) {
		authctx.SetAuthContext(
			c,
			pulid.MustNew("usr_"),
			pulid.MustNew("bu_"),
			pulid.MustNew("org_"),
		)
		c.Next()
	})
	handler.RegisterRoutes(router.Group(""))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodPost, "/documents/"+documentID+"/import-assistant/thread/", nil,
	))

	return recorder
}

func TestOpenImportAssistantThread_OpensTheDocumentsConversation(t *testing.T) {
	t.Parallel()

	engine := &grantedEngine{granted: map[string]bool{
		"document:read": true, "assistant:create": true,
	}}
	assistant := &pageAssistant{}
	documentID := pulid.MustNew("doc_")

	recorder := openImportThread(t, engine, assistant, documentID.String())

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Len(t, assistant.opened, 1)
	opened := assistant.opened[0]
	assert.Equal(t, conversation.ThreadOriginImport, opened.Origin)
	assert.Equal(t, agent.SubjectDocument, opened.SubjectType)
	assert.Equal(t, documentID, opened.SubjectID)
	require.NotNil(t, assistant.actor)
	assert.Equal(t, opened.TenantInfo.UserID, assistant.actor.UserID)

	var body struct {
		Thread map[string]any `json:"thread"`
		Agent  map[string]any `json:"agent"`
	}
	require.NoError(t, sonic.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "Import", body.Thread["origin"])
	assert.Equal(t, documentID.String(), body.Thread["subjectId"])
	assert.Equal(t, "import_assistant", body.Agent["systemKey"])
	assert.Equal(t, body.Thread["agentDefinitionId"], body.Agent["id"])
}

func TestOpenImportAssistantThread_NeedsTheDocumentAndTheAssistant(t *testing.T) {
	t.Parallel()

	for name, granted := range map[string]map[string]bool{
		"no document":  {"assistant:create": true},
		"no assistant": {"document:read": true, "assistant:read": true},
	} {
		assistant := &pageAssistant{}
		recorder := openImportThread(t, &grantedEngine{granted: granted}, assistant,
			pulid.MustNew("doc_").String())

		assert.Equal(t, http.StatusForbidden, recorder.Code, name)
		assert.Empty(t, assistant.opened, name)
	}
}

func TestOpenImportAssistantThread_RefusesAnIDThatIsNotOne(t *testing.T) {
	t.Parallel()

	assistant := &pageAssistant{}
	recorder := openImportThread(t, &grantedEngine{granted: map[string]bool{
		"document:read": true, "assistant:create": true,
	}}, assistant, "not-an-id")

	assert.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	assert.Empty(t, assistant.opened)
}
