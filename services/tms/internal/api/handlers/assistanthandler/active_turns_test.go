package assistanthandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// liveRecords is a person's replies in progress. Anything else asked of it
// panics, which is how a test learns the route went somewhere it should not.
type liveRecords struct {
	repositories.AssistantTurnRepository

	live  []*repositories.LiveAssistantTurn
	asked *repositories.ListLiveAssistantTurnsRequest
}

func (r *liveRecords) ListLive(
	_ context.Context,
	req repositories.ListLiveAssistantTurnsRequest,
) ([]*repositories.LiveAssistantTurn, error) {
	r.asked = &req

	return r.live, nil
}

func getActiveTurns(
	t *testing.T,
	records *liveRecords,
	userID, buID, orgID pulid.ID,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})
	handler := New(Params{
		Service: &reachedService{},
		Turns: assistantturnservice.New(assistantturnservice.Params{
			Logger: zap.NewNop(),
			Turns:  records,
		}),
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
		authctx.SetAuthContext(c, userID, buID, orgID)
		c.Next()
	})
	handler.RegisterRoutes(router.Group(""))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assistant/turns/active/", nil))

	return recorder
}

/*
Every reply the caller has in progress, across their conversations, so a tab
that did not start one can still show it. "active" sits where a turn id would,
and must never be read as one.
*/
func TestActiveTurns_ListsTheCallersRepliesInProgress(t *testing.T) {
	t.Parallel()

	userID, buID, orgID := pulid.MustNew("usr_"), pulid.MustNew("bu_"), pulid.MustNew("org_")
	turn := conversation.AssistantTurn{
		ID:        pulid.MustNew("atrn_"),
		ThreadID:  pulid.MustNew("athr_"),
		UserID:    userID,
		Origin:    conversation.AssistantTurnOriginDecisionFollowUp,
		Status:    conversation.AssistantTurnStatusRunning,
		StartedAt: 1_700_000_000,
	}
	records := &liveRecords{live: []*repositories.LiveAssistantTurn{{
		AssistantTurn: turn,
		ThreadTitle:   "Late loads this week",
	}}}

	recorder := getActiveTurns(t, records, userID, buID, orgID)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.NotNil(t, records.asked, "the route reached the list, not a turn read")
	assert.Equal(t, userID, records.asked.UserID, "only the caller's own replies are listed")
	assert.Equal(t, orgID, records.asked.TenantInfo.OrgID)
	assert.Equal(t, buID, records.asked.TenantInfo.BuID)
	assert.ElementsMatch(t, conversation.PageBoundOrigins(), records.asked.ExcludeOrigins,
		"a reply on an import or formula page is read on that page, not listed")

	var body map[string][]map[string]any
	require.NoError(t, sonic.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body["items"], 1)
	assert.Equal(t, map[string]any{
		"turnId":      turn.ID.String(),
		"threadId":    turn.ThreadID.String(),
		"threadTitle": "Late loads this week",
		"origin":      "DecisionFollowUp",
		"startedAt":   float64(1_700_000_000),
	}, body["items"][0])
}

// Nothing in progress is an empty list, not a null a client has to guard.
func TestActiveTurns_IsAnEmptyListWhenNothingIsInProgress(t *testing.T) {
	t.Parallel()

	recorder := getActiveTurns(t, &liveRecords{},
		pulid.MustNew("usr_"), pulid.MustNew("bu_"), pulid.MustNew("org_"))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"items":[]}`, recorder.Body.String())
}
