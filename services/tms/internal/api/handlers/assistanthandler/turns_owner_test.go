package assistanthandler

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type strangerThreads struct {
	services.AssistantService

	asked repositories.GetThreadRequest
}

func (s *strangerThreads) GetThread(
	_ context.Context,
	req repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	s.asked = req

	return nil, errortypes.NewNotFoundError("Thread not found")
}

/*
A conversation has one live reply at a time. Recording the turn before
checking whose conversation it was let anyone who knew a thread's id take that
slot, and the owner was told the conversation was already working on a reply.
*/
func TestAskWorker_ChecksTheConversationIsThePersonsBeforeTakingIt(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil)
	userID := pulid.MustNew("usr_")
	authctx.SetAuthContext(c, userID, pulid.MustNew("bu_"), pulid.MustNew("org_"))

	threads := &strangerThreads{}
	// turns is nil: reaching it would panic, which is the point. Nothing is
	// recorded for a conversation that is not the person's.
	h := &Handler{service: threads, logger: zap.NewNop()}
	threadID := pulid.MustNew("athr_")

	turn, run, err := h.askWorker(c, threadID, &sendMessageRequest{Content: "hello"})

	require.Error(t, err)
	assert.Nil(t, turn)
	assert.Nil(t, run)
	assert.Equal(t, threadID, threads.asked.ID)
	assert.Equal(t, userID, threads.asked.UserID)
}
