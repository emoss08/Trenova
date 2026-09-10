package sftp_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/sftp"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func newHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()

	public, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	key, err := ssh.NewPublicKey(public)
	require.NoError(t, err)

	return key
}

func authorizedKey(t *testing.T, key ssh.PublicKey) string {
	t.Helper()

	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	valid := sftp.Config{
		Host:         "sftp.example.com",
		Username:     "trenova",
		AuthMode:     sftp.AuthModePassword,
		KnownHostKey: "ssh-ed25519 AAAA",
		Password:     "secret",
	}

	tests := []struct {
		name    string
		mutate  func(cfg *sftp.Config)
		wantErr string
	}{
		{name: "password auth", mutate: func(*sftp.Config) {}},
		{
			name: "key auth",
			mutate: func(cfg *sftp.Config) {
				cfg.AuthMode = "privateKey"
				cfg.Password = ""
				cfg.PrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----"
			},
		},
		{
			name:    "missing host",
			mutate:  func(cfg *sftp.Config) { cfg.Host = "" },
			wantErr: "SFTP host is required",
		},
		{
			name:    "blank host",
			mutate:  func(cfg *sftp.Config) { cfg.Host = "   " },
			wantErr: "SFTP host is required",
		},
		{
			name:    "missing username",
			mutate:  func(cfg *sftp.Config) { cfg.Username = "" },
			wantErr: "SFTP username is required",
		},
		{
			name:    "missing known host key",
			mutate:  func(cfg *sftp.Config) { cfg.KnownHostKey = "" },
			wantErr: "SFTP known host key is required",
		},
		{
			name:    "missing password secret",
			mutate:  func(cfg *sftp.Config) { cfg.Password = "" },
			wantErr: "SFTP password secret is required",
		},
		{
			name: "missing private key secret",
			mutate: func(cfg *sftp.Config) {
				cfg.AuthMode = "privateKey"
				cfg.Password = ""
			},
			wantErr: "SFTP private key secret is required",
		},
		{
			name:    "non numeric port",
			mutate:  func(cfg *sftp.Config) { cfg.Port = "2 2" },
			wantErr: "SFTP port must be numeric",
		},
		{
			name:   "numeric port",
			mutate: func(cfg *sftp.Config) { cfg.Port = "2222" },
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

func TestConfigAddress(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		"sftp.example.com:22",
		sftp.Config{Host: "sftp.example.com"}.Address(),
	)
	require.Equal(
		t,
		"sftp.example.com:2222",
		sftp.Config{Host: "sftp.example.com", Port: "2222"}.Address(),
	)
}

func TestHostKeyCallbackAcceptsTheConfiguredKey(t *testing.T) {
	t.Parallel()

	key := newHostKey(t)

	callback, err := sftp.HostKeyCallbackFor(authorizedKey(t, key))
	require.NoError(t, err)
	require.NoError(t, callback("sftp.example.com:22", &net.TCPAddr{}, key))
}

// Vendors hand out host keys in known_hosts form, where the key is preceded by
// the host it belongs to. The bare authorized_keys parse fails on that line, so
// the callback has to retry without the leading host field.
func TestHostKeyCallbackAcceptsKnownHostsFormat(t *testing.T) {
	t.Parallel()

	key := newHostKey(t)

	callback, err := sftp.HostKeyCallbackFor("sftp.example.com " + authorizedKey(t, key))
	require.NoError(t, err)
	require.NoError(t, callback("sftp.example.com:22", &net.TCPAddr{}, key))
}

func TestHostKeyCallbackRejectsADifferentKey(t *testing.T) {
	t.Parallel()

	callback, err := sftp.HostKeyCallbackFor(authorizedKey(t, newHostKey(t)))
	require.NoError(t, err)

	err = callback("sftp.example.com:22", &net.TCPAddr{}, newHostKey(t))
	require.ErrorContains(t, err, "does not match configured known host key")
}

func TestHostKeyCallbackRejectsAnUnparseableKey(t *testing.T) {
	t.Parallel()

	_, err := sftp.HostKeyCallbackFor("not-a-key")
	require.ErrorContains(t, err, "parse SFTP known host key")
}
