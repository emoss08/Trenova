// Package insighthandler serves the operational insights panel.
package insighthandler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/insightservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// maxLimit bounds what one request may ask for. The panel is a home-screen
// widget, not a report, and an unbounded limit is a way to make the permission
// filter do a lot of work for nothing.
const (
	maxLimit = 25
	// The page shows more than the widget and can be paged through, but a single
	// request still has a ceiling: the permission filter runs per detector, and
	// an unbounded page is a way to make the database do a lot of work at once.
	maxBrowseLimit  = 100
	maxBrowseOffset = 10_000
)

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

	// The home widget reads the active slice; the page browses the history. They
	// are separate endpoints rather than one with a mode flag because their
	// defaults differ in a way that matters: a widget that accidentally returned
	// dismissed findings would put back the cards someone just cleared.
	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.GET("/browse/", h.pm.RequirePermission(resource, permission.OpRead), h.browse)
	api.GET("/:insightID/", h.pm.RequirePermission(resource, permission.OpRead), h.detail)
	// Dismissing changes what everyone in the organization sees on their home
	// screen, so it is an update to the insight rather than a per-reader
	// preference, and it is gated as one.
	api.POST(
		"/:insightID/dismiss/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.dismiss,
	)
	api.POST(
		"/:insightID/restore/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.restore,
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
		Surface:    parseSurface(c.Query("surface")),
		Limit:      parseLimit(c.Query("limit")),
	})
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	c.JSON(http.StatusOK, gin.H{"results": found})
}

func (h *Handler) browse(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	result, err := h.service.List(c.Request.Context(), serviceports.BrowseInsightsRequest{
		TenantInfo: tenantFromAuthContext(authCtx),
		UserID:     authCtx.UserID,
		Categories: parseCategories(c.QueryArray("category")),
		Severities: parseSeverities(c.QueryArray("severity")),
		Statuses:   parseStatuses(c.QueryArray("status")),
		Limit:      parseInt(c.Query("limit"), maxBrowseLimit),
		Offset:     parseInt(c.Query("offset"), maxBrowseOffset),
	})
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	c.JSON(http.StatusOK, gin.H{"results": result.Items, "total": result.Total})
}

// detail reads one finding with its history and the rule behind it.
//
// A finding the reader may not see is reported as not found rather than
// forbidden. Saying "this exists but is not for you" about a card that names a
// customer and a revenue figure is itself a disclosure, and there is nothing a
// reader can do with the distinction.
func (h *Handler) detail(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	insightID, err := pulid.Parse(c.Param("insightID"))
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	found, err := h.service.GetDetail(c.Request.Context(), serviceports.GetInsightDetailRequest{
		ID:         insightID,
		UserID:     authCtx.UserID,
		TenantInfo: tenantFromAuthContext(authCtx),
	})
	if err != nil {
		if errors.Is(err, insightservice.ErrInsightNotVisible) {
			h.eh.HandleError(c, errortypes.NewNotFoundError("Insight not found"))

			return
		}

		h.eh.HandleError(c, err)

		return
	}

	c.JSON(http.StatusOK, found)
}

func (h *Handler) restore(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	insightID, err := pulid.Parse(c.Param("insightID"))
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	restored, err := h.service.Restore(c.Request.Context(), serviceports.RestoreInsightRequest{
		ID:         insightID,
		TenantInfo: tenantFromAuthContext(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	c.JSON(http.StatusOK, restored)
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

// parseSurface reads the page asking for its own slice.
//
// An unknown surface is treated as absent rather than rejected, for the same
// reason as an unknown category — but note what "absent" means here: the home
// screen's full view, not nothing. A page that names a surface this build has
// not heard of gets everything the reader may see, which is more than it asked
// for and never less.
func parseSurface(value string) insight.Surface {
	surface := insight.Surface(value)
	if !surface.IsValid() {
		return ""
	}

	return surface
}

// parseSeverities and parseStatuses drop values the domain does not recognise,
// for the same reason parseCategories does: a filter this build has not heard of
// should narrow nothing rather than fail the request.
func parseSeverities(values []string) []insight.Severity {
	severities := make([]insight.Severity, 0, len(values))
	for _, value := range values {
		severity := insight.Severity(value)
		if severity.IsValid() {
			severities = append(severities, severity)
		}
	}

	if len(severities) == 0 {
		return nil
	}

	return severities
}

func parseStatuses(values []string) []insight.Status {
	statuses := make([]insight.Status, 0, len(values))
	for _, value := range values {
		status := insight.Status(value)
		if status.IsValid() {
			statuses = append(statuses, status)
		}
	}

	if len(statuses) == 0 {
		return nil
	}

	return statuses
}

func parseLimit(value string) int {
	return parseInt(value, maxLimit)
}

// parseInt reads a bounded non-negative query parameter, treating anything
// unreadable as absent so a malformed URL falls back to the default rather than
// erroring on a page someone is just trying to open.
func parseInt(value string, ceiling int) int {
	if value == "" {
		return 0
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0
	}

	return min(parsed, ceiling)
}
