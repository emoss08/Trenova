package sim

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/as2"
)

const (
	maxRequestBody = 32 << 20
	sendTimeout    = 30 * time.Second
)

type Options struct {
	AS2ID           string
	RemoteAS2ID     string
	TrenovaInbound  string
	AutoAcknowledge bool
	IdentityDir     string
	SFTP            *SFTPServer
	Logger          *slog.Logger
}

type ReceivedDocument struct {
	ReceivedAt     time.Time    `json:"receivedAt"`
	MessageID      string       `json:"messageId"`
	FileName       string       `json:"fileName"`
	Signed         bool         `json:"signed"`
	Encrypted      bool         `json:"encrypted"`
	Payload        string       `json:"payload"`
	Envelope       *X12Envelope `json:"envelope,omitempty"`
	AckSent        bool         `json:"ackSent"`
	AckError       string       `json:"ackError,omitempty"`
	RejectedReason string       `json:"rejectedReason,omitempty"`
}

type SentRecord struct {
	SentAt         time.Time `json:"sentAt"`
	MessageID      string    `json:"messageId"`
	TransactionSet string    `json:"transactionSet"`
	MDNStatus      string    `json:"mdnStatus"`
	Error          string    `json:"error,omitempty"`
}

type Server struct {
	options  Options
	identity *Identity
	logger   *slog.Logger
	client   *http.Client

	sftp *SFTPServer

	mu             sync.RWMutex
	as2ID          string
	remoteAS2ID    string
	partnerCert    *x509.Certificate
	trenovaInbound string
	autoAck        bool
	received       []*ReceivedDocument
	sent           []*SentRecord

	controlNumber atomic.Int64
}

//nolint:gocritic // Options is a constructor value struct by design.
func NewServer(options Options) (*Server, error) {
	identity, err := LoadOrCreateIdentity(
		options.IdentityDir,
		strings.ToLower(options.AS2ID)+".edi-partner-sim.local",
	)
	if err != nil {
		return nil, err
	}
	server := &Server{
		options:        options,
		identity:       identity,
		logger:         options.Logger,
		client:         &http.Client{Timeout: sendTimeout},
		sftp:           options.SFTP,
		as2ID:          options.AS2ID,
		remoteAS2ID:    options.RemoteAS2ID,
		trenovaInbound: options.TrenovaInbound,
		autoAck:        options.AutoAcknowledge,
		received:       make([]*ReceivedDocument, 0),
		sent:           make([]*SentRecord, 0),
	}
	server.controlNumber.Store(time.Now().Unix() % 100000000)
	return server, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /as2", s.handleAS2)
	mux.HandleFunc("POST /as2/", s.handleAS2)
	mux.HandleFunc("GET /control/identity", s.handleIdentity)
	mux.HandleFunc("POST /control/partner", s.handleConfigurePartner)
	mux.HandleFunc("GET /control/received", s.handleReceived)
	mux.HandleFunc("GET /control/sent", s.handleSent)
	mux.HandleFunc("POST /control/send", s.handleSend)
	mux.HandleFunc("POST /control/send-tender", s.handleSendTender)
	mux.HandleFunc("POST /control/reset", s.handleReset)
	mux.HandleFunc("GET /control/sftp", s.handleSFTPInfo)
	mux.HandleFunc("POST /control/sftp/drop", s.handleSFTPDrop)
	mux.HandleFunc("GET /control/sftp/outbound", s.handleSFTPOutbound)
	mux.HandleFunc("GET /control/sftp/inbound", s.handleSFTPInbound)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

func (s *Server) handleIdentity(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.writeJSON(w, http.StatusOK, map[string]any{
		"as2Id":          s.as2ID,
		"remoteAs2Id":    s.remoteAS2ID,
		"certificatePem": s.identity.CertificatePEM,
	})
}

type configurePartnerRequest struct {
	CertificatePEM string `json:"certificatePem"`
	InboundURL     string `json:"inboundUrl"`
	AutoAck        *bool  `json:"autoAck"`
	AS2ID          string `json:"as2Id"`
	RemoteAS2ID    string `json:"remoteAs2Id"`
}

func (s *Server) handleConfigurePartner(w http.ResponseWriter, r *http.Request) {
	req := new(configurePartnerRequest)
	if err := decodeJSON(r.Body, req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(req.CertificatePEM) != "" {
		certificate, err := as2.ParseCertificate([]byte(req.CertificatePEM))
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid partner certificate: "+err.Error())
			return
		}
		s.partnerCert = certificate
	}
	if strings.TrimSpace(req.InboundURL) != "" {
		s.trenovaInbound = strings.TrimSpace(req.InboundURL)
	}
	if req.AutoAck != nil {
		s.autoAck = *req.AutoAck
	}
	if strings.TrimSpace(req.AS2ID) != "" {
		s.as2ID = strings.TrimSpace(req.AS2ID)
	}
	if strings.TrimSpace(req.RemoteAS2ID) != "" {
		s.remoteAS2ID = strings.TrimSpace(req.RemoteAS2ID)
	}
	s.logger.Info("partner configuration updated",
		"hasCertificate", s.partnerCert != nil,
		"inboundUrl", s.trenovaInbound,
		"autoAck", s.autoAck,
	)
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleAS2(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "request body could not be read")
		return
	}
	from := strings.TrimSpace(r.Header.Get("As2-From"))
	to := strings.TrimSpace(r.Header.Get("As2-To"))
	messageID := strings.Trim(strings.TrimSpace(r.Header.Get("Message-Id")), "<>")
	if from == "" || to == "" {
		s.writeError(w, http.StatusBadRequest, "AS2-From and AS2-To headers are required")
		return
	}
	s.mu.RLock()
	partnerCert := s.partnerCert
	localAS2ID := s.as2ID
	remoteAS2ID := s.remoteAS2ID
	s.mu.RUnlock()
	if !strings.EqualFold(to, localAS2ID) {
		s.writeError(w, http.StatusNotFound, "unknown AS2-To identifier "+to)
		return
	}
	_ = remoteAS2ID

	parsed, parseErr := as2.ParseMessage(
		r.Header.Get("Content-Type"),
		body,
		&as2.ParseMessageOptions{
			DecryptionCertificate: s.identity.Certificate,
			DecryptionKey:         s.identity.Key,
			PartnerCertificate:    partnerCert,
			TransferEncoding:      r.Header.Get("Content-Transfer-Encoding"),
			MICAlgorithm: micAlgorithmFromOptions(
				r.Header.Get("Disposition-Notification-Options"),
			),
		},
	)

	document := &ReceivedDocument{
		ReceivedAt: time.Now().UTC(),
		MessageID:  messageID,
	}
	mic := ""
	var mdnError string
	if parseErr != nil {
		document.RejectedReason = parseErr.Error()
		mdnError = parseErr.Error()
		s.logger.Warn("failed to process inbound AS2 message", "error", parseErr)
	} else {
		document.FileName = parsed.FileName
		document.Signed = parsed.Signed
		document.Encrypted = parsed.Encrypted
		document.Payload = string(parsed.Payload)
		mic = parsed.MIC
		if envelope, envErr := ParseX12Envelope(document.Payload); envErr == nil {
			document.Envelope = envelope
		}
		s.logger.Info("received AS2 document",
			"messageId", messageID,
			"fileName", parsed.FileName,
			"signed", parsed.Signed,
			"encrypted", parsed.Encrypted,
		)
	}

	s.mu.Lock()
	s.received = append(s.received, document)
	autoAck := s.autoAck
	s.mu.Unlock()

	mdn, err := as2.BuildMDN(&as2.BuildMDNOptions{
		From:               localAS2ID,
		To:                 from,
		OriginalMessageID:  r.Header.Get("Message-Id"),
		ReceivedContentMIC: mic,
		ErrorText:          mdnError,
		SigningCertificate: s.identity.Certificate,
		SigningKey:         s.identity.Key,
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to build MDN: "+err.Error())
		return
	}

	if parseErr == nil && autoAck && document.Envelope != nil &&
		document.Envelope.TransactionSet != "997" && document.Envelope.TransactionSet != "999" {
		go s.sendFunctionalAck(document)
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

func (s *Server) sendFunctionalAck(document *ReceivedDocument) {
	s.mu.RLock()
	senderID := s.as2ID
	receiverID := s.remoteAS2ID
	s.mu.RUnlock()
	payload := Build997(Build997Input{
		SenderID:      senderID,
		ReceiverID:    receiverID,
		ControlNumber: s.controlNumber.Add(1),
		Original:      document.Envelope,
	})
	record, err := s.sendToTrenova(payload, "997")
	s.mu.Lock()
	defer s.mu.Unlock()
	document.AckSent = err == nil
	if err != nil {
		document.AckError = err.Error()
		s.logger.Error("failed to deliver functional acknowledgment", "error", err)
		return
	}
	s.sent = append(s.sent, record)
	s.logger.Info("functional acknowledgment delivered",
		"originalControlNumber", document.Envelope.TransactionControlNumber,
		"mdnStatus", record.MDNStatus,
	)
}

type sendRequest struct {
	Payload  string `json:"payload"`
	FileName string `json:"fileName"`
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	req := new(sendRequest)
	if err := decodeJSON(r.Body, req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Payload) == "" {
		s.writeError(w, http.StatusBadRequest, "payload is required")
		return
	}
	envelope, err := ParseX12Envelope(req.Payload)
	transactionSet := ""
	if err == nil {
		transactionSet = envelope.TransactionSet
	}
	record, err := s.sendToTrenova(req.Payload, transactionSet)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	s.mu.Lock()
	s.sent = append(s.sent, record)
	s.mu.Unlock()
	s.writeJSON(w, http.StatusOK, record)
}

type sendTenderRequest struct {
	ShipmentID string `json:"shipmentId"`
}

func (s *Server) handleSendTender(w http.ResponseWriter, r *http.Request) {
	req := new(sendTenderRequest)
	if err := decodeJSON(r.Body, req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	shipmentID := strings.TrimSpace(req.ShipmentID)
	if shipmentID == "" {
		shipmentID = fmt.Sprintf("SIM%06d", s.controlNumber.Add(1))
	}
	s.mu.RLock()
	senderID := s.as2ID
	receiverID := s.remoteAS2ID
	s.mu.RUnlock()
	payload := BuildLoadTender204(BuildLoadTenderInput{
		SenderID:      senderID,
		ReceiverID:    receiverID,
		ControlNumber: s.controlNumber.Add(1),
		ShipmentID:    shipmentID,
	})
	record, err := s.sendToTrenova(payload, "204")
	if err != nil {
		s.writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	s.mu.Lock()
	s.sent = append(s.sent, record)
	s.mu.Unlock()
	s.writeJSON(w, http.StatusOK, map[string]any{
		"record":     record,
		"shipmentId": shipmentID,
		"payload":    payload,
	})
}

func (s *Server) sendToTrenova(payload, transactionSet string) (*SentRecord, error) {
	s.mu.RLock()
	inboundURL := s.trenovaInbound
	partnerCert := s.partnerCert
	senderID := s.as2ID
	receiverID := s.remoteAS2ID
	s.mu.RUnlock()
	if inboundURL == "" {
		return nil, errors.New(
			"the Trenova inbound URL is not configured; POST /control/partner first",
		)
	}
	if partnerCert == nil {
		return nil, errors.New(
			"the Trenova certificate is not configured; POST /control/partner first",
		)
	}

	built, err := as2.BuildMessage(&as2.BuildMessageOptions{
		From:    senderID,
		To:      receiverID,
		Subject: "EDI Partner Simulator Document",
		FileName: fmt.Sprintf(
			"sim-%s-%d.edi",
			strings.ToLower(transactionSet),
			time.Now().UnixNano(),
		),
		Payload:               []byte(payload),
		SigningCertificate:    s.identity.Certificate,
		SigningKey:            s.identity.Key,
		EncryptionCertificate: partnerCert,
		RequestMDN:            true,
		RequestSignedMDN:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("build AS2 message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		inboundURL,
		bytes.NewReader(built.Body),
	)
	if err != nil {
		return nil, fmt.Errorf("build inbound request: %w", err)
	}
	request.Header.Set("Content-Type", built.ContentType)
	for key, values := range built.Headers {
		for _, value := range values {
			request.Header.Set(key, value)
		}
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("deliver to Trenova: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxRequestBody))
	if err != nil {
		return nil, fmt.Errorf("read Trenova response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf(
			"trenova rejected the document with HTTP %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	record := &SentRecord{
		SentAt:         time.Now().UTC(),
		MessageID:      built.MessageID,
		TransactionSet: transactionSet,
	}
	mdn, mdnErr := as2.ParseMDN(response.Header.Get("Content-Type"), responseBody, nil)
	switch {
	case mdnErr != nil:
		record.MDNStatus = "unparseable"
		record.Error = mdnErr.Error()
	case mdn.Processed():
		record.MDNStatus = "processed"
	default:
		record.MDNStatus = mdn.Disposition
		record.Error = mdn.Text
	}
	return record, nil
}

func (s *Server) handleReceived(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.writeJSON(w, http.StatusOK, map[string]any{"received": s.received})
}

func (s *Server) handleSent(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.writeJSON(w, http.StatusOK, map[string]any{"sent": s.sent})
}

func (s *Server) handleReset(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.received = make([]*ReceivedDocument, 0)
	s.sent = make([]*SentRecord, 0)
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleSFTPInfo(w http.ResponseWriter, r *http.Request) {
	if s.sftp == nil {
		s.writeError(w, http.StatusNotFound, "SFTP is not enabled on this simulator")
		return
	}
	host, port, err := net.SplitHostPort(s.sftp.Addr())
	if err != nil {
		host = "localhost"
		port = ""
	}
	if host == "::" || host == "0.0.0.0" || host == "" {
		host = "localhost"
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"host":              host,
		"port":              port,
		"username":          s.sftp.options.Username,
		"password":          s.sftp.options.Password,
		"knownHostKey":      s.sftp.HostAuthorizedKey(),
		"inboundDirectory":  s.sftp.InboundDir(),
		"outboundDirectory": s.sftp.OutboundDir(),
		"archiveDirectory":  s.sftp.ArchiveDir(),
	})
	_ = r
}

type sftpDropRequest struct {
	FileName string `json:"fileName"`
	Payload  string `json:"payload"`
}

func (s *Server) handleSFTPDrop(w http.ResponseWriter, r *http.Request) {
	if s.sftp == nil {
		s.writeError(w, http.StatusNotFound, "SFTP is not enabled on this simulator")
		return
	}
	req := new(sftpDropRequest)
	if err := decodeJSON(r.Body, req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Payload) == "" {
		s.writeError(w, http.StatusBadRequest, "payload is required")
		return
	}
	name := strings.TrimSpace(req.FileName)
	if name == "" {
		name = fmt.Sprintf("sim-%d.edi", s.controlNumber.Add(1))
	}
	path, err := s.sftp.DropInbound(name, []byte(req.Payload))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"path": path, "fileName": name})
}

func (s *Server) handleSFTPOutbound(w http.ResponseWriter, _ *http.Request) {
	if s.sftp == nil {
		s.writeError(w, http.StatusNotFound, "SFTP is not enabled on this simulator")
		return
	}
	files, err := s.sftp.ListDir(s.sftp.OutboundDir())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (s *Server) handleSFTPInbound(w http.ResponseWriter, _ *http.Request) {
	if s.sftp == nil {
		s.writeError(w, http.StatusNotFound, "SFTP is not enabled on this simulator")
		return
	}
	inbound, err := s.sftp.ListDir(s.sftp.InboundDir())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	archive, err := s.sftp.ListDir(s.sftp.ArchiveDir())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"inbound": inbound, "archive": archive})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoded, err := sonic.Marshal(payload)
	if err != nil {
		s.logger.Warn("failed to encode response", "error", err)
		return
	}
	if _, writeErr := w.Write(encoded); writeErr != nil {
		s.logger.Warn("failed to write response", "error", writeErr)
	}
}

func decodeJSON(body io.Reader, out any) error {
	data, err := io.ReadAll(io.LimitReader(body, maxRequestBody))
	if err != nil {
		return err
	}
	return sonic.Unmarshal(data, out)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]any{"error": message})
}

func micAlgorithmFromOptions(options string) string {
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
