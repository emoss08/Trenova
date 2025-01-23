package auth

import (
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auth"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

type Handler struct {
	ah  *auth.Service
	eh  *validator.ErrorHandler
	cfg *config.AuthConfig
}

type HandlerParams struct {
	fx.In

	AuthService  *auth.Service
	ErrorHandler *validator.ErrorHandler
	Config       *config.Manager
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		ah:  p.AuthService,
		eh:  p.ErrorHandler,
		cfg: p.Config.Auth(),
	}
}

func (h Handler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/auth")

	api.Post("/login", h.login)
	api.Post("/logout", h.logout)
	api.Post("/check-email", h.checkEmail)
	api.Post("/validate-session", h.validateSession)
}

func (h Handler) login(c *fiber.Ctx) error {
	var req services.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := req.Validate(); err != nil {
		return h.eh.HandleError(c, err)
	}

	sess, err := h.ah.Login(c.UserContext(), c.IP(), c.Get("User-Agent"), &req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// TODO(wolfred): Possibly change this to be set elsewhere
	c.Cookie(&fiber.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    sess.SessionID,
		Expires:  time.Now().Add(time.Hour * 72),
		SameSite: h.cfg.CookieSameSite,
		HTTPOnly: h.cfg.CookieHTTPOnly,
		Secure:   h.cfg.CookieSecure,
		Domain:   h.cfg.CookieDomain,
		Path:     h.cfg.CookiePath,
	})

	// Set the session cookie in local context
	c.Locals(ctx.CTXSessionID, sess.SessionID)
	c.Locals(ctx.CTXUserID, sess.User.ID)
	c.Locals(ctx.CTXBusinessUnitID, sess.User.BusinessUnitID)
	c.Locals(ctx.CTXOrganizationID, sess.User.CurrentOrganizationID)

	return c.Status(fiber.StatusOK).JSON(sess)
}

func (h Handler) logout(c *fiber.Ctx) error {
	// Get the session ID from cookies
	sessionID := c.Cookies(h.cfg.SessionCookieName)
	if sessionID == "" {
		return c.JSON(fiber.Map{"error": "No session ID found"})
	}

	log.Info().Str("sessionID", sessionID).Msg("logging out")

	if err := h.ah.Logout(c.UserContext(), pulid.ID(sessionID), c.IP(), c.Get("User-Agent")); err != nil {
		return h.eh.HandleError(c, err)
	}

	// delete the session cookie
	c.ClearCookie(h.cfg.SessionCookieName)

	// Reset the context
	c.Locals(ctx.CTXSessionID, nil)
	c.Locals(ctx.CTXUserID, nil)
	c.Locals(ctx.CTXBusinessUnitID, nil)
	c.Locals(ctx.CTXOrganizationID, nil)

	// send a cookie to the client to indicate that the session has been deleted
	c.Cookie(&fiber.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		SameSite: h.cfg.CookieSameSite,
		HTTPOnly: h.cfg.CookieHTTPOnly,
		Secure:   h.cfg.CookieSecure,
		Domain:   h.cfg.CookieDomain,
		Path:     h.cfg.CookiePath,
	})

	return c.SendStatus(fiber.StatusNoContent)
}

func (h Handler) checkEmail(c *fiber.Ctx) error {
	var req services.CheckEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := req.Validate(); err != nil {
		return h.eh.HandleError(c, err)
	}

	resp, err := h.ah.CheckEmail(c.UserContext(), &req)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"valid": resp})
}

func (h Handler) validateSession(c *fiber.Ctx) error {
	sessionID := c.Cookies(h.cfg.SessionCookieName)

	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "No session ID found"})
	}

	valid, err := h.ah.ValidateSession(c.UserContext(), pulid.ID(sessionID), c.IP())
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"valid": valid})
}
