package sim

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func TestSFTPServerRoundTrip(t *testing.T) {
	t.Parallel()

	server, err := NewSFTPServer(SFTPOptions{
		Listen:   "127.0.0.1:0",
		Username: "trenova",
		Password: "secret",
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("new sftp server: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })
	go func() { _ = server.Serve() }()

	// The partner drops an inbound file; Trenova (this client) should read it.
	if _, err := server.DropInbound("tender.edi", []byte("ISA*...~")); err != nil {
		t.Fatalf("drop inbound: %v", err)
	}

	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(server.HostAuthorizedKey()))
	if err != nil {
		t.Fatalf("parse host key: %v", err)
	}
	clientConfig := &ssh.ClientConfig{
		User:            "trenova",
		Auth:            []ssh.AuthMethod{ssh.Password("secret")},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	}
	conn, err := ssh.Dial("tcp", server.Addr(), clientConfig)
	if err != nil {
		t.Fatalf("dial sftp: %v", err)
	}
	defer conn.Close()
	client, err := sftp.NewClient(conn)
	if err != nil {
		t.Fatalf("sftp client: %v", err)
	}
	defer client.Close()

	// Read the dropped inbound file.
	entries, err := client.ReadDir(server.InboundDir())
	if err != nil {
		t.Fatalf("read inbound dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "tender.edi" {
		t.Fatalf("unexpected inbound listing: %+v", entries)
	}

	// Push an outbound file the way the SFTP transport does.
	outboundPath := filepath.Join(server.OutboundDir(), "outbound-204.x12")
	remote, err := client.Create(outboundPath)
	if err != nil {
		t.Fatalf("create outbound file: %v", err)
	}
	if _, err := remote.Write([]byte("ISA*...~ST*204~")); err != nil {
		t.Fatalf("write outbound file: %v", err)
	}
	_ = remote.Close()

	written, err := os.ReadFile(outboundPath)
	if err != nil {
		t.Fatalf("read written outbound file: %v", err)
	}
	if string(written) != "ISA*...~ST*204~" {
		t.Fatalf("unexpected outbound contents: %q", written)
	}
}

func TestLoadOrCreateIdentityPersists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	first, err := LoadOrCreateIdentity(dir, "partner.test")
	if err != nil {
		t.Fatalf("first identity: %v", err)
	}
	second, err := LoadOrCreateIdentity(dir, "partner.test")
	if err != nil {
		t.Fatalf("second identity: %v", err)
	}
	if first.CertificatePEM != second.CertificatePEM {
		t.Fatal("certificate was not reused across LoadOrCreateIdentity calls")
	}
	if _, err := os.Stat(filepath.Join(dir, "as2-cert.pem")); err != nil {
		t.Fatalf("certificate was not persisted: %v", err)
	}
}
