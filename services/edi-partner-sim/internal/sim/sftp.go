package sim

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPOptions struct {
	Listen      string
	Username    string
	Password    string
	RootDir     string
	IdentityDir string
	Logger      *slog.Logger
}

type SFTPServer struct {
	options       SFTPOptions
	logger        *slog.Logger
	listener      net.Listener
	signer        ssh.Signer
	hostPublicKey string

	root      string
	inbound   string
	outbound  string
	archive   string
	closeOnce sync.Once
}

// NewSFTPServer starts an SSH+SFTP server that mimics a partner mailbox: Trenova
// pushes outbound documents into outbound/ and polls inbound/ for files the partner
// drops, archiving processed files into archive/.
//
//nolint:gocritic // Options is a constructor value struct by design.
func NewSFTPServer(options SFTPOptions) (*SFTPServer, error) {
	root := options.RootDir
	if root == "" {
		created, err := os.MkdirTemp("", "edi-partner-sim-sftp-")
		if err != nil {
			return nil, fmt.Errorf("create sftp root: %w", err)
		}
		root = created
	}
	server := &SFTPServer{
		options:  options,
		logger:   options.Logger,
		root:     root,
		inbound:  filepath.Join(root, "inbound"),
		outbound: filepath.Join(root, "outbound"),
		archive:  filepath.Join(root, "archive"),
	}
	for _, dir := range []string{server.inbound, server.outbound, server.archive} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create mailbox directory: %w", err)
		}
	}

	signer, publicKey, err := loadOrCreateHostKey(options.IdentityDir)
	if err != nil {
		return nil, err
	}
	server.signer = signer
	server.hostPublicKey = publicKey

	listenConfig := &net.ListenConfig{}
	listener, err := listenConfig.Listen(context.Background(), "tcp", options.Listen)
	if err != nil {
		return nil, fmt.Errorf("listen for sftp: %w", err)
	}
	server.listener = listener
	return server, nil
}

func (s *SFTPServer) Addr() string {
	return s.listener.Addr().String()
}

// HostAuthorizedKey returns the server's host key in authorized_keys form, suitable
// for Trenova's SFTP profile knownHostKey pin.
func (s *SFTPServer) HostAuthorizedKey() string {
	return s.hostPublicKey
}

func (s *SFTPServer) InboundDir() string  { return s.inbound }
func (s *SFTPServer) OutboundDir() string { return s.outbound }
func (s *SFTPServer) ArchiveDir() string  { return s.archive }

func (s *SFTPServer) DropInbound(name string, contents []byte) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("file name is required")
	}
	cleaned := filepath.Base(name)
	target := filepath.Join(s.inbound, cleaned)
	if err := os.WriteFile(target, contents, 0o600); err != nil {
		return "", fmt.Errorf("write inbound file: %w", err)
	}
	return target, nil
}

type MailboxFile struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Contents string `json:"contents"`
}

func (s *SFTPServer) ListDir(dir string) ([]MailboxFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]MailboxFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		contents, _ := os.ReadFile(filepath.Join(dir, entry.Name()))
		files = append(files, MailboxFile{
			Name:     entry.Name(),
			Size:     info.Size(),
			Contents: string(contents),
		})
	}
	return files, nil
}

func (s *SFTPServer) Serve() error {
	config := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if conn.User() == s.options.Username &&
				string(password) == s.options.Password {
				return &ssh.Permissions{}, nil
			}
			return nil, errors.New("invalid credentials")
		},
	}
	config.AddHostKey(s.signer)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go s.handleConn(conn, config)
	}
}

func (s *SFTPServer) Close() error {
	var err error
	s.closeOnce.Do(func() {
		err = s.listener.Close()
	})
	return err
}

func (s *SFTPServer) handleConn(conn net.Conn, config *ssh.ServerConfig) {
	sshConn, channels, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		s.logger.Warn("sftp handshake failed", "error", err)
		_ = conn.Close()
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(requests)

	for newChannel := range channels {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "only session channels are supported")
			continue
		}
		channel, channelRequests, acceptErr := newChannel.Accept()
		if acceptErr != nil {
			s.logger.Warn("sftp channel accept failed", "error", acceptErr)
			continue
		}
		go s.handleSubsystem(channel, channelRequests)
	}
}

func (s *SFTPServer) handleSubsystem(channel ssh.Channel, requests <-chan *ssh.Request) {
	for req := range requests {
		ok := req.Type == "subsystem" && len(req.Payload) >= 4 &&
			string(req.Payload[4:]) == "sftp"
		if req.WantReply {
			_ = req.Reply(ok, nil)
		}
		if !ok {
			continue
		}
		server, err := sftp.NewServer(channel)
		if err != nil {
			s.logger.Warn("sftp subsystem init failed", "error", err)
			_ = channel.Close()
			return
		}
		if serveErr := server.Serve(); serveErr != nil && !errors.Is(serveErr, io.EOF) {
			s.logger.Warn("sftp session ended", "error", serveErr)
		}
		_ = server.Close()
		return
	}
}

func loadOrCreateHostKey(dir string) (ssh.Signer, string, error) {
	keyPath := ""
	if dir != "" {
		keyPath = filepath.Join(dir, "sftp-host-key.pem")
	}
	keyPEM, err := readOrGenerateHostKeyPEM(dir, keyPath)
	if err != nil {
		return nil, "", err
	}
	signer, err := ssh.ParsePrivateKey(keyPEM)
	if err != nil {
		return nil, "", fmt.Errorf("parse sftp host key: %w", err)
	}
	authorizedKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
	return signer, authorizedKey, nil
}

func readOrGenerateHostKeyPEM(dir, keyPath string) ([]byte, error) {
	if keyPath != "" {
		if data, readErr := os.ReadFile(keyPath); readErr == nil {
			return data, nil
		}
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate sftp host key: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshal sftp host key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	if keyPath == "" {
		return keyPEM, nil
	}
	if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
		return nil, fmt.Errorf("create identity directory: %w", mkErr)
	}
	if writeErr := os.WriteFile(keyPath, keyPEM, 0o600); writeErr != nil {
		return nil, fmt.Errorf("persist sftp host key: %w", writeErr)
	}
	return keyPEM, nil
}
