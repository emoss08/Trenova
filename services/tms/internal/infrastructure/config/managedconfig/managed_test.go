package managedconfig_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	token := strings.Repeat("a", 32)
	tests := []struct {
		name string
		cfg  managedconfig.Config
		err  error
	}{
		{name: "off needs nothing", cfg: managedconfig.Config{}},
		{name: "on with a token", cfg: managedconfig.Config{Enabled: true, Token: token}},
		{
			name: "on without a token",
			cfg:  managedconfig.Config{Enabled: true},
			err:  managedconfig.ErrTokenRequired,
		},
		{
			name: "on with a short token",
			cfg:  managedconfig.Config{Enabled: true, Token: "short"},
			err:  managedconfig.ErrTokenRequired,
		},
		{
			name: "on with a bad port",
			cfg:  managedconfig.Config{Enabled: true, Token: token, Port: 70000},
			err:  managedconfig.ErrPortInvalid,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, tt.cfg.Validate(), tt.err)
		})
	}
}

func TestDefaults(t *testing.T) {
	t.Parallel()

	var c managedconfig.Config
	require.Equal(t, "127.0.0.1", c.GetHost())
	require.Equal(t, 8090, c.GetPort())
}
