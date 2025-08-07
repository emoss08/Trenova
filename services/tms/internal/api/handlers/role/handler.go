/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package role

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	permdomain "github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/role"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	rs *role.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	RoleService  *role.Service
	ErrorHandler *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{rs: p.RoleService, eh: p.ErrorHandler}
}
func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	roleAPI := r.Group("/roles")

	roleAPI.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.listRoles},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	roleAPI.Get("/:roleID", rl.WithRateLimit(
		[]fiber.Handler{h.getRole},
		middleware.PerMinute(120), // 120 reads per minute
	)...)
}

func (h *Handler) listRoles(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*permdomain.Role], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.rs.List(fc.UserContext(), &repositories.ListRolesRequest{
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

func (h *Handler) getRole(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	roleID, err := pulid.MustParse(c.Params("roleID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	r, err := h.rs.Get(c.UserContext(), &repositories.GetRoleByIDRequest{
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

	return c.Status(fiber.StatusOK).JSON(r)
}
