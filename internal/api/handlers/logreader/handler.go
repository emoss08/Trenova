package logreader

import (
	"time"

	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/logreader"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	LogReaderService *logreader.Service
	ErrorHandler     *validator.ErrorHandler
	Logger           *logger.Logger
}

type Handler struct {
	ls *logreader.Service
	eh *validator.ErrorHandler
	l  *zerolog.Logger
}

func NewHandler(p HandlerParams) *Handler {
	log := p.Logger.With().
		Str("component", "logreader").
		Str("handler", "Handler").
		Logger()

	return &Handler{
		ls: p.LogReaderService,
		eh: p.ErrorHandler,
		l:  &log,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/logs")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.getCurrentLogs},
		middleware.PerMinute(60),
	)...)

	api.Get("/files", rl.WithRateLimit(
		[]fiber.Handler{h.getLogFiles},
		middleware.PerMinute(30),
	)...)

	api.Get("/files/:filename", rl.WithRateLimit(
		[]fiber.Handler{h.getLogFileInfo},
		middleware.PerMinute(30),
	)...)

	// WebSocket endpoint
	api.Use("/stream", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			// Get request context
			reqCtx, err := ctx.WithRequestContext(c)
			if err != nil {
				return h.eh.HandleError(c, err)
			}

			// Store context data in locals for the WebSocket handler
			c.Locals("orgID", reqCtx.OrgID)
			c.Locals("buID", reqCtx.BuID)
			c.Locals("userID", reqCtx.UserID)

			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	api.Get("/stream", websocket.New(h.handleWebSocket))
}

func (h *Handler) getCurrentLogs(c *fiber.Ctx) error {
	reqCtx, err := ctx.GetRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &repositories.ListLogOptions{
		LimitOffsetQueryOptions: &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				OrgID:  reqCtx.OrgID,
				BuID:   reqCtx.BuID,
				UserID: reqCtx.UserID,
			},
			Limit:  c.QueryInt("limit", 100),
			Offset: c.QueryInt("offset", 0),
		},
	}

	// Parse optional date filters
	if startDate := c.Query("startDate"); startDate != "" {
		if t, tErr := time.Parse(time.RFC3339, startDate); tErr == nil {
			opts.StartDate = t
		}
	}

	if endDate := c.Query("endDate"); endDate != "" {
		if t, tErr := time.Parse(time.RFC3339, endDate); tErr == nil {
			opts.EndDate = t
		}
	}

	entries, err := h.ls.GetCurrentLogs(c.UserContext(), opts)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ports.Response[[]repositories.LogEntry]{
		Results: entries,
		Count:   len(entries),
		Next:    "",
		Prev:    "",
	})
}

func (h *Handler) getLogFiles(c *fiber.Ctx) error {
	files, err := h.ls.GetAvailableLogFiles()
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ports.Response[[]string]{
		Results: files,
		Count:   len(files),
		Next:    "",
		Prev:    "",
	})
}

// getLogFileInfo returns detailed information about a specific log file
func (h *Handler) getLogFileInfo(c *fiber.Ctx) error {
	_, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	filename := c.Params("filename")
	if filename == "" {
		return h.eh.HandleError(c, fiber.ErrBadRequest)
	}

	info, err := h.ls.GetLogFileInfo(filename)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(info)
}

func (h *Handler) handleWebSocket(c *websocket.Conn) {
	// Get request context from the Locals
	orgID, ok := c.Locals("orgID").(pulid.ID)
	if !ok {
		h.l.Error().Msg("organization ID not found in connection locals")
		return
	}

	buID, ok := c.Locals("buID").(pulid.ID)
	if !ok {
		h.l.Error().Msg("business unit ID not found in connection locals")
		return
	}

	userID, ok := c.Locals("userID").(pulid.ID)
	if !ok {
		h.l.Error().Msg("user ID not found in connection locals")
		return
	}

	// Create a new client connection
	client := &logreader.LogClient{
		UserID: userID,
		OrgID:  orgID,
		BuID:   buID,
		Conn:   c,
		Logger: h.l,
	}

	// Register the client with the service
	if err := h.ls.RegisterClient(client); err != nil {
		h.l.Error().Err(err).Msg("failed to register client")
		return
	}
	defer h.ls.UnregisterClient(client)

	// Create a done channel for graceful shutdown
	done := make(chan struct{})
	defer close(done)

	// Start ping/pong
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
					h.l.Warn().Err(err).Msg("ping failed")
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Keep the connection alive until there's an error
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.l.Error().Err(err).Msg("unexpected websocket close")
			}
			return
		}
	}
}
