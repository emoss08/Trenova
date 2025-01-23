package search

import (
	"github.com/emoss08/trenova/internal/core/services/search"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	ss *search.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	SearchService *search.Service
	ErrorHandler  *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ss: p.SearchService, eh: p.ErrorHandler}
}

func (h Handler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/search")

	api.Get("/", h.get)
}

func (h Handler) get(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var req Request
	if err = c.QueryParser(&req); err != nil {
		return h.eh.HandleError(c, err)
	}

	params := &search.Request{
		Query:       req.Query,
		Types:       req.Types,
		Limit:       req.Limit,
		Offset:      req.Offset,
		RequesterID: reqCtx.UserID,
		OrgID:       reqCtx.OrgID,
		BuID:        reqCtx.BuID,
	}

	results, err := h.ss.Search(c.UserContext(), params)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(results)
}
