package assistanthandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// readOnlyEngine allows exactly one operation on the assistant and refuses
// everything else, which is what a person who may use the assistant but holds
// no administrative grant over it looks like.
type readOnlyEngine struct {
	permissionports.PermissionEngine

	asked []permission.Operation
}

func (e *readOnlyEngine) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	e.asked = append(e.asked, req.Operation)

	return &permissionports.PermissionCheckResult{
		Allowed: req.Resource == permission.ResourceAssistant.String() &&
			req.Operation == permission.OpRead,
	}, nil
}

// reachedService records that a request got past the permission middleware.
type reachedService struct {
	assistantports.AssistantService

	reached       bool
	listed        *conversationports.ListThreadsRequest
	counted       *conversationports.CountThreadShelvesRequest
	commands      *assistantports.CommandAvailabilityRequest
	commandsActor *permissionports.RequestActor
}

func (s *reachedService) ListThreads(
	_ context.Context,
	req conversationports.ListThreadsRequest,
) (*conversationports.ThreadPage, error) {
	s.reached = true
	s.listed = &req

	return &conversationports.ThreadPage{}, nil
}

func (s *reachedService) CountThreadShelves(
	_ context.Context,
	req *conversationports.CountThreadShelvesRequest,
) (*conversationports.ThreadShelfCounts, error) {
	s.reached = true
	s.counted = req

	return &conversationports.ThreadShelfCounts{}, nil
}

func (s *reachedService) UpdateThread(
	context.Context,
	*assistantports.UpdateThreadRequest,
	*permissionports.RequestActor,
) (*conversation.Thread, error) {
	s.reached = true

	return &conversation.Thread{}, nil
}

func (s *reachedService) DeleteThread(context.Context, conversationports.GetThreadRequest) error {
	s.reached = true

	return nil
}

func (s *reachedService) PinArtifact(
	context.Context,
	conversationports.GetThreadRequest,
	pulid.ID,
	bool,
) (*assistantartifactports.AssistantArtifact, error) {
	s.reached = true

	return &assistantartifactports.AssistantArtifact{}, nil
}

func (s *reachedService) SaveProposalEdits(
	context.Context,
	conversationports.GetThreadRequest,
	pulid.ID,
	map[string]any,
) (*assistantproposalports.ProposalEdits, error) {
	s.reached = true

	return &assistantproposalports.ProposalEdits{}, nil
}

func (s *reachedService) CommandAvailability(
	_ context.Context,
	req *assistantports.CommandAvailabilityRequest,
	actor *permissionports.RequestActor,
) (*assistantports.CommandAvailability, error) {
	s.reached = true
	s.commands = req
	s.commandsActor = actor

	return &assistantports.CommandAvailability{
		Commands: []agentextension.Command{agentextension.CommandSearch},
	}, nil
}

func call(t *testing.T, method, path, body string) (*reachedService, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	service := &reachedService{}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    zap.NewNop(),
		AppConfig: &appconfig.Config{Debug: true},
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
		authctx.SetAuthContext(
			c,
			pulid.MustNew("usr_"),
			pulid.MustNew("bu_"),
			pulid.MustNew("org_"),
		)
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
		{
			name:   "save edits to a draft",
			method: http.MethodPut,
			path:   threadPath + "proposals/" + pulid.MustNew("ap_").String() + "/edits/",
			body:   `{"modifications":{"subject":"PO needed"}}`,
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

func TestCommandAvailabilityNeedsOnlyTheAssistant(t *testing.T) {
	t.Parallel()

	agentID := pulid.MustNew("agdef_")
	service, status := call(
		t, http.MethodGet, "/assistant/agents/"+agentID.String()+"/commands/", "",
	)

	assert.Equal(t, http.StatusOK, status)
	require.NotNil(t, service.commands)
	assert.Equal(t, agentID, service.commands.AgentID)
	require.NotNil(t, service.commandsActor)
	assert.Equal(t, service.commandsActor.OrganizationID, service.commands.TenantInfo.OrgID)
	assert.Equal(t, service.commandsActor.UserID, service.commands.TenantInfo.UserID)
}

func TestCommandAvailabilityRefusesAMalformedAgent(t *testing.T) {
	t.Parallel()

	service, status := call(t, http.MethodGet, "/assistant/agents/not-an-id/commands/", "")

	assert.NotEqual(t, http.StatusOK, status)
	assert.False(t, service.reached)
}
