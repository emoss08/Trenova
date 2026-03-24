package pagefavoritehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/pagefavoriteservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

var _ *pagefavorite.PageFavorite

type Params struct {
	fx.In

	Service      *pagefavoriteservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *pagefavoriteservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/page-favorites")
	api.GET("/", h.list)
	api.POST("/toggle", h.toggle)
	api.GET("/check", h.check)
}

// @Summary List page favorites
// @Description Returns the current user's page favorites.
// @ID listPageFavorites
// @Tags Page Favorites
// @Produce json
// @Success 200 {array} pagefavorite.PageFavorite
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /page-favorites/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	favorites, err := h.service.List(c.Request.Context(), &repositories.ListPageFavoritesRequest{
		UserID: authCtx.UserID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, favorites)
}

type toggleRequest struct {
	PageURL   string `json:"pageUrl"   binding:"required"`
	PageTitle string `json:"pageTitle" binding:"required"`
}

// @Summary Toggle a page favorite
// @Description Adds or removes a page favorite for the current user.
// @ID togglePageFavorite
// @Tags Page Favorites
// @Accept json
// @Produce json
// @Param request body toggleRequest true "Page favorite toggle request"
// @Success 200 {object} pagefavoriteservice.ToggleResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /page-favorites/toggle [post]
func (h *Handler) toggle(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req toggleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.Toggle(c.Request.Context(), &pagefavoriteservice.ToggleRequest{
		PageURL:   req.PageURL,
		PageTitle: req.PageTitle,
		UserID:    authCtx.UserID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Check whether a page is favorited
// @Description Returns whether the current user has favorited the requested page URL.
// @ID checkPageFavorite
// @Tags Page Favorites
// @Produce json
// @Param pageUrl query string true "Page URL"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /page-favorites/check [get]
func (h *Handler) check(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	pageURL := helpers.QueryString(c, "pageUrl")

	if pageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pageUrl query parameter is required"})
		return
	}

	favorited, err := h.service.IsFavorited(
		c.Request.Context(),
		pageURL,
		authCtx.UserID,
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"favorited": favorited})
}
