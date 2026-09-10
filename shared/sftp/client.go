package sftp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"path"
	"strings"
	"time"

	pkgsftp "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const dialTimeout = 30 * time.Second

type RemoteFile struct {
	Path     string
	Name     string
	Size     int64
	Contents string
}

type Client struct {
	fs  *pkgsftp.Client
	ssh *ssh.Client
}

func Dial(ctx context.Context, cfg Config) (*Client, error) {
	authMethod, err := authMethodFor(cfg)
	if err != nil {
		return nil, err
	}

	hostKeyCallback, err := HostKeyCallbackFor(cfg.KnownHostKey)
	if err != nil {
		return nil, err
	}

	address := cfg.Address()
	dialer := net.Dialer{Timeout: dialTimeout}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("connect SFTP server: %w", err)
	}

	sshConn, channels, requests, err := ssh.NewClientConn(conn, address, &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: hostKeyCallback,
		Timeout:         dialTimeout,
	})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("establish SSH connection: %w", err)
	}

	sshClient := ssh.NewClient(sshConn, channels, requests)

	fsClient, err := pkgsftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, fmt.Errorf("open SFTP session: %w", err)
	}

	return &Client{fs: fsClient, ssh: sshClient}, nil
}

func (c *Client) Close() error {
	fsErr := c.fs.Close()
	sshErr := c.ssh.Close()
	if fsErr != nil {
		return fsErr
	}

	return sshErr
}

func (c *Client) Stat(remotePath string) (fs.FileInfo, error) {
	return c.fs.Stat(remotePath)
}

func (c *Client) WriteFile(remotePath string, contents []byte) error {
	if err := c.fs.MkdirAll(path.Dir(remotePath)); err != nil {
		return fmt.Errorf("create remote directory: %w", err)
	}

	file, err := c.fs.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file: %w", err)
	}
	defer file.Close()

	if _, err = io.Copy(file, bytes.NewReader(contents)); err != nil {
		return fmt.Errorf("write remote file: %w", err)
	}

	return nil
}

func (c *Client) ReadFile(remotePath string) (string, error) {
	file, err := c.fs.Open(remotePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (c *Client) ListFiles(directory string) ([]RemoteFile, error) {
	entries, err := c.fs.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("list directory %s: %w", directory, err)
	}

	files := make([]RemoteFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		files = append(files, RemoteFile{
			Path: path.Join(directory, entry.Name()),
			Name: entry.Name(),
			Size: entry.Size(),
		})
	}

	return files, nil
}

func (c *Client) Archive(remotePath, archiveDirectory string) error {
	if err := c.fs.MkdirAll(archiveDirectory); err != nil {
		return fmt.Errorf("create archive directory: %w", err)
	}

	archivePath := path.Join(archiveDirectory, path.Base(remotePath))
	if err := c.fs.PosixRename(remotePath, archivePath); err != nil {
		if renameErr := c.fs.Rename(remotePath, archivePath); renameErr != nil {
			return fmt.Errorf("archive file: %w", renameErr)
		}
	}

	return nil
}

func authMethodFor(cfg Config) (ssh.AuthMethod, error) {
	if cfg.AuthMode == AuthModePassword {
		return ssh.Password(cfg.Password), nil
	}

	signer, err := ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("parse SFTP private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

func HostKeyCallbackFor(knownHostKey string) (ssh.HostKeyCallback, error) {
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(strings.TrimSpace(knownHostKey)))
	if err != nil {
		fields := strings.Fields(knownHostKey)
		if len(fields) >= 3 {
			publicKey, _, _, _, err = ssh.ParseAuthorizedKey(
				[]byte(strings.Join(fields[1:], " ")),
			)
		}

		if err != nil {
			return nil, fmt.Errorf("parse SFTP known host key: %w", err)
		}
	}

	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		if !bytes.Equal(key.Marshal(), publicKey.Marshal()) {
			return errors.New("SFTP host key does not match configured known host key")
		}

		return nil
	}, nil
}
