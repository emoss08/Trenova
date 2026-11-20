package sim

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/transcript"
	"github.com/emoss08/trenova/shared/as2"
	"github.com/emoss08/trenova/shared/smime"
)

const (
	transportAS2  = "as2"
	transportSFTP = "sftp"
	mdnSync       = "sync"
	mdnAsync      = "async"
	mdnNone       = "none"
	mdnPending    = "pending"
	mdnProcessed  = "processed"
	mdnUnreadable = "unparseable"
	mdnSent       = "sent"
	mdnFailed     = "failed"
)

var (
	errNoInboundURL = errors.New("the Trenova inbound URL is not configured")
	errNoTrenovaKey = errors.New("the Trenova certificate is not configured")
	errNoMailbox    = errors.New("the SFTP mailbox is not running")
)

type outgoing struct {
	purpose    journal.Purpose
	raw        []byte
	respondsTo string
	keys       []string
	seed       uint64
	variation  string
	resend     bool
}

func (p *peer) transport() string {
	if p.def.Partner.Transport.SFTP != nil {
		return transportSFTP
	}
	return transportAS2
}

func (s *Server) prepare(p *peer, o *outgoing) (*journal.Message, error) {
	j, err := describe(o.raw)
	if err != nil {
		return nil, fmt.Errorf("the X12 to send cannot be read: %w", err)
	}
	m := &journal.Message{
		Partner:    p.name(),
		Direction:  journal.ToTrenova,
		Purpose:    o.purpose,
		Transport:  p.transport(),
		Raw:        string(o.raw),
		RespondsTo: o.respondsTo,
		Seed:       o.seed,
		Variation:  o.variation,
		Resend:     o.resend,
	}
	fill(m, j)
	m.Documents = transcript.Build(j, transcript.Options{
		Variation: o.variation,
		Resend:    o.resend,
		Earlier:   s.earlierCharge(p.name()),
	})
	for _, key := range o.keys {
		if !slices.Contains(m.Keys, key) {
			m.Keys = append(m.Keys, key)
		}
	}
	m.FileName = fileNameFor(p, j, s.local(p))
	return m, nil
}

func (s *Server) enqueue(p *peer, m *journal.Message, due time.Time) journal.Message {
	m.State = journal.StateScheduled
	dueAt := due.UTC()
	m.DueAt = &dueAt
	recorded := s.journal.Add("message.scheduled", m)
	s.scheduleDelivery(p, recorded)
	return recorded
}

func (s *Server) scheduleDelivery(p *peer, m journal.Message) {
	epoch := p.epoch.Load()
	due := s.clock.Now()
	if m.DueAt != nil {
		due = *m.DueAt
	}
	s.scheduler.At(due, func(ctx context.Context) {
		if p.epoch.Load() != epoch {
			return
		}
		s.deliver(ctx, p, m)
	})
}

func (s *Server) sendNow(ctx context.Context, p *peer, m *journal.Message) journal.Message {
	m.State = journal.StateScheduled
	recorded := s.journal.Add("message.scheduled", m)
	return s.deliver(ctx, p, recorded)
}

func (s *Server) deliver(ctx context.Context, p *peer, m journal.Message) journal.Message {
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	result := delivery{}
	var err error
	switch m.Transport {
	case transportSFTP:
		err = s.dropFile(p, &m)
	default:
		result, err = s.postAS2(ctx, p, &m)
	}
	now := s.clock.Now().UTC()
	event := "message.sent"
	if err != nil || result.failed {
		event = "message.failed"
	}
	expects := p.def.Partner.Acknowledgment.Expects
	updated, ok := s.journal.Update(event, m.ID, func(stored *journal.Message) {
		stored.DeliveredAt = &now
		stored.AS2 = result.as2
		stored.MDN = result.mdn
		switch {
		case err != nil:
			stored.State, stored.Error = journal.StateFailed, err.Error()
		case result.failed:
			stored.State, stored.Error = journal.StateFailed, result.mdn.Text
		default:
			stored.State = journal.StateDelivered
			awaitAcks(stored, expects, p.def.Partner.Acknowledgment.TA1)
		}
	})
	if !ok {
		updated = m
	}
	s.settleResponse(&updated, err == nil && !result.failed)
	s.changed()
	return updated
}

func awaitAcks(m *journal.Message, expects partner.AckSet, ta1 bool) {
	if m.Purpose == journal.PurposeAck || m.Purpose == journal.PurposeTA1 {
		return
	}
	if expects != partner.AckNone && len(m.Groups) > 0 && m.Ack == nil {
		m.Ack = &journal.Ack{Set: string(expects), Status: journal.AckAwaiting}
	}
	if ta1 && m.TA1 == nil {
		m.TA1 = &journal.TA1{Status: journal.AckAwaiting}
	}
}

func (s *Server) settleResponse(m *journal.Message, delivered bool) {
	if m.RespondsTo == "" {
		return
	}
	status, suffix := journal.AckSent, ".sent"
	if !delivered {
		status, suffix = journal.AckFailed, ".failed"
	}
	at := s.clock.Now().UTC()
	switch m.Purpose {
	case journal.PurposeAck:
		s.journal.Update("ack"+suffix, m.RespondsTo, func(stored *journal.Message) {
			if stored.Ack != nil && stored.Ack.MessageID == m.ID {
				stored.Ack.Status, stored.Ack.At, stored.Ack.Error = status, &at, m.Error
			}
		})
	case journal.PurposeTA1:
		s.journal.Update("ta1"+suffix, m.RespondsTo, func(stored *journal.Message) {
			if stored.TA1 != nil && stored.TA1.MessageID == m.ID {
				stored.TA1.Status, stored.TA1.At = status, &at
			}
		})
	}
}

func (s *Server) dropFile(p *peer, m *journal.Message) error {
	if s.mailbox == nil {
		return errNoMailbox
	}
	_, err := s.mailbox.Drop(p.name(), m.FileName, []byte(m.Raw))
	return err
}

type delivery struct {
	as2    *journal.AS2
	mdn    *journal.MDN
	failed bool
}

func (s *Server) postAS2(ctx context.Context, p *peer, m *journal.Message) (delivery, error) {
	settings := p.def.Partner.Transport.AS2
	if settings == nil {
		return delivery{}, fmt.Errorf("%s does not trade over AS2", p.name())
	}
	cfg := p.trenovaSettings()
	if cfg.InboundURL == "" {
		return delivery{}, fmt.Errorf(
			"%w; POST /control/%s/trenova first",
			errNoInboundURL,
			p.name(),
		)
	}
	options := &as2.BuildMessageOptions{
		From:       settings.ID,
		To:         cfg.AS2ID,
		Subject:    p.def.Partner.Display + " EDI",
		FileName:   m.FileName,
		Payload:    []byte(m.Raw),
		Compress:   settings.Compressed,
		RequestMDN: settings.MDN != mdnNone,
	}
	if settings.Signed {
		options.SigningCertificate, options.SigningKey = p.identity.Certificate, p.identity.Key
		options.RequestSignedMDN = options.RequestMDN
	}
	if settings.Encrypted {
		if cfg.certificate == nil {
			return delivery{}, fmt.Errorf(
				"%w; POST /control/%s/trenova first", errNoTrenovaKey, p.name())
		}
		options.EncryptionCertificate = cfg.certificate
		options.EncryptionAlgorithm = smime.EncryptionAlgorithmAES256CBC
	}
	if settings.MDN == mdnAsync {
		options.AsyncMDNURL = s.AS2URL(p.name())
	}
	built, err := as2.BuildMessage(options)
	if err != nil {
		return delivery{}, fmt.Errorf("build the AS2 message: %w", err)
	}
	result := delivery{as2: &journal.AS2{
		MessageID:  built.MessageID,
		Signed:     settings.Signed,
		Encrypted:  settings.Encrypted,
		Compressed: settings.Compressed,
	}}
	response, body, err := s.post(ctx, cfg.InboundURL, built.ContentType, built.Headers, built.Body)
	if err != nil {
		return result, err
	}
	result.mdn = &journal.MDN{Mode: settings.MDN}
	switch settings.MDN {
	case mdnSync:
		s.readSyncMDN(&result, cfg, built.MessageID, response.Header.Get("Content-Type"), body)
	case mdnAsync:
		result.mdn.Status, result.mdn.ReturnURL = mdnPending, options.AsyncMDNURL
	default:
		result.mdn.Status = mdnNone
	}
	return result, nil
}

func (s *Server) readSyncMDN(
	result *delivery,
	cfg trenovaConfig,
	messageID, contentType string,
	body []byte,
) {
	at := s.clock.Now().UTC()
	result.mdn.At = &at
	mdn, err := as2.ParseMDN(contentType, body, cfg.certificate)
	switch {
	case err != nil:
		result.mdn.Status, result.mdn.Text = mdnUnreadable, err.Error()
	case !as2.MessageIDMatches(messageID, mdn.OriginalMessageID):
		result.mdn.Status = mdnUnreadable
		result.mdn.Text = "the MDN names message " + mdn.OriginalMessageID
	case mdn.Processed():
		result.mdn.Status, result.mdn.Signed = mdnProcessed, mdn.Signed
	default:
		result.mdn.Status, result.mdn.Signed = mdnFailed, mdn.Signed
		result.mdn.Text = strings.TrimSpace(mdn.Disposition + " " + mdn.Text)
		result.failed = true
	}
}

func (s *Server) post(
	ctx context.Context,
	url, contentType string,
	headers map[string][]string,
	body []byte,
) (*http.Response, []byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("build the request to %s: %w", url, err)
	}
	for key, values := range headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	request.Header.Set("Content-Type", contentType)
	response, err := s.client.Do(request)
	if err != nil {
		return nil, nil, fmt.Errorf("deliver to %s: %w", url, err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxRequestBody))
	if err != nil {
		return nil, nil, fmt.Errorf("read the response from %s: %w", url, err)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, nil, fmt.Errorf(
			"%s answered HTTP %d: %s",
			url,
			response.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}
	return response, responseBody, nil
}
