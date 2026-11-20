package sim

import (
	"bytes"
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/identity"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/schedule/scheduletest"
	"github.com/emoss08/trenova/shared/as2"
	"github.com/emoss08/trenova/shared/smime"
)

const (
	partnersRoot = "../../partners"
	waitFor      = 10 * time.Second
)

var startedAt = time.Date(2026, 3, 2, 15, 0, 0, 0, time.UTC)

type harness struct {
	t       *testing.T
	sim     *Server
	clock   *scheduletest.Clock
	url     string
	trenova *fakeTrenova
	events  *journal.Subscription
}

type setup struct {
	partners []string
	edit     func(*partner.Definition)
	options  func(*Options)
}

func openDefinitions(t *testing.T, names []string) []*partner.Definition {
	t.Helper()
	definitions := make([]*partner.Definition, 0, len(names))
	for _, name := range names {
		definition, err := partner.Open(partnersRoot + "/" + name)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		definitions = append(definitions, definition)
	}
	return definitions
}

func newHarness(t *testing.T, s setup) *harness {
	t.Helper()
	definitions := openDefinitions(t, s.partners)
	if s.edit != nil {
		for _, definition := range definitions {
			s.edit(definition)
		}
	}
	var handler http.Handler = http.NotFoundHandler()
	var handlerMu sync.RWMutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerMu.RLock()
		current := handler
		handlerMu.RUnlock()
		current.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	clock := scheduletest.NewClock(startedAt)
	options := Options{
		Partners:    definitions,
		IdentityDir: t.TempDir(),
		PublicURL:   server.URL,
		Capacity:    200,
		Seed:        1,
		Clock:       clock,
		Logger:      slog.New(slog.DiscardHandler),
	}
	if s.options != nil {
		s.options(&options)
	}
	sim, err := New(options)
	if err != nil {
		t.Fatalf("new simulator: %v", err)
	}
	handlerMu.Lock()
	handler = sim.Handler()
	handlerMu.Unlock()
	h := &harness{t: t, sim: sim, clock: clock, url: server.URL}
	h.events = sim.Journal().Events().Subscribe(0, "")
	ctx, cancel := context.WithCancel(context.Background())
	ran := make(chan struct{})
	go func() {
		_ = sim.Run(ctx)
		close(ran)
	}()
	t.Cleanup(func() {
		cancel()
		<-ran
		h.events.Close()
	})
	h.trenova = newFakeTrenova(t)
	for _, info := range sim.Partners() {
		if info.AS2 != nil {
			h.trenova.trust(info.AS2.ID, info.AS2.CertificatePEM)
		}
		if _, err := sim.Configure(info.Name, &TrenovaRequest{
			CertificatePEM: h.trenova.id.CertificatePEM,
			InboundURL:     h.trenova.server.URL + "/as2",
		}); err != nil {
			t.Fatalf("configure %s: %v", info.Name, err)
		}
	}
	return h
}

func (h *harness) await(kind string, match func(*journal.Message) bool) journal.Message {
	h.t.Helper()
	deadline := time.After(waitFor)
	for {
		select {
		case event, open := <-h.events.Events():
			if !open {
				h.t.Fatalf("the event stream closed while waiting for %s", kind)
			}
			if event.Type == kind && event.Message != nil &&
				(match == nil || match(event.Message)) {
				return *event.Message
			}
		case <-deadline:
			h.t.Fatalf("no %s event arrived", kind)
			return journal.Message{}
		}
	}
}

func (h *harness) awaitEach(match func(*journal.Message) bool, kinds ...string) {
	h.t.Helper()
	left := slices.Clone(kinds)
	deadline := time.After(waitFor)
	for len(left) > 0 {
		select {
		case event, open := <-h.events.Events():
			if !open {
				h.t.Fatalf("the event stream closed while waiting for %v", left)
			}
			index := slices.Index(left, event.Type)
			if index >= 0 && event.Message != nil && match(event.Message) {
				left = slices.Delete(left, index, index+1)
			}
		case <-deadline:
			h.t.Fatalf("no %v event arrived", left)
		}
	}
}

func (h *harness) send(name string, req *SendRequest) []journal.Message {
	h.t.Helper()
	messages, err := h.sim.Send(h.t.Context(), name, req)
	if err != nil {
		h.t.Fatalf("send to %s: %v", name, err)
	}
	return messages
}

func (h *harness) peerOf(name string) *peer {
	p, ok := h.sim.peer(name)
	if !ok {
		h.t.Fatalf("no partner %s", name)
	}
	return p
}

func (h *harness) message(id string) journal.Message {
	h.t.Helper()
	m, ok := h.sim.Journal().Get(id)
	if !ok {
		h.t.Fatalf("no message %s", id)
	}
	return m
}

type delivered struct {
	from    string
	payload string
}

type fakeTrenova struct {
	t      *testing.T
	id     *identity.Identity
	server *httptest.Server
	mu     sync.Mutex
	certs  map[string]*x509.Certificate
	got    chan delivered
	posts  sync.WaitGroup
}

func newFakeTrenova(t *testing.T) *fakeTrenova {
	t.Helper()
	id, err := identity.New("trenova.test")
	if err != nil {
		t.Fatalf("trenova identity: %v", err)
	}
	f := &fakeTrenova{
		t:     t,
		id:    id,
		certs: map[string]*x509.Certificate{},
		got:   make(chan delivered, 64),
	}
	f.server = httptest.NewServer(f)
	t.Cleanup(func() {
		f.server.Close()
		f.posts.Wait()
	})
	return f
}

func (f *fakeTrenova) trust(as2ID, certificatePEM string) {
	certificate, err := smime.ParseCertificate([]byte(certificatePEM))
	if err != nil {
		f.t.Fatalf("parse %s certificate: %v", as2ID, err)
	}
	f.mu.Lock()
	f.certs[strings.ToUpper(as2ID)] = certificate
	f.mu.Unlock()
}

func (f *fakeTrenova) cert(as2ID string) *x509.Certificate {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.certs[strings.ToUpper(as2ID)]
}

func (f *fakeTrenova) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	from := r.Header.Get("As2-From")
	parsed, err := as2.ParseMessage(r.Header.Get("Content-Type"), body, &as2.ParseMessageOptions{
		DecryptionCertificate: f.id.Certificate,
		DecryptionKey:         f.id.Key,
		PartnerCertificate:    f.cert(from),
		EntityHeaders:         r.Header,
	})
	if err != nil {
		f.t.Errorf("trenova could not open a message from %s: %v", from, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.got <- delivered{from: from, payload: string(parsed.Payload)}
	if r.Header.Get(as2.HeaderDispositionNotificationTo) == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	mdn, err := as2.BuildMDN(&as2.BuildMDNOptions{
		From:               r.Header.Get("As2-To"),
		To:                 from,
		OriginalMessageID:  r.Header.Get("Message-Id"),
		ReceivedContentMIC: parsed.MIC,
		SigningCertificate: f.id.Certificate,
		SigningKey:         f.id.Key,
	})
	if err != nil {
		f.t.Errorf("trenova could not build an MDN: %v", err)
		return
	}
	if url := r.Header.Get(as2.HeaderReceiptDeliveryOption); url != "" {
		f.posts.Go(func() { f.post(url, mdn) })
		w.WriteHeader(http.StatusOK)
		return
	}
	for key, values := range mdn.Headers {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}
	w.Header().Set("Content-Type", mdn.ContentType)
	_, _ = w.Write(mdn.Body)
}

func (f *fakeTrenova) post(url string, mdn *as2.BuiltMDN) {
	request, err := http.NewRequestWithContext(
		context.Background(), http.MethodPost, url, bytes.NewReader(mdn.Body))
	if err != nil {
		f.t.Errorf("async MDN request: %v", err)
		return
	}
	for key, values := range mdn.Headers {
		for _, value := range values {
			request.Header.Set(key, value)
		}
	}
	request.Header.Set("Content-Type", mdn.ContentType)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		f.t.Errorf("async MDN post: %v", err)
		return
	}
	_ = response.Body.Close()
}

func (f *fakeTrenova) next(t *testing.T) delivered {
	t.Helper()
	select {
	case d := <-f.got:
		return d
	case <-time.After(waitFor):
		t.Fatal("trenova received nothing")
		return delivered{}
	}
}

func (f *fakeTrenova) nothing(t *testing.T) {
	t.Helper()
	select {
	case d := <-f.got:
		t.Fatalf("trenova received %q too early", firstLine(d.payload))
	case <-time.After(100 * time.Millisecond):
	}
}

type as2Send struct {
	partner    string
	signed     bool
	encrypted  bool
	compressed bool
	algorithm  string
	payload    string
}

func (h *harness) post(send *as2Send) (*as2.ParsedMDN, error) {
	h.t.Helper()
	info := h.sim.info(h.peerOf(send.partner))
	certificate := h.trenova.cert(info.AS2.ID)
	options := &as2.BuildMessageOptions{
		From:                info.Trenova.AS2ID,
		To:                  info.AS2.ID,
		FileName:            "trenova.edi",
		Payload:             []byte(send.payload),
		Compress:            send.compressed,
		EncryptionAlgorithm: send.algorithm,
		RequestMDN:          true,
		RequestSignedMDN:    true,
	}
	if send.signed {
		options.SigningCertificate, options.SigningKey = h.trenova.id.Certificate, h.trenova.id.Key
	}
	if send.encrypted {
		options.EncryptionCertificate = certificate
	}
	built, err := as2.BuildMessage(options)
	if err != nil {
		h.t.Fatalf("build: %v", err)
	}
	request, err := http.NewRequestWithContext(h.t.Context(), http.MethodPost,
		h.url+"/as2/"+send.partner, bytes.NewReader(built.Body))
	if err != nil {
		h.t.Fatalf("request: %v", err)
	}
	for key, values := range built.Headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	request.Header.Set("Content-Type", built.ContentType)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		h.t.Fatalf("post: %v", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		h.t.Fatalf("the simulator answered %d: %s", response.StatusCode, body)
	}
	return as2.ParseMDN(response.Header.Get("Content-Type"), body, certificate)
}

func trenovaX12(
	def *partner.Definition,
	ta1 bool,
	control int,
	group, version string,
	bodies ...string,
) string {
	pad := func(value string) string { return fmt.Sprintf("%-15s", value) }
	request := "0"
	if ta1 {
		request = "1"
	}
	us, them := def.Partner.Trenova, def.Partner.Them
	var out strings.Builder
	fmt.Fprintf(
		&out,
		"ISA*00*          *00*          *%s*%s*%s*%s*260302*0900*^*00401*%09d*%s*T*>~\n",
		us.Qualifier,
		pad(us.ID),
		them.Qualifier,
		pad(them.ID),
		control,
		request,
	)
	fmt.Fprintf(
		&out,
		"GS*%s*%s*%s*20260302*0900*%d*X*%s~\n",
		group,
		us.Group,
		them.Group,
		control,
		version,
	)
	for index, body := range bodies {
		segments := strings.Split(body, "~")
		fmt.Fprintf(&out, "ST*%s*%04d~\n", segments[0], index+1)
		for _, segment := range segments[1:] {
			fmt.Fprintf(&out, "%s~\n", segment)
		}
		fmt.Fprintf(&out, "SE*%d*%04d~\n", len(segments)+1, index+1)
	}
	fmt.Fprintf(&out, "GE*%d*%d~\nIEA*1*%09d~\n", len(bodies), control, control)
	return out.String()
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "~")
	return line
}
