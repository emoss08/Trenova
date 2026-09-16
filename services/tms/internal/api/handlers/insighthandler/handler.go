// Package insighthandler serves the operational insights panel.
package insighthandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/insightservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// maxLimit bounds what one request may ask for. The panel is a home-screen
// widget, not a report, and an unbounded limit is a way to make the permission
// filter do a lot of work for nothing.
const maxLimit = 25

type Params struct {
	fx.In

	Service              *insightservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *insightservice.Service
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
	api := rg.Group("/insights")
	resource := permission.ResourceInsight.String()

	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	// Dismissing changes what everyone in the organization sees on their home
	// screen, so it is an update to the insight rather than a per-reader
	// preference, and it is gated as one.
	api.POST(
		"/:insightID/dismiss/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.dismiss,
	)
}

func tenantFromAuthContext(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	found, err := h.service.ListActive(c.Request.Context(), serviceports.ListInsightsRequest{
		TenantInfo: tenantFromAuthContext(authCtx),
		UserID:     authCtx.UserID,
		Categories: parseCategories(c.QueryArray("category")),
		Limit:      parseLimit(c.Query("limit")),
	})
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	c.JSON(http.StatusOK, gin.H{"results": found})
}

type dismissRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) dismiss(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	insightID, err := pulid.Parse(c.Param("insightID"))
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	// A reason is optional: making it mandatory would only teach people to type
	// a full stop. The field exists because "why did we ignore this" is the
	// question asked three months later.
	var body dismissRequest
	if err = c.ShouldBindJSON(&body); err != nil && c.Request.ContentLength > 0 {
		h.eh.HandleError(c, err)

		return
	}

	dismissed, err := h.service.Dismiss(c.Request.Context(), serviceports.DismissInsightRequest{
		ID:         insightID,
		UserID:     authCtx.UserID,
		Reason:     body.Reason,
		TenantInfo: tenantFromAuthContext(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	c.JSON(http.StatusOK, dismissed)
}

// parseCategories keeps only categories the domain recognises.
//
// An unknown value is dropped rather than rejected: a client sending a category
// this build has not heard of should get the insights it can understand, not an
// error, which is what makes adding a category a non-breaking change.
func parseCategories(values []string) []insight.Category {
	if len(values) == 0 {
		return nil
	}

	categories := make([]insight.Category, 0, len(values))
	for _, value := range values {
		category := insight.Category(value)
		if category.IsValid() {
			categories = append(categories, category)
		}
	}

	return categories
}

func parseLimit(value string) int {
	if value == "" {
		return 0
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return 0
	}

	return min(limit, maxLimit)
}
