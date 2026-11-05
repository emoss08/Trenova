package edihandler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func as2Router() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := &Handler{eh: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    zap.NewNop(),
		AppConfig: &appconfig.Config{},
	})}
	handler.RegisterPublicRoutes(router.Group("/"))

	return router
}

func TestReceiveAS2RefusesADeclaredOversizedBody(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(
		http.MethodPost,
		"/edi/as2/inbound/PROFILEADDRESS/",
		bytes.NewReader([]byte("x")),
	)
	request.ContentLength = as2MaxRequestBody + 1

	recorder := httptest.NewRecorder()
	as2Router().ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
}

func TestReceiveAS2RefusesAnUndeclaredBodyPastTheLimitInsteadOfTruncatingIt(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(
		http.MethodPost,
		"/edi/as2/inbound/PROFILEADDRESS/",
		io.LimitReader(bytes.NewReader(make([]byte, as2MaxRequestBody+1)), as2MaxRequestBody+1),
	)
	request.ContentLength = -1

	recorder := httptest.NewRecorder()
	as2Router().ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
}

func TestAS2MessagesAreOnlyReceivedAtAProfilesOwnAddress(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	as2Router().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/edi/as2/inbound/", bytes.NewReader([]byte("x"))),
	)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}
