package assistanthandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// readOnlyEngine allows exactly one operation on the assistant and refuses
// everything else, which is what a person who may use the assistant but holds
// no administrative grant over it looks like.
type readOnlyEngine struct {
	services.PermissionEngine

	asked []permission.Operation
}

func (e *readOnlyEngine) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	e.asked = append(e.asked, req.Operation)

	return &services.PermissionCheckResult{
		Allowed: req.Resource == permission.ResourceAssistant.String() &&
			req.Operation == permission.OpRead,
	}, nil
}

// reachedService records that a request got past the permission middleware.
type reachedService struct {
	services.AssistantService

	reached bool
}

func (s *reachedService) UpdateThread(
	context.Context,
	*services.UpdateThreadRequest,
	*services.RequestActor,
) (*conversation.Thread, error) {
	s.reached = true

	return &conversation.Thread{}, nil
}

func (s *reachedService) DeleteThread(context.Context, repositories.GetThreadRequest) error {
	s.reached = true

	return nil
}

func (s *reachedService) PinArtifact(
	context.Context,
	repositories.GetThreadRequest,
	pulid.ID,
	bool,
) (*services.AssistantArtifact, error) {
	s.reached = true

	return &services.AssistantArtifact{}, nil
}

func call(t *testing.T, method, path, body string) (*reachedService, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	service := &reachedService{}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})
	handler := New(Params{
		Service: service,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: &readOnlyEngine{},
				ErrorHandler:     errorHandler,
			},
		),
		ErrorHandler: errorHandler,
		Logger:       zap.NewNop(),
	})

	router := gin.New()
	router.Use(func(c *gin.Context) {
		authctx.SetAuthContext(c, pulid.MustNew("usr_"), pulid.MustNew("bu_"), pulid.MustNew("org_"))
		c.Next()
	})
	handler.RegisterRoutes(router.Group(""))

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return service, recorder.Code
}

/*
A conversation is the person's own. Every route here reads it under their user
id, so someone else's is not found rather than refused — ownership is the
authorization and there is no way around it.

Asking for assistant:update on top of that gated a person's own desk behind an
organization-wide grant most people have no reason to hold: they could hold a
conversation and not rename it, pin it, or throw it away.
*/
func TestOwnConversationNeedsOnlyTheAssistant(t *testing.T) {
	t.Parallel()

	threadPath := "/assistant/threads/" + pulid.MustNew("athr_").String() + "/"

	for _, tt := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "rename or pin", method: http.MethodPatch, path: threadPath, body: `{"pinned":true}`},
		{name: "delete", method: http.MethodDelete, path: threadPath, body: ""},
		{
			name:   "pin an artifact",
			method: http.MethodPost,
			path:   threadPath + "artifacts/" + pulid.MustNew("art_").String() + "/pin/",
			body:   `{"pinned":true}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, status := call(t, tt.method, tt.path, tt.body)

			assert.NotEqual(t, http.StatusForbidden, status)
			assert.True(t, service.reached, "the request never reached the service")
		})
	}
}

// Starting a conversation and sending into it spend a budget and reach tools,
// so they stay behind their own grant.
func TestStartingAConversationStillNeedsMoreThanReading(t *testing.T) {
	t.Parallel()

	service, status := call(t, http.MethodPost, "/assistant/threads/", `{"agentDefinitionId":"x"}`)

	assert.Equal(t, http.StatusForbidden, status)
	assert.False(t, service.reached)
}
