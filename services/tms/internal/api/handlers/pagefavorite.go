package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	pagefavoriteservice "github.com/emoss08/trenova/internal/core/services/pagefavorite"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type PageFavoriteHandlerParams struct {
	fx.In

	Service      *pagefavoriteservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type PageFavoriteHandler struct {
	service *pagefavoriteservice.Service
	eh      *helpers.ErrorHandler
}

func NewPageFavoriteHandler(p PageFavoriteHandlerParams) *PageFavoriteHandler {
	return &PageFavoriteHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *PageFavoriteHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/favorites/")
	api.GET("", h.list)
	api.POST("check/", h.checkFavoriteByPost)
	api.POST("toggle/", h.toggle)
	api.DELETE(":id/", h.delete)
}

type CheckFavoriteRequest struct {
	PageURL string `json:"pageUrl" binding:"required,url,max=500"`
}

func (h *PageFavoriteHandler) list(c *gin.Context) {
	pagination.Handle[*pagefavorite.PageFavorite](c, context.GetAuthContext(c)).
		WithErrorHandler(h.eh).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*pagefavorite.PageFavorite], error) {
			return h.service.List(c.Request.Context(), opts)
		})
}

func (h *PageFavoriteHandler) checkFavoriteByPost(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req CheckFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	fav, err := h.service.GetByURL(c.Request.Context(), repositories.GetPageFavoriteByURLRequest{
		OrgID:   authCtx.OrganizationID,
		BuID:    authCtx.BusinessUnitID,
		UserID:  authCtx.UserID,
		PageURL: req.PageURL,
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"isFavorite": false, "favorite": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"isFavorite": true, "favorite": fav})
}

func (h *PageFavoriteHandler) toggle(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req pagefavoriteservice.TogglePageFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	context.AddContextToRequest(authCtx, &req)

	result, err := h.service.Toggle(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if result == nil {
		c.JSON(http.StatusOK, gin.H{"action": "removed", "favorite": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"action": "added", "favorite": result})
}

func (h *PageFavoriteHandler) delete(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	favoriteID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.Delete(c.Request.Context(), repositories.DeletePageFavoriteRequest{
		OrgID:      authCtx.OrganizationID,
		BuID:       authCtx.BusinessUnitID,
		UserID:     authCtx.UserID,
		FavoriteID: favoriteID,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
