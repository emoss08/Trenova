package reporting

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	eh *validator.ErrorHandler
	rs *reporting.Service
}

type HandlerParams struct {
	fx.In

	ErrorHandler     *validator.ErrorHandler
	ReportingService *reporting.Service
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{eh: p.ErrorHandler, rs: p.ReportingService}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/reporting")

	api.Get("/template", rl.WithRateLimit(
		[]fiber.Handler{h.getTemplate},
		middleware.PerMinute(120), // 120 reads per minute
	)...)
}

type RequestTemplate struct {
	Entity string `query:"entity"`
}

func (h *Handler) getTemplate(c *fiber.Ctx) error {
	var req RequestTemplate
	if err := c.QueryParser(&req); err != nil {
		return h.eh.HandleError(c, err)
	}

	// Sanitize and validate entity name
	entity := strings.TrimSpace(strings.ToLower(req.Entity))
	if entity == "" {
		return h.eh.HandleError(c, fiber.NewError(fiber.StatusBadRequest, "Entity name is required"))
	}

	// Prevent path traversal attempts
	if strings.Contains(entity, "..") || strings.Contains(entity, "/") || strings.Contains(entity, "\\") {
		return h.eh.HandleError(c, fiber.NewError(fiber.StatusBadRequest, "Invalid entity name"))
	}

	obj, err := h.rs.GetReportTemplate(req.Entity)
	if err != nil {
		if strings.Contains(err.Error(), "template not found") {
			return h.eh.HandleError(c, fiber.NewError(fiber.StatusNotFound, "Template not found"))
		}
		return h.eh.HandleError(c, err)
	}

	// Set appropriate headers for file download
	c.Response().Header.Set("Content-Type", "text/csv")
	c.Response().Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.csv\"", entity))
	c.Response().Header.Set("Cache-Control", "no-store, no-cache, must-revalidate")

	return c.SendFile(obj)
}
