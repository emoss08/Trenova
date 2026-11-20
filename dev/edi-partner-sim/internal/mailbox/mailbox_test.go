package mailbox

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type recorder struct {
	mu        sync.Mutex
	delivered map[string]string
	pickedUp  []string
	arrived   chan struct{}
}

func newRecorder() *recorder {
	return &recorder{delivered: map[string]string{}, arrived: make(chan struct{}, 16)}
}

func (r *recorder) Delivered(user, name string, contents []byte) {
	r.mu.Lock()
	r.delivered[user+"/"+name] = string(contents)
	r.mu.Unlock()
	r.arrived <- struct{}{}
}

func (r *recorder) PickedUp(user, name string) {
	r.mu.Lock()
	r.pickedUp = append(r.pickedUp, user+"/"+name)
	r.mu.Unlock()
	r.arrived <- struct{}{}
}

func (r *recorder) wait(t *testing.T) {
	t.Helper()
	select {
	case <-r.arrived:
	case <-time.After(5 * time.Second):
		t.Fatal("the mailbox never told the watcher")
	}
}

func startMailbox(t *testing.T, watcher Watcher) *Server {
	t.Helper()
	server, err := New(Options{
		Listen:      "127.0.0.1:0",
		Password:    "secret",
		RootDir:     t.TempDir(),
		IdentityDir: t.TempDir(),
		Logger:      slog.New(slog.DiscardHandler),
		Watcher:     watcher,
	}, []Spec{
		{User: "alpha", Inbound: "/inbound", Outbound: "/outbound", Archive: "/archive"},
		{User: "beta", Inbound: "/acme/to-carrier", Outbound: "/acme/from-carrier",
			Archive: "/acme/archive"},
	})
	if err != nil {
		t.Fatalf("new mailbox: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })
	go func() { _ = server.Serve() }()
	return server
}

func dial(t *testing.T, server *Server, user, password string) (*sftp.Client, error) {
	t.Helper()
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(server.HostAuthorizedKey()))
	if err != nil {
		t.Fatalf("parse host key: %v", err)
	}
	conn, err := ssh.Dial("tcp", server.Addr(), &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	})
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = conn.Close() })
	client, err := sftp.NewClient(conn)
	if err != nil {
		t.Fatalf("sftp client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client, nil
}

func TestEachPartnerHasItsOwnMailbox(t *testing.T) {
	t.Parallel()

	watcher := newRecorder()
	server := startMailbox(t, watcher)
	if _, err := server.Drop("beta", "../../tender.edi", []byte("ISA*beta~")); err != nil {
		t.Fatalf("drop: %v", err)
	}
	alpha, err := dial(t, server, "alpha", "secret")
	if err != nil {
		t.Fatalf("dial alpha: %v", err)
	}
	beta, err := dial(t, server, "beta", "secret")
	if err != nil {
		t.Fatalf("dial beta: %v", err)
	}
	if _, err := alpha.Stat("/acme/to-carrier/tender.edi"); err == nil {
		t.Fatal("alpha can see beta's mailbox")
	}
	if _, err := alpha.Stat("/../beta/acme/to-carrier/tender.edi"); err == nil {
		t.Fatal("alpha climbed out of its mailbox")
	}
	entries, err := beta.ReadDir("/acme/to-carrier")
	if err != nil || len(entries) != 1 || entries[0].Name() != "tender.edi" {
		t.Fatalf("beta's inbound: %v %v", entries, err)
	}
	if _, err := dial(t, server, "beta", "wrong"); err == nil {
		t.Fatal("a wrong password signed in")
	}
	if _, err := dial(t, server, "gamma", "secret"); err == nil {
		t.Fatal("an unknown user signed in")
	}
}

func TestTheWatcherHearsDeliveriesAndPickups(t *testing.T) {
	t.Parallel()

	watcher := newRecorder()
	server := startMailbox(t, watcher)
	if _, err := server.Drop("alpha", "tender.edi", []byte("ISA*tender~")); err != nil {
		t.Fatalf("drop: %v", err)
	}
	client, err := dial(t, server, "alpha", "secret")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	remote, err := client.Create("/outbound/990.x12")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := remote.Write([]byte("ISA*990~")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := remote.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	watcher.wait(t)
	if err := client.PosixRename("/inbound/tender.edi", "/archive/tender.edi"); err != nil {
		t.Fatalf("archive: %v", err)
	}
	watcher.wait(t)

	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	if got := watcher.delivered["alpha/990.x12"]; got != "ISA*990~" {
		t.Fatalf("delivered %q", got)
	}
	if len(watcher.pickedUp) != 1 || watcher.pickedUp[0] != "alpha/tender.edi" {
		t.Fatalf("picked up %v", watcher.pickedUp)
	}
	box, _ := server.Box("alpha")
	if _, err := os.Stat(filepath.Join(box.Dir, "archive", "tender.edi")); err != nil {
		t.Fatalf("the archived file is not on disk: %v", err)
	}
	listing, err := server.List("alpha")
	if err != nil || len(listing.Outbound) != 1 || len(listing.Archive) != 1 ||
		len(listing.Inbound) != 0 {
		t.Fatalf("listing %+v %v", listing, err)
	}
}

func TestTheHostKeyPersists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	first, firstKey, err := loadOrCreateHostKey(dir)
	if err != nil || first == nil {
		t.Fatalf("first: %v", err)
	}
	_, secondKey, err := loadOrCreateHostKey(dir)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if firstKey != secondKey {
		t.Fatal("the host key changed across restarts")
	}
}

func TestADropNeedsAKnownUserAndName(t *testing.T) {
	t.Parallel()

	server := startMailbox(t, nil)
	if _, err := server.Drop("gamma", "x.edi", nil); !errors.Is(err, errUnknownUser) {
		t.Fatalf("unknown user: %v", err)
	}
	if _, err := server.Drop("alpha", "  ", nil); err == nil {
		t.Fatal("an empty name was dropped")
	}
}
