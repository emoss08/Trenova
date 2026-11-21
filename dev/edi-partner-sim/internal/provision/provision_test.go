package provision

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emoss08/trenova/edi-partner-sim/internal/identity"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/sim"
	"github.com/emoss08/trenova/edi-partner-sim/internal/simclient"
	"github.com/emoss08/trenova/edi-partner-sim/internal/trenova"
	"github.com/emoss08/trenova/edi-partner-sim/internal/trenovatest"
)

func copyDir(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(from, path)
		target := filepath.Join(to, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

type setup struct {
	options *Options
	def     *partner.Definition
	fake    *trenovatest.Fake
	simInfo *sim.Server
}

func newSetup(t *testing.T) *setup {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "acme")
	copyDir(t, "../play/testdata/partners/acme", dir)
	def, err := partner.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	server, err := sim.New(sim.Options{
		Partners:    []*partner.Definition{def},
		IdentityDir: t.TempDir(),
		PublicURL:   "http://sim.test",
		Logger:      slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	simServer := httptest.NewServer(server.Handler())
	t.Cleanup(simServer.Close)
	fake := trenovatest.New(t, trenovatest.Envelope{})
	client, err := trenova.New(fake.API(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = client.Login(context.Background(), "admin@trenova.app", "admin123!"); err != nil {
		t.Fatal(err)
	}
	id, err := identity.New("trenova.test")
	if err != nil {
		t.Fatal(err)
	}
	return &setup{
		options: &Options{
			Trenova:   client,
			Sim:       simclient.New(simServer.URL, nil),
			Identity:  id,
			PublicURL: "http://trenova.test",
		},
		def:     def,
		fake:    fake,
		simInfo: server,
	}
}

func TestProvisioningIsIdempotent(t *testing.T) {
	s := newSetup(t)
	first := Partner(context.Background(), s.options, s.def)
	if first.Failure != "" || first.PartnerID == "" || first.ProfileID == "" ||
		first.VersionNumber != 1 || first.Unchanged {
		t.Fatalf("first provisioning: %+v", first)
	}
	if !strings.HasPrefix(first.InboundURL, "http://trenova.test/api/v1/edi/as2/inbound/") {
		t.Fatalf("the inbound URL: %s", first.InboundURL)
	}
	info := s.simInfo.Partners()[0]
	if info.Trenova.InboundURL != first.InboundURL || !info.Trenova.HasCertificate {
		t.Fatalf("the simulator was not given Trenova's side: %+v", info.Trenova)
	}
	creates := func() int {
		return s.fake.Count("POST /edi/partners/") +
			s.fake.Count("POST /edi/communication-profiles/") +
			s.fake.Count("POST /edi/lading/projects/")
	}
	before := creates()
	second := Partner(context.Background(), s.options, s.def)
	if second.Failure != "" || !second.Unchanged || second.PartnerID != first.PartnerID ||
		second.ProfileID != first.ProfileID || second.VersionID != first.VersionID {
		t.Fatalf("second provisioning: %+v", second)
	}
	if before != 3 || creates() != before {
		t.Fatalf("a second provisioning created again: %v", s.fake.Calls("POST"))
	}
	if first.CustomerID == "" || second.CustomerID != first.CustomerID ||
		s.fake.Created("CUSTOMER") != 1 || s.fake.Created("CUSTOMER_UPDATE") != 1 {
		t.Fatalf("the partner's customer: %q then %q, %d created", first.CustomerID,
			second.CustomerID, s.fake.Created("CUSTOMER"))
	}
	linked, err := s.options.Trenova.Partner(context.Background(), first.PartnerID)
	if err != nil || linked.CustomerID != first.CustomerID {
		t.Fatalf("the partner is not linked to its customer: %v %+v", err, linked)
	}
	if saves := s.fake.Calls("PUT /edi/lading/versions/"); len(saves) != 1 {
		t.Fatalf("a second provisioning saved the unchanged files: %v", saves)
	}
}

func TestChangedFilesBecomeANewActiveVersion(t *testing.T) {
	s := newSetup(t)
	first := Partner(context.Background(), s.options, s.def)
	if first.Failure != "" {
		t.Fatalf("first: %s", first.Failure)
	}
	file := filepath.Join(s.def.Dir, "lading", "partner.lading")
	if err := os.WriteFile(file, []byte("partner acme\n// changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second := Partner(context.Background(), s.options, s.def)
	if second.Failure != "" || second.Unchanged || second.VersionNumber != 2 {
		t.Fatalf("second: %+v", second)
	}
	if replays := s.fake.Calls("POST /edi/lading/versions/"); !containsSuffix(replays,
		"/replays/") || !containsSuffix(replays, "/activate/") {
		t.Fatalf("a new version over an active one was not replayed and activated: %v", replays)
	}
}

func TestAnUncertifiableProjectNamesItsFailingTests(t *testing.T) {
	s := newSetup(t)
	s.fake.NotReady = true
	result := Partner(context.Background(), s.options, s.def)
	if !strings.Contains(result.Failure, "not ready to certify") ||
		len(result.FailingTests) != 1 || result.Readiness == nil || result.Readiness.Ready {
		t.Fatalf("an uncertifiable project: %+v", result)
	}
	if certify := s.fake.Calls("POST /edi/lading/versions/"); containsSuffix(certify,
		"/certify/") {
		t.Fatalf("certified a project whose tests fail: %v", certify)
	}
}

func containsSuffix(calls []string, suffix string) bool {
	for _, call := range calls {
		if strings.HasSuffix(call, suffix) {
			return true
		}
	}
	return false
}
