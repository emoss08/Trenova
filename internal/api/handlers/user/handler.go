package user

import (
	"github.com/gofiber/fiber/v2"
	"github.com/trenova-app/transport/internal/api/middleware"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/core/services/user"
	"github.com/trenova-app/transport/internal/pkg/ctx"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"github.com/trenova-app/transport/pkg/types"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"go.uber.org/fx"
)

type Handler struct {
	uh *user.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	UserService  *user.Service
	ErrorHandler *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{uh: p.UserService, eh: p.ErrorHandler}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/users")

	api.Get("/select-options", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Get("/me", rl.WithRateLimit(
		[]fiber.Handler{h.me},
		middleware.PerSecond(20), // 20 reads per second
	)...)

	api.Get("/:userID", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)
}

func (h Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &ports.LimitOffsetQueryOptions{
		TenantOpts: &ports.TenantOptions{
			OrgID:  reqCtx.OrgID,
			BuID:   reqCtx.BuID,
			UserID: reqCtx.UserID,
		},
		Query:  c.Query("search"),
		Limit:  c.QueryInt("limit", 100),
		Offset: c.QueryInt("offset", 0),
	}

	options, err := h.uh.SelectOptions(c.UserContext(), opts)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ports.Response[[]*types.SelectOption]{
		Results: options,
		Next:    "",
		Prev:    "",
	})
}

func (h Handler) get(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	userID, err := pulid.MustParse(c.Params("userID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	usr, err := h.uh.GetByID(c.UserContext(), &repositories.GetUserByIDOptions{
		OrgID:        reqCtx.OrgID,
		BuID:         reqCtx.BuID,
		UserID:       userID,
		IncludeRoles: true,
		IncludeOrgs:  true,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(usr)
}

func (h Handler) me(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	usr, err := h.uh.GetByID(c.UserContext(), &repositories.GetUserByIDOptions{
		OrgID:        reqCtx.OrgID,
		BuID:         reqCtx.BuID,
		UserID:       reqCtx.UserID,
		IncludeRoles: true,
		IncludeOrgs:  false,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(usr)
}
