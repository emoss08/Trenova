package logreader

import (
	"sync"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog"
)

// LogClient represents a WebSocket client connection
type LogClient struct {
	UserID    pulid.ID
	OrgID     pulid.ID
	BuID      pulid.ID
	Logger    *zerolog.Logger
	Conn      *websocket.Conn
	isClosing bool
	mu        sync.Mutex
}

// SendLogEntry sends a log entry to the client
func (c *LogClient) SendLogEntry(entry *repositories.LogEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosing {
		return nil
	}

	if err := c.Conn.WriteJSON(entry); err != nil {
		c.isClosing = true
		if err = c.Conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
			c.Logger.Error().Err(err).Msg("failed to send close message")
		}
		if closeErr := c.Conn.Close(); closeErr != nil {
			c.Logger.Error().Err(closeErr).Msg("failed to close connection")
		}
		return err
	}

	return nil
}

// Close marks the client as closing and closes the connection
func (c *LogClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isClosing {
		c.isClosing = true
		if err := c.Conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
			c.Logger.Error().Err(err).Msg("failed to send close message")
		}
		if err := c.Conn.Close(); err != nil {
			c.Logger.Error().Err(err).Msg("failed to close connection")
		}
	}
}
