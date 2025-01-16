package search

import (
	"github.com/gofiber/fiber/v2"
	"github.com/trenova-app/transport/internal/core/services/search"
	"github.com/trenova-app/transport/internal/pkg/ctx"
	"github.com/trenova-app/transport/internal/pkg/validator"
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
