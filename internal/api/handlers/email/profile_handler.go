package email

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	EmailProfileService services.EmailProfileService
	ErrorHandler        *validator.ErrorHandler
}

type Handler struct {
	ps services.EmailProfileService
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ps: p.EmailProfileService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/email-profiles")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Get("/:profileID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:profileID", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	eo, err := paginationutils.ParseEnhancedQueryFromJSON(c, reqCtx)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	qo := new(repositories.ListEmailProfileRequest)
	if err = paginationutils.ParseAdditionalQueryParams(c, qo); err != nil {
		return h.eh.HandleError(c, err)
	}

	listOpts := repositories.BuildEmailProfileListOptions(eo, qo)

	handler := func(fc *fiber.Ctx, filter *ports.QueryOptions) (*ports.ListResult[*email.Profile], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.ps.List(fc.Context(), listOpts)
	}

	return limitoffsetpagination.HandleEnhancedPaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	profileID, err := pulid.MustParse(c.Params("profileID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	profile, err := h.ps.Get(c.Context(), repositories.GetEmailProfileByIDRequest{
		OrgID:      reqCtx.OrgID,
		BuID:       reqCtx.BuID,
		UserID:     reqCtx.UserID,
		ProfileID:  profileID,
		ExpandData: c.QueryBool("expandData", false),
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(profile)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	profile := new(email.Profile)
	profile.OrganizationID = reqCtx.OrgID
	profile.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(profile); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.ps.Create(c.Context(), profile, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(entity)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	profileID, err := pulid.MustParse(c.Params("profileID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	profile := new(email.Profile)
	profile.ID = profileID
	profile.OrganizationID = reqCtx.OrgID
	profile.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(profile); err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.ps.Update(c.Context(), profile, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}
