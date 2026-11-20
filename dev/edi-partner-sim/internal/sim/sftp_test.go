package sim

import (
	"io"
	"path"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func withMailbox(t *testing.T) func(*Options) {
	return func(o *Options) {
		o.SFTPListen = "127.0.0.1:0"
		o.SFTPPassword = "mailbox-secret"
		o.SFTPRoot = t.TempDir()
	}
}

func (h *harness) sftpClient(name string) *sftp.Client {
	h.t.Helper()
	info := h.sim.info(h.peerOf(name)).SFTP
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(info.KnownHostKey))
	if err != nil {
		h.t.Fatalf("host key: %v", err)
	}
	conn, err := ssh.Dial("tcp", h.sim.Mailbox().Addr(), &ssh.ClientConfig{
		User:            info.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(info.Password)},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	})
	if err != nil {
		h.t.Fatalf("dial the mailbox: %v", err)
	}
	h.t.Cleanup(func() { _ = conn.Close() })
	client, err := sftp.NewClient(conn)
	if err != nil {
		h.t.Fatalf("sftp: %v", err)
	}
	h.t.Cleanup(func() { _ = client.Close() })
	return client
}

func readRemote(t *testing.T, client *sftp.Client, name string) string {
	t.Helper()
	file, err := client.Open(name)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	defer file.Close()
	contents, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(contents)
}

func TestAnSFTPPartnerDropsBatchesAndPicksUpDeliveries(t *testing.T) {
	t.Parallel()

	h := newHarness(t, setup{partners: []string{"retailer"}, options: withMailbox(t)})
	info := h.sim.info(h.peerOf("retailer")).SFTP
	if info.InboundDirectory != "/tamberlane/to-carrier" ||
		info.ArchiveDirectory != "/tamberlane/archive" || info.Username != "retailer" {
		t.Fatalf("the mailbox: %+v", info)
	}
	sent := h.send("retailer", &SendRequest{Set: "204", Seed: 2, With: []uint64{3, 4}})
	if len(sent) != 1 || len(sent[0].Groups) != 1 || len(sent[0].Groups[0].Transactions) != 3 {
		t.Fatalf("three tenders should share one group: %+v", sent)
	}
	tenders := sent[0]
	if !regexp.MustCompile(`^TMB_204_0076543210_\d{14}_\d{9}\.edi$`).MatchString(tenders.FileName) ||
		tenders.State != journal.StateDelivered ||
		len(tenders.Keys) != 3 {
		t.Fatalf("the dropped file: %s %s %v", tenders.FileName, tenders.State, tenders.Keys)
	}

	client := h.sftpClient("retailer")
	inbound := path.Join(info.InboundDirectory, tenders.FileName)
	if got := readRemote(t, client, inbound); got != tenders.Raw {
		t.Fatal("the file in the mailbox is not what was sent")
	}
	if err := client.PosixRename(inbound, path.Join(info.ArchiveDirectory, tenders.FileName)); err != nil {
		t.Fatalf("archive: %v", err)
	}
	picked := h.await("file.picked_up", func(m *journal.Message) bool { return m.ID == tenders.ID })
	if picked.PickedUpAt == nil {
		t.Fatal("the pickup was not recorded")
	}

	response := trenovaX12(h.peerOf("retailer").def, true, 77, "GF", "004010",
		"990~B1*TRNV*"+tenders.Keys[0]+"*20260302*A")
	remote, err := client.Create(path.Join(info.OutboundDirectory, "990-77.edi"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := remote.Write([]byte(response)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := remote.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	received := h.await("message.received", nil)
	if received.Transport != transportSFTP || received.FileName != "990-77.edi" ||
		!*received.Accepted {
		t.Fatalf("the delivered 990: %s %s accepted %v %+v", received.Transport, received.FileName,
			*received.Accepted, received.Judgement)
	}
	isReceived := func(m *journal.Message) bool { return m.ID == received.ID }
	h.awaitEach(isReceived, "ta1.scheduled", "ack.scheduled")
	h.clock.Advance(5 * time.Minute)
	h.awaitEach(isReceived, "ta1.sent", "ack.sent")

	listing, err := h.sim.Mailbox().List("retailer")
	if err != nil || len(listing.Inbound) != 2 || len(listing.Archive) != 1 {
		t.Fatalf("the mailbox after the acknowledgments: %+v %v", listing, err)
	}
	var sawTA1, saw999 bool
	for _, file := range listing.Inbound {
		sawTA1 = sawTA1 || strings.Contains(file.Contents, "TA1|000000077|260302|0900|A|000")
		saw999 = saw999 || strings.Contains(file.Contents, "ST|999|0001|005010X231A1") &&
			strings.Contains(file.Contents, "IK5|A")
	}
	if !sawTA1 || !saw999 {
		t.Fatalf(
			"the inbound folder lacks the TA1 (%v) or the 999 (%v): %+v",
			sawTA1,
			saw999,
			listing.Inbound,
		)
	}
}
