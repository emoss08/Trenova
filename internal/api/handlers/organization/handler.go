package organization

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	orgdomain "github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/organization"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	os *organization.Service
	eh *validator.ErrorHandler
}

func NewHandler(os *organization.Service, eh *validator.ErrorHandler) *Handler {
	return &Handler{os: os, eh: eh}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/organizations")

	api.Get("/select-options/", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Get("/me/", rl.WithRateLimit(
		[]fiber.Handler{h.getUserOrganizations},
		middleware.PerSecond(10), // 10 reads per second
	)...)

	api.Get("/:orgID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(30), // 30 writes per minute
	)...)

	api.Put("/:orgID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(30), // 30 writes per minute
	)...)

	api.Post("/:orgID/logo/", rl.WithRateLimit(
		[]fiber.Handler{h.uploadLogo},
		middleware.PerMinute(5), // 5 writes per minute
	)...)

	api.Delete("/:orgID/logo/", rl.WithRateLimit(
		[]fiber.Handler{h.clearLogo},
		middleware.PerMinute(5), // 5 writes per minute (Matches the upload rate limit)
	)...)
}

func (h *Handler) selectOptions(c *fiber.Ctx) error {
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
		Limit:  100,
		Offset: 0,
	}

	options, err := h.os.SelectOptions(c.UserContext(), opts)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(options)
}

// TODO(wolfred): We should deprecate this endpoint in favor of the user organizations endpoint
// This endpoint returns all organizations within a business unit
// but in theory, the user organizations endpoint facilitates the same thing
// but by joining user_organizations and filtering by user_id
// There may be instance where business units have multiple administrators
// and they don't have access to all organizations within a business unit
func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*orgdomain.Organization], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.os.List(fc.UserContext(), filter)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	orgID, err := pulid.MustParse(c.Params("orgID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	org, err := h.os.Get(c.UserContext(), repositories.GetOrgByIDOptions{
		OrgID:        orgID,
		BuID:         reqCtx.BuID,
		IncludeState: c.QueryBool("includeState", false),
		IncludeBu:    c.QueryBool("includeBu", false),
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(org)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	org := new(orgdomain.Organization)
	// Automatically set the business unit ID
	org.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(org); err != nil {
		return h.eh.HandleError(c, err)
	}

	createdOrg, err := h.os.Create(c.UserContext(), org, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(createdOrg)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	orgID, err := pulid.MustParse(c.Params("orgID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	org := new(orgdomain.Organization)
	org.ID = orgID
	org.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(org); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedOrg, err := h.os.Update(c.UserContext(), org, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedOrg)
}

func (h *Handler) uploadLogo(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	logo, err := c.FormFile("logo")
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	orgID, err := pulid.MustParse(c.Params("orgID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	org, err := h.os.SetLogo(c.UserContext(), orgID, reqCtx.BuID, reqCtx.UserID, logo)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(org)
}

func (h *Handler) clearLogo(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	orgID, err := pulid.MustParse(c.Params("orgID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	org, err := h.os.ClearLogo(c.UserContext(), orgID, reqCtx.BuID, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(org)
}

func (h *Handler) getUserOrganizations(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*orgdomain.Organization], error) {
		return h.os.GetUserOrganizations(fc.UserContext(), filter)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}
