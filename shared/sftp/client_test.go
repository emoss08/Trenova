package sftp_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/sftp"
	pkgsftp "github.com/pkg/sftp"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

type testServer struct {
	address string
	hostKey string
	accepts atomic.Int32
}

func startTestServer(t *testing.T) *testServer {
	t.Helper()

	_, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(private)
	require.NoError(t, err)

	config := &ssh.ServerConfig{
		PasswordCallback: func(meta ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if meta.User() == "trenova" && string(password) == "secret" {
				return &ssh.Permissions{}, nil
			}
			return nil, errors.New("denied")
		},
	}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	server := &testServer{
		address: listener.Addr().String(),
		hostKey: strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))),
	}
	handlers := pkgsftp.InMemHandler()
	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			server.accepts.Add(1)
			go serveSFTP(conn, config, handlers)
		}
	}()
	return server
}

func serveSFTP(conn net.Conn, config *ssh.ServerConfig, handlers pkgsftp.Handlers) {
	defer conn.Close()
	_, channels, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	go ssh.DiscardRequests(requests)
	for newChannel := range channels {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unsupported")
			continue
		}
		channel, channelRequests, acceptErr := newChannel.Accept()
		if acceptErr != nil {
			return
		}
		go func() {
			for request := range channelRequests {
				_ = request.Reply(request.Type == "subsystem", nil)
			}
		}()
		server := pkgsftp.NewRequestServer(channel, handlers)
		_ = server.Serve()
		_ = server.Close()
	}
}

func (s *testServer) config(t *testing.T, host string, policy httpsafe.Policy) sftp.Config {
	t.Helper()
	_, port, err := net.SplitHostPort(s.address)
	require.NoError(t, err)
	return sftp.Config{
		Host:         host,
		Port:         port,
		Username:     "trenova",
		AuthMode:     sftp.AuthModePassword,
		Password:     "secret",
		KnownHostKey: s.hostKey,
		EgressPolicy: policy,
	}
}

func TestDialRefusesPrivateHostsByDefault(t *testing.T) {
	t.Parallel()

	server := startTestServer(t)
	for _, host := range []string{"127.0.0.1", "localhost", "::ffff:127.0.0.1"} {
		_, err := sftp.Dial(t.Context(), server.config(t, host, httpsafe.Policy{}))
		require.ErrorIs(t, err, httpsafe.ErrBlockedAddress, host)
	}
	_, err := sftp.Dial(
		t.Context(),
		server.config(t, "169.254.169.254", httpsafe.Policy{AllowPrivateNetworks: true}),
	)
	require.ErrorIs(t, err, httpsafe.ErrBlockedAddress)
	require.Zero(t, server.accepts.Load())
}

func TestReadFileEnforcesTheSizeLimit(t *testing.T) {
	t.Parallel()

	server := startTestServer(t)
	cfg := server.config(t, "127.0.0.1", httpsafe.Policy{AllowPrivateNetworks: true})
	cfg.MaxFileBytes = 16

	client, err := sftp.Dial(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	require.NoError(t, client.WriteFile("/inbound/small.edi", []byte("ISA*00*~")))
	require.NoError(t, client.WriteFile("/inbound/exact.edi", []byte(strings.Repeat("x", 16))))
	require.NoError(t, client.WriteFile("/inbound/large.edi", []byte(strings.Repeat("x", 17))))

	contents, err := client.ReadFile("/inbound/small.edi")
	require.NoError(t, err)
	require.Equal(t, "ISA*00*~", contents)

	contents, err = client.ReadFile("/inbound/exact.edi")
	require.NoError(t, err)
	require.Len(t, contents, 16)

	_, err = client.ReadFile("/inbound/large.edi")
	require.ErrorIs(t, err, sftp.ErrFileTooLarge)
	var tooLarge *sftp.FileTooLargeError
	require.ErrorAs(t, err, &tooLarge)
	require.Equal(t, int64(16), tooLarge.Limit)
	require.Equal(t, "/inbound/large.edi", tooLarge.Path)
}
