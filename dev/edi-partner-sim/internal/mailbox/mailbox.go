package mailbox

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	userExtension = "mailbox-user"
	hostKeyFile   = "sftp-host-key.pem"
	maxListed     = 500
)

var (
	errUnknownUser = errors.New("mailbox: no such user")
	errBadPath     = errors.New("mailbox: a mailbox directory must be an absolute path below /")
)

type Watcher interface {
	Delivered(user, name string, contents []byte)
	PickedUp(user, name string)
}

type Spec struct {
	User     string
	Inbound  string
	Outbound string
	Archive  string
}

type Options struct {
	Listen      string
	Password    string
	RootDir     string
	IdentityDir string
	Logger      *slog.Logger
	Watcher     Watcher
}

type Box struct {
	User     string
	Inbound  string
	Outbound string
	Archive  string
	Dir      string
	root     *os.Root
}

type Server struct {
	options       Options
	logger        *slog.Logger
	listener      net.Listener
	signer        ssh.Signer
	hostPublicKey string
	boxes         map[string]*Box
	root          string
	closeOnce     sync.Once
}

func New(options Options, specs []Spec) (*Server, error) {
	root := options.RootDir
	if root == "" {
		created, err := os.MkdirTemp("", "edi-sim-sftp-")
		if err != nil {
			return nil, fmt.Errorf("create the sftp root: %w", err)
		}
		root = created
	}
	server := &Server{
		options: options,
		logger:  options.Logger,
		boxes:   make(map[string]*Box, len(specs)),
		root:    root,
	}
	for _, spec := range specs {
		box, err := openBox(root, spec)
		if err != nil {
			server.closeBoxes()
			return nil, err
		}
		server.boxes[spec.User] = box
	}
	signer, publicKey, err := loadOrCreateHostKey(options.IdentityDir)
	if err != nil {
		server.closeBoxes()
		return nil, err
	}
	server.signer = signer
	server.hostPublicKey = publicKey
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", options.Listen)
	if err != nil {
		server.closeBoxes()
		return nil, fmt.Errorf("listen for sftp: %w", err)
	}
	server.listener = listener
	return server, nil
}

func openBox(root string, spec Spec) (*Box, error) {
	box := &Box{
		User:     spec.User,
		Inbound:  path.Clean(spec.Inbound),
		Outbound: path.Clean(spec.Outbound),
		Archive:  path.Clean(spec.Archive),
		Dir:      filepath.Join(root, spec.User),
	}
	for _, dir := range []string{box.Inbound, box.Outbound, box.Archive} {
		if !strings.HasPrefix(dir, "/") || dir == "/" {
			return nil, fmt.Errorf("%w: %s has %q", errBadPath, spec.User, dir)
		}
		if err := os.MkdirAll(filepath.Join(box.Dir, filepath.FromSlash(dir)), 0o755); err != nil {
			return nil, fmt.Errorf("create the %s mailbox: %w", spec.User, err)
		}
	}
	opened, err := os.OpenRoot(box.Dir)
	if err != nil {
		return nil, fmt.Errorf("open the %s mailbox: %w", spec.User, err)
	}
	box.root = opened
	return box, nil
}

func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

func (s *Server) Root() string {
	return s.root
}

func (s *Server) HostAuthorizedKey() string {
	return s.hostPublicKey
}

func (s *Server) Password() string {
	return s.options.Password
}

func (s *Server) Box(user string) (*Box, bool) {
	box, ok := s.boxes[user]
	return box, ok
}

func (s *Server) Drop(user, name string, contents []byte) (string, error) {
	box, ok := s.boxes[user]
	if !ok {
		return "", fmt.Errorf("%w: %s", errUnknownUser, user)
	}
	base := path.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == "/" {
		return "", errors.New("mailbox: a file name is required")
	}
	target := path.Join(box.Inbound, base)
	if err := box.root.WriteFile(relative(target), contents, 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", target, err)
	}
	return target, nil
}

type File struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Contents string `json:"contents"`
}

type Listing struct {
	Inbound  []File `json:"inbound"`
	Outbound []File `json:"outbound"`
	Archive  []File `json:"archive"`
}

func (s *Server) List(user string) (Listing, error) {
	box, ok := s.boxes[user]
	if !ok {
		return Listing{}, fmt.Errorf("%w: %s", errUnknownUser, user)
	}
	var listing Listing
	var err error
	if listing.Inbound, err = box.list(box.Inbound); err != nil {
		return Listing{}, err
	}
	if listing.Outbound, err = box.list(box.Outbound); err != nil {
		return Listing{}, err
	}
	if listing.Archive, err = box.list(box.Archive); err != nil {
		return Listing{}, err
	}
	return listing, nil
}

func (b *Box) list(dir string) ([]File, error) {
	entries, err := fs.ReadDir(b.root.FS(), relative(dir))
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", dir, err)
	}
	files := make([]File, 0, min(len(entries), maxListed))
	for _, entry := range entries {
		if entry.IsDir() || len(files) == maxListed {
			continue
		}
		contents, readErr := b.root.ReadFile(relative(path.Join(dir, entry.Name())))
		if readErr != nil {
			continue
		}
		files = append(files, File{
			Name:     entry.Name(),
			Size:     int64(len(contents)),
			Contents: string(contents),
		})
	}
	return files, nil
}

func (s *Server) Serve() error {
	config := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if _, ok := s.boxes[conn.User()]; ok && string(password) == s.options.Password {
				return &ssh.Permissions{
					Extensions: map[string]string{userExtension: conn.User()},
				}, nil
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

func (s *Server) Close() error {
	var err error
	s.closeOnce.Do(func() {
		err = s.listener.Close()
		s.closeBoxes()
	})
	return err
}

func (s *Server) closeBoxes() {
	for _, box := range s.boxes {
		if box.root != nil {
			_ = box.root.Close()
		}
	}
}

func (s *Server) handleConn(conn net.Conn, config *ssh.ServerConfig) {
	sshConn, channels, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		s.logger.Debug("sftp handshake did not complete", "error", err)
		_ = conn.Close()
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(requests)
	box := s.boxes[sshConn.Permissions.Extensions[userExtension]]
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
		go s.handleSubsystem(box, channel, channelRequests)
	}
}

func (s *Server) handleSubsystem(box *Box, channel ssh.Channel, requests <-chan *ssh.Request) {
	for req := range requests {
		ok := req.Type == "subsystem" && len(req.Payload) >= 4 &&
			string(req.Payload[4:]) == "sftp"
		if req.WantReply {
			_ = req.Reply(ok, nil)
		}
		if !ok {
			continue
		}
		h := &handler{box: box, watcher: s.options.Watcher, logger: s.logger}
		server := sftp.NewRequestServer(channel, sftp.Handlers{
			FileGet:  h,
			FilePut:  h,
			FileCmd:  h,
			FileList: h,
		})
		if serveErr := server.Serve(); serveErr != nil && !errors.Is(serveErr, io.EOF) {
			s.logger.Warn("sftp session ended", "user", box.User, "error", serveErr)
		}
		_ = server.Close()
		return
	}
}

func relative(p string) string {
	cleaned := strings.TrimPrefix(path.Clean("/"+p), "/")
	if cleaned == "" {
		return "."
	}
	return cleaned
}

func loadOrCreateHostKey(dir string) (ssh.Signer, string, error) {
	keyPath := ""
	if dir != "" {
		keyPath = filepath.Join(dir, hostKeyFile)
	}
	keyPEM, err := readOrGenerateHostKeyPEM(dir, keyPath)
	if err != nil {
		return nil, "", err
	}
	signer, err := ssh.ParsePrivateKey(keyPEM)
	if err != nil {
		return nil, "", fmt.Errorf("parse the sftp host key: %w", err)
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
		return nil, fmt.Errorf("generate the sftp host key: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshal the sftp host key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	if keyPath == "" {
		return keyPEM, nil
	}
	if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
		return nil, fmt.Errorf("create the identity directory: %w", mkErr)
	}
	if writeErr := os.WriteFile(keyPath, keyPEM, 0o600); writeErr != nil {
		return nil, fmt.Errorf("persist the sftp host key: %w", writeErr)
	}
	return keyPEM, nil
}
