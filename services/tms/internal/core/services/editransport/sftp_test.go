package editransport

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sftp"
	"github.com/stretchr/testify/require"
)

func TestEndpointConfigFromProfile(t *testing.T) {
	t.Parallel()

	profile := &edi.EDICommunicationProfile{
		Config: map[string]any{
			"host":              " sftp.example.com ",
			"port":              22,
			"username":          "trenova",
			"authMode":          "password",
			"knownHostKey":      "ssh-ed25519 AAAA",
			"outboundDirectory": "/mailbox/out",
			"fileNamingPattern": "{messageId}.edi",
		},
	}
	cfg := endpointConfigFromProfile(profile, map[string]string{
		"password":   "secret ",
		"privateKey": "",
	})

	require.Equal(t, "sftp.example.com", cfg.Host)
	require.Equal(t, "22", cfg.Port)
	require.Equal(t, "trenova", cfg.Username)
	require.Equal(t, "password", cfg.AuthMode)
	require.Equal(t, "/mailbox/out", cfg.outboundDirectory)
	require.Equal(t, "{messageId}.edi", cfg.fileNamingPattern)
	require.Equal(t, "secret", cfg.Password)
	require.Empty(t, cfg.PrivateKey)
}

func TestValidateEndpointConfig(t *testing.T) {
	t.Parallel()

	valid := endpointConfig{
		Config: sftp.Config{
			Host:         "sftp.example.com",
			Username:     "trenova",
			AuthMode:     AuthModePassword,
			KnownHostKey: "ssh-ed25519 AAAA",
			Password:     "secret",
		},
	}

	tests := []struct {
		name    string
		mutate  func(cfg *endpointConfig)
		wantErr string
	}{
		{name: "valid password auth", mutate: func(*endpointConfig) {}},
		{
			name:    "missing host",
			mutate:  func(cfg *endpointConfig) { cfg.Host = "" },
			wantErr: "SFTP host is required",
		},
		{
			name:    "missing username",
			mutate:  func(cfg *endpointConfig) { cfg.Username = "" },
			wantErr: "SFTP username is required",
		},
		{
			name:    "missing known host key",
			mutate:  func(cfg *endpointConfig) { cfg.KnownHostKey = "" },
			wantErr: "SFTP known host key is required",
		},
		{
			name:    "missing password secret",
			mutate:  func(cfg *endpointConfig) { cfg.Password = "" },
			wantErr: "SFTP password secret is required",
		},
		{
			name: "missing private key secret",
			mutate: func(cfg *endpointConfig) {
				cfg.AuthMode = "privateKey"
				cfg.Password = ""
			},
			wantErr: "SFTP private key secret is required",
		},
		{
			name:    "non numeric port",
			mutate:  func(cfg *endpointConfig) { cfg.Port = "abc" },
			wantErr: "SFTP port must be numeric",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := valid
			tt.mutate(&cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestOutboundFileName(t *testing.T) {
	t.Parallel()

	message := &edi.EDIMessage{
		ID:             pulid.MustNew("edimsg_"),
		EDIPartnerID:   pulid.MustNew("edip_"),
		TransactionSet: edi.TransactionSet204,
	}

	t.Run("default pattern", func(t *testing.T) {
		t.Parallel()
		name := OutboundFileName(&edi.EDICommunicationProfile{}, message)
		require.Equal(
			t,
			message.EDIPartnerID.String()+"-204-"+message.ID.String()+".x12",
			name,
		)
	})

	t.Run("custom pattern with timestamp", func(t *testing.T) {
		t.Parallel()
		profile := &edi.EDICommunicationProfile{
			Config: map[string]any{"fileNamingPattern": "out/{transactionSet}_{timestamp}"},
		}
		name := OutboundFileName(profile, message)
		require.True(t, strings.HasPrefix(name, "out_204_"))
		require.True(t, strings.HasSuffix(name, ".x12"))
	})

	t.Run("nil profile falls back to default", func(t *testing.T) {
		t.Parallel()
		name := OutboundFileName(nil, message)
		require.Contains(t, name, message.ID.String())
	})
}

func TestVANTransportRequiresMailboxID(t *testing.T) {
	t.Parallel()

	transport := NewVANTransport()
	_, err := transport.Deliver(t.Context(), &services.EDITransportRequest{
		Profile:  &edi.EDICommunicationProfile{Config: map[string]any{}},
		FileName: "file.x12",
	})
	require.ErrorContains(t, err, "VAN mailbox ID is required")
}

func TestDispatcherRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	dispatcher := NewDispatcher(DispatcherParams{
		Transports: []services.EDITransport{NewSFTPTransport(), NewVANTransport()},
	})
	_, err := dispatcher.Deliver(
		t.Context(),
		edi.ConnectionMethodAS2,
		&services.EDITransportRequest{},
	)
	require.ErrorContains(t, err, "not supported")
}
