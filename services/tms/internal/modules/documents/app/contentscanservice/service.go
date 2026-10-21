package contentscanservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"sync"

	"github.com/emoss08/trenova/internal/infrastructure/clamav"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Config    *config.Config
	Audit     auditports.AuditService
	Logger    *zap.Logger
	Lifecycle fx.Lifecycle `optional:"true"`
}

type Service struct {
	scanner  contentscanports.Scanner
	failOpen bool
	audit    auditports.AuditService
	l        *zap.Logger
}

var _ contentscanports.Gate = (*Service)(nil)

func New(p Params) contentscanports.Gate {
	cfg := p.Config.GetContentScanConfig()
	var scanner contentscanports.Scanner
	if cfg.Enabled() {
		scanner = clamav.New(cfg.ClamAV)
	}
	svc := NewWithScanner(scanner, cfg.FailOpen, p.Audit, p.Logger)
	if p.Lifecycle != nil {
		p.Lifecycle.Append(fx.Hook{OnStart: svc.announce})
	}
	return svc
}

func NewWithScanner(
	scanner contentscanports.Scanner,
	failOpen bool,
	auditService auditports.AuditService,
	logger *zap.Logger,
) *Service {
	return &Service{
		scanner:  scanner,
		failOpen: failOpen,
		audit:    auditService,
		l:        logger.Named("service.content-scan"),
	}
}

func (s *Service) announce(ctx context.Context) error {
	if s.scanner == nil {
		s.l.Warn("malware scanning is off: no contentScan.clamav.address is configured, " +
			"so uploads, inbound attachments, capture pages and inbound EDI files are " +
			"parsed without being scanned")
		return nil
	}
	if err := s.scanner.Ping(ctx); err != nil {
		s.l.Warn("the malware scanner did not answer; files will be held until it does",
			zap.Bool("failOpen", s.failOpen), zap.Error(err))
	}
	return nil
}

func (s *Service) Check(
	ctx context.Context,
	req *contentscanports.ScanRequest,
) (*contentscanports.ScanResult, error) {
	return contentscanports.CheckWith(ctx, s, req)
}

func (s *Service) Stream(
	ctx context.Context,
	req *contentscanports.StreamRequest,
) (contentscanports.Stream, error) {
	if s.scanner == nil {
		return unscannedStream{}, nil
	}
	reader, writer := io.Pipe()
	stream := &scanStream{
		service: s,
		req:     req,
		writer:  writer,
		digest:  sha256.New(),
		done:    make(chan struct{}),
	}
	go func() {
		defer close(stream.done)
		stream.result, stream.err = s.scanner.Scan(ctx, reader, req.Size)
		if stream.err == nil {
			stream.err = drained(reader)
		}
		_ = reader.CloseWithError(errScanEnded)
	}()
	return stream, nil
}

var errScanEnded = errors.New("the malware scanner stopped reading")

func drained(reader io.Reader) error {
	var probe [1]byte
	if n, _ := reader.Read(probe[:]); n > 0 {
		return fmt.Errorf(
			"%w: %w before the end of the file",
			contentscanports.ErrScannerUnavailable,
			errScanEnded,
		)
	}
	return nil
}

type unscannedStream struct{}

func (unscannedStream) Write(p []byte) (int, error) { return len(p), nil }

func (unscannedStream) Finish() (*contentscanports.ScanResult, error) {
	return &contentscanports.ScanResult{Verdict: contentscanports.VerdictUnscanned}, nil
}

func (unscannedStream) Abort() {}

type scanStream struct {
	service  *Service
	req      *contentscanports.StreamRequest
	writer   *io.PipeWriter
	digest   hash.Hash
	size     int64
	broken   bool
	done     chan struct{}
	result   *contentscanports.ScanResult
	err      error
	finished sync.Once
	verdict  *contentscanports.ScanResult
	refusal  error
}

func (w *scanStream) Write(p []byte) (int, error) {
	w.digest.Write(p)
	w.size += int64(len(p))
	if w.broken {
		return len(p), nil
	}
	if _, err := w.writer.Write(p); err != nil {
		<-w.done
		if !w.service.failOpen {
			return 0, w.unavailable()
		}
		w.broken = true
	}
	return len(p), nil
}

func (w *scanStream) unavailable() error {
	if w.err != nil {
		return w.err
	}
	return fmt.Errorf("%w: %w", contentscanports.ErrScannerUnavailable, errScanEnded)
}

func (w *scanStream) Finish() (*contentscanports.ScanResult, error) {
	w.finished.Do(func() {
		_ = w.writer.Close()
		<-w.done
		w.verdict, w.refusal = w.decide()
	})
	return w.verdict, w.refusal
}

func (w *scanStream) Abort() {
	w.finished.Do(func() {
		_ = w.writer.CloseWithError(context.Canceled)
		<-w.done
		w.refusal = context.Canceled
	})
}

func (w *scanStream) decide() (*contentscanports.ScanResult, error) {
	s := w.service
	if w.err != nil || w.broken || w.result == nil {
		err := w.unavailable()
		if !s.failOpen {
			return nil, err
		}
		s.l.Warn("let a file through unscanned because the scanner could not check it",
			zap.String("source", string(w.req.Source)),
			zap.String("fileName", w.req.FileName),
			zap.Error(err))
		return &contentscanports.ScanResult{Verdict: contentscanports.VerdictUnscanned}, nil
	}

	if w.result.Verdict != contentscanports.VerdictInfected {
		return w.result, nil
	}

	s.recordInfected(w.req, w.result, hex.EncodeToString(w.digest.Sum(nil)), w.size)
	return w.result, &contentscanports.InfectedError{Signature: w.result.Signature}
}

func (s *Service) recordInfected(
	req *contentscanports.StreamRequest,
	result *contentscanports.ScanResult,
	checksum string,
	size int64,
) {
	s.l.Warn("refused a file the malware scanner flagged",
		zap.String("source", string(req.Source)),
		zap.String("fileName", req.FileName),
		zap.String("signature", result.Signature),
		zap.String("sha256", checksum),
		zap.String("organizationId", req.TenantInfo.OrgID.String()))

	if s.audit == nil {
		return
	}
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceDocument,
		ResourceID:     checksum,
		Operation:      permission.OpReject,
		UserID:         req.TenantInfo.UserID,
		PrincipalType:  permissionports.PrincipalTypeSystem,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Critical:       true,
		CurrentState: map[string]any{
			"fileName":  req.FileName,
			"source":    string(req.Source),
			"signature": result.Signature,
			"sha256":    checksum,
			"sizeBytes": size,
		},
	}
	comment := fmt.Sprintf("Refused %q from %s: the malware scanner found %s",
		req.FileName, req.Source, result.Signature)
	if err := s.audit.LogAction(params, func(entry *audit.Entry) error {
		entry.Comment = comment
		return nil
	}); err != nil {
		s.l.Error("could not record a refused infected file", zap.Error(err))
	}
}
