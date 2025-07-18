package user

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/session"
	userdomain "github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/user"

	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	UserService  *user.Service
	ErrorHandler *validator.ErrorHandler
}

type Handler struct {
	uh *user.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{uh: p.UserService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/users")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Post("/change-password/", rl.WithRateLimit(
		[]fiber.Handler{h.changePassword},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Get("/select-options/", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Get("/me/", rl.WithRateLimit(
		[]fiber.Handler{h.me},
		middleware.PerSecond(50), // 50 reads per second
	)...)

	api.Get("/:userID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:userID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Put("/:userID/switch-organization/", rl.WithRateLimit(
		[]fiber.Handler{h.switchOrganization},
		middleware.PerMinute(30), // 30 organization switches per minute
	)...)
}

func (h *Handler) selectOptions(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &ports.LimitOffsetQueryOptions{
		TenantOpts: ports.TenantOptions{
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

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*userdomain.User], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.uh.List(fc.UserContext(), repositories.ListUserRequest{
			Filter:       filter,
			IncludeRoles: fc.QueryBool("includeRoles", false),
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
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

func (h *Handler) me(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
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

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	u := new(userdomain.User)
	u.CurrentOrganizationID = reqCtx.OrgID
	u.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(u); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.uh.Create(c.UserContext(), u, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	userID, err := pulid.MustParse(c.Params("userID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	u := new(userdomain.User)
	u.ID = userID
	u.CurrentOrganizationID = reqCtx.OrgID
	u.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(u); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.uh.Update(c.UserContext(), u, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h *Handler) changePassword(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var req repositories.ChangePasswordRequest
	if err = c.BodyParser(&req); err != nil {
		return h.eh.HandleError(c, err)
	}

	req.OrgID = reqCtx.OrgID
	req.BuID = reqCtx.BuID
	req.UserID = reqCtx.UserID

	entity, err := h.uh.ChangePassword(c.UserContext(), req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

type SwitchOrganizationRequest struct {
	OrganizationID pulid.ID `json:"organizationId" validate:"required"`
}

func (h *Handler) switchOrganization(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	userID, err := pulid.MustParse(c.Params("userID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// * Only allow users to switch their own organization or admin users
	if userID != reqCtx.UserID {
		return h.eh.HandleError(c, fiber.NewError(
			fiber.StatusForbidden,
			"You can only switch your own organization",
		))
	}

	var req SwitchOrganizationRequest
	if err = c.BodyParser(&req); err != nil {
		return h.eh.HandleError(c, err)
	}

	// * Get the session ID from context
	var sessionID pulid.ID
	if sess, ok := c.Locals(appctx.CTXSessionID).(*session.Session); ok && sess != nil {
		sessionID = sess.ID
	}

	updatedUser, err := h.uh.SwitchOrganization(
		c.UserContext(),
		userID,
		req.OrganizationID,
		sessionID,
	)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedUser)
}
