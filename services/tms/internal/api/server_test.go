//revive:disable-next-line:var-naming
package api

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestNewServer_UsesConfiguredHTTPTimeouts(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              8081,
			Mode:              "test",
			ReadTimeout:       11 * time.Second,
			ReadHeaderTimeout: 3 * time.Second,
			WriteTimeout:      17 * time.Second,
			IdleTimeout:       29 * time.Second,
			ShutdownTimeout:   5 * time.Second,
		},
	}

	server := NewServer(Params{
		Config: cfg,
		Logger: zap.NewNop(),
		LC:     fxtest.NewLifecycle(t),
	})

	assert.Equal(t, "127.0.0.1:8081", server.httpServer.Addr)
	assert.Equal(t, cfg.Server.ReadTimeout, server.httpServer.ReadTimeout)
	assert.Equal(t, cfg.Server.ReadHeaderTimeout, server.httpServer.ReadHeaderTimeout)
	assert.Equal(t, cfg.Server.WriteTimeout, server.httpServer.WriteTimeout)
	assert.Equal(t, cfg.Server.IdleTimeout, server.httpServer.IdleTimeout)
}
