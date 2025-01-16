package tableconfiguration

import (
	"github.com/gofiber/fiber/v2"
	"github.com/trenova-app/transport/internal/api/middleware"
	tableconfigurationdomain "github.com/trenova-app/transport/internal/core/domain/tableconfiguration"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/core/services/tableconfiguration"
	"github.com/trenova-app/transport/internal/pkg/ctx"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"go.uber.org/fx"
)

type Handler struct {
	ts *tableconfiguration.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	TableConfigurationService *tableconfiguration.Service
	ErrorHandler              *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ts: p.TableConfigurationService, eh: p.ErrorHandler}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/table-configurations")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60),
	)...)
}

func (h Handler) list(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &ports.FilterQueryOptions{
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
		Query:  c.Query("query"),
	}

	result, err := h.ts.List(c.UserContext(),
		&repositories.TableConfigurationFilters{
			Base:           opts,
			Search:         c.Query("search"),
			IncludeShares:  c.Query("include_shares") == "true",
			IncludeCreator: c.Query("include_creator") == "true",
		})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func (h Handler) create(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	config := new(tableconfigurationdomain.Configuration)
	config.OrganizationID = reqCtx.OrgID
	config.BusinessUnitID = reqCtx.BuID
	config.UserID = reqCtx.UserID

	if err = c.BodyParser(config); err != nil {
		return h.eh.HandleError(c, err)
	}

	created, err := h.ts.Create(c.UserContext(), config)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(created)
}
