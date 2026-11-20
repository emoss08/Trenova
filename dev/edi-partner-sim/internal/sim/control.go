package sim

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/reply"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/viewer"
	"github.com/emoss08/trenova/shared/smime"
)

const (
	keepAlive    = 15 * time.Second
	defaultLimit = 200
)

func (s *Server) Handler() http.Handler {
	runs := s.runsHandler()
	top := http.NewServeMux()
	top.Handle("/control/runs", runs)
	top.Handle("/control/runs/", runs)
	top.Handle("/", s.controlHandler())
	return top
}

func (s *Server) controlHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /", viewer.Handler())
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /as2", s.handleAS2)
	mux.HandleFunc("POST /as2/{partner}", s.handleAS2)
	mux.HandleFunc("GET /control/partners", s.handlePartners)
	mux.HandleFunc("GET /control/events", s.handleEvents)
	mux.HandleFunc("POST /control/reset", s.handleReset)
	mux.HandleFunc("GET /control/{partner}", s.handlePartner)
	mux.HandleFunc("POST /control/{partner}/trenova", s.handleTrenova)
	mux.HandleFunc("POST /control/{partner}/send", s.handleSend)
	mux.HandleFunc("GET /control/{partner}/messages", s.handleMessages)
	mux.HandleFunc("GET /control/{partner}/messages/{id}", s.handleMessage)
	mux.HandleFunc("GET /control/{partner}/conversations", s.handleConversations)
	mux.HandleFunc("GET /control/{partner}/conversations/{key}", s.handleConversation)
	mux.HandleFunc("GET /control/{partner}/script", s.handleScript)
	mux.HandleFunc("POST /control/{partner}/script", s.handleAddScript)
	mux.HandleFunc("GET /control/{partner}/mailbox", s.handleMailbox)
	mux.HandleFunc("GET /control/{partner}/acks", s.handleOverrides)
	mux.HandleFunc("POST /control/{partner}/acks", s.handleAddOverrides)
	return mux
}

type PartnerInfo struct {
	Name           string        `json:"name"`
	Display        string        `json:"display"`
	Role           partner.Role  `json:"role"`
	Zone           string        `json:"zone"`
	Transport      string        `json:"transport"`
	Them           EnvelopeID    `json:"them"`
	Us             EnvelopeID    `json:"trenovaEnvelope"`
	Version        string        `json:"version"`
	AS2            *AS2Info      `json:"as2,omitempty"`
	SFTP           *SFTPInfo     `json:"sftp,omitempty"`
	Trenova        TrenovaInfo   `json:"trenova"`
	Acknowledgment AckRules      `json:"acknowledgment"`
	Answers        *AnswerRules  `json:"answers,omitempty"`
	Sends          []string      `json:"sends"`
	Receives       []string      `json:"receives"`
	Run            *runlog.Tally `json:"run,omitempty"`
}

type EnvelopeID struct {
	Qualifier string `json:"qualifier"`
	ID        string `json:"id"`
	Group     string `json:"group"`
	SCAC      string `json:"scac,omitempty"`
}

type AckRules struct {
	Sends      string   `json:"sends"`
	Expects    string   `json:"expects"`
	TA1        bool     `json:"ta1"`
	Delay      string   `json:"delay"`
	RejectWhen string   `json:"rejectWhen"`
	RejectSets []string `json:"rejectSets"`
	Never      float64  `json:"never"`
}

type AnswerRules struct {
	Policy  string `json:"policy"`
	Decline string `json:"decline,omitempty"`
	Pace    string `json:"pace"`
}

func envelopeID(identity partner.Identity) EnvelopeID {
	return EnvelopeID{
		Qualifier: identity.Qualifier,
		ID:        identity.ID,
		Group:     identity.Group,
		SCAC:      identity.SCAC,
	}
}

func ackRules(rules *partner.Acknowledgment) AckRules {
	return AckRules{
		Sends:      string(rules.Sends),
		Expects:    string(rules.Expects),
		TA1:        rules.TA1,
		Delay:      rules.Delay.String(),
		RejectWhen: string(rules.Reject.When),
		RejectSets: rules.Reject.Sets,
		Never:      rules.Never,
	}
}

type AS2Info struct {
	ID             string `json:"id"`
	URL            string `json:"url"`
	CertificatePEM string `json:"certificatePem"`
	Signed         bool   `json:"signed"`
	Encrypted      bool   `json:"encrypted"`
	Compressed     bool   `json:"compressed"`
	MDN            string `json:"mdn"`
}

type SFTPInfo struct {
	Host              string `json:"host"`
	Port              string `json:"port"`
	Username          string `json:"username"`
	Password          string `json:"password"`
	KnownHostKey      string `json:"knownHostKey"`
	InboundDirectory  string `json:"inboundDirectory"`
	OutboundDirectory string `json:"outboundDirectory"`
	ArchiveDirectory  string `json:"archiveDirectory"`
	Disk              string `json:"disk"`
	FileName          string `json:"fileName,omitempty"`
}

type TrenovaInfo struct {
	InboundURL     string `json:"inboundUrl"`
	AS2ID          string `json:"as2Id"`
	HasCertificate bool   `json:"hasCertificate"`
}

func (s *Server) Partners() []PartnerInfo {
	out := make([]PartnerInfo, 0, len(s.order))
	for _, name := range s.order {
		out = append(out, s.info(s.peers[name]))
	}
	return out
}

func (s *Server) info(p *peer) PartnerInfo {
	def := &p.def.Partner
	cfg := p.trenovaSettings()
	info := PartnerInfo{
		Name:           def.Name,
		Display:        def.Display,
		Role:           def.Role,
		Zone:           def.Zone,
		Transport:      p.transport(),
		Them:           envelopeID(def.Them),
		Us:             envelopeID(def.Trenova),
		Version:        def.Envelope.Version,
		Acknowledgment: ackRules(&def.Acknowledgment),
		Sends:          def.Sends,
		Receives:       def.Receives,
		Trenova: TrenovaInfo{
			InboundURL:     cfg.InboundURL,
			AS2ID:          cfg.AS2ID,
			HasCertificate: cfg.certificate != nil,
		},
	}
	if current, ok := s.runs.Current(); ok {
		if tally, found := current.Partners[def.Name]; found {
			info.Run = &tally
		}
	}
	if answers := def.Answers; answers != nil {
		info.Answers = &AnswerRules{
			Policy:  string(answers.Policy),
			Decline: answers.Decline,
			Pace:    answers.Pace.String(),
		}
	}
	if settings := def.Transport.AS2; settings != nil {
		info.AS2 = &AS2Info{
			ID:             settings.ID,
			URL:            s.AS2URL(def.Name),
			CertificatePEM: p.identity.CertificatePEM,
			Signed:         settings.Signed,
			Encrypted:      settings.Encrypted,
			Compressed:     settings.Compressed,
			MDN:            settings.MDN,
		}
	}
	if s.mailbox != nil {
		if box, ok := s.mailbox.Box(def.Name); ok {
			_, port, _ := net.SplitHostPort(s.mailbox.Addr())
			info.SFTP = &SFTPInfo{
				Host:              s.options.SFTPHost,
				Port:              port,
				Username:          box.User,
				Password:          s.mailbox.Password(),
				KnownHostKey:      s.mailbox.HostAuthorizedKey(),
				InboundDirectory:  box.Inbound,
				OutboundDirectory: box.Outbound,
				ArchiveDirectory:  box.Archive,
				Disk:              box.Dir,
			}
			if sftp := def.Transport.SFTP; sftp != nil {
				info.SFTP.FileName = sftp.FileName
			}
		}
	}
	return info
}

func (s *Server) handlePartners(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"partners": s.Partners()})
}

func (s *Server) handlePartner(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requirePeer(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.info(p))
}

type TrenovaRequest struct {
	CertificatePEM string `json:"certificatePem"`
	InboundURL     string `json:"inboundUrl"`
	AS2ID          string `json:"as2Id"`
}

func (s *Server) Configure(name string, req *TrenovaRequest) (PartnerInfo, error) {
	p, ok := s.peer(name)
	if !ok {
		return PartnerInfo{}, fmt.Errorf("%w: %s", ErrUnknownPartner, name)
	}
	next := p.trenovaSettings()
	if pemText := strings.TrimSpace(req.CertificatePEM); pemText != "" {
		certificate, err := smime.ParseCertificate([]byte(pemText))
		if err != nil {
			return PartnerInfo{}, fmt.Errorf("%w: the certificate: %w", ErrInvalid, err)
		}
		next.CertificatePEM, next.certificate = pemText, certificate
	}
	if inbound := strings.TrimSpace(req.InboundURL); inbound != "" {
		parsed, err := url.Parse(inbound)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.Host == "" {
			return PartnerInfo{}, fmt.Errorf("%w: inboundUrl %q is not an http(s) URL",
				ErrInvalid, inbound)
		}
		next.InboundURL = inbound
	}
	if id := strings.TrimSpace(req.AS2ID); id != "" {
		next.AS2ID = id
	}
	p.mu.Lock()
	p.trenova = next
	p.mu.Unlock()
	s.changed()
	s.logger.Info("Trenova's side of the connection is set", "partner", name,
		"inboundUrl", next.InboundURL, "hasCertificate", next.certificate != nil)
	return s.info(p), nil
}

func (s *Server) handleTrenova(w http.ResponseWriter, r *http.Request) {
	req := new(TrenovaRequest)
	if !decode(w, r, req) {
		return
	}
	info, err := s.Configure(r.PathValue("partner"), req)
	if err != nil {
		writeFailure(w, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	req := new(SendRequest)
	if !decode(w, r, req) {
		return
	}
	messages, err := s.Send(r.Context(), r.PathValue("partner"), req)
	if err != nil {
		writeFailure(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": messages})
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requirePeer(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	filter := journal.Filter{
		Direction: journal.Direction(query.Get("direction")),
		Set:       query.Get("set"),
		Key:       query.Get("key"),
		Limit:     defaultLimit,
	}
	if limit := query.Get("limit"); limit != "" {
		parsed, err := strconv.Atoi(limit)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive number")
			return
		}
		filter.Limit = parsed
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"messages": s.journal.Messages(p.name(), filter),
	})
}

func (s *Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requirePeer(w, r)
	if !ok {
		return
	}
	m, found := s.journal.Get(r.PathValue("id"))
	if !found || m.Partner != p.name() {
		writeError(w, http.StatusNotFound, "no such message")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) handleConversations(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requirePeer(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": s.conversations(p.name())})
}

type ScriptAnswer struct {
	Key        string  `json:"key,omitempty"`
	Seed       uint64  `json:"seed,omitempty"`
	Decline    string  `json:"decline,omitempty"`
	Delay      float64 `json:"delay,omitempty"`
	DeltaCents int64   `json:"deltaCents,omitempty"`
	Misfooted  bool    `json:"misfooted,omitempty"`
	Stray      bool    `json:"stray,omitempty"`
	Silent     bool    `json:"silent,omitempty"`
}

type ScriptRequest struct {
	Answers []ScriptAnswer `json:"answers"`
	Replace bool           `json:"replace,omitempty"`
}

func (s *Server) Script(name string, req *ScriptRequest) ([]ScriptAnswer, error) {
	p, ok := s.peer(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownPartner, name)
	}
	if p.carrier == nil {
		return nil, fmt.Errorf("%w: %s is not a carrier; only a carrier answers tenders",
			ErrConflict, name)
	}
	entries := make([]scriptEntry, 0, len(req.Answers))
	for index, answer := range req.Answers {
		if answer.Delay < 0 || answer.Delay > 1 {
			return nil, fmt.Errorf("%w: answers[%d].delay is a probability between 0 and 1",
				ErrInvalid, index)
		}
		entries = append(entries, scriptEntry{
			Key:    strings.TrimSpace(answer.Key),
			Silent: answer.Silent,
			Scenario: reply.Scenario{
				Seed:       answer.Seed,
				Decline:    answer.Decline,
				Delay:      answer.Delay,
				DeltaCents: answer.DeltaCents,
				Misfooted:  answer.Misfooted,
				Stray:      answer.Stray,
			},
		})
	}
	p.mu.Lock()
	if req.Replace {
		p.script = nil
	}
	p.script = append(p.script, entries...)
	queued := scriptAnswers(p.script)
	p.mu.Unlock()
	s.changed()
	return queued, nil
}

func scriptAnswers(entries []scriptEntry) []ScriptAnswer {
	out := make([]ScriptAnswer, 0, len(entries))
	for _, entry := range entries {
		out = append(out, ScriptAnswer{
			Key:        entry.Key,
			Seed:       entry.Scenario.Seed,
			Decline:    entry.Scenario.Decline,
			Delay:      entry.Scenario.Delay,
			DeltaCents: entry.Scenario.DeltaCents,
			Misfooted:  entry.Scenario.Misfooted,
			Stray:      entry.Scenario.Stray,
			Silent:     entry.Silent,
		})
	}
	return out
}

func (s *Server) handleScript(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requirePeer(w, r)
	if !ok {
		return
	}
	p.mu.Lock()
	queued := scriptAnswers(p.script)
	p.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"answers": queued})
}

func (s *Server) handleAddScript(w http.ResponseWriter, r *http.Request) {
	req := new(ScriptRequest)
	if !decode(w, r, req) {
		return
	}
	queued, err := s.Script(r.PathValue("partner"), req)
	if err != nil {
		writeFailure(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"answers": queued})
}

func (s *Server) handleMailbox(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requirePeer(w, r)
	if !ok {
		return
	}
	if s.mailbox == nil {
		writeError(w, http.StatusNotFound, "the SFTP mailbox is not running")
		return
	}
	listing, err := s.mailbox.List(p.name())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, listing)
}

type resetRequest struct {
	Partner string `json:"partner"`
}

func (s *Server) Reset(name string) error {
	if name != "" {
		if _, ok := s.peer(name); !ok {
			return fmt.Errorf("%w: %s", ErrUnknownPartner, name)
		}
	}
	for _, candidate := range s.order {
		if name != "" && candidate != name {
			continue
		}
		p := s.peers[candidate]
		p.epoch.Add(1)
		p.mu.Lock()
		p.script = nil
		p.overrides = nil
		p.mu.Unlock()
		p.originals.reset(nil)
	}
	s.journal.Reset(name)
	s.runs.Reset(name)
	s.changed()
	return nil
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	req := new(resetRequest)
	if r.ContentLength != 0 && !decode(w, r, req) {
		return
	}
	if err := s.Reset(strings.TrimSpace(req.Partner)); err != nil {
		writeFailure(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}
	partnerName := r.URL.Query().Get("partner")
	if _, known := s.peer(partnerName); partnerName != "" && !known {
		writeError(w, http.StatusNotFound, "no such partner: "+partnerName)
		return
	}
	after, _ := strconv.ParseUint(
		firstOf(r.Header.Get("Last-Event-ID"), r.URL.Query().Get("after")),
		10,
		64,
	)
	sub := s.journal.Events().Subscribe(after, partnerName)
	defer sub.Close()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, ": connected\n\n")
	flusher.Flush()
	ticker := time.NewTicker(keepAlive)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			_, _ = io.WriteString(w, ": keep-alive\n\n")
			flusher.Flush()
		case event, open := <-sub.Events():
			if !open {
				return
			}
			data, err := sonic.Marshal(event)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Type, data)
			flusher.Flush()
		}
	}
}

func firstOf(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *Server) requirePeer(w http.ResponseWriter, r *http.Request) (*peer, bool) {
	p, ok := s.peer(r.PathValue("partner"))
	if !ok {
		writeError(w, http.StatusNotFound, "no such partner: "+r.PathValue("partner"))
	}
	return p, ok
}

func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody))
	if err == nil && len(strings.TrimSpace(string(data))) > 0 {
		err = sonic.Unmarshal(data, out)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

func writeFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUnknownPartner):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	encoded, err := sonic.Marshal(payload)
	if err != nil {
		http.Error(w, "the response could not be encoded", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(encoded)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
