package user

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	userdomain "github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/user"

	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
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

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Get("/select-options/", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Get("/me/", rl.WithRateLimit(
		[]fiber.Handler{h.me},
		middleware.PerSecond(20), // 20 reads per second
	)...)

	api.Get("/:userID/", rl.WithRateLimit(
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

func (h Handler) list(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*userdomain.User], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.uh.List(fc.UserContext(), filter)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
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

	usr, err := h.uh.Get(c.UserContext(), repositories.GetUserByIDOptions{
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

	usr, err := h.uh.Get(c.UserContext(), repositories.GetUserByIDOptions{
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
