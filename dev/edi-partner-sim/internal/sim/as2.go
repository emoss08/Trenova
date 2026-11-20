package sim

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/shared/as2"
)

type as2Request struct {
	from        string
	to          string
	messageID   string
	contentType string
	body        []byte
	header      http.Header
}

func (s *Server) handleAS2(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody))
	if err != nil {
		writeError(w, http.StatusBadRequest, "the request body could not be read")
		return
	}
	req := &as2Request{
		from:        strings.TrimSpace(r.Header.Get("As2-From")),
		to:          strings.TrimSpace(r.Header.Get("As2-To")),
		messageID:   r.Header.Get("Message-Id"),
		contentType: r.Header.Get("Content-Type"),
		body:        body,
		header:      r.Header,
	}
	if req.from == "" || req.to == "" {
		writeError(w, http.StatusBadRequest, "AS2-From and AS2-To headers are required")
		return
	}
	p, ok := s.as2Peer(r.PathValue("partner"), req.to)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown AS2-To identifier "+req.to)
		return
	}
	if as2.IsMDN(req.contentType, req.body) {
		s.receiveMDN(p, req)
		w.WriteHeader(http.StatusOK)
		return
	}
	s.receiveAS2(w, p, req)
}

func (s *Server) as2Peer(name, to string) (*peer, bool) {
	if name != "" {
		p, ok := s.peers[name]
		if !ok || p.def.Partner.Transport.AS2 == nil ||
			!strings.EqualFold(p.def.Partner.Transport.AS2.ID, to) {
			return nil, false
		}
		return p, true
	}
	for _, candidate := range s.order {
		p := s.peers[candidate]
		if settings := p.def.Partner.Transport.AS2; settings != nil &&
			strings.EqualFold(settings.ID, to) {
			return p, true
		}
	}
	return nil, false
}

func (s *Server) receiveAS2(w http.ResponseWriter, p *peer, req *as2Request) {
	settings := p.def.Partner.Transport.AS2
	cfg := p.trenovaSettings()
	parsed, parseErr := as2.ParseMessage(req.contentType, req.body, &as2.ParseMessageOptions{
		DecryptionCertificate: p.identity.Certificate,
		DecryptionKey:         p.identity.Key,
		PartnerCertificate:    cfg.certificate,
		EntityHeaders:         req.header,
		MICAlgorithm: micAlgorithm(
			req.header.Get(as2.HeaderDispositionNotificationOptions),
		),
		RequireSignature:  settings.Signed,
		RequireEncryption: settings.Encrypted,
	})
	wantsMDN := req.header.Get(as2.HeaderDispositionNotificationTo) != ""
	returnURL := strings.TrimSpace(req.header.Get(as2.HeaderReceiptDeliveryOption))
	in := &inbound{transport: transportAS2, mdn: &journal.MDN{Mode: mdnNone, Status: mdnNone}}
	mic := ""
	if parseErr != nil {
		in.err = parseErr.Error()
		s.logger.Warn("an AS2 message could not be opened", "partner", p.name(), "error", parseErr)
	} else {
		in.raw, in.fileName, mic = parsed.Payload, parsed.FileName, parsed.MIC
		in.as2 = &journal.AS2{
			MessageID:  strings.Trim(strings.TrimSpace(req.messageID), "<>"),
			Signed:     parsed.Signed,
			Encrypted:  parsed.Encrypted,
			Compressed: parsed.Compressed,
		}
	}
	if wantsMDN {
		in.mdn = &journal.MDN{Mode: mdnSync, Status: mdnSent, Text: in.err}
		if returnURL != "" {
			in.mdn.Mode, in.mdn.Status, in.mdn.ReturnURL = mdnAsync, mdnPending, returnURL
		}
	}
	recorded := s.receive(p, in)
	if !wantsMDN {
		w.WriteHeader(http.StatusOK)
		return
	}
	signed := strings.Contains(
		strings.ToLower(req.header.Get(as2.HeaderDispositionNotificationOptions)),
		"signed-receipt-protocol",
	)
	options := &as2.BuildMDNOptions{
		From:               settings.ID,
		To:                 req.from,
		OriginalMessageID:  req.messageID,
		ReceivedContentMIC: mic,
		ErrorText:          in.err,
	}
	if signed {
		options.SigningCertificate, options.SigningKey = p.identity.Certificate, p.identity.Key
	}
	mdn, err := as2.BuildMDN(options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "the MDN could not be built: "+err.Error())
		return
	}
	if returnURL != "" {
		epoch := p.epoch.Load()
		s.scheduler.After(0, func(ctx context.Context) {
			if p.epoch.Load() == epoch {
				s.returnMDN(ctx, recorded.ID, returnURL, mdn)
			}
		})
		w.WriteHeader(http.StatusOK)
		return
	}
	for key, values := range mdn.Headers {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}
	w.Header().Set("Content-Type", mdn.ContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(mdn.Body)
}

func (s *Server) returnMDN(ctx context.Context, id, url string, mdn *as2.BuiltMDN) {
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	_, _, err := s.post(ctx, url, mdn.ContentType, mdn.Headers, mdn.Body)
	at := s.clock.Now().UTC()
	event := "mdn.sent"
	if err != nil {
		event = "mdn.failed"
	}
	s.journal.Update(event, id, func(stored *journal.Message) {
		if stored.MDN == nil {
			return
		}
		stored.MDN.At = &at
		stored.MDN.Status = mdnSent
		if err != nil {
			stored.MDN.Status, stored.MDN.Text = mdnFailed, err.Error()
		}
	})
	s.changed()
}

func (s *Server) receiveMDN(p *peer, req *as2Request) {
	cfg := p.trenovaSettings()
	mdn, err := as2.ParseMDN(req.contentType, req.body, cfg.certificate)
	if err != nil {
		s.logger.Warn("an asynchronous MDN could not be read", "partner", p.name(), "error", err)
		return
	}
	original := strings.Trim(strings.TrimSpace(mdn.OriginalMessageID), "<>")
	sent, ok := s.journal.Find(p.name(), func(c *journal.Message) bool {
		return c.AS2 != nil && c.Direction == journal.ToTrenova &&
			as2.MessageIDMatches(c.AS2.MessageID, original)
	})
	if !ok {
		s.logger.Warn("an asynchronous MDN names no message the partner sent",
			"partner", p.name(), "messageId", original)
		return
	}
	at := s.clock.Now().UTC()
	s.journal.Update("mdn.received", sent.ID, func(stored *journal.Message) {
		if stored.MDN == nil {
			stored.MDN = &journal.MDN{Mode: mdnAsync}
		}
		stored.MDN.At, stored.MDN.Signed = &at, mdn.Signed
		if mdn.Processed() {
			stored.MDN.Status = mdnProcessed
			return
		}
		stored.MDN.Status = mdnFailed
		stored.MDN.Text = strings.TrimSpace(mdn.Disposition + " " + mdn.Text)
		stored.State, stored.Error = journal.StateFailed, stored.MDN.Text
	})
	s.changed()
}

func micAlgorithm(options string) string {
	for part := range strings.SplitSeq(options, ";") {
		key, value, found := strings.Cut(part, "=")
		if !found || !strings.EqualFold(strings.TrimSpace(key), "signed-receipt-micalg") {
			continue
		}
		fields := strings.Split(value, ",")
		algorithm := strings.TrimSpace(fields[len(fields)-1])
		return strings.ReplaceAll(strings.ToLower(algorithm), "sha-", "sha")
	}
	return ""
}
