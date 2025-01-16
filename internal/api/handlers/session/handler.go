package session

import (
	"github.com/gofiber/fiber/v2"
	"github.com/trenova-app/transport/internal/core/services/session"
	"github.com/trenova-app/transport/internal/pkg/ctx"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"github.com/trenova-app/transport/pkg/types/pulid"
)

type Handler struct {
	ss *session.Service
	eh *validator.ErrorHandler
}

func NewHandler(ss *session.Service, eh *validator.ErrorHandler) *Handler {
	return &Handler{ss: ss, eh: eh}
}

func (h Handler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/sessions")

	// No rate limiting
	api.Get("/me", h.get)
	api.Delete("/:sessID", h.revoke)
}

func (h Handler) get(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	sessions, err := h.ss.GetSessions(c.UserContext(), reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(sessions)
}

func (h Handler) revoke(c *fiber.Ctx) error {
	_, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	sessionID, err := pulid.MustParse(c.Params("sessID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	err = h.ss.RevokeSession(c.UserContext(), sessionID, c.IP(), c.Get("User-Agent"), "User Requested")
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "session revoked"})
}
