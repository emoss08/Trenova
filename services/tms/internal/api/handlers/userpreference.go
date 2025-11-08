package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/userpreference"
	userpreferencesvc "github.com/emoss08/trenova/internal/core/services/userpreference"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type UserPreferenceHandlerParams struct {
	fx.In

	Service      *userpreferencesvc.Service
	ErrorHandler *helpers.ErrorHandler
}

type UserPreferenceHandler struct {
	service *userpreferencesvc.Service
	eh      *helpers.ErrorHandler
}

func NewUserPreferenceHandler(p UserPreferenceHandlerParams) *UserPreferenceHandler {
	return &UserPreferenceHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *UserPreferenceHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/users/me/preferences")
	api.GET("", h.get)
	api.PUT("", h.update)
	api.PATCH("", h.merge)
}

func (h *UserPreferenceHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	up, err := h.service.GetOrCreateByUserID(
		c.Request.Context(),
		authCtx.UserID,
		authCtx.OrganizationID,
		authCtx.BusinessUnitID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, up)
}

func (h *UserPreferenceHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	up := new(userpreference.UserPreference)
	up.UserID = authCtx.UserID
	up.OrganizationID = authCtx.OrganizationID
	up.BusinessUnitID = authCtx.BusinessUnitID

	if err := c.ShouldBindJSON(up); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.Upsert(c.Request.Context(), up)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *UserPreferenceHandler) merge(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	updates := new(userpreference.PreferenceData)
	if err := c.ShouldBindJSON(updates); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	merged, err := h.service.MergePreferences(c.Request.Context(), authCtx.UserID, updates)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, merged)
}
