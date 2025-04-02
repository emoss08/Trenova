package documenttype

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/billing"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/documenttype"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	DocumentTypeService *documenttype.Service
	ErrorHandler        *validator.ErrorHandler
}

type Handler struct {
	dts *documenttype.Service
	eh  *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{dts: p.DocumentTypeService, eh: p.ErrorHandler}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/document-types")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Get("/select-options/", rl.WithRateLimit(
		[]fiber.Handler{h.selectOptions},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(60), // 60 writes per minute
	)...)

	api.Get("/:documentTypeID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Put("/:documentTypeID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(60), // 60 writes per minute
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
		Limit:  c.QueryInt("limit", 100),
		Offset: c.QueryInt("offset", 0),
		Query:  c.Query("search"),
	}

	if err := c.QueryParser(opts); err != nil {
		return h.eh.HandleError(c, err)
	}

	options, err := h.dts.SelectOptions(c.UserContext(), opts)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ports.Response[[]*types.SelectOption]{
		Results: options,
		Count:   len(options),
		Next:    "",
		Prev:    "",
	})
}

func (h Handler) list(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*billing.DocumentType], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.dts.List(fc.UserContext(), filter)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h Handler) get(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	documentTypeID, err := pulid.MustParse(c.Params("documentTypeID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	entity, err := h.dts.Get(c.UserContext(), repositories.GetDocumentTypeByIDRequest{
		ID:     documentTypeID,
		BuID:   reqCtx.BuID,
		OrgID:  reqCtx.OrgID,
		UserID: reqCtx.UserID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(entity)
}

func (h Handler) create(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	entity := new(billing.DocumentType)
	entity.OrganizationID = reqCtx.OrgID
	entity.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(entity); err != nil {
		return h.eh.HandleError(c, err)
	}

	createEntity, err := h.dts.Create(c.UserContext(), entity, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(createEntity)
}

func (h Handler) update(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	documentTypeID, err := pulid.MustParse(c.Params("documentTypeID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	entity := new(billing.DocumentType)
	entity.ID = documentTypeID
	entity.OrganizationID = reqCtx.OrgID
	entity.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(entity); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedEntity, err := h.dts.Update(c.UserContext(), entity, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedEntity)
}
