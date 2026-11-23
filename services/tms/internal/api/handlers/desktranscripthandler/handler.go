package desktranscripthandler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              desktranscriptports.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service desktranscriptports.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/desk-transcripts")
	api.GET(
		"/:threadID/download/",
		h.pm.RequirePermission(permission.ResourceDeskTranscript.String(), permission.OpExport),
		h.download,
	)
}

func (h *Handler) download(c *gin.Context) {
	threadID, err := pulid.Parse(c.Param("threadID"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"threadID",
			errortypes.ErrInvalidFormat,
			"Conversation is not one Trenova knows",
		))
		return
	}

	file, err := h.service.DownloadConversation(
		c.Request.Context(),
		desktranscriptports.ConversationRequest{
			Actor: actorutil.FromAuthContext(authctx.GetAuthContext(c)),
			ID:    threadID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Content-Disposition", contentDisposition(file.FileName))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(file.Body))
}

func contentDisposition(name string) string {
	ascii := strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, name)

	return fmt.Sprintf(
		"attachment; filename=%q; filename*=UTF-8''%s",
		ascii,
		url.PathEscape(name),
	)
}
