package inboundhandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

/*
The webhook URL a person copies when a mailbox is created is built by
WebhookPath; the one that answers is the route the handler registers. If they
drifted apart, every mailbox would be handed a URL that 404s — so the path is
routed through a real engine mounted the way the router mounts it.
*/
func TestWebhookPathIsTheRouteTheHandlerAnswers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	matched := ""
	engine.Group(apiPrefix).POST(webhookRoute, func(c *gin.Context) {
		matched = c.Param("mailboxToken")
		c.Status(http.StatusNoContent)
	})

	token := "Ab-_09xyz"
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, WebhookPath(token), nil))

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, token, matched)
}
