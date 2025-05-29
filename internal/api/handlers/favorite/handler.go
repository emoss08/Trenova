package favorite

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/core/ports"
	favoriteservice "github.com/emoss08/trenova/internal/core/services/favorite"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	FavoriteService *favoriteservice.Service
	ErrorHandler    *validator.ErrorHandler
}

type Handler struct {
	fs *favoriteservice.Service
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{fs: p.FavoriteService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/favorites")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(10), // 10 reads per second
	)...)

	api.Get("/:favoriteID/", rl.WithRateLimit(
		[]fiber.Handler{h.get},
		middleware.PerMinute(60), // 60 reads per minute
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerMinute(30), // 30 creates per minute
	)...)

	api.Put("/:favoriteID/", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerMinute(30), // 30 updates per minute
	)...)

	api.Delete("/:favoriteID/", rl.WithRateLimit(
		[]fiber.Handler{h.delete},
		middleware.PerMinute(30), // 30 deletes per minute
	)...)

	api.Post("/toggle/", rl.WithRateLimit(
		[]fiber.Handler{h.toggle},
		middleware.PerMinute(60), // 60 toggles per minute
	)...)

	api.Get("/check/:pageURL/", rl.WithRateLimit(
		[]fiber.Handler{h.checkFavorite},
		middleware.PerSecond(20), // 20 checks per second
	)...)

	api.Post("/check/", rl.WithRateLimit(
		[]fiber.Handler{h.checkFavoriteByPost},
		middleware.PerSecond(20), // 20 checks per second
	)...)
}

type CreateFavoriteRequest struct {
	PageURL     string `json:"pageUrl"     validate:"required,url,max=500"`
	PageTitle   string `json:"pageTitle"   validate:"required,max=255"`
	PageSection string `json:"pageSection" validate:"max=100"`
	Icon        string `json:"icon"        validate:"max=50"`
	Description string `json:"description" validate:"max=1000"`
}

type UpdateFavoriteRequest struct {
	PageTitle   string `json:"pageTitle"   validate:"required,max=255"`
	PageSection string `json:"pageSection" validate:"max=100"`
	Icon        string `json:"icon"        validate:"max=50"`
	Description string `json:"description" validate:"max=1000"`
}

type ToggleFavoriteRequest struct {
	PageURL     string `json:"pageUrl"     validate:"required,url,max=500"`
	PageTitle   string `json:"pageTitle"   validate:"required,max=255"`
	PageSection string `json:"pageSection" validate:"max=100"`
	Icon        string `json:"icon"        validate:"max=50"`
	Description string `json:"description" validate:"max=1000"`
}

type CheckFavoriteRequest struct {
	PageURL string `json:"pageUrl" validate:"required,url,max=500"`
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	favorites, err := h.fs.List(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ports.Response[[]*pagefavorite.PageFavorite]{
		Results: favorites,
		Next:    "",
		Prev:    "",
	})
}

func (h *Handler) get(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	favoriteID, err := pulid.MustParse(c.Params("favoriteID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	fav, err := h.fs.Get(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, reqCtx.UserID, favoriteID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fav)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var req CreateFavoriteRequest
	if err = c.BodyParser(&req); err != nil {
		return h.eh.HandleError(c, err)
	}

	fav := &pagefavorite.PageFavorite{
		PageURL:   req.PageURL,
		PageTitle: req.PageTitle,
	}

	created, err := h.fs.Create(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, reqCtx.UserID, fav)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	favoriteID, err := pulid.MustParse(c.Params("favoriteID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var req UpdateFavoriteRequest
	if err = c.BodyParser(&req); err != nil {
		return h.eh.HandleError(c, err)
	}

	// Get the existing favorite to preserve the URL
	existing, err := h.fs.Get(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, reqCtx.UserID, favoriteID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	fav := &pagefavorite.PageFavorite{
		PageURL:   existing.PageURL, // Preserve the original URL
		PageTitle: req.PageTitle,
	}

	updated, err := h.fs.Update(
		c.UserContext(),
		reqCtx.OrgID,
		reqCtx.BuID,
		reqCtx.UserID,
		favoriteID,
		fav,
	)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updated)
}

func (h *Handler) delete(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	favoriteID, err := pulid.MustParse(c.Params("favoriteID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	err = h.fs.Delete(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, reqCtx.UserID, favoriteID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) toggle(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var req ToggleFavoriteRequest
	if err = c.BodyParser(&req); err != nil {
		return h.eh.HandleError(c, err)
	}

	result, err := h.fs.ToggleFavorite(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, reqCtx.UserID,
		req.PageURL, req.PageTitle)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	if result == nil {
		// Favorite was removed
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"action":   "removed",
			"favorite": nil,
		})
	}

	// Favorite was added
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"action":   "added",
		"favorite": result,
	})
}

func (h *Handler) checkFavorite(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	pageURL := c.Params("pageURL")
	if pageURL == "" {
		return h.eh.HandleError(
			c,
			fiber.NewError(fiber.StatusBadRequest, "pageURL parameter is required"),
		)
	}

	fav, err := h.fs.GetByURL(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, reqCtx.UserID, pageURL)
	if err != nil {
		// If not found, return false
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"isFavorite": false,
			"favorite":   nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"isFavorite": true,
		"favorite":   fav,
	})
}

func (h *Handler) checkFavoriteByPost(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	var req CheckFavoriteRequest
	if err = c.BodyParser(&req); err != nil {
		return h.eh.HandleError(c, err)
	}

	fav, err := h.fs.GetByURL(
		c.UserContext(),
		reqCtx.OrgID,
		reqCtx.BuID,
		reqCtx.UserID,
		req.PageURL,
	)
	if err != nil {
		// If not found, return false
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"isFavorite": false,
			"favorite":   nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"isFavorite": true,
		"favorite":   fav,
	})
}
