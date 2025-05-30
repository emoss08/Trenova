package permission

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	permdomain "github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	ps services.PermissionService
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	PermissionService services.PermissionService
	ErrorHandler      *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ps: p.PermissionService, eh: p.ErrorHandler}
}
func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/roles")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Get("/:roleID", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(120), // 120 reads per minute
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*permdomain.Role], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.ps.ListRoles(fc.UserContext(), &repositories.ListRolesRequest{
			Filter: filter,
			QueryOptions: repositories.RolesQueryOptions{
				IncludeChildren:    fc.QueryBool("includeChildren", false),
				IncludeParent:      fc.QueryBool("includeParent", false),
				IncludePermissions: fc.QueryBool("includePermissions", false),
			},
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	roleID, err := pulid.MustParse(c.Params("roleID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	role, err := h.ps.GetRoleByID(c.UserContext(), &repositories.GetRoleByIDRequest{
		RoleID: roleID,
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
		QueryOptions: repositories.RolesQueryOptions{
			IncludeChildren:    c.QueryBool("includeChildren", false),
			IncludeParent:      c.QueryBool("includeParent", false),
			IncludePermissions: c.QueryBool("includePermissions", false),
		},
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(role)
}
