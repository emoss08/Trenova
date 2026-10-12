package notificationpreferencehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationpreferenceservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      *notificationpreferenceservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *notificationpreferenceservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/users/me/notification-preferences")
	api.GET("/", h.get)
	api.PUT("/", h.update)
}

type UpdateRequest struct {
	DisabledEventTypes    []string                  `json:"disabledEventTypes"`
	DesktopMode           notification.DeliveryMode `json:"desktopMode"`
	EmailMode             notification.DeliveryMode `json:"emailMode"`
	EmailDigest           bool                      `json:"emailDigest"`
	Sound                 bool                      `json:"sound"`
	QuietHoursEnabled     bool                      `json:"quietHoursEnabled"`
	QuietHoursStart       string                    `json:"quietHoursStart"`
	QuietHoursEnd         string                    `json:"quietHoursEnd"`
	QuietHoursDays        []int                     `json:"quietHoursDays"`
	QuietHoursAllowUrgent bool                      `json:"quietHoursAllowUrgent"`
	PausedUntil           *int64                    `json:"pausedUntil"`
}

func selfTenant(c *gin.Context) pagination.TenantInfo {
	authCtx := authctx.GetAuthContext(c)
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

// @Summary Get the current user's notification preferences
// @ID getCurrentUserNotificationPreferences
// @Tags Notifications
// @Produce json
// @Success 200 {object} notification.Preference
// @Failure 401 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/notification-preferences/ [get]
func (h *Handler) get(c *gin.Context) {
	pref, err := h.service.Get(c.Request.Context(), selfTenant(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, pref)
}

// @Summary Replace the current user's notification preferences
// @ID updateCurrentUserNotificationPreferences
// @Tags Notifications
// @Accept json
// @Produce json
// @Param request body UpdateRequest true "Preferences"
// @Success 200 {object} notification.Preference
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/notification-preferences/ [put]
func (h *Handler) update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	saved, err := h.service.Update(c.Request.Context(), selfTenant(c), &notification.Preference{
		DisabledEventTypes:    req.DisabledEventTypes,
		DesktopMode:           req.DesktopMode,
		EmailMode:             req.EmailMode,
		EmailDigest:           req.EmailDigest,
		Sound:                 req.Sound,
		QuietHoursEnabled:     req.QuietHoursEnabled,
		QuietHoursStart:       req.QuietHoursStart,
		QuietHoursEnd:         req.QuietHoursEnd,
		QuietHoursDays:        req.QuietHoursDays,
		QuietHoursAllowUrgent: req.QuietHoursAllowUrgent,
		PausedUntil:           req.PausedUntil,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, saved)
}
