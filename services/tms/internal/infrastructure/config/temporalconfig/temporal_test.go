package temporalconfig

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemporalConfig_GetNamespace(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		assert.Equal(t, "default", c.GetNamespace())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &Config{Namespace: "custom-ns"}
		assert.Equal(t, "custom-ns", c.GetNamespace())
	})
}

func TestTemporalConfig_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "host port only",
			cfg:  Config{HostPort: "localhost:7233"},
		},
		{
			name: "profile without host port",
			cfg:  Config{Profile: "cloud"},
		},
		{
			name:    "neither host port nor profile",
			cfg:     Config{},
			wantErr: true,
		},
		{
			name:    "invalid host port",
			cfg:     Config{HostPort: "not a host"},
			wantErr: true,
		},
		{
			name: "client certificate without key",
			cfg: Config{
				HostPort: "localhost:7233",
				TLS:      TLSConfig{ClientCertPath: "client.pem"},
			},
			wantErr: true,
		},
		{
			name: "client certificate with key",
			cfg: Config{
				HostPort: "localhost:7233",
				TLS: TLSConfig{
					ClientCertPath: "client.pem",
					ClientKeyPath:  "client.key",
				},
			},
		},
		{
			name: "encryption on with a key id",
			cfg: Config{
				HostPort: "localhost:7233",
				Security: SecurityConfig{EnableEncryption: true, EncryptionKeyID: "primary"},
			},
		},
		{
			name: "encryption on without a key id",
			cfg: Config{
				HostPort: "localhost:7233",
				Security: SecurityConfig{EnableEncryption: true},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validator.New().Struct(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestTemporalConfig_UsesProfile(t *testing.T) {
	t.Parallel()

	assert.False(t, (&Config{}).UsesProfile())
	assert.False(t, (&Config{Profile: "   "}).UsesProfile())
	assert.True(t, (&Config{Profile: "cloud"}).UsesProfile())
}

func TestTemporalTLSConfig_IsEnabled(t *testing.T) {
	t.Parallel()

	assert.False(t, (&TLSConfig{}).IsEnabled())
	assert.True(t, (&TLSConfig{Enabled: true}).IsEnabled())
	assert.True(t, (&TLSConfig{ServerName: "temporal.example.com"}).IsEnabled())
	assert.True(t, (&TLSConfig{ServerCACertPath: "ca.pem"}).IsEnabled())
	assert.True(
		t,
		(&TLSConfig{ClientCertPath: "c.pem", ClientKeyPath: "c.key"}).IsEnabled(),
	)
}

func TestTemporalConfig_GetIdentity(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		assert.Equal(t, "trenova-tms", c.GetIdentity())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &Config{Identity: "my-service"}
		assert.Equal(t, "my-service", c.GetIdentity())
	})
}

func TestTemporalInterceptorConfig_GetLogLevel(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &InterceptorConfig{}
		assert.Equal(t, "info", c.GetLogLevel())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &InterceptorConfig{LogLevel: "debug"}
		assert.Equal(t, "debug", c.GetLogLevel())
	})
}
